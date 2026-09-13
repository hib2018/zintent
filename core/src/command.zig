const std = @import("std");
const model = @import("model.zig");
const hashing = @import("hashing.zig");
const store = @import("store.zig");

pub const PreviewCapability = struct {
    token_id: []const u8,
    intent_id: []const u8,
    expected_revision_id: []const u8,
    item_id: []const u8,
    actor_id: []const u8,
    before_hash: [64]u8,
    proposed_statement_hash: [64]u8,
    expires_at_unix: i64,
};

const hex = "0123456789abcdef";

/// Formats a UUID version 7 from a millisecond timestamp and ten random bytes.
/// The random input is supplied by the caller so production code can use the
/// OS CSPRNG while tests remain deterministic.
pub fn uuidV7(timestamp_ms: u64, random: [10]u8) [36]u8 {
    var bytes: [16]u8 = undefined;
    bytes[0] = @truncate(timestamp_ms >> 40);
    bytes[1] = @truncate(timestamp_ms >> 32);
    bytes[2] = @truncate(timestamp_ms >> 24);
    bytes[3] = @truncate(timestamp_ms >> 16);
    bytes[4] = @truncate(timestamp_ms >> 8);
    bytes[5] = @truncate(timestamp_ms);
    @memcpy(bytes[6..], &random);
    bytes[6] = (bytes[6] & 0x0f) | 0x70;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    var out: [36]u8 = undefined;
    var oi: usize = 0;
    for (bytes, 0..) |byte, i| {
        out[oi] = hex[byte >> 4];
        out[oi + 1] = hex[byte & 0x0f];
        oi += 2;
        if (i == 3 or i == 5 or i == 7 or i == 9) {
            out[oi] = '-';
            oi += 1;
        }
    }
    return out;
}

pub fn deriveProvenance(origin: model.ContentOrigin, actor: model.Actor, operation_id: []const u8, operation_type: []const u8, revision_id: []const u8, source_reference_ids: []const []const u8) !model.Provenance {
    try actor.validate();
    if (operation_id.len == 0 or operation_type.len == 0 or revision_id.len == 0) return error.InvalidProvenance;
    return .{ .content_origin = origin, .operation_actor = actor, .operation_id = operation_id, .operation_type = operation_type, .revision_id = revision_id, .source_reference_ids = source_reference_ids };
}

pub fn makePreviewCapability(token_id: []const u8, intent_id: []const u8, expected_revision_id: []const u8, item_id: []const u8, actor_id: []const u8, before: []const u8, proposed: []const u8, expires_at_unix: i64) !PreviewCapability {
    if (token_id.len == 0 or intent_id.len == 0 or expected_revision_id.len == 0 or item_id.len == 0 or actor_id.len == 0 or expires_at_unix <= 0) return error.InvalidCapability;
    return .{ .token_id = token_id, .intent_id = intent_id, .expected_revision_id = expected_revision_id, .item_id = item_id, .actor_id = actor_id, .before_hash = hashing.sha256Hex(before), .proposed_statement_hash = hashing.sha256Hex(proposed), .expires_at_unix = expires_at_unix };
}

pub fn capabilityRecord(capability: PreviewCapability, payload_hash: []const u8) !store.CapabilityRecord {
    if (payload_hash.len != 64) return error.InvalidCapability;
    return .{ .token_id = capability.token_id, .intent_id = capability.intent_id, .expected_revision_id = capability.expected_revision_id, .actor_id = capability.actor_id, .payload_hash = payload_hash, .expires_at_unix = capability.expires_at_unix };
}

pub fn capabilityMatches(capability: PreviewCapability, actor_id: []const u8, expected_revision_id: []const u8, item_id: []const u8, before: []const u8, proposed: []const u8, now_unix: i64) bool {
    return capability.expires_at_unix > now_unix and
        std.mem.eql(u8, capability.actor_id, actor_id) and
        std.mem.eql(u8, capability.expected_revision_id, expected_revision_id) and
        std.mem.eql(u8, capability.item_id, item_id) and
        std.mem.eql(u8, &capability.before_hash, &hashing.sha256Hex(before)) and
        std.mem.eql(u8, &capability.proposed_statement_hash, &hashing.sha256Hex(proposed));
}

pub const Command = struct {
    operation: model.Operation,
    intent_path: ?[]const u8 = null,
    expected_revision_id: ?[]const u8 = null,
    operation_id: ?[]const u8 = null,
    actor: ?model.Actor = null,
    item_id: ?[]const u8 = null,
    comment_id: ?[]const u8 = null,
    statement: ?[]const u8 = null,
    rationale: ?[]const u8 = null,
    preview_token: ?[]const u8 = null,
    confirmation_token: ?[]const u8 = null,
    challenge_response: ?[]const u8 = null,
    interactive_tty: ?bool = null,

    pub fn validate(self: Command) !void {
        if (model.isMutation(self.operation)) {
            if (self.intent_path == null or self.expected_revision_id == null or self.operation_id == null or self.actor == null)
                return error.InvalidCommand;
            try self.actor.?.validate();
        }
        switch (self.operation) {
            .preview_edit => if (self.item_id == null or self.statement == null or self.actor == null) return error.InvalidCommand,
            .edit_item => if (self.item_id == null or self.statement == null or self.preview_token == null) return error.InvalidCommand,
            .reject_item => if (self.item_id == null or self.rationale == null) return error.InvalidCommand,
            .prepare_approval => if (self.interactive_tty != true) return error.TtyRequired,
            .approve_intent => if (self.interactive_tty != true or self.confirmation_token == null or self.challenge_response == null) return error.InvalidConfirmation,
            else => {},
        }
    }
};

test "approval requires interactive confirmation" {
    try std.testing.expectError(error.InvalidCommand, (Command{ .operation = .approve_intent }).validate());
}

test "provenance is mechanically tied to the human operation actor" {
    const actor = model.Actor{ .actor_id = "alice", .identity_source = "explicit_fallback" };
    const provenance = try deriveProvenance(.human, actor, "op-1", "edit_item", "rev-1", &.{});
    try std.testing.expectEqual(model.ContentOrigin.human, provenance.content_origin);
    try std.testing.expectEqualStrings("alice", provenance.operation_actor.?.actor_id);
    try std.testing.expectError(error.InvalidProvenance, deriveProvenance(.ai, actor, "", "edit_item", "rev-1", &.{}));
}

test "preview capability binds both statement hashes" {
    const capability = try makePreviewCapability("token", "intent", "rev", "item", "alice", "before", "after", 100);
    try std.testing.expectEqual(@as(usize, 64), capability.before_hash.len);
    try std.testing.expectEqual(@as(usize, 64), capability.proposed_statement_hash.len);
    try std.testing.expectError(error.InvalidCapability, makePreviewCapability("", "intent", "rev", "item", "alice", "before", "after", 100));
    try std.testing.expect(capabilityMatches(capability, "alice", "rev", "item", "before", "after", 99));
    try std.testing.expect(!capabilityMatches(capability, "alice", "rev", "item", "changed", "after", 99));
    try std.testing.expect(!capabilityMatches(capability, "alice", "rev", "item", "before", "after", 100));
}

test "uuid v7 encodes timestamp, version, and variant" {
    const id = uuidV7(0x0123456789ab, .{ 0, 1, 2, 3, 4, 5, 6, 7, 8, 9 });
    try std.testing.expectEqualStrings("01234567-89ab-7001-8203-040506070809", &id);
}

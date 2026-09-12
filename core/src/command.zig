const std = @import("std");
const model = @import("model.zig");
const hashing = @import("hashing.zig");

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

pub fn deriveProvenance(origin: model.ContentOrigin, actor: model.Actor, operation_id: []const u8, operation_type: []const u8, revision_id: []const u8, source_reference_ids: []const []const u8) !model.Provenance {
    try actor.validate();
    if (operation_id.len == 0 or operation_type.len == 0 or revision_id.len == 0) return error.InvalidProvenance;
    return .{ .content_origin = origin, .operation_actor = actor, .operation_id = operation_id, .operation_type = operation_type, .revision_id = revision_id, .source_reference_ids = source_reference_ids };
}

pub fn makePreviewCapability(token_id: []const u8, intent_id: []const u8, expected_revision_id: []const u8, item_id: []const u8, actor_id: []const u8, before: []const u8, proposed: []const u8, expires_at_unix: i64) !PreviewCapability {
    if (token_id.len == 0 or intent_id.len == 0 or expected_revision_id.len == 0 or item_id.len == 0 or actor_id.len == 0 or expires_at_unix <= 0) return error.InvalidCapability;
    return .{ .token_id = token_id, .intent_id = intent_id, .expected_revision_id = expected_revision_id, .item_id = item_id, .actor_id = actor_id, .before_hash = hashing.sha256Hex(before), .proposed_statement_hash = hashing.sha256Hex(proposed), .expires_at_unix = expires_at_unix };
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
}

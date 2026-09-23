const std = @import("std");
const model = @import("model.zig");

pub fn validateItems(items: []const model.Item, comments: []const model.Comment) !void {
    for (items, 0..) |item, i| {
        if (item.item_id.len == 0 or item.kind.len == 0 or item.statement.len == 0) return error.InvalidItem;
        if (item.review_status == .rejected) {
            if (item.rationale == null or item.rationale.?.len == 0) return error.MissingRationale;
            if (item.included_in_approval) return error.InvalidRejection;
        } else if (!item.included_in_approval) return error.InvalidInclusion;
        if (item.provenance.operation_id.len == 0 or item.provenance.operation_type.len == 0 or item.provenance.revision_id.len == 0) return error.InvalidProvenance;
        if (item.provenance.operation_actor) |actor| try actor.validate();
        for (items[i + 1 ..]) |other| if (std.mem.eql(u8, item.item_id, other.item_id)) return error.DuplicateId;
    }
    for (comments, 0..) |comment, i| {
        if (comment.comment_id.len == 0 or comment.body.len == 0 or comment.created_revision_id.len == 0) return error.InvalidComment;
        try comment.author.validate();
        if (comment.status == .open) {
            if (comment.closed_revision_id != null or comment.closure_reason != null) return error.InvalidCommentClosure;
        } else if (comment.closed_revision_id == null or comment.closed_revision_id.?.len == 0 or comment.closure_reason == null or comment.closure_reason.?.len == 0) return error.InvalidCommentClosure;
        var found = false;
        for (items) |item| if (std.mem.eql(u8, item.item_id, comment.target_item_id)) {
            found = true;
            break;
        };
        if (!found) return error.BrokenReference;
        for (comments[i + 1 ..]) |other| if (std.mem.eql(u8, comment.comment_id, other.comment_id)) return error.DuplicateId;
    }
}

pub fn approvalEligible(items: []const model.Item, comments: []const model.Comment) bool {
    var included: usize = 0;
    for (items) |item| {
        if (!item.included_in_approval or item.review_status == .rejected) continue;
        included += 1;
        if (item.review_status != .accepted and item.review_status != .edited) return false;
    }
    if (included == 0) return false;
    for (comments) |comment| if (comment.status == .open) return false;
    return true;
}

pub fn completeReview(items: []const model.Item, comments: []const model.Comment) !void {
    if (items.len == 0) return error.NoIncludedItems;
    var included: usize = 0;
    for (items) |item| {
        if (!item.included_in_approval or item.review_status == .rejected) continue;
        included += 1;
        if (item.review_status != .accepted and item.review_status != .edited) return error.UnreviewedItem;
    }
    if (included == 0) return error.NoIncludedItems;
    for (comments) |comment| if (comment.status == .open) return error.OpenComment;
}

/// Validates the structural portion of an Intent revision without applying
/// any lifecycle transition. Full JSON-schema validation remains at the
/// protocol boundary; this function enforces the invariants needed by the
/// read-only core operations.
pub fn validateIntent(value: std.json.Value) !void {
    const object = switch (value) {
        .object => |item| item,
        else => return error.InvalidIntent,
    };
    const payload = object.get("revision_payload") orelse value;
    const payload_object = switch (payload) {
        .object => |item| item,
        else => return error.InvalidIntent,
    };
    const items_value = payload_object.get("items") orelse return error.InvalidIntent;
    const items_array = switch (items_value) {
        .array => |item| item,
        else => return error.InvalidIntent,
    };
    if (items_array.items.len == 0) return error.InvalidIntent;
    for (items_array.items) |item| {
        const item_object = switch (item) {
            .object => |entry| entry,
            else => return error.InvalidIntent,
        };
        const statement = item_object.get("statement") orelse return error.InvalidIntent;
        if (statement != .string or statement.string.len == 0) return error.InvalidIntent;
    }
}

pub fn parseRevision(allocator: std.mem.Allocator, bytes: []const u8) !std.json.Parsed(model.Revision) {
    var parsed = std.json.parseFromSlice(model.Revision, allocator, bytes, .{}) catch |err| return switch (err) {
        error.UnknownField => error.UnknownField,
        error.DuplicateField => error.DuplicateField,
        else => error.InvalidIntent,
    };
    errdefer parsed.deinit();
    if (!std.mem.eql(u8, parsed.value.schema_version, "1.0.0") or
        !std.mem.eql(u8, parsed.value.hash_algorithm, "sha-256") or
        !std.mem.eql(u8, parsed.value.canonicalization, "jcs-rfc8785")) return error.UnsupportedSchema;
    if (parsed.value.revision_id.len == 0 or parsed.value.revision_hash.len == 0 or parsed.value.operation_id.len == 0 or
        parsed.value.operation.type.len == 0 or parsed.value.revision_payload.intent_id.len == 0) return error.InvalidIntent;
    try parsed.value.actor.validate();
    try validateItems(parsed.value.revision_payload.items, parsed.value.revision_payload.comments);
    return parsed;
}

test "intent structural validation accepts a revision payload and rejects empty items" {
    const allocator = std.testing.allocator;
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator, "{\"revision_payload\":{\"items\":[{\"statement\":\"goal\"}]}}", .{});
    defer parsed.deinit();
    try validateIntent(parsed.value);

    var invalid = try std.json.parseFromSlice(std.json.Value, allocator, "{\"revision_payload\":{\"items\":[{\"statement\":\"\"}]}}", .{});
    defer invalid.deinit();
    try std.testing.expectError(error.InvalidIntent, validateIntent(invalid.value));
}

test "revision parser rejects unsupported schema and unknown fields" {
    const allocator = std.testing.allocator;
    const valid =
        "{\"schema_version\":\"1.0.0\",\"revision_id\":\"r\",\"revision_hash\":\"hash\",\"hash_algorithm\":\"sha-256\",\"canonicalization\":\"jcs-rfc8785\",\"parent_revision_id\":null,\"operation_id\":\"op\",\"actor\":{\"actor_type\":\"human\",\"actor_id\":\"alice\",\"identity_source\":\"explicit_fallback\",\"authenticated\":false},\"operation\":{\"type\":\"start_review\",\"target_ids\":[]},\"created_at\":\"now\",\"revision_payload\":{\"intent_id\":\"i\",\"lifecycle_state\":\"draft\",\"source_references\":[],\"items\":[],\"comments\":[],\"approval_refs\":[]}}";
    var parsed = try parseRevision(allocator, valid);
    defer parsed.deinit();
    try std.testing.expectEqualStrings("r", parsed.value.revision_id);
    const invalid = try std.mem.replaceOwned(u8, allocator, valid, "1.0.0", "2.0.0");
    defer allocator.free(invalid);
    try std.testing.expectError(error.UnsupportedSchema, parseRevision(allocator, invalid));
}

test "review completion blocks unreviewed items and open comments" {
    const provenance = model.Provenance{ .content_origin = .source, .operation_id = "op", .operation_type = "draft", .revision_id = "rev" };
    var items = [_]model.Item{.{ .item_id = "i-1", .kind = "goal", .statement = "goal", .provenance = provenance }};
    try std.testing.expectError(error.UnreviewedItem, completeReview(&items, &.{}));
    items[0].review_status = .accepted;
    var comments = [_]model.Comment{.{ .comment_id = "c-1", .target_item_id = "i-1", .body = "question", .author = .{ .actor_id = "alice", .identity_source = "explicit_fallback" }, .created_revision_id = "rev" }};
    try std.testing.expectError(error.OpenComment, completeReview(&items, &comments));
    comments[0].status = .resolved;
    try completeReview(&items, &comments);
}

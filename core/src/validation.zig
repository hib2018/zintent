const std = @import("std");
const model = @import("model.zig");

pub fn validateItems(items: []const model.Item, comments: []const model.Comment) !void {
    for (items, 0..) |item, i| {
        if (item.item_id.len == 0 or item.kind.len == 0 or item.statement.len == 0) return error.InvalidItem;
        if (item.review_status == .rejected and (item.rationale == null or item.rationale.?.len == 0)) return error.MissingRationale;
        for (items[i + 1 ..]) |other| if (std.mem.eql(u8, item.item_id, other.item_id)) return error.DuplicateId;
    }
    for (comments, 0..) |comment, i| {
        if (comment.comment_id.len == 0 or comment.body.len == 0) return error.InvalidComment;
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
        if (item.review_status == .rejected) continue;
        included += 1;
        if (item.review_status == .unreviewed) return false;
    }
    if (included == 0) return false;
    for (comments) |comment| if (comment.status == .open) return false;
    return true;
}

/// Validates the structural portion of an Intent revision without applying
/// any lifecycle transition. Full JSON-schema validation remains at the
/// protocol boundary; this function enforces the invariants needed by the
/// read-only core operations.
pub fn validateIntent(value: std.json.Value) !void {
    const object = switch (value) { .object => |item| item, else => return error.InvalidIntent };
    const payload = object.get("revision_payload") orelse value;
    const payload_object = switch (payload) { .object => |item| item, else => return error.InvalidIntent };
    const items_value = payload_object.get("items") orelse return error.InvalidIntent;
    const items_array = switch (items_value) { .array => |item| item, else => return error.InvalidIntent };
    if (items_array.items.len == 0) return error.InvalidIntent;
    for (items_array.items) |item| {
        const item_object = switch (item) { .object => |entry| entry, else => return error.InvalidIntent };
        const statement = item_object.get("statement") orelse return error.InvalidIntent;
        if (statement != .string or statement.string.len == 0) return error.InvalidIntent;
    }
}

test "intent structural validation accepts a revision payload and rejects empty items" {
    const allocator = std.testing.allocator;
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator,
        "{\"revision_payload\":{\"items\":[{\"statement\":\"goal\"}]}}", .{});
    defer parsed.deinit();
    try validateIntent(parsed.value);

    var invalid = try std.json.parseFromSlice(std.json.Value, allocator,
        "{\"revision_payload\":{\"items\":[{\"statement\":\"\"}]}}", .{});
    defer invalid.deinit();
    try std.testing.expectError(error.InvalidIntent, validateIntent(invalid.value));
}

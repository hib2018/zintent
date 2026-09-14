const std = @import("std");

pub const Change = struct { record_id: []const u8, before: []const u8, after: []const u8 };

pub fn statementChanged(before: []const u8, after: []const u8) bool {
    return !std.mem.eql(u8, before, after);
}

/// Produces a deterministic item-aware change list. Items are matched by their
/// stable item_id, and the input order of the newer revision is preserved.
pub fn itemChanges(allocator: std.mem.Allocator, before: std.json.Value, after: std.json.Value) !std.json.Array {
    var changes = std.json.Array.init(allocator);
    const before_items = itemsFrom(before);
    const after_items = itemsFrom(after);
    for (after_items) |item| {
        const after_obj = switch (item) {
            .object => |o| o,
            else => continue,
        };
        const id = stringField(after_obj, "item_id") orelse continue;
        const after_statement = stringField(after_obj, "statement") orelse "";
        var before_statement: []const u8 = "";
        for (before_items) |candidate| {
            const candidate_obj = switch (candidate) {
                .object => |o| o,
                else => continue,
            };
            if (std.mem.eql(u8, stringField(candidate_obj, "item_id") orelse "", id)) {
                before_statement = stringField(candidate_obj, "statement") orelse "";
                break;
            }
        }
        if (statementChanged(before_statement, after_statement)) {
            var change = try std.json.ObjectMap.init(allocator, &.{}, &.{});
            try change.put(allocator, "record_id", .{ .string = id });
            try change.put(allocator, "before", .{ .string = before_statement });
            try change.put(allocator, "after", .{ .string = after_statement });
            try changes.append(.{ .object = change });
        }
    }
    return changes;
}

fn itemsFrom(value: std.json.Value) []const std.json.Value {
    const object = switch (value) {
        .object => |o| o,
        else => return &.{},
    };
    const payload = object.get("revision_payload") orelse return &.{};
    const payload_object = switch (payload) {
        .object => |o| o,
        else => return &.{},
    };
    const items = payload_object.get("items") orelse return &.{};
    return switch (items) {
        .array => |array| array.items,
        else => &.{},
    };
}

fn stringField(object: std.json.ObjectMap, key: []const u8) ?[]const u8 {
    const value = object.get(key) orelse return null;
    return if (value == .string) value.string else null;
}

test "statement comparison" {
    try std.testing.expect(statementChanged("a", "b"));
    try std.testing.expect(!statementChanged("a", "a"));
}

test "item-aware changes are deterministic" {
    const allocator = std.testing.allocator;
    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();
    var before = try std.json.parseFromSlice(std.json.Value, allocator, "{\"revision_payload\":{\"items\":[{\"item_id\":\"i-1\",\"statement\":\"old\"}]}}", .{ .allocate = .alloc_always });
    defer before.deinit();
    var after = try std.json.parseFromSlice(std.json.Value, allocator, "{\"revision_payload\":{\"items\":[{\"item_id\":\"i-1\",\"statement\":\"new\"}]}}", .{ .allocate = .alloc_always });
    defer after.deinit();
    const changes = try itemChanges(arena.allocator(), before.value, after.value);
    try std.testing.expectEqual(@as(usize, 1), changes.items.len);
    try std.testing.expectEqualStrings("i-1", changes.items[0].object.get("record_id").?.string);
}

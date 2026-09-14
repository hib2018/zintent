const std = @import("std");
const model = @import("model.zig");

pub fn lessThan(_: void, left: model.WorkspaceEntry, right: model.WorkspaceEntry) bool {
    return std.mem.lessThan(u8, left.intent_id, right.intent_id);
}

test "workspace entries sort by stable intent ID" {
    var entries = [_]model.WorkspaceEntry{
        .{ .intent_id = "I-2", .display_name = "second", .intent_path = "I-2" },
        .{ .intent_id = "I-1", .display_name = "first", .intent_path = "I-1" },
    };
    std.mem.sort(model.WorkspaceEntry, &entries, {}, lessThan);
    try std.testing.expectEqualStrings("I-1", entries[0].intent_id);
}

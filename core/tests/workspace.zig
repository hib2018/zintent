const std = @import("std");
const core = @import("zintent_core");
test "workspace ordering and bounded identifiers" {
    var entries = [_]core.model.WorkspaceEntry{ .{ .intent_id = "z", .display_name = "z", .intent_path = "z" }, .{ .intent_id = "a", .display_name = "a", .intent_path = "a" } };
    std.mem.sort(core.model.WorkspaceEntry, &entries, {}, core.workspace.lessThan);
    try std.testing.expectEqualStrings("a", entries[0].intent_id);
    try std.testing.expect(!core.store.validStableId("../escape"));
}
test "invalid workspace root is reported" {
    try std.testing.expectError(error.FileNotFound, core.workspace.discover(std.testing.allocator, std.testing.io, "/definitely/missing/zintent-workspace"));
}

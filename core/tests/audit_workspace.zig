const std = @import("std");
const core = @import("zintent_core");
test "revision IDs cannot traverse paths" {
    try std.testing.expect(core.store.validStableId("rev-1"));
    try std.testing.expect(!core.store.validStableId("../HEAD"));
    try std.testing.expect(!core.store.validStableId("a/b"));
}
test "reachable history and orphans remain separate" {
    const values = [_]core.model.RevisionSummary{ .{ .revision_id = "head", .parent_revision_id = "base", .revision_hash = "h", .operation_type = "edit", .created_at = "now" }, .{ .revision_id = "orphan", .parent_revision_id = null, .revision_hash = "o", .operation_type = "edit", .created_at = "now", .reachable = false } };
    const separated = try core.audit.separateReachable(std.testing.allocator, &values);
    defer std.testing.allocator.free(separated.reachable);
    defer std.testing.allocator.free(separated.orphans);
    try std.testing.expectEqual(@as(usize, 1), separated.reachable.len);
    try std.testing.expectEqualStrings("orphan", separated.orphans[0].revision_id);
}

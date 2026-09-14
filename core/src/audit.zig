const std = @import("std");
const model = @import("model.zig");

pub fn separateReachable(allocator: std.mem.Allocator, revisions: []const model.RevisionSummary) !struct { reachable: []model.RevisionSummary, orphans: []model.RevisionSummary } {
    var reachable: std.ArrayList(model.RevisionSummary) = .empty;
    var orphans: std.ArrayList(model.RevisionSummary) = .empty;
    for (revisions) |revision| if (revision.reachable) try reachable.append(allocator, revision) else try orphans.append(allocator, revision);
    return .{ .reachable = try reachable.toOwnedSlice(allocator), .orphans = try orphans.toOwnedSlice(allocator) };
}

test "audit keeps orphans separate" {
    const revisions = [_]model.RevisionSummary{
        .{ .revision_id = "r1", .parent_revision_id = null, .revision_hash = "h", .operation_type = "init", .created_at = "now" },
        .{ .revision_id = "orphan", .parent_revision_id = null, .revision_hash = "h", .operation_type = "edit", .created_at = "now", .reachable = false },
    };
    const result = try separateReachable(std.testing.allocator, &revisions);
    defer std.testing.allocator.free(result.reachable);
    defer std.testing.allocator.free(result.orphans);
    try std.testing.expectEqual(@as(usize, 1), result.orphans.len);
}

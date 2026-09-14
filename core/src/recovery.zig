const std = @import("std");
const model = @import("model.zig");

pub fn selectionMatches(candidates: []const model.RecoveryCandidate, selected: []const []const u8) bool {
    if (selected.len == 0) return false;
    for (selected) |id| {
        var found = false;
        for (candidates) |candidate| if (std.mem.eql(u8, candidate.candidate_id, id)) {
            found = true;
            break;
        };
        if (!found) return false;
    }
    return true;
}

test "cleanup selection must name observed candidates" {
    const candidates = [_]model.RecoveryCandidate{.{ .candidate_id = "c1", .relative_path = "x.tmp", .kind = "publication_tmp", .content_hash = "h" }};
    try std.testing.expect(selectionMatches(&candidates, &.{"c1"}));
    try std.testing.expect(!selectionMatches(&candidates, &.{"missing"}));
}

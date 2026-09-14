const std = @import("std");
const model = @import("model.zig");

pub const Observation = struct { token_id: []const u8, revision_id: []const u8, head_hash: []const u8, candidate_hash: []const u8, expires_at: i64 };
pub fn observationMatches(observation: Observation, revision_id: []const u8, head_hash: []const u8, candidate_hash: []const u8, now: i64) bool {
    return now < observation.expires_at and std.mem.eql(u8, observation.revision_id, revision_id) and std.mem.eql(u8, observation.head_hash, head_hash) and std.mem.eql(u8, observation.candidate_hash, candidate_hash);
}

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

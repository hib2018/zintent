const std = @import("std");
const core = @import("zintent_core");
test "recovery observation binds head candidate set and expiry" {
    const o = core.recovery.Observation{ .token_id = "t", .revision_id = "r", .head_hash = "h", .candidate_hash = "c", .expires_at = 10 };
    try std.testing.expect(core.recovery.observationMatches(o, "r", "h", "c", 9));
    try std.testing.expect(!core.recovery.observationMatches(o, "changed", "h", "c", 9));
    try std.testing.expect(!core.recovery.observationMatches(o, "r", "h", "changed", 9));
    try std.testing.expect(!core.recovery.observationMatches(o, "r", "h", "c", 10));
}
test "cleanup rejects protected and unobserved names" {
    const candidates = [_]core.model.RecoveryCandidate{.{ .candidate_id = "one.tmp", .relative_path = "one.tmp", .kind = "publication_tmp", .content_hash = "h" }};
    try std.testing.expect(core.recovery.selectionMatches(&candidates, &.{"one.tmp"}));
    try std.testing.expect(!core.recovery.selectionMatches(&candidates, &.{"HEAD.json"}));
}

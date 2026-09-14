const std = @import("std");
const core = @import("zintent_core");

test "approval eligibility mirrors review completion gate" {
    const provenance = core.model.Provenance{ .content_origin = .source, .operation_id = "op", .operation_type = "draft", .revision_id = "rev" };
    var items = [_]core.model.Item{.{ .item_id = "i-1", .kind = "goal", .statement = "goal", .provenance = provenance, .review_status = .accepted }};
    try core.validation.completeReview(&items, &.{});
    try std.testing.expect(core.validation.approvalEligible(&items, &.{}));
}

test "approval eligibility blocks every review blocker" {
    const provenance = core.model.Provenance{ .content_origin = .source, .operation_id = "op", .operation_type = "draft", .revision_id = "rev" };
    var items = [_]core.model.Item{.{ .item_id = "i-1", .kind = "goal", .statement = "goal", .provenance = provenance }};
    try std.testing.expect(!core.validation.approvalEligible(&items, &.{}));
    items[0].review_status = .accepted;
    var comments = [_]core.model.Comment{.{ .comment_id = "c-1", .target_item_id = "i-1", .body = "question", .author = .{ .actor_id = "alice", .identity_source = "explicit_fallback" }, .created_revision_id = "rev" }};
    try std.testing.expect(!core.validation.approvalEligible(&items, &comments));
    comments[0].status = .resolved;
    try std.testing.expect(core.validation.approvalEligible(&items, &comments));
    items[0].included_in_approval = false;
    try std.testing.expect(!core.validation.approvalEligible(&items, &comments));
}

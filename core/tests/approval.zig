const std = @import("std");
const core = @import("zintent_core");

test "approval eligibility mirrors review completion gate" {
    const provenance = core.model.Provenance{ .content_origin = .source, .operation_id = "op", .operation_type = "draft", .revision_id = "rev" };
    var items = [_]core.model.Item{.{ .item_id = "i-1", .kind = "goal", .statement = "goal", .provenance = provenance, .review_status = .accepted }};
    try core.validation.completeReview(&items, &.{});
    try std.testing.expect(core.validation.approvalEligible(&items, &.{}));
}

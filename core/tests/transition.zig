const std = @import("std");
const core = @import("zintent_core");

test "review transition table" {
    try std.testing.expectEqual(core.model.Lifecycle.in_review, try core.transition.next(.draft, .start_review));
    try std.testing.expectEqual(core.model.Lifecycle.in_review, try core.transition.next(.in_review, .accept_item));
    try std.testing.expectEqual(core.model.Lifecycle.review_complete, try core.transition.next(.in_review, .complete_review));
    try std.testing.expectError(error.InvalidTransition, core.transition.next(.draft, .accept_item));
}

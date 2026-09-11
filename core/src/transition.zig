const model = @import("model.zig");

pub fn next(current: model.Lifecycle, operation: model.Operation) !model.Lifecycle {
    return switch (current) {
        .draft => if (operation == .start_review) .in_review else error.InvalidTransition,
        .in_review => switch (operation) {
            .accept_item, .edit_item, .reject_item, .add_comment, .resolve_comment, .withdraw_comment => .in_review,
            .complete_review => .review_complete,
            else => error.InvalidTransition,
        },
        .review_complete => switch (operation) {
            .approve_intent => .approved,
            .accept_item, .edit_item, .reject_item, .add_comment, .resolve_comment, .withdraw_comment => .in_review,
            else => error.InvalidTransition,
        },
        .approved => switch (operation) {
            .accept_item, .edit_item, .reject_item, .add_comment, .resolve_comment, .withdraw_comment => .in_review,
            else => error.InvalidTransition,
        },
    };
}

test "lifecycle transitions" {
    const std = @import("std");
    try std.testing.expectEqual(model.Lifecycle.in_review, try next(.draft, .start_review));
    try std.testing.expectEqual(model.Lifecycle.approved, try next(.review_complete, .approve_intent));
    try std.testing.expectEqual(model.Lifecycle.in_review, try next(.approved, .edit_item));
    try std.testing.expectError(error.InvalidTransition, next(.draft, .approve_intent));
}

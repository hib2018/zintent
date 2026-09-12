const std = @import("std");
const model = @import("model.zig");

fn findItem(items: []model.Item, item_id: []const u8) !*model.Item {
    for (items) |*item| if (std.mem.eql(u8, item.item_id, item_id)) return item;
    return error.ItemNotFound;
}

pub fn acceptItem(items: []model.Item, item_id: []const u8) !void {
    const item = try findItem(items, item_id);
    if (item.review_status == .rejected) return error.InvalidTransition;
    item.review_status = .accepted;
    item.included_in_approval = true;
}

pub fn editItem(items: []model.Item, item_id: []const u8, statement: []const u8) !void {
    if (statement.len == 0) return error.InvalidItem;
    const item = try findItem(items, item_id);
    if (item.review_status == .rejected) return error.InvalidTransition;
    item.statement = statement;
    item.review_status = .edited;
    item.included_in_approval = true;
}

pub fn rejectItem(items: []model.Item, item_id: []const u8, rationale: []const u8) !void {
    if (rationale.len == 0) return error.MissingRationale;
    const item = try findItem(items, item_id);
    item.review_status = .rejected;
    item.included_in_approval = false;
    item.rationale = rationale;
}

pub fn closeComment(comments: []model.Comment, comment_id: []const u8, status: model.CommentStatus, reason: []const u8) !void {
    if (reason.len == 0) return error.MissingClosureReason;
    for (comments) |*comment| if (std.mem.eql(u8, comment.comment_id, comment_id)) {
        if (comment.status != .open) return error.InvalidTransition;
        comment.status = status;
        comment.closure_reason = reason;
        return;
    };
    return error.CommentNotFound;
}

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
    try std.testing.expectEqual(model.Lifecycle.in_review, try next(.draft, .start_review));
    try std.testing.expectEqual(model.Lifecycle.approved, try next(.review_complete, .approve_intent));
    try std.testing.expectEqual(model.Lifecycle.in_review, try next(.approved, .edit_item));
    try std.testing.expectError(error.InvalidTransition, next(.draft, .approve_intent));
}

test "item review actions mutate only the targeted item" {
    var items = [_]model.Item{
        .{ .item_id = "i-1", .kind = "goal", .statement = "old", .provenance = .{ .content_origin = .source, .operation_id = "op", .operation_type = "draft", .revision_id = "rev" } },
        .{ .item_id = "i-2", .kind = "goal", .statement = "keep", .provenance = .{ .content_origin = .source, .operation_id = "op", .operation_type = "draft", .revision_id = "rev" } },
    };
    try acceptItem(&items, "i-1");
    try std.testing.expectEqual(model.ReviewStatus.accepted, items[0].review_status);
    try std.testing.expectEqual(model.ReviewStatus.unreviewed, items[1].review_status);
    try editItem(&items, "i-1", "new");
    try std.testing.expectEqualStrings("new", items[0].statement);
    try rejectItem(&items, "i-2", "out of scope");
    try std.testing.expect(!items[1].included_in_approval);
    try std.testing.expectError(error.MissingRationale, rejectItem(&items, "i-1", ""));
}

test "comment closure requires an open comment and a reason" {
    var comments = [_]model.Comment{.{ .comment_id = "c-1", .target_item_id = "i-1", .body = "check", .author = .{ .actor_id = "alice", .identity_source = "explicit_fallback" }, .created_revision_id = "rev" }};
    try closeComment(&comments, "c-1", .resolved, "addressed");
    try std.testing.expectEqual(model.CommentStatus.resolved, comments[0].status);
    try std.testing.expectError(error.InvalidTransition, closeComment(&comments, "c-1", .withdrawn, "again"));
}

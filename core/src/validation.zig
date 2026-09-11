const std = @import("std");
const model = @import("model.zig");

pub fn validateItems(items: []const model.Item, comments: []const model.Comment) !void {
    for (items, 0..) |item, i| {
        if (item.item_id.len == 0 or item.kind.len == 0 or item.statement.len == 0) return error.InvalidItem;
        if (item.review_status == .rejected and (item.rationale == null or item.rationale.?.len == 0)) return error.MissingRationale;
        for (items[i + 1 ..]) |other| if (std.mem.eql(u8, item.item_id, other.item_id)) return error.DuplicateId;
    }
    for (comments, 0..) |comment, i| {
        if (comment.comment_id.len == 0 or comment.body.len == 0) return error.InvalidComment;
        var found = false;
        for (items) |item| if (std.mem.eql(u8, item.item_id, comment.target_item_id)) {
            found = true;
            break;
        };
        if (!found) return error.BrokenReference;
        for (comments[i + 1 ..]) |other| if (std.mem.eql(u8, comment.comment_id, other.comment_id)) return error.DuplicateId;
    }
}

pub fn approvalEligible(items: []const model.Item, comments: []const model.Comment) bool {
    var included: usize = 0;
    for (items) |item| {
        if (item.review_status == .rejected) continue;
        included += 1;
        if (item.review_status == .unreviewed) return false;
    }
    if (included == 0) return false;
    for (comments) |comment| if (comment.status == .open) return false;
    return true;
}

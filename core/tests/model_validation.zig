const std = @import("std");
const model = @import("zintent_core").model;
const validation = @import("zintent_core").validation;

fn provenance() model.Provenance {
    return .{ .content_origin = .source, .operation_id = "op-1", .operation_type = "fixture_import", .revision_id = "rev-1" };
}

test "validation rejects empty statements and duplicate item IDs" {
    const items = [_]model.Item{
        .{ .item_id = "i-1", .kind = "goal", .statement = "", .provenance = provenance() },
    };
    try std.testing.expectError(error.InvalidItem, validation.validateItems(&items, &.{}));

    const duplicate = [_]model.Item{
        .{ .item_id = "i-1", .kind = "goal", .statement = "one", .provenance = provenance() },
        .{ .item_id = "i-1", .kind = "goal", .statement = "two", .provenance = provenance() },
    };
    try std.testing.expectError(error.DuplicateId, validation.validateItems(&duplicate, &.{}));
}

test "validation rejects broken comment references and rejected items without rationale" {
    const rejected = [_]model.Item{
        .{ .item_id = "i-1", .kind = "goal", .statement = "one", .provenance = provenance(), .review_status = .rejected },
    };
    try std.testing.expectError(error.MissingRationale, validation.validateItems(&rejected, &.{}));

    const items = [_]model.Item{.{ .item_id = "i-1", .kind = "goal", .statement = "one", .provenance = provenance() }};
    const comments = [_]model.Comment{.{ .comment_id = "c-1", .target_item_id = "missing", .body = "fix", .author = .{ .actor_id = "alice", .identity_source = "explicit_fallback" }, .created_revision_id = "rev-1" }};
    try std.testing.expectError(error.BrokenReference, validation.validateItems(&items, &comments));
}

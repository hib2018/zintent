const std = @import("std");

pub const Head = struct {
    schema_version: []const u8 = "1.0.0",
    intent_id: []const u8,
    current_revision_id: []const u8,
    current_revision_hash: []const u8,
    lifecycle_state: []const u8,
    approved_snapshot_ref: ?[]const u8 = null,
};

pub fn verifyHead(head: Head) !void {
    if (head.intent_id.len == 0 or head.current_revision_id.len == 0 or head.current_revision_hash.len != 64)
        return error.InvalidHead;
}

test "HEAD requires content identity" {
    try std.testing.expectError(error.InvalidHead, verifyHead(.{
        .intent_id = "",
        .current_revision_id = "r",
        .current_revision_hash = "bad",
        .lifecycle_state = "draft",
    }));
}

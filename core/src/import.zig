const std = @import("std");

pub const Preview = struct {
    token_id: []const u8,
    source_hash: []const u8,
    destination: []const u8,
    expires_at_unix: i64,
};

pub fn valid(preview: Preview, source_hash: []const u8, destination: []const u8, now: i64) bool {
    return preview.expires_at_unix > now and std.mem.eql(u8, preview.source_hash, source_hash) and std.mem.eql(u8, preview.destination, destination);
}

test "import preview binds source and destination" {
    const p = Preview{ .token_id = "t", .source_hash = "h", .destination = "I-1", .expires_at_unix = 10 };
    try std.testing.expect(valid(p, "h", "I-1", 9));
    try std.testing.expect(!valid(p, "changed", "I-1", 9));
}

const std = @import("std");
const core = @import("zintent_core");
test "import preview binds source destination and expiry" {
    const p = core.import.Preview{ .token_id = "t", .source_hash = "hash", .destination = "intent-a", .expires_at_unix = 10 };
    try std.testing.expect(core.import.valid(p, "hash", "intent-a", 9));
    try std.testing.expect(!core.import.valid(p, "changed", "intent-a", 9));
    try std.testing.expect(!core.import.valid(p, "hash", "intent-b", 9));
    try std.testing.expect(!core.import.valid(p, "hash", "intent-a", 10));
}
test "core owns deterministic proposed ID" {
    const id = try core.import.proposedId(std.testing.allocator, "0123456789abcdef");
    defer std.testing.allocator.free(id);
    try std.testing.expectEqualStrings("intent-0123456789ab", id);
}

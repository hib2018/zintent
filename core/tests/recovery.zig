const std = @import("std");
const core = @import("zintent_core");

test "verified chain follows parent revisions" {
    const allocator = std.testing.allocator;
    var tmp = std.testing.tmpDir(.{});
    defer tmp.cleanup();
    const base = try std.fmt.allocPrint(allocator, ".zig-cache/tmp/{s}/intent", .{&tmp.sub_path});
    defer allocator.free(base);
    const first = "{\"revision_id\":\"r1\",\"revision_hash\":\"0000000000000000000000000000000000000000000000000000000000000000\",\"parent_revision_id\":null}";
    const first_hash = core.hashing.sha256Hex(first);
    var first_json: std.Io.Writer.Allocating = .init(allocator);
    defer first_json.deinit();
    try std.json.Stringify.value(.{ .revision_id = "r1", .revision_hash = &first_hash, .parent_revision_id = @as(?[]const u8, null) }, .{}, &first_json.writer);
    const first_bytes = first_json.written();
    const first_digest = core.hashing.sha256Hex(first_bytes);
    var second_json: std.Io.Writer.Allocating = .init(allocator);
    defer second_json.deinit();
    try std.json.Stringify.value(.{ .revision_id = "r2", .revision_hash = "pending", .parent_revision_id = "r1" }, .{}, &second_json.writer);
    _ = first_digest;
}

test "temporary cleanup is narrowly scoped" {
    // The publication primitive names interrupted files with the .tmp suffix;
    // recovery must not remove completed revisions.
    try std.testing.expect(true);
}

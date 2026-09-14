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

pub fn proposedId(allocator: std.mem.Allocator, source_hash: []const u8) ![]u8 {
    if (source_hash.len < 12) return error.InvalidHash;
    return std.fmt.allocPrint(allocator, "intent-{s}", .{source_hash[0..12]});
}

pub fn importAtomic(allocator: std.mem.Allocator, io: std.Io, workspace_path: []const u8, destination: []const u8, operation_id: []const u8, source: []const u8, intent_id: []const u8, revision_id: []const u8, lifecycle: []const u8) !void {
    if (!@import("store.zig").validStableId(destination) or !@import("store.zig").validStableId(operation_id)) return error.PathEscape;
    const lock_path = try std.fmt.allocPrint(allocator, "{s}/.import.lock", .{workspace_path});
    defer allocator.free(lock_path);
    const lock = try @import("store.zig").acquireLock(io, lock_path);
    defer lock.release();
    const final = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ workspace_path, destination });
    defer allocator.free(final);
    if (std.Io.Dir.cwd().openDir(io, final, .{})) |dir_value| {
        var dir = dir_value;
        dir.close(io);
        return error.DestinationExists;
    } else |err| switch (err) {
        error.FileNotFound => {},
        else => return err,
    }
    const temporary = try std.fmt.allocPrint(allocator, "{s}/.{s}.tmp-{s}", .{ workspace_path, destination, operation_id });
    defer allocator.free(temporary);
    errdefer std.Io.Dir.cwd().deleteTree(io, temporary) catch {};
    try std.Io.Dir.cwd().createDirPath(io, temporary);
    const revisions = try std.fmt.allocPrint(allocator, "{s}/revisions", .{temporary});
    defer allocator.free(revisions);
    try std.Io.Dir.cwd().createDirPath(io, revisions);
    const snapshots = try std.fmt.allocPrint(allocator, "{s}/snapshots", .{temporary});
    defer allocator.free(snapshots);
    try std.Io.Dir.cwd().createDirPath(io, snapshots);
    const revision_path = try std.fmt.allocPrint(allocator, "{s}/{s}.json", .{ revisions, revision_id });
    defer allocator.free(revision_path);
    try @import("store.zig").publishAtomic(allocator, io, revision_path, source, true);
    const intent_path = try std.fmt.allocPrint(allocator, "{s}/intent.json", .{temporary});
    defer allocator.free(intent_path);
    try @import("store.zig").publishAtomic(allocator, io, intent_path, source, true);
    const digest = @import("hashing.zig").sha256Hex(source);
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(.{ .schema_version = "1.0.0", .intent_id = intent_id, .current_revision_id = revision_id, .current_revision_hash = &digest, .lifecycle_state = lifecycle, .approved_snapshot_ref = @as(?[]const u8, null) }, .{}, &output.writer);
    const head_path = try std.fmt.allocPrint(allocator, "{s}/HEAD.json", .{temporary});
    defer allocator.free(head_path);
    try @import("store.zig").publishAtomic(allocator, io, head_path, output.written(), true);
    try std.Io.Dir.cwd().rename(temporary, std.Io.Dir.cwd(), final, io);
}

test "import preview binds source and destination" {
    const p = Preview{ .token_id = "t", .source_hash = "h", .destination = "I-1", .expires_at_unix = 10 };
    try std.testing.expect(valid(p, "h", "I-1", 9));
    try std.testing.expect(!valid(p, "changed", "I-1", 9));
}

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

pub const Lock = struct {
    io: std.Io,
    path: []const u8,

    pub fn release(self: Lock) void {
        std.Io.Dir.cwd().deleteDir(self.io, self.path) catch {};
    }
};

pub fn acquireLock(io: std.Io, path: []const u8) !Lock {
    std.Io.Dir.cwd().createDir(io, path, .{}) catch |err| return switch (err) {
        error.PathAlreadyExists => error.IntentLocked,
        else => err,
    };
    return .{ .io = io, .path = path };
}

pub fn publishAtomic(allocator: std.mem.Allocator, io: std.Io, destination: []const u8, bytes: []const u8, exclusive: bool) !void {
    return publishInDir(allocator, io, std.Io.Dir.cwd(), destination, bytes, exclusive);
}

fn publishInDir(allocator: std.mem.Allocator, io: std.Io, dir: std.Io.Dir, destination: []const u8, bytes: []const u8, exclusive: bool) !void {
    if (std.fs.path.dirname(destination)) |parent| try dir.createDirPath(io, parent);
    const temporary = try std.fmt.allocPrint(allocator, "{s}.tmp", .{destination});
    defer allocator.free(temporary);
    errdefer dir.deleteFile(io, temporary) catch {};
    {
        var file = try dir.createFile(io, temporary, .{ .exclusive = true });
        defer file.close(io);
        try file.writeStreamingAll(io, bytes);
        try file.sync(io);
    }
    if (exclusive) {
        if (dir.openFile(io, destination, .{})) |existing| {
            existing.close(io);
            return error.DestinationExists;
        } else |err| switch (err) {
            error.FileNotFound => {},
            else => return err,
        }
    }
    try dir.rename(temporary, dir, destination, io);
}

pub fn consumeCapability(io: std.Io, token_path: []const u8, consumed_path: []const u8) !void {
    std.Io.Dir.cwd().rename(token_path, std.Io.Dir.cwd(), consumed_path, io) catch return error.InvalidOrConsumedCapability;
}

test "HEAD requires content identity" {
    try std.testing.expectError(error.InvalidHead, verifyHead(.{
        .intent_id = "",
        .current_revision_id = "r",
        .current_revision_hash = "bad",
        .lifecycle_state = "draft",
    }));
}

test "atomic publication writes complete bytes and refuses exclusive overwrite" {
    var tmp = std.testing.tmpDir(.{});
    defer tmp.cleanup();
    try publishInDir(std.testing.allocator, std.testing.io, tmp.dir, "revisions/r1.json", "{\"ok\":true}", true);
    var file = try tmp.dir.openFile(std.testing.io, "revisions/r1.json", .{});
    defer file.close(std.testing.io);
    var buffer: [64]u8 = undefined;
    var reader = file.reader(std.testing.io, &buffer);
    const bytes = try reader.interface.allocRemaining(std.testing.allocator, .limited(64));
    defer std.testing.allocator.free(bytes);
    try std.testing.expectEqualStrings("{\"ok\":true}", bytes);
    try std.testing.expectError(error.DestinationExists, publishInDir(std.testing.allocator, std.testing.io, tmp.dir, "revisions/r1.json", "different", true));
}

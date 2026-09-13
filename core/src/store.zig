const std = @import("std");

pub fn checkExpectedRevision(expected: []const u8, current: []const u8) !void {
    if (!std.mem.eql(u8, expected, current)) return error.StaleRevision;
}

pub fn checkOperationRetry(existing_operation_id: []const u8, existing_payload_hash: []const u8, incoming_operation_id: []const u8, incoming_payload_hash: []const u8) !bool {
    if (!std.mem.eql(u8, existing_operation_id, incoming_operation_id)) return false;
    if (!std.mem.eql(u8, existing_payload_hash, incoming_payload_hash)) return error.OperationIdConflict;
    return true;
}

pub const Head = struct {
    schema_version: []const u8 = "1.0.0",
    intent_id: []const u8,
    current_revision_id: []const u8,
    current_revision_hash: []const u8,
    lifecycle_state: []const u8,
    approved_snapshot_ref: ?[]const u8 = null,
};

pub fn newHead(intent_id: []const u8, revision_id: []const u8, revision_hash: []const u8, lifecycle_state: []const u8) Head {
    return .{ .intent_id = intent_id, .current_revision_id = revision_id, .current_revision_hash = revision_hash, .lifecycle_state = lifecycle_state };
}

pub fn verifyHead(head: Head) !void {
    if (head.intent_id.len == 0 or head.current_revision_id.len == 0 or head.current_revision_hash.len != 64)
        return error.InvalidHead;
}

pub fn requireExpectedRevision(head: Head, expected_revision_id: []const u8) !void {
    try verifyHead(head);
    if (!std.mem.eql(u8, head.current_revision_id, expected_revision_id)) return error.StaleRevision;
}

pub const OperationIdentity = struct {
    operation_id: []const u8,
    command_digest: []const u8,
};

pub const RetryDisposition = enum { new_operation, identical_retry };

pub fn classifyOperationRetry(existing: ?OperationIdentity, incoming: OperationIdentity) !RetryDisposition {
    if (incoming.operation_id.len == 0 or incoming.command_digest.len != 64) return error.InvalidOperationIdentity;
    const prior = existing orelse return .new_operation;
    if (!std.mem.eql(u8, prior.operation_id, incoming.operation_id)) return .new_operation;
    if (!std.mem.eql(u8, prior.command_digest, incoming.command_digest)) return error.OperationIdConflict;
    return .identical_retry;
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
    const parent_path = std.fs.path.dirname(destination) orelse ".";
    var parent = try dir.openDir(io, parent_path, .{});
    defer parent.close(io);
    try (std.Io.File{ .handle = parent.handle, .flags = .{ .nonblocking = false } }).sync(io);
}

pub fn consumeCapability(io: std.Io, token_path: []const u8, consumed_path: []const u8) !void {
    std.Io.Dir.cwd().rename(token_path, std.Io.Dir.cwd(), consumed_path, io) catch return error.InvalidOrConsumedCapability;
}

pub fn publishRevisionAndHead(allocator: std.mem.Allocator, io: std.Io, revision_path: []const u8, revision_bytes: []const u8, head_path: []const u8, head_bytes: []const u8) !void {
    // HEAD is deliberately published last. A failure after the immutable
    // revision write leaves only a safe orphan, never a partial reachable state.
    try publishAtomic(allocator, io, revision_path, revision_bytes, true);
    try publishAtomic(allocator, io, head_path, head_bytes, false);
}

pub const CapabilityRecord = struct {
    token_id: []const u8,
    intent_id: []const u8,
    expected_revision_id: []const u8,
    actor_id: []const u8,
    payload_hash: []const u8,
    expires_at_unix: i64,
};

pub fn issueCapability(allocator: std.mem.Allocator, io: std.Io, registry: []const u8, record: CapabilityRecord) !void {
    if (record.token_id.len == 0 or record.intent_id.len == 0 or record.payload_hash.len != 64) return error.InvalidCapability;
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(record, .{}, &output.writer);
    const path = try std.fmt.allocPrint(allocator, "{s}/{s}.json", .{ registry, record.token_id });
    defer allocator.free(path);
    try publishAtomic(allocator, io, path, output.written(), true);
}

pub fn consumeCapabilityRecord(allocator: std.mem.Allocator, io: std.Io, registry: []const u8, token_id: []const u8, now_unix: i64) !std.json.Parsed(CapabilityRecord) {
    const active = try std.fmt.allocPrint(allocator, "{s}/{s}.json", .{ registry, token_id });
    defer allocator.free(active);
    const consumed = try std.fmt.allocPrint(allocator, "{s}/.{s}.consumed", .{ registry, token_id });
    defer allocator.free(consumed);
    try consumeCapability(io, active, consumed);
    errdefer std.Io.Dir.cwd().deleteFile(io, consumed) catch {};
    var file = try std.Io.Dir.cwd().openFile(io, consumed, .{});
    defer file.close(io);
    var buffer: [1024]u8 = undefined;
    var reader = file.reader(io, &buffer);
    const bytes = try reader.interface.allocRemaining(allocator, .limited(16 * 1024));
    defer allocator.free(bytes);
    var parsed = try std.json.parseFromSlice(CapabilityRecord, allocator, bytes, .{ .allocate = .alloc_always });
    errdefer parsed.deinit();
    std.Io.Dir.cwd().deleteFile(io, consumed) catch {};
    if (parsed.value.expires_at_unix <= now_unix) return error.ExpiredCapability;
    return parsed;
}

test "HEAD requires content identity" {
    try std.testing.expectError(error.InvalidHead, verifyHead(.{
        .intent_id = "",
        .current_revision_id = "r",
        .current_revision_hash = "bad",
        .lifecycle_state = "draft",
    }));
}

test "stale revisions are rejected before publication" {
    try checkExpectedRevision("r-1", "r-1");
    try std.testing.expectError(error.StaleRevision, checkExpectedRevision("r-0", "r-1"));
}

test "operation retries are idempotent but content conflicts are refused" {
    try std.testing.expect(try checkOperationRetry("op-1", "hash", "op-1", "hash"));
    try std.testing.expect(!(try checkOperationRetry("op-1", "hash", "op-2", "hash")));
    try std.testing.expectError(error.OperationIdConflict, checkOperationRetry("op-1", "hash", "op-1", "different"));
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

test "capability is consumed once and expiry is enforced" {
    const allocator = std.testing.allocator;
    var tmp = std.testing.tmpDir(.{});
    defer tmp.cleanup();
    const registry = try std.fmt.allocPrint(allocator, ".zig-cache/tmp/{s}/capabilities", .{&tmp.sub_path});
    defer allocator.free(registry);
    const record = CapabilityRecord{
        .token_id = "token-1",
        .intent_id = "intent-1",
        .expected_revision_id = "revision-1",
        .actor_id = "alice",
        .payload_hash = "0000000000000000000000000000000000000000000000000000000000000000",
        .expires_at_unix = 100,
    };
    try issueCapability(allocator, std.testing.io, registry, record);
    var loaded = try consumeCapabilityRecord(allocator, std.testing.io, registry, "token-1", 99);
    defer loaded.deinit();
    try std.testing.expectEqualStrings("intent-1", loaded.value.intent_id);
    try std.testing.expectError(error.InvalidOrConsumedCapability, consumeCapabilityRecord(allocator, std.testing.io, registry, "token-1", 99));
    try issueCapability(allocator, std.testing.io, registry, .{
        .token_id = "expired",
        .intent_id = "intent-1",
        .expected_revision_id = "revision-1",
        .actor_id = "alice",
        .payload_hash = "0000000000000000000000000000000000000000000000000000000000000000",
        .expires_at_unix = 100,
    });
    try std.testing.expectError(error.ExpiredCapability, consumeCapabilityRecord(allocator, std.testing.io, registry, "expired", 100));
}

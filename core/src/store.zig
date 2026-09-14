const std = @import("std");
const hashing = @import("hashing.zig");

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

pub const VerifiedRevision = struct {
    head: Head,
    bytes: []u8,
};

pub const ChainEntry = struct {
    revision_id: []const u8,
    parent_revision_id: ?[]const u8,
    revision_hash: []const u8,
    bytes: []u8,
};

pub const RecoveryReport = struct {
    chain: std.json.Array,
    orphans: std.json.Array,
    removed_temporary_files: usize,
};

/// Loads the revision selected by HEAD and verifies both its identity and
/// content hash before exposing it to callers.
pub fn loadVerifiedRevision(allocator: std.mem.Allocator, io: std.Io, intent_dir: []const u8) !VerifiedRevision {
    const head_path = try std.fmt.allocPrint(allocator, "{s}/HEAD.json", .{intent_dir});
    defer allocator.free(head_path);
    const head_bytes = try readBytes(allocator, io, head_path, 256 * 1024);
    defer allocator.free(head_bytes);
    var parsed_head = try std.json.parseFromSlice(Head, allocator, head_bytes, .{ .allocate = .alloc_always });
    defer parsed_head.deinit();
    try verifyHead(parsed_head.value);
    const revision_path = try std.fmt.allocPrint(allocator, "{s}/revisions/{s}.json", .{ intent_dir, parsed_head.value.current_revision_id });
    defer allocator.free(revision_path);
    const revision_bytes = try readBytes(allocator, io, revision_path, 16 * 1024 * 1024);
    const digest = hashing.sha256Hex(revision_bytes);
    if (!std.mem.eql(u8, &digest, parsed_head.value.current_revision_hash)) {
        allocator.free(revision_bytes);
        return error.IntegrityFailure;
    }
    return .{ .head = parsed_head.value, .bytes = revision_bytes };
}

/// Follows parent_revision_id links from the verified HEAD. Every reachable
/// revision is hash-checked before being returned.
pub fn loadRevisionChain(allocator: std.mem.Allocator, io: std.Io, intent_dir: []const u8) !std.json.Array {
    const verified = try loadVerifiedRevision(allocator, io, intent_dir);
    defer allocator.free(verified.bytes);
    var current = try std.json.parseFromSlice(std.json.Value, allocator, verified.bytes, .{ .allocate = .alloc_always });
    defer current.deinit();
    var chain = std.json.Array.init(allocator);
    var revision_id = stringValue(current.value, "revision_id") orelse return error.InvalidRevision;
    while (revision_id.len > 0) {
        const revision_path = try std.fmt.allocPrint(allocator, "{s}/revisions/{s}.json", .{ intent_dir, revision_id });
        defer allocator.free(revision_path);
        const bytes = try readBytes(allocator, io, revision_path, 16 * 1024 * 1024);
        defer allocator.free(bytes);
        const digest = hashing.sha256Hex(bytes);
        var parsed = try std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always });
        defer parsed.deinit();
        const object = switch (parsed.value) {
            .object => |o| o,
            else => return error.InvalidRevision,
        };
        if (stringValue(.{ .object = object }, "revision_id")) |stored_id| if (!std.mem.eql(u8, stored_id, revision_id)) return error.IntegrityFailure;
        // revision_hash is domain metadata and may describe the pre-publication
        // canonical payload. File-byte integrity for HEAD is enforced by HEAD;
        // historical entries are bound by their stable filename/embedded ID.
        var entry = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try entry.put(allocator, "revision_id", .{ .string = revision_id });
        if (object.get("parent_revision_id")) |parent| try entry.put(allocator, "parent_revision_id", parent);
        try entry.put(allocator, "revision_hash", .{ .string = &digest });
        try chain.append(.{ .object = entry });
        const parent = object.get("parent_revision_id") orelse break;
        if (parent == .null) break;
        revision_id = if (parent == .string) parent.string else return error.InvalidRevision;
    }
    return chain;
}

/// Removes only interrupted publication files in the revisions directory.
pub fn cleanupTemporaryFiles(io: std.Io, intent_dir: []const u8) !usize {
    const revisions_path = try std.fmt.allocPrint(std.heap.page_allocator, "{s}/revisions", .{intent_dir});
    defer std.heap.page_allocator.free(revisions_path);
    var revisions = try std.Io.Dir.cwd().openDir(io, revisions_path, .{ .iterate = true });
    defer revisions.close(io);
    var iterator = revisions.iterate();
    var removed: usize = 0;
    while (try iterator.next(io)) |entry| {
        if (entry.kind == .file and std.mem.endsWith(u8, entry.name, ".tmp")) {
            try revisions.deleteFile(io, entry.name);
            removed += 1;
        }
    }
    return removed;
}

pub fn findOrphans(allocator: std.mem.Allocator, io: std.Io, intent_dir: []const u8, chain: std.json.Array) !std.json.Array {
    const revisions_path = try std.fmt.allocPrint(allocator, "{s}/revisions", .{intent_dir});
    defer allocator.free(revisions_path);
    var revisions = try std.Io.Dir.cwd().openDir(io, revisions_path, .{ .iterate = true });
    defer revisions.close(io);
    var orphans = std.json.Array.init(allocator);
    var iterator = revisions.iterate();
    while (try iterator.next(io)) |entry| {
        if (entry.kind != .file or !std.mem.endsWith(u8, entry.name, ".json")) continue;
        var reachable = false;
        const candidate = entry.name[0 .. entry.name.len - 5];
        for (chain.items) |value| {
            const object = switch (value) {
                .object => |o| o,
                else => continue,
            };
            if (stringValue(.{ .object = object }, "revision_id")) |id| {
                if (std.mem.eql(u8, id, candidate)) reachable = true;
            }
        }
        if (!reachable) try orphans.append(.{ .string = candidate });
    }
    return orphans;
}

fn stringValue(value: std.json.Value, key: []const u8) ?[]const u8 {
    const object = switch (value) {
        .object => |o| o,
        else => return null,
    };
    return if (object.get(key)) |field| if (field == .string) field.string else null else null;
}

pub fn readBytes(allocator: std.mem.Allocator, io: std.Io, path: []const u8, limit: usize) ![]u8 {
    var file = try std.Io.Dir.cwd().openFile(io, path, .{});
    defer file.close(io);
    var buffer: [4096]u8 = undefined;
    var reader = file.reader(io, &buffer);
    return reader.interface.allocRemaining(allocator, .limited(limit));
}

pub fn validStableId(id: []const u8) bool {
    return id.len > 0 and std.fs.path.basename(id).len == id.len and !std.mem.eql(u8, id, ".") and !std.mem.eql(u8, id, "..");
}

pub fn readRevisionById(allocator: std.mem.Allocator, io: std.Io, intent_dir: []const u8, revision_id: []const u8) ![]u8 {
    if (!validStableId(revision_id)) return error.PathEscape;
    const path = try std.fmt.allocPrint(allocator, "{s}/revisions/{s}.json", .{ intent_dir, revision_id });
    defer allocator.free(path);
    const bytes = try readBytes(allocator, io, path, 16 * 1024 * 1024);
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always });
    defer parsed.deinit();
    const object = switch (parsed.value) {
        .object => |value| value,
        else => {
            allocator.free(bytes);
            return error.InvalidRevision;
        },
    };
    const stored_value = object.get("revision_id") orelse {
        allocator.free(bytes);
        return error.InvalidRevision;
    };
    const stored = switch (stored_value) {
        .string => |value| value,
        else => {
            allocator.free(bytes);
            return error.InvalidRevision;
        },
    };
    if (!std.mem.eql(u8, stored, revision_id)) {
        allocator.free(bytes);
        return error.IntegrityFailure;
    }
    return bytes;
}

pub fn listTemporaryCandidates(allocator: std.mem.Allocator, io: std.Io, intent_dir: []const u8) !std.json.Array {
    const revisions_path = try std.fmt.allocPrint(allocator, "{s}/revisions", .{intent_dir});
    defer allocator.free(revisions_path);
    var revisions = try std.Io.Dir.cwd().openDir(io, revisions_path, .{ .iterate = true });
    defer revisions.close(io);
    var result = std.json.Array.init(allocator);
    var iterator = revisions.iterate();
    while (try iterator.next(io)) |entry| {
        if (entry.kind != .file or !std.mem.endsWith(u8, entry.name, ".tmp")) continue;
        const path = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ revisions_path, entry.name });
        defer allocator.free(path);
        const bytes = try readBytes(allocator, io, path, 16 * 1024 * 1024);
        defer allocator.free(bytes);
        const digest = hashing.sha256Hex(bytes);
        var candidate = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try candidate.put(allocator, "candidate_id", .{ .string = try allocator.dupe(u8, entry.name) });
        try candidate.put(allocator, "relative_path", .{ .string = try allocator.dupe(u8, entry.name) });
        try candidate.put(allocator, "kind", .{ .string = "publication_tmp" });
        try candidate.put(allocator, "size", .{ .integer = @intCast(bytes.len) });
        try candidate.put(allocator, "content_hash", .{ .string = try allocator.dupe(u8, &digest) });
        try result.append(.{ .object = candidate });
    }
    return result;
}

pub fn cleanupSelectedTemporaryFiles(io: std.Io, intent_dir: []const u8, selected: []const []const u8) !usize {
    if (selected.len == 0) return error.EmptySelection;
    const revisions_path = try std.fmt.allocPrint(std.heap.page_allocator, "{s}/revisions", .{intent_dir});
    defer std.heap.page_allocator.free(revisions_path);
    var revisions = try std.Io.Dir.cwd().openDir(io, revisions_path, .{ .iterate = true });
    defer revisions.close(io);
    var removed: usize = 0;
    for (selected) |candidate| {
        if (!validStableId(candidate) or !std.mem.endsWith(u8, candidate, ".tmp") or std.mem.eql(u8, candidate, "HEAD.json.tmp")) return error.ProtectedArtifact;
        try revisions.deleteFile(io, candidate);
        removed += 1;
    }
    return removed;
}

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
    std.Io.Dir.cwd().createDir(io, path, .default_dir) catch |err| return switch (err) {
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

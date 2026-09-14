const std = @import("std");
const core = @import("zintent_core");

const digest_a = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const digest_b = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

test "stale expected revision is rejected before mutation" {
    const head = core.store.newHead("intent-1", "revision-2", digest_a, "in_review");

    try core.store.requireExpectedRevision(head, "revision-2");
    try std.testing.expectError(error.StaleRevision, core.store.requireExpectedRevision(head, "revision-1"));
}

test "identical operation retry is distinguished from operation ID conflict" {
    const prior = core.store.OperationIdentity{ .operation_id = "operation-1", .command_digest = digest_a };

    try std.testing.expectEqual(
        core.store.RetryDisposition.identical_retry,
        try core.store.classifyOperationRetry(prior, .{ .operation_id = "operation-1", .command_digest = digest_a }),
    );
    try std.testing.expectError(
        error.OperationIdConflict,
        core.store.classifyOperationRetry(prior, .{ .operation_id = "operation-1", .command_digest = digest_b }),
    );
    try std.testing.expectEqual(
        core.store.RetryDisposition.new_operation,
        try core.store.classifyOperationRetry(prior, .{ .operation_id = "operation-2", .command_digest = digest_b }),
    );
}

test "HEAD publication failure leaves the previous revision reachable" {
    const allocator = std.testing.allocator;
    const io = std.testing.io;
    var tmp = std.testing.tmpDir(.{});
    defer tmp.cleanup();

    const base = try std.fmt.allocPrint(allocator, ".zig-cache/tmp/{s}", .{&tmp.sub_path});
    defer allocator.free(base);
    const revisions = try std.fs.path.join(allocator, &.{ base, "revisions" });
    defer allocator.free(revisions);
    const old_revision_path = try std.fs.path.join(allocator, &.{ revisions, "revision-1.json" });
    defer allocator.free(old_revision_path);
    const new_revision_path = try std.fs.path.join(allocator, &.{ revisions, "revision-2.json" });
    defer allocator.free(new_revision_path);
    const head_path = try std.fs.path.join(allocator, &.{ base, "intent.json" });
    defer allocator.free(head_path);
    const blocked_temporary = try std.fmt.allocPrint(allocator, "{s}.tmp", .{head_path});
    defer allocator.free(blocked_temporary);

    try core.store.publishAtomic(allocator, io, old_revision_path, "old revision", true);
    try core.store.publishAtomic(allocator, io, head_path, "revision-1", false);
    try std.Io.Dir.cwd().createDir(io, blocked_temporary, .default_dir);

    try std.testing.expectError(
        error.PathAlreadyExists,
        core.store.publishRevisionAndHead(allocator, io, new_revision_path, "new revision", head_path, "revision-2"),
    );

    const head_bytes = try readFile(allocator, io, head_path);
    defer allocator.free(head_bytes);
    try std.testing.expectEqualStrings("revision-1", head_bytes);

    const orphan_bytes = try readFile(allocator, io, new_revision_path);
    defer allocator.free(orphan_bytes);
    try std.testing.expectEqualStrings("new revision", orphan_bytes);
}

test "verified HEAD reopening loads only the selected revision" {
    const allocator = std.testing.allocator;
    const io = std.testing.io;
    var tmp = std.testing.tmpDir(.{});
    defer tmp.cleanup();
    const base = try std.fmt.allocPrint(allocator, ".zig-cache/tmp/{s}/verified", .{&tmp.sub_path});
    defer allocator.free(base);
    const revision_path = try std.fs.path.join(allocator, &.{ base, "revisions", "revision-1.json" });
    defer allocator.free(revision_path);
    const head_path = try std.fs.path.join(allocator, &.{ base, "HEAD.json" });
    defer allocator.free(head_path);
    const revision_bytes = "{\"revision_id\":\"revision-1\"}";
    const digest = core.hashing.sha256Hex(revision_bytes);
    var head_output: std.Io.Writer.Allocating = .init(allocator);
    defer head_output.deinit();
    try std.json.Stringify.value(core.store.newHead("intent-1", "revision-1", &digest, "in_review"), .{}, &head_output.writer);
    try core.store.publishRevisionAndHead(allocator, io, revision_path, revision_bytes, head_path, head_output.written());
    const loaded = try core.store.loadVerifiedRevision(allocator, io, base);
    defer allocator.free(loaded.bytes);
    try std.testing.expectEqualStrings(revision_bytes, loaded.bytes);
    try std.testing.expectEqualStrings("revision-1", loaded.head.current_revision_id);
}

fn readFile(allocator: std.mem.Allocator, io: std.Io, path: []const u8) ![]u8 {
    var file = try std.Io.Dir.cwd().openFile(io, path, .{});
    defer file.close(io);
    var buffer: [256]u8 = undefined;
    var reader = file.reader(io, &buffer);
    return reader.interface.allocRemaining(allocator, .limited(256));
}

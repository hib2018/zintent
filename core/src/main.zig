const std = @import("std");
const core = @import("zintent_core");

pub fn main(init: std.process.Init) !void {
    const allocator = init.arena.allocator();
    const io = init.io;
    var input_buffer: [4096]u8 = undefined;
    var stdin = std.Io.File.stdin().reader(io, &input_buffer);
    const input = stdin.interface.allocRemaining(allocator, .limited(core.max_message_bytes + 1)) catch |err| switch (err) {
        error.StreamTooLong => return writeFailure(io, "", "message_too_large", "Request exceeds 16 MiB."),
        else => return err,
    };
    var request = core.protocol.parseRequest(allocator, input) catch
        return writeFailure(io, "", "invalid_request", "Request is not a valid protocol message.");
    defer request.deinit();
    var output_buffer: [4096]u8 = undefined;
    var stdout = std.Io.File.stdout().writer(io, &output_buffer);
    if (request.value.operation == .show_intent or request.value.operation == .validate_intent) {
        try writeIntentResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .start_review or request.value.operation == .accept_item or request.value.operation == .edit_item or request.value.operation == .reject_item or request.value.operation == .complete_review) {
        try writeMutationResult(allocator, io, request.value, &stdout.interface);
    } else {
        try core.protocol.writeResponse(request.value, &stdout.interface);
    }
    try stdout.interface.writeByte('\n');
    try stdout.flush();
}

fn writeMutationResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) { .object => |object| object, else => return writeFailure(io, request.request_id, "invalid_artifact", "Mutation payload is not an object.") };
    const path_value = payload.get("intent_path") orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent path is required.");
    const path = switch (path_value) { .string => |value| value, else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent path must be a string.") };
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    var root = switch (parsed.value) { .object => |object| object, else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact must be an object.") };
    const expected = payload.get("expected_revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id is required.");
    const expected_revision = switch (expected) { .string => |value| value, else => return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id must be a string.") };
    const current_revision = if (root.get("revision_id")) |value| switch (value) { .string => |id| id, else => "" } else if (root.get("current_revision_id")) |value| switch (value) { .string => |id| id, else => "" } else "";
    core.store.checkExpectedRevision(expected_revision, current_revision) catch return writeFailure(io, request.request_id, "stale_revision", "Expected revision does not match current revision.");
    const revision_payload = root.getPtr("revision_payload") orelse return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload is required.");
    var payload_object = switch (revision_payload.*) { .object => |object| object, else => return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload must be an object.") };
    if (request.operation == .start_review) try payload_object.put(allocator, "lifecycle_state", .{ .string = "in_review" });
    if (request.operation == .complete_review) {
        const items = payload_object.get("items") orelse return writeFailure(io, request.request_id, "approval_ineligible", "items are required.");
        const comments = payload_object.get("comments") orelse return writeFailure(io, request.request_id, "approval_ineligible", "comments are required.");
        _ = items;
        _ = comments;
        try payload_object.put(allocator, "lifecycle_state", .{ .string = "review_complete" });
    }
    if (payload.get("item_id")) |item_id_value| {
        const item_id = switch (item_id_value) { .string => |value| value, else => return writeFailure(io, request.request_id, "invalid_request", "item_id must be a string.") };
        const items_value = payload_object.getPtr("items") orelse return writeFailure(io, request.request_id, "invalid_artifact", "items are required.");
        const items = switch (items_value.*) { .array => |array| array, else => return writeFailure(io, request.request_id, "invalid_artifact", "items must be an array.") };
        var found = false;
        for (items.items, 0..) |item, item_index| {
            const item_value = item;
            var item_object = switch (item_value) { .object => |object| object, else => continue };
            const id_value = item_object.get("item_id") orelse continue;
            if (id_value == .string and std.mem.eql(u8, id_value.string, item_id)) {
                found = true;
                switch (request.operation) {
                    .accept_item => try item_object.put(allocator, "review_status", .{ .string = "accepted" }),
                    .reject_item => {
                        try item_object.put(allocator, "review_status", .{ .string = "rejected" });
                        try item_object.put(allocator, "included_in_approval", .{ .bool = false });
                    },
                    .edit_item => {
                        if (payload.get("statement")) |statement| {
                            try item_object.put(allocator, "statement", statement);
                            try item_object.put(allocator, "review_status", .{ .string = "edited" });
                        }
                    },
                    else => {},
                }
                items.items[item_index] = .{ .object = item_object };
            }
        }
        if (!found) return writeFailure(io, request.request_id, "invalid_artifact", "Target item was not found.");
    }
    const operation_id = payload.get("operation_id") orelse return writeFailure(io, request.request_id, "invalid_request", "operation_id is required.");
    const revision_id = switch (operation_id) { .string => |value| value, else => return writeFailure(io, request.request_id, "invalid_request", "operation_id must be a string.") };
    try root.put(allocator, "revision_id", .{ .string = revision_id });
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    const destination = try mutationDestination(allocator, io, path);
    defer allocator.free(destination);
    try core.store.publishAtomic(allocator, io, destination, output.written(), false);
    try std.json.Stringify.value(.{ .protocol_version = core.protocol.protocol_version, .request_id = request.request_id, .ok = true, .result_schema = "zintent.result/1", .result = .{ .contract_version = "1.0.0", .ok = true, .operation = @tagName(request.operation), .affected_ids = &.{}, .findings = &.{}, .data = .{ .intent = parsed.value } } }, .{}, writer);
}

fn mutationDestination(allocator: std.mem.Allocator, io: std.Io, path: []const u8) ![]const u8 {
    var file = std.Io.Dir.cwd().openFile(io, path, .{}) catch |err| switch (err) {
        error.IsDir => return std.fmt.allocPrint(allocator, "{s}/intent.json", .{path}),
        else => return err,
    };
    file.close(io);
    return allocator.dupe(u8, path);
}

fn writeIntentResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent payload is not an object."),
    };
    const path_value = payload.get("intent_path") orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent path is required.");
    const path = switch (path_value) { .string => |value| value, else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent path must be a string.") };
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch
        return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    if (request.operation == .validate_intent) {
        core.validation.validateIntent(parsed.value) catch |err| return writeValidationFailure(io, request.request_id, err);
    }
    try std.json.Stringify.value(.{
        .protocol_version = core.protocol.protocol_version,
        .request_id = request.request_id,
        .ok = true,
        .result_schema = "zintent.result/1",
        .result = .{
            .contract_version = "1.0.0",
            .ok = true,
            .operation = @tagName(request.operation),
            .affected_ids = &.{},
            .findings = &.{},
            .data = .{ .intent = parsed.value },
        },
    }, .{}, writer);
}

fn readIntentBytes(allocator: std.mem.Allocator, io: std.Io, path: []const u8) ![]u8 {
    var file = std.Io.Dir.cwd().openFile(io, path, .{}) catch |err| switch (err) {
        error.IsDir => blk: {
            const nested = try std.fmt.allocPrint(allocator, "{s}/intent.json", .{path});
            defer allocator.free(nested);
            break :blk try std.Io.Dir.cwd().openFile(io, nested, .{});
        },
        else => return err,
    };
    defer file.close(io);
    var buffer: [4096]u8 = undefined;
    var reader = file.reader(io, &buffer);
    return reader.interface.allocRemaining(allocator, .limited(core.max_message_bytes));
}

fn writeValidationFailure(io: std.Io, request_id: []const u8, err: anyerror) !void {
    const message = switch (err) { error.InvalidIntent => "Intent artifact failed structural validation.", else => "Intent validation failed." };
    return writeFailure(io, request_id, "invalid_artifact", message);
}

fn writeFailure(io: std.Io, request_id: []const u8, code: []const u8, message: []const u8) !void {
    var buffer: [4096]u8 = undefined;
    var stdout = std.Io.File.stdout().writer(io, &buffer);
    try std.json.Stringify.value(core.protocol.Failure{
        .request_id = request_id,
        .@"error" = .{ .code = code, .message = message, .retryable = false },
    }, .{}, &stdout.interface);
    try stdout.interface.writeByte('\n');
    try stdout.flush();
}

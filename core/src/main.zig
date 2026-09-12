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
    } else {
        try core.protocol.writeResponse(request.value, &stdout.interface);
    }
    try stdout.interface.writeByte('\n');
    try stdout.flush();
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

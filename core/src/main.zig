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
    try core.protocol.writeResponse(request.value, &stdout.interface);
    try stdout.interface.writeByte('\n');
    try stdout.flush();
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

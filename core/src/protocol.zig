const std = @import("std");
const model = @import("model.zig");
const command = @import("command.zig");

pub const protocol_version = "1.0";
pub const max_message_bytes: usize = 16 * 1024 * 1024;
pub const Operation = model.Operation;

pub const Request = struct {
    protocol_version: []const u8,
    request_id: []const u8,
    operation: Operation,
    payload_schema: []const u8,
    payload: std.json.Value,
};

pub const ProtocolError = struct { code: []const u8, message: []const u8, retryable: bool };
pub const Failure = struct {
    protocol_version: []const u8 = protocol_version,
    request_id: []const u8,
    ok: bool = false,
    @"error": ProtocolError,
};
pub const ParsedRequest = std.json.Parsed(Request);

pub fn parseStrict(comptime T: type, allocator: std.mem.Allocator, bytes: []const u8) !std.json.Parsed(T) {
    if (bytes.len > max_message_bytes) return error.MessageTooLarge;
    return std.json.parseFromSlice(T, allocator, bytes, .{}) catch |err| return switch (err) {
        error.DuplicateField => error.DuplicateField,
        error.UnknownField => error.UnknownField,
        else => error.InvalidJson,
    };
}

pub fn parseRequest(allocator: std.mem.Allocator, bytes: []const u8) !ParsedRequest {
    var parsed = try parseStrict(Request, allocator, bytes);
    errdefer parsed.deinit();
    if (!std.mem.eql(u8, parsed.value.protocol_version, protocol_version)) return error.UnsupportedProtocolVersion;
    if (parsed.value.request_id.len == 0 or parsed.value.request_id.len > 128) return error.InvalidRequestId;
    if (!std.mem.eql(u8, parsed.value.payload_schema, "zintent.command/1")) return error.InvalidPayloadSchema;
    const payload = switch (parsed.value.payload) {
        .object => |o| o,
        else => return error.InvalidOperationPayload,
    };
    const op = payload.get("operation") orelse return error.InvalidOperationPayload;
    if (op != .string or !std.mem.eql(u8, op.string, @tagName(parsed.value.operation))) return error.OperationMismatch;
    var typed = std.json.parseFromValue(command.Command, allocator, parsed.value.payload, .{}) catch
        return error.InvalidOperationPayload;
    defer typed.deinit();
    typed.value.validate() catch return error.InvalidOperationPayload;
    return parsed;
}

pub fn capabilityInfo() struct {
    contract_version: []const u8,
    ok: bool,
    operation: []const u8,
    affected_ids: []const []const u8,
    findings: []const []const u8,
    data: struct { capabilities: []const []const u8, limits: struct { stream_bytes: usize, deadline_seconds: u8 } },
} {
    return .{
        .contract_version = "1.0.0",
        .ok = true,
        .operation = "protocol_info",
        .affected_ids = &.{},
        .findings = &.{},
        .data = .{
            .capabilities = &.{ "strict_json", "immutable_revisions", "approval_challenge", "workspace_listing", "draft_import", "verified_audit", "safe_recovery" },
            .limits = .{ .stream_bytes = max_message_bytes, .deadline_seconds = 10 },
        },
    };
}

pub fn writeResponse(request: Request, writer: *std.Io.Writer) !void {
    if (request.operation == .protocol_info) {
        return std.json.Stringify.value(.{
            .protocol_version = protocol_version,
            .request_id = request.request_id,
            .ok = true,
            .result_schema = "zintent.result/1",
            .result = capabilityInfo(),
        }, .{}, writer);
    }
    try std.json.Stringify.value(Failure{
        .request_id = request.request_id,
        .@"error" = .{ .code = "operation_unavailable", .message = "Operation is not implemented yet.", .retryable = false },
    }, .{}, writer);
}

test "strict protocol and operation equality" {
    const allocator = std.testing.allocator;
    var parsed = try parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"r1","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"protocol_info"}}
    );
    defer parsed.deinit();
    try std.testing.expectEqual(Operation.protocol_info, parsed.value.operation);
    try std.testing.expectError(error.OperationMismatch, parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"r1","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"show_intent"}}
    ));
}

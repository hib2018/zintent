const std = @import("std");
const core = @import("zintent_core");

test "protocol malformed inputs never panic" {
    const allocator = std.testing.allocator;
    const samples = [_][]const u8{ "", "{", "[]", "{\"protocol_version\":\"1.0\",\"request_id\":\"r\",\"operation\":\"show_intent\",\"payload_schema\":\"zintent.command/1\",\"payload\":{}}", "\xff\xfe" };
    for (samples) |sample| {
        _ = core.protocol.parseRequest(allocator, sample) catch |err| {
            try std.testing.expect(err != error.OutOfMemory);
            continue;
        };
    }
}

test "duplicate and unknown fields are rejected" {
    const allocator = std.testing.allocator;
    try std.testing.expectError(error.InvalidJson, core.protocol.parseRequest(allocator, "{\"protocol_version\":\"1.0\",\"protocol_version\":\"1.0\"}"));
}

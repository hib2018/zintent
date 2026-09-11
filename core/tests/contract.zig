const std = @import("std");
const core = @import("zintent_core");

test "strict request contract" {
    const allocator = std.testing.allocator;
    var request = try core.protocol.parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"00000000-0000-7000-8000-000000000001","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"protocol_info"}}
    );
    defer request.deinit();
    try std.testing.expectEqual(core.model.Operation.protocol_info, request.value.operation);
}

test "unknown envelope field and operation mismatch are rejected" {
    const allocator = std.testing.allocator;
    try std.testing.expectError(error.UnknownField, core.protocol.parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"r","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"protocol_info"},"extra":1}
    ));
    try std.testing.expectError(error.OperationMismatch, core.protocol.parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"r","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"show_intent"}}
    ));
}

test "duplicate and unknown nested command fields are rejected" {
    const allocator = std.testing.allocator;
    try std.testing.expectError(error.DuplicateField, core.protocol.parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"r","request_id":"r","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"protocol_info"}}
    ));
    try std.testing.expectError(error.InvalidOperationPayload, core.protocol.parseRequest(allocator,
        \\{"protocol_version":"1.0","request_id":"r","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"protocol_info","unknown":true}}
    ));
}

test "provenance external contract has strict typed representation" {
    const allocator = std.testing.allocator;
    var value = try core.protocol.parseStrict(core.model.Provenance, allocator,
        \\{"content_origin":"source","operation_actor":null,"operation_id":"op","operation_type":"fixture_import","revision_id":"rev","source_reference_ids":[]}
    );
    defer value.deinit();
    try std.testing.expectEqual(core.model.ContentOrigin.source, value.value.content_origin);
    try std.testing.expectError(error.UnknownField, core.protocol.parseStrict(core.model.Provenance, allocator,
        \\{"content_origin":"source","operation_actor":null,"operation_id":"op","operation_type":"fixture_import","revision_id":"rev","source_reference_ids":[],"unknown":true}
    ));
}

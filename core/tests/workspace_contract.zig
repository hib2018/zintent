const std = @import("std");
const core = @import("zintent_core");

test "workspace operations decode strictly and classify mutations" {
    const cases = [_]struct { json: []const u8, op: core.model.Operation, mutation: bool }{
        .{ .json = "{\"operation\":\"list_intents\",\"workspace_path\":\"/tmp/ws\"}", .op = .list_intents, .mutation = false },
        .{ .json = "{\"operation\":\"inspect_draft\",\"workspace_path\":\"/tmp/ws\",\"source_path\":\"/tmp/d.json\",\"actor\":{\"actor_id\":\"u\",\"identity_source\":\"os_user\"}}", .op = .inspect_draft, .mutation = false },
        .{ .json = "{\"operation\":\"import_draft\",\"workspace_path\":\"/tmp/ws\",\"source_path\":\"/tmp/d.json\",\"destination\":\"I-1\",\"import_token\":\"t\",\"operation_id\":\"o\",\"actor\":{\"actor_id\":\"u\",\"identity_source\":\"os_user\"},\"intent_path\":\"/tmp/ws/I-1\",\"expected_revision_id\":\"r\"}", .op = .import_draft, .mutation = true },
        .{ .json = "{\"operation\":\"list_revisions\",\"intent_path\":\"/tmp/i\"}", .op = .list_revisions, .mutation = false },
        .{ .json = "{\"operation\":\"inspect_revision\",\"intent_path\":\"/tmp/i\",\"revision_id\":\"r\"}", .op = .inspect_revision, .mutation = false },
        .{ .json = "{\"operation\":\"inspect_snapshot\",\"intent_path\":\"/tmp/i\",\"snapshot_id\":\"s\"}", .op = .inspect_snapshot, .mutation = false },
        .{ .json = "{\"operation\":\"recovery_status\",\"intent_path\":\"/tmp/i\"}", .op = .recovery_status, .mutation = false },
        .{ .json = "{\"operation\":\"cleanup_temporary_files\",\"intent_path\":\"/tmp/i\",\"expected_revision_id\":\"r\",\"expected_head_hash\":\"h\",\"recovery_token\":\"t\",\"candidate_ids\":[\"c\"],\"operation_id\":\"o\",\"actor\":{\"actor_id\":\"u\",\"identity_source\":\"os_user\"}}", .op = .cleanup_temporary_files, .mutation = true },
    };
    for (cases) |case| {
        var parsed = try core.protocol.parseStrict(core.command.Command, std.testing.allocator, case.json);
        defer parsed.deinit();
        try parsed.value.validate();
        try std.testing.expectEqual(case.op, parsed.value.operation);
        try std.testing.expectEqual(case.mutation, core.model.isMutation(case.op));
    }
    try std.testing.expectError(error.UnknownField, core.protocol.parseStrict(core.command.Command, std.testing.allocator, "{\"operation\":\"list_intents\",\"workspace_path\":\"x\",\"unknown\":true}"));
}

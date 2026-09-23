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
    } else if (request.value.operation == .diff_revisions) {
        try writeDiffResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .preview_edit) {
        try writePreviewResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .start_review or request.value.operation == .accept_item or request.value.operation == .edit_item or request.value.operation == .reject_item or request.value.operation == .add_comment or request.value.operation == .resolve_comment or request.value.operation == .withdraw_comment or request.value.operation == .complete_review) {
        try writeMutationResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .prepare_approval) {
        try writePrepareApprovalResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .approve_intent) {
        try writeApproveResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .list_revisions or request.value.operation == .inspect_revision or request.value.operation == .inspect_snapshot) {
        try writeAuditWorkspaceResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .recovery_status or request.value.operation == .cleanup_temporary_files) {
        try writeRecoveryWorkspaceResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .list_intents or request.value.operation == .inspect_draft or request.value.operation == .import_draft) {
        try writeWorkspaceImportResult(allocator, io, request.value, &stdout.interface);
    } else {
        try core.protocol.writeResponse(request.value, &stdout.interface);
    }
    try stdout.interface.writeByte('\n');
    try stdout.flush();
}

fn writeWorkspaceImportResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "Workspace payload must be an object."),
    };
    const workspace_path = stringField(payload, "workspace_path") orelse return writeFailure(io, request.request_id, "invalid_workspace", "workspace_path is required.");
    if (request.operation == .list_intents) {
        const discovery = core.workspace.discover(allocator, io, workspace_path) catch return writeFailure(io, request.request_id, "invalid_workspace", "Workspace root could not be opened.");
        var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try data.put(allocator, "entries", .{ .array = discovery.entries });
        try data.put(allocator, "findings", .{ .array = discovery.findings });
        return writeWorkspaceSuccess(allocator, request.request_id, "list_intents", .{ .object = data }, writer);
    }
    const source_path = stringField(payload, "source_path") orelse return writeFailure(io, request.request_id, "invalid_request", "source_path is required.");
    const source = core.store.readBytes(allocator, io, source_path, core.max_message_bytes) catch return writeFailure(io, request.request_id, "draft_invalid", "Draft source could not be read.");
    defer allocator.free(source);
    var validated = core.validation.parseRevision(allocator, source) catch return writeFailure(io, request.request_id, "draft_invalid", "Draft failed validation.");
    defer validated.deinit();
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, source, .{ .allocate = .alloc_always }) catch unreachable;
    defer parsed.deinit();
    const source_hash = core.hashing.sha256Hex(source);
    const proposed = try core.import.proposedId(allocator, &source_hash);
    defer allocator.free(proposed);
    const actor_id = actorID(payload) orelse return writeFailure(io, request.request_id, "missing_actor", "Human actor is required.");
    if (request.operation == .inspect_draft) {
        const destination = proposed;
        const destination_path = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ workspace_path, destination });
        defer allocator.free(destination_path);
        var collision = false;
        if (std.Io.Dir.cwd().openDir(io, destination_path, .{})) |dir_value| {
            var dir = dir_value;
            dir.close(io);
            collision = true;
        } else |_| {}
        const token_input = try std.fmt.allocPrint(allocator, "{s}\x00{s}\x00{s}\x00{s}", .{ &source_hash, destination, actor_id, workspace_path });
        defer allocator.free(token_input);
        const token = core.hashing.sha256Hex(token_input);
        const registry = try std.fmt.allocPrint(allocator, "{s}/.capabilities", .{workspace_path});
        defer allocator.free(registry);
        core.store.issueCapability(allocator, io, registry, .{ .token_id = &token, .intent_id = proposed, .expected_revision_id = destination, .actor_id = actor_id, .payload_hash = &source_hash, .expires_at_unix = unixNow(io) + 600 }) catch |err| switch (err) {
            error.DestinationExists => {},
            else => return writeFailure(io, request.request_id, "persistence_failure", "Import preview could not be issued."),
        };
        var findings = std.json.Array.init(allocator);
        if (collision) {
            var finding = try std.json.ObjectMap.init(allocator, &.{}, &.{});
            try finding.put(allocator, "code", .{ .string = "destination_conflict" });
            try finding.put(allocator, "severity", .{ .string = "blocking" });
            try finding.put(allocator, "message", .{ .string = "Proposed destination already exists." });
            try findings.append(.{ .object = finding });
        }
        var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try data.put(allocator, "source_hash", .{ .string = &source_hash });
        try data.put(allocator, "proposed_intent_id", .{ .string = proposed });
        try data.put(allocator, "proposed_destination", .{ .string = destination });
        try data.put(allocator, "import_token", .{ .string = &token });
        try data.put(allocator, "expires_in_seconds", .{ .integer = 600 });
        try data.put(allocator, "findings", .{ .array = findings });
        return writeWorkspaceSuccess(allocator, request.request_id, "inspect_draft", .{ .object = data }, writer);
    }
    const destination = stringField(payload, "destination") orelse return writeFailure(io, request.request_id, "invalid_request", "destination is required.");
    const token = stringField(payload, "import_token") orelse return writeFailure(io, request.request_id, "invalid_request", "import_token is required.");
    const operation_id = stringField(payload, "operation_id") orelse return writeFailure(io, request.request_id, "invalid_request", "operation_id is required.");
    if (!core.store.validStableId(operation_id)) return writeFailure(io, request.request_id, "path_escape", "operation_id must be a stable ID.");
    const marker_path = try std.fmt.allocPrint(allocator, "{s}/.imports/{s}.json", .{ workspace_path, operation_id });
    defer allocator.free(marker_path);
    if (core.store.readBytes(allocator, io, marker_path, 16 * 1024)) |marker| {
        defer allocator.free(marker);
        var prior = std.json.parseFromSlice(std.json.Value, allocator, marker, .{}) catch return writeFailure(io, request.request_id, "operation_id_conflict", "Import operation record is corrupt.");
        defer prior.deinit();
        const prior_destination = stringFromValue(prior.value, "destination") orelse "";
        const prior_token = stringFromValue(prior.value, "import_token") orelse "";
        if (!std.mem.eql(u8, prior_destination, destination) or !std.mem.eql(u8, prior_token, token)) return writeFailure(io, request.request_id, "operation_id_conflict", "Operation ID was reused with different import content.");
        var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try data.put(allocator, "intent_id", .{ .string = proposed });
        try data.put(allocator, "destination", .{ .string = destination });
        try data.put(allocator, "source_preserved", .{ .bool = true });
        try data.put(allocator, "identical_retry", .{ .bool = true });
        return writeWorkspaceSuccess(allocator, request.request_id, "import_draft", .{ .object = data }, writer);
    } else |_| {}
    const registry = try std.fmt.allocPrint(allocator, "{s}/.capabilities", .{workspace_path});
    defer allocator.free(registry);
    var capability = core.store.consumeCapabilityRecord(allocator, io, registry, token, unixNow(io)) catch return writeFailure(io, request.request_id, "import_preview_expired", "Import preview is expired, consumed, or unknown.");
    defer capability.deinit();
    if (!std.mem.eql(u8, capability.value.actor_id, actor_id) or !std.mem.eql(u8, capability.value.expected_revision_id, destination) or !std.mem.eql(u8, capability.value.payload_hash, &source_hash)) return writeFailure(io, request.request_id, "draft_invalid", "Draft source, destination, or actor changed after preview.");
    var root = switch (parsed.value) {
        .object => |value| value,
        else => unreachable,
    };
    const revision_payload = root.getPtr("revision_payload") orelse return writeFailure(io, request.request_id, "draft_invalid", "revision_payload is required.");
    var draft_payload = switch (revision_payload.*) {
        .object => |value| value,
        else => return writeFailure(io, request.request_id, "draft_invalid", "revision_payload must be an object."),
    };
    try draft_payload.put(allocator, "intent_id", .{ .string = proposed });
    revision_payload.* = .{ .object = draft_payload };
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    const revision_id = stringFromObject(root, "revision_id") orelse return writeFailure(io, request.request_id, "draft_invalid", "revision_id is required.");
    const lifecycle = stringFromObject(draft_payload, "lifecycle_state") orelse "draft";
    core.import.importAtomic(allocator, io, workspace_path, destination, operation_id, output.written(), proposed, revision_id, lifecycle) catch |err| return writeFailure(io, request.request_id, if (err == error.DestinationExists) "destination_conflict" else if (err == error.PathEscape) "path_escape" else "persistence_failure", "Draft could not be imported atomically.");
    var marker_output: std.Io.Writer.Allocating = .init(allocator);
    defer marker_output.deinit();
    try std.json.Stringify.value(.{ .destination = destination, .import_token = token }, .{}, &marker_output.writer);
    core.store.publishAtomic(allocator, io, marker_path, marker_output.written(), true) catch return writeFailure(io, request.request_id, "persistence_failure", "Import operation record could not be published.");
    var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try data.put(allocator, "intent_id", .{ .string = proposed });
    try data.put(allocator, "destination", .{ .string = destination });
    try data.put(allocator, "source_preserved", .{ .bool = true });
    return writeWorkspaceSuccess(allocator, request.request_id, "import_draft", .{ .object = data }, writer);
}

fn writeAuditWorkspaceResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "Audit payload must be an object."),
    };
    const intent_path = stringField(payload, "intent_path") orelse return writeFailure(io, request.request_id, "invalid_request", "Intent path is required.");
    var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    if (request.operation == .list_revisions) {
        const chain = core.store.loadRevisionChain(allocator, io, intent_path) catch return writeFailure(io, request.request_id, "integrity_failure", "Reachable revision chain is invalid.");
        const orphans = core.store.findOrphans(allocator, io, intent_path, chain) catch return writeFailure(io, request.request_id, "integrity_failure", "Revision directory could not be inspected.");
        try data.put(allocator, "reachable", .{ .array = chain });
        try data.put(allocator, "orphans", .{ .array = orphans });
    } else if (request.operation == .inspect_revision) {
        const id = stringField(payload, "revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "revision_id is required.");
        const bytes = core.store.readRevisionById(allocator, io, intent_path, id) catch |err| return writeFailure(io, request.request_id, if (err == error.PathEscape) "path_escape" else "revision_not_found", "Revision could not be verified.");
        defer allocator.free(bytes);
        var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "integrity_failure", "Revision JSON is invalid.");
        defer parsed.deinit();
        try data.put(allocator, "revision", parsed.value);
    } else {
        const id = stringField(payload, "snapshot_id") orelse return writeFailure(io, request.request_id, "invalid_request", "snapshot_id is required.");
        if (!core.store.validStableId(id)) return writeFailure(io, request.request_id, "path_escape", "Snapshot ID must not contain a path.");
        const path = try std.fmt.allocPrint(allocator, "{s}/snapshots/{s}.json", .{ intent_path, id });
        defer allocator.free(path);
        const bytes = core.store.readBytes(allocator, io, path, core.max_message_bytes) catch return writeFailure(io, request.request_id, "snapshot_not_found", "Snapshot was not found.");
        defer allocator.free(bytes);
        var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "integrity_failure", "Snapshot JSON is invalid.");
        defer parsed.deinit();
        const stored = stringFromValue(parsed.value, "snapshot_id") orelse return writeFailure(io, request.request_id, "integrity_failure", "Snapshot identity is missing.");
        if (!std.mem.eql(u8, stored, id)) return writeFailure(io, request.request_id, "integrity_failure", "Snapshot identity does not match filename.");
        const snapshot_object = switch (parsed.value) {
            .object => |value| value,
            else => return writeFailure(io, request.request_id, "integrity_failure", "Snapshot must be an object."),
        };
        const approved = snapshot_object.get("approved_content") orelse return writeFailure(io, request.request_id, "integrity_failure", "Approved content is missing.");
        const approval_value = snapshot_object.get("approval") orelse return writeFailure(io, request.request_id, "integrity_failure", "Approval linkage is missing.");
        const approval = switch (approval_value) {
            .object => |value| value,
            else => return writeFailure(io, request.request_id, "integrity_failure", "Approval linkage is invalid."),
        };
        const expected_hash = stringFromObject(approval, "approved_content_hash") orelse return writeFailure(io, request.request_id, "integrity_failure", "Approved content hash is missing.");
        const canonical = try core.hashing.canonicalize(allocator, approved);
        defer allocator.free(canonical);
        const actual_hash = core.hashing.sha256Hex(canonical);
        if (!std.mem.eql(u8, expected_hash, &actual_hash)) return writeFailure(io, request.request_id, "integrity_failure", "Approved content hash does not match.");
        const revision_id = stringFromObject(approval, "approved_revision_id") orelse return writeFailure(io, request.request_id, "integrity_failure", "Approved revision linkage is missing.");
        const revision_bytes = core.store.readRevisionById(allocator, io, intent_path, revision_id) catch return writeFailure(io, request.request_id, "integrity_failure", "Approved revision linkage is broken.");
        defer allocator.free(revision_bytes);
        try data.put(allocator, "snapshot", parsed.value);
        try data.put(allocator, "verified", .{ .bool = true });
    }
    try writeWorkspaceSuccess(allocator, request.request_id, @tagName(request.operation), .{ .object = data }, writer);
}

fn writeRecoveryWorkspaceResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "Recovery payload must be an object."),
    };
    const intent_path = stringField(payload, "intent_path") orelse return writeFailure(io, request.request_id, "invalid_request", "Intent path is required.");
    const verified = core.store.loadVerifiedRevision(allocator, io, intent_path) catch return writeFailure(io, request.request_id, "integrity_failure", "HEAD is invalid.");
    defer allocator.free(verified.bytes);
    if (request.operation == .recovery_status) {
        const candidates = core.store.listTemporaryCandidates(allocator, io, intent_path) catch return writeFailure(io, request.request_id, "integrity_failure", "Recovery candidates could not be observed.");
        const canonical = try core.hashing.canonicalize(allocator, .{ .array = candidates });
        defer allocator.free(canonical);
        const observation_hash = core.hashing.sha256Hex(canonical);
        const token_input = try std.fmt.allocPrint(allocator, "{s}\x00{s}", .{ verified.head.current_revision_id, &observation_hash });
        defer allocator.free(token_input);
        const token = core.hashing.sha256Hex(token_input);
        const registry = try capabilityRegistry(allocator, io, intent_path);
        defer allocator.free(registry);
        core.store.issueCapability(allocator, io, registry, .{ .token_id = &token, .intent_id = verified.head.intent_id, .expected_revision_id = verified.head.current_revision_id, .actor_id = "", .payload_hash = &observation_hash, .expires_at_unix = unixNow(io) + 600 }) catch |err| switch (err) {
            error.DestinationExists => {},
            else => return writeFailure(io, request.request_id, "persistence_failure", "Recovery token could not be issued."),
        };
        var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try data.put(allocator, "observed_revision_id", .{ .string = verified.head.current_revision_id });
        try data.put(allocator, "observed_head_hash", .{ .string = verified.head.current_revision_hash });
        try data.put(allocator, "temporary_candidates", .{ .array = candidates });
        try data.put(allocator, "recovery_token", .{ .string = &token });
        try data.put(allocator, "expires_in_seconds", .{ .integer = 600 });
        return writeWorkspaceSuccess(allocator, request.request_id, "recovery_status", .{ .object = data }, writer);
    }
    const lock_path = try std.fmt.allocPrint(allocator, "{s}/.lock", .{intent_path});
    defer allocator.free(lock_path);
    const held = core.store.acquireLock(io, lock_path) catch return writeFailure(io, request.request_id, "intent_locked", "Intent is locked by another mutation.");
    defer held.release();
    const locked_verified = core.store.loadVerifiedRevision(allocator, io, intent_path) catch return writeFailure(io, request.request_id, "integrity_failure", "HEAD is invalid under lock.");
    defer allocator.free(locked_verified.bytes);
    const expected = stringField(payload, "expected_revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id is required.");
    const expected_head = stringField(payload, "expected_head_hash") orelse return writeFailure(io, request.request_id, "invalid_request", "expected_head_hash is required.");
    if (!std.mem.eql(u8, expected, locked_verified.head.current_revision_id) or !std.mem.eql(u8, expected_head, locked_verified.head.current_revision_hash)) return writeFailure(io, request.request_id, "recovery_observation_stale", "HEAD changed after recovery observation.");
    const token = stringField(payload, "recovery_token") orelse return writeFailure(io, request.request_id, "invalid_request", "recovery_token is required.");
    const registry = try capabilityRegistry(allocator, io, intent_path);
    defer allocator.free(registry);
    var capability = core.store.consumeCapabilityRecord(allocator, io, registry, token, unixNow(io)) catch return writeFailure(io, request.request_id, "recovery_observation_stale", "Recovery token is expired or consumed.");
    defer capability.deinit();
    const candidates = core.store.listTemporaryCandidates(allocator, io, intent_path) catch return writeFailure(io, request.request_id, "integrity_failure", "Recovery candidates changed.");
    const canonical = try core.hashing.canonicalize(allocator, .{ .array = candidates });
    defer allocator.free(canonical);
    const observation_hash = core.hashing.sha256Hex(canonical);
    if (!std.mem.eql(u8, capability.value.payload_hash, &observation_hash)) return writeFailure(io, request.request_id, "cleanup_target_changed", "Candidate set changed after observation.");
    const ids_value = payload.get("candidate_ids") orelse return writeFailure(io, request.request_id, "invalid_request", "candidate_ids are required.");
    const ids_array = switch (ids_value) {
        .array => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "candidate_ids must be an array."),
    };
    var ids = std.ArrayList([]const u8).empty;
    for (ids_array.items) |value| {
        if (value != .string) return writeFailure(io, request.request_id, "invalid_request", "candidate ID must be a string.");
        try ids.append(allocator, value.string);
    }
    const removed = core.store.cleanupSelectedTemporaryFiles(io, intent_path, ids.items) catch return writeFailure(io, request.request_id, "cleanup_target_changed", "Selected cleanup target is not an unchanged temporary file.");
    var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try data.put(allocator, "removed_count", .{ .integer = @intCast(removed) });
    return writeWorkspaceSuccess(allocator, request.request_id, "cleanup_temporary_files", .{ .object = data }, writer);
}

fn writeWorkspaceSuccess(allocator: std.mem.Allocator, request_id: []const u8, operation: []const u8, data: std.json.Value, writer: *std.Io.Writer) !void {
    _ = allocator;
    try std.json.Stringify.value(.{ .protocol_version = core.protocol.protocol_version, .request_id = request_id, .ok = true, .result_schema = "zintent.result/1", .result = .{ .contract_version = "1.0.0", .ok = true, .operation = operation, .affected_ids = &.{}, .findings = &.{}, .data = data } }, .{}, writer);
}

fn writePreviewResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Mutation payload is not an object."),
    };
    const path_value = payload.get("intent_path") orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent path is required.");
    const path = switch (path_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent path must be a string."),
    };
    const bytes = readIntentBytes(allocator, io, path) catch |err| switch (err) {
        error.IntegrityFailure => return writeFailure(io, request.request_id, "integrity_failure", "HEAD does not match the selected revision."),
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read."),
    };
    defer allocator.free(bytes);
    var decoded = std.json.parseFromSlice(core.model.Revision, allocator, bytes, .{}) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact does not match the revision model.");
    defer decoded.deinit();
    _ = core.transition.next(decoded.value.revision_payload.lifecycle_state, .edit_item) catch return writeFailure(io, request.request_id, "invalid_transition", "Edit is not allowed from the current lifecycle.");
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    const root = switch (parsed.value) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact must be an object."),
    };
    const expected = payload.get("expected_revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id is required.");
    const expected_revision = switch (expected) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id must be a string."),
    };
    const current_revision = if (root.get("revision_id")) |value| switch (value) {
        .string => |id| id,
        else => "",
    } else if (root.get("current_revision_id")) |value| switch (value) {
        .string => |id| id,
        else => "",
    } else "";
    core.store.checkExpectedRevision(expected_revision, current_revision) catch return writeFailure(io, request.request_id, "stale_revision", "Expected revision does not match current revision.");
    const item_id_value = payload.get("item_id") orelse return writeFailure(io, request.request_id, "invalid_request", "item_id is required.");
    const item_id = switch (item_id_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "item_id must be a string."),
    };
    const proposed_value = payload.get("statement") orelse return writeFailure(io, request.request_id, "invalid_request", "statement is required.");
    const proposed = switch (proposed_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "statement must be a string."),
    };
    if (proposed.len == 0) return writeFailure(io, request.request_id, "invalid_request", "Edited statement must not be empty.");
    const revision_payload = root.get("revision_payload") orelse return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload is required.");
    const payload_object = switch (revision_payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload must be an object."),
    };
    const items_value = payload_object.get("items") orelse return writeFailure(io, request.request_id, "invalid_artifact", "items are required.");
    const items = switch (items_value) {
        .array => |array| array,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "items must be an array."),
    };
    var before: []const u8 = "";
    for (items.items) |item| {
        const object = switch (item) {
            .object => |value| value,
            else => continue,
        };
        const id = object.get("item_id") orelse continue;
        if (id == .string and std.mem.eql(u8, id.string, item_id)) {
            const statement = object.get("statement") orelse continue;
            before = switch (statement) {
                .string => |value| value,
                else => "",
            };
            break;
        }
    }
    if (before.len == 0) return writeFailure(io, request.request_id, "invalid_artifact", "Target item was not found.");
    const token_input = try std.fmt.allocPrint(allocator, "{s}\x00{s}\x00{s}\x00{s}", .{ expected_revision, item_id, before, proposed });
    defer allocator.free(token_input);
    const token = core.hashing.sha256Hex(token_input);
    const actor_id = actorID(payload) orelse return writeFailure(io, request.request_id, "missing_actor", "Human actor is required.");
    const intent_id = intentID(payload_object) orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent ID is required.");
    const registry = try capabilityRegistry(allocator, io, path);
    defer allocator.free(registry);
    core.store.issueCapability(allocator, io, registry, .{
        .token_id = &token,
        .intent_id = intent_id,
        .expected_revision_id = expected_revision,
        .actor_id = actor_id,
        .payload_hash = &token,
        .expires_at_unix = unixNow(io) + 600,
    }) catch |err| switch (err) {
        error.DestinationExists => {},
        else => return writeFailure(io, request.request_id, "persistence_failure", "Edit preview capability could not be issued."),
    };
    try std.json.Stringify.value(.{ .protocol_version = core.protocol.protocol_version, .request_id = request.request_id, .ok = true, .result_schema = "zintent.result/1", .result = .{ .contract_version = "1.0.0", .ok = true, .operation = "preview_edit", .affected_ids = &.{item_id}, .findings = &.{}, .data = .{ .preview = .{ .item_id = item_id, .before_statement = before, .after_statement = proposed, .preview_token = &token } } } }, .{}, writer);
}

fn writeMutationResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Mutation payload is not an object."),
    };
    const path_value = payload.get("intent_path") orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent path is required.");
    const path = switch (path_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent path must be a string."),
    };
    var held_lock: ?core.store.Lock = null;
    var lock_path: ?[]u8 = null;
    if (isIntentDirectory(io, path)) {
        lock_path = try std.fmt.allocPrint(allocator, "{s}/.lock", .{path});
        held_lock = core.store.acquireLock(io, lock_path.?) catch return writeFailure(io, request.request_id, "persistence_failure", "Intent is locked by another process.");
    }
    defer {
        if (held_lock) |lock| lock.release();
        if (lock_path) |value| allocator.free(value);
    }
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
    var decoded = std.json.parseFromSlice(core.model.Revision, allocator, bytes, .{}) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact does not match the revision model.");
    defer decoded.deinit();
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    var root = switch (parsed.value) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact must be an object."),
    };
    const expected = payload.get("expected_revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id is required.");
    const expected_revision = switch (expected) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "expected_revision_id must be a string."),
    };
    const current_revision = if (root.get("revision_id")) |value| switch (value) {
        .string => |id| id,
        else => "",
    } else if (root.get("current_revision_id")) |value| switch (value) {
        .string => |id| id,
        else => "",
    } else "";
    const incoming_digest = try payloadDigest(allocator, request.payload);
    if (root.get("operation_id")) |prior_operation| if (prior_operation == .string and std.mem.eql(u8, prior_operation.string, operation_idValue(payload) orelse "")) {
        const prior_digest = if (root.get("operation")) |operation| if (operation == .object) if (operation.object.get("content_digest")) |digest| if (digest == .string) digest.string else "" else "" else "" else "";
        if (!std.mem.eql(u8, prior_digest, &incoming_digest)) return writeFailure(io, request.request_id, "operation_id_conflict", "Operation ID was already used with different content.");
        try writeMutationEnvelope(request, parsed.value, writer);
        return;
    };
    core.store.checkExpectedRevision(expected_revision, current_revision) catch return writeFailure(io, request.request_id, "stale_revision", "Expected revision does not match current revision.");
    const operation_id_value = payload.get("operation_id") orelse return writeFailure(io, request.request_id, "invalid_request", "operation_id is required.");
    const operation_id = switch (operation_id_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "operation_id must be a string."),
    };
    const revision_id = operation_id;
    try root.put(allocator, "parent_revision_id", .{ .string = current_revision });
    try root.put(allocator, "operation_id", .{ .string = operation_id });
    if (payload.get("actor")) |actor| try root.put(allocator, "actor", actor);
    var operation_record = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try operation_record.put(allocator, "type", .{ .string = @tagName(request.operation) });
    try operation_record.put(allocator, "content_digest", .{ .string = &incoming_digest });
    var target_ids = std.json.Array.init(allocator);
    switch (request.operation) {
        .accept_item, .edit_item, .reject_item => if (stringField(payload, "item_id")) |id| try target_ids.append(.{ .string = id }),
        .add_comment => try target_ids.append(.{ .string = operation_id }),
        .resolve_comment, .withdraw_comment => if (stringField(payload, "comment_id")) |id| try target_ids.append(.{ .string = id }),
        else => {},
    }
    try operation_record.put(allocator, "target_ids", .{ .array = target_ids });
    try root.put(allocator, "operation", .{ .object = operation_record });
    const revision_payload = root.getPtr("revision_payload") orelse return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload is required.");
    var payload_object = switch (revision_payload.*) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload must be an object."),
    };
    try normalizeItemInclusion(allocator, payload_object);
    var next_lifecycle = core.transition.next(decoded.value.revision_payload.lifecycle_state, request.operation) catch
        return writeFailure(io, request.request_id, "invalid_transition", "Operation is not allowed from the current lifecycle.");
    if (request.operation == .complete_review)
        next_lifecycle = reviewCompletionLifecycle(payload_object) catch |err| return writeFailure(io, request.request_id, "approval_ineligible", approvalErrorMessage(err));
    try payload_object.put(allocator, "lifecycle_state", .{ .string = @tagName(next_lifecycle) });
    if (payload.get("item_id")) |item_id_value| {
        const item_id = switch (item_id_value) {
            .string => |value| value,
            else => return writeFailure(io, request.request_id, "invalid_request", "item_id must be a string."),
        };
        const items_value = payload_object.getPtr("items") orelse return writeFailure(io, request.request_id, "invalid_artifact", "items are required.");
        const items = switch (items_value.*) {
            .array => |array| array,
            else => return writeFailure(io, request.request_id, "invalid_artifact", "items must be an array."),
        };
        var found = false;
        for (items.items, 0..) |item, item_index| {
            const item_value = item;
            var item_object = switch (item_value) {
                .object => |object| object,
                else => continue,
            };
            const id_value = item_object.get("item_id") orelse continue;
            if (id_value == .string and std.mem.eql(u8, id_value.string, item_id)) {
                found = true;
                switch (request.operation) {
                    .accept_item => {
                        try item_object.put(allocator, "review_status", .{ .string = "accepted" });
                        try item_object.put(allocator, "included_in_approval", .{ .bool = true });
                        try item_object.put(allocator, "rationale", .null);
                    },
                    .reject_item => {
                        const rationale = stringField(payload, "rationale") orelse return writeFailure(io, request.request_id, "invalid_request", "A rejection rationale is required.");
                        if (rationale.len == 0) return writeFailure(io, request.request_id, "invalid_request", "A rejection rationale is required.");
                        try item_object.put(allocator, "review_status", .{ .string = "rejected" });
                        try item_object.put(allocator, "included_in_approval", .{ .bool = false });
                        try item_object.put(allocator, "rationale", .{ .string = rationale });
                    },
                    .edit_item => {
                        if (payload.get("statement")) |statement| {
                            const token_value = payload.get("preview_token") orelse return writeFailure(io, request.request_id, "invalid_confirmation", "preview_token is required.");
                            const token = switch (token_value) {
                                .string => |value| value,
                                else => return writeFailure(io, request.request_id, "invalid_confirmation", "preview_token must be a string."),
                            };
                            const before_value = item_object.get("statement") orelse return writeFailure(io, request.request_id, "invalid_artifact", "Target item has no statement.");
                            const before = switch (before_value) {
                                .string => |value| value,
                                else => return writeFailure(io, request.request_id, "invalid_artifact", "Target item statement must be a string."),
                            };
                            const after = switch (statement) {
                                .string => |value| value,
                                else => return writeFailure(io, request.request_id, "invalid_request", "statement must be a string."),
                            };
                            const token_input = try std.fmt.allocPrint(allocator, "{s}\x00{s}\x00{s}\x00{s}", .{ expected_revision, item_id, before, after });
                            defer allocator.free(token_input);
                            const derived_token = core.hashing.sha256Hex(token_input);
                            if (!std.mem.eql(u8, token, &derived_token)) return writeFailure(io, request.request_id, "invalid_confirmation", "preview_token does not match the requested edit.");
                            const registry = try capabilityRegistry(allocator, io, path);
                            defer allocator.free(registry);
                            var capability = core.store.consumeCapabilityRecord(allocator, io, registry, token, unixNow(io)) catch return writeFailure(io, request.request_id, "invalid_confirmation", "preview_token is expired, consumed, or unknown.");
                            defer capability.deinit();
                            const actor_id = actorID(payload) orelse return writeFailure(io, request.request_id, "missing_actor", "Human actor is required.");
                            const intent_id = intentID(payload_object) orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent ID is required.");
                            if (!std.mem.eql(u8, capability.value.intent_id, intent_id) or !std.mem.eql(u8, capability.value.expected_revision_id, expected_revision) or !std.mem.eql(u8, capability.value.actor_id, actor_id) or !std.mem.eql(u8, capability.value.payload_hash, token)) return writeFailure(io, request.request_id, "invalid_confirmation", "preview_token is not bound to this edit.");
                            if (after.len == 0) return writeFailure(io, request.request_id, "invalid_request", "Edited statement must not be empty.");
                            try item_object.put(allocator, "statement", statement);
                            try item_object.put(allocator, "review_status", .{ .string = "edited" });
                            try item_object.put(allocator, "included_in_approval", .{ .bool = true });
                            try item_object.put(allocator, "rationale", .null);
                        }
                    },
                    else => {},
                }
                if (request.operation == .accept_item or request.operation == .edit_item or request.operation == .reject_item) {
                    try setHumanProvenance(allocator, &item_object, payload.get("actor") orelse return writeFailure(io, request.request_id, "missing_actor", "Human actor is required."), operation_id, @tagName(request.operation), revision_id);
                }
                items.items[item_index] = .{ .object = item_object };
            }
        }
        if (!found) return writeFailure(io, request.request_id, "invalid_artifact", "Target item was not found.");
    }
    if (request.operation == .add_comment) {
        const item_id_value = payload.get("item_id") orelse return writeFailure(io, request.request_id, "invalid_request", "item_id is required.");
        const item_id = switch (item_id_value) {
            .string => |value| value,
            else => return writeFailure(io, request.request_id, "invalid_request", "item_id must be a string."),
        };
        const body_value = payload.get("body") orelse return writeFailure(io, request.request_id, "invalid_request", "body is required.");
        const body = switch (body_value) {
            .string => |value| value,
            else => return writeFailure(io, request.request_id, "invalid_request", "body must be a string."),
        };
        if (body.len == 0) return writeFailure(io, request.request_id, "invalid_request", "Comment body must not be empty.");
        if (!payloadHasItem(payload_object, item_id)) return writeFailure(io, request.request_id, "invalid_artifact", "Comment target item was not found.");
        const actor = payload.get("actor") orelse return writeFailure(io, request.request_id, "missing_actor", "actor is required.");
        const comment_operation_id_value = payload.get("operation_id") orelse return writeFailure(io, request.request_id, "invalid_request", "operation_id is required.");
        const comment_operation_id = switch (comment_operation_id_value) {
            .string => |value| value,
            else => return writeFailure(io, request.request_id, "invalid_request", "operation_id must be a string."),
        };
        var comment = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try comment.put(allocator, "comment_id", .{ .string = comment_operation_id });
        try comment.put(allocator, "target_item_id", .{ .string = item_id });
        try comment.put(allocator, "body", .{ .string = body });
        try comment.put(allocator, "author", actor);
        try comment.put(allocator, "status", .{ .string = "open" });
        try comment.put(allocator, "created_revision_id", .{ .string = revision_id });
        const comments_value = payload_object.getPtr("comments") orelse return writeFailure(io, request.request_id, "invalid_artifact", "comments are required.");
        const comments = switch (comments_value.*) {
            .array => |*array| array,
            else => return writeFailure(io, request.request_id, "invalid_artifact", "comments must be an array."),
        };
        try comments.append(.{ .object = comment });
    }
    if (request.operation == .resolve_comment or request.operation == .withdraw_comment) {
        const reason = stringField(payload, "reason") orelse return writeFailure(io, request.request_id, "invalid_request", "A comment closure reason is required.");
        if (reason.len == 0) return writeFailure(io, request.request_id, "invalid_request", "A comment closure reason is required.");
        const comment_id_value = payload.get("comment_id") orelse return writeFailure(io, request.request_id, "invalid_request", "comment_id is required.");
        const comment_id = switch (comment_id_value) {
            .string => |value| value,
            else => return writeFailure(io, request.request_id, "invalid_request", "comment_id must be a string."),
        };
        const comments_value = payload_object.getPtr("comments") orelse return writeFailure(io, request.request_id, "invalid_artifact", "comments are required.");
        const comments = switch (comments_value.*) {
            .array => |array| array,
            else => return writeFailure(io, request.request_id, "invalid_artifact", "comments must be an array."),
        };
        var found_comment = false;
        for (comments.items, 0..) |comment_value, index| {
            var comment_object = switch (comment_value) {
                .object => |object| object,
                else => continue,
            };
            const id = comment_object.get("comment_id") orelse continue;
            if (id == .string and std.mem.eql(u8, id.string, comment_id)) {
                found_comment = true;
                const status = stringFromObject(comment_object, "status") orelse "";
                if (!std.mem.eql(u8, status, "open")) return writeFailure(io, request.request_id, "invalid_transition", "Only an open comment may be closed.");
                try comment_object.put(allocator, "status", .{ .string = if (request.operation == .resolve_comment) "resolved" else "withdrawn" });
                try comment_object.put(allocator, "closure_reason", .{ .string = reason });
                try comment_object.put(allocator, "closed_revision_id", .{ .string = revision_id });
                if (payload.get("resolution_revision_id")) |resolution| try comment_object.put(allocator, "resolution_revision_id", resolution);
                comments.items[index] = .{ .object = comment_object };
            }
        }
        if (!found_comment) return writeFailure(io, request.request_id, "invalid_artifact", "Target comment was not found.");
    }
    try root.put(allocator, "revision_id", .{ .string = revision_id });
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    const revision_digest = core.hashing.sha256Hex(output.written());
    try root.put(allocator, "revision_hash", .{ .string = &revision_digest });
    output.clearRetainingCapacity();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    var validated = core.validation.parseRevision(allocator, output.written()) catch return writeFailure(io, request.request_id, "invalid_artifact", "Mutation would violate Intent invariants.");
    defer validated.deinit();
    try publishMutation(allocator, io, path, revision_id, output.written(), root, held_lock != null, null);
    var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try data.put(allocator, "intent", parsed.value);
    var result = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try result.put(allocator, "contract_version", .{ .string = "1.0.0" });
    try result.put(allocator, "ok", .{ .bool = true });
    try result.put(allocator, "operation", .{ .string = @tagName(request.operation) });
    try result.put(allocator, "affected_ids", .{ .array = target_ids });
    try result.put(allocator, "findings", .{ .array = std.json.Array.init(allocator) });
    try result.put(allocator, "data", .{ .object = data });
    var envelope = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try envelope.put(allocator, "protocol_version", .{ .string = core.protocol.protocol_version });
    try envelope.put(allocator, "request_id", .{ .string = request.request_id });
    try envelope.put(allocator, "ok", .{ .bool = true });
    try envelope.put(allocator, "result_schema", .{ .string = "zintent.result/1" });
    try envelope.put(allocator, "result", .{ .object = result });
    try std.json.Stringify.value(std.json.Value{ .object = envelope }, .{}, writer);
}

fn payloadHasItem(payload: std.json.ObjectMap, item_id: []const u8) bool {
    const items_value = payload.get("items") orelse return false;
    const items = switch (items_value) {
        .array => |array| array,
        else => return false,
    };
    for (items.items) |item_value| {
        const item = switch (item_value) {
            .object => |object| object,
            else => continue,
        };
        if (stringFromObject(item, "item_id")) |id| if (std.mem.eql(u8, id, item_id)) return true;
    }
    return false;
}

fn normalizeItemInclusion(allocator: std.mem.Allocator, payload: std.json.ObjectMap) !void {
    const items_value = payload.getPtr("items") orelse return;
    const items = switch (items_value.*) {
        .array => |*array| array,
        else => return,
    };
    for (items.items, 0..) |item_value, index| {
        var item = switch (item_value) {
            .object => |object| object,
            else => continue,
        };
        const status = stringFromObject(item, "review_status") orelse "unreviewed";
        try item.put(allocator, "included_in_approval", .{ .bool = !std.mem.eql(u8, status, "rejected") });
        if (!std.mem.eql(u8, status, "rejected")) try item.put(allocator, "rationale", .null);
        items.items[index] = .{ .object = item };
    }
}

fn setHumanProvenance(allocator: std.mem.Allocator, item: *std.json.ObjectMap, actor: std.json.Value, operation_id: []const u8, operation_type: []const u8, revision_id: []const u8) !void {
    var provenance = if (item.getPtr("provenance")) |value| switch (value.*) {
        .object => |object| object,
        else => try std.json.ObjectMap.init(allocator, &.{}, &.{}),
    } else try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try provenance.put(allocator, "content_origin", .{ .string = "human" });
    try provenance.put(allocator, "operation_actor", actor);
    try provenance.put(allocator, "operation_id", .{ .string = operation_id });
    try provenance.put(allocator, "operation_type", .{ .string = operation_type });
    try provenance.put(allocator, "revision_id", .{ .string = revision_id });
    try item.put(allocator, "provenance", .{ .object = provenance });
}

fn reviewCompletionLifecycle(payload: std.json.ObjectMap) !core.model.Lifecycle {
    const items_value = payload.get("items") orelse return error.NoIncludedItems;
    const items = switch (items_value) {
        .array => |value| value,
        else => return error.NoIncludedItems,
    };
    if (items.items.len == 0) return error.NoIncludedItems;
    var included: usize = 0;
    var rejected: usize = 0;
    for (items.items) |item| {
        if (item != .object) return error.UnreviewedItem;
        const object = item.object;
        const status = object.get("review_status") orelse return error.UnreviewedItem;
        if (status == .string and std.mem.eql(u8, status.string, "rejected")) {
            rejected += 1;
            continue;
        }
        if (object.get("included_in_approval")) |value| if (value == .bool and !value.bool) continue;
        included += 1;
        if (status != .string or (!std.mem.eql(u8, status.string, "accepted") and !std.mem.eql(u8, status.string, "edited"))) return error.UnreviewedItem;
    }
    if (included == 0 and rejected != items.items.len) return error.NoIncludedItems;
    const comments_value = payload.get("comments") orelse return error.OpenComment;
    const comments = switch (comments_value) {
        .array => |value| value,
        else => return error.OpenComment,
    };
    for (comments.items) |comment| {
        if (comment != .object) return error.OpenComment;
        const status = comment.object.get("status") orelse return error.OpenComment;
        if (status == .string and std.mem.eql(u8, status.string, "open")) return error.OpenComment;
    }
    return if (rejected == items.items.len) .rejected else .review_complete;
}

fn checkApprovalEligibility(payload: std.json.ObjectMap) !void {
    const items_value = payload.get("items") orelse return error.NoIncludedItems;
    const items = switch (items_value) {
        .array => |value| value,
        else => return error.NoIncludedItems,
    };
    var included: usize = 0;
    for (items.items) |item| {
        if (item != .object) continue;
        const object = item.object;
        const status = object.get("review_status") orelse return error.UnreviewedItem;
        if (status == .string and std.mem.eql(u8, status.string, "rejected")) continue;
        if (object.get("included_in_approval")) |included_value| if (included_value == .bool and !included_value.bool) continue;
        included += 1;
        if (status != .string or (!std.mem.eql(u8, status.string, "accepted") and !std.mem.eql(u8, status.string, "edited"))) return error.UnreviewedItem;
    }
    if (included == 0) return error.NoIncludedItems;
    const comments_value = payload.get("comments") orelse return error.OpenComment;
    const comments = switch (comments_value) {
        .array => |value| value,
        else => return error.OpenComment,
    };
    for (comments.items) |comment| {
        if (comment != .object) continue;
        const status = comment.object.get("status") orelse return error.OpenComment;
        if (status == .string and std.mem.eql(u8, status.string, "open")) return error.OpenComment;
    }
}

fn approvalErrorMessage(err: anyerror) []const u8 {
    return switch (err) {
        error.UnreviewedItem => "Every included item must be accepted or edited.",
        error.OpenComment => "Open comments block approval.",
        error.NoIncludedItems => "At least one included item is required.",
        else => "Intent is not eligible for approval.",
    };
}

fn operation_idValue(payload: std.json.ObjectMap) ?[]const u8 {
    const value = payload.get("operation_id") orelse return null;
    return if (value == .string) value.string else null;
}

fn actorID(payload: std.json.ObjectMap) ?[]const u8 {
    const actor = payload.get("actor") orelse return null;
    if (actor != .object) return null;
    const value = actor.object.get("actor_id") orelse return null;
    return if (value == .string) value.string else null;
}

fn intentID(payload: std.json.ObjectMap) ?[]const u8 {
    const value = payload.get("intent_id") orelse return null;
    return if (value == .string) value.string else null;
}

fn capabilityRegistry(allocator: std.mem.Allocator, io: std.Io, intent_path: []const u8) ![]u8 {
    if (isIntentDirectory(io, intent_path)) return std.fmt.allocPrint(allocator, "{s}/.capabilities", .{intent_path});
    return std.fmt.allocPrint(allocator, "{s}.capabilities", .{intent_path});
}

fn unixNow(io: std.Io) i64 {
    return @intCast(@divTrunc(std.Io.Clock.now(.real, io).nanoseconds, 1_000_000_000));
}

fn payloadDigest(allocator: std.mem.Allocator, payload: std.json.Value) ![64]u8 {
    const canonical = try core.hashing.canonicalize(allocator, payload);
    defer allocator.free(canonical);
    return core.hashing.sha256Hex(canonical);
}

fn writeMutationEnvelope(request: core.protocol.Request, value: std.json.Value, writer: *std.Io.Writer) !void {
    try std.json.Stringify.value(.{ .protocol_version = core.protocol.protocol_version, .request_id = request.request_id, .ok = true, .result_schema = "zintent.result/1", .result = .{ .contract_version = "1.0.0", .ok = true, .operation = @tagName(request.operation), .affected_ids = &.{}, .findings = &.{}, .data = .{ .intent = value } } }, .{}, writer);
}

fn stringField(object: std.json.ObjectMap, key: []const u8) ?[]const u8 {
    const value = object.get(key) orelse return null;
    return if (value == .string) value.string else null;
}

fn stringFromObject(object: std.json.ObjectMap, key: []const u8) ?[]const u8 {
    return stringField(object, key);
}

fn interactiveTTY(payload: std.json.ObjectMap) bool {
    const value = payload.get("interactive_tty") orelse return false;
    return value == .bool and value.bool;
}

fn challengeFor(token: []const u8, approved_hash: []const u8) [12]u8 {
    var input: [256]u8 = undefined;
    const length = std.fmt.bufPrint(&input, "{s}\x00{s}", .{ token, approved_hash }) catch unreachable;
    const digest = core.hashing.sha256Hex(length);
    var challenge: [12]u8 = undefined;
    @memcpy(&challenge, digest[0..12]);
    return challenge;
}

fn buildApprovedContent(allocator: std.mem.Allocator, payload: std.json.ObjectMap) !std.json.ObjectMap {
    var approved = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    if (payload.get("intent_id")) |value| try approved.put(allocator, "intent_id", value);
    try approved.put(allocator, "schema_version", .{ .string = "1.0.0" });
    if (payload.get("source_references")) |value| try approved.put(allocator, "source_references", value) else try approved.put(allocator, "source_references", .{ .array = std.json.Array.init(allocator) });
    const items_value = payload.get("items") orelse return error.NoIncludedItems;
    const items = switch (items_value) {
        .array => |value| value,
        else => return error.NoIncludedItems,
    };
    var approved_items = std.json.Array.init(allocator);
    for (items.items) |item| {
        if (item != .object) continue;
        const object = item.object;
        if (object.get("included_in_approval")) |included| if (included == .bool and !included.bool) continue;
        var projected = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        for ([_][]const u8{ "item_id", "kind", "statement", "rationale", "provenance" }) |key| if (object.get(key)) |value| try projected.put(allocator, key, value);
        try approved_items.append(.{ .object = projected });
    }
    try approved.put(allocator, "items", .{ .array = approved_items });
    return approved;
}

fn publishApprovedSnapshot(allocator: std.mem.Allocator, io: std.Io, path: []const u8, approved_hash: [64]u8, approved_content: std.json.Value, confirmed_revision: []const u8, confirmed_revision_hash: []const u8, approved_revision_hash: []const u8, approval_id: []const u8, actor_id: []const u8, token_id: []const u8) ![]u8 {
    const snapshot_name = try std.fmt.allocPrint(allocator, "sha256-{s}.json", .{&approved_hash});
    defer allocator.free(snapshot_name);
    const snapshot_path = if (isIntentDirectory(io, path)) try std.fmt.allocPrint(allocator, "{s}/snapshots/{s}", .{ path, snapshot_name }) else try std.fmt.allocPrint(allocator, "{s}.snapshot-{s}", .{ path, snapshot_name });
    var snapshot_output: std.Io.Writer.Allocating = .init(allocator);
    defer snapshot_output.deinit();
    try std.json.Stringify.value(.{ .schema_version = "1.0.0", .snapshot_id = try std.fmt.allocPrint(allocator, "sha256-{s}", .{&approved_hash}), .hash_algorithm = "sha-256", .canonicalization = "jcs-rfc8785", .approved_content = approved_content, .approval = .{ .approval_id = approval_id, .confirmed_revision_id = confirmed_revision, .confirmed_revision_hash = confirmed_revision_hash, .approved_revision_id = approval_id, .approved_revision_hash = approved_revision_hash, .approved_content_hash = &approved_hash, .confirmation_token_id = token_id, .approving_actor = .{ .actor_id = actor_id, .identity_source = "explicit_fallback", .authenticated = false }, .approved_at = "now", .validation_result = .{ .eligible = true, .blocking_findings = &.{} } } }, .{}, &snapshot_output.writer);
    core.store.publishAtomic(allocator, io, snapshot_path, snapshot_output.written(), true) catch |err| switch (err) {
        error.DestinationExists => {
            var existing = std.Io.Dir.cwd().openFile(io, snapshot_path, .{}) catch return error.SnapshotCollision;
            defer existing.close(io);
            var buffer: [4096]u8 = undefined;
            var reader = existing.reader(io, &buffer);
            const existing_bytes = reader.interface.allocRemaining(allocator, .limited(16 * 1024 * 1024)) catch return error.SnapshotCollision;
            defer allocator.free(existing_bytes);
            if (!std.mem.eql(u8, existing_bytes, snapshot_output.written())) return error.SnapshotCollision;
        },
        else => return err,
    };
    return snapshot_path;
}

fn writePrepareApprovalResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_request", "Approval payload must be an object."),
    };
    if (!interactiveTTY(payload)) return writeFailure(io, request.request_id, "tty_required", "Approval preparation requires an interactive TTY.");
    const path = stringField(payload, "intent_path") orelse return writeFailure(io, request.request_id, "invalid_request", "Intent path is required.");
    const expected = stringField(payload, "expected_revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "revision is required.");
    const actor_id = actorID(payload) orelse return writeFailure(io, request.request_id, "missing_actor", "Human actor is required.");
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
    var validated = core.validation.parseRevision(allocator, bytes) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact failed validation.");
    defer validated.deinit();
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    const root = switch (parsed.value) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact must be an object."),
    };
    const current = stringFromObject(root, "revision_id") orelse stringFromObject(root, "current_revision_id") orelse "";
    core.store.checkExpectedRevision(expected, current) catch return writeFailure(io, request.request_id, "stale_revision", "Expected revision does not match current revision.");
    const revision_payload = root.get("revision_payload") orelse return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload is required.");
    const payload_object = switch (revision_payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload must be an object."),
    };
    const lifecycle = stringFromObject(payload_object, "lifecycle_state") orelse "";
    if (!std.mem.eql(u8, lifecycle, "review_complete")) return writeFailure(io, request.request_id, "approval_ineligible", "Intent must be review_complete before approval.");
    checkApprovalEligibility(payload_object) catch |err| return writeFailure(io, request.request_id, "approval_ineligible", approvalErrorMessage(err));
    const approved_content = try buildApprovedContent(allocator, payload_object);
    const canonical = try core.hashing.canonicalize(allocator, .{ .object = approved_content });
    defer allocator.free(canonical);
    const approved_hash = core.hashing.sha256Hex(canonical);
    const token_input = try std.fmt.allocPrint(allocator, "{s}\x00{s}\x00{s}", .{ current, actor_id, &approved_hash });
    defer allocator.free(token_input);
    const token_id = core.hashing.sha256Hex(token_input);
    const challenge = challengeFor(&token_id, &approved_hash);
    const registry = try capabilityRegistry(allocator, io, path);
    defer allocator.free(registry);
    core.store.issueCapability(allocator, io, registry, .{ .token_id = &token_id, .intent_id = intentID(payload_object) orelse "", .expected_revision_id = current, .actor_id = actor_id, .payload_hash = &approved_hash, .expires_at_unix = unixNow(io) + 600 }) catch |err| switch (err) {
        error.DestinationExists => {},
        else => return writeFailure(io, request.request_id, "persistence_failure", "Approval challenge could not be issued."),
    };
    try std.json.Stringify.value(.{ .protocol_version = core.protocol.protocol_version, .request_id = request.request_id, .ok = true, .result_schema = "zintent.result/1", .result = .{ .contract_version = "1.0.0", .ok = true, .operation = "prepare_approval", .affected_ids = &.{current}, .findings = &.{}, .data = .{ .confirmation = .{ .token_id = &token_id, .confirmed_revision_id = current, .confirmed_revision_hash = stringFromObject(root, "revision_hash") orelse "", .approved_content_hash = &approved_hash, .actor_id = actor_id, .challenge = &challenge, .interactive_tty = true, .consumed = false } } } }, .{}, writer);
}

fn writeApproveResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_request", "Approval payload must be an object."),
    };
    if (!interactiveTTY(payload)) return writeFailure(io, request.request_id, "tty_required", "Approval requires an interactive TTY.");
    const path = stringField(payload, "intent_path") orelse return writeFailure(io, request.request_id, "invalid_request", "Intent path is required.");
    const expected = stringField(payload, "expected_revision_id") orelse return writeFailure(io, request.request_id, "invalid_request", "revision is required.");
    const actor_id = actorID(payload) orelse return writeFailure(io, request.request_id, "missing_actor", "Human actor is required.");
    const token = stringField(payload, "confirmation_token") orelse return writeFailure(io, request.request_id, "invalid_confirmation", "confirmation_token is required.");
    const response = stringField(payload, "challenge_response") orelse return writeFailure(io, request.request_id, "invalid_confirmation", "challenge_response is required.");
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
    var validated = core.validation.parseRevision(allocator, bytes) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact failed validation.");
    defer validated.deinit();
    _ = core.transition.next(validated.value.revision_payload.lifecycle_state, .approve_intent) catch return writeFailure(io, request.request_id, "invalid_transition", "Intent must be review_complete before approval.");
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    var root = switch (parsed.value) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact must be an object."),
    };
    const current = stringFromObject(root, "revision_id") orelse "";
    const confirmed_revision_hash = stringFromObject(root, "revision_hash") orelse "";
    core.store.checkExpectedRevision(expected, current) catch return writeFailure(io, request.request_id, "stale_revision", "Expected revision does not match current revision.");
    const revision_payload = root.getPtr("revision_payload") orelse return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload is required.");
    var payload_object = switch (revision_payload.*) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload must be an object."),
    };
    checkApprovalEligibility(payload_object) catch |err| return writeFailure(io, request.request_id, "approval_ineligible", approvalErrorMessage(err));
    const approved_content = try buildApprovedContent(allocator, payload_object);
    const canonical = try core.hashing.canonicalize(allocator, .{ .object = approved_content });
    defer allocator.free(canonical);
    const approved_hash = core.hashing.sha256Hex(canonical);
    const registry = try capabilityRegistry(allocator, io, path);
    defer allocator.free(registry);
    var capability = core.store.consumeCapabilityRecord(allocator, io, registry, token, unixNow(io)) catch return writeFailure(io, request.request_id, "invalid_confirmation", "Approval challenge is expired, consumed, or unknown.");
    defer capability.deinit();
    if (!std.mem.eql(u8, capability.value.expected_revision_id, current) or !std.mem.eql(u8, capability.value.actor_id, actor_id) or !std.mem.eql(u8, capability.value.payload_hash, &approved_hash) or !std.mem.eql(u8, response, &challengeFor(token, &approved_hash))) return writeFailure(io, request.request_id, "invalid_confirmation", "Approval challenge does not match this revision.");
    const operation_id = stringField(payload, "operation_id") orelse token;
    try root.put(allocator, "parent_revision_id", .{ .string = current });
    try root.put(allocator, "revision_id", .{ .string = operation_id });
    try root.put(allocator, "operation_id", .{ .string = operation_id });
    if (payload.get("actor")) |actor| try root.put(allocator, "actor", actor);
    try payload_object.put(allocator, "lifecycle_state", .{ .string = "approved" });
    var operation = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try operation.put(allocator, "type", .{ .string = "approve_intent" });
    try operation.put(allocator, "target_ids", .{ .array = std.json.Array.init(allocator) });
    try root.put(allocator, "operation", .{ .object = operation });
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    const approved_revision_hash = core.hashing.sha256Hex(output.written());
    try root.put(allocator, "revision_hash", .{ .string = &approved_revision_hash });
    output.clearRetainingCapacity();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    var approved_revision = core.validation.parseRevision(allocator, output.written()) catch return writeFailure(io, request.request_id, "invalid_artifact", "Approval would violate Intent invariants.");
    defer approved_revision.deinit();
    const snapshot_id = publishApprovedSnapshot(allocator, io, path, approved_hash, .{ .object = approved_content }, current, confirmed_revision_hash, &approved_revision_hash, operation_id, actor_id, token) catch return writeFailure(io, request.request_id, "persistence_failure", "Approved snapshot could not be published.");
    defer allocator.free(snapshot_id);
    try publishMutation(allocator, io, path, operation_id, output.written(), root, false, snapshot_id);
    const snapshot_id_string: []const u8 = snapshot_id;
    var response_data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try response_data.put(allocator, "snapshot", .{ .string = snapshot_id_string });
    try response_data.put(allocator, "snapshot_path", .{ .string = snapshot_id_string });
    var affected = std.json.Array.init(allocator);
    try affected.append(.{ .string = operation_id });
    const findings = std.json.Array.init(allocator);
    var result = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try result.put(allocator, "contract_version", .{ .string = "1.0.0" });
    try result.put(allocator, "ok", .{ .bool = true });
    try result.put(allocator, "operation", .{ .string = "approve_intent" });
    try result.put(allocator, "affected_ids", .{ .array = affected });
    try result.put(allocator, "findings", .{ .array = findings });
    try result.put(allocator, "data", .{ .object = response_data });
    var envelope = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try envelope.put(allocator, "protocol_version", .{ .string = core.protocol.protocol_version });
    try envelope.put(allocator, "request_id", .{ .string = request.request_id });
    try envelope.put(allocator, "ok", .{ .bool = true });
    try envelope.put(allocator, "result_schema", .{ .string = "zintent.result/1" });
    try envelope.put(allocator, "result", .{ .object = result });
    try std.json.Stringify.value(std.json.Value{ .object = envelope }, .{}, writer);
}

fn publishMutation(allocator: std.mem.Allocator, io: std.Io, path: []const u8, revision_id: []const u8, revision_bytes: []const u8, root: std.json.ObjectMap, lock_already_held: bool, approved_snapshot_ref: ?[]const u8) !void {
    var directory = std.Io.Dir.cwd().openDir(io, path, .{}) catch |err| switch (err) {
        error.NotDir, error.FileNotFound => null,
        else => return err,
    };
    if (directory) |*dir| {
        dir.close(io);
        var lock_path: ?[]u8 = null;
        var lock: ?core.store.Lock = null;
        if (!lock_already_held) {
            lock_path = try std.fmt.allocPrint(allocator, "{s}/.lock", .{path});
            lock = try core.store.acquireLock(io, lock_path.?);
        }
        defer {
            if (lock) |value| value.release();
            if (lock_path) |value| allocator.free(value);
        }
        const revision_path = try std.fmt.allocPrint(allocator, "{s}/revisions/{s}.json", .{ path, revision_id });
        defer allocator.free(revision_path);
        const head_path = try std.fmt.allocPrint(allocator, "{s}/HEAD.json", .{path});
        defer allocator.free(head_path);
        const intent_path = try std.fmt.allocPrint(allocator, "{s}/intent.json", .{path});
        defer allocator.free(intent_path);
        const hash = core.hashing.sha256Hex(revision_bytes);
        const intent_id = if (root.get("revision_payload")) |payload| if (payload == .object) if (payload.object.get("intent_id")) |value| if (value == .string) value.string else "unknown" else "unknown" else "unknown" else "unknown";
        const lifecycle = if (root.get("revision_payload")) |payload| if (payload == .object) if (payload.object.get("lifecycle_state")) |state| if (state == .string) state.string else "draft" else "draft" else "draft" else "draft";
        var head_output: std.Io.Writer.Allocating = .init(allocator);
        defer head_output.deinit();
        try std.json.Stringify.value(.{ .schema_version = "1.0.0", .intent_id = intent_id, .current_revision_id = revision_id, .current_revision_hash = &hash, .lifecycle_state = lifecycle, .approved_snapshot_ref = approved_snapshot_ref }, .{}, &head_output.writer);
        try core.store.publishRevisionAndHead(allocator, io, revision_path, revision_bytes, head_path, head_output.written());
        try core.store.publishAtomic(allocator, io, intent_path, revision_bytes, false);
        return;
    }
    try core.store.publishAtomic(allocator, io, path, revision_bytes, false);
}

fn writeIntentResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent payload is not an object."),
    };
    const path_value = payload.get("intent_path") orelse return writeFailure(io, request.request_id, "invalid_artifact", "Intent path is required.");
    const path = switch (path_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent path must be a string."),
    };
    const bytes = readIntentBytes(allocator, io, path) catch |err| switch (err) {
        error.IntegrityFailure => return writeFailure(io, request.request_id, "integrity_failure", "HEAD does not match the selected revision."),
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read."),
    };
    defer allocator.free(bytes);
    var parsed = std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always }) catch
        return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer parsed.deinit();
    if (request.operation == .validate_intent) {
        var validated = core.validation.parseRevision(allocator, bytes) catch |err| return writeValidationFailure(io, request.request_id, err);
        validated.deinit();
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

fn writeDiffResult(allocator: std.mem.Allocator, io: std.Io, request: core.protocol.Request, writer: *std.Io.Writer) !void {
    const payload = switch (request.payload) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_request", "Diff payload must be an object."),
    };
    const path = stringField(payload, "intent_path") orelse return writeFailure(io, request.request_id, "invalid_request", "Intent path is required.");
    const current_bytes = readIntentBytes(allocator, io, path) catch |err| switch (err) {
        error.IntegrityFailure => return writeFailure(io, request.request_id, "integrity_failure", "HEAD does not match the selected revision."),
        else => return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read."),
    };
    defer allocator.free(current_bytes);
    var current = std.json.parseFromSlice(std.json.Value, allocator, current_bytes, .{ .allocate = .alloc_always }) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact is not valid JSON.");
    defer current.deinit();
    var before = current.value;
    const from_id = stringField(payload, "from_revision_id");
    const to_id = stringField(payload, "to_revision_id");
    const target_id = to_id orelse stringFromValue(current.value, "revision_id") orelse "";
    if (isIntentDirectory(io, path) and from_id != null) {
        if (readRevisionValue(allocator, io, path, from_id.?)) |parsed| {
            before = parsed.value;
            // parsed is intentionally retained by the arena allocator for this request.
        } else |_| return writeFailure(io, request.request_id, "invalid_artifact", "Requested source revision could not be read.");
    }
    if (isIntentDirectory(io, path) and to_id != null and !std.mem.eql(u8, target_id, stringFromValue(current.value, "revision_id") orelse "")) {
        if (readRevisionValue(allocator, io, path, target_id)) |parsed| {
            current.value = parsed.value;
        } else |_| return writeFailure(io, request.request_id, "invalid_artifact", "Requested target revision could not be read.");
    }
    const changes = core.diff.itemChanges(allocator, before, current.value) catch return writeFailure(io, request.request_id, "invalid_artifact", "Revisions could not be compared.");
    var data = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try data.put(allocator, "from_revision_id", .{ .string = from_id orelse "" });
    try data.put(allocator, "to_revision_id", .{ .string = target_id });
    try data.put(allocator, "changes", .{ .array = changes });
    var result = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try result.put(allocator, "contract_version", .{ .string = "1.0.0" });
    try result.put(allocator, "ok", .{ .bool = true });
    try result.put(allocator, "operation", .{ .string = "diff_revisions" });
    try result.put(allocator, "affected_ids", .{ .array = std.json.Array.init(allocator) });
    try result.put(allocator, "findings", .{ .array = std.json.Array.init(allocator) });
    try result.put(allocator, "data", .{ .object = data });
    var envelope = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try envelope.put(allocator, "protocol_version", .{ .string = core.protocol.protocol_version });
    try envelope.put(allocator, "request_id", .{ .string = request.request_id });
    try envelope.put(allocator, "ok", .{ .bool = true });
    try envelope.put(allocator, "result_schema", .{ .string = "zintent.result/1" });
    try envelope.put(allocator, "result", .{ .object = result });
    try std.json.Stringify.value(std.json.Value{ .object = envelope }, .{}, writer);
}

fn readRevisionValue(allocator: std.mem.Allocator, io: std.Io, intent_dir: []const u8, revision_id: []const u8) !std.json.Parsed(std.json.Value) {
    const path = try std.fmt.allocPrint(allocator, "{s}/revisions/{s}.json", .{ intent_dir, revision_id });
    defer allocator.free(path);
    const bytes = try readIntentBytes(allocator, io, path);
    defer allocator.free(bytes);
    return std.json.parseFromSlice(std.json.Value, allocator, bytes, .{ .allocate = .alloc_always });
}

fn stringFromValue(value: std.json.Value, key: []const u8) ?[]const u8 {
    const object = switch (value) {
        .object => |o| o,
        else => return null,
    };
    return if (object.get(key)) |field| if (field == .string) field.string else null else null;
}

fn readIntentBytes(allocator: std.mem.Allocator, io: std.Io, path: []const u8) ![]u8 {
    if (isIntentDirectory(io, path)) {
        const verified = try core.store.loadVerifiedRevision(allocator, io, path);
        return verified.bytes;
    }
    var file = try std.Io.Dir.cwd().openFile(io, path, .{});
    defer file.close(io);
    var buffer: [4096]u8 = undefined;
    var reader = file.reader(io, &buffer);
    return reader.interface.allocRemaining(allocator, .limited(core.max_message_bytes));
}

fn isIntentDirectory(io: std.Io, path: []const u8) bool {
    var directory = std.Io.Dir.cwd().openDir(io, path, .{}) catch return false;
    directory.close(io);
    return true;
}

fn writeValidationFailure(io: std.Io, request_id: []const u8, err: anyerror) !void {
    const message = switch (err) {
        error.InvalidIntent => "Intent artifact failed structural validation.",
        else => "Intent validation failed.",
    };
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

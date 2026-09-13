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
    } else if (request.value.operation == .preview_edit) {
        try writePreviewResult(allocator, io, request.value, &stdout.interface);
    } else if (request.value.operation == .start_review or request.value.operation == .accept_item or request.value.operation == .edit_item or request.value.operation == .reject_item or request.value.operation == .add_comment or request.value.operation == .resolve_comment or request.value.operation == .withdraw_comment or request.value.operation == .complete_review) {
        try writeMutationResult(allocator, io, request.value, &stdout.interface);
    } else {
        try core.protocol.writeResponse(request.value, &stdout.interface);
    }
    try stdout.interface.writeByte('\n');
    try stdout.flush();
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
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
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
    const bytes = readIntentBytes(allocator, io, path) catch return writeFailure(io, request.request_id, "invalid_artifact", "Intent artifact could not be read.");
    defer allocator.free(bytes);
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
    core.store.checkExpectedRevision(expected_revision, current_revision) catch return writeFailure(io, request.request_id, "stale_revision", "Expected revision does not match current revision.");
    const operation_id_value = payload.get("operation_id") orelse return writeFailure(io, request.request_id, "invalid_request", "operation_id is required.");
    const operation_id = switch (operation_id_value) {
        .string => |value| value,
        else => return writeFailure(io, request.request_id, "invalid_request", "operation_id must be a string."),
    };
    try root.put(allocator, "parent_revision_id", .{ .string = current_revision });
    try root.put(allocator, "operation_id", .{ .string = operation_id });
    if (payload.get("actor")) |actor| try root.put(allocator, "actor", actor);
    var operation_record = try std.json.ObjectMap.init(allocator, &.{}, &.{});
    try operation_record.put(allocator, "type", .{ .string = @tagName(request.operation) });
    const target_ids = std.json.Array.init(allocator);
    try operation_record.put(allocator, "target_ids", .{ .array = target_ids });
    try root.put(allocator, "operation", .{ .object = operation_record });
    const revision_payload = root.getPtr("revision_payload") orelse return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload is required.");
    var payload_object = switch (revision_payload.*) {
        .object => |object| object,
        else => return writeFailure(io, request.request_id, "invalid_artifact", "revision_payload must be an object."),
    };
    if (request.operation == .start_review) try payload_object.put(allocator, "lifecycle_state", .{ .string = "in_review" });
    if (request.operation == .complete_review) {
        const items = payload_object.get("items") orelse return writeFailure(io, request.request_id, "approval_ineligible", "items are required.");
        const comments = payload_object.get("comments") orelse return writeFailure(io, request.request_id, "approval_ineligible", "comments are required.");
        _ = items;
        _ = comments;
        try payload_object.put(allocator, "lifecycle_state", .{ .string = "review_complete" });
    }
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
                    .accept_item => try item_object.put(allocator, "review_status", .{ .string = "accepted" }),
                    .reject_item => {
                        try item_object.put(allocator, "review_status", .{ .string = "rejected" });
                        try item_object.put(allocator, "included_in_approval", .{ .bool = false });
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
        try comment.put(allocator, "created_revision_id", .{ .string = current_revision });
        const comments_value = payload_object.getPtr("comments") orelse return writeFailure(io, request.request_id, "invalid_artifact", "comments are required.");
        const comments = switch (comments_value.*) {
            .array => |*array| array,
            else => return writeFailure(io, request.request_id, "invalid_artifact", "comments must be an array."),
        };
        try comments.append(.{ .object = comment });
    }
    if (request.operation == .resolve_comment or request.operation == .withdraw_comment) {
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
                try comment_object.put(allocator, "status", .{ .string = if (request.operation == .resolve_comment) "resolved" else "withdrawn" });
                if (payload.get("reason")) |reason| try comment_object.put(allocator, "closure_reason", reason);
                comments.items[index] = .{ .object = comment_object };
            }
        }
        if (!found_comment) return writeFailure(io, request.request_id, "invalid_artifact", "Target comment was not found.");
    }
    const revision_id = operation_id;
    try root.put(allocator, "revision_id", .{ .string = revision_id });
    var output: std.Io.Writer.Allocating = .init(allocator);
    defer output.deinit();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    const revision_digest = core.hashing.sha256Hex(output.written());
    try root.put(allocator, "revision_hash", .{ .string = &revision_digest });
    output.clearRetainingCapacity();
    try std.json.Stringify.value(parsed.value, .{}, &output.writer);
    try publishMutation(allocator, io, path, revision_id, output.written(), root);
    try std.json.Stringify.value(.{ .protocol_version = core.protocol.protocol_version, .request_id = request.request_id, .ok = true, .result_schema = "zintent.result/1", .result = .{ .contract_version = "1.0.0", .ok = true, .operation = @tagName(request.operation), .affected_ids = &.{}, .findings = &.{}, .data = .{ .intent = parsed.value } } }, .{}, writer);
}

fn publishMutation(allocator: std.mem.Allocator, io: std.Io, path: []const u8, revision_id: []const u8, revision_bytes: []const u8, root: std.json.ObjectMap) !void {
    var directory = std.Io.Dir.cwd().openDir(io, path, .{}) catch |err| switch (err) {
        error.NotDir, error.FileNotFound => null,
        else => return err,
    };
    if (directory) |*dir| {
        dir.close(io);
        const lock_path = try std.fmt.allocPrint(allocator, "{s}/.lock", .{path});
        defer allocator.free(lock_path);
        var lock = try core.store.acquireLock(io, lock_path);
        defer lock.release();
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
        try std.json.Stringify.value(.{ .schema_version = "1.0.0", .intent_id = intent_id, .current_revision_id = revision_id, .current_revision_hash = &hash, .lifecycle_state = lifecycle, .approved_snapshot_ref = null }, .{}, &head_output.writer);
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

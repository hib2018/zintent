const std = @import("std");
const model = @import("model.zig");
const store = @import("store.zig");

pub const Discovery = struct { entries: std.json.Array, findings: std.json.Array };

pub fn discover(allocator: std.mem.Allocator, io: std.Io, workspace_path: []const u8) !Discovery {
    var root = try std.Io.Dir.cwd().openDir(io, workspace_path, .{ .iterate = true });
    defer root.close(io);
    var entries = std.json.Array.init(allocator);
    var findings = std.json.Array.init(allocator);
    var iterator = root.iterate();
    while (try iterator.next(io)) |entry| {
        if (entry.kind != .directory) continue;
        if (std.mem.startsWith(u8, entry.name, ".") or std.mem.endsWith(u8, entry.name, ".tmp")) continue;
        const child = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ workspace_path, entry.name });
        defer allocator.free(child);
        const verified = store.loadVerifiedRevision(allocator, io, child) catch {
            var finding = try std.json.ObjectMap.init(allocator, &.{}, &.{});
            try finding.put(allocator, "code", .{ .string = "corrupt_entry" });
            try finding.put(allocator, "severity", .{ .string = "blocking" });
            try finding.put(allocator, "record_id", .{ .string = try allocator.dupe(u8, entry.name) });
            try finding.put(allocator, "message", .{ .string = "Intent entry could not be verified." });
            try findings.append(.{ .object = finding });
            continue;
        };
        defer allocator.free(verified.bytes);
        var revision = std.json.parseFromSlice(std.json.Value, allocator, verified.bytes, .{ .allocate = .alloc_always }) catch {
            var finding = try std.json.ObjectMap.init(allocator, &.{}, &.{});
            try finding.put(allocator, "code", .{ .string = "corrupt_entry" });
            try finding.put(allocator, "severity", .{ .string = "blocking" });
            try finding.put(allocator, "record_id", .{ .string = try allocator.dupe(u8, entry.name) });
            try finding.put(allocator, "message", .{ .string = "Intent revision is not valid JSON." });
            try findings.append(.{ .object = finding });
            continue;
        };
        defer revision.deinit();
        var object = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try object.put(allocator, "intent_id", .{ .string = try allocator.dupe(u8, verified.head.intent_id) });
        try object.put(allocator, "display_name", .{ .string = try allocator.dupe(u8, entry.name) });
        try object.put(allocator, "intent_path", .{ .string = try allocator.dupe(u8, entry.name) });
        try object.put(allocator, "current_revision_id", .{ .string = try allocator.dupe(u8, verified.head.current_revision_id) });
        try object.put(allocator, "lifecycle_state", .{ .string = try allocator.dupe(u8, verified.head.lifecycle_state) });
        try object.put(allocator, "blocker_count", .{ .integer = @intCast(blockerCount(revision.value)) });
        try object.put(allocator, "approval_state", .{ .string = if (verified.head.approved_snapshot_ref != null) "approved" else "none" });
        if (verified.head.approved_snapshot_ref) |id| try object.put(allocator, "snapshot_id", .{ .string = try allocator.dupe(u8, id) }) else try object.put(allocator, "snapshot_id", .null);
        try entries.append(.{ .object = object });
    }
    std.mem.sort(std.json.Value, entries.items, {}, struct {
        fn less(_: void, a: std.json.Value, b: std.json.Value) bool {
            const ai = a.object.get("intent_id").?.string;
            const bi = b.object.get("intent_id").?.string;
            return std.mem.lessThan(u8, ai, bi);
        }
    }.less);
    return .{ .entries = entries, .findings = findings };
}

pub fn blockerCount(revision: std.json.Value) usize {
    const root = switch (revision) {
        .object => |value| value,
        else => return 0,
    };
    const payload_value = root.get("revision_payload") orelse return 0;
    const payload = switch (payload_value) {
        .object => |value| value,
        else => return 0,
    };
    var count: usize = 0;
    var included: usize = 0;
    var item_count: usize = 0;
    var rejected: usize = 0;
    if (payload.get("items")) |items_value| switch (items_value) {
        .array => |items| for (items.items) |item_value| {
            const item = switch (item_value) {
                .object => |value| value,
                else => continue,
            };
            item_count += 1;
            const status = item.get("review_status");
            if (status != null and status.? == .string and std.mem.eql(u8, status.?.string, "rejected")) {
                rejected += 1;
                continue;
            }
            if (item.get("included_in_approval")) |value| if (value == .bool and !value.bool) continue;
            included += 1;
            if (status == null or status.? != .string or
                (!std.mem.eql(u8, status.?.string, "accepted") and !std.mem.eql(u8, status.?.string, "edited"))) count += 1;
        },
        else => {},
    };
    if (included == 0 and (item_count == 0 or rejected != item_count)) count += 1;
    if (payload.get("comments")) |comments_value| switch (comments_value) {
        .array => |comments| for (comments.items) |comment_value| {
            const comment = switch (comment_value) {
                .object => |value| value,
                else => continue,
            };
            const status = comment.get("status") orelse continue;
            if (status == .string and std.mem.eql(u8, status.string, "open")) count += 1;
        },
        else => {},
    };
    return count;
}

pub fn lessThan(_: void, left: model.WorkspaceEntry, right: model.WorkspaceEntry) bool {
    return std.mem.lessThan(u8, left.intent_id, right.intent_id);
}

test "blocker count includes unreviewed items and open comments" {
    const source =
        \\{"revision_payload":{"items":[{"review_status":"unreviewed"},{"review_status":"accepted"},{}],"comments":[{"status":"open"},{"status":"resolved"}]}}
    ;
    var parsed = try std.json.parseFromSlice(std.json.Value, std.testing.allocator, source, .{ .allocate = .alloc_always });
    defer parsed.deinit();
    try std.testing.expectEqual(@as(usize, 3), blockerCount(parsed.value));

    var rejected = try std.json.parseFromSlice(std.json.Value, std.testing.allocator, "{\"revision_payload\":{\"items\":[{\"review_status\":\"rejected\",\"included_in_approval\":false}],\"comments\":[]}}", .{});
    defer rejected.deinit();
    try std.testing.expectEqual(@as(usize, 0), blockerCount(rejected.value));
}

test "workspace entries sort by stable intent ID" {
    var entries = [_]model.WorkspaceEntry{
        .{ .intent_id = "I-2", .display_name = "second", .intent_path = "I-2" },
        .{ .intent_id = "I-1", .display_name = "first", .intent_path = "I-1" },
    };
    std.mem.sort(model.WorkspaceEntry, &entries, {}, lessThan);
    try std.testing.expectEqualStrings("I-1", entries[0].intent_id);
}

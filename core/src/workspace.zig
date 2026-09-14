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
        var object = try std.json.ObjectMap.init(allocator, &.{}, &.{});
        try object.put(allocator, "intent_id", .{ .string = try allocator.dupe(u8, verified.head.intent_id) });
        try object.put(allocator, "display_name", .{ .string = try allocator.dupe(u8, entry.name) });
        try object.put(allocator, "intent_path", .{ .string = try allocator.dupe(u8, entry.name) });
        try object.put(allocator, "current_revision_id", .{ .string = try allocator.dupe(u8, verified.head.current_revision_id) });
        try object.put(allocator, "lifecycle_state", .{ .string = try allocator.dupe(u8, verified.head.lifecycle_state) });
        try object.put(allocator, "blocker_count", .{ .integer = 0 });
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

pub fn lessThan(_: void, left: model.WorkspaceEntry, right: model.WorkspaceEntry) bool {
    return std.mem.lessThan(u8, left.intent_id, right.intent_id);
}

test "workspace entries sort by stable intent ID" {
    var entries = [_]model.WorkspaceEntry{
        .{ .intent_id = "I-2", .display_name = "second", .intent_path = "I-2" },
        .{ .intent_id = "I-1", .display_name = "first", .intent_path = "I-1" },
    };
    std.mem.sort(model.WorkspaceEntry, &entries, {}, lessThan);
    try std.testing.expectEqualStrings("I-1", entries[0].intent_id);
}

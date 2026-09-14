const std = @import("std");
const core = @import("zintent_core");

test "arbitrary stable identifiers never admit path separators" {
    var prng = std.Random.DefaultPrng.init(0x5a1e);
    const random = prng.random();
    var bytes: [64]u8 = undefined;
    for (0..10_000) |_| {
        const len = random.uintLessThan(usize, bytes.len);
        random.bytes(bytes[0..len]);
        const candidate = bytes[0..len];
        if (std.mem.indexOfScalar(u8, candidate, '/') != null) {
            try std.testing.expect(!core.store.validStableId(candidate));
        }
    }
}

test "workspace entry comparator is deterministic" {
    var entries = [_]core.model.WorkspaceEntry{
        .{ .intent_id = "b", .display_name = "b", .intent_path = "b" },
        .{ .intent_id = "a", .display_name = "a", .intent_path = "a" },
    };
    for (0..100) |_| {
        std.mem.sort(core.model.WorkspaceEntry, &entries, {}, core.workspace.lessThan);
        try std.testing.expectEqualStrings("a", entries[0].intent_id);
    }
}

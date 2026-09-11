const std = @import("std");

pub const Change = struct { record_id: []const u8, before: []const u8, after: []const u8 };

pub fn statementChanged(before: []const u8, after: []const u8) bool {
    return !std.mem.eql(u8, before, after);
}

test "statement comparison" {
    try std.testing.expect(statementChanged("a", "b"));
    try std.testing.expect(!statementChanged("a", "a"));
}

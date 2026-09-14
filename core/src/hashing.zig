const std = @import("std");

pub fn sha256Hex(bytes: []const u8) [64]u8 {
    var digest: [32]u8 = undefined;
    std.crypto.hash.sha2.Sha256.hash(bytes, &digest, .{});
    return std.fmt.bytesToHex(digest, .lower);
}

pub fn canonicalize(allocator: std.mem.Allocator, value: std.json.Value) ![]u8 {
    var out: std.Io.Writer.Allocating = .init(allocator);
    errdefer out.deinit();
    try writeCanonical(allocator, value, &out.writer);
    return out.toOwnedSlice();
}

fn writeCanonical(allocator: std.mem.Allocator, value: std.json.Value, writer: *std.Io.Writer) !void {
    switch (value) {
        .object => |object| {
            try writer.writeByte('{');
            var keys: std.ArrayList([]const u8) = .empty;
            defer keys.deinit(allocator);
            var iterator = object.iterator();
            while (iterator.next()) |entry| try keys.append(allocator, entry.key_ptr.*);
            std.mem.sort([]const u8, keys.items, {}, struct {
                fn lessThan(_: void, a: []const u8, b: []const u8) bool {
                    return utf16LessThan(a, b);
                }
            }.lessThan);
            for (keys.items, 0..) |key, index| {
                if (index != 0) try writer.writeByte(',');
                try std.json.Stringify.value(key, .{}, writer);
                try writer.writeByte(':');
                try writeCanonical(allocator, object.get(key).?, writer);
            }
            try writer.writeByte('}');
        },
        .array => |array| {
            try writer.writeByte('[');
            for (array.items, 0..) |item, index| {
                if (index != 0) try writer.writeByte(',');
                try writeCanonical(allocator, item, writer);
            }
            try writer.writeByte(']');
        },
        .float => |number| {
            if (number == 0) try writer.writeByte('0') else try std.json.Stringify.value(value, .{}, writer);
        },
        else => try std.json.Stringify.value(value, .{}, writer),
    }
}

const Utf16Iterator = struct {
    bytes: []const u8,
    index: usize = 0,
    low: ?u16 = null,

    fn next(self: *Utf16Iterator) ?u16 {
        if (self.low) |value| {
            self.low = null;
            return value;
        }
        if (self.index >= self.bytes.len) return null;
        const length = std.unicode.utf8ByteSequenceLength(self.bytes[self.index]) catch return null;
        const cp = std.unicode.utf8Decode(self.bytes[self.index .. self.index + length]) catch return null;
        self.index += length;
        if (cp <= 0xffff) return @intCast(cp);
        const adjusted = cp - 0x10000;
        self.low = @intCast(0xdc00 + (adjusted & 0x3ff));
        return @intCast(0xd800 + (adjusted >> 10));
    }
};

fn utf16LessThan(a: []const u8, b: []const u8) bool {
    var left = Utf16Iterator{ .bytes = a };
    var right = Utf16Iterator{ .bytes = b };
    while (true) {
        const x = left.next();
        const y = right.next();
        if (x == null or y == null) return x == null and y != null;
        if (x.? != y.?) return x.? < y.?;
    }
}

test "sha256 vector" {
    try std.testing.expectEqualStrings("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", &sha256Hex(""));
}

test "canonical object order and unicode" {
    const allocator = std.testing.allocator;
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator, "{\"z\":1,\"é\":\"値\",\"a\":[2,1]}", .{});
    defer parsed.deinit();
    const result = try canonicalize(allocator, parsed.value);
    defer allocator.free(result);
    try std.testing.expectEqualStrings("{\"a\":[2,1],\"z\":1,\"é\":\"値\"}", result);
}

test "JCS uses UTF-16 property ordering" {
    const allocator = std.testing.allocator;
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator, "{\"�\":1,\"😀\":2}", .{});
    defer parsed.deinit();
    const result = try canonicalize(allocator, parsed.value);
    defer allocator.free(result);
    try std.testing.expectEqualStrings("{\"😀\":2,\"�\":1}", result);
}

test "JCS normalizes representable numbers" {
    const allocator = std.testing.allocator;
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator, "{\"fraction\":1.0,\"negativeZero\":-0.0}", .{});
    defer parsed.deinit();
    const result = try canonicalize(allocator, parsed.value);
    defer allocator.free(result);
    try std.testing.expectEqualStrings("{\"fraction\":1,\"negativeZero\":0}", result);
}

test "approved content projection hash is stable" {
    const allocator = std.testing.allocator;
    var parsed = try std.json.parseFromSlice(std.json.Value, allocator, "{\"intent_id\":\"i\",\"items\":[{\"item_id\":\"x\",\"kind\":\"goal\",\"provenance\":{\"content_origin\":\"human\"},\"statement\":\"Ship\"}],\"schema_version\":\"1.0.0\",\"source_references\":[]}", .{});
    defer parsed.deinit();
    const canonical = try canonicalize(allocator, parsed.value);
    defer allocator.free(canonical);
    try std.testing.expectEqualStrings("3523215b80d6bdc899ab38b939e248e76c50ff1c8a9fd84828ddd9e5887b6ed9", &sha256Hex(canonical));
}

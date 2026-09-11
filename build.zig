const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});
    const core = b.addModule("zintent_core", .{
        .root_source_file = b.path("core/src/root.zig"),
        .target = target,
    });
    const exe = b.addExecutable(.{
        .name = "zintent-core",
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/src/main.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    b.installArtifact(exe);
    const run = b.addRunArtifact(exe);
    if (b.args) |args| run.addArgs(args);
    b.step("run", "Run zintent-core").dependOn(&run.step);

    const tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/src/root.zig"),
            .target = target,
            .optimize = optimize,
        }),
    });
    const test_step = b.step("test", "Run core tests");
    test_step.dependOn(&b.addRunArtifact(tests).step);
    const contract_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/contract.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(contract_tests).step);
}

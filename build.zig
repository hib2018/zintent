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
    const model_validation_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/model_validation.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(model_validation_tests).step);
    const transition_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/transition.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(transition_tests).step);
    const persistence_failure_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/persistence_failure.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(persistence_failure_tests).step);
    const approval_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/approval.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(approval_tests).step);
    const diff_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/src/diff.zig"),
            .target = target,
            .optimize = optimize,
        }),
    });
    test_step.dependOn(&b.addRunArtifact(diff_tests).step);
    const recovery_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/recovery.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(recovery_tests).step);
    const workspace_contract_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("core/tests/workspace_contract.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{.{ .name = "zintent_core", .module = core }},
        }),
    });
    test_step.dependOn(&b.addRunArtifact(workspace_contract_tests).step);
    inline for (.{ "audit_workspace", "recovery_workspace", "workspace", "import", "workspace_fuzz" }) |name| {
        const workspace_test = b.addTest(.{ .root_module = b.createModule(.{ .root_source_file = b.path("core/tests/" ++ name ++ ".zig"), .target = target, .optimize = optimize, .imports = &.{.{ .name = "zintent_core", .module = core }} }) });
        test_step.dependOn(&b.addRunArtifact(workspace_test).step);
    }
}

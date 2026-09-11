const model = @import("model.zig");

pub const Command = struct {
    operation: model.Operation,
    intent_path: ?[]const u8 = null,
    expected_revision_id: ?[]const u8 = null,
    operation_id: ?[]const u8 = null,
    actor: ?model.Actor = null,
    item_id: ?[]const u8 = null,
    comment_id: ?[]const u8 = null,
    statement: ?[]const u8 = null,
    rationale: ?[]const u8 = null,
    preview_token: ?[]const u8 = null,
    confirmation_token: ?[]const u8 = null,
    challenge_response: ?[]const u8 = null,
    interactive_tty: ?bool = null,

    pub fn validate(self: Command) !void {
        if (model.isMutation(self.operation)) {
            if (self.intent_path == null or self.expected_revision_id == null or self.operation_id == null or self.actor == null)
                return error.InvalidCommand;
            try self.actor.?.validate();
        }
        switch (self.operation) {
            .preview_edit => if (self.item_id == null or self.statement == null or self.actor == null) return error.InvalidCommand,
            .edit_item => if (self.item_id == null or self.statement == null or self.preview_token == null) return error.InvalidCommand,
            .reject_item => if (self.item_id == null or self.rationale == null) return error.InvalidCommand,
            .prepare_approval => if (self.interactive_tty != true) return error.TtyRequired,
            .approve_intent => if (self.interactive_tty != true or self.confirmation_token == null or self.challenge_response == null) return error.InvalidConfirmation,
            else => {},
        }
    }
};

test "approval requires interactive confirmation" {
    const std = @import("std");
    try std.testing.expectError(error.InvalidCommand, (Command{ .operation = .approve_intent }).validate());
}

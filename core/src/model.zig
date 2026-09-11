const std = @import("std");

pub const Lifecycle = enum { draft, in_review, review_complete, approved };
pub const ReviewStatus = enum { unreviewed, accepted, edited, rejected };
pub const CommentStatus = enum { open, resolved, withdrawn };
pub const ContentOrigin = enum { source, human, ai, system };

pub const Actor = struct {
    actor_type: []const u8 = "human",
    actor_id: []const u8,
    identity_source: []const u8,
    authenticated: bool = false,

    pub fn validate(self: Actor) !void {
        if (!std.mem.eql(u8, self.actor_type, "human") or self.actor_id.len == 0 or self.authenticated)
            return error.InvalidActor;
    }
};

pub const Provenance = struct {
    content_origin: ContentOrigin,
    operation_actor: ?Actor = null,
    operation_id: []const u8,
    operation_type: []const u8,
    revision_id: []const u8,
    source_reference_ids: []const []const u8 = &.{},
};

pub const Item = struct {
    item_id: []const u8,
    kind: []const u8,
    statement: []const u8,
    provenance: Provenance,
    resolution_status: []const u8 = "determined",
    review_status: ReviewStatus = .unreviewed,
    included_in_approval: bool = true,
    rationale: ?[]const u8 = null,
    source_reference_ids: []const []const u8 = &.{},
    supersedes: []const []const u8 = &.{},
    superseded_by: []const []const u8 = &.{},
};

pub const Comment = struct {
    comment_id: []const u8,
    target_item_id: []const u8,
    body: []const u8,
    author: Actor,
    status: CommentStatus = .open,
    created_revision_id: []const u8,
    closed_revision_id: ?[]const u8 = null,
    closure_reason: ?[]const u8 = null,
    resolution_revision_id: ?[]const u8 = null,
};

pub const Finding = struct {
    code: []const u8,
    severity: []const u8 = "blocking",
    record_type: ?[]const u8 = null,
    record_id: ?[]const u8 = null,
    path: ?[]const u8 = null,
    message: []const u8,
};

pub const Operation = enum {
    protocol_info,
    show_intent,
    validate_intent,
    diff_revisions,
    start_review,
    accept_item,
    preview_edit,
    edit_item,
    reject_item,
    add_comment,
    resolve_comment,
    withdraw_comment,
    complete_review,
    prepare_approval,
    approve_intent,
};

pub fn isMutation(op: Operation) bool {
    return switch (op) {
        .protocol_info, .show_intent, .validate_intent, .diff_revisions, .preview_edit, .prepare_approval => false,
        else => true,
    };
}

test "actor is local unauthenticated human" {
    try (Actor{ .actor_id = "alice", .identity_source = "os_user" }).validate();
    try std.testing.expectError(error.InvalidActor, (Actor{ .actor_type = "ai", .actor_id = "bot", .identity_source = "explicit_fallback" }).validate());
}

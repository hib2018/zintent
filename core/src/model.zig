const std = @import("std");

pub const Id = []const u8;
pub const Timestamp = []const u8;

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

pub const OperationRecord = struct {
    type: []const u8,
    target_ids: []const []const u8,
    content_digest: ?[]const u8 = null,
};

pub const SourceReference = struct {
    source_reference_id: []const u8,
    kind: []const u8,
    locator: []const u8,
    content_hash: ?[]const u8 = null,
    excerpt: ?[]const u8 = null,
};

pub const RevisionPayload = struct {
    intent_id: []const u8,
    lifecycle_state: Lifecycle,
    source_references: []const SourceReference,
    items: []const Item,
    comments: []const Comment,
    approval_refs: []const []const u8,
};

pub const Revision = struct {
    schema_version: []const u8,
    revision_id: []const u8,
    revision_hash: []const u8,
    hash_algorithm: []const u8,
    canonicalization: []const u8,
    parent_revision_id: ?[]const u8,
    operation_id: []const u8,
    actor: Actor,
    operation: OperationRecord,
    created_at: []const u8,
    revision_payload: RevisionPayload,
};

pub const Approval = struct {
    approval_id: []const u8,
    confirmed_revision_id: []const u8,
    confirmed_revision_hash: []const u8,
    approved_revision_id: []const u8,
    approved_revision_hash: []const u8,
    approved_content_hash: []const u8,
    confirmation_token_id: []const u8,
    approving_actor: Actor,
    approved_at: []const u8,
    validation_result: struct { eligible: bool, blocking_findings: []const []const u8 },
};

pub const ApprovedSnapshot = struct {
    schema_version: []const u8,
    snapshot_id: []const u8,
    hash_algorithm: []const u8,
    canonicalization: []const u8,
    approved_content: std.json.Value,
    approval: Approval,
};

pub fn newItem(item_id: []const u8, kind: []const u8, statement: []const u8, provenance: Provenance) Item {
    return .{ .item_id = item_id, .kind = kind, .statement = statement, .provenance = provenance };
}

pub fn newComment(comment_id: []const u8, target_item_id: []const u8, body: []const u8, author: Actor, created_revision_id: []const u8) Comment {
    return .{ .comment_id = comment_id, .target_item_id = target_item_id, .body = body, .author = author, .created_revision_id = created_revision_id };
}

pub fn newRevisionPayload(intent_id: []const u8, lifecycle_state: Lifecycle, source_references: []const SourceReference, items: []const Item, comments: []const Comment) RevisionPayload {
    return .{ .intent_id = intent_id, .lifecycle_state = lifecycle_state, .source_references = source_references, .items = items, .comments = comments, .approval_refs = &.{} };
}

pub fn newRevision(schema_version: []const u8, revision_id: []const u8, revision_hash: []const u8, parent_revision_id: ?[]const u8, operation_id: []const u8, actor: Actor, operation: OperationRecord, created_at: []const u8, payload: RevisionPayload) Revision {
    return .{ .schema_version = schema_version, .revision_id = revision_id, .revision_hash = revision_hash, .hash_algorithm = "sha-256", .canonicalization = "jcs-rfc8785", .parent_revision_id = parent_revision_id, .operation_id = operation_id, .actor = actor, .operation = operation, .created_at = created_at, .revision_payload = payload };
}

pub const Finding = struct {
    code: []const u8,
    severity: []const u8 = "blocking",
    record_type: ?[]const u8 = null,
    record_id: ?[]const u8 = null,
    path: ?[]const u8 = null,
    message: []const u8,
};

pub const WorkspaceEntry = struct {
    intent_id: []const u8,
    display_name: []const u8,
    intent_path: []const u8,
    current_revision_id: ?[]const u8 = null,
    lifecycle_state: ?Lifecycle = null,
    blocker_count: usize = 0,
    approval_state: []const u8 = "unapproved",
    snapshot_id: ?[]const u8 = null,
};

pub const RevisionSummary = struct {
    revision_id: []const u8,
    parent_revision_id: ?[]const u8,
    revision_hash: []const u8,
    operation_type: []const u8,
    created_at: []const u8,
    reachable: bool = true,
};

pub const RecoveryCandidate = struct {
    candidate_id: []const u8,
    relative_path: []const u8,
    kind: []const u8,
    content_hash: []const u8,
};

pub const CapabilityBinding = struct {
    token_id: []const u8,
    operation: []const u8,
    actor_id: []const u8,
    resource_hash: []const u8,
    expires_at_unix: i64,
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
    list_intents,
    inspect_draft,
    import_draft,
    list_revisions,
    inspect_revision,
    inspect_snapshot,
    recovery_status,
    cleanup_temporary_files,
};

pub fn isMutation(op: Operation) bool {
    return switch (op) {
        .protocol_info, .show_intent, .validate_intent, .diff_revisions, .preview_edit, .prepare_approval, .list_intents, .inspect_draft, .list_revisions, .inspect_revision, .inspect_snapshot, .recovery_status => false,
        else => true,
    };
}

test "actor is local unauthenticated human" {
    try (Actor{ .actor_id = "alice", .identity_source = "os_user" }).validate();
    try std.testing.expectError(error.InvalidActor, (Actor{ .actor_type = "ai", .actor_id = "bot", .identity_source = "explicit_fallback" }).validate());
}

test "constructors create a draft payload with safe review defaults" {
    const actor = Actor{ .actor_id = "alice", .identity_source = "explicit_fallback" };
    const provenance = Provenance{ .content_origin = .source, .operation_id = "op", .operation_type = "fixture_import", .revision_id = "rev" };
    const item = newItem("i-1", "goal", "Ship it", provenance);
    const comment = newComment("c-1", item.item_id, "Clarify", actor, "rev");
    const items = [_]Item{item};
    const comments = [_]Comment{comment};
    const payload = newRevisionPayload("intent-1", .draft, &.{}, &items, &comments);
    try std.testing.expectEqual(Lifecycle.draft, payload.lifecycle_state);
    try std.testing.expectEqual(ReviewStatus.unreviewed, payload.items[0].review_status);
    try std.testing.expectEqual(CommentStatus.open, payload.comments[0].status);
}

# Feature Specification: Intent Review Walking Skeleton

**Feature Branch**: `001-intent-review-skeleton`

**Created**: 2026-09-09

**Status**: Draft

**Input**: User description: "先ほど合意したskill中心のzintent設計から、Feature 001:
Intent review walking skeletonの仕様を作成して"

## Clarifications

### Session 2026-09-09

- Q: Feature 001 の受入条件として、Hunk風の対話型TUIを必須にしますか？ → A: 最小Hunk風TUIを必須とする
- Q: 項目を reject した後、そのIntent全体はどの条件で承認可能にしますか？ → A: Rejectした項目を理由付きで承認対象から除外する
- Q: Feature 001 では、open commentをどの操作で承認可能な状態へ解消しますか？ → A: 人間がresolveまたはwithdrawを明示する
- Q: 認証機能を持たないFeature 001で、承認者や編集者のactor identityをどのように記録しますか？ → A: OSユーザー名を自動取得する
- Q: Intent Documentのライフサイクル状態を、どの状態遷移として明示しますか？ → A: draft → in_review → review_complete → approved

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review Intent Items (Priority: P1)

As an intent owner, I can open an existing Intent Draft and review each item individually so that
I can accept correct interpretations, edit incorrect ones, comment on unresolved concerns, or
reject items before any approval occurs.

**Why this priority**: Item-level human intervention is the single behavior this walking skeleton
must prove. Without it, zintent does not provide meaningful control over interpretation.

**Independent Test**: Load a valid Draft containing several items, perform each available review
action on a different item, close and reopen the review from HEAD, and verify that every action and
resulting item and comment status is preserved. Advanced history inspection is not required.

**Acceptance Scenarios**:

1. **Given** a valid Draft with unreviewed items, **When** the reviewer opens it, **Then** every
   item is shown with its stable ID, kind, statement, provenance, source reference, and review
   status.
2. **Given** an unreviewed item, **When** the reviewer accepts it, **Then** the item is recorded as
   accepted by the human reviewer in a new revision.
3. **Given** an unreviewed or accepted item, **When** the reviewer edits its statement, **Then** the
   proposed before/after diff is shown and the reviewer confirms that exact preview, and the
   human-authored text is stored in a new revision with human provenance while the prior value
   remains available for comparison.
4. **Given** an item requiring follow-up, **When** the reviewer adds a comment, **Then** the comment
   receives a stable identity, remains open, references that item, and blocks final approval.
5. **Given** an open comment, **When** the reviewer explicitly resolves or withdraws it with a
   reason, **Then** it stops blocking approval and records the reason and any related revision.
6. **Given** an item the reviewer does not want in the approved Intent, **When** the reviewer
   rejects it with a rationale, **Then** the item is excluded from approval content without
   preventing approval solely because of its rejected status, and its history remains available.

---

### User Story 2 - Validate and Approve Reviewed Intent (Priority: P2)

As an intent owner, I can see whether the reviewed Intent is eligible for approval and explicitly
approve an eligible revision so that I receive a fixed, trustworthy representation of exactly what
I accepted.

**Why this priority**: The walking skeleton is complete only when human review can produce a safe
approved artifact rather than merely modifying a Draft.

**Independent Test**: Attempt approval with an unresolved comment and observe rejection; resolve the
blocker, approve explicitly, and verify that the resulting snapshot identifies the approved
revision and cannot be altered.

**Acceptance Scenarios**:

1. **Given** an Intent with an open comment, invalid data, or an unreviewed active item, **When**
   approval eligibility is checked, **Then** approval is denied with findings that identify every
   blocking record.
2. **Given** an eligible reviewed Intent, **When** the human confirms a short-lived challenge in an
   interactive terminal, **Then** a new approved revision and its Approved Intent Snapshot are
   issued from the exact confirmed review-complete revision.
3. **Given** an Approved Intent Snapshot, **When** it is inspected, **Then** it records the content
   hash, approving actor, approval time, and validation result.
4. **Given** an approved Intent, **When** a subsequent change is made to the working Intent,
   **Then** the prior snapshot remains unchanged and the working Intent is no longer approved.
5. **Given** a request to change an existing Approved Intent Snapshot, **When** the change is
   attempted, **Then** it is refused.
6. **Given** an approved working Intent, **When** the reviewer begins a permitted new change,
   **Then** a new working revision enters `in_review` while the approved snapshot remains unchanged.

---

### User Story 3 - Resume and Audit a Review (Priority: P3)

As an intent owner, I can leave a review and later resume it from persisted artifacts so that no
review decision depends on conversational memory or one uninterrupted session.

**Why this priority**: Resumability validates that artifacts, rather than a particular interface or
Agent session, are the real process boundary.

**Independent Test**: Perform a partial review, end the session, begin a new session using only the
saved Intent artifact, and verify that the reviewer can identify completed and remaining work and
continue without reconstructing prior context.

**Acceptance Scenarios**:

1. **Given** a partially reviewed Intent, **When** it is reopened, **Then** all item decisions,
   comments, revisions, provenance, and remaining blockers are restored.
2. **Given** multiple revisions, **When** the reviewer requests a comparison, **Then** changed items
   and their before-and-after statements are identifiable.
3. **Given** a recorded operation, **When** its provenance is inspected, **Then** the actor type,
   operation, affected record, and revision are available.

### Edge Cases

- A missing, unreadable, or schema-incompatible Draft is refused without creating a partial review
  revision, and the reviewer receives an actionable validation finding.
- Duplicate Intent, item, comment, revision, or approval identifiers are treated as invalid.
- An empty Draft cannot be approved.
- An item edit that produces an empty statement is refused.
- Rejecting an item with open comments does not silently resolve those comments; each comment must
  be explicitly resolved or withdrawn before approval.
- Editing an item does not automatically resolve its comments, even when the edit appears to
  address them.
- A review action against a stale revision is refused and reports the current revision so that a
  prior human edit cannot be overwritten.
- If the operating environment cannot provide a non-empty user identity, state-changing review and
  approval actions are refused until the reviewer supplies an actor ID explicitly.
- An interrupted write leaves either the previous valid revision or the complete new revision
  available, never a partially updated artifact.
- Reopening an approved Intent for change creates a new working revision and preserves the
  immutable approved snapshot.
- A request to mark review complete is refused unless every non-rejected item is accepted or
  human-edited and every comment is resolved or withdrawn.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST accept an existing, schema-valid Intent Draft as the starting artifact
  for review.
- **FR-002**: The system MUST assign and preserve stable identities for the Intent, every item,
  comment, revision, and approval record.
- **FR-003**: The system MUST present every active Intent Item with its kind, statement, provenance,
  source reference, resolution status, and current review status.
- **FR-004**: A human reviewer MUST be able to accept, edit, comment on, or reject an individual
  Intent Item.
- **FR-005**: Every governed state-changing operation MUST create a new revision and mechanically record
  its actor type, operation type, affected record, and prior revision.
- **FR-021**: For human operations, the system MUST automatically use the operating environment's
  current user name as the actor ID. It MUST label the identity as locally sourced and not
  authenticated. If no non-empty user name is available, an explicit actor ID MUST be supplied
  before a state-changing operation can proceed.
- **FR-006**: Human edits MUST be distinguishable from source content and AI-authored content and
  MUST NOT be overwritten by any automated action in this feature.
- **FR-007**: Rejection MUST require a rationale, exclude the item from active approval content,
  and preserve the item and its revision history. A human reviewer MAY reverse rejection by
  accepting or editing the same item; doing so MUST include it in approval content and clear the
  rejection rationale. Rejected status alone MUST NOT block approval.
- **FR-008**: Comments MUST have independent identities, target exactly one Intent Item, record
  author type and creation revision, and have an explicit open, resolved, or withdrawn status.
  Only a human reviewer may resolve or withdraw a comment; the operation MUST record a reason and
  MAY reference the revision that addressed it.
- **FR-009**: Every interface MUST obtain a core-issued edit preview containing the before/after
  comparison and a short-lived, one-use preview token before applying an item edit. The token MUST
  bind the Intent, expected revision, item, actor, and proposed statement; applying an edit with a
  missing, expired, used, or mismatched token MUST be refused.
- **FR-010**: The system MUST validate artifact structure, identity uniqueness, allowed state
  transitions, required fields, and referential integrity before persisting a state change.
- **FR-011**: Approval eligibility MUST require at least one non-rejected item, every non-rejected
  item reviewed and accepted or human-edited, no open comments including comments on rejected
  items, a valid current revision, and complete provenance.
- **FR-012**: Approval MUST use a two-step core-governed flow. The core MUST first prepare the exact
  eligible revision, hashes, blockers, and a short-lived one-use confirmation challenge. The final
  approval MUST require a fresh response from a human through an interactive TTY. Non-TTY approval,
  and confirmation supplied by a skill or automated actor, MUST be refused.
- **FR-013**: Successful approval MUST create a new `approved` revision whose parent is the exact
  human-confirmed `review_complete` revision and whose Intent content is unchanged. The immutable
  Approved Intent Snapshot MUST record both revision IDs, the content hash, confirmation token ID,
  approving actor, approval time, and validation result.
- **FR-014**: Any normal edit, comment, or review operation after approval MUST create a new working
  revision in `in_review`, preserve every issued snapshot, and invalidate working approval without
  requiring a separate reopen operation.
- **FR-015**: The system MUST refuse direct modification of an Approved Intent Snapshot.
- **FR-016**: Review and approval operations MUST return a structured result that identifies
  success or failure, the resulting revision, affected record IDs, and actionable validation
  findings.
- **FR-017**: A partially reviewed Intent MUST contain enough persisted information to resume the
  review without relying on prior interface or conversation state.
- **FR-018**: The minimum zintent-review process MUST select a Draft and open a Hunk-style
  interactive TUI that supports item navigation, accept, edit, comment, comment resolution or
  withdrawal, reject, and approval confirmation; it MUST record actions through governed
  operations and summarize remaining blockers on exit.
- **FR-019**: The minimum zintent-approve process MUST check approval eligibility, present blockers
  or the exact eligible revision, open the interactive confirmation step, and return the issued
  snapshot. The skill MUST NOT accept or synthesize the confirmation response itself.
- **FR-020**: Review actions performed through any interface included in this feature MUST use the
  same validation, transition, provenance, and revision rules.
- **FR-022**: Intent Documents MUST use the explicit lifecycle states `draft`, `in_review`,
  `review_complete`, and `approved`. Starting review moves `draft` to `in_review`. An explicit
  review-completion operation moves `in_review` to `review_complete` only after all non-rejected
  items are accepted or human-edited and no comments remain open. Approval moves only an eligible
  `review_complete` revision to a new `approved` child revision after human confirmation. Any
  permitted content or review change from `review_complete` or `approved` creates a new working
  revision in `in_review`; an issued snapshot never changes state.

### Scope Boundaries

- The feature includes a minimal Intent schema, loading a fixture or manually authored Draft,
  item-level review, validation, revision history, approval eligibility, immutable snapshot output,
  and the minimum zintent-review and zintent-approve processes.
- The feature excludes AI generation of an Intent Draft, AI-generated revision proposals, semantic
  contradiction detection, Planner or Spec Kit integration, multi-reviewer approval policy,
  authentication and authorization, remote collaboration, and production deployment.
- A minimal Hunk-style interactive TUI is required for the primary review and approval journey.
  Advanced navigation, customization, search, and history visualization are deferred.

### Key Entities *(include if feature involves data)*

- **Intent Document**: The working review artifact, identified by an Intent ID and schema version;
  contains one of the lifecycle states `draft`, `in_review`, `review_complete`, or `approved`, plus
  its current revision, source references, items, comments, and approval references.
- **Intent Item**: A stable, individually reviewable statement with an item ID, kind, statement,
  provenance, resolution status, rationale, source reference, review status, approval-inclusion
  status, and optional supersession relationships. A rejected item is retained but excluded from
  approval content.
- **Comment**: An independently identified reviewer concern targeting one item, with body, author
  type, status, creation revision, closure reason, and optional resolution revision. Status changes
  from open to resolved or withdrawn only through an explicit human operation.
- **Revision**: An immutable record of one or more governed changes, including its revision
  identity, parent revision, actor type, operations, timestamp, and resulting content identity.
- **Approval**: A record binding a human approving actor and validation result to one exact revision
  lineage and content hash. It records the human-confirmed review-complete revision, resulting
  approved revision, confirmation token ID, and locally sourced unauthenticated actor.
- **Approved Intent Snapshot**: An immutable artifact containing the approved Intent content and its
  Approval record; it remains separate from later working revisions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In an acceptance test, a reviewer can complete accept, edit, comment, and reject
  actions on four separate items and recover all four outcomes after reopening the Intent.
- **SC-002**: Across the defined invalid and ineligible approval scenarios, 100% of approval
  attempts are blocked and identify every applicable blocking record.
- **SC-003**: Across the defined eligible scenarios, 100% of explicit human approvals produce a
  snapshot whose content identity matches the selected revision.
- **SC-004**: In 100% of post-approval change tests, the original snapshot remains unchanged and the
  changed working Intent requires a new approval.
- **SC-005**: A reviewer can resume a partially completed review using only persisted artifacts and
  identify the next unresolved item within 30 seconds.
- **SC-006**: All state-changing acceptance tests produce an auditable revision and provenance
  record with no missing actor, operation, target, or parent revision.
- **SC-007**: At least 9 of 10 representative first-time reviewers can complete the primary journey
  from opening a prepared Draft through producing an Approved Intent Snapshot without external
  assistance.
- **SC-008**: A 1,000-record, 10 MiB Intent is validated and initially displayed within 1 second on
  reference development hardware.
- **SC-009**: A normal single-item review operation is durably persisted within 250 milliseconds on
  reference development hardware, excluding human input time.

## Assumptions

- Feature 001 begins with a fixture or manually authored valid Draft; interpretation from natural
  language belongs to Feature 002.
- One human acts as the Intent owner and reviewer in a working review. Multi-reviewer identity,
  permissions, consensus, and concurrent collaboration are deferred.
- Human actor IDs are obtained from the operating environment and provide local audit attribution,
  not proof of identity or authorization.
- Review decisions apply at item level. Final approval applies to one complete Intent revision, not
  to isolated items or sections.
- Review completion is an explicit governed operation rather than a display-only status derived at
  read time.
- Rejecting an item means it is excluded from the active approved content but retained in history.
- Resolving and withdrawing a comment both require a human-provided reason; a related revision is
  optional because a comment may be withdrawn without a content change.
- Time is recorded for audit purposes, but no time-based expiration or retention policy is required
  in this feature.
- The walking skeleton targets local, single-user operation and does not require network services,
  user accounts, or organizational compliance controls.
- The required Hunk-style TUI may remain visually minimal if it exposes item navigation, all
  required review actions and outcomes, and approval confirmation; visual polish and advanced
  productivity features are not acceptance conditions.

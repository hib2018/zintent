---

description: "Dependency-ordered tasks for Intent Review Walking Skeleton"
---

# Tasks: Intent Review Walking Skeleton

**Input**: Design documents from specs/001-intent-review-skeleton/

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are mandatory because the Constitution requires schema, transition, invariant,
contract, routing, behavior, and failure-path verification. Write each listed test before its
corresponding implementation and confirm it fails for the intended missing behavior.

**Organization**: Tasks are grouped by user story. Zig owns all domain rules and persistence; Go
owns CLI/TUI presentation and bounded core-process orchestration.

## Format: ID, parallel marker, story, description

- [P] means the task may run concurrently because it targets different files and has no dependency
  on an incomplete task in the same group.
- [US1], [US2], and [US3] map directly to the prioritized stories in spec.md.

## Phase 1: Setup

**Purpose**: Establish the two-project Go/Zig build and verification skeleton.

- [X] T001 Pin Zig 0.16.0 and Go 1.27.1 in .tool-versions, install any missing pinned toolchain, and create core/src/, core/tests/, tui/cmd/zintent/, tui/internal/, tests/contract/fixtures/, tests/integration/, and tests/fixtures/
- [X] T002 Initialize the Zig 0.16.0 build, zintent-core executable, and core test steps in build.zig and build.zig.zon
- [X] T003 [P] Initialize the Go 1.27.1 module and pin Bubble Tea v2.0.8 with compatible Bubbles and Lip Gloss v2 dependencies in tui/go.mod and tui/go.sum
- [X] T004 [P] Add Zig formatting and test commands plus Go formatting, vet, and test commands to ./Makefile
- [X] T005 [P] Configure repository ignores for Zig caches, built binaries, temporary Intent locks, and local fixture output in ./.gitignore
- [X] T006 Add macOS and Linux matrix jobs for Zig and Go build/test validation in .github/workflows/ci.yml

**Checkpoint**: Both empty executables build and both test runners execute on supported platforms.

---

## Phase 2: Foundational

**Purpose**: Build the versioned process boundary, trusted domain types, and durable store primitives
that block every user story.

**CRITICAL**: No user-story implementation starts until the shared contract fixtures pass in both
languages and the Zig core is the only path capable of changing an Intent.

- [X] T007 [P] Create valid and invalid protocol-envelope and strict command-payload fixtures from contracts/message.schema.json and contracts/command.schema.json in tests/contract/fixtures/messages/
- [X] T008 [P] Create valid and invalid Intent, HEAD, provenance, source-reference, edit-preview, approval-confirmation, result, and snapshot fixtures from every contracts/*.schema.json artifact schema in tests/contract/fixtures/artifacts/
- [X] T009 [P] Create RFC 8785 and SHA-256 golden vectors including key-order, whitespace, Unicode, and array-order cases in tests/contract/fixtures/hashes/
- [X] T010 Write failing strict protocol decoding, envelope/payload operation equality, external schema-reference, unknown nested-field, and request/response correlation tests in core/tests/contract.zig
- [X] T011 [P] Write failing Go protocol compatibility tests against the shared fixture corpus in tests/contract/compatibility_test.go
- [X] T012 Implement strict protocol message decoding, 16 MiB limits, unknown-field rejection, request-ID echoing, and one-response stdout discipline in core/src/protocol.zig
- [X] T013 [P] Implement matching strict Go request/response types and schema identifiers in tui/internal/protocol/message.go
- [X] T014 Implement the no-shell, 10-second, bounded stdin/stdout/stderr Zig subprocess runner with cancellation in tui/internal/runner/core.go
- [X] T015 [P] Define ID, timestamp, human OperationActor, independent ContentOrigin, Provenance, Finding, lifecycle, and operation types in core/src/model.zig
- [X] T016 [P] Define strict command payloads with expected revision, operation ID, edit-preview capability, and approval-confirmation capability types in core/src/command.zig
- [X] T017 Implement RFC 8785 canonicalization projections and SHA-256 verification against golden vectors in core/src/hashing.zig
- [X] T018 Implement per-Intent locking, core-owned transient capability registry with atomic one-use consumption and expiry cleanup, same-directory temp publication, file and directory sync, exclusive immutable-file creation, and HEAD-last replacement in core/src/store.zig
- [X] T019 Implement core operation dispatch and stable success/error envelope mapping in core/src/main.zig
- [X] T020 Wire Go command parsing, OS actor discovery with explicit fallback, coarse exit statuses, and human/JSON output selection in tui/cmd/zintent/main.go and tui/internal/output/result.go

**Checkpoint**: Go can invoke read-only protocol_info through Zig; both languages pass the same
strict contract corpus; no mutation bypass exists.

---

## Phase 3: User Story 1 - Review Intent Items (Priority: P1) MVP

**Goal**: Open a valid Draft in the Hunk-style TUI and persist human accept, edit, comment,
resolve/withdraw, and reject decisions as governed immutable revisions.

**Independent Test**: Load a prepared Draft, perform each review action on separate items, close and
reopen from verified HEAD, and verify that item states, comments, and provenance are preserved while
stale or invalid actions create no reachable revision. Deep chain inspection remains in US3.

### Tests for User Story 1

- [X] T021 [P] [US1] Write failing typed-decoding, duplicate-ID, broken-reference, empty-statement, and rejected-rationale tests in core/tests/model_validation.zig
- [X] T022 [P] [US1] Write failing draft-to-review and item/comment transition table tests in core/tests/transition.zig
- [X] T023 [P] [US1] Write failing stale-revision, idempotent-retry, operation-ID conflict, and no-partial-publication tests in core/tests/persistence_failure.zig
- [ ] T024 [P] [US1] Write failing public CLI JSON-envelope tests for validate, show, item actions, comment actions, and complete-review syntax in tests/integration/review_cli_test.go
- [ ] T025 [P] [US1] Write failing Bubble Tea reducer tests for navigation, modal cancellation, confirmation, stable-ID selection, resize, and stale reload in tui/internal/ui/review_test.go
- [ ] T026 [P] [US1] Write failing 80x24, narrow, monochrome, long-statement, and empty-findings render tests in tui/internal/ui/view_test.go
- [ ] T027 [US1] Write the failing end-to-end accept/edit-preview/edit/comment/resolve/reject/quit/reopen-from-HEAD journey in tests/integration/review_journey_test.go

### Implementation for User Story 1

- [X] T028 [P] [US1] Implement strict Intent revision decoding and structural/referential validation in core/src/validation.zig
- [X] T029 [P] [US1] Implement Draft, Intent Item, Comment, Revision, and HEAD manifest domain constructors in core/src/model.zig
- [ ] T030 [US1] Implement start_review, accept_item, preview_edit, capability-bound edit_item, reject_item, add_comment, resolve_comment, and withdraw_comment transition rules in core/src/transition.zig
- [ ] T031 [US1] Implement independent ContentOrigin plus mechanically derived human OperationActor provenance, UUID v7 allocation, operation idempotency, capability issuance/consumption, and parent revision creation in core/src/command.zig
- [ ] T032 [US1] Integrate review mutations with lock/revalidate/publish/HEAD-last storage flow and verified HEAD loading needed to reopen the current review in core/src/store.zig and core/src/main.zig
- [ ] T033 [P] [US1] Implement Go CLI subcommands for validate, show, item accept/edit-preview/edit/reject, and comment add/resolve/withdraw, requiring the preview token on edit apply, in tui/internal/command/review.go
- [X] T034 [P] [US1] Implement the pure Bubble Tea review model, messages, and update reducer with no filesystem mutation in tui/internal/ui/review.go
- [X] T035 [US1] Implement Hunk-style header, item list, before/after detail, findings footer, narrow layout, and keyboard help in tui/internal/ui/view.go
- [ ] T036 [US1] Connect TUI edit preview and all confirmed actions to the bounded Zig runner, consume the exact preview token, reload canonical state after success, and never replay stale mutations in tui/internal/ui/commands.go
- [ ] T037 [US1] Implement verified HEAD reopening, immediate persistence, safe quit summary, terminal restoration, and TTY refusal behavior in tui/cmd/zintent/main.go
- [ ] T038 [US1] Create .agents/skills/zintent-catalog.md and register the zintent-review skill; create its explicit artifact contract, allowed CLI calls, exclusions, and blocker summary behavior in .agents/skills/zintent-review/SKILL.md

**Checkpoint**: User Story 1 is a usable MVP for item-level human intervention and is independently
demonstrable without approval or downstream adapters.

---

## Phase 4: User Story 2 - Validate and Approve Reviewed Intent (Priority: P2)

**Goal**: Enforce review completion and approval eligibility, require fresh human confirmation of
an exact revision, and publish an immutable content-addressed Approved Intent Snapshot.

**Independent Test**: Prove every invalid/open/unreviewed case is blocked, then approve one eligible
revision and verify its actor, hashes, validation result, content-addressed filename, immutability,
idempotent retry, and invalidation after a later working change.

### Tests for User Story 2

- [ ] T039 [P] [US2] Write failing review-completion and approval eligibility matrix tests for all blockers in core/tests/approval.zig
- [ ] T040 [P] [US2] Write failing approved-content projection and snapshot hash golden tests in core/tests/hashing.zig
- [ ] T041 [P] [US2] Write failing snapshot exclusive-create, collision, idempotent retry, orphan, and HEAD publication failure tests in core/tests/persistence_failure.zig
- [ ] T042 [P] [US2] Write failing CLI and protocol tests for complete-review, prepare_approval, challenge mismatch/expiry/reuse/staleness, non-TTY refusal, and approval error envelopes in tests/integration/approval_safety_test.go
- [ ] T043 [P] [US2] Write failing TUI blocker display, disabled approval, exact-revision/hash preview, fresh challenge response, and cancellation tests in tui/internal/ui/approval_test.go

### Implementation for User Story 2

- [ ] T044 [P] [US2] Add Approval with confirmed and approved revision lineage, validation result, approved content, and snapshot envelope types in core/src/model.zig
- [ ] T045 [US2] Implement complete_review eligibility, approval as a new approved child revision, and normal review mutation from review_complete or approved into in_review in core/src/transition.zig
- [ ] T046 [US2] Implement prepare_approval and approve_intent with TTY-only one-use challenge issuance/consumption, locked revalidation, confirmed-to-approved revision binding, canonical projections, and approved-content hashing in core/src/command.zig and core/src/hashing.zig
- [ ] T047 [US2] Implement content-addressed snapshot exclusive publication, equivalence checks, and HEAD approved reference update in core/src/store.zig
- [ ] T048 [P] [US2] Implement Go complete-review and approve CLI commands; approval must prepare, display, and read the challenge directly from the same TTY, reject non-TTY use, and expose no confirmation flag/stdin substitute in tui/internal/command/approval.go
- [ ] T049 [US2] Implement TUI completion blockers, exact snapshot preview, fresh typed challenge response, cancellation, and approval result display in tui/internal/ui/approval.go
- [ ] T050 [US2] Register zintent-approve in .agents/skills/zintent-catalog.md and create its eligible input contract, structured findings, and snapshot output in .agents/skills/zintent-approve/SKILL.md; require it to launch but never answer or synthesize the interactive confirmation

**Checkpoint**: User Story 2 independently converts an eligible reviewed Intent into a verified,
immutable snapshot and refuses every unapproved path.

---

## Phase 5: User Story 3 - Resume and Audit a Review (Priority: P3)

**Goal**: Reconstruct the exact review from persisted artifacts, compare revisions, expose complete
provenance, and recover safely from interrupted publication.

**Independent Test**: Stop after a partial review, start a new process with only the store artifact,
locate the next unresolved item within 30 seconds, compare revisions, inspect provenance, and prove
that corrupt HEAD targets and interrupted writes fail safely without losing the last valid state.

### Tests for User Story 3

- [ ] T051 [P] [US3] Write failing revision-chain, HEAD/hash integrity, orphan, and temporary-file recovery tests in core/tests/recovery.zig
- [ ] T052 [P] [US3] Write failing deterministic item-aware revision diff tests in core/tests/diff.zig
- [ ] T053 [P] [US3] Write failing show, validate, and diff JSON contract tests including corrupt-store findings in tests/integration/audit_cli_test.go
- [ ] T054 [P] [US3] Write failing TUI resume-selection, remaining-blocker summary, and normal post-approval mutation-to-in_review tests in tui/internal/ui/resume_test.go
- [ ] T055 [US3] Write the failing process-restart resumption and crash-recovery journey in tests/integration/crash_recovery_test.go

### Implementation for User Story 3

- [ ] T056 [P] [US3] Implement verified HEAD loading, revision-chain traversal, orphan reporting, and safe temporary cleanup in core/src/store.zig
- [ ] T057 [P] [US3] Implement item-aware revision comparisons and auditable change records in core/src/diff.zig
- [ ] T058 [US3] Implement show_intent, validate_intent, and diff_revisions protocol operations with actionable integrity findings in core/src/main.zig
- [ ] T059 [P] [US3] Implement Go show, validate, and diff presentation for human and JSON modes in tui/internal/command/audit.go
- [ ] T060 [US3] Restore TUI state from canonical artifacts, select by stable item ID, summarize remaining blockers, and present the normal governed mutation that creates an in_review child of an approved Intent in tui/internal/ui/review.go

**Checkpoint**: All three stories work across process restarts and expose a complete, verifiable
history without conversational memory.

---

## Phase 6: Polish and Cross-Cutting Concerns

**Purpose**: Close performance, portability, documentation, and constitutional quality gates.

- [ ] T061 [P] Add generated 1,000-record and 10 MiB Intent fixtures plus timing assertions in tests/fixtures/large/ and tests/integration/performance_test.go
- [ ] T062 [P] Add protocol fuzz/property tests for malformed lengths, UTF-8, duplicate names, unknown fields, and extra stdout in core/tests/protocol_fuzz.zig and tui/internal/protocol/fuzz_test.go
- [ ] T063 [P] Add terminal failure and cancellation tests proving restoration and no mutation after timeout in tui/internal/ui/terminal_test.go and tests/integration/core_timeout_test.go
- [ ] T064 Verify native atomic replacement, file/directory sync, lock behavior, and snapshot exclusive-create on macOS and Linux in tests/integration/platform_persistence_test.go
- [ ] T065 Document installation, store layout, CLI commands, key bindings, local unauthenticated actor semantics, and recovery in README.md
- [ ] T066 Execute every scenario in specs/001-intent-review-skeleton/quickstart.md and record deviations in specs/001-intent-review-skeleton/validation-results.md
- [ ] T067 Re-run Constitution gates, verify both skill contracts and every machine-readable contract, and record any approved exceptions in specs/001-intent-review-skeleton/validation-results.md
- [ ] T068 Run the primary Draft-to-snapshot journey with 10 representative first-time reviewers, record completion without external assistance, and verify at least 9 succeed in specs/001-intent-review-skeleton/usability-results.md

**Checkpoint**: The feature satisfies supported-platform, performance, protocol, safety, skill, and
documentation gates.

---

## Dependencies and Execution Order

### Phase Dependencies

- Setup has no dependency.
- Foundational depends on Setup and blocks all user stories.
- US1, US2, and US3 require the Foundation.
- US2 uses reviewed artifacts produced by US1 for the integrated journey, but its eligibility,
  hashing, and snapshot tests can begin after Foundation using fixtures.
- US3 uses revisions from US1 and snapshots from US2 for the integrated journey, but diff and
  integrity work can begin after Foundation using fixtures.
- Polish follows all stories selected for delivery.

### User Story Dependency Graph

    Setup -> Foundation -> US1 review MVP
                        -> US2 approval core
                        -> US3 audit core

    US1 + US2 -> complete review-to-snapshot flow
    US1 + US2 + US3 -> resumable walking skeleton

### Within Each User Story

- Write the listed tests first and verify the intended failure.
- Implement Zig models and rules before exposing them through the protocol.
- Implement core operations before Go CLI/TUI integration.
- Run cross-language fixtures whenever an envelope or payload changes.
- Complete the independent test at each checkpoint before progressing.

### Parallel Opportunities

- T002 through T005 can proceed in parallel after T001.
- T007 through T011 and T013, T015, and T016 can be prepared in parallel.
- US1 test tasks T021 through T026 touch separate test files and can run in parallel.
- After US1 tests exist, Zig validation/model work and Go reducer scaffolding can proceed in
  parallel.
- US2 test tasks T039 through T043 are parallelizable.
- US3 test tasks T051 through T054 and implementation tasks T056, T057, and T059 have separate
  files and can proceed in parallel.
- Cross-cutting tests T061 through T063 can proceed in parallel.

## Parallel Example: User Story 1

    Task: "T021 Write model validation tests in core/tests/model_validation.zig"
    Task: "T022 Write transition tests in core/tests/transition.zig"
    Task: "T024 Write CLI review contract tests in tests/integration/review_cli_test.go"
    Task: "T025 Write Bubble Tea reducer tests in tui/internal/ui/review_test.go"

## Parallel Example: User Story 2

    Task: "T039 Write approval eligibility matrix tests in core/tests/approval.zig"
    Task: "T040 Write snapshot hash vectors in core/tests/hashing.zig"
    Task: "T042 Write approval CLI/protocol tests in tests/integration/approval_safety_test.go"
    Task: "T043 Write approval TUI tests in tui/internal/ui/approval_test.go"

## Parallel Example: User Story 3

    Task: "T051 Write recovery tests in core/tests/recovery.zig"
    Task: "T052 Write diff tests in core/tests/diff.zig"
    Task: "T053 Write audit CLI tests in tests/integration/audit_cli_test.go"
    Task: "T054 Write resume TUI tests in tui/internal/ui/resume_test.go"

## Implementation Strategy

### MVP First: User Story 1

1. Complete Setup and Foundation.
2. Write US1 tests and confirm their expected failures.
3. Implement the Zig review operations and durable revision publication.
4. Connect the Go CLI and minimal Hunk-style TUI.
5. Stop and validate US1 independently before implementing approval.

### Incremental Delivery

1. US1 proves item-level intervention and governed review persistence.
2. US2 adds explicit approval and immutable snapshots.
3. US3 proves artifact-only resumption, audit, and recovery.
4. Polish validates scale and native filesystem behavior.

### Parallel Team Strategy

After Foundation, contributors may work on US1 review, US2 approval primitives, and US3 diff and
integrity primitives against contract fixtures. Integrate in priority order so every checkpoint
remains demonstrable.

## Notes

- Every task follows the required checkbox, sequential ID, optional parallel marker, optional story
  label, actionable description, and exact file-path format.
- Go never writes Intent artifacts; Zig never renders UI or interprets skill instructions.
- Contract fixture changes must land before implementations that consume them.
- Do not add AI interpretation, AI revision, Spec Kit adapters, authentication, remote
  collaboration, Windows support, network filesystems, or a database under this task list.

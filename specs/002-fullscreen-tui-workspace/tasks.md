---

description: "Dependency-ordered tasks for Full-screen Intent Workspace"
---

# Tasks: Full-screen Intent Workspace

**Input**: Design documents from `specs/002-fullscreen-tui-workspace/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are mandatory because the plan and Constitution require reducer, contract, PTY, failure-path, semantic-parity, and performance verification. Write each listed test before its corresponding implementation and confirm the intended failure.

**Organization**: Tasks are grouped by user story. Zig remains the sole owner of domain and persistence rules; Go owns full-screen presentation and bounded core-process orchestration.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because the task targets different files and does not depend on an incomplete task in the same phase.
- **[Story]**: Maps the task to User Story 1, 2, or 3.
- Every task names the concrete file path to change.

## Phase 1: Setup

**Purpose**: Prepare the existing Go/Zig projects for a multi-screen full-screen workspace without adding a new executable or persistence layer.

- [X] T001 Verify Feature 001 baseline tests and record the pre-feature result in specs/002-fullscreen-tui-workspace/validation-results.md
- [X] T002 Promote the existing Bubble Tea, Bubbles, and Lip Gloss dependencies required by workspace screens and text inputs in tui/go.mod and tui/go.sum
- [X] T003 [P] Add shared full-screen golden, PTY, workspace, import, audit, and recovery fixture directories in tests/fixtures/workspace/ and tui/internal/ui/testdata/
- [X] T004 [P] Register new Zig workspace, import, audit, and recovery modules plus their test targets in core/src/root.zig and build.zig
- [X] T005 [P] Extend repository ignores only for generated PTY transcripts and local workspace fixtures in .gitignore

**Checkpoint**: Existing Feature 001 behavior remains green and all new test locations are discoverable.

---

## Phase 2: Foundational

**Purpose**: Establish shared protocol types, root UI routing, correlated command execution, and terminal ownership required by every story.

**CRITICAL**: No user-story implementation begins until this phase is complete.

- [ ] T006 [P] Add valid and invalid fixtures for every operation in contracts/workspace-command.schema.json under tests/contract/fixtures/workspace/
- [X] T007 Write failing Zig strict-decoding and mutation-classification tests for all workspace operations in core/tests/workspace_contract.zig
- [ ] T008 [P] Write failing Go cross-language compatibility tests for workspace command and result envelopes in tests/contract/workspace_compatibility_test.go
- [X] T009 [P] Define workspace entries, audit summaries, recovery observations, and transient capability bindings in core/src/model.zig and core/src/command.zig
- [X] T010 Implement additive operation decoding, validation, mutation classification, and protocol_info capability reporting in core/src/model.zig, core/src/command.zig, and core/src/protocol.zig
- [X] T011 [P] Define typed Go workspace requests, results, findings, request correlation metadata, and screen-independent canonical Intent views in tui/internal/protocol/workspace.go
- [ ] T012 [P] Write failing reducer tests for root screen routing, navigation stack, stable-ID selection, modal exclusivity, key-release rejection, and repeated-confirm suppression in tui/internal/ui/workspace_test.go
- [X] T013 Implement the root WorkspaceModel, explicit Screen enum, navigation stack, shared header/status/help frame, and single active modal routing in tui/internal/ui/workspace.go and tui/internal/ui/navigation.go
- [ ] T014 [P] Implement program-lifetime cancellation, typed request correlation, one-active-mutation enforcement, late-response rejection, and indeterminate-result handling in tui/internal/command/executor.go
- [ ] T015 Implement alternate-screen ownership, interactive-TTY refusal, cursor behavior, graceful shutdown, and canonical reload recovery in tui/cmd/zintent/main.go and tui/internal/ui/workspace.go
- [ ] T016 [P] Add wide, narrow, loading, empty, long-text, error, and blocker-heavy root frame golden tests in tui/internal/ui/workspace_view_test.go and tui/internal/ui/testdata/

**Checkpoint**: A full-screen shell can route among placeholder screens, owns the terminal safely, and cannot mutate artifacts directly.

---

## Phase 3: User Story 1 - Reviewから承認までTUIで完結する (Priority: P1) 🎯 MVP

**Goal**: Open one Draft and complete every review, comment, completion, approval, and snapshot-confirmation action without leaving the full-screen workspace.

**Independent Test**: Open a valid Draft, accept/edit/comment/resolve/reject items, complete review, type the fresh approval challenge, and inspect the issued snapshot within one TUI process.

### Tests for User Story 1

- [ ] T017 [P] [US1] Write failing table tests for edit, comment, reject, closure, completion, and approval modal state machines in tui/internal/ui/modals_test.go
- [ ] T018 [P] [US1] Write failing command-adapter tests proving exact preview/token/reason/challenge dispatch, no generic-message approval, and canonical reload after every mutation in tui/internal/command/review_flow_test.go
- [ ] T019 [P] [US1] Write failing review and comment screen reducer tests for stable-ID navigation, blocker jumps, focus, scrolling, cancellation, stale reload, and double-submit rejection in tui/internal/ui/review_workspace_test.go
- [ ] T020 [P] [US1] Write failing completion and approval reducer/view tests for exact revision/hash display, non-TTY refusal, expiry, mismatch, cancellation, and snapshot result navigation in tui/internal/ui/approval_workspace_test.go
- [ ] T021 [US1] Write the failing PTY-backed Draft-to-snapshot journey, terminal restoration cases, and concurrent stale mutation scenario in tests/integration/fullscreen_review_journey_test.go

### Implementation for User Story 1

- [ ] T022 [P] [US1] Implement dedicated edit, comment, reject, comment-closure, completion, and approval modal states with explicit phases in tui/internal/ui/modals.go
- [ ] T023 [P] [US1] Implement review list/detail panes, viewport scrolling, stable item selection, full statement display, provenance details, and contextual keys in tui/internal/ui/review.go and tui/internal/ui/view.go
- [ ] T024 [P] [US1] Implement comment list/detail navigation, add input, resolve/withdraw reason validation, and target-item return behavior in tui/internal/ui/comments.go
- [ ] T025 [US1] Connect edit input to preview_edit, show the exact before/after and expiry, submit edit_item with the unchanged token, and discard previews on cancel/stale/navigation in tui/internal/command/review.go and tui/internal/ui/modals.go
- [ ] T026 [US1] Connect accept, reject, add-comment, resolve, withdraw, and post-approval review mutations to the shared executor and canonical reload flow in tui/internal/command/review.go
- [ ] T027 [US1] Implement completion blocker presentation, stable-record jump, explicit completion confirmation, and complete_review reload transition in tui/internal/ui/completion.go and tui/internal/command/review.go
- [ ] T028 [US1] Implement foreground-TTY approval preparation, exact actor/revision/hash/count preview, KeyPress-only fresh challenge input, token disposal rules, and approve_intent submission in tui/internal/ui/approval.go and tui/internal/command/approval.go
- [ ] T029 [US1] Implement approval success display and verified snapshot inspection navigation in tui/internal/ui/snapshot.go and tui/internal/command/audit.go
- [ ] T030 [US1] Integrate dashboard-to-review/comments/completion/approval/snapshot navigation and canonical lifecycle refresh in tui/internal/ui/dashboard.go and tui/internal/ui/workspace.go
- [ ] T031 [US1] Pass the PTY-backed one-process Draft-to-snapshot journey and preserve all existing CLI/skill review and approval tests in tests/integration/fullscreen_review_journey_test.go and tests/integration/approval_safety_test.go

**Checkpoint**: User Story 1 is a usable MVP; Draft-to-Approved-Snapshot requires no shell return and no domain rule exists only in the TUI.

---

## Phase 4: User Story 2 - 履歴・検証・復旧をTUIから監査する (Priority: P2)

**Goal**: Inspect verified history, item-aware diffs, provenance, validation, snapshots, and recovery candidates from the same workspace.

**Independent Test**: Open a multi-revision Intent, compare two revisions, inspect actor/provenance and snapshot linkage, display integrity findings, and clean only an explicitly confirmed unchanged temporary candidate.

### Tests for User Story 2

- [ ] T032 [P] [US2] Write failing Zig tests for reachable-chain summaries, verified revision/snapshot inspection, orphan separation, path traversal refusal, and integrity errors in core/tests/audit_workspace.zig
- [ ] T033 [P] [US2] Write failing Zig recovery observation/token tests for exact candidate binding, expiry, HEAD change, candidate change, exclusive locking, and protected artifact refusal in core/tests/recovery_workspace.zig
- [ ] T034 [P] [US2] Write failing protocol integration tests for list_revisions, inspect_revision, inspect_snapshot, recovery_status, and cleanup_temporary_files envelopes in tests/integration/workspace_audit_protocol_test.go
- [ ] T035 [P] [US2] Write failing history/diff/provenance, validation, snapshot, and recovery reducer/golden tests in tui/internal/ui/audit_workspace_test.go and tui/internal/ui/testdata/
- [ ] T036 [US2] Write the failing PTY audit-and-recovery journey proving confirmed cleanup and terminal restoration in tests/integration/fullscreen_audit_journey_test.go

### Implementation for User Story 2

- [ ] T037 [P] [US2] Implement verified reachable revision summaries, orphan separation, ID-only revision resolution, and snapshot linkage verification in core/src/audit.zig and core/src/store.zig
- [ ] T038 [P] [US2] Implement recovery_status observation hashing, candidate IDs, short-lived token issuance, and non-destructive reporting in core/src/recovery.zig
- [ ] T039 [US2] Implement locked cleanup_temporary_files revalidation and deletion of only unchanged explicitly selected core temporary files in core/src/recovery.zig and core/src/store.zig
- [ ] T040 [US2] Dispatch list_revisions, inspect_revision, inspect_snapshot, recovery_status, and cleanup_temporary_files with stable findings and errors in core/src/main.zig
- [ ] T041 [P] [US2] Implement Go typed audit and recovery command adapters with JSON/human parity in tui/internal/command/audit.go and tui/internal/command/recovery.go
- [ ] T042 [P] [US2] Implement history selection, revision metadata, two-revision item-aware diff, and provenance detail screens in tui/internal/ui/history.go
- [ ] T043 [P] [US2] Implement validation finding filters, severity/record navigation, and full message display in tui/internal/ui/validation.go
- [ ] T044 [P] [US2] Implement verified snapshot detail, approval linkage, actor, hash, and validation-result display in tui/internal/ui/snapshot.go
- [ ] T045 [US2] Implement recovery status, candidate selection, explicit cleanup confirmation, stale observation handling, and protected-orphan display in tui/internal/ui/recovery.go
- [ ] T046 [US2] Integrate dashboard audit navigation and pass the PTY audit/recovery journey in tui/internal/ui/dashboard.go and tests/integration/fullscreen_audit_journey_test.go

**Checkpoint**: User Story 2 independently provides a complete verified audit and safe recovery experience without direct Go filesystem access.

---

## Phase 5: User Story 3 - 複数Intentをワークスペースで管理する (Priority: P3)

**Goal**: List, search, import, select, and resume multiple local Intents from one bounded workspace.

**Independent Test**: Open a workspace containing valid, corrupt, approved, and in-review Intents; filter and select by stable ID; preview and atomically import a Draft; reopen the imported review at its unresolved item.

### Tests for User Story 3

- [ ] T047 [P] [US3] Write failing Zig tests for bounded direct-child discovery, deterministic ordering, corrupt-entry findings, symlink/path escape refusal, and workspace-root errors in core/tests/workspace.zig
- [ ] T048 [P] [US3] Write failing Zig tests for inspect_draft/import_draft source binding, collision, expiry, source change, atomic publication, rollback, source preservation, and idempotency in core/tests/import.zig
- [ ] T049 [P] [US3] Write failing protocol integration tests for list_intents, inspect_draft, and import_draft JSON contracts in tests/integration/workspace_import_protocol_test.go
- [ ] T050 [P] [US3] Write failing Intent-list search, stable selection, corrupt-entry, import-modal, and resume reducer/golden tests in tui/internal/ui/intent_list_test.go and tui/internal/ui/testdata/
- [ ] T051 [US3] Write the failing PTY workspace list/import/resume journey in tests/integration/fullscreen_workspace_journey_test.go

### Implementation for User Story 3

- [ ] T052 [P] [US3] Implement canonical bounded workspace discovery, direct-child verification, deterministic ordering, summaries, and per-entry findings in core/src/workspace.zig
- [ ] T053 [P] [US3] Implement inspect_draft validation, proposed core-owned IDs/destination, source hashing, collision findings, and one-use import capability in core/src/import.zig
- [ ] T054 [US3] Implement locked import_draft source revalidation, operation idempotency, temporary sibling construction, fsync, atomic rename, rollback, and source preservation in core/src/import.zig and core/src/store.zig
- [ ] T055 [US3] Dispatch list_intents, inspect_draft, and import_draft with strict result envelopes and stable errors in core/src/main.zig
- [ ] T056 [P] [US3] Implement Go workspace listing and two-step Draft import adapters in tui/internal/command/workspace.go
- [ ] T057 [P] [US3] Implement Intent-list search/filter, lifecycle/blocker/approval summaries, stable-ID selection, corrupt-entry display, and workspace reload in tui/internal/ui/intent_list.go
- [ ] T058 [US3] Implement Draft source input, inspect preview, destination/hash/finding confirmation, import submission, and failure-safe modal reset in tui/internal/ui/import.go
- [ ] T059 [US3] Implement imported and existing Intent dashboard opening, canonical resume, and first-unresolved-record fallback in tui/internal/ui/workspace.go and tui/internal/ui/review.go
- [ ] T060 [US3] Add the public interactive workspace command while preserving path-targeted review and machine-readable CLI commands in tui/cmd/zintent/main.go
- [ ] T061 [US3] Pass the PTY workspace list/import/resume journey and verify no Go code directly writes Intent artifacts in tests/integration/fullscreen_workspace_journey_test.go

**Checkpoint**: All three stories operate independently and compose into the complete local full-screen Intent workspace.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Close accessibility, performance, portability, documentation, and Constitution gates across all stories.

- [ ] T062 [P] Add 1,000-item reducer/render latency assertions and visible-row rendering checks in tui/internal/ui/workspace_performance_test.go
- [ ] T063 [P] Add PTY tests for normal quit, Ctrl+C, timeout, core crash, alternate-screen restoration, cursor restoration, and non-TTY refusal in tests/integration/fullscreen_terminal_test.go
- [ ] T064 [P] Add property/fuzz tests for arbitrary key sequences, late request IDs, modal transitions, and invalid workspace responses in tui/internal/ui/workspace_fuzz_test.go and core/tests/workspace_fuzz.zig
- [ ] T065 [P] Update installation, workspace command, screen map, key bindings, Draft import, approval, audit, and recovery documentation in README.md and docs/README.md
- [ ] T066 Verify all Feature 001 CLI and zintent-review/zintent-approve skill contracts remain compatible in tests/integration/review_cli_test.go and .agents/skills/zintent-catalog.md
- [ ] T067 Execute every scenario in specs/002-fullscreen-tui-workspace/quickstart.md and record results and deviations in specs/002-fullscreen-tui-workspace/validation-results.md
- [ ] T068 Re-run Constitution gates, JSON Schema validation, cross-interface semantic parity, macOS/Linux CI, and first-time-user completion checks in specs/002-fullscreen-tui-workspace/validation-results.md

**Checkpoint**: The workspace satisfies performance, terminal safety, interface parity, documentation, and governance requirements.

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup has no dependency.
- Foundational depends on Setup and blocks every user story.
- US1, US2, and US3 require Foundational.
- US1 is the recommended MVP and proves full-screen review-to-snapshot.
- US2 can begin after Foundational using audit fixtures, but dashboard/snapshot integration uses US1 navigation primitives.
- US3 can begin after Foundational using workspace fixtures, but opening an Intent reuses US1 dashboard/review primitives.
- Polish follows all selected stories.

### User Story Dependency Graph

```text
Setup → Foundation → US1 review-to-snapshot MVP
                   → US2 audit/recovery
                   → US3 workspace/import

US1 navigation + US2 + US3 → complete full-screen workspace
```

### Within Each User Story

- Write and run the listed failing tests before implementation.
- Implement Zig core operations before exposing them through Go adapters and screens.
- Models and command adapters precede screen integration.
- Every mutation success is followed by canonical reload.
- Complete the independent journey before declaring the story done.

### Parallel Opportunities

- T003 through T005 target different setup files.
- T006, T008, T009, T011, T012, and T016 can proceed independently after setup.
- US1 test tasks T017 through T020 are parallelizable.
- US2 core audit/recovery tests and UI tests T032 through T035 are parallelizable.
- US3 discovery/import/protocol/UI tests T047 through T050 are parallelizable.
- Polish tests T062 through T065 target separate files.

## Parallel Example: User Story 1

```text
Task: "T017 Write modal state-machine tests in tui/internal/ui/modals_test.go"
Task: "T018 Write exact-dispatch adapter tests in tui/internal/command/review_flow_test.go"
Task: "T019 Write review/comments reducer tests in tui/internal/ui/review_workspace_test.go"
Task: "T020 Write completion/approval tests in tui/internal/ui/approval_workspace_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T032 Write verified audit tests in core/tests/audit_workspace.zig"
Task: "T033 Write recovery-token tests in core/tests/recovery_workspace.zig"
Task: "T034 Write audit protocol tests in tests/integration/workspace_audit_protocol_test.go"
Task: "T035 Write audit screen tests in tui/internal/ui/audit_workspace_test.go"
```

## Parallel Example: User Story 3

```text
Task: "T047 Write bounded discovery tests in core/tests/workspace.zig"
Task: "T048 Write two-step import tests in core/tests/import.zig"
Task: "T049 Write import protocol tests in tests/integration/workspace_import_protocol_test.go"
Task: "T050 Write Intent-list/import reducer tests in tui/internal/ui/intent_list_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete US1 through T031.
3. Validate the one-process Draft-to-snapshot PTY journey.
4. Demonstrate full-screen review and approval before adding audit and multi-Intent management.

### Incremental Delivery

1. Foundation: safe multi-screen shell and correlated core execution.
2. US1: review-to-snapshot full-screen MVP.
3. US2: verified audit and safe recovery.
4. US3: bounded workspace discovery and atomic Draft import.
5. Polish: performance, PTY safety, compatibility, documentation, and governance.

## Notes

- `[P]` tasks must not edit the same file or depend on an incomplete task.
- TUI/session state is transient; no task may serialize it into Intent domain artifacts.
- No TUI task may implement validation, state transitions, hashing, persistent IDs, approval eligibility, or storage mutation.
- Approval challenge input must remain a direct human foreground-TTY action.
- Orphan revisions are reported but not deleted by this feature.

# Quickstart Validation: Intent Review Walking Skeleton

This guide defines end-to-end checks for the implemented feature. It assumes macOS or Linux, a
TTY terminal, Zig 0.16.0, and Go 1.27.1.

## 1. Build and Verify

```sh
zig version
go version
zig build
zig build test
cd tui && go test ./... && go build -o ../zig-out/bin/zintent ./cmd/zintent
cd ..
```

Expected: Zig is 0.16.0, Go is 1.27.1, both executables build, and all core, protocol, CLI, and TUI
tests pass. See [data-model.md](data-model.md) and
[cli.md](contracts/cli.md) for the normative design contracts.

## 2. Validate and Inspect a Draft

```sh
zig-out/bin/zintent validate tests/fixtures/valid-draft --output json
zig-out/bin/zintent show tests/fixtures/valid-draft --output json
```

Expected: each command writes exactly one JSON result envelope to stdout. Validation succeeds, and
show reports the Intent ID, current revision, `draft` state, item IDs, provenance, and review status.

## 3. Complete the Human Review

```sh
zig-out/bin/zintent review tests/fixtures/valid-draft
```

In the Hunk-style TUI:

1. Confirm the header shows the local OS actor as unauthenticated.
2. Navigate with arrows or `j/k`.
3. Accept one item with `a`.
4. Edit one item with `e`; inspect the core-issued before/after hunk before confirming the exact
   one-use preview.
5. Add a comment with `c`, then resolve it with `r` and a reason.
6. Reject one item with `x` and provide a rationale.
7. Press `f` to complete review.
8. Press `p`, verify the exact revision/hash and fresh challenge, and type the requested response.
9. Quit with `q`.

Expected: every confirmed operation is immediately persisted as a new revision. Quitting needs no
save prompt. Completion is blocked until included items are reviewed and comments are closed.
Approval emits a content-addressed immutable snapshot.

## 4. Verify Resumption and Integrity

```sh
zig-out/bin/zintent show tests/fixtures/valid-draft --output json
zig-out/bin/zintent diff tests/fixtures/valid-draft --output json
zig-out/bin/zintent validate tests/fixtures/valid-draft --output json
```

Expected:

- lifecycle is `approved`;
- every action records actor, operation, target, and parent revision;
- the rejected item remains in history but is absent from approved content;
- the resolved comment retains its closure reason;
- snapshot filename and SHA-256 match recomputed canonical approved content; and
- reopening the process loses no state.

## 5. Exercise Safety Gates

Use the current revision reported by show as `CURRENT_REV`, and an older revision as `STALE_REV`.

```sh
zig-out/bin/zintent item accept tests/fixtures/valid-draft ITEM_ID \
  --expected-revision STALE_REV --operation-id OPERATION_ID --output json
```

Expected: exit status 4, error code `stale_revision`, and no reachable revision is created.

For a fixture with an open comment, run the interactive command from a TTY:

```sh
zig-out/bin/zintent approve tests/fixtures/open-comment \
  --revision CURRENT_REV --operation-id OPERATION_ID --output json
```

Expected: exit status 5, error code `approval_ineligible`, a finding identifying the open comment,
and no snapshot.

Pipe or redirect the same approval command so it has no TTY. Expected: exit status 5, error code
`tty_required`, and no confirmation capability, approved revision, or snapshot. No command-line
flag or stdin payload can replace the interactive challenge response.

Attempt snapshot mutation through every exposed command. Expected: no update command exists,
snapshot input is rejected, and original bytes and hash remain unchanged.

## 6. Verify Interface Parity

Run equivalent accept, edit, comment, closure, reject, completion, and approval journeys through
CLI integration tests and TUI reducer tests.

Expected: both routes produce identical domain result codes, transitions, revision payloads,
provenance, and findings for equivalent commands.

Also run the shared protocol fixture corpus through the Go encoder/decoder and the Zig core.
Expected: both sides accept every valid fixture, reject every invalid fixture, reject unknown major
versions and unknown fields, and correlate each response to the request ID.

## 7. Performance and Layout

Run the generated 1,000-record, 10 MiB fixture test and TUI snapshots at 80x24 and the documented
narrow-terminal size.

Expected: initial validation/display completes within 1 second on reference hardware, normal
single-item persistence completes within 250 ms excluding human input, navigation has no visible
input backlog, and both layouts expose items, details, findings, and key help.

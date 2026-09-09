# CLI Contract: zintent 1.0

## Rules

- The Go `zintent` executable owns the public CLI and TUI; users and skills do not need to invoke
  `zintent-core` directly.
- CLI and TUI encode the same operations over the versioned Zig core protocol.
- Mutation commands require --expected-revision and --operation-id.
- Multiline user text is accepted through a file path or - for stdin.
- There is no generic transition, raw patch, snapshot update, or snapshot delete command.
- With JSON output, stdout contains exactly one result object. Consumers do not parse stderr or
  human messages.
- TUI launch requires a TTY. Non-TTY use returns tty_required.
- Go starts Zig directly with an argument array, never through a shell; each invocation is bounded
  to one request, one response, 16 MiB per stream, and a 10-second deadline.

## Command Surface

    zintent show <intent> [--output human|json]
    zintent validate <intent> [--output human|json]
    zintent review <intent> [--actor-id <fallback>]
    zintent item accept <intent> <item-id> --expected-revision <rev> --operation-id <id>
    zintent item edit <intent> <item-id> --statement-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent item reject <intent> <item-id> --reason-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent comment add <intent> <item-id> --body-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent comment resolve <intent> <comment-id> --reason-file <path|-> [--resolution-revision <rev>] --expected-revision <rev> --operation-id <id>
    zintent comment withdraw <intent> <comment-id> --reason-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent complete-review <intent> --expected-revision <rev> --operation-id <id>
    zintent diff <intent> [--from <rev>] [--to <rev>] [--output human|json]
    zintent approve <intent> --revision <rev> --confirm <rev> --operation-id <id> [--output human|json]

Mutations also accept human or JSON output; JSON is required for Agent skill use.

## Approval Authority

The revision argument selects the candidate and confirm must repeat that exact revision. The domain
layer requires a human actor and rechecks eligibility under the Intent lock. This is local,
unauthenticated attribution rather than proof of identity.

The TUI approval modal displays actor, revision, revision hash, approved-content hash preview, and
blockers. A fresh human keypress dispatches approval. Approval is unavailable while ineligible.

## JSON Result Envelope

All results conform to [result.schema.json](result.schema.json). Required top-level fields are
contract_version, ok, operation, affected_ids, and findings. A failure also has an error containing
a stable code, display message, and retryable flag. Success includes relevant revision IDs and data.
Consumers branch only on stable codes and fields, never display messages.

## Exit Status

| Status | Meaning |
|--------|---------|
| 0 | Operation succeeded |
| 2 | Usage or command syntax error |
| 3 | Artifact or schema validation failure |
| 4 | Stale revision or idempotency conflict |
| 5 | Eligibility or governed transition refusal |
| 6 | Persistence or I/O failure |
| 70 | Unexpected internal failure |

The JSON envelope is authoritative; exit status is only a coarse shell signal.

## Stable Error Codes

Initial codes include invalid_artifact, unsupported_schema, duplicate_id, broken_reference,
invalid_transition, stale_revision, operation_id_conflict, approval_ineligible, open_comment,
unreviewed_item, missing_actor, tty_required, integrity_failure, persistence_failure, and
internal_error.

Adding a code is backward-compatible within contract major version 1. Removing or changing a
code's meaning requires a major-version increment.

## TUI Contract

The header shows Intent ID, current revision, lifecycle, blocker count, and unauthenticated actor.
The body has a selectable item list and hunk detail pane; narrow terminals stack them. The footer
contains findings, status, and key help.

| Key | Browse action |
|-----|---------------|
| arrows, j/k | Navigate |
| Enter | Focus details |
| a | Accept item |
| e | Edit with before/after confirmation |
| c | Add comment |
| x | Reject with rationale |
| r | Resolve or withdraw selected comment |
| d | Show diff |
| f | Complete review |
| p | Approve eligible exact revision |
| ? | Help |
| q | Exit without a save prompt |

Only Press events trigger mutations; Repeat may navigate; Release is ignored. Escape cancels a
modal without dispatch. Success reloads canonical state and retains selection by item ID. A stale
result reloads and displays conflict without replay. Terminal state is restored on exit and panic.

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
    zintent item edit-preview <intent> <item-id> --statement-file <path|-> --expected-revision <rev> [--output human|json]
    zintent item edit <intent> <item-id> --statement-file <path|-> --expected-revision <rev> --preview-token <token> --operation-id <id>
    zintent item reject <intent> <item-id> --reason-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent comment add <intent> <item-id> --body-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent comment resolve <intent> <comment-id> --reason-file <path|-> [--resolution-revision <rev>] --expected-revision <rev> --operation-id <id>
    zintent comment withdraw <intent> <comment-id> --reason-file <path|-> --expected-revision <rev> --operation-id <id>
    zintent complete-review <intent> --expected-revision <rev> --operation-id <id>
    zintent diff <intent> [--from <rev>] [--to <rev>] [--output human|json]
    zintent approve <intent> --revision <rev> --operation-id <id> [--output human|json]

Mutations also accept human or JSON output. Skills may consume JSON for review operations, but an
approval command always requires direct interactive TTY input even when its final result is JSON.

## Approval Authority

The revision argument selects the candidate. The command asks the core to prepare approval, displays
the exact revision and hashes plus a fresh challenge, and reads the response directly from its TTY.
It then submits the bound one-use capability and response. There is no flag, stdin field, environment
variable, or JSON-only route that can supply approval confirmation. Non-TTY invocation returns
`tty_required`. The domain layer rechecks actor, capability, eligibility, and revision under the
Intent lock. Attribution is local and unauthenticated rather than proof of identity.

The TUI approval modal displays actor, revision, revision hash, approved-content hash preview,
challenge, and blockers. A fresh human response dispatches approval. Approval is unavailable while
ineligible. Success creates a new `approved` child revision of the confirmed `review_complete`
revision and then publishes the immutable snapshot.

Item editing has the same two-step semantics in CLI and TUI: obtain a core-issued preview, show the
before/after hunk, then apply only with the matching short-lived one-use preview token.

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
unreviewed_item, missing_actor, tty_required, invalid_confirmation, confirmation_expired,
confirmation_consumed, integrity_failure, persistence_failure, and internal_error.

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

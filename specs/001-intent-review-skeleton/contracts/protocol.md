# Core Process Protocol v1

## Transport

Every core operation is one non-interactive zintent-core invocation:

1. Go writes exactly one UTF-8 RFC 8259 JSON request to stdin and closes stdin.
2. Zig writes exactly one JSON response to stdout and closes stdout.
3. Human diagnostics use stderr.
4. Invalid UTF-8, duplicate names, unknown fields, extra stdout bytes, oversized output, timeout,
   cancellation, or a mismatched request ID fails the invocation without retrying a mutation.

Request, response, and captured stderr are each limited to 16 MiB. The core deadline is 10 seconds.
There is no streaming, batch, notification, persistent connection, network transport, or shell
evaluation in protocol v1.

## Envelope

Requests and responses conform to [message.schema.json](message.schema.json). Envelope version and
payload schema version evolve independently. Each message carries protocol_version 1.0, a UUID
request_id, one operation, a named payload or result schema, and one object payload. A response
request ID must exactly match its request.

Unknown major protocol or document schemas are rejected before an operation. Unknown fields are
rejected in v1. Adding required fields, removing fields, or changing meaning requires a new major.

## Operations

| Operation | Purpose | State-changing |
|-----------|---------|----------------|
| protocol_info | Report versions, operations, limits, and capabilities | No |
| show_intent | Load verified HEAD and current revision | No |
| validate_intent | Validate store, schema, references, state, and hashes | No |
| diff_revisions | Compare two verified revisions | No |
| start_review | Move a valid Draft into review | Yes |
| accept_item | Accept one item | Yes |
| edit_item | Replace one item statement after hunk confirmation | Yes |
| reject_item | Exclude one item with rationale | Yes |
| add_comment | Add an open item comment | Yes |
| resolve_comment | Resolve a comment with reason and optional revision | Yes |
| withdraw_comment | Withdraw a comment with reason | Yes |
| complete_review | Enter review_complete after gate checks | Yes |
| approve_intent | Confirm and snapshot one exact eligible revision | Yes |

Every mutation payload includes intent location, expected revision ID, operation ID, and actor
context. The core rechecks actor policy, locks the Intent, verifies current state, and returns a
result conforming to [result.schema.json](result.schema.json). Go does not infer success from process
exit alone.

## Core Ownership

Only Zig may allocate persistent IDs, validate artifacts, derive provenance, apply transitions,
decide approval eligibility, canonicalize and hash content, acquire locks, or publish revisions,
snapshots, and HEAD. Go owns arguments, actor discovery, deadlines, presentation, TUI state, and
human confirmation. It may hold draft input in memory but cannot serialize an Intent mutation.

## Retry and Failure

Read-only operations may retry after transport failure. Mutations never retry automatically. A
caller may explicitly retry the exact command with the same operation ID; the core returns the
original result if content matches and operation_id_conflict otherwise.

A stale response includes expected and current revisions. The TUI reloads current state and asks
the human to reconsider; it never replays the old decision.

## Compatibility Tests

The shared fixture corpus must prove strict decoding, request-ID correlation, operation/schema
identifiers, size/timeout classification, result and error preservation, valid and invalid artifact
fixtures, and canonical hash vectors in both Zig and Go.

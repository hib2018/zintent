---
name: zintent-review
description: Open or resume a zintent Intent in the Hunk-style TUI and support human item review while preserving core-governed revisions. Use for reviewing an existing Draft or resumable Intent; do not use for approval, Draft generation, or Planner handoff.
---

# Review an Intent

Use the public `zintent` CLI/TUI to let a human review an existing Intent. The skill orchestrates;
the Zig core, reached only through `zintent`, validates and owns every state change and persisted
artifact.

## Input artifact contract

Accept one Intent selector understood by the public CLI. It must resolve through the store's HEAD to
a version `1.0.0` Intent revision whose referenced artifact and hash verify. Supported starting
lifecycle states are `draft`, `in_review`, `review_complete`, and `approved`; the core decides the
permitted transition. An `approved` Intent may be reviewed only through a normal governed mutation,
which creates an `in_review` child and leaves every issued Snapshot unchanged.

Do not treat conversation state, a copied JSON object, a snapshot, or a caller-supplied revision as
canonical. Use `zintent validate <intent> --output json` and, when state is needed, `zintent show
<intent> --output json`. Branch only on result-envelope fields and stable codes.

## Workflow and allowed CLI calls

Launch `zintent review <intent> [--actor-id <fallback>]` in an interactive TTY. The human controls
accept, edit confirmation, comment creation and closure, rejection, and review completion. If direct
CLI assistance is necessary, only these public calls are allowed:

```text
zintent validate <intent> --output json
zintent show <intent> --output json
zintent review <intent> [--actor-id <fallback>]
zintent item accept <intent> <item-id> --expected-revision <rev> --operation-id <id> [--output human|json]
zintent item edit-preview <intent> <item-id> --statement-file <path|-> --expected-revision <rev> [--output human|json]
zintent item edit <intent> <item-id> --statement-file <path|-> --expected-revision <rev> --preview-token <token> --operation-id <id> [--output human|json]
zintent item reject <intent> <item-id> --reason-file <path|-> --expected-revision <rev> --operation-id <id> [--output human|json]
zintent comment add <intent> <item-id> --body-file <path|-> --expected-revision <rev> --operation-id <id> [--output human|json]
zintent comment resolve <intent> <comment-id> --reason-file <path|-> [--resolution-revision <rev>] --expected-revision <rev> --operation-id <id> [--output human|json]
zintent comment withdraw <intent> <comment-id> --reason-file <path|-> --expected-revision <rev> --operation-id <id> [--output human|json]
zintent complete-review <intent> --expected-revision <rev> --operation-id <id> [--output human|json]
```

Text passed to edit, reject, or comment operations must come from the human's explicit input. Obtain
and display an edit preview before applying an edit; use only its matching short-lived one-use token.
Do not silently perform review decisions on the human's behalf.

## Output artifact contract

The authoritative output is the HEAD-selected immutable Intent revision persisted by the core, not
a skill-authored copy. Report `intent_id`, `current_revision_id`, `lifecycle_state`, and the blocker
summary below from a fresh canonical `show`/`validate` result. A successful command result may also
identify `previous_revision_id`, `resulting_revision_id`, and `affected_ids` under result contract
`1.0.0`.

On quit, failure, or incomplete review, return this summary shape:

```json
{
  "intent_id": "<uuid-or-null>",
  "current_revision_id": "<uuid-or-null>",
  "lifecycle_state": "<state-or-null>",
  "blockers": [
    {
      "code": "open_comment|unreviewed_item|invalid_artifact|stale_revision|persistence_failure",
      "count": 1,
      "affected_ids": ["<stable-record-id>"],
      "retryable": false
    }
  ]
}
```

Preserve each finding's `record_type`, `record_id`, and `path` when available. Group identical codes
only if no affected ID or path is lost. Derive `retryable` from the authoritative error envelope;
otherwise use `false`. Do not infer success from display text or an empty screen.

## Stale revision rule

On `stale_revision`, do not replay, rebase, or regenerate the mutation. Discard any selected
revision, preview token, and pending operation derived from it; reload verified HEAD and its current
revision, show the conflict and refreshed state to the human, and require a new explicit decision.
A new edit requires a new preview. A new mutation requires the newly observed expected revision and
a fresh operation ID.

## Prohibitions

- Never edit Intent, revision, HEAD, capability, or Snapshot files directly.
- Never invoke `zintent-core`, reproduce its transition logic, or bypass it with filesystem writes.
- Never run `zintent approve`, answer an approval challenge, or perform approval processing.
- Never invoke a Planner, Spec Kit, handoff adapter, or downstream implementation workflow.
- Never modify or delete an Approved Intent Snapshot.


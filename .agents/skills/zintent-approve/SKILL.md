---
name: zintent-approve
description: Check an exact eligible zintent review-complete revision and launch its human-only interactive TTY approval, returning the immutable Approved Intent Snapshot or structured blockers. Use only for approval, not review, artifact editing, or downstream planning.
---

# Approve an Intent

Connect an eligible exact Intent revision to the public interactive approval command. The skill may
inspect results and launch the flow, but approval authority remains with the human at the TTY and
all eligibility, challenge, revision, hashing, and persistence rules remain in the Zig core.

## Eligible input contract

Require an Intent selector plus the exact candidate revision ID. Accept it only when fresh canonical
results establish all of the following:

- the artifact and verified HEAD are valid under contract `1.0.0`;
- HEAD currently selects the exact candidate revision and its lifecycle is `review_complete`;
- at least one non-rejected item is included;
- every included item is accepted or human-edited with complete provenance;
- no comment is open, including comments on rejected items;
- zintent-check-equivalent mechanical validation has zero blocking findings.

Perform the mechanical preflight with `zintent validate <intent> --output json` and inspect canonical
state with `zintent show <intent> --output json`. The `zintent approve` command's core-governed
prepare step is authoritative and must recheck eligibility and the exact revision under lock.

Do not accept a Draft, an `in_review` or `approved` revision, an Approved Intent Snapshot, a detached
revision that is not current HEAD, or conversation-only content as approval input.

## Allowed CLI calls and human confirmation

Only these public calls are allowed:

```text
zintent validate <intent> --output json
zintent show <intent> --output json
zintent approve <intent> --revision <rev> --operation-id <id> [--output human|json]
```

Launch `zintent approve` attached directly to an interactive TTY. It prepares and displays the exact
revision ID, revision hash, approved-content hash, actor, blockers, and fresh one-use challenge, then
reads the response from that same TTY.

The skill and AI must never answer, guess, copy, transform, synthesize, prefill, relay, or submit the
challenge response. Do not request the response in chat. Do not use a flag, redirected stdin,
environment variable, automation, or non-TTY substitute. Leave response entry entirely to the human
using the attached TTY; cancellation or `tty_required` means approval did not occur.

## Output artifact contract

On success, return the exact `data.snapshot` and `data.snapshot_path` from the successful result
envelope. The Snapshot must conform to approved-snapshot contract `1.0.0`, be immutable and
content-addressed as `sha256-<approved_content_hash>`, and contain:

- approved content with the Intent ID, included items, provenance, and applicable sources;
- confirmed and approved revision IDs and hashes;
- approved-content hash and confirmation token ID;
- approving human actor, approval timestamp, and an eligible validation result with zero blockers.

Do not rewrite, normalize, copy as a replacement artifact, or claim a path not returned by the CLI.
If no successful envelope contains both snapshot and path, report that no Snapshot was issued.

## Structured ineligibility result

Return failures using stable fields rather than prose-only conclusions:

```json
{
  "eligible": false,
  "intent_id": "<uuid-or-null>",
  "candidate_revision_id": "<uuid>",
  "current_revision_id": "<uuid-or-null>",
  "findings": [
    {
      "code": "<stable-error-or-finding-code>",
      "severity": "blocking",
      "record_type": "<type-or-null>",
      "record_id": "<id-or-null>",
      "path": "<path-or-null>",
      "message": "<display-text>"
    }
  ],
  "error": {
    "code": "<stable-error-code>",
    "retryable": false
  }
}
```

Preserve the CLI result's findings and error code. Typical blockers include `invalid_artifact`,
`unreviewed_item`, `open_comment`, `approval_ineligible`, `stale_revision`, `tty_required`, and
`persistence_failure`. Display messages are explanatory only and are never branching contracts.

On `stale_revision`, stop the attempt. Do not reuse a challenge, token, or operation derived from
the old revision. Reload and validate HEAD, report the new revision and findings, and require the
human to explicitly start a new approval attempt for that exact revision.

## Prohibitions

- Never edit Intent, revision, HEAD, capability, or Snapshot files directly.
- Never invoke `zintent-core` or reproduce/bypass its checks.
- Never make review decisions, mutate an Intent to make it eligible, or call review mutation commands.
- Never answer or automate the TTY challenge.
- Never invoke a Planner, Spec Kit, handoff adapter, or downstream implementation workflow.
- Never modify or delete an Approved Intent Snapshot.

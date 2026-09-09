# Data Model: Intent Review Walking Skeleton

## Conventions

- IDs are immutable UUID v7 strings; timestamps are RFC 3339 UTC strings.
- Schemas carry a semantic `schema_version`; enum values use lower snake case.
- Published revisions and snapshots are immutable.
- Hash inputs satisfy I-JSON and use RFC 8785 canonicalization plus SHA-256.
- Every mutation supplies an `operation_id` and `expected_revision_id`.

## Store Layout

```text
<store>/<intent-id>/
├── intent.json
├── revisions/<revision-id>.json
├── snapshots/sha256-<approved-content-hash>.json
└── .lock/
```

The lock directory exists only while a mutation owns the per-Intent lock. Published revision and
snapshot files are never opened for modification. The HEAD manifest is the only replaced artifact.

## Head Manifest

| Field | Type | Rules |
|-------|------|-------|
| schema_version | string | Required supported major version |
| intent_id | UUID v7 | Must match directory and referenced revision |
| current_revision_id | UUID v7 | Must resolve under `revisions/` |
| current_revision_hash | SHA-256 hex | Must match the referenced revision |
| lifecycle_state | enum | `draft`, `in_review`, `review_complete`, or `approved` |
| approved_snapshot_ref | string or null | Required only in `approved` |

Readers verify the referenced artifact, hash, and state. A mismatch is an integrity failure.

## Revision Envelope

| Field | Type | Rules |
|-------|------|-------|
| schema_version | string | Required |
| revision_id | UUID v7 | Unique within the Intent |
| revision_hash | SHA-256 hex | Hash of the defined revision projection |
| hash_algorithm | string | `sha-256` |
| canonicalization | string | `jcs-rfc8785` |
| parent_revision_id | UUID v7 or null | Null only for initial Draft |
| operation_id | UUID v7 | Unique idempotency key |
| actor | Actor | Responsible actor |
| operation | Operation | Governed producing operation |
| created_at | timestamp | Required |
| revision_payload | Revision Payload | Complete resumable state |

Repeating an operation ID with the same command returns the original result. Reuse with different
content is an idempotency conflict. The hash excludes only the `revision_hash` field.

## Revision Payload

| Field | Type | Rules |
|-------|------|-------|
| intent_id | UUID v7 | Stable across revisions |
| lifecycle_state | enum | Must match the transition and HEAD |
| source_references | Source Reference[] | May be empty for manual Drafts |
| items | Intent Item[] | Unique IDs; at least one included item for completion |
| comments | Comment[] | Unique IDs; targets must resolve |
| approval_refs | Approval Reference[] | Append-only issued snapshot references |

## Intent Item

| Field | Type | Rules |
|-------|------|-------|
| item_id | UUID v7 | Stable and unique |
| kind | string enum | Extensible schema-defined category |
| statement | non-empty string | Required for included items |
| provenance | Provenance | Required and mechanically generated |
| resolution_status | enum | `determined` or `unresolved` |
| review_status | enum | `unreviewed`, `accepted`, `edited`, or `rejected` |
| included_in_approval | boolean | False iff rejected in Feature 001 |
| rationale | string or null | Required when rejected |
| source_reference_ids | UUID v7[] | References must resolve |
| supersedes / superseded_by | UUID v7[] | No self-reference or cycles |

A rejected item is terminal in its lineage, retained in history, and excluded from approved
content. Reintroduction creates a new item that supersedes the rejected item.

## Comment

| Field | Type | Rules |
|-------|------|-------|
| comment_id | UUID v7 | Stable and unique |
| target_item_id | UUID v7 | Must resolve |
| body | non-empty string | Immutable original concern |
| author | Actor | Required |
| status | enum | `open`, `resolved`, or `withdrawn` |
| created_revision_id | UUID v7 | Ancestor or current revision |
| closed_revision_id | UUID v7 or null | Required when closed |
| closure_reason | string or null | Required when closed |
| resolution_revision_id | UUID v7 or null | Optional addressing revision |

Item edit and rejection never close comments automatically. Only an explicit human resolve or
withdraw operation changes status.

## Actor and Provenance

Actor contains `actor_type` (`human`, `ai`, or `system`), non-empty `actor_id`, `identity_source`
(`os_user` or `explicit_fallback`), and `authenticated`. Human actors are unauthenticated in Feature
001. If the OS username is unavailable, mutations require an explicit fallback actor ID.

Provenance contains the Actor, operation ID, stable operation type, producing revision ID, and
optional source references. The domain service generates it; operation content cannot override it.

## Operations

Supported types are `start_review`, `accept_item`, `edit_item`, `reject_item`, `add_comment`,
`resolve_comment`, `withdraw_comment`, `complete_review`, `approve_intent`, and
`reopen_after_approval`. Operations store target IDs and audit content; identity, provenance,
revision IDs, hashes, and lifecycle effects are derived.

## Validation Finding

| Field | Type | Rules |
|-------|------|-------|
| code | string | Stable machine-readable code |
| severity | enum | `blocking`, `warning`, or `info` |
| record_type / record_id | string or null | Exact affected entity |
| path | string or null | Artifact field path when applicable |
| message | string | Display text, not a branching contract |

Feature 001 findings are mechanical: schema, identity, reference, transition, hash, stale revision,
eligibility, or persistence integrity. AI semantic findings are out of scope.

## Approval and Snapshot

Approval records an approval ID, approved revision ID and hash, approved-content hash, approving
Actor, timestamp, and validation result with zero blockers. Approved content contains the Intent
ID, schema version, included accepted/edited items, applicable sources, and downstream lineage.
Rejected items and review comments remain in history but are excluded from approved content.

The snapshot envelope contains `approved_content` and `approval`. Its ID and filename derive from
the approved-content hash. Approval metadata is outside that hash projection to avoid recursion.

## Lifecycle State Machine

| Current | Command | Preconditions | Next |
|---------|---------|---------------|------|
| draft | start review | Valid Draft and available human actor | in_review |
| in_review | review mutation | Expected revision matches and command validates | in_review |
| in_review | complete review | Included items reviewed; no open comments | review_complete |
| review_complete | approve | Eligibility rechecked; human confirms exact revision | approved |
| review_complete | review mutation | Valid mutation | in_review |
| approved | reopen/change | Valid mutation creates working revision | in_review |

No other transitions are permitted. A failed command publishes no reachable revision, and an
issued snapshot never transitions.

## Publication Sequence

1. Acquire the per-Intent lock.
2. Re-read and validate HEAD and its referenced revision.
3. Compare expected revision; return `stale_revision` on mismatch.
4. Detect an idempotent operation retry or conflicting operation-ID reuse.
5. Apply the typed command in memory and validate the complete next state.
6. Publish the immutable revision through same-directory temp write, sync, and atomic rename.
7. For approval, exclusive-create and verify the content-addressed snapshot.
8. Atomically replace and sync HEAD last.
9. Release the lock and return the structured result.

A crash may leave a complete orphan but cannot leave HEAD pointing to partial or missing content.
Discovery follows HEAD rather than directory enumeration.

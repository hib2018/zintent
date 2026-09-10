# Phase 0 Research: Intent Review Walking Skeleton

## Go and Zig Baseline

**Decision**: Mirror zconfig's separation: Zig 0.16.0 implements a reusable non-interactive core,
and Go 1.27.1 implements CLI/TUI orchestration. Pin Bubble Tea v2.0.8 with compatible Bubbles and
Lip Gloss v2 modules. The Zig core uses only the standard library in Feature 001.

**Rationale**: zconfig already establishes this local-tool pattern. Zig keeps schema validation,
state transitions, canonical hashing, approval, and artifact publication in a small deterministic
trust boundary. Go's process, cancellation, and terminal ecosystem fits orchestration and human
interaction. The language boundary prevents UI code from silently becoming a second domain layer.
Zig 0.16 is installed locally; Go must be installed before implementation validation.

**Alternatives considered**:

- Rust with Ratatui: strong single-binary domain modeling, but diverges from the requested zconfig
  architecture and cannot reuse its cross-process design conventions.
- One Go executable: simpler build, but weakens isolation of the deterministic safety core.
- One Zig executable with a hand-built TUI: mixes terminal complexity into the trust boundary.

**Sources**: [Zig 0.16 release notes](https://ziglang.org/download/0.16.0/release-notes.html),
[Go releases](https://go.dev/doc/devel/release),
[Bubble Tea releases](https://github.com/charmbracelet/bubbletea/releases),
[Go process execution](https://pkg.go.dev/os/exec).

## Artifact Store and Atomic Publication

**Decision**: Store each Intent in a directory containing a small mutable `intent.json` HEAD
manifest, complete immutable revision envelopes under `revisions/`, and content-addressed approved
snapshots under `snapshots/`. For each mutation: acquire the per-Intent lock, re-read HEAD, compare
the mandatory expected revision, validate and build the full new revision, publish it through a
same-directory temporary file, sync and rename it, then replace HEAD atomically last. Sync the
parent directory where supported.

**Rationale**: Files remain inspectable process artifacts while immutable revisions bound the
corruption surface. Publishing HEAD last means readers see either the prior valid state or the new
valid state. The expected-revision check prevents a decision based on stale content even when
separate TUI and CLI processes overlap.

**Alternatives considered**:

- One mutable JSON document: rewrites history and increases corruption impact.
- Append-only JSON Lines: adds crash-tail recovery and compaction to the MVP.
- SQLite: offers transactions but makes the primary artifact opaque and adds export semantics that
  Feature 001 does not need.
- Last-write-wins: violates the no-overwrite invariant.

**Source**: [POSIX rename semantics](https://pubs.opengroup.org/onlinepubs/9799919799/functions/rename.html).

## Canonical Hashing and Snapshot Immutability

**Decision**: Canonicalize hash projections with RFC 8785 JSON Canonicalization Scheme, encode as
UTF-8, and hash with SHA-256 lowercase hexadecimal. Define separate projections:

- `revision_hash = SHA-256(JCS(revision_payload))`; the payload excludes its envelope hash field.
- `approved_content_hash = SHA-256(JCS(approved_content))`; approval metadata is outside the
  `approved_content` object.

Publish snapshots by exclusive-create at
`snapshots/sha256-<approved_content_hash>.json`. An existing equivalent snapshot makes an approval
retry idempotent; different content at that address is an integrity failure. No update or delete
operation exists for snapshots, and reads verify the filename, envelope, and hashes.

**Rationale**: Hashing formatted JSON would make whitespace or map ordering change identity. Named
projections avoid recursive or ambiguous definitions. Content addressing plus exclusive creation
enforces immutability at the tool boundary; filesystem read-only flags are defense-in-depth only.

**Alternatives considered**: raw-file hashing and custom sorted-key JSON were rejected because they
are formatting-sensitive or risk cross-language disagreement.

**Source**: [RFC 8785 JSON Canonicalization Scheme](https://www.rfc-editor.org/rfc/rfc8785.html).

## Core Process Protocol and CLI Results

**Decision**: Expose `zintent-core` as a one-shot Zig process and `zintent` as the Go CLI/TUI. For
each operation, Go starts the core directly without a shell, writes one UTF-8 JSON request to stdin,
closes stdin, accepts exactly one JSON response from stdout, and treats extra bytes as failure.
Envelope and payload schemas version independently. Mutations require `expected_revision_id` and
idempotency `operation_id`. Requests, responses, and captured stderr are capped at 16 MiB; the core
deadline is 10 seconds. Human diagnostics use stderr.

The public Go CLI preserves a single versioned JSON result envelope on stdout when `--output json`
is selected. Consumers branch on stable codes, never message text. Text bodies come from files or
stdin, not shell-expanded command strings.

Exit status is coarse: 0 success, 2 usage, 3 artifact/schema validation, 4 stale conflict,
5 eligibility/transition refusal, 6 persistence I/O, and 70 unexpected internal failure. The JSON
envelope remains authoritative.

**Rationale**: A one-shot protocol is easy to bound, cancel, test, and recover. It gives the TUI,
CLI, and later skills the same Zig-enforced semantics without a C ABI or network service. Strict
decoding catches misspellings and contract drift before mutation.

**Alternatives considered**: a shared library ABI couples toolchains; JSON Lines and JSON-RPC add
streaming, notification, and batching semantics Feature 001 does not need; shell invocation is an
injection and quoting risk; direct JSON editing bypasses the core.

**Sources**: [Go os/exec](https://pkg.go.dev/os/exec),
[RFC 8259 JSON](https://www.rfc-editor.org/rfc/rfc8259),
[JSON Schema 2020-12](https://json-schema.org/draft/2020-12).

## Human Confirmation Capabilities

**Decision**: Item edits and final approval use separate two-step, core-issued, short-lived,
one-use capabilities. An edit capability binds the Intent, expected revision, item, actor, and
before/after statement hashes. An approval capability binds the eligible review-complete revision,
revision hash, approved-content hash, actor, and a fresh challenge. Approval preparation and
consumption are available only when the Go frontend is attached to a TTY; the human types the
challenge response during that same interaction. The core stores capability records in a transient
registry outside revisions and snapshots, consumes them atomically under the Intent lock, and may
delete expired records without changing HEAD. They are invalidated by expiry, use, actor mismatch,
payload mismatch, or stale revision.

**Rationale**: A `--confirm` flag or direct mutation request proves only that a caller copied data;
an Agent skill could do that without human involvement. Binding a fresh TTY response to the exact
reviewed content preserves human authority. The same capability pattern gives CLI and TUI edits an
enforceable preview-before-apply invariant.

**Alternatives considered**: revision repetition on the command line, stdin-only confirmation, and
frontend-generated tokens were rejected because automation could synthesize them or because they
move an invariant outside the Zig trust boundary.

## Minimal Hunk-Style TUI

**Decision**: Implement the Go TUI with Bubble Tea v2 as a pure presentation reducer around the
bounded core runner.
It holds selection and modal input only; confirmed operations persist immediately. The screen has
an Intent/revision header, item list, hunk detail pane, and findings/key-help footer. Narrow screens
stack list and detail. Support arrows or `j/k`, `a` accept, `e` edit, `c` comment, `x` reject,
`r` resolve/withdraw, `d` diff, `f` complete review, `p` approve, `?` help, and `q` exit.

Process key Press events for mutations and optional Repeat events only for navigation. After a
successful command, reload canonical state and retain selection by stable item ID. On a stale
revision, reload and show the conflict without replaying the operation. Restore terminal state on
normal exit, error, and panic.

**Rationale**: A pure UI state machine is testable and cannot become a second domain implementation.
Immediate governed persistence makes exit safe without a separate save concept.

**Alternatives considered**: TUI-only mutations and a full component framework were rejected for
contract drift and unnecessary abstraction.

**Sources**: [Bubble Tea repository](https://github.com/charmbracelet/bubbletea),
[Bubble Tea releases](https://github.com/charmbracelet/bubbletea/releases).

## Supported Platform and Scale

**Decision**: Feature 001 supports local interactive terminals on macOS and Linux, one human
reviewer, one Intent open in a TUI, up to 1,000 active and historical items/comments, and artifacts
up to 10 MiB per revision. Windows, network filesystems, remote collaboration, and authenticated
multi-user workflows are deferred.

**Rationale**: Atomic replacement and terminal behavior can be tested concretely on POSIX targets.
Explicit MVP limits prevent untested durability or rendering claims while remaining well above the
walking-skeleton fixtures.

**Alternatives considered**: claiming Windows support now would require separate replace, flush,
terminal, and path contract testing; leaving scale unbounded would make performance gates
unverifiable.

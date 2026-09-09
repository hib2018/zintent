# Implementation Plan: Intent Review Walking Skeleton

**Branch**: `001-intent-review-skeleton` | **Date**: 2026-09-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-intent-review-skeleton/spec.md`

## Summary

Build the first end-to-end zintent slice with a deterministic Zig core and a Go CLI/TUI frontend.
The Zig core exclusively owns validation, state transitions, provenance, hashing, and persistence.
The Go frontend presents the Hunk-style review and invokes one core operation at a time through a
versioned JSON subprocess protocol. Project-local `zintent-review` and `zintent-approve` skills
orchestrate the Go tool without directly editing artifacts.

## Technical Context

**Language/Version**: Zig 0.16.0 for the deterministic core; Go 1.27.1 for CLI/TUI orchestration

**Primary Dependencies**: Zig standard library; Bubble Tea v2.0.8 with compatible Bubbles and Lip
Gloss v2 modules; Go standard library for subprocess, JSON, filesystem, and CLI handling

**Storage**: Local JSON artifact directory with mutable HEAD manifest, immutable full revisions,
and content-addressed approved snapshots; RFC 8785 canonicalization and SHA-256 identities

**Testing**: `zig build test`, `go test ./...`, cross-language protocol fixtures, transition tables,
JSON Schema/golden hash contracts, fault-injected persistence tests, and Bubble Tea reducer tests

**Target Platform**: Local macOS and Linux interactive terminals; Windows and network filesystems
are out of scope for Feature 001

**Project Type**: Dual-executable local process tool: reusable Zig core plus Go CLI/TUI, with two
project-local Agent skills

**Performance Goals**: Validate and display a 1,000-item, 10 MiB Intent within 1 second on reference
development hardware; persist a normal single-item review action within 250 ms excluding human
input; maintain responsive navigation without visible input backlog

**Constraints**: Offline-capable; no service or database; one request/response per core process;
16 MiB maximum protocol request, response, and captured stderr; 10-second core deadline; atomic
visibility of mutations; immutable revisions/snapshots; no shell evaluation or direct artifact
mutation; single active TUI per Intent with stale-revision rejection across processes

**Scale/Scope**: One local human reviewer; one active TUI per Intent; up to 1,000 active/historical
items and comments and 10 MiB per revision; fixture or manually authored Draft input only

## Constitution Check

*GATE: Passed before research and re-checked after Phase 1 design.*

| Principle | Design Evidence | Status |
|-----------|-----------------|--------|
| Skill-centered process | `zintent-review` and `zintent-approve` are use-case skills; atomic actions remain executable commands | PASS |
| Skills orchestrate; tools enforce | Zig core exclusively owns validation, transitions, provenance, hashing, and persistence | PASS |
| Artifact process contracts | Versioned schemas, immutable revisions, result envelopes, and resumable HEAD are specified in `contracts/` | PASS |
| Human authority | Approval requires OS-derived human actor and fresh confirmation of the exact revision | PASS |
| Interface semantic parity | Go CLI and TUI send the same versioned operations to the Zig core and consume one result model | PASS |
| Adapter isolation | No planner integration is included; later adapters consume only approved snapshots | PASS |
| Thin orchestration | Skills call commands and report outcomes; they do not own state or duplicate transitions | PASS |
| Verification and safety | Contract, transition, hash, persistence-fault, CLI, and TUI tests are required | PASS |

No constitutional exceptions or unjustified abstractions are required. Post-design re-check also
passes: schemas define artifact boundaries, the store owns persistence, and TUI/skills cannot
bypass domain commands.

## Project Structure

### Documentation (this feature)

```text
specs/001-intent-review-skeleton/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── protocol.md
│   ├── message.schema.json
│   ├── cli.md
│   ├── intent.schema.json
│   ├── result.schema.json
│   └── snapshot.schema.json
└── tasks.md
```

### Source Code (repository root)

```text
build.zig
build.zig.zon
core/
├── src/
│   ├── main.zig
│   ├── protocol.zig
│   ├── model.zig
│   ├── command.zig
│   ├── transition.zig
│   ├── validation.zig
│   ├── hashing.zig
│   └── store.zig
└── tests/
    ├── contract.zig
    ├── transition.zig
    ├── hashing.zig
    └── persistence_failure.zig

tui/
├── go.mod
├── go.sum
├── cmd/zintent/main.go
└── internal/
    ├── protocol/
    ├── runner/
    ├── command/
    ├── ui/
    └── output/

.agents/skills/
├── zintent-review/
│   └── SKILL.md
└── zintent-approve/
    └── SKILL.md

tests/
├── contract/
│   ├── fixtures/
│   └── compatibility_test.go
├── integration/
│   ├── review_journey_test.go
│   ├── approval_safety_test.go
│   ├── stale_revision_test.go
│   └── crash_recovery_test.go
└── fixtures/
    ├── valid-draft.json
    └── invalid-drafts/
```

**Structure Decision**: Match zconfig's proven two-project boundary. `core/` is a non-interactive
Zig executable that owns all trusted domain and storage operations. `tui/` is a Go executable that
owns CLI parsing, terminal presentation, process lifecycle, and human interaction. They share no
language ABI and communicate only through versioned contracts and single-request/single-response
JSON over stdin/stdout. Skills invoke the Go executable and consume its structured output.

## Complexity Tracking

No Constitution violation requires justification. Two languages are intentional: Zig isolates the
portable deterministic trust boundary, while Go and Bubble Tea isolate integration-heavy terminal
interaction. Cross-language duplication is prohibited and verified through contract fixtures.

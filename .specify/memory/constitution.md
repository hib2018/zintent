<!--
Sync Impact Report
- Version change: 1.0.0 -> 1.1.0
- Modified principles:
  - I. Skills Are the Primary Unit -> I. Skill-Centered Process Foundation
  - II. Explicit Skill Contracts -> III. Artifacts Are the Process Contract
  - III. Thin, Deterministic Orchestration -> VII. Thin and Optional Orchestration
  - IV. Progressive Context and Dependency Discipline -> incorporated into I, III, and VII
  - V. Verifiable Outcomes and Safe Execution -> incorporated into II, IV, and constraints
- Added principles:
  - II. Skills Orchestrate; Tools Enforce
  - IV. Human Authority Is Preserved
  - V. One Domain Semantics Across Interfaces
  - VI. Adapters Are Isolated
- Added sections: none
- Removed sections: none
- Follow-up TODOs: none
-->
# zintent Constitution

## Core Principles

### I. Skill-Centered Process Foundation
zintent MUST be designed as a process foundation composed of small, user-recognizable skills,
deterministic tools, and shared artifacts. It MUST NOT be modeled as a TUI-centered application.
The review TUI is one tool for human review and MUST remain replaceable by conversation or CLI
interaction. A skill MUST represent a use case such as interpreting, reviewing, revising,
checking, approving, or handing off an Intent; atomic operations such as saving, commenting, or
issuing an ID MUST remain tool commands rather than separate skills.

Rationale: user-goal boundaries keep skills composable, while treating the TUI as one frontend
prevents the process model from becoming coupled to a particular interface.

### II. Skills Orchestrate; Tools Enforce
Skills MUST define the process: when to invoke tools, where AI interpretation is appropriate, how
to interact with a human, how to explain unresolved matters, and which subsequent skill or adapter
is applicable. Deterministic tools or the domain library MUST exclusively enforce persistence,
schema validation, identifiers, revisions, state transitions, diffs, provenance, approval
eligibility, content hashes, immutable snapshots, and export. A skill or Agent MUST NOT edit
persistent Intent data directly or claim that an invariant holds without invoking the responsible
tool.

At minimum, tools MUST enforce these invariants:

- an Intent with unresolved comments cannot be approved;
- an AI revision cannot overwrite a human edit without human acceptance;
- any post-approval content change invalidates the approval;
- provenance is derived mechanically from the recorded operation; and
- an approved snapshot cannot be modified after issuance.

Rationale: natural-language instructions guide judgment but are not a trustworthy enforcement
boundary for durable state.

### III. Artifacts Are the Process Contract
Skills MUST connect through explicit, versioned artifacts or machine-readable standard I/O, not
through hidden conversational state or mandatory direct skill invocation. Every skill MUST declare
its accepted input artifact, output artifact, valid starting states, permitted transitions,
observable success criteria, side effects, and failure or escalation conditions. Stable Intent
Item IDs MUST survive revisions and downstream handoff. The persisted artifact MUST contain enough
state to resume safely after interruption from any process boundary.

The canonical progression is Intent Draft, Reviewed Intent, Approved Intent Snapshot, then Handoff
Artifact. Comments and revision proposals MUST be independently addressable records rather than
untraceable text embedded in an item.

Rationale: artifact contracts allow humans, Codex skills, other Agent frameworks, and scripted
automation to participate in the same resumable workflow.

### IV. Human Authority Is Preserved
AI output MUST remain a draft or proposal until a human performs the required acceptance or
approval transition. Revision generation MUST follow comment, proposed revision, human review,
then accept, edit, or reject; generating a proposal MUST NOT apply it. Approval MUST name the exact
revision and content hash, record the approving actor and validation result, and emit an immutable
snapshot. In interactive use, final approval MUST require an explicit human action and MUST NOT be
completed autonomously by a skill.

Rationale: zintent exists to let people intervene in interpretation before it becomes an approved
input to downstream planning.

### V. One Domain Semantics Across Interfaces
TUI, conversational, and CLI operations MUST invoke the same domain transitions and produce the
same provenance, revisions, validation behavior, and approval effects. Interface-specific code
MUST NOT implement alternate Intent semantics. The CLI and other tools MUST provide stable,
machine-readable results in addition to human-readable output, including success state, validation
findings, changed identifiers, and actionable failure details.

Rationale: multiple interaction paths remain trustworthy only when they share one deterministic
operation contract.

### VI. Adapters Are Isolated
Planner-, Agent-, model-, transport-, and interface-specific behavior MUST be isolated behind an
adapter skill or adapter module. The generic handoff path MUST accept only an approved snapshot and
produce a planner-independent handoff artifact. Downstream artifacts MUST retain their originating
Intent Item IDs. An adapter MUST detect interpretations introduced outside the approved snapshot
and MUST return them to Intent review rather than silently treating them as approved.

For the initial Spec Kit integration, zintent-speckit MUST remain outside the generic core. It MUST
map requirements to Intent Item IDs and verify candidate specifications for added interpretation;
it MUST NOT grant approval to additions made by Spec Kit.

Rationale: adapter isolation protects the Intent lifecycle from the assumptions and evolution of
any particular downstream planner.

### VII. Thin and Optional Orchestration
A workflow skill MAY inspect artifact state and recommend the next applicable skill, but it MUST
remain optional. It MUST NOT mutate Intent data, perform independent Intent interpretation, or
make individual skills unusable on their own. Routing MUST be reproducible from declared artifact
state: no Intent routes to interpretation, a Draft to review, open comments to revision, a reviewed
Intent to checking, an eligible Intent to approval, and an approved snapshot to handoff. Skills
MUST load only the instructions and resources needed for their current responsibility.

Rationale: orchestration is useful guidance, not a second application layer or a hidden owner of
domain state.

## Skill Architecture Constraints

- The repository MUST maintain a discoverable skill catalog containing each skill's name,
  use case, trigger boundary, artifact contracts, exclusions, and owner.
- Skill-local instructions and resources MUST remain colocated. Cross-skill duplication MUST be
  removed or promoted into a deliberately owned shared primitive.
- The domain model MUST distinguish Intent Documents, Intent Items, Comments, Revision Proposals,
  and Approvals as addressable concepts with stable identifiers and explicit relationships.
- Mechanical validation findings and AI-derived semantic findings MUST be labeled separately.
- The domain library MUST own model, transitions, validation, revisions, approval, and
  serialization. CLI, TUI, and adapter layers MUST depend on that domain behavior and MUST NOT
  reimplement it.
- Physical package separation is optional until justified; logical module and dependency
  boundaries are mandatory from the first implementation.
- New platform infrastructure MUST be justified by at least two concrete skill consumers or by a
  non-negotiable security, reliability, or operability requirement.
- Secrets and sensitive data MUST use the narrowest available access scope and MUST NOT be placed
  in prompts, logs, fixtures, or generated artifacts unless explicitly required and protected.

## Development Workflow and Quality Gates

1. Specify the user intent, exclusions, and observable acceptance criteria before implementation.
2. Identify the owning skill, deterministic tool operations, and artifact contracts; document any
   new or changed cross-skill handoff.
3. Add or update schema, transition, invariant, contract, routing, and behavior tests before
   considering the capability complete.
4. Implement the smallest skill-local change that satisfies the contract. Any new shared
   abstraction MUST include its concrete consumers and ownership rationale in review materials.
5. Verify affected skills independently, verify TUI/conversation/CLI semantic parity where
   applicable, then verify composed workflows at artifact boundaries.
6. Review every change for Constitution compliance, including context loading, dependency
   direction, side-effect scope, and evidence of completion.
7. Record intentional exceptions with an owner, rationale, affected scope, and removal or review
   date. An undocumented exception is non-compliant.

## Governance

This Constitution is the highest-level engineering policy for zintent. Specifications, plans,
tasks, reviews, and implementation guidance MUST comply with it. When another project document
conflicts with this Constitution, this Constitution prevails until it is formally amended.

Amendments MUST be proposed as a documented change that states the motivation, affected
principles, compatibility impact, and migration requirements. Adoption requires explicit project
maintainer approval. Materially affected specifications and skills MUST be migrated in the same
change or tracked with an approved owner and deadline.

Constitution versions follow semantic versioning: MAJOR for removal or incompatible redefinition
of a principle or governance rule; MINOR for a new principle or materially expanded obligation;
PATCH for clarifications that do not change required behavior. The ratification date remains the
date of initial adoption, and the last-amended date changes whenever normative text changes.

Every feature specification and implementation review MUST include a Constitution check. Reviewers
MUST reject unexplained violations. The project MUST perform a compliance review when skill
contracts, orchestration boundaries, or shared infrastructure change, and MUST either remediate
findings or record a governed exception under the workflow above.

**Version**: 1.1.0 | **Ratified**: 2026-09-09 | **Last Amended**: 2026-09-09

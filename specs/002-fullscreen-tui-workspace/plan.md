# Implementation Plan: Full-screen Intent Workspace

**Branch**: `002-fullscreen-tui-workspace` | **Date**: 2026-09-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-fullscreen-tui-workspace/spec.md`

## Summary

Feature 001のreview、approval、audit、recoveryを、一つの全画面Intent workspaceへ統合する。Go側は共有sessionを所有するroot modelと独立したscreen/modal modelを持ち、Zig coreのtyped operationだけを呼ぶ。永続状態はZigだけが変更し、mutation成功後は必ずcanonical artifactを再読込する。Intent一覧、two-step Draft import、verified revision/snapshot inspection、token-bound recovery cleanupをcoreへ追加する。

## Technical Context

**Language/Version**: Zig 0.16.0（core）、Go 1.27.1（CLI/TUI）

**Primary Dependencies**: Zig標準ライブラリ、Bubble Tea v2.0.8、Bubbles v2、Lip Gloss v2、Go標準ライブラリ

**Storage**: ローカルworkspace配下のIntent directory。HEAD、immutable revisions、content-addressed snapshots、短命capability registryを継続利用

**Testing**: `zig build test`、`go test ./...`、reducer table、view golden、fake-command contract、PTY integration、real-core journey、1,000-item performance

**Target Platform**: macOS/Linuxの対話型POSIX terminal。Windows、network filesystem、非対話approvalは対象外

**Project Type**: Zig core processとGo CLI/full-screen TUIからなるdual-executable local tool

**Performance Goals**: 1,000 itemでnavigation/renderの95%を100ms以内、mutation後2秒以内にcanonical result表示、通常の画面遷移を1秒以内

**Constraints**: offline、16 MiB protocol limit、10秒core timeout、one request/response per process、no direct artifact mutation、one active mutation per Intent、stable-ID selection、fresh one-use approval challenge

**Scale/Scope**: 単一workspace、単一local reviewer、最大1,000 items/comments、10 MiB revision、約10 screen

## Constitution Check

*GATE: Phase 0前にPASS。Phase 1設計後もPASS。*

| Principle | Design Evidence | Status |
|---|---|---|
| Skill-centered process | TUIはreview/approve skillからも利用できる代替可能なfrontendであり、skillやCLIを置換しない | PASS |
| Skills orchestrate; tools enforce | validation、transition、ID、hash、import、cleanup、approvalをZig coreが所有する | PASS |
| Artifact process contracts | UI sessionは一時状態だけを持ち、正式な再開状態はversioned artifactから復元する | PASS |
| Human authority | exact revision/hashとfresh challengeを表示し、foreground TTYの人間入力だけを承認へ渡す | PASS |
| Interface parity | TUI、CLI、skillは同じoperation/result envelopeを使う | PASS |
| Adapter isolation | Planner、Spec Kit、AI interpretationは対象外 | PASS |
| Thin orchestration | root modelはroutingとcore invocationだけを担いdomain判断を実装しない | PASS |
| Verification | reducer、contract、PTY、failure、performanceを独立検証する | PASS |

設計後も例外はない。workspace discovery、Draft import、recovery cleanupをGoのfilesystem処理にせずcore operationへ置き、TUIへ信頼境界が移らないようにする。

## Project Structure

### Documentation (this feature)

```text
specs/002-fullscreen-tui-workspace/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── workspace-protocol.md
│   ├── workspace-command.schema.json
│   └── tui-navigation.md
└── tasks.md
```

### Source Code (repository root)

```text
core/src/
├── model.zig
├── command.zig
├── workspace.zig
├── import.zig
├── audit.zig
├── recovery.zig
├── store.zig
└── main.zig

tui/cmd/zintent/main.go
tui/internal/
├── protocol/
├── runner/
├── command/
│   ├── workspace.go
│   ├── review.go
│   ├── approval.go
│   ├── audit.go
│   └── recovery.go
├── ui/
│   ├── workspace.go
│   ├── navigation.go
│   ├── dashboard.go
│   ├── review.go
│   ├── comments.go
│   ├── completion.go
│   ├── approval.go
│   ├── history.go
│   ├── validation.go
│   ├── snapshot.go
│   ├── recovery.go
│   └── modals.go
└── output/

core/tests/
tui/internal/ui/*_test.go
tests/contract/
tests/integration/
```

**Structure Decision**: Feature 001のZig/Go境界と既存executablesを維持する。Zig内ではworkspace/import/audit/recoveryを論理moduleへ分割し、Go側はroot workspace modelとscreen-local modelへ再編する。新しい永続化層は追加しない。

## Phase 0: Research Outcomes

- root workspace model、explicit screen enum、single active modalを採用する。
- 人間入力操作を`editing → preview/loading → confirming → submitting → canonical reload`として管理する。
- workspace discovery、Draft import、revision/snapshot inspection、recoveryをadditive protocol operationとして追加する。
- importとcleanupは観測と実行を分離し、one-use capabilityでexact source/candidate setへbindする。
- approvalのTTY情報はfrontend attestationであり暗号学的証明ではないことを明記し、coreのcapability/eligibility再検証と組み合わせる。

詳細は[research.md](research.md)を参照する。

## Phase 1: Design Outcomes

- [data-model.md](data-model.md)にworkspace session、screen、modal、preview、audit selection、recovery observationを定義した。
- [workspace-protocol.md](contracts/workspace-protocol.md)に追加core operationsとfailure semanticsを定義した。
- [workspace-command.schema.json](contracts/workspace-command.schema.json)に追加payloadのmachine-readable contractを定義した。
- [tui-navigation.md](contracts/tui-navigation.md)に画面遷移、key ownership、canonical reload規則を定義した。
- [quickstart.md](quickstart.md)にDraft importからsnapshot、history、recoveryまでの検証journeyを定義した。

## Complexity Tracking

Constitution違反はなく、例外記録は不要。

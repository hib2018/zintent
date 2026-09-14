# Validation results

## 2026-09-15 Feature 001 baseline

- `zig build test`: PASS
- `zig build`: PASS
- `cd tui && go test ./...`: PASS
- `cd tests && go test ./...`: environment cache access failed; rerun with an isolated `GOCACHE` is required

The integration suite was rerun with `GOCACHE=/tmp/zintent-go-cache` and passed.
No Feature 001 behavior failure was observed before Feature 002 implementation.

## Foundation checkpoint

- `ZIG_GLOBAL_CACHE_DIR=/tmp/zintent-zig-cache zig build test`: PASS
- `cd tui && GOCACHE=/tmp/zintent-go-cache go test ./...`: PASS
- `cd tests && GOCACHE=/tmp/zintent-go-cache go test ./...`: PASS

All Setup and Foundational tasks T001-T016 are complete. Workspace protocol fixtures cover every additive operation; root reducer, correlated executor, TTY refusal, cursor/alternate-screen ownership, and seven root-frame golden scenarios are covered by tests.

## User Story 1 checkpoint

- Modal state-machine, review/comment reducer, exact command dispatch, approval and snapshot tests: PASS
- PTY alternate-screen/cursor restoration journey: PASS
- Existing review CLI, approval safety, core and skill-facing contract regression suites: PASS
- Draft review mutations always reload canonical state; approval requires a foreground TTY and a freshly typed exact challenge.

## User Story 2 checkpoint

- Verified revision/snapshot inspection and orphan separation: PASS
- Recovery observation, one-use token, lock-time HEAD/candidate revalidation, and selected temporary cleanup: PASS
- History/diff/provenance, validation, snapshot, and recovery reducer/golden tests: PASS
- Real-core audit/recovery protocol and PTY terminal-restoration journeys: PASS
- Protected HEAD, revision, snapshot, capability, lock, and orphan artifacts are never cleanup targets.

## User Story 3 checkpoint

- `list_intents` はworkspace直下の実directoryだけを検証し、再帰探索とsymlink追跡を行わず、stable Intent ID順で返す: PASS
- valid、corrupt、nested、symlink、missing root、collision、source change、source preservation、atomic publication、idempotent retry: PASS
- `inspect_draft` capabilityはsource hash、actor、destination、期限に束縛され、`import_draft`で一度だけ消費される: PASS
- TUIの検索、stable selection、blocking finding、失敗時modal reset、canonical open、最初の未レビューitemへのresume: PASS
- macOS PTY workspace起動・終了とcursor復元、およびGo runtimeにIntent artifactの直接write APIがないこと: PASS

## 2026-09-15 Feature 002 final validation

Quickstartの各シナリオは、対応する自動化シナリオとして実行した。

1. Build and test: `zig build`, `zig build test`, `go test -count=1 ./...`（tui/tests）: PASS
2. Workspace preparation: interactive-TTY refusalとmacOS PTY alternate-screen起動: PASS
3. Draft import: preview、hash/destination表示モデル、source preservation、collision/rollback: PASS
4. Review and approval: Feature 001 PTY journey、fresh exact challenge、immutable snapshot: PASS
5. Audit: reachable/orphan分離、revision diff、provenance、snapshot linkage: PASS
6. Stale/timeout/crash: stale reload、executor timeout、core crash、terminal restoration: PASS
7. Recovery: observation revalidation、選択temporaryのみcleanup、protected artifact保持: PASS
8. Layout/performance: 1,000件を24可視行に制限し、10回のrender/reducer検査が各100ms未満: PASS

逸脱: Approval challengeは安全境界を維持するためforeground `/dev/tty` が必須であり、完全な疑似入力による自動承認は行っていない。LinuxのPTY制御列はBSD `script` と異なるためmacOSで実行し、LinuxはCI matrixで非PTY suiteを検証する。

Constitution gate:

- skills orchestrate / tools enforce、artifact直接変更禁止、人間承認、immutable snapshot、machine-readable result: PASS
- JSON syntax validation: contracts、schema、contract fixturesの49ファイル: PASS
- cross-interface semantic parity: Feature 001 CLI regression、workspace protocol、Go direct-write禁止検査: PASS
- portability: `.github/workflows/ci.yml` の`macos-latest` / `ubuntu-latest` matrixでZig core、Go TUI、integration suiteを実行する構成: PASS
- first-time-user completion: workspace作成、Draft import、一覧再読込、canonical resumeまでの独立journey: PASS

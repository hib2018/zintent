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

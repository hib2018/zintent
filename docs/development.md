# 開発ガイド

## 現在の状態

Feature 001のspec、plan、data model、JSON Schema、quickstart、tasksが揃っています。Go/Zigの
source treeはこれから実装します。本書のcommandは現時点ではcontractです。

## 技術構成

| 領域 | 技術 | 責務 |
|---|---|---|
| Core | Zig 0.16.0 | domain、validation、transition、hash、persistence |
| CLI/TUI | Go 1.27.1 | command、process、TTY、Bubble Tea UI |
| Contract | JSON Schema 2020-12 | versioned protocolとArtifact |
| Identity | RFC 8785 + SHA-256 | revisionとapproved content |
| Storage | local filesystem | HEAD、immutable revision、Snapshot |

## 実装順

1. Go/Zig projectとCI
2. strict schema、protocol fixture、hash vector
3. Zig domain modelとatomic store
4. User Story 1のreview CLI/TUI
5. User Story 2のeligibilityとapproval
6. User Story 3のresume、diff、recovery
7. performance、platform、usability、Constitution gate

依存順は[Tasks](../specs/001-intent-review-skeleton/tasks.md)を正とします。testを先に作り、意図した
理由で失敗することを確認してから実装します。

## 実装後の標準検証

```sh
zig build
zig build test

cd tui
go test ./...
go vet ./...
```

さらにGo/Zig共通fixture、hash golden vector、transition table、fault-injected persistence test、
Bubble Tea reducer testを実行します。

## 変更の進め方

Domain ruleを変える場合は、Constitution、Feature spec、data modelとSchema、protocol/CLI contract、
plan/tasks、docsの順に同期します。UIだけにruleを追加してはいけません。

Schema v1はunknown fieldを拒否します。required field追加、field削除、意味変更はmajor version変更が
必要です。

skillはuser-recognizable use case単位にし、入力・出力Artifact、開始state、許可tool、side effect、
成功条件、escalation、対象外を宣言してcatalogへ登録します。atomic commandをskill化しません。

## 完了条件

- specのacceptance scenarioとSuccess Criteriaを満たす
- contract、transition、invariant、failure pathをtestする
- CLI/TUIが同じdomain semanticsを使う
- skillがArtifactを直接変更しない
- crash後もHEADから安全に再開できる
- Constitution checkを再実行する

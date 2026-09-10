# zintentドキュメント

このディレクトリは、zintentの目的と設計を日本語で理解するための入口です。厳密な受入条件や
machine-readable contractは`specs/`を正とし、ここでは背景と関係性を説明します。

## 読む順番

1. [プロジェクト概要](overview.md) — なぜ必要か、何を対象とするか
2. [アーキテクチャ](architecture.md) — skill、Go、Zig、Artifactの境界
3. [Intentモデル](intent-model.md) — Item、Comment、Revision、Approval
4. [レビューと承認](workflow.md) — DraftからSnapshotまで
5. [安全性と不変条件](safety.md) — 人間の権限、競合、永続化
6. [開発ガイド](development.md) — 実装順、テスト、変更規則

## 規範文書

| 文書 | 役割 |
|---|---|
| [Constitution](../.specify/memory/constitution.md) | 最上位原則 |
| [Feature spec](../specs/001-intent-review-skeleton/spec.md) | 要求、受入条件、範囲 |
| [Plan](../specs/001-intent-review-skeleton/plan.md) | 技術構成と実装方針 |
| [Data model](../specs/001-intent-review-skeleton/data-model.md) | Entity、状態、永続化 |
| [Contracts](../specs/001-intent-review-skeleton/contracts/protocol.md) | CLI、protocol、Schema |
| [Tasks](../specs/001-intent-review-skeleton/tasks.md) | 依存順の実装作業 |

`docs/`は説明資料です。振る舞いを変える場合は対応するspecやcontractを先に更新し、この説明を
同期してください。

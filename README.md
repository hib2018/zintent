# zintent

zintentは、人間の要求をPlannerやAgentへ渡す前に、AIによる解釈を項目単位でレビューし、
承認済みIntentとして固定するためのプロセス基盤です。

中心にあるのは単一アプリケーションではなく、ユースケース単位のskill、状態と不変条件を
管理する決定論的tool、工程間を接続するversioned Artifactです。Hunk風TUIは、人間がレビューへ
介入するための主要なフロントエンドの一つです。

> [!IMPORTANT]
> 現在はFeature 001「Intent Review Walking Skeleton」の仕様・設計・実装タスクを作成した
> 段階です。Go/Zigの実装と実行可能なCLI/TUIはまだ完成していません。

## 目的

- AIの解釈をstable IDを持つIntent Itemへ分解する
- 人間が各項目をaccept、edit、comment、rejectできるようにする
- 変更、comment、provenance、revisionを追跡する
- 未解決事項がある間は承認を拒否する
- 人間が確認した内容だけをimmutableなApproved Intent Snapshotとして発行する
- Planner固有の処理をadapter skillへ隔離する

## 全体像

```text
Human request
    ↓
zintent-interpret skill
    ↓
Intent Draft
    ↓
zintent-review / zintent-revise
    ↓
zintent-check
    ↓
zintent-approve
    ↓
Approved Intent Snapshot
    ↓
zintent-handoff / adapter skill
    ↓
Planner / Agent
```

| 層 | 責務 |
|---|---|
| Skills | 作業手順、AIによる判断、人間との対話、次工程の案内 |
| Tools | 保存、検証、状態遷移、差分、provenance、承認、hash、export |
| Artifacts | skill間の入力・出力・状態を接続する正式な契約 |

原則は「Skills orchestrate; tools enforce」です。skillやAgentはIntentの永続データを直接編集
しません。

## Feature 001

最初のFeatureでは、既存のIntent Draftを人間がレビューし、承認済みSnapshotを得る最小の
縦切りを構築します。

含まれる予定の機能：

- Draftの読込と厳密な検証
- 項目単位のaccept、edit、comment、reject
- commentのresolve、withdraw
- immutable revisionとHEADによる中断・再開
- review completionとapproval eligibility
- TTY上の明示的な人間確認
- immutableなApproved Intent Snapshot
- Go製CLI/TUIとZig製core
- 最小の`zintent-review`、`zintent-approve` skill

対象外は、自然言語からのDraft生成、AI改訂案、意味的矛盾検出、Spec Kit連携、複数人承認、
認証・認可、remote collaborationです。

## アーキテクチャ

```text
Go: CLI / Hunk-style TUI / skill integration
                 │ versioned JSON protocol
                 ▼
Zig: model / validation / transitions / revisions
     provenance / approval / hashing / persistence
                 │
                 ▼
Local Artifact Store
```

Goは引数、表示、TTY、人間とのinteractionを担当します。Zig coreはID、検証、状態遷移、
provenance、hash、revision、snapshot、永続化を所有します。

## ライフサイクル

```text
draft ──start review──▶ in_review ──complete review──▶ review_complete
                           ▲                                  │
                           │                                  │ approve
                           │                                  ▼
                           └────── review mutation ─────── approved

review_complete ──review mutation──▶ in_review
```

承認は、確認された`review_complete` revisionを親に持つ新しい`approved` revisionを作ります。
承認後の変更は新しい`in_review` revisionになりますが、過去のSnapshotは変更されません。

## ドキュメント

- [ドキュメント案内](docs/README.md)
- [プロジェクト概要](docs/overview.md)
- [アーキテクチャ](docs/architecture.md)
- [Intentモデル](docs/intent-model.md)
- [レビューと承認](docs/workflow.md)
- [安全性と不変条件](docs/safety.md)
- [開発ガイド](docs/development.md)
- [Constitution](.specify/memory/constitution.md)
- [Feature 001仕様](specs/001-intent-review-skeleton/spec.md)
- [実装計画](specs/001-intent-review-skeleton/plan.md)
- [実装タスク](specs/001-intent-review-skeleton/tasks.md)

仕様と実装が競合する場合は、ConstitutionとFeature specを起点にcontracts、plan、tasksの順で
整合させます。

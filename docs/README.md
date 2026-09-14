# zintent ドキュメントガイド

このページは、zintentのドキュメントを読む順番と各文書の役割をまとめた入口です。

## まず読む

1. [プロジェクト概要](../README.md)
2. [機能仕様](../specs/001-intent-review-skeleton/spec.md)
3. [クイックスタート](../specs/001-intent-review-skeleton/quickstart.md)

## 設計を理解する

- [実装計画](../specs/001-intent-review-skeleton/plan.md)：Zigコア、Go CLI/TUI、プロトコル境界
- [データモデル](../specs/001-intent-review-skeleton/data-model.md)：Intent、Item、Comment、Revision、Approval、Snapshot
- [調査結果](../specs/001-intent-review-skeleton/research.md)：採用した技術・永続化方針・制約
- [タスク一覧](../specs/001-intent-review-skeleton/tasks.md)：実装済み範囲と検証タスク

## 機械可読契約

`specs/001-intent-review-skeleton/contracts/` に、次の契約を置いています。

- `protocol.md`：標準入出力、サイズ上限、タイムアウト、エラー方針
- `cli.md`：CLIコマンドと終了コード
- `message.schema.json`：リクエスト・レスポンス envelope
- `command.schema.json`：coreへ渡すコマンド
- `intent.schema.json` / `head.schema.json`：IntentとHEAD
- `result.schema.json`：成功結果とfinding
- `edit-preview.schema.json`：編集preview capability
- `approval-confirmation.schema.json`：承認challenge
- `snapshot.schema.json`：immutable Approved Intent Snapshot
- `provenance.schema.json` / `source-reference.schema.json`：出所と入力参照

## Skill契約

- [skill catalog](../.agents/skills/zintent-catalog.md)
- [zintent-review](../.agents/skills/zintent-review/SKILL.md)
- [zintent-approve](../.agents/skills/zintent-approve/SKILL.md)

Skillは工程を案内し、永続化・検証・状態遷移・承認判定はZig coreが強制します。SkillやTUIがIntent JSONを直接変更することはありません。

## 検証記録

- [validation-results.md](../specs/001-intent-review-skeleton/validation-results.md)：build、unit/integration、性能、fuzz、quickstart
- [usability-results.md](../specs/001-intent-review-skeleton/usability-results.md)：代表シナリオによる操作性確認

なお、usability結果の10件は実ユーザー調査ではなく、初回利用を想定した自動化シナリオです。実機のLinux検証と定性的な利用者インタビューは別途必要です。

## 状態と成果物の流れ

```text
Draft
  ↓ review
in_review
  ↓ complete_review
review_complete
  ↓ human TTY approval
approved + immutable snapshot
  ↓ normal change
in_review（snapshotは不変）
```

各状態はrevisionとして保存され、HEADは現在revisionとhashを指します。承認済みsnapshotはcontent-addressedで、後から上書きできません。

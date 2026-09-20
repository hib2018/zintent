# zintent ドキュメントガイド

このページは、zintentのドキュメントを読む順番と各文書の役割をまとめた入口です。

## まず読む

1. [プロジェクト概要](../README.md)
2. [機能仕様](../specs/001-intent-review-skeleton/spec.md)
3. [クイックスタート](../specs/001-intent-review-skeleton/quickstart.md)
4. [全画面ワークスペースのクイックスタート](../specs/002-fullscreen-tui-workspace/quickstart.md)

## 設計を理解する

- [実装計画](../specs/001-intent-review-skeleton/plan.md)：Zigコア、Go CLI/TUI、プロトコル境界
- [データモデル](../specs/001-intent-review-skeleton/data-model.md)：Intent、Item、Comment、Revision、Approval、Snapshot
- [調査結果](../specs/001-intent-review-skeleton/research.md)：採用した技術・永続化方針・制約
- [タスク一覧](../specs/001-intent-review-skeleton/tasks.md)：実装済み範囲と検証タスク
- [Feature 002仕様](../specs/002-fullscreen-tui-workspace/spec.md)：複数Intent、全画面操作、監査、復旧
- [TUIナビゲーション契約](../specs/002-fullscreen-tui-workspace/contracts/tui-navigation.md)：画面遷移、キー、modal所有権
- [Workspaceプロトコル](../specs/002-fullscreen-tui-workspace/contracts/workspace-protocol.md)：一覧、取込、監査、復旧操作

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
- [Feature 002 validation-results.md](../specs/002-fullscreen-tui-workspace/validation-results.md)：全画面workspace、PTY、性能、互換性

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

## 全画面TUIの運用

```sh
./zig-out/bin/zintent workspace /path/to/workspace
```

ワークスペース直下の各Intent directoryだけが候補です。一覧で`n`を押すと、workspaceと同じ階層の`draft/`以下にあるJSONが選択候補として表示されます。`j/k`または矢印でDraftを選び、検証結果、source hash、提案保存先を確認後にatomic importします。symlink、JSON以外のファイル、`draft/`外のパスは候補になりません。`Enter`でIntentを開くと常にZig coreからcanonical revisionを再取得し、前回選択がなければ最初の未レビューitemへ復帰します。

Dashboardから`r` Review、`c` Comments、`f` Completion、`p` Approval、`h` History、`v` Validation、`s` Snapshot、`R` Recoveryへ移動します。監査表示は到達可能revisionとorphanを分離し、復旧は明示選択した変更のない一時ファイルだけを対象にします。core timeout/crashやstale revision時は操作を自動再実行せず、canonical reloadを要求します。

# レビューと承認

## DraftからReviewへ

Feature 001は既存Draftから開始します。schema、ID重複、参照、hash、lifecycleに問題があれば、
revisionを作らず拒否します。予定commandは次のとおりです。

```sh
zintent validate <intent> --output json
zintent show <intent> --output json
zintent review <intent>
```

> 現時点では実装予定のcontractであり、まだ利用可能なCLIではありません。

`start_review`が`draft`から`in_review`へのrevisionを作ります。TUIはItem list、before/after、
findings、current revision、blocker数を表示します。

## Item review

- accept: 内容をそのまま承認候補にする
- edit: 人間の表現へ直す
- comment: 懸念を独立recordとして残す
- reject: 理由付きで承認対象から除外する。後からacceptまたはeditすると取り消せる

各mutationは新revisionを作り、UIは成功後にcanonical stateを再読込します。

### Editの二段階操作

```text
proposed statement
  → preview_edit
  → before/after + short-lived one-use token
  → human confirmation
  → edit_item
  → new revision
```

tokenはIntent、expected revision、Item、actor、提案内容へbindingされます。不足、期限切れ、再利用、
内容不一致、staleなtokenでは変更できません。

## Commentを閉じる

open commentは承認をblockします。内容を編集しても自動では閉じません。人間が理由付きで
`resolve`または`withdraw`します。必要なら解決に関係するrevisionを参照できます。

## Review completion

open commentがなくArtifactとprovenanceが有効な状態でReviewを完了すると、結果に応じて遷移します。

- 承認対象Itemが一つ以上あり、すべてacceptedまたはhuman-edited: `review_complete`
- 全Itemがreject済み: `rejected`

`rejected` Intentは承認できません。同じItemをacceptまたはeditすると`in_review`へ戻ります。
不足があれば、対象record IDを持つmachine-readable findingを返します。

## Approval

```text
review_complete revision
  → prepare_approval
  → revision/hash/content hash/blockers + fresh challenge
  → human types response in the same TTY
  → approve_intent
  → approved child revision + immutable Snapshot
```

CLI flag、redirectされたstdin、skillによる応答生成で人間確認を代替できません。非TTY承認は
`tty_required`です。`zintent-approve` skillは画面を起動できますがchallengeへ回答できません。

## 中断、再開、承認後の変更

操作は確認ごとに永続化するためquit時のsave promptは不要です。再開時はHEADとrevisionを検証し、
status、Comment、provenance、blockerを復元します。stale mutationは拒否し、自動再実行しません。

発行済みSnapshotを編集するcommandはありません。approved Intentへの通常mutationは新しい
`in_review` revisionを作り、以前のSnapshotを残します。専用reopen commandはありません。

正式なsurfaceは[CLI契約](../specs/001-intent-review-skeleton/contracts/cli.md)を参照してください。

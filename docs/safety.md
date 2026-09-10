# 安全性と不変条件

## 信頼境界

skillやAIの指示文を永続状態の信頼境界にしません。domain mutationはZig coreが実行し、Goは
CLI/TUIとTTY上の人間interactionを担当します。

## 必須の不変条件

- open comment、未レビューItem、不正Artifactがあれば承認できない
- `expected_revision_id`不一致を拒否し、人間の編集を古い判断で上書きしない
- 最終承認は同じTTY上のfresh challengeへ人間が応答する
- 承認後の変更は新しい`in_review` revisionにする
- provenanceはcaller入力ではなくoperationから機械生成する
- Approved Snapshotを更新・削除するcommandを設けない

## 競合とidempotency

| 条件 | 結果 |
|---|---|
| expected revision不一致 | `stale_revision` |
| 同じoperation ID、同じcommand | 元の結果 |
| 同じoperation ID、異なるcommand | `operation_id_conflict` |
| stale responseを受けたTUI | reloadし、人間へ再判断を求める |

mutationは自動retryしません。read-only operationだけがtransport failure後にretry可能です。

## Confirmation capability

edit previewとapproval preparationは短命なone-use capabilityを発行します。対象内容、actor、revisionへ
bindingし、core-owned transient registryでatomicに消費します。revisionやSnapshotには格納せず、
Approval auditには消費token IDだけを残します。

## Crash safety

1. Intent単位のlockを取得
2. HEADとrevisionを再検証
3. expected revisionとidempotencyを確認
4. 次状態をmemory上で構築・検証
5. temporary fileへwrite、sync、atomic rename
6. Snapshotをexclusive create
7. HEADを最後にatomic replace

障害時は以前の正常revisionか、新しい完全なrevisionだけが到達可能です。

## HashとSnapshot

I-JSONをRFC 8785でcanonicalizeし、SHA-256を計算します。Snapshotは
`sha256-<approved-content-hash>.json`へexclusive createします。同じaddressに異なる内容があれば
integrity failureです。

## Feature 001の限界

- OSユーザー名は認証・認可ではない
- local single-user環境が対象
- malicious local administratorからの保護は保証しない
- Windows、network filesystem、remote collaborationは対象外
- capabilityは人間性の暗号学的証明ではなく、公開toolの非対話経路を閉じる仕組み

詳細は[Constitution](../.specify/memory/constitution.md)を参照してください。

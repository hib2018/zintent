# Intentモデル

## Entity

### Intent Document

レビュー中の作業Artifactです。Intent ID、schema version、lifecycle、source、Item、Comment、
Approval参照を持ちます。現在状態はHEADが指すimmutable revisionで決まります。

### Intent Item

人間が個別に判断できる最小の解釈単位です。stable ID、kind、statement、provenance、resolution、
review status、rationale、source referenceを持ちます。

| Review status | 意味 |
|---|---|
| `unreviewed` | 未判断 |
| `accepted` | 提示内容を受入済み |
| `edited` | 人間が編集済み |
| `rejected` | 理由付きで承認対象外 |

rejectされたItemも履歴から削除しません。再導入時は元Itemをsupersedeする新Itemを作ります。

### Comment

stable comment IDを持つ独立Entityで、一つのItemをtargetにします。状態は`open`、`resolved`、
`withdrawn`です。Itemをeditまたはrejectしても自動では閉じず、人間が理由付きで閉じるまで
承認をblockします。

### Revision

一つのgoverned mutationで生まれるimmutableな完全状態です。parent revision、actor、operation、
timestamp、content hashを持ち、各revisionからレビューを再開できます。

### ApprovalとSnapshot

Approvalは、人間が確認した`review_complete` revisionと、新しく作られた`approved` revision、
approved content hash、actor、validation resultを結びます。Snapshotは承認内容を固定した
immutable Artifactです。reject Itemとreview commentは監査履歴に残りますがapproved contentには
含まれません。

## Actor、Origin、Provenance

- Operation Actor: 操作主体。Feature 001ではローカルの人間
- Content Origin: statementの由来。`source`、`human`、`ai`、`system`
- Provenance: origin、操作、actor、revision、sourceを結ぶ機械生成記録

OSユーザー名はローカル監査用で、認証済みidentityではありません。

## Lifecycle

| 現在 | 操作 | 次 |
|---|---|---|
| `draft` | start review | `in_review` |
| `in_review` | review mutation | `in_review` |
| `in_review` | complete review | `review_complete` |
| `review_complete` | approve | 新しい`approved` child |
| `review_complete` / `approved` | review mutation | 新しい`in_review` child |

詳細は[Data model](../specs/001-intent-review-skeleton/data-model.md)とJSON Schemaを参照してください。

# Data Model: Full-screen Intent Workspace

Feature 001のIntent、Item、Comment、Revision、Approval、Snapshotを変更せず再利用する。この文書はworkspace操作と一時的なTUI状態だけを追加定義する。

## Workspace

| Field | Type | Rules |
|---|---|---|
| workspace_path | path | 明示されたlocal root。canonicalize後も許可範囲内であること |
| entries | Workspace Entry[] | stable Intent ID順 |
| findings | Finding[] | 個別entryの破損をworkspace全体の失敗と分離 |

## Workspace Entry

| Field | Type | Rules |
|---|---|---|
| intent_id | stable ID | directory artifactから検証済みで取得 |
| display_name | string | 表示用。identity判定には使用しない |
| intent_path | relative path | workspace外へescapeしない |
| current_revision_id | stable ID | verified HEADが指すrevision |
| lifecycle_state | enum | Feature 001と同じ4状態 |
| blocker_count | integer | coreのmechanical findingsから算出 |
| approval_state | enum | none、eligible、approved、invalidated |
| snapshot_id | ID or null | verified snapshotだけを参照 |

## Workspace Session

永続的なIntent artifactではない。process終了時に破棄可能なpresentation stateである。

| Field | Type | Rules |
|---|---|---|
| active_screen | Screen | 現在表示中のscreen |
| navigation_stack | Screen[] | `Esc`で戻るための履歴 |
| selected_intent_id | ID or null | indexではなくstable ID |
| selected_record_id | ID or null | item、comment、revision、finding等 |
| canonical_intent | verified view or null | 最終reload成功時の状態 |
| actor | Actor | local unauthenticated actor semanticsを維持 |
| terminal_size | width/height | layout専用。domainへ保存しない |
| active_request | Request State or null | mutationはIntentごとに最大1件 |
| modal | Modal State or null | 初期リリースは同時に1件 |
| status | Notice | info、success、warning、error |

## Screen

`intent_list`、`dashboard`、`review`、`comments`、`completion`、`approval`、`history`、`diff`、`validation`、`snapshot`、`recovery`。

遷移は[contracts/tui-navigation.md](contracts/tui-navigation.md)に従う。screen遷移自体はdomain mutationではない。

## Modal State

| Field | Type | Rules |
|---|---|---|
| kind | enum | edit、comment、reject、closure、completion、approval、import、cleanup |
| phase | enum | editing、preview_loading、confirming、submitting、reload_loading、error |
| target_id | ID or null | stable record ID |
| starting_revision_id | ID or null | dispatch時のexpected revision |
| text | transient string | 未確定入力。artifactへ直接保存しない |
| capability_token | transient token or null | modal退出、stale、expiryで破棄 |
| expires_at | timestamp or null | preview/challengeの期限 |
| request_id | ID or null | late response除外用 |

### Modal transitions

```text
closed
  → editing
  → preview_loading（previewが必要な場合）
  → confirming
  → submitting
  → reload_loading
  → closed

任意の確定前phase → closed（cancel、永続変更なし）
任意のrequest phase → error → reload_loadingまたはclosed
```

## Active Request

| Field | Type | Rules |
|---|---|---|
| request_id | ID | response correlation |
| operation | operation enum | core contractのoperation |
| intent_id | ID or null | mutation lockの論理対象 |
| target_id | ID or null | stable affected record |
| starting_revision_id | ID or null | stale判定とUI説明に使用 |
| kind | enum | read、mutation、canonical_reload |
| state | enum | pending、cancel_requested、indeterminate |

mutation responseだけではcanonical Intentを更新しない。success後のreload responseが同じrequest chainに属するときだけ共有状態を置換する。

## Draft Import Preview

| Field | Type | Rules |
|---|---|---|
| source_path | path | read-only source |
| source_hash | SHA-256 | exact source bytesへbind |
| proposed_intent_id | ID | core発行 |
| proposed_destination | relative path | workspace内、collisionなし |
| findings | Finding[] | blockingがあればimport不可 |
| import_token | transient token | source/workspace/destination/actorへbind、one-use |
| expires_at | timestamp | 期限後は再inspect |

## Revision Summary and Audit Selection

Revision Summaryはrevision ID/hash、parent、operation ID/type、actor、created time、lifecycle、reachabilityを持つ。Audit Selectionはcomparison base/target revision ID、selected change ID、selected provenance IDを保持するだけで、revision内容を変更しない。

## Recovery Observation

| Field | Type | Rules |
|---|---|---|
| intent_id | ID | 対象Intent |
| observed_revision_id | ID | 観測時HEAD |
| observed_head_hash | SHA-256 | cleanup直前に再確認 |
| head_valid | boolean | current revisionとの整合性 |
| reachable_revision_ids | ID[] | chain traversal結果 |
| orphan_candidates | Candidate[] | 表示専用。このFeatureでは削除しない |
| temporary_candidates | Candidate[] | core publication tempのみ |
| recovery_token | transient token | exact candidate setへbind |
| expires_at | timestamp | 期限後は再scan |

Candidateはrelative name、kind、size、content hashを持つ。cleanup対象は、利用者が明示選択し、再走査で同一と確認されたtemporary candidateだけである。

## State and authority boundaries

- Workspace Session、selection、filter、未送信textはpresentation stateでありIntentへ保存しない。
- Intent lifecycleとrevision transitionはFeature 001のcore rulesだけが変更する。
- approval challenge responseはprefillせず、foreground TUIの人間入力からだけ作る。
- mutation cancel、timeout、late responseでは成功を推測せずcanonical reloadで確定する。
- CLI、skill、TUIは同じoperationとartifactを共有する。

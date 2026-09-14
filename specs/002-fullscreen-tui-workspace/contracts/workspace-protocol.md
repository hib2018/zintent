# Workspace Protocol Additions

Feature 001のprotocol version `1.0`、`zintent.command/1`、`zintent.result/1` envelopeを維持し、operationをadditiveに追加する。すべてのpathはcoreがcanonicalizeし、workspace/Intent境界外へのescapeを拒否する。

## Operations

| Operation | Purpose | Mutation |
|---|---|---|
| list_intents | workspace直下のverified Intent summaryを列挙 | No |
| inspect_draft | Draft取込候補を検証しexact import previewを発行 | No |
| import_draft | one-use previewを消費してIntentをatomic import | Yes |
| list_revisions | HEAD-reachable chainとorphan summaryを返す | No |
| inspect_revision | stable revision IDをhash検証して返す | No |
| inspect_snapshot | stable snapshot IDとapproval linkageを検証して返す | No |
| recovery_status | HEAD、chain、orphan、temporary candidateを観測 | No |
| cleanup_temporary_files | confirmed candidateだけを再検証後に削除 | Yes |

既存のreview、comment、completion、approval、show、validate、diff操作は変更しない。

## list_intents

Input: `workspace_path`。

Output: stable Intent ID順のentry。各entryは`intent_id`、`display_name`、relative `intent_path`、`current_revision_id`、`lifecycle_state`、`blocker_count`、`approval_state`、`snapshot_id`を持つ。個別の破損はfindingとして返し、workspace root自体が不正な場合だけ全体を失敗させる。

## inspect_draft / import_draft

`inspect_draft`は`workspace_path`、`source_path`、actorを受け取り、strict validation、source hash、proposed Intent ID/destination、collision findings、expiry、one-use `import_token`を返す。

`import_draft`は上記に加え、`operation_id`、token、確認したdestinationを要求する。lock内でsource hashとdestinationを再検証し、complete Intent directoryをatomic publishする。source artifactは変更しない。

## list/inspect audit operations

- `list_revisions(intent_path)`はnewest-to-oldestのreachable chainを返し、orphansを別配列にする。
- `inspect_revision(intent_path, revision_id)`は任意pathを受け付けず、IDから解決してhashを検証する。
- `inspect_snapshot(intent_path, snapshot_id)`はfilename/content identity、approved-content hash、approval/revision linkageを検証する。
- orphan inspectionを許す場合は`include_orphan:true`を明示し、canonical historyとは表示上区別する。

## recovery_status / cleanup_temporary_files

`recovery_status`はobserved revision/HEAD hash、reachable IDs、orphan candidates、temporary candidates、expiry、candidate-setへbindした`recovery_token`を返す。

`cleanup_temporary_files`は次を必須とする。

- human actor
- operation ID
- expected current revisionとHEAD hash
- fresh recovery token
- 明示選択したcandidate IDs

coreはlock内で再走査し、内容・名前・kindが一致するcore publication temporary fileだけを削除する。HEAD、revision JSON、snapshot、capability、lock、orphan revisionは削除しない。

## Approval from full-screen TUI

TUIはforeground interactive terminalを確認してから`prepare_approval`を実行する。actor、exact revision/hash、approved content hash、challengeを表示し、KeyPress由来のresponseを`approve_intent`へ渡す。flag、environment、prefill、generic program messageはresponse sourceとして使用しない。

`interactive_tty`はfrontend attestationであり暗号学的proofではない。coreはone-use token、actor、revision、hash、expiry、eligibilityをlock内で再検証する。

## Error codes

既存codeに加え、少なくとも次を安定したcodeとして返す。

- `invalid_workspace`
- `path_escape`
- `draft_invalid`
- `destination_conflict`
- `import_preview_expired`
- `revision_not_found`
- `snapshot_not_found`
- `integrity_failure`
- `recovery_observation_stale`
- `cleanup_target_changed`
- `terminal_required`

mutationは自動retryしない。timeoutやtransport failureで結果が不明な場合、frontendはcanonical reloadを要求する。

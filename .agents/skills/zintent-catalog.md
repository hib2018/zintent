# zintent skill catalog

このカタログは、zintent の versioned artifact を受け渡す project-local skill を登録する。
domain rule、状態遷移、provenance、hash、revision、永続化は Zig core が所有し、skill は公開
`zintent` CLI/TUI だけを編成する。

| Skill | 入力 | 出力 | 用途 |
|---|---|---|---|
| [`zintent-review`](zintent-review/SKILL.md) | store 内の Intent Draft または再開可能な current Intent revision | Zig core が永続化した current Intent revision と blocker summary | Hunk-style TUI で人間による item review を開始または再開する |
| [`zintent-approve`](zintent-approve/SKILL.md) | approval eligibility を満たす exact `review_complete` Intent revision | immutable、content-addressed Approved Intent Snapshot、または structured ineligibility findings | eligibility を確認し、人間だけが応答できる TTY approval へ接続する |

共通境界:

- Intent revision、HEAD、Snapshot を skill が直接作成、変更、置換、削除しない。
- `zintent-core` を直接呼ばず、公開 `zintent` CLI/TUI を使用する。
- display message を分岐条件にせず、JSON result envelope の stable code と field を使用する。
- Planner、Spec Kit、handoff adapter は Feature 001 の対象外であり、ここから呼び出さない。


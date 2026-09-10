# アーキテクチャ

## 境界

```text
Skills
  │ process / dialogue / AI judgment
  ▼
Go frontend
  │ CLI / Bubble Tea TUI / TTY / presentation
  │ versioned JSON, one request / one response
  ▼
Zig core
  │ validation / transition / revision / provenance / hash
  ▼
Artifact store
  └ HEAD / immutable revisions / snapshots
```

## Skill層

skillは、いつどのtoolを使うか、人間へ何を説明するか、AIの判断をどこで使うかを定義します。
Artifactを直接書き換えず、正式な入出力はArtifactとstructured command resultです。

## Go frontend

Go側は利用者とcoreの境界を担当します。

- public CLIのargument parsing
- OSユーザー名によるactor discovery
- human-readable / JSON output
- Hunk風TUIとTTY上の人間確認
- Zig processの起動、timeout、stream上限
- 成功後のcanonical state再読込

独自の状態遷移規則は持たず、CLIとTUIを同じcore commandへ変換します。

## Zig core

Zig coreは信頼境界です。

- Schema、ID、参照、hashの検証
- stable IDとrevisionの生成
- expected revisionによる競合検出
- lifecycle transitionとprovenance
- edit/approval capabilityの発行と消費
- RFC 8785 canonicalizationとSHA-256
- revision、HEAD、Snapshotの安全なpublication

coreは一回の起動で一つのJSON requestを読み、一つのresponseを返します。streaming、batch、
network transport、shell evaluationはFeature 001に含めません。

## Store

```text
<store>/<intent-id>/
├── intent.json                       # mutable HEAD
├── revisions/<revision-id>.json     # immutable
├── snapshots/sha256-<hash>.json     # immutable
├── capabilities/                    # short-lived, core-owned
└── .lock/                           # mutation中のみ
```

revisionは完全な状態を持つため、会話やTUIのmemoryなしに再開できます。mutationではlockを取り、
HEADを再検証し、新revisionをpublishしてからHEADを最後にatomic replaceします。

## 依存方向

```text
skills → Go public commands → Zig protocol → domain/store
adapters → Approved Intent Snapshot
```

TUIからstoreへの直接書込は禁止です。Planner固有処理はcoreではなくadapterへ隔離します。詳細は
[Protocol](../specs/001-intent-review-skeleton/contracts/protocol.md)を参照してください。

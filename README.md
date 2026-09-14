# zintent

zintent は、Intentを人間がレビューし、決定論的なコアで検証・承認するためのプロセス基盤です。Zigコアが状態遷移、ハッシュ、永続化を所有し、Go CLI/TUIは同じプロトコルを呼び出します。

## ビルド

Zig 0.16.0 と Go 1.27.1 を用意し、次を実行します。

```sh
zig build
zig build test
(cd tui && go test ./... && go build -o ../zig-out/bin/zintent ./cmd/zintent)
```

## CLI

```sh
zintent validate PATH --output json
zintent show PATH --output json
zintent diff PATH --output json
zintent item accept PATH ITEM_ID --expected-revision REV --operation-id OP --actor-id NAME
zintent complete-review PATH --expected-revision REV --operation-id OP --actor-id NAME
zintent review PATH
zintent approve PATH --revision REV --operation-id OP --actor-id NAME
zintent workspace WORKSPACE_DIR
```

`approve` は `/dev/tty` からchallengeを読み、非TTYやstdinによる代替確認を拒否します。

## 全画面ワークスペース

`zintent workspace WORKSPACE_DIR` は、指定ディレクトリの直下にある複数のIntentを一覧・検索し、レビュー、承認、監査、復旧までを一つのalternate-screen TUIで扱います。サブディレクトリの再帰探索やsymlink追跡は行いません。破損したIntentはfindingとして表示され、開くことはできません。

Intent一覧では `/` で検索、`n` で既存Draft JSONの取込、`Enter` で選択Intentをcanonical stateから開きます。取込はsource hash、coreが提案したID・保存先、findingを確認してから確定します。元Draftは変更されず、失敗時に部分的なIntentは公開されません。

画面は `Intent list → Dashboard → Review / Comments / Completion / Approval / History / Validation / Snapshot / Recovery` の構成です。共通キーは `Esc` 戻る、`q` 終了、`r/c/f/p/h/v/s/R` が各画面への移動です。入力中は`Esc`で取消し、確定操作は`Enter`で行います。Approvalはfresh challengeの完全一致を必要とし、Recoveryは観測後に変化していない一時ファイルだけを削除します。

## Storeレイアウトと復旧

ディレクトリ型Intentは `HEAD.json`、`revisions/<revision-id>.json`、`snapshots/sha256-*.json`、`capabilities/` を持ちます。HEADは常に最後に置き換えられ、revisionとsnapshotはimmutableです。再起動時はHEADのhashを検証して現在revisionを復元し、orphan revisionと一時ファイルを監査できます。

## TUIキー

`j/k` または矢印で移動、`a` accept、`e` edit、`c` comment、`x` reject、`f` review完了、`p` approval案内、`q`終了です。操作はすべてZigコアへ送られ、成功後にcanonical artifactを再読み込みします。

## Actor

ローカル利用ではOSユーザー名をactor IDとして使います。取得できない場合のみ `--actor-id` の明示値を `explicit_fallback` として記録します。これは認証を意味せず、操作のprovenanceを残すための識別です。

## 開発

仕様とタスクは `specs/001-intent-review-skeleton/` と `specs/002-fullscreen-tui-workspace/`、skillsは `.agents/skills/` にあります。CLI/TUIがIntent JSONを直接変更することは禁止され、永続的な変更はcore protocol経由で行います。

## ドキュメント

文書の入口は [`docs/README.md`](docs/README.md) です。仕様、データモデル、JSON契約、skill契約、quickstart、検証結果をまとめています。

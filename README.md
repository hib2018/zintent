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
```

`approve` は `/dev/tty` からchallengeを読み、非TTYやstdinによる代替確認を拒否します。

## Storeレイアウトと復旧

ディレクトリ型Intentは `HEAD.json`、`revisions/<revision-id>.json`、`snapshots/sha256-*.json`、`capabilities/` を持ちます。HEADは常に最後に置き換えられ、revisionとsnapshotはimmutableです。再起動時はHEADのhashを検証して現在revisionを復元し、orphan revisionと一時ファイルを監査できます。

## TUIキー

`j/k` または矢印で移動、`a` accept、`e` edit、`c` comment、`x` reject、`f` review完了、`p` approval案内、`q`終了です。操作はすべてZigコアへ送られ、成功後にcanonical artifactを再読み込みします。

## Actor

ローカル利用ではOSユーザー名をactor IDとして使います。取得できない場合のみ `--actor-id` の明示値を `explicit_fallback` として記録します。これは認証を意味せず、操作のprovenanceを残すための識別です。

## 開発

仕様とタスクは `specs/001-intent-review-skeleton/`、skillsは `.agents/skills/` にあります。CLI/TUIがIntent JSONを直接変更することは禁止され、永続的な変更はcore protocol経由で行います。

## ドキュメント

文書の入口は [`docs/README.md`](docs/README.md) です。仕様、データモデル、JSON契約、skill契約、quickstart、検証結果をまとめています。

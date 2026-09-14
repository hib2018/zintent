# Full-screen TUI Navigation Contract

## Screen hierarchy

```text
intent_list
  → dashboard
      → review → comments
      → completion
      → approval → snapshot
      → history → diff
      → validation
      → recovery
```

## Global behavior

- `Esc`: top modalを閉じる、detail focusを離れる、または1 screen戻る。mutationしない。
- `Ctrl+C`: application shutdownを要求する。active requestはcancelし、成功を推測しない。
- `q`: text入力以外のscreenで終了する。入力中の文字`q`は通常入力として扱う。
- `?`: contextual help。
- `Tab` / `Shift+Tab`: 表示中pane間のfocus移動。
- `j/k`とarrow: text inputがfocusされていないときだけlist navigation。
- confirmation dispatchはKeyPressだけを受け付け、KeyReleaseとrepeat confirmを無視する。

## Shared frame

各screenはheader、body、status/error、contextual helpを表示する。headerはIntent ID、lifecycle、revision、actor、blocker countを含む。狭い端末でもcurrent target、error、Esc/quit方法を残す。

## Stable selection

selectionはentity IDで保持する。reload/filter/resize後に同じIDが存在すれば維持する。存在しない場合は最初のunresolved record、次に最も近いremaining recordを選ぶ。

## Mutation flow

```text
input
  → optional core preview
  → exact confirmation
  → one core mutation
  → canonical reload
  → result display
```

request中はmutation shortcutと二重confirmを無効化する。stale responseではpending preview/tokenを破棄し、reload後に人間へ再判断を求める。late request IDのresponseは表示状態へ適用しない。

## Approval flow

approval modalはconfirmed revision ID/hash、approved content hash、included/excluded count、findings、challengeを常時表示する。responseは空欄から人間が入力する。cancel、screen移動、revision change、expiry、terminal lossでtokenと入力を消去する。

## Terminal lifecycle

root applicationだけがalternate screen、raw mode、cursor、shutdownを所有する。screen modelは独自のterminal escapeを出力しない。正常終了、error、timeout、Ctrl+Cのすべてでterminal restoration後にprocessを返す。

# Quickstart Validation: Full-screen Intent Workspace

このガイドはFeature 002の実装後に、Draft取込からApproved Snapshot、監査、復旧までを一つのTUI sessionで検証する手順である。

## 1. Build and test

```sh
zig build
zig build test
cd tui
go test ./...
go build -o ../zig-out/bin/zintent ./cmd/zintent
cd ..
```

Expected: core、CLI/TUI、contract、reducer、PTY testが成功する。

## 2. Prepare a workspace

空のlocal workspace directoryとFeature 001互換のDraft fixtureを用意する。

```sh
mkdir -p /tmp/zintent-workspace
./zig-out/bin/zintent workspace /tmp/zintent-workspace
```

Expected: alternate-screenのIntent listが表示され、header、status、help、終了操作が確認できる。

## 3. Import a Draft without leaving the TUI

1. Intent listで`n`を押す。
2. Draft source pathを入力する。
3. validation findings、source hash、proposed destinationを確認する。
4. importを明示確認する。

Expected: 元Draftは変更されず、partial directoryを残さず、新しいIntent entryが一覧へ追加される。不正Draftまたはcollisionでは何も登録されない。

## 4. Complete review and approval

1. Intentを開きdashboardからReviewへ移動する。
2. `a`でacceptする。
3. `e`でstatementを編集し、before/after previewを確認して適用する。
4. `c`でcommentを追加し、Comments画面で理由付きresolveまたはwithdrawを行う。
5. `x`で別itemを理由付きrejectする。
6. Completion画面でblockerがないことを確認する。
7. Approval画面でexact revision ID/hashとapproved content hashを照合する。
8. 表示されたfresh challengeを手入力する。

Expected: 途中でシェルへ戻らず`approved` revisionとimmutable snapshotが発行され、snapshot IDと保存先が表示される。cancel、KeyRelease、空理由、期限切れtokenではmutationされない。

## 5. Audit history and provenance

DashboardからHistoryを開く。2 revisionを選択してDiffへ進み、変更item、before/after、actor、operation、producing revisionを確認する。Snapshot画面ではapproval linkageとhashを確認する。

Expected: historyはreachable chainとorphanを混在させず、任意pathではなくstable IDからverified artifactを表示する。

## 6. Validate stale and timeout behavior

TUIで確認modalを開いた後、別processから同じIntentへ有効なmutationを行い、古いmodalを確定する。またcore timeout/crash fixtureを使ってrequestを中断する。

Expected: stale mutationは自動再実行されず、preview/challengeは破棄される。canonical stateがreloadされ、同じstable itemが存在すれば選択が維持される。terminalは正常に復元される。

## 7. Recovery workflow

core publication形式のtemporary fixtureとorphan revision fixtureを用意し、Recovery画面を開く。

1. HEAD validity、reachable chain、orphan、temporary candidateを確認する。
2. temporary candidateだけを選択する。
3. cleanup確認後、処理結果を確認する。

Expected: cleanup直前にcandidate setとHEADが再検証される。選択後に変化した対象、reachable revision、snapshot、orphan revisionは削除されない。

## 8. Layout and performance

1,000 itemsを持つfixtureでwide/narrow terminalのgolden testとnavigation benchmarkを実行する。

Expected: navigation/renderの95%が100ms以内で、narrow layoutでもIntent/revision/lifecycle、current target、blocker/error、Esc/quit helpが表示される。

契約の詳細は[workspace-protocol.md](contracts/workspace-protocol.md)、画面操作は[tui-navigation.md](contracts/tui-navigation.md)、状態モデルは[data-model.md](data-model.md)を参照する。

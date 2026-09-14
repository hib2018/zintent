# Research: Full-screen Intent Workspace

## Application composition

**Decision**: 単一のroot workspace modelが共有sessionとactive screenを所有し、Intent list、dashboard、review、comments、completion、approval、history/diff、validation、snapshot、recoveryへmessageをroutingする。初期リリースは同時に1 modalだけを許す。

**Rationale**: canonical Intent、actor、terminal size、active requestを一元化しつつ、各screen reducerを独立テストできる。

**Alternatives considered**: 巨大な単一review reducerは状態の組合せが増えすぎる。screenごとの独立programはterminal lifecycleとshared stateを分断する。

## Navigation and human input

**Decision**: `Intent list → dashboard → task screen → detail/modal`の階層とstable-ID selectionを採用する。入力を要する操作は`closed → editing → preview-loading → confirming → submitting → reload-loading → closed/error`で管理する。

**Rationale**: text入力中のshortcut誤発火、double submission、preview bypass、expired capability reuseを状態として防げる。

**Alternatives considered**: numeric cursor永続化は並び替えに弱い。文字列だけの汎用modalは操作ごとの必須入力と多段階確認を表現できない。

## Canonical reload and concurrency

**Decision**: one-shot core processを維持し、request ID、operation、target ID、starting revisionを持つtyped resultを使う。mutation成功後は必ず`show_intent`を実行し、canonical reload成功後だけ表示状態を置換する。stale/timeout時にmutationを自動再実行しない。

**Rationale**: late responseとuncommitted stateの表示を防ぎ、人間判断の重複適用を避ける。

**Alternatives considered**: optimistic updateとautomatic retryはartifact stateと表示が乖離する可能性がある。

## Workspace discovery

**Decision**: workspace rootを明示入力とし、直下のIntent directoryをcoreが検証してstable Intent ID順に返す。path escapeとsymlink traversalを拒否し、壊れたentryは個別findingとして残す。

**Rationale**: 探索範囲が予測可能で、Go側にstore validationを複製しない。

**Alternatives considered**: recursive scanは範囲と重複が曖昧になる。Goによるfilesystem列挙はcore ownershipに反する。

## Draft import

**Decision**: `inspect_draft`でvalidation、source hash、proposed destination、collisionを提示して短命tokenを発行し、`import_draft`がexact source/workspace/destinationを再検証してatomic publicationする。

**Rationale**: 元Draftを変更せず、確認前mutation、TOCTOU、partial importを防げる。

**Alternatives considered**: 単純copyはuntrusted IDやpartial stateを残し得る。TUIによるID発行はdomain ownership違反になる。

## Verified audit reads

**Decision**: `list_revisions`、`inspect_revision`、`inspect_snapshot`を追加し、stable IDだけを入力としてhash/linkageを検証する。orphanはcanonical historyと分離する。

**Rationale**: TUIが任意pathを開かず、監査表示がverified artifactだけに基づく。

**Alternatives considered**: `show_intent`への多数のflag追加は責務が曖昧になる。Goの直接file readはintegrity検査を分散する。

## Recovery cleanup

**Decision**: `recovery_status`がcandidate-set tokenを返し、`cleanup_temporary_files`はexpected HEAD、token、明示選択candidate IDsを要求する。lock内で再走査し、一致するcore publication tempだけを削除する。orphan revisionは削除しない。

**Rationale**: 観測後の競合とunconfirmed deletionを防ぎ、reachable revision/snapshotを保護する。

**Alternatives considered**: startup時の自動cleanup、suffix一括削除、orphan自動削除はいずれも確認と復旧可能性を損なう。

## Approval and TTY authority

**Decision**: foreground interactive workspaceだけがapproval modalを開き、actor、exact revision/hashes、fresh challengeを表示する。KeyPress由来のresponseを即時送信し、flag、environment、prefill、generic messageからresponseを供給できないようにする。cancel、navigation、stale、expiry、terminal lossでtokenと入力を破棄する。

**Rationale**: full-screen continuityを保ちながらFeature 001のhuman gestureとone-use core capabilityを維持する。

**Alternatives considered**: 外部approval CLIへのsuspendはTUI完結要件に反する。`interactive_tty:true`だけをauthorityの証明とみなすのは不十分である。

## Terminal safety and test strategy

**Decision**: terminal modeとalternate screenはroot programだけが所有する。pure reducer、view golden、fake-command contract、PTY integration、real-core journeyで検証する。shutdown中のmutationは成功と断定せず、次回canonical reloadで確定する。

**Rationale**: reducerだけではterminal restorationを証明できず、PTYだけでは全状態分岐を高速に検査できない。

**Alternatives considered**: manual ANSI cleanupはterminal ownerを競合させる。snapshot testだけではdispatchとfailure safetyを検証できない。

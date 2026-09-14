# Feature Specification: Full-screen Intent Workspace

**Feature Branch**: `[002-fullscreen-tui-workspace]`

**Created**: 2026-09-15

**Status**: Draft

**Input**: User description: "全ての操作を全画面のTUI上で完結できるようにする"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reviewから承認までTUIで完結する (Priority: P1)

Intent ownerは、Draftを開いた後にシェルへ戻ることなく、項目の確認、編集、コメント、却下、レビュー完了、最終承認を一つの全画面ワークスペースで完了できる。

**Why this priority**: zintentの主要価値は、人間が解釈へ介入し、確認済みの成果物を安全に発行することにある。主要工程が複数の画面外コマンドへ分断されると、対象revisionの取り違えや操作漏れが起きやすい。

**Independent Test**: 有効なDraftをTUIで開き、accept、edit、comment、resolve、reject、complete review、approval challenge入力を順に行い、TUIを終了せずApproved Intent Snapshotを取得できることを確認する。

**Acceptance Scenarios**:

1. **Given** 有効なDraft、**When** 利用者がレビュー画面を開く、**Then** Intentの状態、現在revision、actor、全item、review status、残りblockerが表示される。
2. **Given** レビュー中のitem、**When** 利用者がaccept、edit、comment、rejectのいずれかを選ぶ、**Then** 必要な入力と確認画面がTUI内に表示され、確定後に新しいcanonical revisionが再読込される。
3. **Given** itemの編集内容、**When** 利用者が編集を進める、**Then** 適用前にbefore/afterと対象itemが表示され、表示された内容への明示確認後にだけ変更される。
4. **Given** open comment、**When** 利用者がresolveまたはwithdrawを選ぶ、**Then** 理由の入力が要求され、完了後にcomment statusと残りblockerが更新される。
5. **Given** 未レビューitemまたはopen comment、**When** 利用者がレビュー完了を選ぶ、**Then** 完了操作は拒否され、該当blockerから対象itemまたはcommentへ移動できる。
6. **Given** review completion条件を満たしたIntent、**When** 利用者が完了を明示確認する、**Then** lifecycleが`review_complete`となり、承認画面へ進める。
7. **Given** 承認可能なrevision、**When** 利用者が承認画面を開く、**Then** exact revision、revision hash、approved content hash、対象件数、除外件数、blocker、fresh challengeが表示される。
8. **Given** fresh challenge、**When** 人間が同じ対話端末でchallengeを正確に入力する、**Then** Approved Intent Snapshotが発行され、その識別子と保存先がTUIに表示される。
9. **Given** stale、期限切れ、再利用済み、または不一致の確認情報、**When** 承認を試みる、**Then** 承認は発行されず、再確認または再読込のための案内が表示される。

---

### User Story 2 - 履歴・検証・復旧をTUIから監査する (Priority: P2)

Intent ownerは、現在状態だけでなくrevision履歴、item単位の差分、provenance、validation finding、snapshot、復旧候補を同じワークスペースで確認できる。

**Why this priority**: 承認の信頼性には、何が誰によって変わったかを会話履歴に頼らず確認できることが必要である。

**Independent Test**: 複数revisionを持つIntentを開き、履歴から2つのrevisionを選択して差分を表示し、actorとprovenanceを確認し、validationおよびrecovery statusを閲覧できることを確認する。

**Acceptance Scenarios**:

1. **Given** 複数revisionを持つIntent、**When** 履歴画面を開く、**Then** 各revisionのID、親revision、操作、actor、日時、lifecycleが識別できる。
2. **Given** 2つのrevision、**When** 比較する、**Then** stable item IDごとにbefore/afterと変更種別が決定論的な順序で表示される。
3. **Given** 記録済み操作、**When** 詳細を開く、**Then** content origin、operation actor、operation ID、producing revision、source referenceを確認できる。
4. **Given** current Intent、**When** validationを実行する、**Then** findingがseverity、対象record、path、解決のための説明とともに表示される。
5. **Given** Approved Intent Snapshot、**When** snapshot詳細を開く、**Then** approved revision、content hash、approving actor、validation resultを確認できる。
6. **Given** orphan revisionまたは一時ファイル、**When** recovery画面を開く、**Then** 自動変更せず候補を一覧表示し、安全に削除可能な対象だけを明示確認後に処理できる。

---

### User Story 3 - 複数Intentをワークスペースで管理する (Priority: P3)

ローカル利用者は、対象パスを毎回入力せず、利用可能なIntentの一覧から作業対象を選択し、Draftの取込、再開、検索を行える。

**Why this priority**: 日常運用では複数Intentを並行して扱うため、現在状態と次の操作を一覧から判断できる入口が必要になる。

**Independent Test**: 複数の異なるlifecycleとblocker数を持つIntentを一覧表示し、検索、選択、再開、Draft取込をTUI内で実行できることを確認する。

**Acceptance Scenarios**:

1. **Given** 複数のIntent、**When** ワークスペースを開く、**Then** 各Intentの識別子、名称、lifecycle、blocker数、承認状態が一覧表示される。
2. **Given** Intent一覧、**When** 利用者が検索または絞り込みを行う、**Then** 一致するIntentだけが表示され、選択中のstable Intent IDが維持される。
3. **Given** 中断されたreview、**When** Intentを再度開く、**Then** canonical artifactから状態が復元され、最初の未解決対象へすぐ移動できる。
4. **Given** schema-validな外部Draft、**When** 利用者が取込を選び対象を確認する、**Then** 新しい管理対象として登録され、元ファイルを上書きせずreviewを開始できる。
5. **Given** 不正または重複したDraft、**When** 取込を試みる、**Then** 登録は行われず、修正可能なfindingが表示される。

### Edge Cases

- 端末幅または高さが推奨値より小さい場合、重要な状態、blocker、選択対象、終了方法を失わず縮退表示する。
- 長いstatement、comment、findingは切り捨てによって意味を失わせず、スクロールまたは詳細表示で全文を確認できる。
- 画面表示後に別プロセスがrevisionを更新した場合、古い判断を自動再実行せず、最新状態を再読込して利用者へ再確認を求める。
- core処理のtimeout、cancel、異常終了時は端末を復元し、成功が確認されていないmutationを画面へ反映しない。
- edit previewまたはapproval challengeの期限切れ後は、古い入力を流用せずfresh capabilityを取得する。
- approval中にrevisionが変化した場合、入力済みchallengeを破棄し、新しいrevisionとhashを表示する。
- validation findingが多数ある場合でも、severityおよび対象recordによる移動と絞り込みができる。
- orphanやtemporary fileのcleanup対象が処理直前に変わった場合、再検証し、確認していない対象を削除しない。
- Approved Intentへの通常変更は新しい`in_review` revisionとして扱い、既存snapshotを変更しない。
- TUIを強制終了して再起動しても、確定済み操作だけがcanonical artifactから復元される。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: システムは、Intent選択、review、comment管理、review completion、approval、history、diff、validation、snapshot inspection、recoveryを一つの全画面ワークスペースから利用可能にしなければならない。
- **FR-002**: システムは、どの画面でもIntent ID、current revision、lifecycle、actor、未解決blocker数を確認可能にしなければならない。
- **FR-003**: システムは、itemのaccept、edit、comment、reject、およびcommentのresolve、withdrawに必要な入力・取消・確認をTUI内で完結させなければならない。
- **FR-004**: editは、対象item ID、before、afterを提示したexact previewへの人間の明示確認なしに適用してはならない。
- **FR-005**: rejectは理由を、comment resolveおよびwithdrawはclosure reasonを要求しなければならない。
- **FR-006**: mutation成功後、システムはcanonical artifactを再読込し、成功が確認されたrevisionだけを表示しなければならない。
- **FR-007**: stale revisionを検出した場合、システムは保留中の操作を自動再実行せず、最新状態と変更された対象を利用者へ示さなければならない。
- **FR-008**: review completion画面はすべてのblocking findingを表示し、対象recordへ移動できるようにしなければならない。
- **FR-009**: approval画面は、confirmed revision IDとhash、approved content hash、対象・除外件数、validation resultをchallenge入力前から表示し続けなければならない。
- **FR-010**: approvalは、同一の対話端末上で人間がfresh challengeを入力した場合に限り実行可能でなければならない。
- **FR-011**: システムはapproval challengeを生成、推測、自動入力してはならず、非対話環境ではapproval操作を拒否しなければならない。
- **FR-012**: approval成功後、システムはsnapshot ID、保存先、approved revision、content hashを表示し、snapshot詳細へ移動できるようにしなければならない。
- **FR-013**: history画面はrevision chainを親子関係が分かる順序で表示し、各revisionの操作とactorを確認可能にしなければならない。
- **FR-014**: diff画面は2つのverified revisionをstable item IDで比較し、追加、変更、除外およびbefore/afterを識別可能にしなければならない。
- **FR-015**: validation画面はfindingのcode、severity、対象record、path、messageを表示しなければならない。
- **FR-016**: recovery画面はHEAD整合性、revision chain、orphan、temporary fileを表示し、読み取り専用の検査と明示確認を要するcleanupを区別しなければならない。
- **FR-017**: cleanupは直前に対象を再検証し、利用者が確認していないファイルまたはreachable artifactを削除してはならない。
- **FR-018**: Intent一覧は各Intentのstable ID、表示名、lifecycle、blocker数、approval状態を表示し、検索および選択を提供しなければならない。
- **FR-019**: Draft取込は取込前に検証結果と登録先を表示し、元Draftを変更せず、失敗時に部分的なIntentを残してはならない。
- **FR-020**: TUIはIntent、revision、snapshot、capability registryを直接変更せず、すべての永続操作を正式なdomain operationとして実行しなければならない。
- **FR-021**: TUI、CLI、conversation経路は同じ操作に対して同じvalidation、transition、provenance、revision、approval効果を持たなければならない。
- **FR-022**: すべての入力画面は確定と取消を区別し、取消、終了、timeoutでは永続状態を変更してはならない。
- **FR-023**: 端末終了時には表示状態を復元し、処理中の操作が成功したか不明な場合はcanonical stateの再検証を案内しなければならない。
- **FR-024**: システムはキーボードだけで全操作を完了可能にし、現在利用可能なキーと次に必要な操作を画面内で確認可能にしなければならない。
- **FR-025**: 狭い端末でも現在対象、lifecycle、blocker、error、取消・終了操作を表示しなければならない。
- **FR-026**: TUIを閉じて再起動した場合、選択していたstable recordを可能な範囲で復元し、存在しない場合は最初の未解決対象を選択しなければならない。
- **FR-027**: Approved Intentへの通常変更は、新しいworking revisionを`in_review`として表示し、既存Approved Intent Snapshotを保持しなければならない。

### Scope Boundaries

- このFeatureは、Feature 001で提供されたIntent review、approval、audit、recovery操作を全画面TUIから利用可能にする。
- Intentの一覧表示、schema-valid Draftの取込、revision/snapshot inspection、安全なtemporary cleanupを追加対象とする。
- CLIおよびskill経路は廃止せず、同じdomain semanticsを利用する代替インターフェースとして維持する。
- AIによるIntent解釈、AI revision提案、Planner handoff、Spec Kit adapter、認証・認可、remote collaboration、multi-reviewer policyは対象外とする。
- snapshotおよびrevisionの直接編集は対象外であり、TUIからも許可しない。
- mouse操作、テーマの高度なカスタマイズ、全文検索、複数window、network storeは初期リリースの対象外とする。

### Key Entities

- **Workspace Entry**: 選択可能なIntentを表し、stable Intent ID、表示名、lifecycle、current revision、blocker summary、approval状態を持つ。
- **Workspace Session**: 現在選択中のIntent、画面、stable record selection、保留中の人間入力を表す。永続的なIntent状態とは区別される。
- **Review Draft Input**: edit statement、comment body、rejection rationale、closure reasonなど、確定前の一時的な人間入力。
- **Action Preview**: 対象record、before/after、exact revision、期限、確認状態を表す一時的な操作候補。
- **Approval Preview**: confirmed revisionとhash、approved content hash、validation result、fresh challengeを表す一時的な承認候補。
- **Audit Selection**: 比較元・比較先revision、選択したfinding、provenance、snapshotを表す読み取り専用の選択状態。
- **Recovery Candidate**: orphanまたはtemporary fileの検査結果であり、reachable artifactとは区別される。cleanup直前に再検証される。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 代表的な初回利用者の90%以上が、シェルへ戻らずDraftの選択からApproved Intent Snapshotの確認までを15分以内に完了できる。
- **SC-002**: すべてのFeature 001操作がキーボードのみで到達可能であり、各操作の確定・取消・失敗を画面上で識別できる。
- **SC-003**: mutation成功後、利用者が2秒以内に新しいrevision、状態、対象recordの結果を確認できる。
- **SC-004**: stale、timeout、cancel、challenge mismatchを含む失敗シナリオの100%で、未確認のmutationが適用されず、次に必要な復旧行動が表示される。
- **SC-005**: 1,000件のitemを持つIntentで、通常の移動入力の95%以上が100ミリ秒以内に画面へ反映される。
- **SC-006**: 利用者は履歴画面を開いてから30秒以内に、指定した2 revisionの変更itemとactorを特定できる。
- **SC-007**: approval前の確認試験の100%で、利用者が対象revision ID、revision hash、approved content hashを画面上で照合できる。
- **SC-008**: 強制終了後の再開試験の100%で、最後に確定したcanonical revisionが復元され、未確定入力は永続状態へ反映されない。
- **SC-009**: recovery試験の100%で、reachable revisionおよびApproved Intent Snapshotがcleanup対象に含まれない。
- **SC-010**: 推奨端末サイズおよび狭い端末の表示試験で、現在対象、状態、blocker、error、help、終了操作が常に確認できる。

## Assumptions

- 対象利用者はローカル端末を使用する単一の人間reviewerであり、同時編集時は既存のstale revision制御を利用する。
- Feature 001のschema、lifecycle、revision、approval、snapshot、provenance契約を変更せず再利用する。
- Intent一覧の探索範囲は、起動時に指定された単一のローカルworkspace配下とする。
- Draft取込元はローカルのschema-valid artifactであり、AIによる内容生成は行わない。
- 一時的な入力や画面選択はIntentの承認内容ではなく、確定操作が成功した場合だけcanonical artifactへ反映される。
- approvalの人間確認は、既存のTTY要件と同等以上の明示性を維持する。
- UI固有の表示状態はdomain stateへ混入させず、再開に必要な正式状態はartifactから復元する。

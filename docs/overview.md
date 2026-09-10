# プロジェクト概要

## Intentとは

Intentは単なるプロンプトではありません。人間の要求をAIがどう解釈したかを、個別に確認可能な
項目、その根拠、未決定事項、レビュー結果、履歴として表現したArtifactです。

目的は、実装や計画の前に「何を合意したか」を固定することです。Plannerが優れた設計を作っても、
前提の解釈が人間の意図と違えば成果は正しくありません。zintentはこの解釈境界へ人間が介入
できるようにします。

## TUI製品ではなくプロセス基盤

zintentは次の組み合わせです。

- user-recognizable use case単位のskill
- 状態を決定論的に扱うtoolとdomain core
- 工程間を接続するversioned Artifact
- TUI、CLI、会話インターフェース
- Planner固有のadapter skill

TUIを交換しても状態遷移や承認条件は変わりません。別のAgentがCLIを利用しても、skillを使わず
人間がCLIを実行しても、同じdomain ruleが適用されます。

## 想定するskill群

| Skill | 役割 |
|---|---|
| `zintent-interpret` | 自然言語要求からDraftを提案する |
| `zintent-review` | 人間のレビュー操作を支援する |
| `zintent-revise` | commentから改訂案を作る。自動適用しない |
| `zintent-check` | 機械的・意味的な問題を区別して検査する |
| `zintent-approve` | eligibility確認と人間の承認へ接続する |
| `zintent-handoff` | Approved Snapshotから汎用handoffを作る |
| `zintent-speckit` | Spec Kit固有の変換と追加解釈を検出する |

Feature 001の実装対象は、最小の`zintent-review`と`zintent-approve`です。

## Walking Skeletonの仮説

Feature 001は、次の一点を端から端まで検証します。

> 人間が項目レベルでAIの解釈へ介入し、その結果だけを承認済みArtifactとして取得できるか。

最初はfixtureまたは手入力のDraftから始めます。AIによるDraft生成や改訂は、安全なレビュー・
承認経路が成立してから追加します。

## 成功の判断

- accept、edit、comment、rejectがrevisionとして残る
- 終了後もHEADから正確に再開できる
- 未レビューItemやopen commentが承認を止める
- revisionとhashを人間が明示的に確認する
- Snapshotが後続変更の影響を受けない
- CLIとTUIが同じtransitionを使う

定量条件は[Feature spec](../specs/001-intent-review-skeleton/spec.md)を参照してください。

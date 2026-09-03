---
purpose: 対象ユーザーと答える問いの決定（ADR の却下基準を供給する）
triggers: [persona, ペルソナ, product, 対象ユーザー, usecase, 目的]
audience: all
token_weight: small
---

# 製品決定

> **未記入。** ここが空の間は `docs/adr/` の ADR は書けない（却下の基準が無いため）。
> 埋めるのは1人・1問だけ。2人以上・2問以上書いた場合、それは決定できていないという意味である。

## 対象ユーザー（1人だけ）

<!-- 属性 / 保有物件数 / 手元にある情報 / 使う瞬間 を具体的に -->

## この道具が答える問い（1つだけ）

<!-- 「〜すべきか」の形の疑問文1つ -->

## 対象外（明示的に捨てる段階）

<!-- 下の段階表から番号で。捨てる理由も書く -->

## 採用する正の実装

| 分岐 | 採用 | 却下 | 理由 |
|------|------|------|------|
| | | | |

---

# 決定のための参考資料

## 実装が現在カバーしている7段階

投資家がどの段階の、どの判断に使うかで実装を分類したもの。
`〈E〉` は API エンドポイント、それ以外はフロントのパネル。

| 段階 | 判断 | 実装 | この段階の利用者が持っていない情報 |
|---|---|---|---|
| 1. エリアを決める | どの市区町村を狙うか | `AreaDiscovery`, `InvestmentScoreHeatmap`, `InvestmentScoreCard`, 〈E〉`area-discovery`, `investment-score`, `investment-score-heatmap`, `population-forecast` | 物件が無い（価格・賃料・築年数すべて不明） |
| 2. 物件を値踏みする | この価格は割安か | `LandPriceAnalysis`, `YieldAnalysis`, `YieldGauge`, `KpiStrip`, `StatusSummary`, `CriticalErrorBanner`, `HazardAlertBanner`, `RentValidationHint`, 〈E〉`land-prices/*`, `station-ridership`, `land-appraisals`, `urban-risks`, `hazard`, `rent-stats` | 融資条件が未確定 |
| 3. 融資を組む | いくら借りてどう返すか | `LoanComparePanel`, `LoanOptimizationPanel`, `DscrBadge`, `CashFlowChart`, `DeadCrossChart`, `CostBreakdown` | — |
| 4. リスクを検証する | 悪化しても耐えるか | `StressScenarioTable`, `CustomScenarioSlider`, `MonteCarloChart`, `TaxSimulationPanel`, 〈E〉`investment/simulate`, `investment/rent-decline-hint` | — |
| 5. 出口を決める | 何年後にいくらで売るか | `MultiExitCompareTable` | — |
| 6. 実際に買う | 交渉と実務 | `NegotiationPanel`, `DueDiligenceChecklist`, `RenovationPanel`, 〈E〉`renovation/analyze` | — |
| 7. 候補を管理する | 複数物件の比較 | `WatchlistPanel`, `WatchlistCompareTable` | — |

読み方:

- **段階1と段階2〜6は別の製品**。段階1の利用者は物件を持たないため、詳細モードの
  入力前提（建築費・想定賃料・築年数）をひとつも満たせない。
  実装上も「エリアを探す」は独立タブに分離されている。
- **段階3〜5は同一人物・同一セッションの連続した判断**。ここは1製品として成立する。
- **段階6・7は購入プロセスの実務**で、滞在時間もセッション頻度もシミュレーターと異なる。
- 7段階を1製品で持つことは、物件検索サイトと投資シミュレーターと管理台帳を
  同時に作ることに等しい。

削減の判断は「機能の良し悪し」ではなく **どの段階を捨てるか** で行う。
個々の機能はいずれも妥当なので、機能単位で見ると何も削れない。

## 目的の選択肢

| | 案A 自分専用 | 案B 実ユーザー | 案C ポートフォリオ |
|---|---|---|---|
| 対象ユーザー | 開発者本人 | 1棟目を探す会社員（要・1人に確定） | 採用担当・技術者（読む人） |
| 残す段階 | 2-5 | 1-2 | 全部 |
| 当面使われない実装 | 約40% | 約50% | 0% |
| 作業量 | 小 | 大 | 小〜中 |
| 「何のためか」の回復 | 即座 | 数週間後 | 回復しない（仕様化する） |

案A→案B への移行は段階2が共通なので可能。案C は他2案の後からでも取れる。

## 決定に効く現状の事実（2026-09 時点）

- **`feat` コミットが 2026-06 以降ゼロ**（2026-04 は197件）。直近は依存更新と CI のみ
- **計測が無い**。分析・エラートラッキングの実装はゼロなので、誰が何をしているか分からない
- **README に稼働URL・スクリーンショット・デモが無い**。ユーザーとして製品に触れる導線が
  存在せず、開発者自身もコードベースとしてしか製品に接触できない
- `docs/usecases.md` の UC-01〜UC-15 は **15件すべての「誰が」が「投資家」**。
  内容も人物像ではなく機能の前提条件（例: UC-13「築年数・駅距離を把握している投資家」）

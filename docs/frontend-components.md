# フロントエンドコンポーネント仕様

`frontend/src/components/` 配下。フレームワーク: Next.js 16 (App Router)

---

## 状態フローの概要

```
page.tsx
  └── Dashboard
        ├── InvestmentForm         (入力 → onAnalyze, onFetchLandPrices)
        ├── LandPriceAnalysis      (comparison + populationForecast を受け取り表示)
        ├── YieldAnalysis          (result + populationForecast を受け取り表示)
        ├── LoanOptimizationPanel  (result + loanMethod を受け取り DSCR・LTV感度を表示)
        ├── CostBreakdown          (result.acquisitionCosts + yearlyResults を受け取り表示)
        ├── CashFlowChart          (result + equityInvested を受け取り表示)
        ├── DeadCrossChart         (result を受け取り表示)
        ├── MonteCarloChart        (monteCarloResult を受け取り確率分布を表示。ボタン押下後のみ表示)
        └── ReportPDF              (result + lastInput を受け取り PDF 生成。SSR無効)
```

`Dashboard` が `result: InvestmentResult | null`、`comparison: LandPriceComparison | null`、`populationForecast: PopulationForecastResult | null` を管理する。

---

## Dashboard

`frontend/src/components/Dashboard.tsx`

> テストカバレッジ: `Dashboard.test.tsx`（Vitest + RTL）— 状態管理・API呼び出しフロー・子コンポーネントへの props 伝達を検証。

**状態管理**:
- `result`: `InvestmentResult | null`
- `lastInput`: `InvestmentInput | null` — 直近のシミュレーション入力。PDF 出力に使用
- `loanMethod`: `LoanMethod` — `"equal-payment"` | `"equal-principal"`。初期値 `"equal-payment"`。変更時に直近の `lastInput` で再解析を実行する
- `comparison`: `LandPriceComparison | null`
- `stationRidership`: `StationRidershipResult[] | null` — 緯度・経度が指定された場合に `GET /api/station-ridership` から取得
- `populationForecast`: `PopulationForecastResult | null` — 緯度・経度が指定された場合に `GET /api/population-forecast` から取得
- `externalUrbanRisks`: `UrbanRisk[] | null` — 緯度・経度が指定された場合に `GET /api/urban-risks` から取得（XKT003/020/030/XST001）
- `loading`, `error`: ローディング・エラー状態

**「PDFレポート出力」ボタン**:

`result && lastInput` が両方存在する場合のみヘッダーにボタンを表示。`@react-pdf/renderer` の `PDFDownloadLink` を `next/dynamic(..., { ssr: false })` でクライアント専用モジュールとして読み込む。ファイル名: `yield-guard-report-YYYYMMDD.pdf`。

**`handleFetchLandPrices(area, city, lat?, lng?)`**:

lat/lng が両方渡された場合、`fetchStationRidership`・`fetchPopulationForecast`・`fetchUrbanRisks` を `Promise.allSettled` で並行実行する。いずれか一方が失敗しても他の結果は利用する。呼び出し開始時に全ステートを `null` でリセットする。

**`equityInvested` の計算**:
```typescript
const equityInvested = result.totalInvestment - input.loanAmount
```
自己資金（頭金 + 諸経費）。`CashFlowChart` に渡して投資回収年計算に使用。

---

## LoanOptimizationPanel

`frontend/src/components/LoanOptimizationPanel.tsx`

> テストカバレッジ: `LoanOptimizationPanel.test.tsx`（Vitest + RTL）— DSCR バッジ・LTV感度テーブル・返済方式セレクタの6ケースを検証。

**props**:
- `result: InvestmentResult`
- `loanMethod: LoanMethod`
- `onLoanMethodChange: (method: LoanMethod) => void`

**DSCR 表示**:

`result.dscr` の値に応じてバッジの色を変化させる。

| 条件 | バッジ | 色 |
|------|--------|-----|
| `dscr >= 1.0` | 「安全（≥ 1.0）」 + `CheckCircle` | 緑 |
| `dscr < 1.0` | 「危険（< 1.0）」 + `AlertTriangle` | 赤 |

表示値は小数点2桁（`.toFixed(2)`）。副題に「NOI ÷ 年間返済額（1年目）」を表示。

**返済方式セレクタ**:

カードヘッダー右に `<select>` を配置。`value = loanMethod`（`"equal-payment"` / `"equal-principal"`）。変更時に `onLoanMethodChange` を呼び出し、`Dashboard` が再解析を実行する。

**LTV 感度分析テーブル**（`result.ltvSensitivity.length > 0` の場合のみ表示）:

| 列 | 内容 |
|----|------|
| LTV | `formatPct(row.ltv)` |
| 自己資金 | `formatMan(row.equity)` |
| 借入額 | `formatMan(row.loanAmount)` |
| DSCR | `row.dscr.toFixed(2)` — `>= 1.0`: 緑文字 / `< 1.0`: 赤文字 |
| 年間CF | `formatMan(row.annualCF)` — 負値は赤文字 |
| CF利回り | `formatPct(row.cfYield)` — 負値は赤文字 |

ヘッダー副題に「ベースケース基準」を表示。元金均等の場合は「元金均等は1年目返済額で試算」を追記。

---

## InvestmentForm

`frontend/src/components/InvestmentForm.tsx`

**props**:
- `onAnalyze(input: InvestmentInput): void`
- `onFetchLandPrices(params): void`
- `loading: boolean`

**万円・パーセント変換ヘルパー**:
- `toMan(v: number) = v / 10_000` — 円→万円（表示用）
- `fromMan(v: number) = v * 10_000` — 万円→円（送信時）
- `toPct(v: number) = v * 100` — 率→%（表示用）
- `fromPct(v: number) = v / 100` — %→率（送信時）

**市区町村インクリメンタルサーチ**:

都道府県選択に連動して `GET /api/municipalities?area=XX`（XIT002）を呼び出し、市区町村一覧を動的に取得する。取得後はテキスト入力でリアルタイムフィルタリングが可能。

```
初回マウント時: 初期都道府県（"10" = 群馬県）の市区町村を取得
都道府県変更時: loadMunicipalities(code) → setCity("") → setMuniFilter("") → fetch → setCity(data[0].id)
フィルタ入力時: filteredMunicipalities（useMemo）が再計算 → ドロップダウンを絞り込み
               → 結果が1件になると useEffect が setCity() で自動選択
```

| 状態 | ドロップダウン表示 |
|------|-----------------|
| 取得中 | 「読み込み中...」（disabled） |
| 取得失敗 / データなし | 「（全市区町村）」のみ |
| フィルタ一致なし | 「該当なし」（disabled） |
| フィルタあり・複数一致 | 絞り込み結果 + 件数表示 |
| フィルタなし | 「（全市区町村）」+ 全市区町村名 |

「（全市区町村）」は `city=""` に対応し、バックエンドで市区町村絞り込みなしの都道府県全体検索になる。

**状態変数**:
- `muniFilter: string` — テキスト入力の現在値
- `filteredMunicipalities` — `useMemo([municipalities, muniFilter])` でメモ化済み

**最寄り駅徒歩（`stationMinutes`）**:

`InvestmentInput.stationMinutes`（0=未入力）。「相場データを取得」押下時に `GET /api/land-prices/estimate` へ渡され、理論価格の駅距離補正に使用される。0のままでも理論価格は築年数補正のみで算出される。

**物件の緯度・経度（任意入力）**:

`propertyLat`, `propertyLng`（コンポーネント内ローカル状態、`InvestmentInput` には含まれない）。都道府県・市区町村セレクトの下に2カラムグリッドで配置。「相場データを取得」押下時に有効な数値が入力されていれば `Dashboard.handleFetchLandPrices` に渡され、`GET /api/station-ridership?lat=&lng=` の呼び出しに使用される。未入力または不正値（`parseFloat` → `NaN`）の場合は undefined として渡され、駅別乗降客数の取得はスキップされる。

**クイック / 詳細モード（`simulationMode`）**:

`SimulationMode = "quick" | "full"` で切り替える2モード設計。モード切替ボタンにサブタイトル（「内覧でサッと試す」／「じっくり分析する」）を表示。

| | クイックモード | 詳細モード |
|---|---|---|
| 入力項目 | 3項目（物件価格総額・月額賃料・ローン金額） | 全フィールド（16項目+） |
| 土地相場データ | 折りたたみ（初期非表示・任意） | 常時展開 |
| フォーム構造 | フラット | 4セクション（物件情報/収益条件/ローン条件/出口戦略） |
| ストレステスト | 非表示 | 常時表示 |
| デフォルト値バナー | 表示（空室率5%・経費率20%等を明示。現金購入時は「現金購入（ローンなし）」に変化） | なし |
| 現金購入チェックボックス | ローン金額フィールドの上に表示 | ローン条件セクションのヘッダー右側に表示 |

**現金購入モード（`isCashPurchase`）**:

「現金購入（ローンなし）」チェックボックスをクイック・詳細モード両方に設置。

- チェック ON: `loanAmount=0`・`loanYears=0` をセットし、ローン金額・年利・返済期間フィールドを `disabled` にする。チェック前の値は `savedLoanAmount`・`savedLoanYears` に退避する。
- チェック OFF: 退避値を復元してフィールドを再有効化する。
- モード切り替え時（クイック→詳細、詳細→クイック）: `useEffect` でチェック状態をリセットし退避値を復元する。
- バックエンドの `calcMonthlyPayment` は `LoanYears <= 0` で 0 を返すガードがあるため、`loanYears=0` を送信しても安全。

クイックモード送信時は「物件価格総額」（`quickTotalPriceMan`）を `landPrice = total × 0.7`・`buildingCost = total × 0.3` に自動分割し、`QUICK_MODE_DEFAULTS` とマージして API へ送信する。

旧来の「詳細設定トグル（`showAdvanced`）」は廃止済み。`expenseRate`・`incomeTaxRate`・`buildingAge`・`buildingType`・`exitYieldTarget`・`rentDeclineRate` は詳細モードの各セクションにフラット配置。

**レスポンシブグリッド**:

すべての入力グリッドは `grid-cols-1 sm:grid-cols-2`（または `sm:grid-cols-3`）で実装されており、モバイル（`sm` ブレークポイント未満）では縦1列に、タブレット以上では2〜3列に切り替わる。

**クライアントサイドバリデーション**:
「シミュレーション実行」押下時に `validateQuick` または `validateFull` を実行し、エラーがあれば API を呼ばずフィールド直下にエラーメッセージを表示する。フィールドの値を変更するとそのフィールドのエラーをクリアする。

| モード | バリデーション対象 |
|---|---|
| クイック | 物件価格総額・月額賃料・ローン金額（3項目） |
| 詳細 | 全フィールド（バックエンドの `validateInvestmentInput()` と同一） |

| フィールド | 条件 |
|-----------|------|
| `landPrice`, `buildingCost` | 1〜100億円 |
| `monthlyRent` | 正の値 |
| `vacancyRate` | 0〜99% |
| `loanAmount` | 0以上 |
| `annualLoanRate` | 0〜30% |
| `loanYears`, `holdingYears` | 0〜50年 |
| `miscExpenseRate` | 0〜50% |
| `expenseRate` | 0〜90% |
| `incomeTaxRate` | 0〜60% |
| `exitYieldTarget` | 0%超〜50% |

**NPV / IRR 設定サブセクション**（詳細モードの 出口戦略 セクション末尾）:

| 入力欄 | フィールド | デフォルト | 備考 |
|--------|-----------|----------|------|
| 割引率 | `discountRate` | 5% | step=0.1%、0〜30% |
| 物件価格下落率 | `priceDeclineRate` | 0% | step=0.1%、0〜10% |
| 減価償却方式 | `depreciationMethod` | 定額法 | セレクト: 定額法/定率法 |

**ストレステストスライダー**:
- `vacancyRateDelta`: 0〜30%（空室率の追加シナリオ）
- `loanRateDelta`: 0〜3%（金利上昇シナリオ）
- `InvestmentInput.vacancyRateDelta`, `loanRateDelta` にそのまま渡す

**用途地域リスク警告（`zoningType`）**:

`lib/zoning.ts` の `ZONING_META` を参照してインラインで色分けバナーを表示する。バックエンド呼び出しなし。

| riskLevel | 色 | ラベル |
|-----------|-----|--------|
| 0 | 緑 | 良好な住環境です |
| 1 | 青 | 低リスク |
| 2 | 黄 | 中リスク |
| 3 | 赤 | 高リスク |

`zoningType` は `InvestmentInput` に含まれない（表示のみのローカル状態）。

#### 建物価格 按分ヘルパー

「建物価格がわからない場合」トグルで展開する折りたたみUI。中古物件で建物・土地の内訳が不明な場合に補助する。

**方法①: 消費税額から計算**
消費税は建物部分にのみ課税されるため逆算できる。
建物価格 = 消費税額 ÷ 0.1

**方法②: 固定資産税評価額の比率で按分**
建物価格 = 購入総額 × 建物評価額 ÷（土地評価額 + 建物評価額）

「適用」ボタンで計算結果を建物価格フィールドに反映する。

---

## LandPriceAnalysis

`frontend/src/components/LandPriceAnalysis.tsx`

**props**:
- `comparison: LandPriceComparison`
- `input?: InvestmentInput | null`
- `theoreticalPrice?: TheoreticalPriceResult | null`
- `stationRidership?: StationRidershipResult[] | null`
- `populationForecast?: PopulationForecastResult | null`
- `landAppraisal?: AppraisalComparisonResult | null`
- `externalUrbanRisks?: UrbanRisk[] | null` — `GET /api/urban-risks` 由来（XKT003/020/030/XST001）

**表示の3状態**:

| 状態 | 条件 | 表示内容 |
|---|---|---|
| データなし | `stats.count === 0` | 「取引データが見つかりませんでした」メッセージのみ。統計グリッド・グラフ・判定バッジは非表示 |
| データ不足 | `stats.lowDataWarning === true`（count < 10） | 目立つ警告バナー（太枠・大アイコン）を表示。判定バッジに「（参考値）」を付与。比較エリアに「※ データ件数不足のため参考値」を追記 |
| 通常 | `stats.count >= 10` | 全要素を表示 |

**assessment の表示バリエーション**:
- `"割安"` → 緑バッジ（データ不足時は「割安（参考値）」）
- `"相場"` → 黄バッジ（データ不足時は「相場（参考値）」）
- `"割高"` → 赤バッジ（データ不足時は「割高（参考値）」）

**理論価格パネル（`theoreticalPrice`）**:

`theoreticalPrice?: TheoreticalPriceResult` prop が存在する場合のみ表示。`GET /api/land-prices/estimate` のレスポンス。

| `deviationPct` | 背景色 | ラベル |
|----------------|--------|--------|
| > +20% | 赤 | 割高 |
| < -20% | 緑 | 割安 |
| ±20%以内 | 青 | 相場 |

補正係数の内訳（築年数補正・駅距離補正・需要スコア補正）を `%` で表示。`hasStationData=false` のとき駅距離補正行は、`hasRidershipData=false` のとき需要スコア補正行はそれぞれ非表示。

**駅別乗降客数パネル（`stationRidership`）**:

`stationRidership` prop が1件以上ある場合のみ紫パネルで表示。`GET /api/station-ridership` のレスポンス（上位5件まで表示）。

| フィールド | 表示内容 |
|-----------|---------|
| `stationName` + `lineName` | 駅名（路線名） |
| `passengers` | 乗降客数/日（カンマ区切り） |
| `demandScore` | A〜E のカラーラベル |

需要スコアのカラー: A=紫・B=青・C=緑・D=黄・E=赤

**都市計画リスク警告パネル**:

`stats.urbanRisks`（XIT001 テキスト検出）と `externalUrbanRisks`（XKT003/020/030/XST001 API 検出）を `allUrbanRisks` としてマージして表示する。

```typescript
const allUrbanRisks: UrbanRisk[] = [
  ...(stats.urbanRisks ?? []),
  ...(externalUrbanRisks ?? []).filter(r => !(stats.urbanRisks ?? []).some(e => e.code === r.code)),
];
```

`allUrbanRisks.length > 0` の場合のみ表示。`RISK_STYLE` テーブル（ERROR=赤・WARNING=黄・INFO=青）でスタイルを決定。

**用途地域情報パネル（`stats.zoning`）**:

`stats.zoning` が存在する場合のみ青いパネルで表示。取引データの最頻値から抽出した参考情報。

| フィールド | 表示ラベル |
|-----------|----------|
| `cityPlanning` | 用途地域 |
| `buildingCoverage` | 建ぺい率 |
| `floorAreaRatio` | 容積率 |

各フィールドは個別に `&&` で存在チェックし、空文字の場合は非表示。

**人口動態インジケーター（`populationForecast`）**:

`populationForecast` prop が存在する場合のみ表示（都市計画リスクパネルの直後）。`GET /api/population-forecast` のレスポンス。

| `trend` | 枠色 | 背景色 |
|---------|------|--------|
| `"増加"` | 緑 | 緑薄 |
| `"現状維持"` | 黄 | 黄薄 |
| `"緩やかな減少"` | オレンジ | オレンジ薄 |
| `"急激な減少"` | 赤 | 赤薄 |

2020/2030/2040/2050年の推計人口を4カラムグリッドで表示。2020年以外は2020年比の増減率（%）を色付きで併記。右上にトレンドラベルを表示。フッターに30年後変化率・推定空室率増加幅を記載。

**都市計画リスクパネル（`stats.urbanRisks`）**:

`stats.urbanRisks` が1件以上ある場合のみ表示。レベル別カラーリング（`RISK_STYLE` マップ）でカード一覧を描画。

| レベル | 色 | アイコン |
|--------|-----|---------|
| `ERROR` | 赤 | `ShieldAlert` |
| `WARNING` | 黄 | `AlertTriangle` |
| `INFO` | 青 | `ShieldCheck` |

`key={risk.code}` で安定したキーを使用。

---

## YieldAnalysis

`frontend/src/components/YieldAnalysis.tsx`

**props**:
- `result: InvestmentResult`
- `input: InvestmentInput`
- `populationForecast?: PopulationForecastResult | null`

**ゲージ設計**:
```typescript
const MAX_YIELD_PCT = 16  // 上限（8%が中央に来る設計）
const TARGET_PCT = 8

gaugePosition = Math.min(yieldPct / MAX_YIELD_PCT, 1) * 100  // 現在値マーカー位置
targetPosition = (TARGET_PCT / MAX_YIELD_PCT) * 100           // = 50%（常に中央）
```

グラデーション: 赤（0%）→ 黄（8%）→ 緑（16%+）

**8%未達時（`!isAbove8Percent`）の表示**:
- `requiredCostReduction`: 「土地または建築費いずれか一方を削減すべき額」
- `requiredMonthlyRent`: 「必要な月額賃料（満室想定）」

**8%超え時の表示**:
- `(grossYield - 0.08)`: 目標に対する余裕度（%表示）
- 「賃料が何%下落すると8%を下回るか」も表示: `(grossYield - 0.08) / grossYield`

**空室シナリオ別 年間賃料収入カード（`VacancyScenarioCard`）**:

バックエンドの `result.yieldScenarios`（`POST /api/investment/analyze` レスポンス）を受け取り表示する。フロントでの再計算は行わない。

| シナリオ | 想定空室率 | 表示内容 |
|---------|-----------|---------|
| 楽観 | `vacancyRate × 0.5` | `yieldScenarios.optimistic.annualRent` / `grossYield` |
| 標準 | `vacancyRate × 1.0` | `yieldScenarios.standard.annualRent` / `grossYield` |
| 悲観 | `vacancyRate × 1.5` | `yieldScenarios.pessimistic.annualRent` / `grossYield` |

`result.yieldScenarios` が存在する場合のみカードをレンダリングする（`null`チェック済み）。表面利回り（`grossYield`）は全シナリオ共通値（満室想定）。

**人口減少シナリオ（`populationForecast`）**:

`populationForecast` prop が存在する場合のみシナリオテーブル直下に表示。

| 表示項目 | 計算式 |
|---------|--------|
| 想定空室率 | `min(actualV + vacancyRateDelta, 0.99)` |
| 表面利回り | `grossYield × (1 - popV)` |
| 実質利回り | `grossYield × (1 - popV) × (1 - expenseRate)` |
| 年間CF（概算） | `grossYield × totalInvestment × (1 - popV) − loanPayment − expenses` |

CF がマイナスの場合は「赤字転落 ⚠️」バッジを表示。フッターに「30年後人口推計: XX%（現在比）／トレンド: ○○」を記載。

---

## PDF生成（generatePdf）

`frontend/src/lib/generatePdf.ts`

> テストカバレッジ: `src/lib/__tests__/generatePdf.test.ts`（Vitest）— 正常生成・空ストレステスト・取得コストなし・新築・XSSサニタイズ・フォント取得失敗・ファイル名・メタデータ・ヘッダー/フッター の11ケースを検証。

**概要**: `pdfmake ^0.3.7` を使い投資分析結果を PDF ドキュメントとして生成する非同期関数。`Dashboard.tsx` のボタンクリックから `downloadReportPDF(input, result)` を呼び出すとブラウザのダウンロード機能で自動保存される。日本語対応のため Noto Sans JP（TTF）を `/public/fonts/` から `fetch` して `pdfMake.virtualfs` に登録する（pdfmake は woff2 非対応のため TTF を使用）。

**エントリポイント**:

```typescript
export async function downloadReportPDF(
  input: InvestmentInput,
  result: InvestmentResult,
): Promise<void>
```

**ユーティリティモジュール**（`frontend/src/lib/pdf/`）:

| ファイル | 役割 |
|---------|------|
| `format.ts` | `fmtYen`（億/百万/円の3段階・Math.round済み）・`fmtPct`・`fmtDate`・`sanitize`（XSS除去） |
| `verdict.ts` | `calcVerdict()` — PASS/CAUTION/REJECT の総合判定・根拠3点・自動コメント生成 |
| `charts.ts` | `buildCfBarChartSvg`・`buildDeadCrossLineSvg`・`buildCostDonutSvg` — SVG文字列を直接生成（Recharts不使用） |

**PDF構成（5ページ）**:

| ページ | 内容 |
|--------|------|
| 表紙 | 物件概要（土地価格・建物費・築年数・構造・月額賃料・ローン情報）・分析日 |
| P1 投資サマリー | 総合判定バッジ（PASS/CAUTION/REJECT）＋根拠3点・KPI 2行×3列（表面利回り・実質利回り・DSCR基本 / DSCR複合・LTV・出口Equity）・ストレステスト要約・出口戦略テーブル・自動生成コメント |
| P2 10年キャッシュフロー | CFバーチャート（SVG）＋ `YearlyResult[]` テーブル。デッドクロスゾーンの年は赤色バー |
| P3 ストレステスト結果 | デッドクロス折れ線チャート（SVG）＋ 6シナリオ表。`stressScenarios.length > 0` の場合のみ表示 |
| P4 取得コスト内訳 | コストドーナツチャート（SVG）＋ 初期投資・`AcquisitionCostBreakdown` 明細・1年目年間経費内訳 |

**ヘッダー/フッター**:
- 表紙（1ページ目）はヘッダーなし。2ページ目以降は「yield-guard 不動産投資分析レポート」と分析日をヘッダーに表示
- 全ページフッター: 免責文 ＋ 「現在ページ / 総ページ数」
- PDFメタデータ（`info`）: `title`・`author`・`subject`・`creator` を設定

**ファイル名**: `yield-guard-report-YYYYMMDD.pdf`（`downloadReportPDF` 内で自動生成）

---

## CostBreakdown

`frontend/src/components/CostBreakdown.tsx`

> テストカバレッジ: `CostBreakdown.test.tsx`（Vitest + RTL）— 初期投資内訳・取得時諸経費明細・年間費用（1年目）の表示を検証。

**props**:
- `input: InvestmentInput`
- `acquisitionCosts: AcquisitionCostBreakdown`
- `yearlyResults: YearlyResult[]`

**表示セクション**:

1. **初期投資内訳**（ドーナツグラフ + 凡例）  
   土地・建物・仲介手数料・印紙税・登録免許税・不動産取得税・固定資産税日割り（> 0 の項目のみ）  
   各項目は割合（%）と金額を表示。

2. **取得時諸経費明細テーブル**  
   `acquisitionCosts` の各費用と合計。`propertyTaxProration === 0` の行は非表示。

3. **年間費用内訳（1年目）**（ドーナツグラフ + 凡例）  
   `yearlyResults[0]` から: ローン返済・運営経費・所得税（> 0 の項目のみ）

**`fmt` ヘルパー**:
```typescript
n >= 10_000_000 → `${(n / 10_000_000).toFixed(1)}千万円`
otherwise       → `${Math.round(n / 10_000).toLocaleString()}万円`
```

**カラーパレット**: `COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4", "#f97316"]`

---

## CashFlowChart

`frontend/src/components/CashFlowChart.tsx`

**props**:
- `result: InvestmentResult`
- `equityInvested: number` — 自己資金（総投資額 - ローン金額）

**データ加工（35年分）**:
```typescript
data = yearlyResults.slice(0, 35).map(y => ({
  year: `${y.year}年`,
  税引後CF: round(y.afterTaxCashFlow / 10_000),  // 万円単位
  累積CF: round((y.cumulativeCashFlow - equityInvested) / 10_000),  // 自己資金を初期コストとして控除
  isDeadCrossZone: y.isInDeadCrossZone,
}))
```

`cumulativeCashFlow - equityInvested` の意味: 「自己資金をゼロ時点として、
累積CFが自己資金を回収した時点からプラスになる」グラフ。

**breakEvenYear（投資回収年）**:
```typescript
const breakEvenYear = yearlyResults.find(
  y => y.cumulativeCashFlow - equityInvested >= 0
)?.year ?? null
```

**グラフ仕様**:
- 左軸: 税引後CF（棒グラフ）
- 右軸: 累積CF（折れ線グラフ・黄色）
- デッドクロスゾーンの棒: `#fca5a5`（赤）、通常: `#60a5fa`（青）
- 回収年の縦線: `#22c55e`（緑）

**出口戦略サマリー**（グラフ下部）:
- `exitSalePrice`: 売却価格（NOI基準）
- `exitNetProceeds`: 売却手取り
- `exitTotalEquity`: 最終手残り（プラス: 緑, マイナス: 赤）

**IRR / NPV サマリーカード**: 出口サマリーグリッドの下に 2 列カードを追加。
- `irr` が `null` → "―"（灰色）表示
- `irr` 正値 → 緑、負値 → 赤
- `npv` 正値 → 緑、負値 → 赤（`formatMan` でフォーマット）

---

## DeadCrossChart

`frontend/src/components/DeadCrossChart.tsx`

**props**:
- `result: InvestmentResult`

**データ加工（35年分）**:
```typescript
data = yearlyResults.slice(0, 35).map(y => ({
  year: `${y.year}年`,
  元金返済: round(y.annualPrincipal / 10_000),
  減価償却費: round(y.annualDepreciation / 10_000),
  isDeadCrossZone: y.isInDeadCrossZone,
}))
```

**deadCrossEndYear**:
```typescript
const deadCrossEndYear = yearlyResults.slice(0, 35)
  .findLast(y => y.isInDeadCrossZone)?.year ?? deadCrossYear
```
`isInDeadCrossZone` が true の最後の年。ローン完済後（元金返済ゼロ）で脱出。

**グラフ仕様**:
- デッドクロスゾーン全体を `ReferenceArea` でハイライト（`fill: "#fee2e2"`）
- 元金返済ライン: 赤（`#ef4444`）
- 減価償却費ライン: 青の破線（`#3b82f6`, `strokeDasharray="5 5"`）
- デッドクロス開始年の縦線: オレンジ（`#f97316`）

---

## MonteCarloChart

`frontend/src/components/MonteCarloChart.tsx`

**props**:
- `result: MonteCarloResult`

**表示内容**:
- サマリーバッジ（3つ）: IRR正値達成率 / デッドクロス発生率 / IRR中央値。閾値（達成率≥50%・発生率<30%・中央値≥0%）で緑/赤を切り替え
- IRR 分布ヒストグラム（20ビン棒グラフ）: IRR≥0 → 青（`#60a5fa`）、IRR<0 → 赤（`#fca5a5`）。`irrHistogram` が `null` の場合はフォールバック文言を表示
- 最終純資産分布ヒストグラム（20ビン棒グラフ）: 純資産≥0 → 緑（`#34d399`）、<0 → 赤（`#fca5a5`）
- パーセンタイル表: P10（悲観）〜P90（楽観）の IRR・最終純資産を表示

**表示タイミング**:
- `full` モードかつ `POST /api/investment/analyze` 実行後に「モンテカルロ実行（1,000試行）」ボタンが出現
- ボタン押下で `POST /api/investment/simulate` を呼び出し、レスポンス受信後に本コンポーネントが描画される
- `handleAnalyze` / `handleLoanMethodChange` が再実行されると `monteCarloResult` がリセットされ非表示に戻る

**フォーマット**:
- `formatPct(v, 1)` / `formatMan(v, 0)` を `lib/utils` から流用

---

## APIクライアント（`lib/api.ts`）

| 関数 | エンドポイント | 説明 |
|------|--------------|------|
| `fetchLandPrices(params)` | `GET /api/land-prices/stats` | 土地取引統計 |
| `compareLandPrice(params)` | `GET /api/land-prices/compare` | 相場比較 |
| `estimateLandPrice(params)` | `GET /api/land-prices/estimate` | 理論価格推定（築年数・駅距離・需要スコア補正） |
| `fetchStationRidership({lat, lng, z?})` | `GET /api/station-ridership` | 物件緯度経度から周辺駅の乗降客数・需要スコア（XKT015） |
| `fetchPopulationForecast({lat, lng, z?})` | `GET /api/population-forecast` | 物件緯度経度から将来推計人口・人口減少シナリオ（XKT013） |
| `analyze(input)` | `POST /api/investment/analyze` | 投資シミュレーション |
| `simulate(input)` | `POST /api/investment/simulate` | モンテカルロ・シミュレーション |
| `fetchMunicipalities(area)` | `GET /api/municipalities` | 市区町村一覧（XIT002） |
| `fetchPrefectures()` | `GET /api/prefectures` | 都道府県一覧 |

**共通エラーハンドリング（`handleResponse`）**:
```typescript
async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error ?? "APIエラーが発生しました")
  }
  return res.json() as Promise<T>
}
```

---

## 型定義のポイント（`types/investment.ts`）

**`DEFAULT_INPUT`**:
```typescript
export const DEFAULT_INPUT: InvestmentInput = {
  landPrice: 5_000_000,      // 500万円
  buildingCost: 10_000_000,  // 1000万円
  monthlyRent: 120_000,      // 12万円
  loanAmount: 13_000_000,    // 1300万円（自己資金 = 16050000 - 13000000 = 3050000）
  annualLoanRate: 0.015,     // 1.5%
  loanYears: 35,
  buildingType: "木造",
  expenseRate: 0.20,
  incomeTaxRate: 0.33,
  holdingYears: 10,
  exitYieldTarget: 0.06,
  vacancyRate: 0.05,
  actualVacancyRate: 0,  // 未入力扱い（シナリオ比較では vacancyRate にフォールバック）
  miscExpenseRate: 0.07,
  vacancyRateDelta: 0,
  loanRateDelta: 0,
}
```

**`BUILDING_USEFUL_LIFE`**: バックエンドの `UsefulLife()` と対応するフロントエンド側の参照用マップ。
計算には使用せず、フォームの表示説明（「法定耐用年数: XX年」）に使用。

---

## RenovationPanel

`frontend/src/components/RenovationPanel.tsx`

### 責務

修繕費回収期間シミュレーションのフォーム入力・API呼び出し・結果表示を自己完結で担う。
`Dashboard` の投資シミュレーション `result` に依存せず、独立した状態を持つ。
`POST /api/renovation/analyze` を呼び出す。

### 表示構成

1. **グローバル入力**（5フィールド）: 物件取得価格・年間家賃・年間経費・実効税率・セルフリフォーム時給
2. **工事項目テーブル**（動的行）: 部位名・工事費(万円)・月額賃料アップ・セルフ toggle（ON時に工数欄を表示）・削除ボタン
3. **「リフォーム分析を実行」ボタン**: items が空または cost ≤ 0 の行があるとフロント側でエラーメッセージを表示
4. **結果セクション**（`isRecoverable` が true の場合のみ回収タイムラインを表示）:
   - 4列サマリーカード: 修繕費回収期間 / 節税効果 / 仮想人件費 / 実質利回り
   - 工事分類テーブル: 各項目を「資本的支出（amber）」または「修繕費（blue）」バッジで表示
   - 回収タイムラインチャート（Recharts `LineChart`）: 累積賃料増加額（青線）vs リフォーム費用（赤点線）、回収年に緑の `ReferenceLine`。X 軸は最大 50 年でキャップ

### 主な実装詳細

- `recoveryChartData` は `useMemo` でメモ化（`result` 変更時のみ再計算）
- `updateItem` でセルフ toggle OFF 時に `selfLaborHours = 0` にリセット（仮想人件費の誤計上を防止）
- テーブル内の入力欄は共通クラス変数 `cellInput` で統一
- `Dashboard.tsx` では投資シミュレーション結果セクションの末尾に `<RenovationPanel />` を常時表示

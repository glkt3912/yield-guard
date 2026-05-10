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
        ├── RenovationPanel        (自己完結・独立状態管理)
        ├── WatchlistPanel         (自己完結・localStorage 永続化)
        └── (PDF出力はコンポーネントではなく downloadReportPDF() で処理)
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

`result && lastInput` が両方存在する場合のみヘッダーにボタンを表示。クリックで `downloadReportPDF(lastInput, result)` を呼び出し、`pdfmake` がクライアント側でPDFを生成してダウンロードする。ファイル名: `yield-guard-report-YYYYMMDD.pdf`。

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
| 取得失敗 | 「（全市区町村）」のみ + ドロップダウン直下に `muniError` を赤文字表示 |
| データなし | 「（全市区町村）」のみ |
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

**Fullモードドラフト自動保存**（PR #477 で追加）:

Fullモードの入力内容を `localStorage` にデバウンス保存し、次回アクセス時に復元できる。

| 項目 | 仕様 |
|------|------|
| localStorage キー | `yield-guard:full-draft` |
| 保存タイミング | `input` または `isQuick` が変化してから 500ms 後（デバウンス） |
| 保存対象 | Fullモード時のみ。Quickモードでは保存しない |
| 復元 | マウント時にドラフトが存在すれば `pendingDraft` にセットし、フォーム上部に黄色バナーを表示 |
| バナー操作 | 「復元する」→ `input` にマージして適用 / 「破棄する」→ ドラフト削除 |
| 自動クリア | シミュレーション実行時（`handleAnalyze` Fullモード分岐）にドラフトを削除 |
| 安全性 | `JSON.parse` 後に `rateAdjustmentSchedule`・`capexSchedule` の `Array.isArray` チェックを実施し、破損データによるランタイムエラーを防止 |

Quickモードの既存履歴（`yield-guard:quick-history`）とはキーが分離されており、相互に影響しない。

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
- `landPriceStats?: LandPriceStats | null` — 地価データ（未取得時は表示なし）

**市場実勢利回りベンチマーク**（PR #478 で追加）:

`landPriceStats` が有効な場合（`medianTsubo > 0` かつ `monthlyRent > 0`）、利回りゲージ直下に市場実勢利回りと判定バッジを表示する。

```typescript
// 実際の入力値から利回りを計算してエリア相場と比較
const userYield = (input.monthlyRent * 12) / (input.landPrice + input.buildingCost)
// calcYieldBenchmark に渡してエリア市場利回りと比較
```

`calcYieldBenchmark`（`frontend/src/lib/yieldBenchmark.ts`）が返す `judgment` に応じてバッジを表示:

| `judgment` | バッジラベル | バッジ variant | 意味 |
|-----------|------------|--------------|------|
| `"realistic"` | 現実的 | `secondary`（グレー） | 実利回り ÷ 市場利回り ≦ 1.0 |
| `"slightly-high"` | やや高め | `outline`（枠線） | 比率 1.0〜1.2。賃料設定の根拠要確認 |
| `"high"` | 大幅に高め | `destructive`（赤） | 比率 > 1.2。エリア相場を大幅に上回る |

バッジの `title` 属性に `judgmentLabel`（詳細説明文）を設定しており、ホバーで確認可能。

`ResultsSection.tsx` が `comparison?.stats ?? null` を `landPriceStats` prop として渡す。地価データ未取得時は `null` が渡り、ベンチマークブロックは非表示。

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

> テストカバレッジ: `src/lib/__tests__/generatePdf.test.ts`（Vitest）— 正常生成・空ストレステスト・取得コストなし・新築・XSSサニタイズ・フォント取得失敗・ファイル名・メタデータ・ヘッダー/フッター の11ケースを検証。`src/lib/__tests__/pdfFormat.test.ts`（21件）

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
| `format.ts` | `fmtYen`（1万円未満は円・1万円以上は万円・1億円以上は億円、万円未満は四捨五入）・`fmtPct`・`fmtDate`・`sanitize`（XSS除去） |
| `verdict.ts` | `calcVerdict()` — PASS/CAUTION/REJECT の総合判定・根拠3点・自動コメント生成 |
| `charts.ts` | `buildCfBarChartSvg`・`buildDeadCrossLineSvg`・`buildCostDonutSvg` — SVG文字列を直接生成（Recharts不使用） |

**PDF構成（5ページ）**:

| ページ | 内容 |
|--------|------|
| 表紙 | 物件概要（土地価格・建物費・築年数・構造・月額賃料・ローン情報）・分析日 |
| P1 投資サマリー | 総合判定バッジ（PASS/CAUTION/REJECT）＋根拠3点・KPI 2行×3列（表面利回り・実質利回り・DSCR基本 / DSCR複合・LTV・出口収益）・ストレステスト要約・出口戦略テーブル・自動生成コメント |
| P2 10年キャッシュフロー | CFバーチャート（SVG）＋ `YearlyResult[]` テーブル。デッドクロスゾーンの年は赤色バー |
| P3 ストレステスト結果 | デッドクロス折れ線チャート（SVG）＋ 6シナリオ表。`stressScenarios.length > 0` の場合のみ表示 |
| P4 取得コスト内訳 | コストドーナツチャート（SVG）＋ 初期投資・`AcquisitionCostBreakdown` 明細・1年目年間経費内訳 |

**ヘッダー/フッター**:
- 表紙（1ページ目）はヘッダーなし。2ページ目以降は「yield-guard 不動産投資分析レポート」と分析日をヘッダーに表示
- 全ページフッター: 免責文 ＋ 「現在ページ / 総ページ数」
- PDFメタデータ（`info`）: `title`・`author`・`subject`・`creator` を設定

**ファイル名**: `yield-guard-report-YYYYMMDD.pdf`（`downloadReportPDF` 内で自動生成）

### `fmtYen()` 金額フォーマットルール

| 金額範囲 | 表示形式 | 例 |
|---|---|---|
| 1万円未満 | 円（カンマ区切り） | `9,999円` |
| 1万円以上〜1億円未満 | 万円（万円未満は四捨五入） | `88万円`、`9500万円` |
| 万換算で10,000万以上になる場合 | 億円に繰り上げ | `99,999,999円 → 1.0億円` |
| 1億円以上 | 億円（小数1桁） | `1.5億円` |

### フォントサブセット管理

`public/fonts/NotoSansJP-{Regular,Bold}.ttf` はPDF出力で使われる全文字を網羅したサブセット（各 ≈265 KB）。  
PDF文言に新しい漢字を追加した場合は `NEEDED_TEXT` に追記の上、再生成スクリプトを実行する:

```bash
cd frontend
python3 scripts/regenerate-pdf-fonts.py
```

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
- 左軸（`yAxisId="left"`）: 税引後CF（棒グラフ）
- 右軸（`yAxisId="right"`）: 累積CF（折れ線グラフ・黄色 `#f59e0b`）
- DSCR軸（`yAxisId="dscr"`）: 右外側に追加された第3軸。domain `[0, 3]`、紫色 `#8b5cf6`
- デッドクロスゾーンの棒: `#fca5a5`（赤）、通常: `#60a5fa`（青）
- 回収年の縦線: `#22c55e`（緑）

**DSCR年次推移ライン**（PR #475 で追加）:

年次 DSCR を `yAxisId="dscr"` の折れ線（紫・`#8b5cf6`）で重ね描画する。

```typescript
// YearlyResult に dscr フィールドがないため、フロントで計算
// capex は NOI から除外（バックエンドの calcDSCR と同定義: NOI = annualRent - annualExpenses）
DSCR: y.annualLoanPayment > 0
  ? Math.round(((y.annualRent - y.annualExpenses) / y.annualLoanPayment) * 100) / 100
  : null  // 無借入年は null → connectNulls でライン断絶
```

参照ライン:

| ライン | y値 | 色 | 意味 |
|--------|-----|-----|------|
| 危険ライン | 1.0 | `#ef4444`（赤） | NOI = 返済額。余裕ゼロ |
| 安全ライン | 1.2 | `#22c55e`（緑） | 業界慣行の最低安全基準 |

Tooltip: DSCR は `"X.XX"` 形式（単位なし）で表示。CF 系は `"X万円"` 形式と別フォーマット。

モバイルでは `<details>` 要素で説明文を初期折りたたみ表示。

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

## InvestmentScoreHeatmap

`frontend/src/components/InvestmentScoreHeatmap.tsx`

エリア別の投資適地スコアをLeaflet地図上にヒートマップ（色付き矩形オーバーレイ）で表示するコンポーネント。

**表示条件**: `Dashboard` 内で `propertyLat !== undefined` の場合のみレンダリング（物件の緯度経度取得後に表示）。

**props**:
- `centerLat?: number` — 地図の初期中心緯度（省略時: 35.6812 = 東京）
- `centerLng?: number` — 地図の初期中心経度（省略時: 139.7671 = 東京）
- `onTileSelect?: (lat: number, lng: number) => void` — タイルクリック時のコールバック。渡すと各タイルがクリック可能になり、クリックで `centerLat/centerLng` を親に通知する

**インポート方法**（SSR回避のため dynamic import 必須）:
```tsx
const InvestmentScoreHeatmap = dynamic(
  () => import("./InvestmentScoreHeatmap"),
  { ssr: false }
);
```

**操作フロー**:
1. 地図（OpenStreetMap）をパン・ズームして分析したいエリアに移動
2. 右上の「このエリアを分析」ボタンをクリック
3. `map.getBounds()` で現在の表示範囲を取得し `GET /api/investment-score-heatmap` を呼び出す
4. 各タイルをスコアに応じた色の矩形で描画。ホバーでグレード・スコアを表示
5. `onTileSelect` が渡されている場合はタイルをクリックするとその中心座標を親コンポーネントへ通知（`InvestmentForm` の緯度経度欄に自動入力）

**ズームとタイル数の制御**:

送信する `z` は `Math.min(map.getZoom(), 15)` でキャップ。バックエンドでタイル上限が z≤14: 50、z=15: 25 に設定されている。

**ヘッダー注記**:

タイトル直下に「スコアは需要・安全性の評価です。表面利回りとは別軸の指標です。」を常時表示。スコアと利回りの混同を防ぐため。

**スコア配色**:

| スコア | 色 | グレード |
|--------|-----|---------|
| 80 以上 | `#22c55e`（緑） | 優良 |
| 65〜79 | `#86efac`（薄緑） | 良好 |
| 50〜64 | `#fde68a`（黄） | 普通 |
| 35〜49 | `#fdba74`（橙） | 注意 |
| 34 以下 | `#f87171`（赤） | 要注意 |

矩形の `fillOpacity: 0.4`、`weight: 0.5` で地図タイルが透けて見える設計。

**内部コンポーネント**:

`AnalyzeButton` — `useMapEvents` フックを使い Leaflet コンテキスト内でボタンを実装（`MapContainer` の外側には配置できないため内部コンポーネントに分離）。

**`tileBounds(x, y, z)`**:

タイル番号から地図上の矩形の4隅を計算するユーティリティ。`centerLat/centerLng`（タイル中心）ではなく、端点から矩形を描くため独自に計算。

**Leaflet CSS**:

`import "leaflet/dist/leaflet.css"` は `src/app/globals.css` に一元管理（`@import "leaflet/dist/leaflet.css"`）。コンポーネント内でのインポートは不要。

---

## APIクライアント（`lib/api.ts`）

| 関数 | エンドポイント | 説明 |
|------|--------------|------|
| `fetchLandPrices(params)` | `GET /api/land-prices/stats` | 土地取引統計 |
| `compareLandPrice(params)` | `GET /api/land-prices/compare` | 相場比較 |
| `estimateLandPrice(params)` | `GET /api/land-prices/estimate` | 理論価格推定（築年数・駅距離・需要スコア補正） |
| `fetchStationRidership({lat, lng, z?})` | `GET /api/station-ridership` | 物件緯度経度から周辺駅の乗降客数・需要スコア（XKT015） |
| `fetchPopulationForecast({lat, lng, z?})` | `GET /api/population-forecast` | 物件緯度経度から将来推計人口・人口減少シナリオ（XKT013） |
| `fetchInvestmentScore({lat, lng})` | `GET /api/investment-score` | 投資適地スコア（1点） |
| `fetchInvestmentScoreHeatmap({minLat, maxLat, minLng, maxLng, z})` | `GET /api/investment-score-heatmap` | viewport 内全タイルの投資スコア一括取得 |
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

## WatchlistPanel

`frontend/src/components/WatchlistPanel.tsx`

### 責務

物件候補を手動で登録し、`localStorage`（キー: `yg_watchlist`）に永続化するウォッチリスト機能を提供する。バックエンドAPIへの依存なし。`Dashboard` の `RenovationPanel` 直下に常時表示。

### データ構造

```typescript
// types/investment.ts
export type WatchlistStatus = "検討中" | "見送り" | "購入済み";

export interface WatchlistItem {
  id: string;        // crypto.randomUUID() または Date.now() で生成
  name: string;      // 物件名（必須）
  memo: string;      // メモ（任意）
  status: WatchlistStatus;
  addedAt: string;   // ISO 8601
}
```

### 機能

- 物件名 + メモ入力・追加ボタン（空白は追加不可、Enter キー対応）
- ステータス変更ドロップダウン（検討中 / 見送り / 購入済み）
- 削除ボタン（`Trash2` アイコン）
- ステータスバッジの色分け: 検討中=青・見送り=グレー・購入済み=緑

### localStorage 永続化

```typescript
// lazy initializer で初期値を直接ロード（saveItems([]) による消去バグを回避）
const [items, setItems] = useState<WatchlistItem[]>(loadItems);

useEffect(() => {
  saveItems(items);
}, [items]);
```

`useState(loadItems)` の lazy initializer を使うことで、初回レンダリング時に `saveItems([])` が誤って発火してデータを消去するバグを防ぐ。`loadItems` は SSR 安全のため `typeof window === "undefined"` チェックを内包している。

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

---

## Toast / Modal UI プリミティブ

`frontend/src/components/ui/toast.tsx` / `frontend/src/components/ui/modal.tsx`

PR #424 で追加。PR #476 でユニットテストを追加。

### Toast

アプリ全体に通知を表示する Context ベースのトーストシステム。`layout.tsx` の `<ToastProvider>` がアプリルートを包んでおり、任意のコンポーネントから `useToast()` で呼び出せる。

```typescript
const { toast } = useToast();
toast({ message: "保存しました", variant: "success" });
```

| prop | 型 | 説明 |
|------|----|------|
| `message` | `string` | 表示するメッセージ |
| `variant` | `"success" \| "warning" \| "danger"` | 色・アイコンを決定 |

**挙動**:
- `role="alert"` の要素として DOM に追加される（スクリーンリーダー対応）
- 4000ms 後に自動消去（`setTimeout` で管理）
- X ボタンクリックで即時消去
- 複数トーストは `fixed top-4 right-4` に縦積みで表示
- `useToast()` を `ToastProvider` 外で呼ぶと `Error` をスロー

**テストカバレッジ** (`__tests__/toast.test.tsx`): 表示・自動消去（`vi.useFakeTimers`）・即時消去・複数表示・Provider外エラーの5ケース

### Modal

アクセシビリティ対応のモーダルダイアログ。

```tsx
<Modal open={isOpen} onClose={() => setIsOpen(false)} title="タイトル">
  <p>コンテンツ</p>
</Modal>
```

| prop | 型 | 説明 |
|------|----|------|
| `open` | `boolean` | `false` のとき `null` を返す（DOM に追加されない） |
| `onClose` | `() => void` | 閉じる操作時のコールバック |
| `title` | `string?` | ヘッダータイトル（省略時はヘッダー・閉じるボタン非表示） |

**挙動**:
- `open=true` で `role="dialog" aria-modal="true"` の要素をレンダリング
- ESC キーで `onClose` を呼び出し（`document.addEventListener("keydown", ...)` でグローバル監視）
- バックドロップ（`aria-hidden="true"`）クリックで `onClose`
- フォーカストラップ: Tab / Shift+Tab でパネル内フォーカスがループ
- `open=true` 時に `document.body.style.overflow = "hidden"`（スクロールロック）
- 複数モーダル同時対応: `document.body.dataset.modalCount` で参照カウントを管理し、最後のモーダルが閉じるまでスクロールロックを維持

**テストカバレッジ** (`__tests__/modal.test.tsx`): 表示/非表示・ESCキー・バックドロップ・スクロールロック・Tab/Shift+Tabフォーカストラップの9ケース


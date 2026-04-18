# フロントエンドコンポーネント仕様

`frontend/src/components/` 配下。フレームワーク: Next.js 14 (App Router)

---

## 状態フローの概要

```
page.tsx
  └── Dashboard
        ├── InvestmentForm     (入力 → onAnalyze, onFetchLandPrices)
        ├── LandPriceAnalysis  (comparison を受け取り表示)
        ├── YieldAnalysis      (result を受け取り表示)
        ├── CostBreakdown      (result.acquisitionCosts + yearlyResults を受け取り表示)
        ├── CashFlowChart      (result + equityInvested を受け取り表示)
        └── DeadCrossChart     (result を受け取り表示)
```

`Dashboard` が `result: InvestmentResult | null` と `comparison: LandPriceComparison | null` を管理する。

---

## Dashboard

`frontend/src/components/Dashboard.tsx`

**状態管理**:
- `result`: `InvestmentResult | null`
- `comparison`: `LandPriceComparison | null`
- `loading`, `error`: ローディング・エラー状態

**`equityInvested` の計算**:
```typescript
const equityInvested = result.totalInvestment - input.loanAmount
```
自己資金（頭金 + 諸経費）。`CashFlowChart` に渡して投資回収年計算に使用。

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

**詳細設定トグル（showAdvanced）**:
`expenseRate`, `incomeTaxRate`, `buildingAge`, `buildingType`, `exitYieldTarget` は
詳細設定パネルに格納されており、デフォルトでは非表示。

**クライアントサイドバリデーション（`validate`）**:
「シミュレーション実行」押下時に `validate()` を実行し、エラーがあれば API を呼ばずフィールド直下にエラーメッセージを表示する。
検証ルールはバックエンドの `validateInvestmentInput()` と同一。フィールドの値を変更するとそのフィールドのエラーをクリアする。

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

**ストレステストスライダー**:
- `vacancyRateDelta`: 0〜30%（空室率の追加シナリオ）
- `loanRateDelta`: 0〜3%（金利上昇シナリオ）
- `InvestmentInput.vacancyRateDelta`, `loanRateDelta` にそのまま渡す

---

## LandPriceAnalysis

`frontend/src/components/LandPriceAnalysis.tsx`

**props**:
- `comparison: LandPriceComparison`

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

**用途地域情報パネル（`stats.zoning`）**:

`stats.zoning` が存在する場合のみ青いパネルで表示。取引データの最頻値から抽出した参考情報。

| フィールド | 表示ラベル |
|-----------|----------|
| `cityPlanning` | 用途地域 |
| `buildingCoverage` | 建ぺい率 |
| `floorAreaRatio` | 容積率 |

各フィールドは個別に `&&` で存在チェックし、空文字の場合は非表示。

---

## YieldAnalysis

`frontend/src/components/YieldAnalysis.tsx`

**props**:
- `result: InvestmentResult`
- `input: InvestmentInput`

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

**空室率シナリオ比較テーブル**:

フロントエンドのみで3シナリオを計算して表示する（バックエンド呼び出しなし）。

| シナリオ | 空室率 | 計算式 |
|---------|--------|--------|
| 満室想定 | 0% | `grossYield × (1 - expenseRate)` |
| 現況 | `actualVacancyRate`（0の場合は`vacancyRate`にフォールバック） | `grossYield × (1 - actualV) × (1 - expenseRate)` |
| ストレス | `actualV + vacancyRateDelta`（上限99%） | `grossYield × (1 - stressV) × (1 - expenseRate)` |

表面利回りが8%以上の行は緑、未満は赤で表示。

---

## CostBreakdown

`frontend/src/components/CostBreakdown.tsx`

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

## APIクライアント（`lib/api.ts`）

| 関数 | エンドポイント | 説明 |
|------|--------------|------|
| `fetchLandPrices(params)` | `GET /api/land-prices` | 土地取引統計 |
| `compareLandPrice(params)` | `GET /api/land-prices/compare` | 相場比較 |
| `analyze(input)` | `POST /api/analyze` | 投資シミュレーション |
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

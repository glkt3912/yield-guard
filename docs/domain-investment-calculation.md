# 投資計算ロジック詳細仕様

`backend/internal/domain/investment.go` の `Analyze` 関数が中心。
`backend/internal/domain/types.go` に型定義がある。

## InvestmentInput 全フィールド解説

| フィールド | 型 | 単位 | 説明 | Defaults() での初期値 |
|-----------|-----|------|------|----------------------|
| `LandPrice` | float64 | 円 | 土地取得費 | — |
| `LandArea` | float64 | m² | 土地面積（CompareLandPrice で使用） | — |
| `BuildingCost` | float64 | 円 | 建築費 | — |
| `BuildingAge` | int | 年 | 築年数（0 = 新築） | — |
| `MiscExpenseRate` | float64 | 率 | 諸経費率（0.07 = 7%） | 0.07 |
| `MonthlyRent` | float64 | 円/月 | 満室想定月額賃料 | — |
| `VacancyRate` | float64 | 率 | 空室率（0.05 = 5%） | — |
| `LoanAmount` | float64 | 円 | ローン金額 | — |
| `AnnualLoanRate` | float64 | 率 | 年利（0.015 = 1.5%） | — |
| `LoanYears` | int | 年 | ローン期間 | 35 |
| `BuildingType` | BuildingType | — | 建物構造 | "木造" |
| `ExpenseRate` | float64 | 率 | 運営経費率（管理・修繕・固定資産税等。ローン利息は含まない） | — |
| `IncomeTaxRate` | float64 | 率 | 実効所得税率（給与との合算後） | — |
| `HoldingYears` | int | 年 | 出口戦略: 売却年数 | 10 |
| `ExitYieldTarget` | float64 | 率 | 売却時目標利回り（NOI / 売却価格） | 0.06 |
| `VacancyRateDelta` | float64 | 率 | ストレステスト用 空室率上昇分 | — |
| `LoanRateDelta` | float64 | 率 | ストレステスト用 金利上昇分 | — |

**注意**: `VacancyRate`・`ExpenseRate`・`IncomeTaxRate` は 0 が有効値のため `Defaults()` では初期化されない。呼び出し側で必ず指定する。

---

## ストレステスト入力の適用タイミング

```go
effectiveVacancy := input.VacancyRate + input.VacancyRateDelta
effectiveRate    := input.AnnualLoanRate + input.LoanRateDelta
```

`Analyze` の先頭で加算し、以降の全計算に `effectiveVacancy` / `effectiveRate` を使用する。
元の `VacancyRate` / `AnnualLoanRate` は変更しない。

---

## 総投資額の計算

```
諸経費        = (LandPrice + BuildingCost) × MiscExpenseRate
総投資額       = LandPrice + BuildingCost + 諸経費
```

`miscExpenses` は後で取得費の計算（`calcExit`）にも使われる。

---

## 表面利回り（GrossYield）

```
表面利回り = (MonthlyRent × 12) / 総投資額
```

**空室率を含まない**満室想定年収で計算する。不動産業界の慣習的な指標。

## 実質利回り（NetYield）

```
実効賃料収入  = MonthlyRent × 12 × (1 - effectiveVacancy)
年間運営経費  = 実効賃料収入 × ExpenseRate
実質利回り    = (実効賃料収入 - 年間運営経費) / 総投資額
```

空室と運営経費を控除した実態に近い利回り。

---

## 8%逆算ロジック（`calcRequired8pct`）

8%境界線（`targetYield8pct = 0.08`）を基準に2つの逆算値を返す。

```go
// 目標年収 = 総投資額 × 8%
requiredMonthlyRent = (totalInvestment × 0.08) / 12

// 現賃料で8%達成に必要な総投資額
requiredTotalInvestment = (MonthlyRent × 12) / 0.08

// 過剰投資額（削減が必要な額）
costReduction = max(totalInvestment - requiredTotalInvestment, 0)
```

- `RequiredMonthlyRent`: 現在の投資額で8%を達成するために必要な月額賃料
- `RequiredCostReduction`: 現在の賃料で8%を達成するために土地 **または** 建築費いずれか一方を削減すべき額

---

## 元利均等返済（`calcMonthlyPayment`）

```
月利 r = AnnualLoanRate / 12
返済回数 n = LoanYears × 12

月次返済額 = P × r × (1+r)^n / ((1+r)^n - 1)
```

- `annualRate == 0` のとき: `P / (years × 12)` （元金均等の特殊ケース）
- `principal <= 0` または `years <= 0` のとき: 0を返す

---

## 年次ローン内訳分解（`calcYearlyLoanComponents`）

12ヶ月ループで月次利息・元金を積算する。

```go
for range 12 {
    monthInterest  = remaining × r
    monthPrincipal = monthlyPayment - monthInterest
    // 最終月: 残高 < 月次元金返済 → 残高のみ返済（端数防止）
    if monthPrincipal > remaining { monthPrincipal = remaining }
    ...
}
```

この積算方式により、年度末残高が正確に計算される。

---

## 年次シミュレーション期間の決定

```go
years := max(LoanYears, HoldingYears, 35)
```

| 下限 | 理由 |
|------|------|
| `LoanYears` | ローン完済まで元金返済額を正確に追う |
| `HoldingYears` | 出口売却年が `yearlyResults` の範囲内に収まる |
| `35` | フロントの `CashFlowChart` が35年固定のため最低35年分を保証 |

---

## 各年次ループの計算順序

```
① ローン返済内訳（利息・元金）の計算
② 実効賃料収入・運営経費の計算
③ 当年の減価償却費（耐用年数内のみ）
④ 課税所得 = 収入 - 利息 - 減価償却 - 経費
⑤ 所得税 = 課税所得 × IncomeTaxRate（課税所得 ≤ 0 なら 0）
⑥ キャッシュフロー（税引前）= 収入 - ローン返済 - 経費
⑦ 税引後CF = キャッシュフロー - 所得税
⑧ 累積CF += 税引後CF
⑨ デッドクロス判定
```

---

## YearlyResult 各フィールドの意味

| フィールド | 説明 |
|-----------|------|
| `AnnualRent` | 実効賃料収入（空室・ストレス控除後） |
| `AnnualLoanPayment` | 年間ローン返済額（= monthlyPayment × 12） |
| `AnnualInterest` | 年間利息支払額 |
| `AnnualPrincipal` | 年間元金返済額 |
| `AnnualDepreciation` | 当年の減価償却費（耐用年数超過後は 0） |
| `AnnualExpenses` | 当年の運営経費（= AnnualRent × ExpenseRate） |
| `TaxableIncome` | 課税所得（マイナスは 0 ではなく実値を格納） |
| `IncomeTax` | 所得税（課税所得 ≤ 0 の場合は 0） |
| `CashFlow` | 税引前CF（= 収入 - ローン返済 - 経費） |
| `AfterTaxCashFlow` | 税引後CF（= CashFlow - IncomeTax） |
| `RemainingLoanBalance` | 年度末残高（完済後は 0） |
| `CumulativeCashFlow` | 税引後CF の累積合計 |
| `IsDeadCrossYear` | デッドクロス初年度フラグ（初回のみ true） |
| `IsInDeadCrossZone` | デッドクロス継続中フラグ（ゾーン全体で true） |

**CashFlow と AfterTaxCashFlow の違い**:
- `CashFlow`: 税金を考慮しない「実際のキャッシュの動き」
- `AfterTaxCashFlow`: 税金支払後の手残り（投資判断の本来の指標）

---

## InvestmentResult 各フィールドの意味

| フィールド | 説明 |
|-----------|------|
| `TotalInvestment` | 総投資額（土地 + 建物 + 諸経費） |
| `MiscExpenses` | 諸経費額 |
| `GrossYield` | 表面利回り |
| `NetYield` | 実質利回り |
| `IsAbove8Percent` | 表面利回り ≥ 8% かどうか |
| `RequiredCostReduction` | 8%達成に必要なコスト削減額（いずれか一方） |
| `RequiredMonthlyRent` | 8%達成に必要な月額賃料 |
| `DeadCrossYear` | デッドクロス初年度（-1 = 35年以内なし） |
| `YearlyResults` | 年次結果配列（`max(LoanYears, HoldingYears, 35)` 件） |
| `ExitSalePrice` | 売却価格（NOI / ExitYieldTarget） |
| `ExitCapitalGain` | 譲渡所得 |
| `ExitTransferTax` | 譲渡所得税 |
| `ExitNetProceeds` | 売却手取り（税・残債・売却費控除後） |
| `ExitTotalEquity` | 最終手残り（売却手取り + 累積CF） |

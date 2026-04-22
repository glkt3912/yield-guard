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
| `VacancyRate` | float64 | 率 | 想定空室率・長期シミュレーション用（0.05 = 5%） | — |
| `ActualVacancyRate` | float64 | 率 | 現況空室率・現時点の実態（0 = 未入力扱い。フロントのシナリオ比較のみで使用。バックエンド計算には影響しない） | — |
| `LoanAmount` | float64 | 円 | ローン金額 | — |
| `AnnualLoanRate` | float64 | 率 | 年利（0.015 = 1.5%） | — |
| `LoanYears` | int | 年 | ローン期間 | 35 |
| `LoanMethod` | string | — | 返済方式: `"equal-payment"`（元利均等）or `"equal-principal"`（元金均等） | `"equal-payment"` |
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

### 諸経費率 7% の内訳根拠

デフォルト値 `MiscExpenseRate = 0.07`（7%）の内訳（概算）:

| 費用項目 | 概算率 | 根拠 |
|----------|--------|------|
| 不動産取得税 | 〜1.5% | 地方税法第73条の15（住宅用 税率3%、課税標準の軽減特例あり） |
| 登録免許税（所有権移転） | 〜1.5% | 登録免許税法別表第一（住宅: 0.3%〜2%、軽減措置あり） |
| 仲介手数料 | 〜3% | 宅地建物取引業法第46条（上限: 売買代金×3%+6万円×1.1） |
| 司法書士・調査費等 | 〜1% | 司法書士法報酬基準（自由化後は相場額） |

実際の諸経費率は物件・地域・取引形態によって異なる（5〜10%程度）。

---

## 表面利回り（GrossYield）

```
表面利回り = (MonthlyRent × 12) / 総投資額
```

**空室率を含まない**満室想定年収で計算する。

> **根拠・出典**: 全国宅地建物取引業協会連合会（全宅連）および不動産情報サービス各社（SUUMO・HOME'S 等）が物件掲載時に使用する慣習的指標。物件間の横断比較に用いる。実際の収入は空室や経費分だけ下振れするため、実質利回りと併用して判断する。

## 空室シナリオ別利回り（calcYieldScenarios）

`Analyze()` 内で呼び出され、`InvestmentResult.YieldScenarios` として返される。

```
楽観シナリオ: effectiveVacancy = min(VacancyRate × 0.5, 0.99)
標準シナリオ: effectiveVacancy = min(VacancyRate × 1.0, 0.99)
悲観シナリオ: effectiveVacancy = min(VacancyRate × 1.5, 0.99)

AnnualRent = MonthlyRent × 12 × (1 - effectiveVacancy)
GrossYield = (MonthlyRent × 12) / 総投資額  ← 全シナリオ共通（満室想定）
```

**0.99 キャップの意図**: 悲観×1.5 で空室率が 100% を超える場合（例: 入力 80% × 1.5 = 120%）、年間賃料がゼロ以下になることを防ぐ。`math.Min(..., 0.99)` で最大99%に制限する。

**GrossYield の一定性**: 表面利回りは「満室想定年収 / 総投資額」で定義され、シナリオ間で変化しない。シナリオで変化するのは `AnnualRent`（実効賃料）のみ。

---

## 実質利回り（NetYield）

```
実効賃料収入  = MonthlyRent × 12 × (1 - effectiveVacancy)
年間運営経費  = 実効賃料収入 × ExpenseRate
実質利回り    = (実効賃料収入 - 年間運営経費) / 総投資額
```

空室と運営経費を控除した実態に近い利回り。

> **根拠・出典**: 不動産鑑定評価基準（国土交通省、令和2年改正）における「純収益利回り」に相当。NOI（Net Operating Income）ベースの利回りとも呼ばれ、CCIM（Certified Commercial Investment Member）等の国際的な投資分析でも標準指標として使用される。

---

## 8%逆算ロジック（`calcRequired8pct`）

**8%の根拠**: 不動産投資の実務において、表面利回り8%は「最低限の投資妥当性」を示す経験則的ベンチマーク。

> **根拠・出典**: 日本不動産研究所「不動産投資家調査」（各年次）によると、住宅系投資物件の期待利回り（NOI利回り）は首都圏で5〜6%台、地方圏で6〜8%台。表面利回りはこれより1〜2%高くなるため、8%を「許容できる最低ライン」として業界で広く参照される。本ツールはこの8%をハードコードした警告閾値として採用している。

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

## DSCR算出（`CalcDSCR`）

```
DSCR = NOI / 年間ローン返済額（1年目）
```

- NOI（Net Operating Income）= 年間実効賃料収入 − 年間運営経費（利息・返済は含まない）
- `annualDebtService <= 0` の場合は 0 を返す（ゼロ除算防止）
- `DSCR >= 1.0` のとき債務返済能力あり（安全）、`< 1.0` のとき要注意

> **根拠・出典**: DSCR（Debt Service Coverage Ratio: 借入金償還余裕率）は、金融機関が投資用不動産ローンの審査・モニタリングで使用する標準指標。日本の主要銀行・信用金庫のローン審査基準（一般的に DSCR ≥ 1.25〜1.3 を要求）や、米国 CMBS（商業用不動産担保証券）市場の格付基準でも広く使用される。

---

## 元金均等返済（`CalcEqualPrincipalPayment` / `calcYearlyLoanComponentsEqualPrincipal`）

### 月次返済額（`CalcEqualPrincipalPayment`）

```
月次元金返済額 = P / (LoanYears × 12)
残高           = P − (month − 1) × 月次元金返済額
月次利息       = 残高 × (AnnualLoanRate / 12)
月次返済額     = 月次元金返済額 + 月次利息
```

- 毎月の元金返済額は一定（`P / n`）
- 利息は残高に比例して逓減するため月次返済額は毎月減少する
- 元利均等返済と比べ、初期の月次返済額は高いが総支払利息は少ない
- 返済額の逆転（クロスオーバー）は一般的に35年ローンで**約17年目**前後

### 年次ローン内訳分解（`calcYearlyLoanComponentsEqualPrincipal`）

元金均等返済用の年次積算ヘルパー。12ヶ月ループで月次利息・元金を積算する。

```go
for range 12 {
    if remaining <= 0 { break }
    mp := monthlyPrincipal
    if mp > remaining { mp = remaining }  // 最終年の端数防止
    interest += remaining * r
    principal += mp
    remaining -= mp
}
```

`Analyze()` の年次ループでは各年の期首残高（`remainingBalance`）から `calcYearlyLoanComponentsEqualPrincipal` を呼び出し、年次利息・元金を求める。

---

## LTV 感度分析（`CalcLTVSensitivity`）

LTV（Loan-to-Value: 借入比率）を 50/60/70/80/90% と変化させたときの DSCR・年間CF・CF利回りを試算する。

```
LTV       = 借入額 / 総投資額
borrowing = 総投資額 × LTV
equity    = 総投資額 × (1 − LTV)

元利均等: 月次返済額 = calcMonthlyPayment(borrowing, rate, years)
           annualDebt = 月次返済額 × 12
元金均等: 1年目月次返済額 = CalcEqualPrincipalPayment(borrowing, rate, years×12, 1)
           annualDebt = 1年目月次返済額 × 12  ← 最保守的（最大値）で試算

NOI = 年間実効賃料収入 − 年間運営経費
DSCR = NOI / annualDebt
annualCF = NOI − annualDebt
cfYield  = annualCF / 総投資額
```

**注意**: LTV 感度分析はベースケース（`VacancyRateDelta = 0` / `LoanRateDelta = 0`）で試算する。ストレス適用後の値は `StressScenarios` を参照。元金均等では1年目（最大）の年間返済額を使用するため、実際の返済額より保守的な DSCR になる。

### LTVSensitivityRow フィールド

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `LTV` | float64 | 借入比率（例: 0.70） |
| `Equity` | float64 | 自己資金（円） |
| `LoanAmount` | float64 | 借入額（円） |
| `DSCR` | float64 | 借入金償還余裕率 |
| `AnnualCF` | float64 | 年間キャッシュフロー（円）|
| `CFYield` | float64 | CF利回り（AnnualCF / 総投資額） |

---

## 元利均等返済（`calcMonthlyPayment`）

```
月利 r = AnnualLoanRate / 12
返済回数 n = LoanYears × 12

月次返済額 = P × r × (1+r)^n / ((1+r)^n - 1)
```

- `annualRate == 0` のとき: `P / (years × 12)` （元金均等の特殊ケース）
- `principal <= 0` または `years <= 0` のとき: 0を返す

> **根拠・出典**: 等比数列の和の公式から導出される金融工学の標準公式（Present Value of Annuity の逆算）。住宅ローン計算で広く使用される。日本銀行「金融機関における住宅ローン計算方法」、財務省「国債の利付債計算方式」等でも同一公式が採用されている。`annualRate = 0` の特殊ケースは極限（r→0）で `P/n` に収束することの実装。

---

## 年次ローン内訳分解（`calcYearlyLoanComponents`）

元利均等返済用の12ヶ月ループで月次利息・元金を積算する。

```go
for range 12 {
    monthInterest  = remaining × r
    monthPrincipal = monthlyPayment - monthInterest
    // 最終月: 残高 < 月次元金返済 → 残高のみ返済（端数防止）
    if monthPrincipal > remaining { monthPrincipal = remaining }
    ...
}
```

この積算方式により、年度末残高が正確に計算される。元金均等返済の場合は `calcYearlyLoanComponentsEqualPrincipal` を使用する（上記「元金均等返済」節を参照）。

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

## 年次賃料下落（`RentDeclineRate`）

毎年の実効賃料に下落係数を乗じることで経年劣化による賃料低下を表現する。

```
yearAnnualRent = baseAnnualRent × (1 - RentDeclineRate)^y
  baseAnnualRent : MonthlyRent × 12 × (1 - effectiveVacancy)
  y              : 経過年数（0始まり。1年目は y=0 で係数 1.0）
```

`RentDeclineRate = 0`（デフォルト）の場合は毎年同一賃料。

**構造別推奨値（経験則）**

| 建物構造 | 推奨下落率 |
|---------|-----------|
| 木造 | 1.0%/年 |
| 軽量鉄骨・重量鉄骨 | 0.8%/年 |
| RC造・SRC造 | 0.5%/年 |

---

## 各年次ループの計算順序

```
① ローン返済内訳（利息・元金）の計算
   - equal-payment: calcYearlyLoanComponents(残高, rate, monthlyPayment)
   - equal-principal: calcYearlyLoanComponentsEqualPrincipal(残高, rate, monthlyPrincipalFixed)
② 実効賃料収入・運営経費の計算（RentDeclineRate を適用）
③ 当年の減価償却費（耐用年数内のみ）
④ 課税所得 = 収入 - 利息 - 減価償却 - 経費
⑤ 所得税 = 課税所得 × IncomeTaxRate（課税所得 ≤ 0 なら 0）
⑥ キャッシュフロー（税引前）= 収入 - ローン返済 - 経費
⑦ 税引後CF = キャッシュフロー - 所得税
⑧ 累積CF += 税引後CF
⑨ デッドクロス判定
```

**元金均等返済時の注意点**:
- 初年度の年間返済額は元利均等より高い（月次元金返済が一定なのに対し利息は残高が多いため）
- 返済額の逆転（クロスオーバー）は約17年目（35年ローンの場合）
- `monthlyPrincipalFixed = LoanAmount / (LoanYears × 12)` を初期化時に算出し、各年次ループで使い回す

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
| `StressScenarios` | ストレスシナリオ結果配列（詳細は下記） |
| `DSCR` | 1年目の DSCR（NOI / 1年目年間ローン返済額）。ローンなしの場合は 0 |
| `LTVSensitivity` | LTV感度分析結果（LTV=50/60/70/80/90%の5行。`LTVSensitivityRow[]`） |

---

## ストレスシナリオ自動生成（`StressScenarioResult`）

`Analyze()` は呼び出し時に 6 つのデフォルトシナリオを自動生成し、`InvestmentResult.StressScenarios` に格納する。`VacancyRateDelta` または `LoanRateDelta` が 0 以外の場合はカスタムシナリオ（7 本目）も追加される。

### StressScenarioResult フィールド

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `Label` | string | シナリオ名（例: `"金利+1%"`） |
| `InterestRateDelta` | float64 | 金利上昇幅（率。例: 0.01 = +1%） |
| `VacancyRateDelta` | float64 | 空室率上昇幅（率。例: 0.10 = +10%pt） |
| `TotalCashFlow` | float64 | `HoldingYears` 期間の税引後累積キャッシュフロー（円） |
| `DSCR` | float64 | 負債返済カバレッジ比率（= 年間NOI / 年間ローン返済額）。ローンなしの場合は 0 |
| `BreakEvenYear` | int | 累積CFが初めてプラスに転じる年次（1-indexed）。期間内に達成できない場合は `-1` |
| `IsSafe` | bool | 安全判定フラグ（判定ロジックは下記参照） |

### 6 つのデフォルトシナリオ

| # | Label | InterestRateDelta | VacancyRateDelta |
|---|-------|-------------------|-----------------|
| 1 | ベースライン | 0 | 0 |
| 2 | 金利+1% | +0.01 | 0 |
| 3 | 金利+2% | +0.02 | 0 |
| 4 | 空室+10% | 0 | +0.10 |
| 5 | 空室+20% | 0 | +0.20 |
| 6 | 複合ストレス | +0.01 | +0.10 |

### IsSafe 判定ロジック

**ローンあり（`LoanAmount > 0`）の場合:**

```
IsSafe = DSCR >= 1.0 && BreakEvenYear != -1 && BreakEvenYear <= HoldingYears
```

- `DSCR >= 1.0`: 年間NOIがローン返済額を上回っている（債務返済能力あり）
- `BreakEvenYear != -1`: 保有期間内に累積CFが黒字転換する
- `BreakEvenYear <= HoldingYears`: 黒字転換が出口売却年以内に収まる

**ローンなし（`LoanAmount == 0`）の場合:**

```
IsSafe = BreakEvenYear != -1 && BreakEvenYear <= HoldingYears
```

DSCR はローン返済額ゼロにより意味をなさないため、BreakEvenYear のみで判定する。

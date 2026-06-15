---
purpose: InvestmentInput 全フィールド定義・計算手順の詳細仕様
triggers: [investment calculation, InvestmentInput, Analyze, 計算ロジック, 収益計算]
audience: backend-dev
token_weight: heavy
reads_next: [docs/llm/domain.md]
---

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
| `DiscountRate` | float64 | 率 | NPV/IRR計算の割引率（0.05 = 5%）。`0` 指定時は `Defaults()` で 0.05 に補完 | 0.05 |
| `PriceDeclineRate` | float64 | 率 | 物件価格の年間下落率（0.02 = 年2%）。IRR/NPVのターミナルバリューにのみ反映 | 0 |
| `DepreciationMethod` | string | — | 減価償却方式: `"straight-line"` または `"declining-balance"` | `"straight-line"` |
| `YieldTarget` | float64 | 率 | 目標表面利回り（例: 0.08 = 8%）。`IsAboveYieldTarget` の判定基準。0 指定時は `Defaults()` で 0.08 に補完 | 0.08 |
| `RateAdjustmentSchedule` | []RateAdjustment | — | 変動金利スケジュール。各要素に `afterYear`（適用開始年）と `rate`（適用金利、絶対値）を持つ。空配列 = 固定金利 | [] |
| `CapexSchedule` | []CapexEvent | — | 大規模修繕費スケジュール（最大5件）。各要素に `year`（発生年）と `amount`（円）を持つ | [] |
| `RentGrowthRate` | float64 | 率 | 年間賃料上昇率（例: 0.02 = 2%）。`RentGrowthYears` と組み合わせ新築・リノベ物件の賃料上昇期を表現 | 0 |
| `RentGrowthYears` | int | 年 | 賃料が `RentGrowthRate` で上昇し続ける年数。この年数を超えると `RentDeclineRate` のみ適用 | 0 |

**注意**: `VacancyRate`・`ExpenseRate`・`IncomeTaxRate` は 0 が有効値のため `Defaults()` では初期化されない。呼び出し側で必ず指定する。

**`RateAdjustment` 型**（`RateAdjustmentSchedule` の要素）:

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `afterYear` | int | この金利を適用する開始年（最小 2）。それ以前は `LoanRate` を使用 |
| `rate` | float64 | 適用金利（絶対値。例: 0.020 = 2%）。`LoanRateDelta` は常に加算される |

---

## ストレステスト入力の適用タイミング

```go
effectiveVacancy := input.VacancyRate + input.VacancyRateDelta
effectiveRate    := input.AnnualLoanRate + input.LoanRateDelta
```

`Analyze` の先頭で加算し、以降の全計算に `effectiveVacancy` / `effectiveRate` を使用する。
元の `VacancyRate` / `AnnualLoanRate` は変更しない。

### 変動金利スケジュール（`RateAdjustmentSchedule`）との重ね合わせ

`resolveRateForYear(baseRate, rateDelta, schedule, year)` は以下のロジックで各年の適用金利を決定する。

```go
rate := baseRate
for _, adj := range schedule {
    if year >= adj.AfterYear {
        rate = adj.Rate  // スケジュール金利で上書き（絶対値）
    }
}
return rate + rateDelta  // rateDelta は常に加算
```

- `RateAdjustmentSchedule` が設定されている場合、該当年以降はスケジュール金利（絶対値）に切り替わる。
- `LoanRateDelta`（ストレスΔ）は **スケジュール適用後の金利にも加算される**。  
  例: スケジュールで6年目以降 2.0%、ストレス +1% → 6年目以降の実効金利 = **3.0%**
- LTV 感度分析（`CalcLTVSensitivity`）も初年度実効金利 `resolveRateForYear(..., year=1)` を使用する。

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

## 表面利回り（MarketGrossYield）と総投資利回り（GrossYield）

本アプリは利回りを 2 つの分母で算出する（#773）。「表面利回り」ラベルで UI 表示し 8% 境界線判定に用いるのは **物件価格ベースの MarketGrossYield** である。

```
表面利回り（MarketGrossYield）= (MonthlyRent × 12) / 物件価格（LandPrice + BuildingCost）  ← 市場慣行・UI表示・8%判定
総投資利回り（GrossYield）      = (MonthlyRent × 12) / 総投資額（物件価格 + 諸経費）          ← 諸経費込みの保守的指標
```

いずれも**空室率を含まない**満室想定年収で計算する。

- **MarketGrossYield**: 物件広告・REINS・業者資料で一般に「表面利回り」と呼ばれる指標と一致する（分母に諸経費を含まない）。
- **GrossYield**: 諸経費まで含めた投下資本に対する利回り。MarketGrossYield より 0.5〜1% 低く出る保守的な参考値で、UI では「総投資利回り（諸費用込み）」として併記する。

### 8% 境界線（IsAboveYieldTarget）の判定基準

`IsAboveYieldTarget = MarketGrossYield ≥ YieldTarget`（YieldTarget デフォルト 0.08）。

業者が「利回り8%」と謳う物件は物件価格ベースで算出されているため、判定も物件価格ベースの MarketGrossYield で行うことで、広告値とアプリ表示の乖離をなくす。目標達成に必要な賃料・コスト削減額（`calcRequiredForTarget`）も物件価格を基準に逆算する。

> **根拠・出典**: 全国宅地建物取引業協会連合会（全宅連）および不動産情報サービス各社（SUUMO・HOME'S 等）が物件掲載時に使用する慣習的指標。物件間の横断比較に用いる。実際の収入は空室や経費分だけ下振れするため、実質利回りと併用して判断する。

## 空室シナリオ別利回り（calcYieldScenarios）

`Analyze()` 内で呼び出され、`InvestmentResult.YieldScenarios` として返される。

```
楽観シナリオ: effectiveVacancy = min(VacancyRate × 0.5, 0.99)
標準シナリオ: effectiveVacancy = min(VacancyRate × 1.0, 0.99)
悲観シナリオ: effectiveVacancy = min(VacancyRate × 1.5, 0.99)

AnnualRent = MonthlyRent × 12 × (1 - effectiveVacancy)
GrossYield = (MonthlyRent × 12) / 総投資額  ← 全シナリオ共通（満室想定・総投資利回り）
```

**0.99 キャップの意図**: 悲観×1.5 で空室率が 100% を超える場合（例: 入力 80% × 1.5 = 120%）、年間賃料がゼロ以下になることを防ぐ。`math.Min(..., 0.99)` で最大99%に制限する。

**GrossYield の一定性**: `YieldScenario.GrossYield` は総投資利回り（満室想定年収 / 総投資額）でシナリオ間で変化しない。シナリオで変化するのは `AnnualRent`（実効賃料）のみ。

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

## 年間経費の計算: `ExpenseRate` と `AnnualPropertyTax` の関係

### フィールドの意味

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `ExpenseRate` | float64 | 管理委託費・修繕積立・保険料などの**率ベース経費**（実効賃料に乗じる比率）。固定資産税は含まない |
| `AnnualPropertyTax` | float64 | **固定資産税の年額（絶対額、円）**。ExpenseRate とは独立した別建て計上 |

### なぜ固定資産税を分離するか

固定資産税は賃料に連動しない固定費（物件評価額に基づく）であるため、賃料 × 経費率で計算すると賃料が下落した年に固定資産税が過小計上される問題が生じる。これを防ぐため、固定資産税は絶対額（`AnnualPropertyTax`）として賃料に依存せず毎年定額で計上する。

> **二重計上の禁止**: `ExpenseRate` の中に固定資産税を含めつつ `AnnualPropertyTax` も設定すると、固定資産税が二重に計上される。どちらか一方のみを使用すること（`types.go` のコメント参照）。

### 年次経費の計算式（`simulateYears` 内）

```
有効経費率 = calcEffectiveExpenseRate(input)
           = ManagementFeeRate + RepairReserveRate + InsuranceFeeRate + OtherExpenseRate  ← 詳細内訳が 1 つでも > 0 の場合
           = ExpenseRate  ← 詳細内訳が全て 0 の場合（後方互換フォールバック）

経費インフレ補正後経費率 = 有効経費率 × (1 + ExpenseInflationRate)^(y-1)   ← y は 1-indexed 年次
年間経費 = 年間実効賃料 × 経費インフレ補正後経費率 + AnnualPropertyTax
```

**ポイント**:
- 詳細内訳フィールド（`ManagementFeeRate`, `RepairReserveRate`, `InsuranceFeeRate`, `OtherExpenseRate`）のいずれかが 0 より大きければ、その合計が `ExpenseRate` より**優先**される
- `AnnualPropertyTax` は経費インフレ補正の対象外（毎年同額を固定費として計上）
- 初年度の実質利回り計算（`initYieldParams`）では `AnnualPropertyTax` を加算しない（利回り計算は経費率ベースの近似値）

### 具体的な計算例

**入力条件:**

| 項目 | 値 |
|------|-----|
| MonthlyRent | 100,000円 |
| VacancyRate | 0.05（5%） |
| ExpenseRate | 0.15（15%） |
| AnnualPropertyTax | 120,000円 |
| ExpenseInflationRate | 0（インフレなし） |

**1年目の計算:**

```
年間実効賃料 = 100,000 × 12 × (1 − 0.05) = 1,140,000円
年間経費     = 1,140,000 × 0.15 + 120,000 = 291,000円
NOI          = 1,140,000 − 291,000 = 849,000円
```

**10年目（RentDeclineRate = 0.01 の場合）:**

```
年間実効賃料 = 1,140,000 × (1 − 0.01)^9 ≈ 1,041,600円
年間経費     = 1,041,600 × 0.15 + 120,000 = 276,240円
NOI          = 1,041,600 − 276,240 = 765,360円
```

賃料が下落しても `AnnualPropertyTax`（120,000円）は変わらないため、経費全体に占める固定資産税の比率が年々上昇する（経費の固定費化）。

---

## 目標利回り逆算ロジック（`calcRequiredForTarget`）

`input.YieldTarget`（デフォルト 0.08 = 8%）を基準に2つの逆算値を返す（`backend/internal/domain/sensitivity.go`）。

8% 境界線は物件価格ベースの表面利回り（MarketGrossYield）で判定するため、逆算も**物件価格（LandPrice + BuildingCost）**を基準に行う（#773。諸経費を含む総投資額ではない）。

```go
target := input.YieldTarget          // 目標表面利回り（例: 0.08）
propertyPrice := LandPrice + BuildingCost

// 目標年収 = 物件価格 × 目標利回り
requiredMonthlyRent = (propertyPrice × target) / 12

// 現賃料で目標利回り達成に必要な物件価格
requiredPropertyPrice = (MonthlyRent × 12) / target

// 過剰額（削減が必要な額）
costReduction = max(propertyPrice - requiredPropertyPrice, 0)
```

- `RequiredMonthlyRent`: 現在の物件価格で目標表面利回りを達成するために必要な月額賃料
- `RequiredCostReduction`: 現在の賃料で目標表面利回りを達成するために土地 **または** 建築費いずれか一方を削減すべき額

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

rate      = resolveRateForYear(AnnualLoanRate, LoanRateDelta, RateAdjustmentSchedule, year=1)

元利均等: 月次返済額 = calcMonthlyPayment(borrowing, rate, years)
           annualDebt = 月次返済額 × 12
元金均等: 1年目月次返済額 = CalcEqualPrincipalPayment(borrowing, rate, years×12, 1)
           annualDebt = 1年目月次返済額 × 12  ← 最保守的（最大値）で試算

NOI = 年間実効賃料収入 − 年間運営経費
DSCR = NOI / annualDebt
annualCF = NOI − annualDebt
cfYield  = annualCF / 総投資額
```

**注意**: LTV 感度分析は初年度実効金利（`resolveRateForYear(..., year=1)`）を使用する。`RateAdjustmentSchedule` が設定されている場合は初年度のスケジュール金利が、`LoanRateDelta` が設定されている場合はそれも加算された金利が適用される（`VacancyRateDelta` は無視）。元金均等では1年目（最大）の年間返済額を使用するため、実際の返済額より保守的な DSCR になる。

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
| `MarketGrossYield` | 表面利回り（満室想定年収 / 物件価格。市場慣行・UI表示・8%判定基準） |
| `GrossYield` | 総投資利回り（満室想定年収 / 総投資額。諸経費込みの保守的指標） |
| `NetYield` | 実質利回り |
| `IsAboveYieldTarget` | `MarketGrossYield` ≥ 目標利回り（`yieldTarget`）かどうか（物件価格ベース） |
| `YieldTarget` | 判定に使用した目標利回り（例: 0.08 = 8%） |
| `RequiredCostReduction` | 目標利回り達成に必要なコスト削減額（いずれか一方） |
| `RequiredMonthlyRent` | 目標利回り達成に必要な月額賃料 |
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
| `IRR` | *float64 | — | 内部収益率（自己資金ベースのレバレッジ考慮済み）。equity ≤ 0 または HoldingYears = 0 のとき `null` | — |
| `NPV` | float64 | 円 | 正味現在価値（`DiscountRate` で割り引いた保有期間CF + ターミナルバリュー - 自己資金） | — |

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
| `DSCR` | float64 | 保有期間内の最悪年 DSCR（賃料下落率を年次適用した yearNOI / 年間ローン返済額の最小値）。ローンなしの場合は 0 |
| `BreakEvenYear` | int | 累積**税引後**CFが初めてプラスに転じる年次（1-indexed）。期間内に達成できない場合は `-1` |
| `IsSafe` | bool | 安全判定フラグ（判定ロジックは下記参照） |

### 6 つのデフォルトシナリオ

| # | Label | InterestRateDelta | VacancyRateDelta |
|---|-------|-------------------|-----------------|
| 1 | ベースライン | 0 | 0 |
| 2 | 金利+1% | +0.01 | 0 |
| 3 | 金利+2% | +0.02 | 0 |
| 4 | 空室+10% | 0 | +0.10 |
| 5 | 空室+20% | 0 | +0.20 |
| 6 | 複合ストレス | +0.02 | +0.10 |

### IsSafe 判定ロジック

**ローンあり（`LoanAmount > 0`）の場合:**

```
IsSafe = DSCR >= 1.2 && BreakEvenYear != -1 && BreakEvenYear <= HoldingYears
```

- `DSCR >= 1.2`: 保有期間内の最悪年においても NOI がローン返済額を上回っている（実務基準の安全域）
- `BreakEvenYear != -1`: 保有期間内に累積**税引後**CFが黒字転換する
- `BreakEvenYear <= HoldingYears`: 黒字転換が出口売却年以内に収まる

> **注意**: UI バッジの「安全」表示は `IsSafe` と同一の閾値 1.2 を使用する。`IsSafe` フラグと UI バッジの判定基準は一致している。

**ローンなし（`LoanAmount == 0`）の場合:**

```
IsSafe = BreakEvenYear != -1 && BreakEvenYear <= HoldingYears
```

DSCR はローン返済額ゼロにより意味をなさないため、BreakEvenYear のみで判定する。

---

## IRR・NPV 計算（`CalcNPV` / `CalcIRR`）

### 前提: 自己資金ベースの計算

IRR・NPVの計算は**自己資金（equity = 総投資額 − ローン金額）**を初期投資として使用する。

- 年次CF: `AfterTaxCashFlow`（ローン返済・税引後の手取り）
- ターミナルバリュー: `ExitNetProceeds`（売却価格 − 売却費用 − 譲渡税 − ローン残債）

これにより、レバレッジ効果を加味した実質的なリターンが計算される。

### CalcNPV

```
NPV = Σ(t=1→n) CF_t / (1+r)^t + TV / (1+r)^n − I
```

- `r` = `DiscountRate`
- `TV` = ターミナルバリュー（`PriceDeclineRate > 0` の場合は下落補正済み）
- `I` = 自己資金（equity）

### CalcIRR（二分法）

```
CalcNPV(cfs, TV, IRR, I) = 0 となる IRR を数値探索
```

- 探索範囲: −50% 〜 +200%
- 最大反復: 200回
- 収束判定: |NPV| < 1円
- 非収束・equity ≤ 0・HoldingYears = 0 の場合は `null` を返す

### PriceDeclineRate の適用

`PriceDeclineRate > 0` の場合、出口売却価格に複利で下落を反映してターミナルバリューを再計算する。

```
調整後売却価格 = ExitSalePrice × (1 − PriceDeclineRate)^HoldingYears
```

既存の `ExitSalePrice` / `ExitNetProceeds` / `ExitTotalEquity` には影響しない（IRR/NPV 専用）。

---

## 修繕費ROI計算（`CalcRenovationROI`）

`backend/internal/domain/renovation.go`

### 入力: `RenovationInput`

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `PropertyPrice` | float64 | 物件取得価格（円） |
| `AnnualBaseRent` | float64 | リフォーム前年間家賃（円） |
| `AnnualExpenses` | float64 | 年間経費（円、絶対額） |
| `EffectiveTaxRate` | float64 | 実効税率（0.0〜1.0） |
| `SelfLaborRatePerHour` | float64 | セルフリフォーム時給（円/時間） |
| `Items` | []RenovationItem | 工事項目一覧（1件以上必須） |

**`RenovationItem`**:

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `Name` | string | 部位名（例: "外壁塗装"） |
| `Cost` | float64 | 工事費（円、正値必須） |
| `ExpectedMonthlyRentIncrease` | float64 | 期待月額賃料アップ（円） |
| `IsSelfWork` | bool | セルフリフォームか |
| `SelfLaborHours` | float64 | 工数（時間） |

### 出力: `RenovationResult`

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `RecoveryYears` | float64 | 修繕費回収期間（年）。家賃アップなしは `0` |
| `IsRecoverable` | bool | 回収可能か（`AnnualRentIncrease > 0` のとき `true`） |
| `TaxSavings` | float64 | 節税効果（円）= 修繕費計上額 × 実効税率 |
| `VirtualLaborCost` | float64 | セルフリフォーム仮想人件費合計（円） |
| `CapitalExpenditures` | float64 | 資本的支出合計（60万円超の工事） |
| `RepairExpenses` | float64 | 修繕費合計（60万円以下の工事） |
| `ActualYield` | float64 | 実質利回り |
| `TotalRenovationCost` | float64 | リフォーム総費用 |
| `AnnualRentIncrease` | float64 | 年間家賃アップ額 |
| `ClassifiedItems` | []ClassifiedRenovationItem | 分類済み工事項目（入力と同数） |

### 計算式

```
資本的支出判定: item.Cost > 600,000 → 資本的支出（即時経費化不可）
修繕費判定:    item.Cost ≤ 600,000 → 修繕費（即時費用計上可能）

根拠: 所得税法施行令第181条（1回の修繕費が60万円未満は修繕費として処理可）

回収期間 = TotalRenovationCost / (月間家賃アップ合計 × 12)

節税効果 = RepairExpenses × EffectiveTaxRate

実質利回り = (AnnualBaseRent + AnnualRentIncrease - AnnualExpenses)
           / (PropertyPrice + TotalRenovationCost)
```

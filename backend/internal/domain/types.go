package domain

import "fmt"

// BuildingType は建物構造種別を表す
type BuildingType string

const (
	BuildingTypeWood        BuildingType = "木造"             // 耐用年数 22年
	BuildingTypeLightSteel  BuildingType = "軽量鉄骨(4mm以下)" // 耐用年数 27年
	BuildingTypeLightSteelThin BuildingType = "軽量鉄骨(3mm以下)" // 耐用年数 19年 (薄板・プレハブ系)
	BuildingTypeHeavySteel  BuildingType = "重量鉄骨"          // 耐用年数 34年
	BuildingTypeRC          BuildingType = "RC造"             // 耐用年数 47年
	BuildingTypeSRC         BuildingType = "SRC造"            // 耐用年数 47年 (鉄骨鉄筋コンクリート)
)

// IsValid は定義済みの建物種別定数のいずれかであるかを返す
func (b BuildingType) IsValid() bool {
	switch b {
	case BuildingTypeWood, BuildingTypeLightSteel, BuildingTypeLightSteelThin,
		BuildingTypeHeavySteel, BuildingTypeRC, BuildingTypeSRC:
		return true
	}
	return false
}

// UsefulLife は建物種別の法定耐用年数（住宅用）を返す
// 根拠: 減価償却資産の耐用年数等に関する省令 別表第一
func (b BuildingType) UsefulLife() int {
	switch b {
	case BuildingTypeWood:
		return 22
	case BuildingTypeLightSteelThin:
		return 19
	case BuildingTypeLightSteel:
		return 27
	case BuildingTypeHeavySteel:
		return 34
	case BuildingTypeRC, BuildingTypeSRC:
		return 47
	default:
		return 22
	}
}

// CalcResidualUsefulLife は中古物件の簡便法耐用年数を算出する
// 根拠: 耐用年数の適用等に関する取扱通達 1-5-3
func CalcResidualUsefulLife(buildingType BuildingType, buildingAge int) int {
	legal := buildingType.UsefulLife()
	if buildingAge <= 0 {
		return legal // 新築
	}
	var residual int
	if buildingAge >= legal {
		// 法定耐用年数を超過した中古: 法定耐用年数 × 0.2（端数切捨て、最低2年）
		residual = int(float64(legal) * 0.2)
	} else {
		// 法定耐用年数内の中古: (法定 - 経過年数) + 経過年数 × 0.2
		residual = (legal - buildingAge) + int(float64(buildingAge)*0.2)
	}
	if residual < 2 {
		return 2
	}
	return residual
}

// RateAdjustment は変動金利スケジュールの1ステップ
type RateAdjustment struct {
	// AfterYear: この年（1始まり）以降に Rate を適用（例: 6 = 6年目から）
	AfterYear int `json:"afterYear"`
	// Rate: 絶対値の年利（例: 0.02 = 2%）
	Rate float64 `json:"rate"`
}

// InvestmentInput は収支シミュレーションの入力値
type InvestmentInput struct {
	LandPrice       float64      `json:"landPrice"`       // 土地取得費 (円)
	LandArea        float64      `json:"landArea"`        // 土地面積 (m²)
	BuildingCost    float64      `json:"buildingCost"`    // 建築費 (円)
	BuildingAge     int          `json:"buildingAge"`     // 築年数 (0=新築)
	MiscExpenseRate float64      `json:"miscExpenseRate"` // 諸経費率 (例: 0.07)
	MonthlyRent     float64      `json:"monthlyRent"`     // 想定月額賃料 (円)
	VacancyRate     float64      `json:"vacancyRate"`     // 空室率 (例: 0.05)
	LoanAmount      float64      `json:"loanAmount"`      // ローン金額 (円)
	AnnualLoanRate  float64      `json:"annualLoanRate"`  // 年利 (例: 0.015)
	LoanYears       int          `json:"loanYears"`       // ローン期間 (年)
	BuildingType    BuildingType `json:"buildingType"`    // 建物構造
	// 運営経費率 (管理費・修繕・保険等。固定資産税・ローン利息は含まない)
	// 固定資産税は AnnualPropertyTax で別途指定する
	ExpenseRate   float64 `json:"expenseRate"`   // 例: 0.20
	IncomeTaxRate float64 `json:"incomeTaxRate"` // 所得税率 (例: 0.33。給与との合算後実効税率)
	HoldingYears  int     `json:"holdingYears"`  // 出口戦略: 売却年数 (例: 10)
	// 出口戦略: 売却時目標利回り（実質ベース。NOI / 売却価格）
	ExitYieldTarget float64 `json:"exitYieldTarget"` // 例: 0.06

	// 現況空室率: 現時点の実際の空室状況（0の場合はVacancyRateと同一とみなす）
	ActualVacancyRate float64 `json:"actualVacancyRate"`

	// ストレステスト用オフセット
	VacancyRateDelta float64 `json:"vacancyRateDelta"` // 空室率上昇分 (例: +0.10)
	LoanRateDelta    float64 `json:"loanRateDelta"`    // 金利上昇分 (例: +0.015)

	// 年間固定資産税 (円)。ExpenseRate には含めないこと（二重計上になる）
	AnnualPropertyTax float64 `json:"annualPropertyTax"`

	// 賃料下落率: 毎年この割合だけ実効賃料が低下する（例: 0.01 = 年1%下落）
	RentDeclineRate float64 `json:"rentDeclineRate"`

	// 割引率: NPV/IRR計算用 (例: 0.05 = 5%)
	DiscountRate float64 `json:"discountRate"`
	// 物件価格下落率: 売却価格に毎年この割合の累乗で下落を反映 (例: 0.02 = 年2%下落)
	PriceDeclineRate float64 `json:"priceDeclineRate"`
	// 減価償却方式: "straight-line"（定額法）| "declining-balance"（定率法）
	DepreciationMethod string `json:"depreciationMethod"`

	// 目標表面利回り（例: 0.08 = 8%）。0 の場合は Defaults() で 0.08 にセットされる。
	YieldTarget float64 `json:"yieldTarget"`

	// 返済方式: "equal-payment"（元利均等）| "equal-principal"（元金均等）
	// 省略時は Defaults() で "equal-payment" にセットされる。
	LoanMethod string `json:"loanMethod"`

	// 変動金利スケジュール: 指定年以降に適用する金利ステップ（空=固定金利）
	// LoanRateDelta はスケジュール後の金利にも加算される。
	RateAdjustmentSchedule []RateAdjustment `json:"rateAdjustmentSchedule"`

	// 大規模修繕費スケジュール（最大5件）
	CapexSchedule []CapexEvent `json:"capexSchedule,omitempty"`

	// 賃料上昇シナリオ（新築・リノベ向け）
	// RentGrowthYears 年間は RentGrowthRate で上昇し、その後 RentDeclineRate で下落する。
	// どちらかが 0 の場合は従来の RentDeclineRate のみが適用される。
	RentGrowthRate  float64 `json:"rentGrowthRate,omitempty"`  // 年間賃料上昇率 (例: 0.02 = 2%)
	RentGrowthYears int     `json:"rentGrowthYears,omitempty"` // 上昇が続く年数

	// 融資諸費用率（保証料・登記費用等の合算）。LoanAmount × LoanFeeRate を取得費に加算。
	LoanFeeRate float64 `json:"loanFeeRate,omitempty"`

	// 複数保有年数の出口比較: 空の場合は [5, 10, 15, 20] をデフォルトとして使用
	ExitYears []int `json:"exitYears,omitempty"`

	// 詳細経費内訳（全てoptional・後方互換）。合計 > 0 の場合は ExpenseRate より優先される。
	ManagementFeeRate    float64 `json:"managementFeeRate,omitempty"`    // 管理委託費率 (例: 0.05 = 5%)
	RepairReserveRate    float64 `json:"repairReserveRate,omitempty"`    // 修繕積立費率 (例: 0.01 = 1%)
	InsuranceFeeRate     float64 `json:"insuranceFeeRate,omitempty"`     // 保険料率 (例: 0.003 = 0.3%)
	OtherExpenseRate     float64 `json:"otherExpenseRate,omitempty"`     // その他経費率 (例: 0.005 = 0.5%)
	ExpenseInflationRate float64 `json:"expenseInflationRate,omitempty"` // 経費インフレ率/年 (例: 0.01 = 1%)

	// 入退去コスト（全て optional。0 または未入力の場合はターンオーバーコスト = 0）
	AvgTenancyYears float64 `json:"avgTenancyYears,omitempty"` // 平均入居期間（年）例: 2.0
	RestorationCost float64 `json:"restorationCost,omitempty"` // 原状回復費（円/回）例: 150000
	AdFee           float64 `json:"adFee,omitempty"`           // 入居者募集 AD（円/回）例: 家賃1ヶ月分
	RentFreePeriod  float64 `json:"rentFreePeriod,omitempty"`  // フリーレント（ヶ月）例: 0.5

	// 税務シミュレーション用（任意）。0 の場合は損益通算・法人比較を計算しない。
	SalaryIncome float64 `json:"salaryIncome,omitempty"` // 給与年収（円）
}

// CapexEvent は大規模修繕費の発生スケジュール1件
type CapexEvent struct {
	Year   int     `json:"year"`   // 何年目に発生（1〜保有年数）
	Amount float64 `json:"amount"` // 金額（円）
}

// Validate は入力値のバリデーションを行い、不正な組み合わせはエラーを返す。
// VacancyRate + VacancyRateDelta > 0.99 の場合、空室率オーバーフローとみなす。
func (i *InvestmentInput) Validate() error {
	if i.VacancyRate+i.VacancyRateDelta > 0.99 {
		return fmt.Errorf(
			"VacancyRate(%.2f) + VacancyRateDelta(%.2f) = %.2f exceeds maximum allowed vacancy of 0.99",
			i.VacancyRate, i.VacancyRateDelta, i.VacancyRate+i.VacancyRateDelta,
		)
	}
	if i.RentDeclineRate < 0 || i.RentDeclineRate > 0.2 {
		return fmt.Errorf(
			"rentDeclineRate は 0.0〜0.2 の範囲で指定してください（指定値: %.3f）",
			i.RentDeclineRate,
		)
	}
	if i.LoanMethod != "" && i.LoanMethod != LoanMethodEqualPayment && i.LoanMethod != LoanMethodEqualPrincipal {
		return fmt.Errorf(
			"loanMethod は %q または %q を指定してください（指定値: %q）",
			LoanMethodEqualPayment, LoanMethodEqualPrincipal, i.LoanMethod,
		)
	}
	loanYears := i.LoanYears
	if loanYears == 0 {
		loanYears = 35 // Defaults() 適用前に呼ばれる場合の保護
	}
	for idx, adj := range i.RateAdjustmentSchedule {
		if adj.AfterYear < 2 {
			return fmt.Errorf(
				"RateAdjustmentSchedule[%d].AfterYear(%d) must be >= 2",
				idx, adj.AfterYear,
			)
		}
		if adj.AfterYear > loanYears {
			return fmt.Errorf(
				"RateAdjustmentSchedule[%d].AfterYear(%d) exceeds LoanYears(%d)",
				idx, adj.AfterYear, loanYears,
			)
		}
		if adj.Rate <= 0 || adj.Rate > 0.3 {
			return fmt.Errorf(
				"RateAdjustmentSchedule[%d].Rate(%.4f) must be between 0 and 0.3",
				idx, adj.Rate,
			)
		}
		if idx > 0 && adj.AfterYear <= i.RateAdjustmentSchedule[idx-1].AfterYear {
			return fmt.Errorf(
				"RateAdjustmentSchedule must be sorted in ascending order by afterYear: [%d].AfterYear(%d) <= [%d].AfterYear(%d)",
				idx, adj.AfterYear, idx-1, i.RateAdjustmentSchedule[idx-1].AfterYear,
			)
		}
	}
	// 建物は平成10年（1998年）4月以降、定率法不可（国税庁 No.2100）
	if i.DepreciationMethod == DepreciationMethodDecliningBalance && i.BuildingCost > 0 {
		return fmt.Errorf(
			"建物の償却方法は定額法のみです（1998年4月以降取得の建物は定率法を選択できません）",
		)
	}
	holdingYears := i.HoldingYears
	if holdingYears == 0 {
		holdingYears = 10 // Defaults() 適用前に呼ばれる場合の保護
	}
	for _, ev := range i.CapexSchedule {
		if ev.Year > holdingYears {
			return fmt.Errorf(
				"CapexSchedule: year %d exceeds HoldingYears %d",
				ev.Year, holdingYears,
			)
		}
	}
	return nil
}

// Defaults は構造的デフォルト（省略可能なフィールド）にのみ適用する。
// VacancyRate・ExpenseRate・IncomeTaxRate は 0 が有効値のため呼び出し側で必ず指定すること。
func (i *InvestmentInput) Defaults() {
	if i.MiscExpenseRate == 0 {
		i.MiscExpenseRate = 0.07
	}
	if i.HoldingYears == 0 {
		i.HoldingYears = 10
	}
	if i.ExitYieldTarget == 0 {
		i.ExitYieldTarget = 0.06
	}
	if i.YieldTarget == 0 {
		i.YieldTarget = 0.08
	}
	if i.LoanYears == 0 {
		i.LoanYears = 35
	}
	if i.BuildingType == "" {
		i.BuildingType = BuildingTypeWood
	}
	if i.LoanMethod == "" {
		i.LoanMethod = LoanMethodEqualPayment
	}
	if i.DiscountRate == 0 {
		i.DiscountRate = 0.05
	}
	if i.DepreciationMethod == "" {
		i.DepreciationMethod = DepreciationMethodStraightLine
	}
}

// YearlyResult は各年の収支シミュレーション結果
type YearlyResult struct {
	Year                 int     `json:"year"`
	AnnualRent           float64 `json:"annualRent"`           // 実効賃料収入（空室控除後）
	AnnualLoanPayment    float64 `json:"annualLoanPayment"`
	AnnualInterest       float64 `json:"annualInterest"`
	AnnualPrincipal      float64 `json:"annualPrincipal"`
	AnnualDepreciation   float64 `json:"annualDepreciation"`
	AnnualExpenses       float64 `json:"annualExpenses"`
	TaxableIncome        float64 `json:"taxableIncome"`
	IncomeTax            float64 `json:"incomeTax"`
	CapexAmount          float64 `json:"capexAmount"`          // 大規模修繕費（当年発生額）
	CashFlow             float64 `json:"cashFlow"`
	AfterTaxCashFlow     float64 `json:"afterTaxCashFlow"`
	RemainingLoanBalance float64 `json:"remainingLoanBalance"`
	CumulativeCashFlow   float64 `json:"cumulativeCashFlow"`
	IsDeadCrossYear      bool    `json:"isDeadCrossYear"`   // デッドクロス初年度
	IsInDeadCrossZone    bool    `json:"isInDeadCrossZone"` // デッドクロス継続中
	EffectiveRate        float64 `json:"effectiveRate"`     // その年の適用金利（年利）
}

// CriticalErrorStatus は重大エラーの深刻度
type CriticalErrorStatus string

const (
	CriticalStatusReject  CriticalErrorStatus = "REJECT"
	CriticalStatusWarning CriticalErrorStatus = "WARNING"
)

// CriticalError は投資判断における重大リスク項目
type CriticalError struct {
	Code    string              `json:"code"`
	Status  CriticalErrorStatus `json:"status"`
	Message string              `json:"message"`
}

// StressScenarioResult はストレステストの1シナリオ結果
type StressScenarioResult struct {
	Label             string  `json:"label"`
	InterestRateDelta float64 `json:"interestRateDelta"`
	VacancyRateDelta  float64 `json:"vacancyRateDelta"`
	TotalCashFlow     float64 `json:"totalCashFlow"`
	DSCR              float64 `json:"dscr"`
	BreakEvenYear     int     `json:"breakEvenYear"` // 累積税引後CFが初めて正転する年（-1=なし）
	IsSafe            bool    `json:"isSafe"`        // DSCR >= 1.0 && BreakEvenYear <= HoldingYears（UIバッジは1.2閾値、フラグ定義は変更しない）
}

// YieldScenario は1つの空室シナリオにおける利回り結果
type YieldScenario struct {
	AnnualRent float64 `json:"annualRent"` // 年間実効賃料収入（空室控除後）
	GrossYield float64 `json:"grossYield"` // 表面利回り（満室想定年収/総投資額）
}

// YieldScenarios は楽観・標準・悲観シナリオの利回り結果セット
type YieldScenarios struct {
	Optimistic  YieldScenario `json:"optimistic"`  // 楽観: 空室率 × 0.5
	Standard    YieldScenario `json:"standard"`    // 標準: 空室率 × 1.0
	Pessimistic YieldScenario `json:"pessimistic"` // 悲観: 空室率 × 1.5
}

// LTVSensitivityRow は1つの LTV 水準における収支試算結果
type LTVSensitivityRow struct {
	LTV        float64 `json:"ltv"`        // 借入比率（例: 0.70）
	Equity     float64 `json:"equity"`     // 自己資金（円）
	LoanAmount float64 `json:"loanAmount"` // 借入額（円）
	DSCR       float64 `json:"dscr"`       // 借入金償還余裕率
	AnnualCF   float64 `json:"annualCF"`   // 年間キャッシュフロー（円）
	CFYield    float64 `json:"cfYield"`    // CF利回り（AnnualCF / 総投資額）
}

// MultiExitRow は複数保有年数の出口比較テーブルの1行
type MultiExitRow struct {
	Year            int      `json:"year"`
	SalePrice       float64  `json:"salePrice"`
	TransferTaxRate float64  `json:"transferTaxRate"`
	TransferTax     float64  `json:"transferTax"`
	RemainingLoan   float64  `json:"remainingLoan"`
	CumulativeCF    float64  `json:"cumulativeCf"`
	ExitEquity      float64  `json:"exitEquity"`
	IRR             *float64 `json:"irr,omitempty"`
	IsShortTermWarn bool     `json:"isShortTermWarn"`
}

// InvestmentResult は収支シミュレーションの結果
type InvestmentResult struct {
	TotalInvestment float64 `json:"totalInvestment"` // 総投資額（土地+建物+諸経費）
	MiscExpenses    float64 `json:"miscExpenses"`
	GrossYield      float64 `json:"grossYield"`      // 表面利回り（満室想定年収/総投資額）
	NetYield        float64 `json:"netYield"`        // 実質利回り（実効収入-経費)/総投資額）
	IsAboveYieldTarget bool    `json:"isAboveYieldTarget"`
	YieldTarget        float64 `json:"yieldTarget"` // 目標表面利回り（例: 0.08）

	// 目標利回り達成に必要な改善額（土地・建築費いずれか一方を削減する額）
	RequiredCostReduction float64 `json:"requiredCostReduction"`
	RequiredMonthlyRent   float64 `json:"requiredMonthlyRent"` // または必要月額賃料

	DeadCrossYear    int                      `json:"deadCrossYear"` // -1 = デッドクロスなし
	YearlyResults    []YearlyResult           `json:"yearlyResults"`
	CriticalErrors   []CriticalError          `json:"criticalErrors"`
	AcquisitionCosts AcquisitionCostBreakdown `json:"acquisitionCosts"`

	ExitSalePrice   float64 `json:"exitSalePrice"`   // 売却価格（NOI/目標利回り）
	ExitCapitalGain float64 `json:"exitCapitalGain"` // 譲渡所得
	ExitTransferTax float64 `json:"exitTransferTax"` // 譲渡所得税
	ExitNetProceeds float64 `json:"exitNetProceeds"` // 売却手取り（税・残債控除後）
	ExitTotalEquity float64 `json:"exitTotalEquity"` // 最終手残り（売却手取り+累積CF）

	IRR *float64 `json:"irr"` // 内部収益率（収束しない場合は null）
	NPV float64  `json:"npv"` // 正味現在価値

	StressScenarios []StressScenarioResult `json:"stressScenarios"`
	YieldScenarios  YieldScenarios         `json:"yieldScenarios"`

	DSCR           float64             `json:"dscr"`           // 1年目の借入金償還余裕率（NOI/年間返済額）
	LTVSensitivity []LTVSensitivityRow `json:"ltvSensitivity"` // LTV感度分析（50%〜90%）
	TotalInterest  float64             `json:"totalInterest"`  // 保有期間の総支払利息

	AISummary string `json:"aiSummary"` // Gemini 生成の投資サマリー（空文字 = 未生成）

	MultiExitComparison []MultiExitRow `json:"multiExitComparison,omitempty"` // 複数保有年数の出口比較

	TaxSimulation *TaxSimulationResult `json:"taxSimulation,omitempty"` // 損益通算・個人/法人比較（SalaryIncome>0時のみ）
}

// TaxSimulationResult は損益通算・個人/法人比較の結果
type TaxSimulationResult struct {
	SalaryLossCarryover SalaryLossCarryoverResult `json:"salaryLossCarryover"`
	OwnershipComparison OwnershipComparisonResult `json:"ownershipComparison"`
}

// SalaryLossCarryoverResult は給与所得との損益通算シミュレーション（#399）
type SalaryLossCarryoverResult struct {
	SalaryIncomeYen   float64      `json:"salaryIncomeYen"`   // 入力給与年収（円）
	BaselineSalaryTax float64      `json:"baselineSalaryTax"` // 不動産なし時の所得税（年額）
	YearlyRows        []TaxSimRow  `json:"yearlyRows"`
	TotalTaxSaving    float64      `json:"totalTaxSaving"` // 保有期間合計節税額
}

// TaxSimRow は各年の損益通算詳細
type TaxSimRow struct {
	Year            int     `json:"year"`
	RETaxableIncome float64 `json:"reTaxableIncome"` // 不動産課税所得（YearlyResult.TaxableIncome）
	CombinedIncome  float64 `json:"combinedIncome"`  // 給与+不動産 合算課税所得
	CombinedTax     float64 `json:"combinedTax"`     // 合算後所得税
	TaxDifference   float64 `json:"taxDifference"`   // 正=節税 / 負=増税
}

// OwnershipComparisonResult は個人 vs 法人の税負担比較（#392）
type OwnershipComparisonResult struct {
	Individual    OwnershipScenario `json:"individual"`
	Corporate     OwnershipScenario `json:"corporate"`
	BreakevenYear int               `json:"breakevenYear"` // 法人が有利になる年 (-1=保有期間内に逆転なし)
}

// OwnershipScenario は保有形態別の年次税負担
type OwnershipScenario struct {
	Label            string    `json:"label"`
	AnnualTax        []float64 `json:"annualTax"`        // 各年の所得税負担（保有年数分）
	TransferTax      float64   `json:"transferTax"`      // 売却時譲渡税
	TotalTaxBurden   float64   `json:"totalTaxBurden"`   // 保有期間合計（所得税+譲渡税）
	CumulativeBurden []float64 `json:"cumulativeBurden"` // 累積税負担（グラフ用）
}

// AcquisitionCostBreakdown は物件取得時の諸経費内訳
type AcquisitionCostBreakdown struct {
	BrokerageFee             float64 `json:"brokerageFee"`             // 仲介手数料（税込）
	StampDuty                float64 `json:"stampDuty"`                // 印紙税（売買契約書）
	RegistrationTax          float64 `json:"registrationTax"`          // 登録免許税（所有権移転+抵当権設定）
	RealEstateAcquisitionTax float64 `json:"realEstateAcquisitionTax"` // 不動産取得税（概算）
	PropertyTaxProration     float64 `json:"propertyTaxProration"`     // 固定資産税日割り精算（買主負担分）
	Total                    float64 `json:"total"`
}

// LandTransaction は国交省APIから取得した土地取引1件
type LandTransaction struct {
	Period           string  `json:"period"`
	District         string  `json:"district"`
	TradePrice       float64 `json:"tradePrice"`
	Area             float64 `json:"area"`
	PricePerSqm      float64 `json:"pricePerSqm"`
	PricePerTsubo    float64 `json:"pricePerTsubo"`
	CityPlanning     string  `json:"cityPlanning"`
	BuildingCoverage string  `json:"buildingCoverage"`
	FloorAreaRatio   string  `json:"floorAreaRatio"`
	BuildingYear     int     `json:"buildingYear,omitempty"`   // 建築年（西暦）
	StationMinutes   int     `json:"stationMinutes,omitempty"` // 最寄り駅徒歩分
}

// TheoreticalPriceResult は築年数・駅距離・乗降客数補正による理論価格推定の結果
type TheoreticalPriceResult struct {
	TheoreticalPriceJPY  float64              `json:"theoreticalPriceJPY"`  // 理論価格 (円)
	DeviationPct         float64              `json:"deviationPct"`          // 乖離率 % (正=割高, 負=割安)
	AgeCorrection        float64              `json:"ageCorrection"`         // 築年数補正 (-0.3〜0.3)
	StationCorrection    float64              `json:"stationCorrection"`     // 駅距離補正 (-0.2〜0.2)
	RidershipCorrection  float64              `json:"ridershipCorrection"`   // 乗降客数補正 (-0.15〜0.15)
	MedianBuildingAge    int                  `json:"medianBuildingAge"`     // 取引事例中央値築年数
	MedianStationMinutes int                  `json:"medianStationMinutes"`  // 取引事例中央値駅距離(分)
	IsLowDataWarning     bool                 `json:"isLowDataWarning"`      // 築年数サンプル不足
	HasStationData       bool                 `json:"hasStationData"`        // 駅距離補正が有効か
	RidershipScore       RidershipDemandScore `json:"ridershipScore,omitempty"` // 需要スコア (有効時のみ)
	HasRidershipData     bool                 `json:"hasRidershipData"`      // 乗降客数補正が有効か
}

// StationRidershipResult は駅別乗降客数APIのレスポンス（ドメイン層）
type StationRidershipResult struct {
	StationName  string               `json:"stationName"`
	LineName     string               `json:"lineName"`
	Passengers   int                  `json:"passengers"`   // 乗降客数/日
	DemandScore  RidershipDemandScore `json:"demandScore"`  // 需要スコア A〜E
	Correction   float64              `json:"correction"`   // 理論価格補正係数
}

// LocationOptimizationItem は XKT003 立地適正化計画のドメイン層受け渡し用軽量型
type LocationOptimizationItem struct {
	KubunNameJa string // 区域名（例: 居住誘導区域、都市機能誘導区域）
}

// EmbankmentItem は XKT020 大規模盛土造成地のドメイン層受け渡し用軽量型
type EmbankmentItem struct {
	Classification string // 盛土区分（例: 谷埋め型）
}

// UrbanRoadItem は XKT030 都市計画道路のドメイン層受け渡し用軽量型
type UrbanRoadItem struct {
	PlanningRoadJa string
	KubunID        int // 3011=都市計画道路、3023=広場
}

// DisasterHistoryItem は XST001 災害履歴のドメイン層受け渡し用軽量型
type DisasterHistoryItem struct {
	Name string // 災害種別名（例: 浸水域）
	Year int    // 発生年（不明時は0）
}

// UrbanZoningItem は XKT001 都市計画区域/区域区分のドメイン層受け渡し用軽量型
type UrbanZoningItem struct {
	AreaClassificationJa string
	KubunID              int
}

// LiquefactionRiskItem は XKT025 液状化発生傾向図のドメイン層受け渡し用軽量型
type LiquefactionRiskItem struct {
	TendencyLevel int    // liquefaction_tendency_level（6段階: 低値ほど高リスク）
	Note          string
}

// FloodHazardItem は XKT026 洪水浸水想定区域のドメイン層受け渡し用軽量型
type FloodHazardItem struct {
	DepthRank int    // A31a_205（浸水深ランク）
	RiverName string
}

// StormHazardItem は XKT027 高潮浸水想定区域のドメイン層受け渡し用軽量型
type StormHazardItem struct {
	DepthJa string // A49_003
}

// TsunamiHazardItem は XKT028 津波浸水想定のドメイン層受け渡し用軽量型
type TsunamiHazardItem struct {
	DepthJa string // A40_003
}

// LandslideHazardItem は XKT029 土砂災害警戒区域のドメイン層受け渡し用軽量型
type LandslideHazardItem struct {
	PhenomenonType int // A33_001（1=急傾斜地崩壊, 2=土石流, 3=地すべり）
	ZoneCode       int // A33_002（1=特別警戒区域, 2=警戒区域）
}

// InvestmentScoreResult は投資適地スコアの算出結果
type InvestmentScoreResult struct {
	TotalScore int            `json:"totalScore"`
	Grade      string         `json:"grade"` // 優良/良好/普通/注意/要注意
	Breakdown  ScoreBreakdown `json:"breakdown"`
}

// ScoreItem は各指標のスコアと説明
type ScoreItem struct {
	Score       int    `json:"score"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// RadarPoint はレーダーチャート用の正規化スコア（0〜100）
type RadarPoint struct {
	Category string  `json:"category"`
	Score    float64 `json:"score"`
}

// ScoreBreakdown は投資適地スコアの内訳
type ScoreBreakdown struct {
	Population       ScoreItem    `json:"population"`
	Ridership        ScoreItem    `json:"ridership"`
	UrbanArea        ScoreItem    `json:"urbanArea"`
	LocationOpt      ScoreItem    `json:"locationOptimization"`
	HazardRisk       ScoreItem    `json:"hazardRisk"`
	LiquefactionRisk ScoreItem    `json:"liquefactionRisk"`
	Embankment       ScoreItem    `json:"embankment"`
	DisasterHistory  ScoreItem    `json:"disasterHistory"`
	LandPriceTrend   ScoreItem    `json:"landPriceTrend"`
	RadarData        []RadarPoint `json:"radarData"`
}

// ZoningSummary はエリア内の取引から抽出した代表的な用途地域情報
type ZoningSummary struct {
	CityPlanning     string `json:"cityPlanning"`     // 最頻の都市計画区域（例: 第一種住居地域）
	BuildingCoverage string `json:"buildingCoverage"` // 最頻の建ぺい率（例: 60%）
	FloorAreaRatio   string `json:"floorAreaRatio"`   // 最頻の容積率（例: 200%）
}

// UrbanRiskLevel は都市計画リスクの深刻度
type UrbanRiskLevel string

const (
	UrbanRiskLevelError   UrbanRiskLevel = "ERROR"
	UrbanRiskLevelWarning UrbanRiskLevel = "WARNING"
	UrbanRiskLevelInfo    UrbanRiskLevel = "INFO"
)

// UrbanRisk は都市計画上のリスク項目
type UrbanRisk struct {
	Code        string         `json:"code"`        // リスクコード（例: URBANIZATION_CONTROL）
	Level       UrbanRiskLevel `json:"level"`       // ERROR / WARNING / INFO
	Title       string         `json:"title"`       // 短いタイトル
	Description string         `json:"description"` // 詳細説明
}

// LandPriceStats は土地価格の統計情報
type LandPriceStats struct {
	Count          int               `json:"count"`
	AverageTsubo   float64           `json:"averageTsubo"`
	MedianTsubo    float64           `json:"medianTsubo"`
	MinTsubo       float64           `json:"minTsubo"`
	MaxTsubo       float64           `json:"maxTsubo"`
	Transactions   []LandTransaction `json:"transactions"`
	LowDataWarning bool              `json:"lowDataWarning"` // 件数 < 10 件時 true
	WarningMessage string            `json:"warningMessage,omitempty"`
	Zoning     *ZoningSummary `json:"zoning,omitempty"`     // 取引データから抽出した用途地域情報
	UrbanRisks []UrbanRisk    `json:"urbanRisks,omitempty"` // 都市計画リスク一覧
}

// LandPriceComparison は検討中の土地価格と相場の比較
type LandPriceComparison struct {
	Stats              LandPriceStats `json:"stats"`
	InputLandPrice     float64        `json:"inputLandPrice"`
	InputArea          float64        `json:"inputArea"`
	InputPricePerTsubo float64        `json:"inputPricePerTsubo"`
	DiffFromAverage    float64        `json:"diffFromAverage"`
	DiffFromMedian     float64        `json:"diffFromMedian"`
	Assessment         string         `json:"assessment"` // "割安" / "相場" / "割高"
}

// RenovationItem はリフォーム1工事項目
type RenovationItem struct {
	Name                        string  `json:"name"`
	Cost                        float64 `json:"cost"`
	ExpectedMonthlyRentIncrease float64 `json:"expectedMonthlyRentIncrease"`
	IsSelfWork                  bool    `json:"isSelfWork"`
	SelfLaborHours              float64 `json:"selfLaborHours"`
}

// ClassifiedRenovationItem は分類付きリフォーム項目
type ClassifiedRenovationItem struct {
	RenovationItem
	IsCapitalExpenditure bool    `json:"isCapitalExpenditure"`
	VirtualLaborCost     float64 `json:"virtualLaborCost"`
}

// RenovationInput はリフォームROIシミュレーションの入力値
type RenovationInput struct {
	PropertyPrice        float64          `json:"propertyPrice"`
	AnnualBaseRent       float64          `json:"annualBaseRent"`
	AnnualExpenses       float64          `json:"annualExpenses"`
	EffectiveTaxRate     float64          `json:"effectiveTaxRate"`
	SelfLaborRatePerHour float64          `json:"selfLaborRatePerHour"`
	Items                []RenovationItem `json:"items"`
}

// RenovationResult はリフォームROIシミュレーションの結果
type RenovationResult struct {
	RecoveryYears       float64                    `json:"recoveryYears"`
	IsRecoverable       bool                       `json:"isRecoverable"` // 家賃アップがある場合のみ true
	TaxSavings          float64                    `json:"taxSavings"`
	VirtualLaborCost    float64                    `json:"virtualLaborCost"`
	CapitalExpenditures float64                    `json:"capitalExpenditures"`
	RepairExpenses      float64                    `json:"repairExpenses"`
	ActualYield         float64                    `json:"actualYield"`
	TotalRenovationCost float64                    `json:"totalRenovationCost"`
	AnnualRentIncrease  float64                    `json:"annualRentIncrease"`
	ClassifiedItems     []ClassifiedRenovationItem `json:"classifiedItems"`
}

// RentStatsResult は賃貸取引価格の統計情報
type RentStatsResult struct {
	Median        float64 `json:"median"`         // 月額賃料 中央値（円）
	Average       float64 `json:"average"`        // 月額賃料 平均値（円）
	Count         int     `json:"count"`          // サンプル数
	LowConfidence bool    `json:"low_confidence"` // サンプル数が3件未満で信頼性低
}

// HeatmapTile はヒートマップの1タイル分の投資スコア情報
type HeatmapTile struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Z          int     `json:"z"`
	CenterLat  float64 `json:"centerLat"`
	CenterLng  float64 `json:"centerLng"`
	TotalScore int     `json:"totalScore"`
	Grade      string  `json:"grade"`
}

// HeatmapResponse はヒートマップエンドポイントのレスポンス
type HeatmapResponse struct {
	Tiles     []HeatmapTile `json:"tiles"`
	TileCount int           `json:"tileCount"`
}

// AreaDiscoveryItem は市区町村ごとの投資適性サマリー
type AreaDiscoveryItem struct {
	MunicipalityCode     string  `json:"municipalityCode"`
	MunicipalityName     string  `json:"municipalityName"`
	MedianTsubo          float64 `json:"medianTsubo"`          // 坪単価中央値（円）
	TransactionCount     int     `json:"transactionCount"`     // 取引件数
	YieldDifficulty      string  `json:"yieldDifficulty"`      // "achievable" | "slightly-difficult" | "difficult"
	YieldDifficultyLabel string  `json:"yieldDifficultyLabel"` // 日本語ラベル
	LandPriceTrend       string  `json:"landPriceTrend"`       // "上昇" | "安定" | "下落" | "不明"
	DataSufficient       bool    `json:"dataSufficient"`       // 取引件数3件以上
}

// AreaDiscoveryResponse は /api/area-discovery のレスポンス
type AreaDiscoveryResponse struct {
	Items      []AreaDiscoveryItem `json:"items"`
	Prefecture string              `json:"prefecture"`
}

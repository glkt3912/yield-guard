package domain

// BuildingType は建物構造種別を表す
type BuildingType string

const (
	BuildingTypeWood       BuildingType = "木造"    // 耐用年数 22年
	BuildingTypeLightSteel BuildingType = "軽量鉄骨" // 耐用年数 27年
	BuildingTypeHeavySteel BuildingType = "重量鉄骨" // 耐用年数 34年
	BuildingTypeRC         BuildingType = "RC造"   // 耐用年数 47年
)

// UsefulLife は建物種別の法定耐用年数を返す
func (b BuildingType) UsefulLife() int {
	switch b {
	case BuildingTypeWood:
		return 22
	case BuildingTypeLightSteel:
		return 27
	case BuildingTypeHeavySteel:
		return 34
	case BuildingTypeRC:
		return 47
	default:
		return 22
	}
}

// InvestmentInput は収支シミュレーションの入力値
type InvestmentInput struct {
	LandPrice       float64      `json:"landPrice"`       // 土地取得費 (円)
	BuildingCost    float64      `json:"buildingCost"`    // 建築費 (円)
	MiscExpenseRate float64      `json:"miscExpenseRate"` // 諸経費率 (例: 0.07)
	MonthlyRent     float64      `json:"monthlyRent"`     // 想定月額賃料 (円)
	VacancyRate     float64      `json:"vacancyRate"`     // 空室率 (例: 0.05)
	LoanAmount      float64      `json:"loanAmount"`      // ローン金額 (円)
	AnnualLoanRate  float64      `json:"annualLoanRate"`  // 年利 (例: 0.015)
	LoanYears       int          `json:"loanYears"`       // ローン期間 (年)
	BuildingType    BuildingType `json:"buildingType"`    // 建物構造
	ExpenseRate     float64      `json:"expenseRate"`     // 運営経費率 (管理費・修繕等、例: 0.20)
	IncomeTaxRate   float64      `json:"incomeTaxRate"`   // 所得税率 (例: 0.33)
	HoldingYears    int          `json:"holdingYears"`    // 出口戦略: 売却年数 (例: 10)
	ExitYieldTarget float64      `json:"exitYieldTarget"` // 出口戦略: 目標利回り (例: 0.06)

	// ストレステスト用オフセット
	VacancyRateDelta float64 `json:"vacancyRateDelta"` // 空室率上昇分 (例: +0.10)
	LoanRateDelta    float64 `json:"loanRateDelta"`    // 金利上昇分 (例: +0.015)
}

// Defaults はゼロ値フィールドにデフォルト値を設定する
func (i *InvestmentInput) Defaults() {
	if i.MiscExpenseRate == 0 {
		i.MiscExpenseRate = 0.07
	}
	if i.VacancyRate == 0 {
		i.VacancyRate = 0.05
	}
	if i.ExpenseRate == 0 {
		i.ExpenseRate = 0.20
	}
	if i.IncomeTaxRate == 0 {
		i.IncomeTaxRate = 0.33
	}
	if i.HoldingYears == 0 {
		i.HoldingYears = 10
	}
	if i.ExitYieldTarget == 0 {
		i.ExitYieldTarget = 0.06
	}
	if i.LoanYears == 0 {
		i.LoanYears = 35
	}
	if i.BuildingType == "" {
		i.BuildingType = BuildingTypeWood
	}
}

// YearlyResult は各年の収支シミュレーション結果
type YearlyResult struct {
	Year                 int     `json:"year"`
	AnnualRent           float64 `json:"annualRent"`
	AnnualLoanPayment    float64 `json:"annualLoanPayment"`
	AnnualInterest       float64 `json:"annualInterest"`
	AnnualPrincipal      float64 `json:"annualPrincipal"`
	AnnualDepreciation   float64 `json:"annualDepreciation"`
	AnnualExpenses       float64 `json:"annualExpenses"`
	TaxableIncome        float64 `json:"taxableIncome"`
	IncomeTax            float64 `json:"incomeTax"`
	CashFlow             float64 `json:"cashFlow"`
	AfterTaxCashFlow     float64 `json:"afterTaxCashFlow"`
	RemainingLoanBalance float64 `json:"remainingLoanBalance"`
	CumulativeCashFlow   float64 `json:"cumulativeCashFlow"`
	IsDeadCrossYear      bool    `json:"isDeadCrossYear"`
}

// InvestmentResult は収支シミュレーションの結果
type InvestmentResult struct {
	TotalInvestment float64 `json:"totalInvestment"`
	MiscExpenses    float64 `json:"miscExpenses"`
	GrossYield      float64 `json:"grossYield"`
	NetYield        float64 `json:"netYield"`
	IsAbove8Percent bool    `json:"isAbove8Percent"`

	RequiredLandPriceDrop    float64 `json:"requiredLandPriceDrop"`
	RequiredBuildingCostDrop float64 `json:"requiredBuildingCostDrop"`
	RequiredMonthlyRent      float64 `json:"requiredMonthlyRent"`

	DeadCrossYear int            `json:"deadCrossYear"` // -1 = デッドクロスなし
	YearlyResults []YearlyResult `json:"yearlyResults"`

	ExitSalePrice   float64 `json:"exitSalePrice"`
	ExitCapitalGain float64 `json:"exitCapitalGain"`
	ExitTransferTax float64 `json:"exitTransferTax"`
	ExitNetProceeds float64 `json:"exitNetProceeds"`
	ExitTotalEquity float64 `json:"exitTotalEquity"`
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
}

// LandPriceStats は土地価格の統計情報
type LandPriceStats struct {
	Count        int               `json:"count"`
	AverageTsubo float64           `json:"averageTsubo"`
	MedianTsubo  float64           `json:"medianTsubo"`
	MinTsubo     float64           `json:"minTsubo"`
	MaxTsubo     float64           `json:"maxTsubo"`
	Transactions []LandTransaction `json:"transactions"`
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

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
	// 運営経費率 (管理費・修繕・固定資産税・保険等。ローン利息は含まない)
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

	// 固定資産税・都市計画税（年間合計）。0 の場合は ExpenseRate に含まれる想定。
	AnnualPropertyTax float64 `json:"annualPropertyTax"`

	// 賃料下落率: 毎年この割合だけ実効賃料が低下する（例: 0.01 = 年1%下落）
	RentDeclineRate float64 `json:"rentDeclineRate"`
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
	AnnualRent           float64 `json:"annualRent"`           // 実効賃料収入（空室控除後）
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
	IsDeadCrossYear      bool    `json:"isDeadCrossYear"`   // デッドクロス初年度
	IsInDeadCrossZone    bool    `json:"isInDeadCrossZone"` // デッドクロス継続中
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
	BreakEvenYear     int     `json:"breakEvenYear"` // 累積CFが正転する年（-1=なし）
	IsSafe            bool    `json:"isSafe"`        // DSCR >= 1.0 && BreakEvenYear <= HoldingYears
}

// InvestmentResult は収支シミュレーションの結果
type InvestmentResult struct {
	TotalInvestment float64 `json:"totalInvestment"` // 総投資額（土地+建物+諸経費）
	MiscExpenses    float64 `json:"miscExpenses"`
	GrossYield      float64 `json:"grossYield"`      // 表面利回り（満室想定年収/総投資額）
	NetYield        float64 `json:"netYield"`        // 実質利回り（実効収入-経費)/総投資額）
	IsAbove8Percent bool    `json:"isAbove8Percent"`

	// 8%達成に必要な改善額（土地・建築費いずれか一方を削減する額）
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

	StressScenarios []StressScenarioResult `json:"stressScenarios"`
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

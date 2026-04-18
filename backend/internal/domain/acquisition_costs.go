package domain

import "math"

// CalcBrokerageFee は仲介手数料（税込）を算出する。
// multiplier: 1.0=標準、0.0=無料、0.5=半額など。
// 根拠: 宅地建物取引業法46条・国土交通省告示
func CalcBrokerageFee(price, multiplier float64) float64 {
	if multiplier <= 0 || price <= 0 {
		return 0
	}
	var base float64
	switch {
	case price <= 2_000_000:
		base = price * 0.05
	case price <= 4_000_000:
		base = price*0.04 + 20_000
	default:
		// 400万円超: (3% + 6万円) × 消費税10%
		base = price*0.03 + 60_000
	}
	return math.Round(base * 1.1 * multiplier)
}

// CalcStampDuty は売買契約書の印紙税を返す。
// 根拠: 印紙税法別表第一 第1号文書（不動産譲渡契約書）
// 軽減措置は2027年3月31日まで延長中だが、本則税率で保守的に試算する（国税庁・令和6年度税制改正）。
func CalcStampDuty(price float64) float64 {
	switch {
	case price <= 100_000:
		return 200
	case price <= 500_000:
		return 400
	case price <= 1_000_000:
		return 1_000
	case price <= 5_000_000:
		return 2_000
	case price <= 10_000_000:
		return 10_000
	case price <= 50_000_000:
		return 20_000
	case price <= 100_000_000:
		return 60_000
	case price <= 500_000_000:
		return 100_000
	case price <= 1_000_000_000:
		return 200_000
	default:
		return 600_000
	}
}

// CalcRegistrationTax は登録免許税合計（所有権移転登記＋抵当権設定登記）を算出する。
//
// 適用税率（根拠: 租税特別措置法・不動産登記法）:
//   - 土地所有権移転: 固定資産税評価額 × 2.0%（本則。軽減措置 〜2026/3/31 期限切れ）
//   - 建物所有権移転（中古）: 固定資産税評価額 × 2.0%
//   - 建物所有権保存（新築）: 固定資産税評価額 × 0.15%（軽減措置 〜2027/3/31）
//   - 抵当権設定: 融資額 × 0.4%（投資用物件は住宅ローン軽減0.1%対象外）
func CalcRegistrationTax(landAssessed, buildingAssessed, loanAmount float64, isNewBuilding bool) float64 {
	landTransfer := landAssessed * 0.02

	var buildingTransfer float64
	if isNewBuilding {
		// 根拠: 租税特別措置法72条の2 新築住宅用建物の所有権保存登記 軽減措置
		buildingTransfer = buildingAssessed * 0.0015
	} else {
		buildingTransfer = buildingAssessed * 0.02
	}

	mortgage := loanAmount * 0.004 // 抵当権設定: 0.4%

	return math.Round(landTransfer + buildingTransfer + mortgage)
}

// CalcRealEstateAcquisitionTax は不動産取得税の概算を算出する。
// 根拠: 地方税法73条の15、租税特別措置法11条の5（特例税率3%、〜2027/3/31）
//
// 計算式:
//   - 土地: 固定資産税評価額 × 1/2 × 3%（1/2課税は租税特別措置法11条の5第2項）
//   - 建物: 固定資産税評価額 × 3%
//
// 注意: 住宅用土地の特例控除（45,000円控除等）は建物面積等が必要なため含まない概算値。
func CalcRealEstateAcquisitionTax(landAssessed, buildingAssessed float64) float64 {
	const taxRate = 0.03 // 特例税率 〜2027/3/31
	landTax := landAssessed * 0.5 * taxRate
	buildingTax := buildingAssessed * taxRate
	return math.Round(landTax + buildingTax)
}

// AcquisitionCostOptions は諸経費計算のオプション
type AcquisitionCostOptions struct {
	// BrokerageMultiplier: 1.0=標準, 0.0=仲介手数料無料, 0.5=半額
	BrokerageMultiplier float64

	// 固定資産税評価額（0 = 推定モード: 土地×70%・建物×60%）
	AssessedLandValue     float64
	AssessedBuildingValue float64

	// LoanAmount は抵当権設定登記の計算に使用（0 = 融資なし）
	LoanAmount float64

	// IsNewBuilding は建物所有権保存登記（新築）か移転登記（中古）かを示す
	IsNewBuilding bool

	// 引渡し日（固定資産税日割り精算用）。Month=0 の場合は日割り計算なし。
	// Year は JST の取引年を呼び出し側で渡す（0 = 日割りなし）。
	DeliveryYear  int
	DeliveryMonth int
	DeliveryDay   int

	// LandAreaSqm は小規模住宅用地特例の判定に使用（0 = 特例非適用）
	LandAreaSqm float64
}

// DefaultAcquisitionCostOptions は標準的な取引条件のオプションを返す
// 評価額は推定モード（取得価格ベース）、融資なし想定
func DefaultAcquisitionCostOptions() AcquisitionCostOptions {
	return AcquisitionCostOptions{BrokerageMultiplier: 1.0}
}

// CalcAcquisitionCosts は取得時の諸経費を算出する。
// 評価額が未入力（0）の場合は推定モード（土地: landPrice×70%、建物: buildingCost×60%）を使用する。
func CalcAcquisitionCosts(landPrice, buildingCost float64, opts AcquisitionCostOptions) AcquisitionCostBreakdown {
	totalPrice := landPrice + buildingCost
	brokerage := CalcBrokerageFee(totalPrice, opts.BrokerageMultiplier)
	stamp := CalcStampDuty(totalPrice)

	assessedLand := opts.AssessedLandValue
	if assessedLand == 0 {
		assessedLand = landPrice * 0.7
	}
	assessedBuilding := opts.AssessedBuildingValue
	if assessedBuilding == 0 {
		assessedBuilding = buildingCost * 0.6
	}

	regTax := CalcRegistrationTax(assessedLand, assessedBuilding, opts.LoanAmount, opts.IsNewBuilding)
	acqTax := CalcRealEstateAcquisitionTax(assessedLand, assessedBuilding)

	propTaxProration := 0.0
	if opts.DeliveryYear > 0 && opts.DeliveryMonth > 0 && opts.DeliveryDay > 0 {
		annual := CalcPropertyTax(assessedLand, assessedBuilding, PropertyTaxOptions{
			LandAreaSqm: opts.LandAreaSqm,
		}).AnnualTotal
		propTaxProration = CalcPropertyTaxProration(annual, opts.DeliveryMonth, opts.DeliveryDay, opts.DeliveryYear)
	}

	return AcquisitionCostBreakdown{
		BrokerageFee:             brokerage,
		StampDuty:                stamp,
		RegistrationTax:          regTax,
		RealEstateAcquisitionTax: acqTax,
		PropertyTaxProration:     propTaxProration,
		Total:                    brokerage + stamp + regTax + acqTax + propTaxProration,
	}
}

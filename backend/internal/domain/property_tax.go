package domain

import "math"

// PropertyTaxOptions は固定資産税・都市計画税の計算オプション
type PropertyTaxOptions struct {
	// LandAreaSqm は土地面積（㎡）。小規模住宅用地特例の適用判定に使用。
	// 0 の場合は特例を適用しない（保守的な概算）。
	LandAreaSqm float64
}

// PropertyTaxBreakdown は固定資産税・都市計画税の内訳
type PropertyTaxBreakdown struct {
	LandFixedAssetTax     float64 `json:"landFixedAssetTax"`     // 土地の固定資産税
	BuildingFixedAssetTax float64 `json:"buildingFixedAssetTax"` // 建物の固定資産税
	LandCityPlanningTax   float64 `json:"landCityPlanningTax"`   // 土地の都市計画税
	BuildingCityTax       float64 `json:"buildingCityTax"`       // 建物の都市計画税
	AnnualTotal           float64 `json:"annualTotal"`           // 年間合計
}

// CalcPropertyTax は年間の固定資産税・都市計画税を算出する。
//
// 税率（根拠: 地方税法349条・702条）:
//   - 固定資産税: 固定資産税評価額 × 1.4%（標準税率）
//   - 都市計画税: 固定資産税評価額 × 0.3%（上限税率）
//
// 小規模住宅用地特例（根拠: 地方税法349条の3の2）:
//   - 土地面積200㎡以下の小規模住宅用地: 固定資産税 1/6・都市計画税 1/3
//   - 200㎡超の一般住宅用地: 固定資産税 1/3・都市計画税 2/3
//
// 注意: LandAreaSqm が 0 の場合は特例を適用しない概算値。
func CalcPropertyTax(assessedLand, assessedBuilding float64, opts PropertyTaxOptions) PropertyTaxBreakdown {
	const (
		fixedAssetTaxRate  = 0.014 // 固定資産税: 標準税率
		cityPlanningTaxRate = 0.003 // 都市計画税: 上限税率
	)

	landFixed := assessedLand * fixedAssetTaxRate
	landCity := assessedLand * cityPlanningTaxRate

	// 小規模住宅用地特例の適用
	if opts.LandAreaSqm > 0 {
		if opts.LandAreaSqm <= 200 {
			// 小規模住宅用地（200㎡以下）
			landFixed = assessedLand * fixedAssetTaxRate / 6
			landCity = assessedLand * cityPlanningTaxRate / 3
		} else {
			// 一般住宅用地（200㎡超）
			landFixed = assessedLand * fixedAssetTaxRate / 3
			landCity = assessedLand * cityPlanningTaxRate * 2 / 3
		}
	}

	buildingFixed := assessedBuilding * fixedAssetTaxRate
	buildingCity := assessedBuilding * cityPlanningTaxRate

	landFixed = math.Round(landFixed)
	landCity = math.Round(landCity)
	buildingFixed = math.Round(buildingFixed)
	buildingCity = math.Round(buildingCity)

	return PropertyTaxBreakdown{
		LandFixedAssetTax:     landFixed,
		BuildingFixedAssetTax: buildingFixed,
		LandCityPlanningTax:   landCity,
		BuildingCityTax:       buildingCity,
		AnnualTotal:           landFixed + buildingFixed + landCity + buildingCity,
	}
}

// CalcPropertyTaxProration は引渡し日基準で買主負担分の固定資産税日割り額を算出する。
//
// 慣行: 売主が1月1日〜引渡し前日分、買主が引渡し日〜12月31日分を負担。
// うるう年は非考慮（誤差軽微のため365日固定）。
func CalcPropertyTaxProration(annualTax float64, deliveryMonth, deliveryDay int) float64 {
	if annualTax <= 0 || deliveryMonth < 1 || deliveryMonth > 12 || deliveryDay < 1 {
		return 0
	}
	// 引渡し日の通算日数（1月1日=1）、うるう年非考慮
	daysInMonth := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	deliveryDOY := deliveryDay
	for m := 0; m < deliveryMonth-1; m++ {
		deliveryDOY += daysInMonth[m]
	}
	// 買主負担日数 = 12月31日(365) - 引渡し日 + 1
	buyerDays := 365 - deliveryDOY + 1
	if buyerDays <= 0 {
		return 0
	}
	return math.Round(annualTax * float64(buyerDays) / 365)
}

package domain

import (
	"math"
	"sort"
	"time"
)

const (
	ageCorrectionRate    = 0.02 // 築年数1年あたりの補正率
	stationCorrRate      = 0.01 // 駅距離1分あたりの補正率
	ageCorrectionMax     = 0.30 // 築年数補正の上限 (±30%)
	stationCorrectionMax = 0.20 // 駅距離補正の上限 (±20%)
)

// TheoreticalPriceInput は理論価格推定の入力値
type TheoreticalPriceInput struct {
	ListingPrice    float64              // 販売価格 (円)
	LandArea        float64              // 土地面積 (m²)
	BuildingAge     int                  // 物件築年数
	StationMinutes  int                  // 最寄り駅徒歩分 (0=未入力)
	RidershipScore  RidershipDemandScore // 最寄り駅需要スコア (空文字=未入力)
}

// EstimateTheoreticalPrice は取引事例の中央値と補正係数から理論価格を推定する。
// 取引事例が不足またはデータ不整合の場合は (TheoreticalPriceResult{}, false) を返す。
//
// 補正式:
//   AgeCorrection        = clamp(-0.02 × (buildingAge - medianAge), -0.3, 0.3)
//   StationCorrection    = clamp(-0.01 × (stationMin - medianStation), -0.2, 0.2)
//   RidershipCorrection  = RidershipCorrectionFactor(score) if score != ""
//   TheoreticalPrice     = medianTsubo × (1+AgeCorr) × (1+StationCorr) × (1+RidershipCorr) × (area/SqmPerTsubo)
func EstimateTheoreticalPrice(stats LandPriceStats, input TheoreticalPriceInput) (TheoreticalPriceResult, bool) {
	if stats.MedianTsubo == 0 || input.LandArea <= 0 || input.ListingPrice <= 0 {
		return TheoreticalPriceResult{}, false
	}

	currentYear := time.Now().Year()

	var buildingAges []int
	var stationMins []int
	for _, t := range stats.Transactions {
		if t.BuildingYear > 0 {
			age := currentYear - t.BuildingYear
			if age >= 0 && age <= 100 {
				buildingAges = append(buildingAges, age)
			}
		}
		if t.StationMinutes > 0 && t.StationMinutes <= 120 {
			stationMins = append(stationMins, t.StationMinutes)
		}
	}

	if len(buildingAges) == 0 {
		return TheoreticalPriceResult{}, false
	}

	medianAge := medianInt(buildingAges)
	medianStation := medianInt(stationMins)

	ageCorrection := clamp(
		-ageCorrectionRate*float64(input.BuildingAge-medianAge),
		-ageCorrectionMax, ageCorrectionMax,
	)

	hasStationData := len(stationMins) > 0 && input.StationMinutes > 0
	stationCorrection := 0.0
	if hasStationData {
		stationCorrection = clamp(
			-stationCorrRate*float64(input.StationMinutes-medianStation),
			-stationCorrectionMax, stationCorrectionMax,
		)
	}

	hasRidershipData := input.RidershipScore != ""
	ridershipCorrection := 0.0
	if hasRidershipData {
		ridershipCorrection = RidershipCorrectionFactor(input.RidershipScore)
	}

	landAreaTsubo := input.LandArea / SqmPerTsubo
	theoreticalPrice := stats.MedianTsubo * (1 + ageCorrection) * (1 + stationCorrection) * (1 + ridershipCorrection) * landAreaTsubo
	deviationPct := (input.ListingPrice - theoreticalPrice) / theoreticalPrice * 100

	return TheoreticalPriceResult{
		TheoreticalPriceJPY:  theoreticalPrice,
		DeviationPct:         deviationPct,
		AgeCorrection:        ageCorrection,
		StationCorrection:    stationCorrection,
		RidershipCorrection:  ridershipCorrection,
		MedianBuildingAge:    medianAge,
		MedianStationMinutes: medianStation,
		IsLowDataWarning:     len(buildingAges) < 10,
		HasStationData:       hasStationData,
		RidershipScore:       input.RidershipScore,
		HasRidershipData:     hasRidershipData,
	}, true
}

func medianInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]int, len(vals))
	copy(sorted, vals)
	sort.Ints(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func clamp(v, min, max float64) float64 {
	return math.Max(min, math.Min(max, v))
}

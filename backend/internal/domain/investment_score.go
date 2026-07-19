package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// InvestmentScoreInput は CalcInvestmentScore への入力パラメータ
type InvestmentScoreInput struct {
	PopulationItems      []PopulationForecastItem
	StationRiderships    []StationRidershipResult
	LocationItems        []LocationOptimizationItem
	EmbankmentItems      []EmbankmentItem
	DisasterItems        []DisasterHistoryItem
	UrbanZoningItems     []UrbanZoningItem
	LiquefactionItems    []LiquefactionRiskItem
	FloodItems           []FloodHazardItem
	StormItems           []StormHazardItem
	TsunamiItems         []TsunamiHazardItem
	LandslideItems       []LandslideHazardItem
	LandPriceChangeRate  float64 // 坪単価の変化率（HasLandPriceTrend=false の場合は未使用）
	HasLandPriceTrend    bool    // true のときのみ地価トレンドスコアを加算
}

// CalcInvestmentScore は複数 MLIT API の結果を統合して投資適地スコアを算出する。
// baseScore=50 を起点に各指標を加減し、0〜100 にクランプして返す。
func CalcInvestmentScore(input InvestmentScoreInput) InvestmentScoreResult {
	pop := calcPopulationScore(input.PopulationItems)
	ridership := calcRidershipScore(input.StationRiderships)
	urbanArea := calcUrbanAreaScore(input.UrbanZoningItems)
	locationOpt := calcLocationOptScore(input.LocationItems)
	hazard := calcHazardScore(input.FloodItems, input.StormItems, input.TsunamiItems, input.LandslideItems)
	liquefaction := calcLiquefactionScore(input.LiquefactionItems)
	embankment := calcEmbankmentScore(input.EmbankmentItems)
	disaster := calcDisasterScore(input.DisasterItems)
	landTrend := calcLandPriceTrendScore(input.LandPriceChangeRate, input.HasLandPriceTrend)

	total := 50 + pop.Score + ridership.Score + urbanArea.Score + locationOpt.Score +
		hazard.Score + liquefaction.Score + embankment.Score + disaster.Score + landTrend.Score
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}

	return InvestmentScoreResult{
		TotalScore: total,
		Grade:      gradeFromScore(total),
		Breakdown: ScoreBreakdown{
			Population:       pop,
			Ridership:        ridership,
			UrbanArea:        urbanArea,
			LocationOpt:      locationOpt,
			HazardRisk:       hazard,
			LiquefactionRisk: liquefaction,
			Embankment:       embankment,
			DisasterHistory:  disaster,
			LandPriceTrend:   landTrend,
			RadarData:        buildRadarData(pop.Score, ridership.Score, urbanArea.Score, locationOpt.Score, hazard.Score, liquefaction.Score, embankment.Score, disaster.Score),
		},
	}
}

// gradeFromScore は総合スコアから評価コードを返す。
// 戻り値は "excellent" | "good" | "average" | "caution" | "warning"。
// 日本語表示ラベルへの変換は呼び出し側（service 層）が担う。
func gradeFromScore(score int) string {
	switch {
	case score >= 80:
		return "excellent"
	case score >= 65:
		return "good"
	case score >= 50:
		return "average"
	case score >= 35:
		return "caution"
	default:
		return "warning"
	}
}

// calcPopulationScore は XKT013 の人口予測から ±15 点を算出する。
// changeRate30yr が +30%→+15, 0%→0, -50%以下→-15 で線形補間。
func calcPopulationScore(items []PopulationForecastItem) ScoreItem {
	if len(items) == 0 {
		return ScoreItem{Score: 0, Label: "人口動態", Description: "データなし"}
	}
	result := CalcPopulationForecast(items)
	r := result.ChangeRate30yr

	var raw float64
	if r >= 0 {
		raw = math.Min(r/0.30, 1.0) * 15
	} else {
		raw = math.Max(r/0.50, -1.0) * 15
	}
	score := int(math.Round(raw))

	desc := fmt.Sprintf("%s（30年後%+.0f%%）", result.Trend, r*100)
	return ScoreItem{Score: score, Label: "人口動態", Description: desc}
}

// calcRidershipScore は XKT015 の最大乗降客数から 0〜+20 点を算出する。
// 20万人/日以上で満点。
func calcRidershipScore(riderships []StationRidershipResult) ScoreItem {
	if len(riderships) == 0 {
		return ScoreItem{Score: 0, Label: "交通利便性", Description: "駅データなし"}
	}
	var maxPassengers int
	var topStation string
	for _, s := range riderships {
		if s.Passengers > maxPassengers {
			maxPassengers = s.Passengers
			topStation = s.StationName
		}
	}
	raw := float64(maxPassengers) / 200_000.0 * 20.0
	score := int(math.Min(math.Round(raw), 20))
	desc := fmt.Sprintf("%s %s人/日", topStation, formatPassengers(maxPassengers))
	return ScoreItem{Score: score, Label: "交通利便性", Description: desc}
}

// calcUrbanAreaScore は XKT001 の区域区分から +10 or 0 点を算出する。
func calcUrbanAreaScore(items []UrbanZoningItem) ScoreItem {
	if len(items) == 0 {
		return ScoreItem{Score: 0, Label: "市街化区域", Description: "都市計画区域データなし"}
	}
	for _, item := range items {
		if strings.Contains(item.AreaClassificationJa, "市街化区域") &&
			!strings.Contains(item.AreaClassificationJa, "調整") {
			return ScoreItem{Score: 10, Label: "市街化区域", Description: "市街化区域内"}
		}
	}
	return ScoreItem{Score: 0, Label: "市街化区域", Description: "市街化区域外"}
}

// calcLocationOptScore は XKT003 の居住誘導区域から +10 or -5 or 0 点を算出する。
// フィーチャなし（未策定自治体）→ 0、居住誘導区域内 → +10、区域外 → -5
func calcLocationOptScore(items []LocationOptimizationItem) ScoreItem {
	if len(items) == 0 {
		return ScoreItem{Score: 0, Label: "居住誘導区域", Description: "立地適正化計画未策定"}
	}
	for _, item := range items {
		if strings.Contains(item.KubunNameJa, "居住誘導区域") {
			return ScoreItem{Score: 10, Label: "居住誘導区域", Description: "居住誘導区域内"}
		}
	}
	return ScoreItem{Score: -5, Label: "居住誘導区域", Description: "居住誘導区域外（将来の行政サービス縮小リスク）"}
}

// calcHazardScore は XKT026〜029 のハザードリスクを合算し 0〜-20 点を算出する。
func calcHazardScore(flood []FloodHazardItem, storm []StormHazardItem, tsunami []TsunamiHazardItem, landslide []LandslideHazardItem) ScoreItem {
	total := 0
	var risks []string

	// 洪水リスク（XKT026）
	if len(flood) > 0 {
		maxRank := 0
		for _, f := range flood {
			if f.DepthRank > maxRank {
				maxRank = f.DepthRank
			}
		}
		if maxRank >= 3 {
			total -= 5
			risks = append(risks, "洪水（深）")
		} else if maxRank >= 1 {
			total -= 3
			risks = append(risks, "洪水")
		}
	}

	// 高潮リスク（XKT027）
	if len(storm) > 0 {
		total -= 5
		risks = append(risks, "高潮")
	}

	// 津波リスク（XKT028）
	if len(tsunami) > 0 {
		total -= 5
		risks = append(risks, "津波")
	}

	// 土砂リスク（XKT029）
	if len(landslide) > 0 {
		minZone := math.MaxInt
		for _, l := range landslide {
			if l.ZoneCode < minZone {
				minZone = l.ZoneCode
			}
		}
		if minZone == 1 {
			total -= 5
			risks = append(risks, "土砂（特別警戒）")
		} else {
			total -= 3
			risks = append(risks, "土砂（警戒）")
		}
	}

	// 最大 -20 にクランプ
	if total < -20 {
		total = -20
	}

	var desc string
	if len(risks) == 0 {
		desc = "ハザードリスクなし"
	} else {
		desc = strings.Join(risks, "・") + "リスクあり"
	}
	return ScoreItem{Score: total, Label: "ハザードリスク", Description: desc}
}

// calcLiquefactionScore は XKT025 の液状化発生傾向から 0〜-10 点を算出する。
// TendencyLevel: 6段階（低値ほど液状化しやすい）
func calcLiquefactionScore(items []LiquefactionRiskItem) ScoreItem {
	if len(items) == 0 {
		return ScoreItem{Score: 0, Label: "液状化リスク", Description: "液状化データなし"}
	}
	minLevel := 9
	for _, item := range items {
		if item.TendencyLevel < minLevel {
			minLevel = item.TendencyLevel
		}
	}
	switch {
	case minLevel <= 2:
		return ScoreItem{Score: -10, Label: "液状化リスク", Description: "液状化リスク：非常に高い"}
	case minLevel <= 4:
		return ScoreItem{Score: -5, Label: "液状化リスク", Description: "液状化リスク：中程度"}
	default:
		return ScoreItem{Score: 0, Label: "液状化リスク", Description: "液状化リスク：低い"}
	}
}

// calcEmbankmentScore は XKT020 の大規模盛土造成地から 0 or -5 点を算出する。
func calcEmbankmentScore(items []EmbankmentItem) ScoreItem {
	if len(items) == 0 {
		return ScoreItem{Score: 0, Label: "大規模盛土", Description: "大規模盛土造成地に非該当"}
	}
	desc := "大規模盛土造成地に該当"
	if c := items[0].Classification; c != "" {
		desc = fmt.Sprintf("大規模盛土造成地（%s）", c)
	}
	return ScoreItem{Score: -5, Label: "大規模盛土", Description: desc}
}

// calcDisasterScore は XST001 の災害履歴から年数に応じた段階評価を算出する。
// 10年以内: -10、30年以内: -5、30年超: -2。年不明: -10（最悪ケース）。
// 複数履歴がある場合は最も厳しいスコアを採用する。
func calcDisasterScore(items []DisasterHistoryItem) ScoreItem {
	if len(items) == 0 {
		return ScoreItem{Score: 0, Label: "災害履歴", Description: "災害履歴なし"}
	}
	currentYear := time.Now().Year()
	minScore := 0
	names := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, d := range items {
		if d.Name != "" && !seen[d.Name] {
			names = append(names, d.Name)
			seen[d.Name] = true
		}
		s := disasterScoreByYear(d.Year, currentYear)
		if s < minScore {
			minScore = s
		}
	}
	var desc string
	if len(names) > 0 {
		desc = fmt.Sprintf("過去の災害記録あり（%s）", strings.Join(names, "・"))
	} else {
		desc = "過去の災害記録あり"
	}
	return ScoreItem{Score: minScore, Label: "災害履歴", Description: desc}
}

// disasterScoreByYear は発生年から年数重み付けスコアを返す。
func disasterScoreByYear(year, currentYear int) int {
	if year == 0 {
		return -10
	}
	yearsAgo := currentYear - year
	switch {
	case yearsAgo <= 10:
		return -10
	case yearsAgo <= 30:
		return -5
	default:
		return -2
	}
}

// buildRadarData はレーダーチャート用の5カテゴリ正規化スコアを生成する（0〜100）。
// disaster スコアはハザードカテゴリに合算し、総合スコアとレーダーの乖離を防ぐ。
func buildRadarData(pop, ridership, urbanArea, locationOpt, hazard, liquefaction, embankment, disaster int) []RadarPoint {
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 100 {
			return 100
		}
		return v
	}

	popNorm := clamp((float64(pop) + 15) / 30.0 * 100)
	ridershipNorm := clamp(float64(ridership) / 20.0 * 100)
	urbanNorm := clamp((float64(urbanArea+locationOpt) + 5) / 25.0 * 100)
	// ハザード＋災害履歴の合算: 最良=0+0=0、最悪=-20+(-10)=-30 → 分母30で正規化
	hazardNorm := clamp((30.0 + float64(hazard) + float64(disaster)) / 30.0 * 100)
	groundNorm := clamp((15.0 + float64(liquefaction+embankment)) / 15.0 * 100)

	return []RadarPoint{
		{Category: "人口動態", Score: math.Round(popNorm)},
		{Category: "交通利便性", Score: math.Round(ridershipNorm)},
		{Category: "都市計画", Score: math.Round(urbanNorm)},
		{Category: "ハザード", Score: math.Round(hazardNorm)},
		{Category: "地盤", Score: math.Round(groundNorm)},
	}
}

func formatPassengers(n int) string {
	if n >= 10_000 {
		return fmt.Sprintf("%.1f万", float64(n)/10_000)
	}
	return fmt.Sprintf("%d", n)
}

// calcLandPriceTrendScore は坪単価変化率から ±10 点を算出する。
// HasLandPriceTrend が false の場合はデータなし扱いで 0 点を返す。
func calcLandPriceTrendScore(changeRate float64, hasData bool) ScoreItem {
	if !hasData {
		return ScoreItem{Score: 0, Label: "地価トレンド", Description: "地価データなし"}
	}
	pct := changeRate * 100
	var score int
	switch {
	case pct > 10:
		score = 10
	case pct > 5:
		score = 5
	case pct >= -5:
		score = 0
	case pct >= -10:
		score = -5
	default:
		score = -10
	}
	var trend string
	switch {
	case pct > 0:
		trend = "上昇"
	case pct < 0:
		trend = "下落"
	default:
		trend = "横ばい"
	}
	return ScoreItem{Score: score, Label: "地価トレンド", Description: fmt.Sprintf("坪単価%s（約%+.1f%%）", trend, pct)}
}

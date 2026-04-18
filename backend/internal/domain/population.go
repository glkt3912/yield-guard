package domain

// PopulationTrend はエリアの人口動向を4段階で分類する
type PopulationTrend string

const (
	PopulationTrendGrowth       PopulationTrend = "増加"
	PopulationTrendStable       PopulationTrend = "現状維持"
	PopulationTrendSlowDecline  PopulationTrend = "緩やかな減少"
	PopulationTrendSteepDecline PopulationTrend = "急激な減少"
)

// PopulationForecastItem は年別人口推計の1エントリ（mlit → domain橋渡し用）
type PopulationForecastItem struct {
	Year int
	Pop  float64
}

// PopulationSnapshot は特定年の人口推計値
type PopulationSnapshot struct {
	Year int     `json:"year"`
	Pop  float64 `json:"pop"`
}

// PopulationForecastResult はエリアの将来人口推計まとめ
type PopulationForecastResult struct {
	Snapshots        []PopulationSnapshot `json:"snapshots"`
	ChangeRate30yr   float64              `json:"changeRate30yr"`   // 30年後変化率 (例: -0.32)
	VacancyRateDelta float64              `json:"vacancyRateDelta"` // 推定空室率増加幅
	Trend            PopulationTrend      `json:"trend"`
}

// CalcPopulationForecast は XKT013 から取得した年別人口をもとに推計結果を算出する。
// 空室率増加幅: max(0, -changeRate * 0.5)
// 例: 30年で -32% → vacancyRateDelta = +0.16
func CalcPopulationForecast(items []PopulationForecastItem) PopulationForecastResult {
	snapshots := make([]PopulationSnapshot, 0, len(items))
	for _, it := range items {
		snapshots = append(snapshots, PopulationSnapshot(it))
	}

	var base, future float64
	for _, it := range items {
		if it.Year == 2020 {
			base = it.Pop
		}
		if it.Year == 2050 {
			future = it.Pop
		}
	}

	var changeRate float64
	if base > 0 {
		changeRate = (future - base) / base
	}

	var vacancyDelta float64
	if changeRate < 0 {
		vacancyDelta = -changeRate * 0.5
	}

	return PopulationForecastResult{
		Snapshots:        snapshots,
		ChangeRate30yr:   changeRate,
		VacancyRateDelta: vacancyDelta,
		Trend:            classifyTrend(changeRate),
	}
}

func classifyTrend(r float64) PopulationTrend {
	switch {
	case r > 0:
		return PopulationTrendGrowth
	case r >= -0.05:
		return PopulationTrendStable
	case r >= -0.20:
		return PopulationTrendSlowDecline
	default:
		return PopulationTrendSteepDecline
	}
}

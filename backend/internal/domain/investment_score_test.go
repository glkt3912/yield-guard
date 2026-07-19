package domain

import (
	"testing"
	"time"
)

// --- gradeFromScore ---

func TestGradeFromScore_Boundaries(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "excellent"},
		{80, "excellent"},
		{79, "good"},
		{65, "good"},
		{64, "average"},
		{50, "average"},
		{49, "caution"},
		{35, "caution"},
		{34, "warning"},
		{0, "warning"},
	}
	for _, tc := range tests {
		got := gradeFromScore(tc.score)
		if got != tc.want {
			t.Errorf("gradeFromScore(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

// --- calcPopulationScore ---

func TestCalcPopulationScore_Empty(t *testing.T) {
	s := calcPopulationScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
	if s.Description != "データなし" {
		t.Errorf("description = %q, want 'データなし'", s.Description)
	}
}

func TestCalcPopulationScore_Growth30pct(t *testing.T) {
	// CalcPopulationForecast は Year=2020 を base、Year=2050 を future として変化率を計算する
	// 2020=100, 2050=130 → +30% → スコア +15
	items := []PopulationForecastItem{
		{Year: 2020, Pop: 100},
		{Year: 2050, Pop: 130},
	}
	s := calcPopulationScore(items)
	if s.Score != 15 {
		t.Errorf("score = %d, want 15", s.Score)
	}
}

func TestCalcPopulationScore_Decline50pct(t *testing.T) {
	// 2020=100, 2050=50 → -50% → スコア -15
	items := []PopulationForecastItem{
		{Year: 2020, Pop: 100},
		{Year: 2050, Pop: 50},
	}
	s := calcPopulationScore(items)
	if s.Score != -15 {
		t.Errorf("score = %d, want -15", s.Score)
	}
}

func TestCalcPopulationScore_NoChange(t *testing.T) {
	// 変化なし → スコア 0
	items := []PopulationForecastItem{
		{Year: 2020, Pop: 100},
		{Year: 2050, Pop: 100},
	}
	s := calcPopulationScore(items)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

// --- calcRidershipScore ---

func TestCalcRidershipScore_Empty(t *testing.T) {
	s := calcRidershipScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
	if s.Description != "駅データなし" {
		t.Errorf("description = %q, want '駅データなし'", s.Description)
	}
}

func TestCalcRidershipScore_FullScore(t *testing.T) {
	// 200,000 人/日 → 上限 +20
	s := calcRidershipScore([]StationRidershipResult{
		{StationName: "渋谷", Passengers: 200_000},
	})
	if s.Score != 20 {
		t.Errorf("score = %d, want 20", s.Score)
	}
}

func TestCalcRidershipScore_CappedAt20(t *testing.T) {
	// 200,000 超でも 20 を超えない
	s := calcRidershipScore([]StationRidershipResult{
		{StationName: "新宿", Passengers: 500_000},
	})
	if s.Score != 20 {
		t.Errorf("score = %d, want 20 (capped)", s.Score)
	}
}

func TestCalcRidershipScore_BestStationChosen(t *testing.T) {
	// 複数駅のうち最大乗降客数の駅が採用される
	s := calcRidershipScore([]StationRidershipResult{
		{StationName: "A駅", Passengers: 50_000},
		{StationName: "B駅", Passengers: 200_000},
	})
	if s.Score != 20 {
		t.Errorf("score = %d, want 20", s.Score)
	}
}

// --- calcUrbanAreaScore ---

func TestCalcUrbanAreaScore_Empty(t *testing.T) {
	s := calcUrbanAreaScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

func TestCalcUrbanAreaScore_InMarketArea(t *testing.T) {
	// 市街化区域 → +10
	s := calcUrbanAreaScore([]UrbanZoningItem{
		{AreaClassificationJa: "市街化区域"},
	})
	if s.Score != 10 {
		t.Errorf("score = %d, want 10", s.Score)
	}
}

func TestCalcUrbanAreaScore_AdjustedArea(t *testing.T) {
	// 市街化調整区域（"調整" を含む） → 0
	s := calcUrbanAreaScore([]UrbanZoningItem{
		{AreaClassificationJa: "市街化調整区域"},
	})
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

// --- calcLocationOptScore ---

func TestCalcLocationOptScore_NotPlanned(t *testing.T) {
	// 立地適正化計画未策定（空） → 0
	s := calcLocationOptScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

func TestCalcLocationOptScore_InResidenceZone(t *testing.T) {
	// 居住誘導区域内 → +10
	s := calcLocationOptScore([]LocationOptimizationItem{
		{KubunNameJa: "居住誘導区域"},
	})
	if s.Score != 10 {
		t.Errorf("score = %d, want 10", s.Score)
	}
}

func TestCalcLocationOptScore_OutsideZone(t *testing.T) {
	// 策定済みだが居住誘導区域外 → -5
	s := calcLocationOptScore([]LocationOptimizationItem{
		{KubunNameJa: "都市機能誘導区域"},
	})
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

// --- calcHazardScore ---

func TestCalcHazardScore_AllEmpty(t *testing.T) {
	s := calcHazardScore(nil, nil, nil, nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

func TestCalcHazardScore_FloodHighRank(t *testing.T) {
	// 浸水深ランク >= 3 → -5
	s := calcHazardScore([]FloodHazardItem{{DepthRank: 3}}, nil, nil, nil)
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

func TestCalcHazardScore_FloodLowRank(t *testing.T) {
	// 浸水深ランク 1 → -3
	s := calcHazardScore([]FloodHazardItem{{DepthRank: 1}}, nil, nil, nil)
	if s.Score != -3 {
		t.Errorf("score = %d, want -3", s.Score)
	}
}

func TestCalcHazardScore_Storm(t *testing.T) {
	// 高潮リスク → -5
	s := calcHazardScore(nil, []StormHazardItem{{DepthJa: "0.5m未満"}}, nil, nil)
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

func TestCalcHazardScore_Tsunami(t *testing.T) {
	// 津波リスク → -5
	s := calcHazardScore(nil, nil, []TsunamiHazardItem{{DepthJa: "0.3m未満"}}, nil)
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

func TestCalcHazardScore_LandslideSpecialZone(t *testing.T) {
	// 特別警戒区域（ZoneCode=1）→ -5
	s := calcHazardScore(nil, nil, nil, []LandslideHazardItem{{ZoneCode: 1}})
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

func TestCalcHazardScore_LandslideWarningZone(t *testing.T) {
	// 警戒区域（ZoneCode=2）→ -3
	s := calcHazardScore(nil, nil, nil, []LandslideHazardItem{{ZoneCode: 2}})
	if s.Score != -3 {
		t.Errorf("score = %d, want -3", s.Score)
	}
}

func TestCalcHazardScore_ClampedToMinus20(t *testing.T) {
	// 全ハザード重複 → -20 を下回らない
	flood := []FloodHazardItem{{DepthRank: 5}}
	storm := []StormHazardItem{{DepthJa: "1m"}}
	tsunami := []TsunamiHazardItem{{DepthJa: "1m"}}
	landslide := []LandslideHazardItem{{ZoneCode: 1}}
	s := calcHazardScore(flood, storm, tsunami, landslide)
	if s.Score < -20 {
		t.Errorf("score %d is below minimum -20", s.Score)
	}
}

// --- calcLiquefactionScore ---

func TestCalcLiquefactionScore_Empty(t *testing.T) {
	s := calcLiquefactionScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

func TestCalcLiquefactionScore_HighRisk(t *testing.T) {
	// TendencyLevel <= 2 → -10
	s := calcLiquefactionScore([]LiquefactionRiskItem{{TendencyLevel: 1}})
	if s.Score != -10 {
		t.Errorf("score = %d, want -10", s.Score)
	}
}

func TestCalcLiquefactionScore_MediumRisk(t *testing.T) {
	// TendencyLevel 3-4 → -5
	s := calcLiquefactionScore([]LiquefactionRiskItem{{TendencyLevel: 3}})
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

func TestCalcLiquefactionScore_LowRisk(t *testing.T) {
	// TendencyLevel > 4 → 0
	s := calcLiquefactionScore([]LiquefactionRiskItem{{TendencyLevel: 5}})
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

// --- calcEmbankmentScore ---

func TestCalcEmbankmentScore_Empty(t *testing.T) {
	// 大規模盛土造成地に非該当 → 0
	s := calcEmbankmentScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

func TestCalcEmbankmentScore_HasItem(t *testing.T) {
	// 該当あり → -5
	s := calcEmbankmentScore([]EmbankmentItem{{Classification: "谷埋め型"}})
	if s.Score != -5 {
		t.Errorf("score = %d, want -5", s.Score)
	}
}

// --- disasterScoreByYear ---

func TestDisasterScoreByYear_Table(t *testing.T) {
	cur := time.Now().Year()
	tests := []struct {
		label string
		year  int
		want  int
	}{
		{"年不明(0)", 0, -10},
		{"5年前", cur - 5, -10},
		{"ちょうど10年前", cur - 10, -10},
		{"11年前", cur - 11, -5},
		{"ちょうど30年前", cur - 30, -5},
		{"31年前", cur - 31, -2},
		{"50年前", cur - 50, -2},
	}
	for _, tc := range tests {
		got := disasterScoreByYear(tc.year, cur)
		if got != tc.want {
			t.Errorf("[%s] disasterScoreByYear(%d, %d) = %d, want %d",
				tc.label, tc.year, cur, got, tc.want)
		}
	}
}

func TestCalcDisasterScore_Empty(t *testing.T) {
	s := calcDisasterScore(nil)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0", s.Score)
	}
}

func TestCalcDisasterScore_RecentDisaster(t *testing.T) {
	cur := time.Now().Year()
	s := calcDisasterScore([]DisasterHistoryItem{
		{Name: "浸水域", Year: cur - 3},
	})
	if s.Score != -10 {
		t.Errorf("score = %d, want -10", s.Score)
	}
}

func TestCalcDisasterScore_Deduplication(t *testing.T) {
	// 同名の災害は重複カウントしない
	cur := time.Now().Year()
	items := []DisasterHistoryItem{
		{Name: "浸水域", Year: cur - 3},
		{Name: "浸水域", Year: cur - 3}, // 重複
	}
	s1 := calcDisasterScore(items[:1])
	s2 := calcDisasterScore(items)
	if s1.Score != s2.Score {
		t.Errorf("duplicate disaster items changed score: s1=%d s2=%d", s1.Score, s2.Score)
	}
}

// --- CalcInvestmentScore 統合 ---

func TestCalcInvestmentScore_AllEmpty_BaseScore(t *testing.T) {
	// 全データ空 → 加減算なしで baseScore=50 が返る
	result := CalcInvestmentScore(InvestmentScoreInput{})
	if result.TotalScore != 50 {
		t.Errorf("TotalScore = %d, want 50 (base score)", result.TotalScore)
	}
	if result.Grade == "" {
		t.Error("Grade must not be empty")
	}
}

func TestCalcInvestmentScore_ScoreNotExceed100(t *testing.T) {
	// 全有利条件でも 100 を超えない
	input := InvestmentScoreInput{
		StationRiderships: []StationRidershipResult{
			{StationName: "渋谷", Passengers: 500_000},
		},
		UrbanZoningItems: []UrbanZoningItem{
			{AreaClassificationJa: "市街化区域"},
		},
		LocationItems: []LocationOptimizationItem{
			{KubunNameJa: "居住誘導区域"},
		},
		HasLandPriceTrend:   true,
		LandPriceChangeRate: 0.20,
		PopulationItems: []PopulationForecastItem{
			{Year: 2020, Pop: 100},
			{Year: 2050, Pop: 130},
		},
	}
	result := CalcInvestmentScore(input)
	if result.TotalScore > 100 {
		t.Errorf("TotalScore %d exceeds 100", result.TotalScore)
	}
}

func TestCalcInvestmentScore_ScoreNotBelow0(t *testing.T) {
	// 全リスク最大でも 0 を下回らない
	cur := time.Now().Year()
	input := InvestmentScoreInput{
		FloodItems:        []FloodHazardItem{{DepthRank: 5}},
		StormItems:        []StormHazardItem{{DepthJa: "1m"}},
		TsunamiItems:      []TsunamiHazardItem{{DepthJa: "1m"}},
		LandslideItems:    []LandslideHazardItem{{ZoneCode: 1}},
		LiquefactionItems: []LiquefactionRiskItem{{TendencyLevel: 1}},
		EmbankmentItems:   []EmbankmentItem{{Classification: "腹付け型"}},
		DisasterItems:     []DisasterHistoryItem{{Name: "浸水域", Year: cur - 1}},
		HasLandPriceTrend:   true,
		LandPriceChangeRate: -0.20,
	}
	result := CalcInvestmentScore(input)
	if result.TotalScore < 0 {
		t.Errorf("TotalScore %d is below 0", result.TotalScore)
	}
}

func TestCalcInvestmentScore_RadarDataHas5Points(t *testing.T) {
	// レーダーチャートは常に5カテゴリ
	result := CalcInvestmentScore(InvestmentScoreInput{})
	if len(result.Breakdown.RadarData) != 5 {
		t.Errorf("RadarData len = %d, want 5", len(result.Breakdown.RadarData))
	}
}

func TestCalcInvestmentScore_RadarScoreInRange(t *testing.T) {
	// 各レーダースコアは 0〜100 の範囲
	result := CalcInvestmentScore(InvestmentScoreInput{})
	for _, p := range result.Breakdown.RadarData {
		if p.Score < 0 || p.Score > 100 {
			t.Errorf("RadarData[%q].Score = %.1f, want [0, 100]", p.Category, p.Score)
		}
	}
}

// --- calcLandPriceTrendScore ---

func TestCalcLandPriceTrendScore_NoData(t *testing.T) {
	s := calcLandPriceTrendScore(0, false)
	if s.Score != 0 {
		t.Errorf("score = %d, want 0 when hasData=false", s.Score)
	}
}

func TestCalcLandPriceTrendScore_Boundaries(t *testing.T) {
	tests := []struct {
		label      string
		changeRate float64
		want       int
	}{
		{">10%", 0.11, 10},
		{"ちょうど10%", 0.10, 5},  // pct=10 は >10 に入らず >5 に入る
		{">5%", 0.06, 5},
		{"ちょうど5%", 0.05, 0},   // pct=5 は >5 に入らず >=-5 に入る
		{"0%（横ばい）", 0.00, 0},
		{"-5%（下限境界）", -0.05, 0}, // pct=-5 は >=-5 に入る
		{"-6%", -0.06, -5},
		{"-10%", -0.10, -5},       // pct=-10 は >=-10 に入る
		{"<-10%", -0.11, -10},
	}
	for _, tc := range tests {
		got := calcLandPriceTrendScore(tc.changeRate, true)
		if got.Score != tc.want {
			t.Errorf("[%s] changeRate=%.2f: score = %d, want %d", tc.label, tc.changeRate, got.Score, tc.want)
		}
	}
}

func TestCalcInvestmentScore_LandPriceTrend_Ignored_WhenNoData(t *testing.T) {
	// HasLandPriceTrend=false の場合、LandPriceChangeRate は無視される
	base := CalcInvestmentScore(InvestmentScoreInput{})
	withRate := CalcInvestmentScore(InvestmentScoreInput{
		LandPriceChangeRate: 0.50,
		HasLandPriceTrend:   false,
	})
	if base.TotalScore != withRate.TotalScore {
		t.Errorf("HasLandPriceTrend=false なのにスコアが変化: base=%d, withRate=%d",
			base.TotalScore, withRate.TotalScore)
	}
}

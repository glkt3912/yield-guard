package domain

import "sort"

// LandAppraisalItem は XCT001 から変換した地価公示1レコード
type LandAppraisalItem struct {
	Year        int
	PricePerSqm float64 // 1㎡当たりの公示価格（円）
	ChangeRate  float64 // 前年比変動率（小数: 0.05 = +5%）
	District    string  // 地域名
}

// AppraisalComparisonResult は地価公示と取引価格の2軸比較結果
type AppraisalComparisonResult struct {
	AppraisalMedianPerSqm float64 `json:"appraisalMedianPerSqm"` // 公示価格中央値（円/m²）
	AppraisalCount        int     `json:"appraisalCount"`        // 標準地点数
	AppraisalTrend        float64 `json:"appraisalTrend"`        // 平均変動率（小数）
	TrendLabel            string  `json:"trendLabel"`            // "上昇" / "安定" / "下落"
}

// CalcAppraisalComparison は地価公示データから比較統計を算出する
func CalcAppraisalComparison(items []LandAppraisalItem) AppraisalComparisonResult {
	if len(items) == 0 {
		return AppraisalComparisonResult{}
	}

	prices := make([]float64, 0, len(items))
	var totalChangeRate float64
	for _, item := range items {
		if item.PricePerSqm > 0 {
			prices = append(prices, item.PricePerSqm)
		}
		totalChangeRate += item.ChangeRate
	}

	sort.Float64s(prices)
	medianPrice := 0.0
	if len(prices) > 0 {
		mid := len(prices) / 2
		if len(prices)%2 == 0 {
			medianPrice = (prices[mid-1] + prices[mid]) / 2
		} else {
			medianPrice = prices[mid]
		}
	}

	trend := totalChangeRate / float64(len(items))

	return AppraisalComparisonResult{
		AppraisalMedianPerSqm: medianPrice,
		AppraisalCount:        len(items),
		AppraisalTrend:        trend,
		TrendLabel:            appraisalTrendLabel(trend),
	}
}

func appraisalTrendLabel(trend float64) string {
	switch {
	case trend > 0.03:
		return "上昇"
	case trend < -0.03:
		return "下落"
	default:
		return "安定"
	}
}

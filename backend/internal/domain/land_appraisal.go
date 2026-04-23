package domain

import (
	"math"
	"sort"
)

// RentDeclineHint は地価公示データから算出した賃料下落率参考値
type RentDeclineHint struct {
	HintRate       float64 `json:"hintRate"`       // 参考下落率（小数: 0.008 = 0.8%/年）
	Basis          string  `json:"basis"`           // "land_appraisal" | "fallback"
	DataPointCount int     `json:"dataPointCount"`  // 使用した地価公示データ件数
	FallbackUsed   bool    `json:"fallbackUsed"`
	Note           string  `json:"note"`
}

const minDataPointsForHint = 5

// CalcRentDeclineHint は複数年分の地価公示データからCAGRを算出し参考下落率を返す。
// itemsByYear の key は西暦年、value はその年の地価公示レコード群。
// 総データ件数 < 5 または有効年数 < 2 の場合は fallback を返す。
// 地価が下落傾向（CAGR < 0）の場合のみ hintRate に絶対値をセットし basis = "land_appraisal" を返す。
func CalcRentDeclineHint(itemsByYear map[int][]LandAppraisalItem) RentDeclineHint {
	fallback := RentDeclineHint{
		Basis:        "fallback",
		FallbackUsed: true,
		Note:         "データ不足のため構造別平均値を推奨します",
	}

	// 総データ件数チェック
	total := 0
	for _, items := range itemsByYear {
		total += len(items)
	}
	if total < minDataPointsForHint {
		fallback.DataPointCount = total
		return fallback
	}

	// 各年の㎡単価中央値を計算
	type yearMedian struct {
		year   int
		median float64
	}
	var medians []yearMedian
	for year, items := range itemsByYear {
		prices := make([]float64, 0, len(items))
		for _, item := range items {
			if item.PricePerSqm > 0 {
				prices = append(prices, item.PricePerSqm)
			}
		}
		if len(prices) == 0 {
			continue
		}
		sort.Float64s(prices)
		mid := len(prices) / 2
		var median float64
		if len(prices)%2 == 0 {
			median = (prices[mid-1] + prices[mid]) / 2
		} else {
			median = prices[mid]
		}
		medians = append(medians, yearMedian{year: year, median: median})
	}

	sort.Slice(medians, func(i, j int) bool { return medians[i].year < medians[j].year })

	if len(medians) < 2 {
		return RentDeclineHint{
			Basis:          "fallback",
			FallbackUsed:   true,
			DataPointCount: total,
			Note:           "取得できた年次が1年分のみのため構造別平均値を推奨します",
		}
	}

	first := medians[0]
	last := medians[len(medians)-1]
	n := float64(last.year - first.year)

	// CAGR = (last/first)^(1/n) - 1
	cagr := math.Pow(last.median/first.median, 1.0/n) - 1

	if cagr >= 0 {
		return RentDeclineHint{
			HintRate:       0,
			Basis:          "fallback",
			DataPointCount: total,
			FallbackUsed:   true,
			Note:           "地価は上昇または横ばいのため構造別平均値を推奨します",
		}
	}

	return RentDeclineHint{
		HintRate:       math.Abs(cagr),
		Basis:          "land_appraisal",
		DataPointCount: total,
		FallbackUsed:   false,
		Note:           "地価公示データに基づく参考値です。賃料下落率と完全には一致しません",
	}
}

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

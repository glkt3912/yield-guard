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

// AcquisitionCostOptions は諸経費計算のオプション
type AcquisitionCostOptions struct {
	// BrokerageMultiplier: 1.0=標準, 0.0=仲介手数料無料, 0.5=半額
	BrokerageMultiplier float64
}

// DefaultAcquisitionCostOptions は標準的な取引条件のオプションを返す
func DefaultAcquisitionCostOptions() AcquisitionCostOptions {
	return AcquisitionCostOptions{BrokerageMultiplier: 1.0}
}

// CalcAcquisitionCosts は取得時の諸経費（#75スコープ）を算出する。
// 登録免許税・不動産取得税（#76）と固定資産税日割り（#77）は後続issueで追加。
func CalcAcquisitionCosts(totalPrice float64, opts AcquisitionCostOptions) AcquisitionCostBreakdown {
	brokerage := CalcBrokerageFee(totalPrice, opts.BrokerageMultiplier)
	stamp := CalcStampDuty(totalPrice)
	return AcquisitionCostBreakdown{
		BrokerageFee: brokerage,
		StampDuty:    stamp,
		Total:        brokerage + stamp,
	}
}

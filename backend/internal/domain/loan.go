package domain

import "math"

// calcMonthlyPayment は元利均等返済の月次返済額を計算する
// P: 元金, annualRate: 年利, years: 返済年数
func calcMonthlyPayment(principal, annualRate float64, years int) float64 {
	if principal <= 0 || years <= 0 {
		return 0
	}
	if annualRate == 0 {
		return principal / float64(years*12)
	}
	r := annualRate / 12
	n := float64(years * 12)
	return principal * r * math.Pow(1+r, n) / (math.Pow(1+r, n) - 1)
}

// calcYearlyLoanComponents は1年分の利息・元金返済額を計算する
// 最終月で月次返済額が残高を超える場合は残高のみを元金返済として扱い、誤差を防ぐ
func calcYearlyLoanComponents(balance, annualRate, monthlyPayment float64) (interest, principal float64) {
	if annualRate == 0 {
		if monthlyPayment*12 > balance {
			return 0, balance
		}
		return 0, monthlyPayment * 12
	}
	r := annualRate / 12
	remaining := balance
	for range 12 {
		if remaining <= 0 {
			break
		}
		monthInterest := remaining * r
		monthPrincipal := monthlyPayment - monthInterest
		// 最終月: 残高が月次元金返済より少ない場合は残高のみ返済
		if monthPrincipal > remaining {
			monthPrincipal = remaining
		}
		interest += monthInterest
		principal += monthPrincipal
		remaining -= monthPrincipal
	}
	return interest, principal
}

// calcYearlyLoanComponentsEqualPrincipal は元金均等返済における1年分の利息・元金返済額を計算する。
// monthlyPrincipal: 毎月の固定元金返済額（= 元本 / 総返済回数）
func calcYearlyLoanComponentsEqualPrincipal(balance, annualRate, monthlyPrincipal float64) (interest, principal float64) {
	r := annualRate / 12
	remaining := balance
	for range 12 {
		if remaining <= 0 {
			break
		}
		mp := monthlyPrincipal
		if mp > remaining {
			mp = remaining
		}
		interest += remaining * r
		principal += mp
		remaining -= mp
	}
	return interest, principal
}

// CalcEqualPrincipalPayment は元金均等返済の month 回目（1始まり）の月次返済額を返す。
// principal: 借入元本, annualRate: 年利, months: 総返済回数, month: 対象月（1〜months）
func CalcEqualPrincipalPayment(principal, annualRate float64, months, month int) float64 {
	if principal <= 0 || months <= 0 || month < 1 || month > months {
		return 0
	}
	monthlyPrincipal := principal / float64(months)
	remainingBalance := principal - float64(month-1)*monthlyPrincipal
	monthlyInterest := remainingBalance * (annualRate / 12)
	return monthlyPrincipal + monthlyInterest
}

// resolveRateForYear はスケジュールと基準金利から指定年の適用金利を返す。
// schedule が空の場合は baseRate + rateDelta を返す（固定金利）。
// rateDelta はストレステスト用の追加オフセット。
func resolveRateForYear(baseRate, rateDelta float64, schedule []RateAdjustment, year int) float64 {
	rate := baseRate
	for _, adj := range schedule {
		if year >= adj.AfterYear {
			rate = adj.Rate
		}
	}
	return rate + rateDelta
}

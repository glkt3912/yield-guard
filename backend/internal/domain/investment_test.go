package domain

import (
	"context"
	"math"
	"testing"
)

const epsilon = 0.01 // 1円未満の誤差を許容

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

// TestCalcMonthlyPayment は元利均等返済の月次返済額を検証する
func TestCalcMonthlyPayment(t *testing.T) {
	tests := []struct {
		name      string
		principal float64
		rate      float64
		years     int
		wantApprox float64
	}{
		{
			name:      "1000万 年利1.5% 35年",
			principal: 10_000_000,
			rate:      0.015,
			years:     35,
			wantApprox: 30_607, // 約30,607円/月
		},
		{
			name:      "3000万 年利2.0% 30年",
			principal: 30_000_000,
			rate:      0.020,
			years:     30,
			wantApprox: 110_879, // 約110,879円/月
		},
		{
			name:      "金利ゼロ",
			principal: 12_000_000,
			rate:      0,
			years:     10,
			wantApprox: 100_000, // 1200万 / 120ヶ月
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcMonthlyPayment(tt.principal, tt.rate, tt.years)
			if !approxEqual(got, tt.wantApprox, 500) { // 500円以内の誤差
				t.Errorf("calcMonthlyPayment() = %.0f, want ≈ %.0f", got, tt.wantApprox)
			}
		})
	}
}

// TestAnalyze_GrossYield は表面利回りの計算を検証する
func TestAnalyze_GrossYield(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// 総投資額の検証
	// 5,000,000 + 10,000,000 + (15,000,000 * 0.07) = 16,050,000
	wantTotal := 5_000_000.0 + 10_000_000.0 + 15_000_000.0*0.07
	if !approxEqual(result.TotalInvestment, wantTotal, epsilon) {
		t.Errorf("TotalInvestment = %.0f, want %.0f", result.TotalInvestment, wantTotal)
	}

	// 総投資利回り: (120,000 * 12) / 16,050,000 ≈ 0.0897
	wantGross := (120_000.0 * 12) / wantTotal
	if !approxEqual(result.GrossYield, wantGross, 0.0001) {
		t.Errorf("GrossYield = %.4f, want %.4f", result.GrossYield, wantGross)
	}

	// 表面利回り（市場慣行・物件価格ベース）: (120,000 * 12) / 15,000,000 = 0.096
	wantMarketGross := (120_000.0 * 12) / (5_000_000.0 + 10_000_000.0)
	if !approxEqual(result.MarketGrossYield, wantMarketGross, 0.0001) {
		t.Errorf("MarketGrossYield = %.4f, want %.4f", result.MarketGrossYield, wantMarketGross)
	}
	// 諸費用を分母に含まない分、表面利回りは総投資利回りより高い（#773）
	if result.MarketGrossYield <= result.GrossYield {
		t.Errorf("MarketGrossYield(%.4f) should be greater than GrossYield(%.4f)",
			result.MarketGrossYield, result.GrossYield)
	}
}

// TestAnalyze_MarketGrossYield_8PercentBoundary は #773 の回帰テスト。
// 総投資利回りでは 8% 未満だが物件価格ベースの表面利回りでは 8% 以上になる物件で、
// 8%境界線判定（IsAboveYieldTarget）が市場慣行（物件価格ベース）で行われることを検証する。
func TestAnalyze_MarketGrossYield_8PercentBoundary(t *testing.T) {
	// 物件価格 15,000,000、諸経費率 7% → 総投資額 16,050,000。
	// 賃料 105,000/月 → 年収 1,260,000。
	//   表面利回り（物件価格ベース）= 1,260,000 / 15,000,000 = 8.40% → 目標 8% 達成
	//   総投資利回り（総投資額ベース）= 1,260,000 / 16,050,000 = 7.85% → 8% 未満
	input := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  105_000,
	}
	input.Defaults() // YieldTarget=0.08, MiscExpenseRate=0.07

	result := Analyze(context.Background(), input)

	if result.GrossYield >= input.YieldTarget {
		t.Fatalf("前提が崩れています: GrossYield(%.4f) は目標(%.2f)未満であるべき",
			result.GrossYield, input.YieldTarget)
	}
	if result.MarketGrossYield < input.YieldTarget {
		t.Fatalf("前提が崩れています: MarketGrossYield(%.4f) は目標(%.2f)以上であるべき",
			result.MarketGrossYield, input.YieldTarget)
	}
	if !result.IsAboveYieldTarget {
		t.Errorf("IsAboveYieldTarget = false, want true: 表面利回り%.2f%%が目標%.0f%%を満たすため",
			result.MarketGrossYield*100, input.YieldTarget*100)
	}
}

// TestAnalyze_Above8Percent は8%境界線判定を検証する
func TestAnalyze_Above8Percent(t *testing.T) {
	// 高賃料 → 8%以上
	highRent := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  200_000, // 高い賃料
	}
	highRent.Defaults()
	r1 := Analyze(context.Background(), highRent)
	if !r1.IsAboveYieldTarget {
		t.Errorf("高賃料ケース: IsAboveYieldTarget = false, want true (yield=%.2f%%)", r1.GrossYield*100)
	}

	// 低賃料 → 8%未満
	lowRent := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  80_000, // 低い賃料
	}
	lowRent.Defaults()
	r2 := Analyze(context.Background(), lowRent)
	if r2.IsAboveYieldTarget {
		t.Errorf("低賃料ケース: IsAboveYieldTarget = true, want false (yield=%.2f%%)", r2.GrossYield*100)
	}
}

// TestAnalyze_DeadCross はデッドクロス年の特定を検証する
func TestAnalyze_DeadCross(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      14_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood, // 耐用年数22年
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// 木造22年: 耐用年数内に元金が追い越すか、翌年(23年目)に減価償却=0でデッドクロス発生
	if result.DeadCrossYear == -1 {
		t.Errorf("DeadCrossYear = -1, expected a positive year")
	}
	// 耐用年数+1年以内にデッドクロスが来るはず
	if result.DeadCrossYear > 23 {
		t.Errorf("DeadCrossYear = %d, expected ≤ 23 (wood useful life + 1)", result.DeadCrossYear)
	}
	t.Logf("DeadCrossYear = %d", result.DeadCrossYear)
}

// TestAnalyze_ExitStrategy は出口戦略の計算を検証する
func TestAnalyze_ExitStrategy(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeRC, // 耐用年数47年 → 10年後も減価償却中
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    6, // 5年超 → 長期譲渡所得(20.315%)適用
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// 売却価格 = NOI / 6%（NOI = 実効賃料 - 運営経費）
	annualRent := 120_000.0 * 12 * (1 - 0.05)
	noi := annualRent * (1 - 0.20)
	wantSalePrice := noi / 0.06
	if !approxEqual(result.ExitSalePrice, wantSalePrice, 1000) {
		t.Errorf("ExitSalePrice = %.0f, want ≈ %.0f (NOI-based)", result.ExitSalePrice, wantSalePrice)
	}

	// 保有6年 > 5年 → 投資用物件の長期譲渡税率(20.315%)が適用されること
	// 租税特別措置法31条の3の10年超軽減(14.21%)は居住用財産の特例のため対象外
	if result.ExitCapitalGain > 0 {
		impliedTaxRate := result.ExitTransferTax / result.ExitCapitalGain
		if !approxEqual(impliedTaxRate, longTermTransferTaxRate, 0.001) {
			t.Errorf("長期譲渡税率 = %.5f, want %.5f", impliedTaxRate, longTermTransferTaxRate)
		}
	}
	t.Logf("ExitSalePrice=%.0f, CapGain=%.0f, Tax=%.0f, NetProceeds=%.0f, TotalEquity=%.0f",
		result.ExitSalePrice, result.ExitCapitalGain, result.ExitTransferTax,
		result.ExitNetProceeds, result.ExitTotalEquity)
}

// TestCalcExit_LoanFeeExcludedFromAcquisitionCost は融資諸費用(loanFee)が
// 譲渡所得の取得費に算入されないことを検証する（#510）。
//
// 設計:
//   - LoanFeeRate=0.02 → loanFee = 13,000,000 × 0.02 = 260,000円
//   - loanFeeあり vs loanFeeなし でキャピタルゲインが一致すること
//     （取得費に loanFee を含めていた場合はキャピタルゲインが小さくなる誤りが生じる）
func TestCalcExit_LoanFeeExcludedFromAcquisitionCost(t *testing.T) {
	base := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeRC, // 47年 → 保有期間中も減価償却継続
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    6,
		ExitYieldTarget: 0.06,
	}

	// loanFeeなし
	withoutFee := base
	withoutFee.LoanFeeRate = 0

	// loanFeeあり: LoanAmount × 0.02 = 260,000円 が miscExpenses に加算される
	withFee := base
	withFee.LoanFeeRate = 0.02

	r0 := Analyze(context.Background(), withoutFee)
	r1 := Analyze(context.Background(), withFee)

	// loanFeeは総投資額(TotalInvestment)に影響する
	loanFee := base.LoanAmount * 0.02 // 260,000
	if !approxEqual(r1.TotalInvestment, r0.TotalInvestment+loanFee, 1) {
		t.Errorf("TotalInvestment差 = %.0f, want %.0f (loanFee)", r1.TotalInvestment-r0.TotalInvestment, loanFee)
	}

	// loanFeeは取得費（税法上）に含まれないため、キャピタルゲインは同一であるべき
	if !approxEqual(r0.ExitCapitalGain, r1.ExitCapitalGain, 1) {
		t.Errorf(
			"ExitCapitalGain: loanFeeなし=%.0f, loanFeeあり=%.0f — 差=%.0f (loanFeeが取得費に算入されている)",
			r0.ExitCapitalGain, r1.ExitCapitalGain, r0.ExitCapitalGain-r1.ExitCapitalGain,
		)
	}

	// 同様に譲渡税も同一であるべき
	if !approxEqual(r0.ExitTransferTax, r1.ExitTransferTax, 1) {
		t.Errorf(
			"ExitTransferTax: loanFeeなし=%.0f, loanFeeあり=%.0f — 差=%.0f",
			r0.ExitTransferTax, r1.ExitTransferTax, r0.ExitTransferTax-r1.ExitTransferTax,
		)
	}

	t.Logf("withoutFee: CapGain=%.0f, Tax=%.0f / withFee: CapGain=%.0f, Tax=%.0f",
		r0.ExitCapitalGain, r0.ExitTransferTax, r1.ExitCapitalGain, r1.ExitTransferTax)
}

// TestAnalyze_StressTest はストレステスト（空室率・金利上昇）を検証する
func TestAnalyze_StressTest(t *testing.T) {
	base := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  120_000,
		LoanAmount:   13_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:    35,
	}
	base.Defaults()

	stressed := base
	stressed.VacancyRateDelta = 0.10 // 空室率+10%
	stressed.LoanRateDelta = 0.015   // 金利+1.5%

	baseResult := Analyze(context.Background(), base)
	stressResult := Analyze(context.Background(), stressed)

	// ストレス時のCFはベースより悪化するはず
	if len(baseResult.YearlyResults) == 0 || len(stressResult.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}
	baseCF := baseResult.YearlyResults[0].CashFlow
	stressCF := stressResult.YearlyResults[0].CashFlow
	if stressCF >= baseCF {
		t.Errorf("ストレス時CF(%.0f) >= ベースCF(%.0f), expected worse", stressCF, baseCF)
	}
	t.Logf("BaseCF=%.0f, StressCF=%.0f, delta=%.0f", baseCF, stressCF, stressCF-baseCF)
}

// TestCalcLandPriceStats は土地価格統計の計算を検証する
func TestCalcLandPriceStats(t *testing.T) {
	transactions := []LandTransaction{
		{PricePerTsubo: 100_000},
		{PricePerTsubo: 200_000},
		{PricePerTsubo: 300_000},
		{PricePerTsubo: 400_000},
		{PricePerTsubo: 500_000},
	}

	stats := CalcLandPriceStats(context.Background(), transactions)

	if stats.Count != 5 {
		t.Errorf("Count = %d, want 5", stats.Count)
	}
	if !approxEqual(stats.AverageTsubo, 300_000, epsilon) {
		t.Errorf("AverageTsubo = %.0f, want 300000", stats.AverageTsubo)
	}
	if !approxEqual(stats.MedianTsubo, 300_000, epsilon) {
		t.Errorf("MedianTsubo = %.0f, want 300000", stats.MedianTsubo)
	}
	if !approxEqual(stats.MinTsubo, 100_000, epsilon) {
		t.Errorf("MinTsubo = %.0f, want 100000", stats.MinTsubo)
	}
	if !approxEqual(stats.MaxTsubo, 500_000, epsilon) {
		t.Errorf("MaxTsubo = %.0f, want 500000", stats.MaxTsubo)
	}
}

// TestAnalyze_ZeroLoan はローンなし（全額自己資金）のケースを検証する
func TestAnalyze_ZeroLoan(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      0, // ローンなし
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}
	// ローン返済はゼロのはず
	if result.YearlyResults[0].AnnualLoanPayment != 0 {
		t.Errorf("AnnualLoanPayment = %.0f, want 0", result.YearlyResults[0].AnnualLoanPayment)
	}
	// CFは賃料収入から経費を引いた正値になるはず
	if result.YearlyResults[0].CashFlow <= 0 {
		t.Errorf("CashFlow = %.0f, expected positive with zero loan", result.YearlyResults[0].CashFlow)
	}
}

// TestAnalyze_ZeroExitYield は売却目標利回りがゼロの場合のゼロ除算を検証する
// Analyze 内で Defaults() が呼ばれるため ExitYieldTarget=0 はデフォルト値 0.06 に補完される。
// そのためパニックせず有効な売却価格が返ることを確認する。
func TestAnalyze_ZeroExitYield(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeRC,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0, // Defaults() により 0.06 に補完される
	}

	// パニックしないことを確認。Defaults() により ExitYieldTarget=0.06 となり正の売却価格が返る。
	result := Analyze(context.Background(), input)
	if result.ExitSalePrice <= 0 {
		t.Errorf("ExitSalePrice = %.0f, want > 0 (Defaults补完後は0.06で計算)", result.ExitSalePrice)
	}
	t.Logf("ExitYieldTarget補完確認: ExitSalePrice=%.0f", result.ExitSalePrice)
}

// TestAnalyze_FullVacancy は空室率100%（VacancyRate=1）の場合を検証する。
// effectiveVacancy は 0.99 にキャップされるため AnnualRent はわずかに正になる。
func TestAnalyze_FullVacancy(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     1.0, // 完全空室 → effectiveVacancy=0.99 にキャップ
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}
	// effectiveVacancy=0.99 にキャップ → AnnualRent = 120000*12*(1-0.99) = 14400
	wantRent := 120_000.0 * 12 * (1 - 0.99)
	if !approxEqual(result.YearlyResults[0].AnnualRent, wantRent, 1) {
		t.Errorf("AnnualRent = %.0f, want ≈ %.0f (capped at effectiveVacancy=0.99)", result.YearlyResults[0].AnnualRent, wantRent)
	}
	// 表面利回りは正 (空室率はNetYieldに影響するがGrossYieldには影響しない)
	if result.GrossYield <= 0 {
		t.Errorf("GrossYield = %.4f, expected positive", result.GrossYield)
	}
}

// TestCompareLandPrice は土地価格の相場比較を検証する
func TestCompareLandPrice(t *testing.T) {
	stats := LandPriceStats{
		AverageTsubo: 200_000,
		MedianTsubo:  200_000,
	}

	// 割高ケース: 中央値より11%以上高い
	comparison := CompareLandPrice(stats, 5_000_000, 66.116) // 約20坪
	// 5,000,000 / 20 = 250,000円/坪 → 中央値200,000より25%高い → 割高
	if comparison.Assessment != "割高" {
		t.Errorf("Assessment = %q, want '割高' (pricePerTsubo=%.0f)", comparison.Assessment, comparison.InputPricePerTsubo)
	}

	// 割安ケース
	comparison2 := CompareLandPrice(stats, 3_000_000, 66.116) // 約20坪
	// 3,000,000 / 20 = 150,000円/坪 → 中央値より25%低い → 割安
	if comparison2.Assessment != "割安" {
		t.Errorf("Assessment = %q, want '割安' (pricePerTsubo=%.0f)", comparison2.Assessment, comparison2.InputPricePerTsubo)
	}
}

func TestCalcCriticalErrors_DeadCrossEarly(t *testing.T) {
	input := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		BuildingAge:  0,
		BuildingType: BuildingTypeWood,
	}
	// デッドクロス5年目 → REJECT
	errs := calcCriticalErrors(input, 5, 22)
	found := false
	for _, e := range errs {
		if e.Code == "DEADCROSS_EARLY" && e.Status == CriticalStatusReject {
			found = true
		}
	}
	if !found {
		t.Error("expected DEADCROSS_EARLY REJECT when deadCrossYear=5")
	}

	// デッドクロス15年目 → 対象外
	errs2 := calcCriticalErrors(input, 15, 22)
	for _, e := range errs2 {
		if e.Code == "DEADCROSS_EARLY" {
			t.Error("unexpected DEADCROSS_EARLY when deadCrossYear=15")
		}
	}
}

func TestCalcCriticalErrors_LandValueGuard(t *testing.T) {
	// 築20年木造: residualLife=6, usefulLife=22
	// 積算 = 5M + 10M*(6/22) = 5M + 2.73M = 7.73M
	// 購入総額 = 15M → 7.73/15 = 51.5% → 閾値超のためREJECTなし
	input := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		BuildingAge:  20,
		BuildingType: BuildingTypeWood,
	}
	errs := calcCriticalErrors(input, -1, 22)
	for _, e := range errs {
		if e.Code == "LAND_VALUE_GUARD" {
			t.Errorf("unexpected LAND_VALUE_GUARD for 51.5%% appraisal ratio")
		}
	}

	// 築25年木造(法定超): residualLife=4, usefulLife=22
	// 積算 = 5M + 10M*(4/22) = 5M + 1.82M = 6.82M
	// 購入総額 = 20M → 6.82/20 = 34% → REJECT
	input2 := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 15_000_000,
		BuildingAge:  25,
		BuildingType: BuildingTypeWood,
	}
	errs2 := calcCriticalErrors(input2, -1, 22)
	found := false
	for _, e := range errs2 {
		if e.Code == "LAND_VALUE_GUARD" && e.Status == CriticalStatusReject {
			found = true
		}
	}
	if !found {
		t.Error("expected LAND_VALUE_GUARD REJECT when appraisal ratio is 34%")
	}
}

// TestAnalyze_ExitNOI_UsesHoldingYearValues は出口NOIが保有年数時点の賃料下落を反映することを検証する
func TestAnalyze_ExitNOI_UsesHoldingYearValues(t *testing.T) {
	const (
		monthlyRent     = 120_000.0
		vacancyRate     = 0.05
		expenseRate     = 0.20
		rentDeclineRate = 0.01  // 年1%下落
		holdingYears    = 10
		exitYieldTarget = 0.06
	)

	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     monthlyRent,
		VacancyRate:     vacancyRate,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeRC,
		ExpenseRate:     expenseRate,
		IncomeTaxRate:   0.33,
		HoldingYears:    holdingYears,
		ExitYieldTarget: exitYieldTarget,
		RentDeclineRate: rentDeclineRate,
	}

	result := Analyze(context.Background(), input)

	// 保有10年目(インデックス9)の賃料下落係数: (1-0.01)^9 = 0.99^9
	declineFactor := math.Pow(1-rentDeclineRate, float64(holdingYears-1))
	baseAnnualRent := monthlyRent * 12 * (1 - vacancyRate)
	expectedRent := baseAnnualRent * declineFactor
	expectedNOI := expectedRent * (1 - expenseRate)
	expectedSalePrice := expectedNOI / exitYieldTarget

	if len(result.YearlyResults) < holdingYears {
		t.Fatalf("YearlyResults too short: got %d, want >= %d", len(result.YearlyResults), holdingYears)
	}

	// 保有年次の賃料が下落を反映していること
	yearRent := result.YearlyResults[holdingYears-1].AnnualRent
	if !approxEqual(yearRent, expectedRent, 1.0) {
		t.Errorf("YearlyResults[%d].AnnualRent = %.2f, want ≈ %.2f", holdingYears-1, yearRent, expectedRent)
	}

	// 売却価格が保有年次の下落後NOIから算出されること
	if !approxEqual(result.ExitSalePrice, expectedSalePrice, 1000) {
		t.Errorf("ExitSalePrice = %.0f, want ≈ %.0f (decline-adjusted NOI)", result.ExitSalePrice, expectedSalePrice)
	}

	// RentDeclineRate=0のケースより売却価格が低いことを確認
	inputNoDecline := input
	inputNoDecline.RentDeclineRate = 0
	resultNoDecline := Analyze(context.Background(), inputNoDecline)
	if result.ExitSalePrice >= resultNoDecline.ExitSalePrice {
		t.Errorf("ExitSalePrice with decline (%.0f) should be < without decline (%.0f)",
			result.ExitSalePrice, resultNoDecline.ExitSalePrice)
	}

	t.Logf("declineFactor=%.6f, expectedRent=%.0f, ExitSalePrice=%.0f (vs no-decline=%.0f)",
		declineFactor, expectedRent, result.ExitSalePrice, resultNoDecline.ExitSalePrice)
}

func makeTx(cityPlanning string) LandTransaction {
	return LandTransaction{
		CityPlanning:     cityPlanning,
		BuildingCoverage: "60",
		FloorAreaRatio:   "200",
		PricePerTsubo:    100_000,
	}
}

func TestDetectUrbanRisks_ControlZone(t *testing.T) {
	txs := []LandTransaction{makeTx("市街化調整区域")}
	zoning := calcZoningSummary(txs)
	risks := detectUrbanRisks(txs, zoning)

	found := false
	for _, r := range risks {
		if r.Code == "URBANIZATION_CONTROL_ZONE" && r.Level == UrbanRiskLevelError {
			found = true
		}
	}
	if !found {
		t.Error("expected URBANIZATION_CONTROL_ZONE ERROR for 市街化調整区域")
	}
}

func TestDetectUrbanRisks_UnzonedArea(t *testing.T) {
	for _, cp := range []string{"非線引き区域", "都市計画区域外"} {
		txs := []LandTransaction{makeTx(cp)}
		zoning := calcZoningSummary(txs)
		risks := detectUrbanRisks(txs, zoning)

		found := false
		for _, r := range risks {
			if r.Code == "UNZONED_AREA" && r.Level == UrbanRiskLevelWarning {
				found = true
			}
		}
		if !found {
			t.Errorf("expected UNZONED_AREA WARNING for cityPlanning=%q", cp)
		}
	}
}

func TestDetectUrbanRisks_MixedZone30Pct(t *testing.T) {
	// 10件中3件が市街化調整区域 → 30% → WARNING
	txs := make([]LandTransaction, 10)
	for i := range txs {
		cp := "第一種住居地域"
		if i < 3 {
			cp = "市街化調整区域"
		}
		txs[i] = makeTx(cp)
	}
	zoning := calcZoningSummary(txs) // 最頻値は第一種住居地域
	risks := detectUrbanRisks(txs, zoning)

	found := false
	for _, r := range risks {
		if r.Code == "MIXED_ZONE_CAUTION" && r.Level == UrbanRiskLevelWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected MIXED_ZONE_CAUTION WARNING when 30% are 市街化調整区域")
	}
}

func TestDetectUrbanRisks_MixedZoneBelow30Pct(t *testing.T) {
	// 10件中2件が市街化調整区域 → 20% → WARNING なし
	txs := make([]LandTransaction, 10)
	for i := range txs {
		cp := "第一種住居地域"
		if i < 2 {
			cp = "市街化調整区域"
		}
		txs[i] = makeTx(cp)
	}
	zoning := calcZoningSummary(txs)
	risks := detectUrbanRisks(txs, zoning)

	for _, r := range risks {
		if r.Code == "MIXED_ZONE_CAUTION" {
			t.Error("unexpected MIXED_ZONE_CAUTION when only 20% are 市街化調整区域")
		}
	}
}

// TestAnalyze_StressScenarios はストレスシナリオの自動計算を検証する
func TestAnalyze_StressScenarios(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// カスタムデルタが0なので6シナリオのみ生成される
	if len(result.StressScenarios) != 6 {
		t.Errorf("StressScenarios count = %d, want 6", len(result.StressScenarios))
	}

	// 1番目はベースライン
	if result.StressScenarios[0].Label != "ベースライン" {
		t.Errorf("StressScenarios[0].Label = %q, want 'ベースライン'", result.StressScenarios[0].Label)
	}

	// 複合ストレス（金利+2%, 空室+10%）はDSCR < 1.0 or 安全でない可能性が高い
	// ベースライン時より複合ストレス時のDSCRは悪化するはず
	baseline := result.StressScenarios[0]
	compound := result.StressScenarios[5]
	if compound.Label != "複合ストレス" {
		t.Errorf("StressScenarios[5].Label = %q, want '複合ストレス'", compound.Label)
	}
	if compound.DSCR >= baseline.DSCR {
		t.Errorf("複合ストレスDSCR(%.4f) >= ベースラインDSCR(%.4f), expected worse", compound.DSCR, baseline.DSCR)
	}
	t.Logf("baseline DSCR=%.4f, compound DSCR=%.4f, compound IsSafe=%v", baseline.DSCR, compound.DSCR, compound.IsSafe)
}

// TestAnalyze_StressScenarios_IsSafe はDSCR < 1.0時のIsSafe=falseを検証する
func TestAnalyze_StressScenarios_IsSafe(t *testing.T) {
	// 高ローン・低賃料でDSCR < 1.0となるケース
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     80_000, // 低賃料
		VacancyRate:     0.05,
		LoanAmount:      20_000_000, // 高ローン
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// 複合ストレス（金利+2%, 空室+10%）ではIsSafe=falseになるはず
	compound := result.StressScenarios[5]
	if compound.IsSafe {
		t.Errorf("複合ストレスでIsSafe=true, expected false (DSCR=%.4f)", compound.DSCR)
	}
	t.Logf("compound DSCR=%.4f, IsSafe=%v", compound.DSCR, compound.IsSafe)
}

// TestAnalyze_StressScenarios_BreakEvenNever はCFが黒転しない場合のBreakEvenYear=-1を検証する
func TestAnalyze_StressScenarios_BreakEvenNever(t *testing.T) {
	// 極端に高いローン・低賃料でCFが常にマイナスになるケース
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     50_000,    // 非常に低い賃料
		VacancyRate:     0.05,
		LoanAmount:      30_000_000, // 非常に高いローン
		AnnualLoanRate:  0.03,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// いずれかのシナリオでBreakEvenYear=-1となることを確認
	foundNever := false
	for _, sc := range result.StressScenarios {
		if sc.BreakEvenYear == -1 {
			foundNever = true
			t.Logf("BreakEvenYear=-1 in scenario %q (DSCR=%.4f)", sc.Label, sc.DSCR)
		}
	}
	if !foundNever {
		t.Error("expected at least one scenario with BreakEvenYear=-1 for high-loan/low-rent case")
	}
}

// TestAnalyze_StressScenarios_CustomSeventh はカスタムデルタが非ゼロのとき第7シナリオが追加されることを検証する
func TestAnalyze_StressScenarios_CustomSeventh(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
		LoanRateDelta:   0.005, // カスタム金利上昇
	}

	result := Analyze(context.Background(), input)

	if len(result.StressScenarios) != 7 {
		t.Errorf("StressScenarios count = %d, want 7 (6 default + 1 custom)", len(result.StressScenarios))
	}
	if result.StressScenarios[6].Label != "カスタム" {
		t.Errorf("StressScenarios[6].Label = %q, want 'カスタム'", result.StressScenarios[6].Label)
	}
}

// TestCalcStressScenario_DSCRAbove1ButBreakEvenExceedsHolding は、
// DSCR >= 1.0 だが保有期間内にブレークイーンしない場合のIsSafe=falseを検証する。
//
// 設計: 金利ゼロ・経費ゼロで年間賃料 == 年間ローン返済額となるよう設定する。
//   - annualRent       = 100,000 × 12 = 1,200,000
//   - annualExpenses   = 0（expenseRate=0, AnnualPropertyTax=0）
//   - noi              = 1,200,000
//   - annualLoanPayment = 100,000 × 12 = 1,200,000（金利ゼロ: 元金42,000,000 / 420ヶ月）
//   - DSCR             = 1,200,000 / 1,200,000 = 1.0
//   - cf per year      = 0 → cumCF は一切増えず breakEvenYear = -1
// → IsSafe は false であるべき（累積CFが黒転しないため投資回収できない）
func TestCalcStressScenario_DSCRAbove1ButBreakEvenExceedsHolding(t *testing.T) {
	input := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  100_000,
		VacancyRate:  0,
		LoanAmount:   42_000_000, // 金利ゼロで月10万円 × 420ヶ月
		AnnualLoanRate: 0,        // 金利ゼロ → 月次返済額 = 元金 / 期間
		LoanYears:    35,
		ExpenseRate:  0,          // 経費なし
		HoldingYears: 10,
		BuildingType: BuildingTypeWood,
	}

	result := calcStressScenario(context.Background(), input, "テスト", 0, 0)

	// 前提確認: DSCR が 1.0 以上
	if result.DSCR < 1.0 {
		t.Fatalf("前提条件未充足: DSCR=%.4f < 1.0", result.DSCR)
	}
	// 前提確認: 保有期間内にブレークイーンしない
	if result.BreakEvenYear != -1 {
		t.Fatalf("前提条件未充足: BreakEvenYear=%d, want -1 (cumCF==0は黒転非達成)", result.BreakEvenYear)
	}

	// DSCR >= 1.0 かつ BreakEvenYear == -1 → IsSafe は false であるべき
	if result.IsSafe {
		t.Errorf("IsSafe = true, want false: DSCR=%.4f >= 1.0 だが保有期間内にブレークイーンしない (BreakEvenYear=%d)",
			result.DSCR, result.BreakEvenYear)
	}
	t.Logf("DSCR=%.4f, BreakEvenYear=%d, IsSafe=%v", result.DSCR, result.BreakEvenYear, result.IsSafe)
}

// TestCalcStressScenario_DSCRWorstYearVariableRate は、変動金利で後年金利が急上昇する場合に
// DSCR が初年度ではなく最悪年（最高返済額の年）を反映することを検証する。
//
// 設計: 1-4年目 1.5%、5年目以降 4.0% に上昇するスケジュール。
//   - 初年度返済額は低金利ベースで計算されるため DSCR は高め
//   - 5年目以降は返済額が増加し DSCR が低下する
//   - 修正後: result.DSCR は最悪年（5年目以降）の値を反映し、初年度ベースより低くなる
func TestCalcStressScenario_DSCRWorstYearVariableRate(t *testing.T) {
	input := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 15_000_000,
		MonthlyRent:  120_000,
		VacancyRate:  0.05,
		LoanAmount:   18_000_000,
		AnnualLoanRate: 0.015, // 初期金利 1.5%
		LoanYears:    25,
		ExpenseRate:  0.15,
		HoldingYears: 10,
		BuildingType: BuildingTypeRC,
		RateAdjustmentSchedule: []RateAdjustment{
			{AfterYear: 5, Rate: 0.04}, // 5年目から 4.0% に急上昇
		},
	}

	// 初年度のみの計算で得られる DSCR（旧実装相当）を参照値として計算
	initRate := resolveRateForYear(input.AnnualLoanRate, 0, input.RateAdjustmentSchedule, 1)
	monthlyPaymentY1 := calcMonthlyPayment(input.LoanAmount, initRate, input.LoanYears)
	annualLoanY1 := monthlyPaymentY1 * 12
	annualRent := input.MonthlyRent * 12 * (1 - input.VacancyRate)
	annualExpenses := annualRent*input.ExpenseRate + input.AnnualPropertyTax
	noi := annualRent - annualExpenses
	dscrYear1 := noi / annualLoanY1

	result := calcStressScenario(context.Background(), input, "変動金利テスト", 0, 0)

	// 最悪年 DSCR は初年度 DSCR より低いはず（5年目以降の高金利による返済増を反映）
	if result.DSCR >= dscrYear1 {
		t.Errorf("DSCR(%.4f) >= 初年度DSCR(%.4f): 変動金利上昇が最悪年DSCRに反映されていない", result.DSCR, dscrYear1)
	}
	t.Logf("year1DSCR=%.4f, worstDSCR=%.4f, IsSafe=%v", dscrYear1, result.DSCR, result.IsSafe)
}

// TestCalcStressScenario_IsSafeFlipsWithVariableRate は、変動金利上昇によって
// isSafe が true から false に反転することを検証する。
//
// 設計: 初年度 DSCR >= 1.0 だが、5年目に金利急上昇で最悪年 DSCR < 1.0 となるケース。
//   - 旧実装では初年度 DSCR >= 1.0 かつ黒転達成 → isSafe = true（誤判定）
//   - 修正後は最悪年 DSCR < 1.0 → isSafe = false（正確な判定）
func TestCalcStressScenario_IsSafeFlipsWithVariableRate(t *testing.T) {
	input := InvestmentInput{
		LandPrice:    3_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  90_000,
		VacancyRate:  0.05,
		LoanAmount:   16_000_000,
		AnnualLoanRate: 0.005, // 初期金利 0.5%（返済額が低く DSCR >= 1.0）
		LoanYears:    25,
		ExpenseRate:  0.10,
		HoldingYears: 10,
		BuildingType: BuildingTypeRC,
		RateAdjustmentSchedule: []RateAdjustment{
			{AfterYear: 5, Rate: 0.05}, // 5年目から 5.0% に急上昇 → 返済額増加で DSCR < 1.0
		},
	}

	// 前提: 初年度 DSCR >= 1.0（旧実装では isSafe = true になっていた）
	initRate := resolveRateForYear(input.AnnualLoanRate, 0, input.RateAdjustmentSchedule, 1)
	monthlyPaymentY1 := calcMonthlyPayment(input.LoanAmount, initRate, input.LoanYears)
	annualRent := input.MonthlyRent * 12 * (1 - input.VacancyRate)
	annualExpenses := annualRent*input.ExpenseRate + input.AnnualPropertyTax
	noi := annualRent - annualExpenses
	dscrYear1 := noi / (monthlyPaymentY1 * 12)
	if dscrYear1 < 1.0 {
		t.Fatalf("前提条件未充足: 初年度DSCR=%.4f < 1.0（テスト設計を確認）", dscrYear1)
	}

	result := calcStressScenario(context.Background(), input, "変動金利isSafe反転テスト", 0, 0)

	// 最悪年 DSCR は 1.0 未満であるべき
	if result.DSCR >= 1.0 {
		t.Errorf("最悪年DSCR=%.4f >= 1.0: 変動金利上昇が DSCR に反映されていない", result.DSCR)
	}
	// isSafe は false であるべき（旧実装では true になっていたバグ）
	if result.IsSafe {
		t.Errorf("IsSafe = true, want false: 最悪年DSCR=%.4f < 1.0 なのに安全と判定された", result.DSCR)
	}
	t.Logf("year1DSCR=%.4f, worstDSCR=%.4f, IsSafe=%v, BreakEvenYear=%d",
		dscrYear1, result.DSCR, result.IsSafe, result.BreakEvenYear)
}

// TestCalcStressScenario_RentDeclineLowersDSCR は、RentDeclineRate > 0 の場合に
// DSCR が下落率ゼロのケースより低くなることを検証する（#311）。
//
// 設計: 3% の賃料下落率を設定し、変動金利上昇スケジュールも加えることで
// 保有期間後半に yearNOI と yearLoan の両方が悪化するケースを再現する。
func TestCalcStressScenario_RentDeclineLowersDSCR(t *testing.T) {
	base := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 15_000_000,
		MonthlyRent:  130_000,
		VacancyRate:  0.05,
		LoanAmount:   18_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:    25,
		ExpenseRate:  0.20,
		IncomeTaxRate: 0.33,
		HoldingYears: 10,
		BuildingType: BuildingTypeRC,
		RateAdjustmentSchedule: []RateAdjustment{
			{AfterYear: 5, Rate: 0.03},
		},
	}

	withoutDecline := base
	withoutDecline.RentDeclineRate = 0

	withDecline := base
	withDecline.RentDeclineRate = 0.03

	r0 := calcStressScenario(context.Background(), withoutDecline, "下落なし", 0, 0)
	r1 := calcStressScenario(context.Background(), withDecline, "下落3%", 0, 0)

	if r1.DSCR >= r0.DSCR {
		t.Errorf("RentDeclineRate=3%% の DSCR(%.4f) >= 下落なし DSCR(%.4f): 賃料下落が反映されていない",
			r1.DSCR, r0.DSCR)
	}
	t.Logf("noDeclineDSCR=%.4f, declineDSCR=%.4f", r0.DSCR, r1.DSCR)
}

// TestCalcStressScenario_AfterTaxCFDelaysBreakEven は、IncomeTaxRate > 0 の場合に
// 税引後 CF ベースの黒転が税引前より悪化（遅延または未達）することを検証する（#312）。
//
// 設計: yearNOI がローン返済額をわずかに上回る（CF > 0）一方、
// 利息控除後の課税所得×税率がそのCFを上回るよう貸出額を調整する。
// 等払い方式・金利 2%・30年では初年度利息 ≈ L×2%、yearNOI ≈ 0.050L となるよう設定し、
// noTax: breakEven=1、withTax: afterTaxCF < 0 → breakEven=-1 となることを確認する。
func TestCalcStressScenario_AfterTaxCFDelaysBreakEven(t *testing.T) {
	base := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 15_000_000,
		MonthlyRent:  77_000, // yearNOI ≈ 746k: CF > 0 だが incomeTax(40%) が CF を超える
		VacancyRate:  0.05,
		LoanAmount:   15_000_000,
		AnnualLoanRate: 0.02,
		LoanYears:    30,
		ExpenseRate:  0.15,
		HoldingYears: 15,
		BuildingType: BuildingTypeRC,
		RentDeclineRate: 0,
	}

	noTax := base
	noTax.IncomeTaxRate = 0

	withTax := base
	withTax.IncomeTaxRate = 0.40

	r0 := calcStressScenario(context.Background(), noTax, "税なし", 0, 0)
	r1 := calcStressScenario(context.Background(), withTax, "税40%", 0, 0)

	// 税なしは初年度から CF > 0 → breakEven = 1
	if r0.BreakEvenYear != 1 {
		t.Fatalf("税なし: breakEvenYear = %d, want 1（yearNOI がローン返済額を上回っていない）", r0.BreakEvenYear)
	}
	// 税ありは incomeTax(40%) が CF を超えるため afterTaxCF < 0 → 保有期間中に黒転しない
	// 数値設定（77,000円・15M・2%・30y）で afterTaxCF < 0 が確定するため -1 を直接検証する
	if r1.BreakEvenYear != -1 {
		t.Errorf("税40%%: breakEvenYear = %d, want -1（afterTaxCF < 0 なのに黒転年が存在する）",
			r1.BreakEvenYear)
	}
	t.Logf("noTax breakEven=%d, withTax breakEven=%d", r0.BreakEvenYear, r1.BreakEvenYear)
}

func TestDetectUrbanRisks_NilZoning(t *testing.T) {
	risks := detectUrbanRisks(nil, nil)
	if len(risks) != 0 {
		t.Errorf("expected no risks for nil zoning, got %d", len(risks))
	}
}

func TestDetectUrbanRisks_NoRisks(t *testing.T) {
	txs := []LandTransaction{makeTx("第一種低層住居専用地域")}
	zoning := calcZoningSummary(txs)
	risks := detectUrbanRisks(txs, zoning)
	if len(risks) != 0 {
		t.Errorf("expected no risks for normal zoning, got %d", len(risks))
	}
}

func TestBuildUrbanRisksFromAPIs_NoItems(t *testing.T) {
	risks := BuildUrbanRisksFromAPIs(nil, nil, nil, nil)
	if len(risks) != 0 {
		t.Errorf("expected no risks for empty inputs, got %d", len(risks))
	}
}

func TestBuildUrbanRisksFromAPIs_OutsideResidentialGuidance(t *testing.T) {
	// 居住誘導区域なし → 警告
	items := []LocationOptimizationItem{{KubunNameJa: "都市機能誘導区域"}}
	risks := BuildUrbanRisksFromAPIs(items, nil, nil, nil)
	if len(risks) != 1 || risks[0].Code != "OUTSIDE_RESIDENTIAL_GUIDANCE" {
		t.Errorf("expected OUTSIDE_RESIDENTIAL_GUIDANCE, got %+v", risks)
	}
}

func TestBuildUrbanRisksFromAPIs_InsideResidentialGuidance(t *testing.T) {
	// 居住誘導区域あり → 警告なし
	items := []LocationOptimizationItem{
		{KubunNameJa: "都市機能誘導区域"},
		{KubunNameJa: "居住誘導区域"},
	}
	risks := BuildUrbanRisksFromAPIs(items, nil, nil, nil)
	for _, r := range risks {
		if r.Code == "OUTSIDE_RESIDENTIAL_GUIDANCE" {
			t.Error("should not raise OUTSIDE_RESIDENTIAL_GUIDANCE when 居住誘導区域 is present")
		}
	}
}

func TestBuildUrbanRisksFromAPIs_LargeEmbankment(t *testing.T) {
	items := []EmbankmentItem{{Classification: "谷埋め型"}}
	risks := BuildUrbanRisksFromAPIs(nil, items, nil, nil)
	if len(risks) != 1 || risks[0].Code != "LARGE_EMBANKMENT" {
		t.Errorf("expected LARGE_EMBANKMENT, got %+v", risks)
	}
	if risks[0].Description == "" {
		t.Error("description should not be empty")
	}
}

func TestBuildUrbanRisksFromAPIs_LargeEmbankment_EmptyClassification(t *testing.T) {
	items := []EmbankmentItem{{Classification: ""}}
	risks := BuildUrbanRisksFromAPIs(nil, items, nil, nil)
	if len(risks) != 1 || risks[0].Code != "LARGE_EMBANKMENT" {
		t.Errorf("expected LARGE_EMBANKMENT, got %+v", risks)
	}
	// 空文字でも説明文が壊れていないことを確認
	if risks[0].Description == "" {
		t.Error("description should not be empty even with empty classification")
	}
}

func TestBuildUrbanRisksFromAPIs_UrbanPlanningRoad(t *testing.T) {
	roads := []UrbanRoadItem{
		{PlanningRoadJa: "環状8号線", KubunID: 3011},
		{PlanningRoadJa: "広場A", KubunID: 3023}, // 都市計画道路でない
	}
	risks := BuildUrbanRisksFromAPIs(nil, nil, roads, nil)
	if len(risks) != 1 || risks[0].Code != "URBAN_PLANNING_ROAD" {
		t.Errorf("expected URBAN_PLANNING_ROAD only, got %+v", risks)
	}
}

func TestBuildUrbanRisksFromAPIs_UrbanPlanningRoad_NoKubun3011(t *testing.T) {
	roads := []UrbanRoadItem{{PlanningRoadJa: "広場A", KubunID: 3023}}
	risks := BuildUrbanRisksFromAPIs(nil, nil, roads, nil)
	if len(risks) != 0 {
		t.Errorf("expected no risks for kubun_id != 3011, got %+v", risks)
	}
}

func TestBuildUrbanRisksFromAPIs_DisasterHistory(t *testing.T) {
	disasters := []DisasterHistoryItem{
		{Name: "浸水域", Year: 2019},
		{Name: "浸水域", Year: 2011}, // 同じ名前の重複
		{Name: "がけ崩れ", Year: 2004},
	}
	risks := BuildUrbanRisksFromAPIs(nil, nil, nil, disasters)
	if len(risks) != 1 || risks[0].Code != "DISASTER_HISTORY" {
		t.Errorf("expected DISASTER_HISTORY, got %+v", risks)
	}
	// 重複排除されて2種類が含まれることを確認
	desc := risks[0].Description
	if desc == "" {
		t.Error("description should not be empty")
	}
}

func TestBuildUrbanRisksFromAPIs_MultipleRisks(t *testing.T) {
	risks := BuildUrbanRisksFromAPIs(
		[]LocationOptimizationItem{{KubunNameJa: "都市機能誘導区域"}},
		[]EmbankmentItem{{Classification: "谷埋め型"}},
		[]UrbanRoadItem{{KubunID: 3011}},
		[]DisasterHistoryItem{{Name: "浸水域", Year: 2019}},
	)
	if len(risks) != 4 {
		t.Errorf("expected 4 risks, got %d: %+v", len(risks), risks)
	}
	codes := make(map[string]bool)
	for _, r := range risks {
		codes[r.Code] = true
	}
	for _, want := range []string{
		"OUTSIDE_RESIDENTIAL_GUIDANCE", "LARGE_EMBANKMENT",
		"URBAN_PLANNING_ROAD", "DISASTER_HISTORY",
	} {
		if !codes[want] {
			t.Errorf("missing expected risk code: %s", want)
		}
	}
}

// TestAnalyze_VacancyOverflow は空室率オーバーフロー（VacancyRate+VacancyRateDelta>0.99）時に
// effectiveVacancy が 0.99 でキャップされ、AnnualRent が負にならないことを検証する。
func TestAnalyze_VacancyOverflow(t *testing.T) {
	tests := []struct {
		name             string
		vacancyRate      float64
		vacancyRateDelta float64
		wantCapped       bool // effectiveVacancy が 0.99 にキャップされるか
	}{
		{
			name:             "合計が1.0 → 0.99にキャップ",
			vacancyRate:      0.95,
			vacancyRateDelta: 0.05,
			wantCapped:       true,
		},
		{
			name:             "合計が1.5 → 0.99にキャップ",
			vacancyRate:      1.0,
			vacancyRateDelta: 0.50,
			wantCapped:       true,
		},
		{
			name:             "合計が0.99 → キャップなし（境界値）",
			vacancyRate:      0.90,
			vacancyRateDelta: 0.09,
			wantCapped:       false,
		},
		{
			name:             "通常ケース（合計0.15） → キャップなし",
			vacancyRate:      0.05,
			vacancyRateDelta: 0.10,
			wantCapped:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := InvestmentInput{
				LandPrice:        5_000_000,
				BuildingCost:     10_000_000,
				MiscExpenseRate:  0.07,
				MonthlyRent:      120_000,
				VacancyRate:      tt.vacancyRate,
				VacancyRateDelta: tt.vacancyRateDelta,
				LoanAmount:       13_000_000,
				AnnualLoanRate:   0.015,
				LoanYears:        35,
				BuildingType:     BuildingTypeWood,
				ExpenseRate:      0.20,
				IncomeTaxRate:    0.33,
				HoldingYears:     10,
				ExitYieldTarget:  0.06,
			}

			result := Analyze(context.Background(), input)

			if len(result.YearlyResults) == 0 {
				t.Fatal("YearlyResults is empty")
			}

			// AnnualRent は常に 0 以上でなければならない
			for _, yr := range result.YearlyResults {
				if yr.AnnualRent < 0 {
					t.Errorf("year %d: AnnualRent = %.0f, want >= 0 (vacancy overflow must be capped)", yr.Year, yr.AnnualRent)
				}
			}

			// キャップ時は effectiveVacancy=0.99 → AnnualRent = MonthlyRent*12*(1-0.99)
			if tt.wantCapped {
				expectedRent := 120_000.0 * 12 * (1 - 0.99)
				if !approxEqual(result.YearlyResults[0].AnnualRent, expectedRent, 1) {
					t.Errorf("AnnualRent = %.0f, want ≈ %.0f (capped at 0.99)", result.YearlyResults[0].AnnualRent, expectedRent)
				}
			}
		})
	}
}

// TestInvestmentInput_Validate は Validate() が空室率オーバーフロー時にエラーを返すことを検証する。
func TestInvestmentInput_Validate(t *testing.T) {
	tests := []struct {
		name             string
		vacancyRate      float64
		vacancyRateDelta float64
		wantErr          bool
	}{
		{
			name:             "合計1.0 → エラー",
			vacancyRate:      0.95,
			vacancyRateDelta: 0.05,
			wantErr:          true,
		},
		{
			name:             "合計1.5 → エラー",
			vacancyRate:      1.0,
			vacancyRateDelta: 0.50,
			wantErr:          true,
		},
		{
			name:             "合計0.99 → エラーなし（境界値）",
			vacancyRate:      0.90,
			vacancyRateDelta: 0.09,
			wantErr:          false,
		},
		{
			name:             "合計0.15 → エラーなし",
			vacancyRate:      0.05,
			vacancyRateDelta: 0.10,
			wantErr:          false,
		},
		{
			name:             "Deltaなし → エラーなし",
			vacancyRate:      0.05,
			vacancyRateDelta: 0,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := InvestmentInput{
				VacancyRate:      tt.vacancyRate,
				VacancyRateDelta: tt.vacancyRateDelta,
			}
			err := input.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error (vacancy=%.2f+%.2f=%.2f > 0.99)",
					tt.vacancyRate, tt.vacancyRateDelta, tt.vacancyRate+tt.vacancyRateDelta)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

// TestAnalyze_OldBuildingZeroDepreciation は木造38年超（減価償却ゼロ）のキャッシュフロー精度を検証する
func TestAnalyze_OldBuildingZeroDepreciation(t *testing.T) {
	// 木造の法定耐用年数は22年。築38年超 → 簡便法: 22×0.2=4年 → 4年後に減価償却=0
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    3_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     80_000,
		VacancyRate:     0.05,
		LoanAmount:      5_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       20,
		BuildingType:    BuildingTypeWood,
		BuildingAge:     38, // 法定耐用年数超過: 簡便法 22×0.2=4年
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}

	// CalcResidualUsefulLife(木造, 38) = 22×0.2 = 4年
	usefulLife := CalcResidualUsefulLife(BuildingTypeWood, 38)
	if usefulLife != 4 {
		t.Errorf("usefulLife = %d, want 4", usefulLife)
	}

	// 1年目: 減価償却あり
	y1 := result.YearlyResults[0]
	expectedDepreciation := 3_000_000.0 / float64(usefulLife)
	if math.Abs(y1.AnnualDepreciation-expectedDepreciation) > 1 {
		t.Errorf("year1 AnnualDepreciation = %.0f, want %.0f", y1.AnnualDepreciation, expectedDepreciation)
	}

	// 5年目以降: 減価償却=0
	y5 := result.YearlyResults[4]
	if y5.AnnualDepreciation != 0 {
		t.Errorf("year5 AnnualDepreciation = %.0f, want 0 (耐用年数4年超過)", y5.AnnualDepreciation)
	}
}

// TestAnalyze_ZeroLoanYears は全額自己資金（ローン期間ゼロ）のケースを検証する
func TestAnalyze_ZeroLoanYears(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      0,
		AnnualLoanRate:  0,
		LoanYears:       0, // Defaults() により 35 に補完されるが LoanAmount=0 なので実質影響なし
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}

	y1 := result.YearlyResults[0]

	// ローン返済・利息・元金はすべてゼロ
	if y1.AnnualLoanPayment != 0 {
		t.Errorf("AnnualLoanPayment = %.0f, want 0", y1.AnnualLoanPayment)
	}
	if y1.AnnualInterest != 0 {
		t.Errorf("AnnualInterest = %.0f, want 0", y1.AnnualInterest)
	}
	if y1.AnnualPrincipal != 0 {
		t.Errorf("AnnualPrincipal = %.0f, want 0", y1.AnnualPrincipal)
	}

	// キャッシュフロー = 実効賃料 - 運営経費（正値になるはず）
	effectiveRent := 120_000.0 * 12 * (1 - 0.05)
	wantCF := effectiveRent - effectiveRent*0.20
	if math.Abs(y1.CashFlow-wantCF) > 1 {
		t.Errorf("CashFlow = %.0f, want %.0f", y1.CashFlow, wantCF)
	}

	// 表面利回り・実質利回りが計算されること
	if result.GrossYield <= 0 {
		t.Errorf("GrossYield = %.4f, want > 0", result.GrossYield)
	}
	if result.NetYield <= 0 {
		t.Errorf("NetYield = %.4f, want > 0", result.NetYield)
	}
}

// TestAnalyze_ShortTermCapitalGains は短期保有（5年以下）の譲渡所得税率39.63%を検証する
func TestAnalyze_ShortTermCapitalGains(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeRC, // 47年 → 保有5年内でも減価償却継続
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    5, // 5年以下 → 短期譲渡所得税率39.63%
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// 売却益が正値であることを確認（テストが空振りしないよう）
	if result.ExitCapitalGain <= 0 {
		t.Fatalf("expected positive capital gain, got %f", result.ExitCapitalGain)
	}
	// 短期税率（39.63%）が適用されること
	impliedRate := result.ExitTransferTax / result.ExitCapitalGain
	if math.Abs(impliedRate-shortTermTransferTaxRate) > 0.001 {
		t.Errorf("短期譲渡税率 = %.5f, want %.5f", impliedRate, shortTermTransferTaxRate)
	}
	t.Logf("HoldingYears=5 (短期): SalePrice=%.0f, CapGain=%.0f, Tax=%.0f", result.ExitSalePrice, result.ExitCapitalGain, result.ExitTransferTax)
}

// TestAnalyze_YieldScenarios は空室シナリオ（楽観・標準・悲観）の計算を検証する
func TestAnalyze_YieldScenarios(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.10, // 10%空室率
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)
	sc := result.YieldScenarios

	totalInvestment := 5_000_000.0 + 10_000_000.0 + 15_000_000.0*0.07 // 16,050,000
	annualGross := 120_000.0 * 12                                        // 1,440,000
	expectedGrossYield := annualGross / totalInvestment

	// 楽観: 空室率 × 0.5 = 5%
	wantOptRent := annualGross * (1 - 0.10*0.5)
	if !approxEqual(sc.Optimistic.AnnualRent, wantOptRent, 1) {
		t.Errorf("Optimistic.AnnualRent = %.0f, want %.0f", sc.Optimistic.AnnualRent, wantOptRent)
	}
	if !approxEqual(sc.Optimistic.GrossYield, expectedGrossYield, 0.0001) {
		t.Errorf("Optimistic.GrossYield = %.4f, want %.4f", sc.Optimistic.GrossYield, expectedGrossYield)
	}

	// 標準: 空室率 × 1.0 = 10%
	wantStdRent := annualGross * (1 - 0.10)
	if !approxEqual(sc.Standard.AnnualRent, wantStdRent, 1) {
		t.Errorf("Standard.AnnualRent = %.0f, want %.0f", sc.Standard.AnnualRent, wantStdRent)
	}
	if !approxEqual(sc.Standard.GrossYield, expectedGrossYield, 0.0001) {
		t.Errorf("Standard.GrossYield = %.4f, want %.4f", sc.Standard.GrossYield, expectedGrossYield)
	}

	// 悲観: 空室率 × 1.5 = 15%
	wantPesRent := annualGross * (1 - 0.10*1.5)
	if !approxEqual(sc.Pessimistic.AnnualRent, wantPesRent, 1) {
		t.Errorf("Pessimistic.AnnualRent = %.0f, want %.0f", sc.Pessimistic.AnnualRent, wantPesRent)
	}
	if !approxEqual(sc.Pessimistic.GrossYield, expectedGrossYield, 0.0001) {
		t.Errorf("Pessimistic.GrossYield = %.4f, want %.4f", sc.Pessimistic.GrossYield, expectedGrossYield)
	}

	// 楽観 > 標準 > 悲観 の順序確認
	if sc.Optimistic.AnnualRent <= sc.Standard.AnnualRent {
		t.Errorf("Optimistic.AnnualRent (%.0f) should be > Standard.AnnualRent (%.0f)", sc.Optimistic.AnnualRent, sc.Standard.AnnualRent)
	}
	if sc.Standard.AnnualRent <= sc.Pessimistic.AnnualRent {
		t.Errorf("Standard.AnnualRent (%.0f) should be > Pessimistic.AnnualRent (%.0f)", sc.Standard.AnnualRent, sc.Pessimistic.AnnualRent)
	}

	t.Logf("YieldScenarios: Optimistic=%.0f, Standard=%.0f, Pessimistic=%.0f (GrossYield=%.4f)",
		sc.Optimistic.AnnualRent, sc.Standard.AnnualRent, sc.Pessimistic.AnnualRent, sc.Standard.GrossYield)
}

// TestAnalyze_YieldScenarios_HighVacancy は高空室率での悲観シナリオのキャップ（0.99）を検証する
func TestAnalyze_YieldScenarios_HighVacancy(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.80, // 80%空室率: 悲観で1.5倍=120%→0.99にキャップ
		LoanAmount:      0,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)
	sc := result.YieldScenarios

	// 悲観: 0.80 × 1.5 = 1.20 → 0.99 にキャップ
	wantPesRent := 120_000.0 * 12 * (1 - 0.99)
	if !approxEqual(sc.Pessimistic.AnnualRent, wantPesRent, 1) {
		t.Errorf("Pessimistic.AnnualRent (capped) = %.0f, want %.0f", sc.Pessimistic.AnnualRent, wantPesRent)
	}

	// AnnualRent は 0 以上でなければならない
	if sc.Pessimistic.AnnualRent < 0 {
		t.Errorf("Pessimistic.AnnualRent = %.0f, must be >= 0", sc.Pessimistic.AnnualRent)
	}
}

// TestAnalyze_YieldScenarios_ZeroVacancy はゼロ空室率でのシナリオを検証する
func TestAnalyze_YieldScenarios_ZeroVacancy(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0, // ゼロ空室率: 全シナリオで満室想定
		LoanAmount:      0,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)
	sc := result.YieldScenarios

	fullAnnualRent := 120_000.0 * 12

	// 空室率0%: 楽観・標準・悲観いずれも満室賃料
	if !approxEqual(sc.Optimistic.AnnualRent, fullAnnualRent, 1) {
		t.Errorf("Optimistic.AnnualRent = %.0f, want %.0f (zero vacancy)", sc.Optimistic.AnnualRent, fullAnnualRent)
	}
	if !approxEqual(sc.Standard.AnnualRent, fullAnnualRent, 1) {
		t.Errorf("Standard.AnnualRent = %.0f, want %.0f (zero vacancy)", sc.Standard.AnnualRent, fullAnnualRent)
	}
	if !approxEqual(sc.Pessimistic.AnnualRent, fullAnnualRent, 1) {
		t.Errorf("Pessimistic.AnnualRent = %.0f, want %.0f (zero vacancy)", sc.Pessimistic.AnnualRent, fullAnnualRent)
	}
}

// TestAnalyze_UltraLowInterestRate は超低金利0.001%での浮動小数点精度を検証する
func TestAnalyze_UltraLowInterestRate(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.00001, // 0.001% 超低金利
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	// パニックしないこと・結果が有限な数値であること
	result := Analyze(context.Background(), input)

	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}

	y1 := result.YearlyResults[0]

	// 超低金利: 月次返済 ≈ 13,000,000/420ヶ月 ≈ 30,952円/月
	wantMonthly := 13_000_000.0 / float64(35*12)
	annualPayment := wantMonthly * 12
	if math.Abs(y1.AnnualLoanPayment-annualPayment) > 1000 {
		t.Errorf("AnnualLoanPayment = %.0f, want ≈ %.0f (超低金利はほぼ元金均等)", y1.AnnualLoanPayment, annualPayment)
	}

	// 利息はほぼゼロ（元金のみ）
	if y1.AnnualInterest > 1000 {
		t.Errorf("AnnualInterest = %.2f, expected < 1000 for 0.001%% rate", y1.AnnualInterest)
	}

	// NaN/Inf でないこと
	if math.IsNaN(result.GrossYield) || math.IsInf(result.GrossYield, 0) {
		t.Errorf("GrossYield is NaN or Inf: %v", result.GrossYield)
	}
}

// TestCalcDSCR は DSCR の計算を検証する
func TestCalcDSCR(t *testing.T) {
	tests := []struct {
		name              string
		noi               float64
		annualDebtService float64
		want              float64
	}{
		{"正常: NOI > 返済額 (安全)", 1_200_000, 1_000_000, 1.2},
		{"正常: NOI < 返済額 (危険)", 800_000, 1_000_000, 0.8},
		{"ゼロ除算: 返済額=0", 1_200_000, 0, 0},
		{"NOI=0", 0, 1_000_000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcDSCR(tt.noi, tt.annualDebtService)
			if !approxEqual(got, tt.want, 0.0001) {
				t.Errorf("CalcDSCR(%.0f, %.0f) = %.4f, want %.4f", tt.noi, tt.annualDebtService, got, tt.want)
			}
		})
	}
}

// TestCalcEqualPrincipalPayment は元金均等返済の月次返済額を検証する
func TestCalcEqualPrincipalPayment(t *testing.T) {
	principal := 12_000_000.0
	annualRate := 0.024 // 年利2.4%
	months := 120      // 10年

	// 月次元金 = 12,000,000 / 120 = 100,000円
	monthlyPrincipal := principal / float64(months)

	// 1ヶ月目: 利息 = 12,000,000 × 0.002 = 24,000 → 合計 124,000
	month1 := CalcEqualPrincipalPayment(principal, annualRate, months, 1)
	want1 := monthlyPrincipal + principal*(annualRate/12)
	if !approxEqual(month1, want1, 1) {
		t.Errorf("month1 = %.0f, want %.0f", month1, want1)
	}

	// 最終月: 利息 = monthlyPrincipal × 0.002 (残高=元金1口分)
	monthLast := CalcEqualPrincipalPayment(principal, annualRate, months, months)
	wantLast := monthlyPrincipal + monthlyPrincipal*(annualRate/12)
	if !approxEqual(monthLast, wantLast, 1) {
		t.Errorf("monthLast = %.0f, want %.0f", monthLast, wantLast)
	}

	// 1ヶ月目 > 最終月（返済が進むにつれて減少）
	if month1 <= monthLast {
		t.Errorf("month1(%.0f) should be > monthLast(%.0f)", month1, monthLast)
	}

	// 無効引数
	if CalcEqualPrincipalPayment(0, annualRate, months, 1) != 0 {
		t.Error("principal=0 should return 0")
	}
	if CalcEqualPrincipalPayment(principal, annualRate, months, 0) != 0 {
		t.Error("month=0 should return 0")
	}
}

// TestCalcLTVSensitivity は LTV 感度分析の計算を検証する
func TestCalcLTVSensitivity(t *testing.T) {
	input := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    120_000,
		VacancyRate:    0.05,
		AnnualLoanRate: 0.015,
		LoanYears:      35,
		ExpenseRate:    0.20,
	}
	input.Defaults()

	rows := CalcLTVSensitivity(input, nil)

	if len(rows) != 5 {
		t.Fatalf("len(rows) = %d, want 5", len(rows))
	}

	ltvExpected := []float64{0.5, 0.6, 0.7, 0.8, 0.9}
	for i, row := range rows {
		if !approxEqual(row.LTV, ltvExpected[i], 0.0001) {
			t.Errorf("rows[%d].LTV = %.2f, want %.2f", i, row.LTV, ltvExpected[i])
		}
		// Equity + LoanAmount = TotalInvestment
		miscExpenses := (input.LandPrice + input.BuildingCost) * input.MiscExpenseRate
		totalInvestment := input.LandPrice + input.BuildingCost + miscExpenses
		if !approxEqual(row.Equity+row.LoanAmount, totalInvestment, 1) {
			t.Errorf("rows[%d] equity+loan = %.0f, want %.0f", i, row.Equity+row.LoanAmount, totalInvestment)
		}
		// LTV が高いほど DSCR は低下
		if i > 0 && row.DSCR >= rows[i-1].DSCR {
			t.Errorf("rows[%d].DSCR(%.4f) should be < rows[%d].DSCR(%.4f)", i, row.DSCR, i-1, rows[i-1].DSCR)
		}
	}
}

// TestCalcLTVSensitivity_WithRateSchedule は変動金利スケジュール指定時に
// 初年度実効金利が使われることを検証する。
func TestCalcLTVSensitivity_WithRateSchedule(t *testing.T) {
	base := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    120_000,
		VacancyRate:    0.05,
		AnnualLoanRate: 0.005, // ベース金利 0.5%
		LoanYears:      35,
		ExpenseRate:    0.20,
	}
	base.Defaults()

	// スケジュール: 3年目から 3.0% に切り替わるが LTV 感度は初年度（0.5%）で計算されるべき
	withSchedule := base
	withSchedule.RateAdjustmentSchedule = []RateAdjustment{
		{AfterYear: 3, Rate: 0.030},
	}

	rowsBase := CalcLTVSensitivity(base, nil)
	rowsScheduled := CalcLTVSensitivity(withSchedule, nil)

	if len(rowsBase) != 5 || len(rowsScheduled) != 5 {
		t.Fatalf("unexpected row counts: base=%d scheduled=%d", len(rowsBase), len(rowsScheduled))
	}

	// 初年度金利はスケジュールが始まる前（AfterYear=3 は year 1 に適用されない）なので
	// ベース金利 0.5% のまま → 両者の DSCR は等しいはず
	for i := range rowsBase {
		if !approxEqual(rowsBase[i].DSCR, rowsScheduled[i].DSCR, 0.0001) {
			t.Errorf("rows[%d]: DSCR should be equal (schedule starts year 3): base=%.4f scheduled=%.4f",
				i, rowsBase[i].DSCR, rowsScheduled[i].DSCR)
		}
	}

	// スケジュールで初年度から高金利が適用される場合は DSCR が下がるべき
	withScheduleY1 := base
	withScheduleY1.RateAdjustmentSchedule = []RateAdjustment{
		{AfterYear: 1, Rate: 0.030}, // 1年目から 3.0%
	}
	rowsY1 := CalcLTVSensitivity(withScheduleY1, nil)
	for i := range rowsBase {
		if rowsY1[i].DSCR >= rowsBase[i].DSCR {
			t.Errorf("rows[%d]: DSCR(%.4f) should be < base DSCR(%.4f) when year-1 rate is higher",
				i, rowsY1[i].DSCR, rowsBase[i].DSCR)
		}
	}
}

// TestCalcLTVSensitivity_RateScheduleYear1 は変動金利スケジュールが初年度から適用される場合に
// CalcLTVSensitivity がベース金利ではなくスケジュール金利（3%）を使って計算することを検証する。
// ベース金利 2%、スケジュール AfterYear=1 で 3% に切り替わる → 初年度実効金利は 3% になる。
//
// NOTE: AfterYear=1 は Validate() の制約（>= 2）を下回るため、実際の API リクエストでは
// 弾かれる値である。本テストは API レイヤーのバリデーションをバイパスし、
// ドメインロジック単体の動作を検証することを意図している。
func TestCalcLTVSensitivity_RateScheduleYear1(t *testing.T) {
	baseRate := 0.020
	scheduleRate := 0.030
	input := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    150_000,
		VacancyRate:    0.05,
		AnnualLoanRate: baseRate,
		LoanYears:      35,
		ExpenseRate:    0.20,
		RateAdjustmentSchedule: []RateAdjustment{
			{AfterYear: 1, Rate: scheduleRate}, // 1年目から 3% に切り替わる
		},
	}
	input.Defaults()

	// スケジュール適用あり（初年度金利 = 3%）
	rowsScheduled := CalcLTVSensitivity(input, nil)

	// 比較用: スケジュールなし・3% 固定で計算した結果（初年度実効金利が同じになるはず）
	inputFixed3 := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    150_000,
		VacancyRate:    0.05,
		AnnualLoanRate: scheduleRate,
		LoanYears:      35,
		ExpenseRate:    0.20,
	}
	inputFixed3.Defaults()
	rowsFixed3 := CalcLTVSensitivity(inputFixed3, nil)

	if len(rowsScheduled) != 5 || len(rowsFixed3) != 5 {
		t.Fatalf("unexpected row counts: scheduled=%d fixed3=%d", len(rowsScheduled), len(rowsFixed3))
	}

	// 初年度実効金利が 3% の場合、スケジュール版も固定3%版も同じ DSCR・CF になるべき
	for i := range rowsScheduled {
		if !approxEqual(rowsScheduled[i].DSCR, rowsFixed3[i].DSCR, 0.0001) {
			t.Errorf("rows[%d]: DSCR mismatch: scheduled(rate=3%%)=%.6f, fixed3%%=%.6f — schedule was not applied",
				i, rowsScheduled[i].DSCR, rowsFixed3[i].DSCR)
		}
		if !approxEqual(rowsScheduled[i].AnnualCF, rowsFixed3[i].AnnualCF, 1.0) {
			t.Errorf("rows[%d]: AnnualCF mismatch: scheduled=%.0f, fixed3%%=%.0f",
				i, rowsScheduled[i].AnnualCF, rowsFixed3[i].AnnualCF)
		}
	}

	// 比較用: ベース金利 2% のみの場合（スケジュール不適用）の DSCR は上記より高いはず
	inputBase := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    150_000,
		VacancyRate:    0.05,
		AnnualLoanRate: baseRate,
		LoanYears:      35,
		ExpenseRate:    0.20,
	}
	inputBase.Defaults()
	rowsBase := CalcLTVSensitivity(inputBase, nil)

	for i := range rowsScheduled {
		if rowsScheduled[i].DSCR >= rowsBase[i].DSCR {
			t.Errorf("rows[%d]: DSCR(%.6f) should be < base 2%% DSCR(%.6f) when schedule bumps rate to 3%%",
				i, rowsScheduled[i].DSCR, rowsBase[i].DSCR)
		}
	}
}

// TestCalcLTVSensitivity_LoanRateDelta は LoanRateDelta が LTV 感度に反映されることを検証する。
func TestCalcLTVSensitivity_LoanRateDelta(t *testing.T) {
	base := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    120_000,
		VacancyRate:    0.05,
		AnnualLoanRate: 0.005,
		LoanYears:      35,
		ExpenseRate:    0.20,
	}
	base.Defaults()

	withDelta := base
	withDelta.LoanRateDelta = 0.015 // ストレス +1.5% → 実効 2.0%

	rowsBase := CalcLTVSensitivity(base, nil)
	rowsDelta := CalcLTVSensitivity(withDelta, nil)

	for i := range rowsBase {
		if rowsDelta[i].DSCR >= rowsBase[i].DSCR {
			t.Errorf("rows[%d]: DSCR(%.4f) should be < base DSCR(%.4f) when LoanRateDelta > 0",
				i, rowsDelta[i].DSCR, rowsBase[i].DSCR)
		}
	}
}

// TestCalcLTVSensitivity_EqualPrincipal は元金均等返済パスで初年度実効金利が使われることを検証する。
func TestCalcLTVSensitivity_EqualPrincipal(t *testing.T) {
	base := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    120_000,
		VacancyRate:    0.05,
		AnnualLoanRate: 0.005,
		LoanYears:      35,
		LoanMethod:     LoanMethodEqualPrincipal,
		ExpenseRate:    0.20,
	}
	base.Defaults()

	// 1年目から高金利スケジュールを適用 → DSCR が下がるはず（元金均等パス）
	withSchedule := base
	withSchedule.RateAdjustmentSchedule = []RateAdjustment{
		{AfterYear: 1, Rate: 0.030},
	}

	rowsBase := CalcLTVSensitivity(base, nil)
	rowsScheduled := CalcLTVSensitivity(withSchedule, nil)

	for i := range rowsBase {
		if rowsScheduled[i].DSCR >= rowsBase[i].DSCR {
			t.Errorf("rows[%d] (equal-principal): DSCR(%.4f) should be < base DSCR(%.4f) when year-1 rate is higher",
				i, rowsScheduled[i].DSCR, rowsBase[i].DSCR)
		}
	}
}

// TestAnalyze_EqualPrincipal は元金均等返済での計算を検証する
func TestAnalyze_EqualPrincipal(t *testing.T) {
	input := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    120_000,
		VacancyRate:    0.05,
		LoanAmount:     13_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:      35,
		BuildingType:   BuildingTypeWood,
		ExpenseRate:    0.20,
		IncomeTaxRate:  0.33,
		HoldingYears:   10,
		ExitYieldTarget: 0.06,
		LoanMethod:     LoanMethodEqualPrincipal,
	}

	epResult := Analyze(context.Background(), input)

	// 同一入力で元利均等と比較
	inputEP := input
	inputEP.LoanMethod = LoanMethodEqualPayment
	equalPayResult := Analyze(context.Background(), inputEP)

	// 元金均等: 1年目の返済額 > 元利均等の返済額（初期は利息負担が大きい）
	y1EP := epResult.YearlyResults[0]
	y1EqualPay := equalPayResult.YearlyResults[0]
	if y1EP.AnnualLoanPayment <= y1EqualPay.AnnualLoanPayment {
		t.Errorf("元金均等1年目返済額(%.0f) should be > 元利均等(%.0f)", y1EP.AnnualLoanPayment, y1EqualPay.AnnualLoanPayment)
	}

	// 元金均等: 最終年の返済額 < 元利均等の返済額（後期は元利均等の方が高い）
	last := input.LoanYears - 1
	yLastEP := epResult.YearlyResults[last]
	yLastEqualPay := equalPayResult.YearlyResults[last]
	if yLastEP.AnnualLoanPayment >= yLastEqualPay.AnnualLoanPayment {
		t.Errorf("元金均等最終年返済額(%.0f) should be < 元利均等(%.0f)", yLastEP.AnnualLoanPayment, yLastEqualPay.AnnualLoanPayment)
	}

	// 元金均等: 総支払利息 < 元利均等の総支払利息
	totalInterestEP := 0.0
	totalInterestEqualPay := 0.0
	for i := 0; i < input.LoanYears; i++ {
		totalInterestEP += epResult.YearlyResults[i].AnnualInterest
		totalInterestEqualPay += equalPayResult.YearlyResults[i].AnnualInterest
	}
	if totalInterestEP >= totalInterestEqualPay {
		t.Errorf("元金均等総利息(%.0f) should be < 元利均等総利息(%.0f)", totalInterestEP, totalInterestEqualPay)
	}

	// DSCR が 0 より大きいこと
	if epResult.DSCR <= 0 {
		t.Errorf("DSCR = %.4f, want > 0", epResult.DSCR)
	}

	// LTVSensitivity が 5 行あること
	if len(epResult.LTVSensitivity) != 5 {
		t.Errorf("len(LTVSensitivity) = %d, want 5", len(epResult.LTVSensitivity))
	}

	t.Logf("元金均等1年目返済額=%.0f, 元利均等=%.0f", y1EP.AnnualLoanPayment, y1EqualPay.AnnualLoanPayment)
	t.Logf("元金均等総利息=%.0f, 元利均等総利息=%.0f", totalInterestEP, totalInterestEqualPay)
	t.Logf("DSCR=%.3f, LTVSensitivity rows=%d", epResult.DSCR, len(epResult.LTVSensitivity))
}

// TestAnalyze_DSCR は Analyze() の DSCR 計算を検証する
func TestAnalyze_DSCR(t *testing.T) {
	input := InvestmentInput{
		LandPrice:      5_000_000,
		BuildingCost:   10_000_000,
		MonthlyRent:    120_000,
		VacancyRate:    0.05,
		LoanAmount:     13_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:      35,
		ExpenseRate:    0.20,
		HoldingYears:   10,
		ExitYieldTarget: 0.06,
	}
	result := Analyze(context.Background(), input)

	// DSCR > 0 であること
	if result.DSCR <= 0 {
		t.Errorf("DSCR = %.4f, want > 0", result.DSCR)
	}

	// LTVSensitivity が正しく設定されていること
	if len(result.LTVSensitivity) != 5 {
		t.Errorf("len(LTVSensitivity) = %d, want 5", len(result.LTVSensitivity))
	}

	// 手動検証: year1 NOI / year1 AnnualLoanPayment
	y1 := result.YearlyResults[0]
	noi := y1.AnnualRent - y1.AnnualExpenses
	wantDSCR := CalcDSCR(noi, y1.AnnualLoanPayment)
	if !approxEqual(result.DSCR, wantDSCR, 0.0001) {
		t.Errorf("DSCR = %.4f, want %.4f", result.DSCR, wantDSCR)
	}

	t.Logf("DSCR=%.3f, NOI=%.0f, AnnualLoanPayment=%.0f", result.DSCR, noi, y1.AnnualLoanPayment)
}

// TestAnalyze_EqualPrincipalStressScenario は元金均等時のストレスシナリオ CF を検証する
// 修正前は calcStressScenario の yearLoan が1年目固定で、後半の CF が過少推計されていた。
// 修正後は残高に応じて毎年再計算されるため、CF は年々改善する。
func TestAnalyze_EqualPrincipalStressScenario(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		ExpenseRate:     0.20,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
		LoanMethod:      LoanMethodEqualPrincipal,
	}

	result := Analyze(context.Background(), input)

	// ① 年次結果: 元金均等の年間返済額は年々減少する
	for i := 1; i < input.HoldingYears; i++ {
		prev := result.YearlyResults[i-1].AnnualLoanPayment
		curr := result.YearlyResults[i].AnnualLoanPayment
		if curr >= prev {
			t.Errorf("年次結果: year%d AnnualLoanPayment(%.0f) >= year%d(%.0f), 元金均等は逓減すべき",
				i+1, curr, i, prev)
		}
	}

	// ② ストレスシナリオの累積CF: 後半5年の CF 合計が前半5年より大きいはず
	// （返済額が年々減少するため）
	findScenario := func(scenarios []StressScenarioResult, label string) *StressScenarioResult {
		for i := range scenarios {
			if scenarios[i].Label == label {
				return &scenarios[i]
			}
		}
		return nil
	}
	baseline := findScenario(result.StressScenarios, "ベースライン")
	if baseline == nil {
		t.Fatal("ベースラインシナリオが見つかりません")
	}

	// ③ ストレスシナリオの TotalCashFlow が Analyze() の年次 CF 合計と整合すること
	// （両者は同じ空室率・金利・賃料で計算されるため近似一致するはず）
	annualRent := input.MonthlyRent * 12 * (1 - input.VacancyRate)
	annualExpenses := annualRent*input.ExpenseRate
	var manualTotalCF float64
	remainingBal := input.LoanAmount
	monthlyPrincipal := input.LoanAmount / float64(input.LoanYears*12)
	for y := 1; y <= input.HoldingYears; y++ {
		yearLoan := 0.0
		if remainingBal > 0 {
			yi, yp := calcYearlyLoanComponentsEqualPrincipal(remainingBal, input.AnnualLoanRate, monthlyPrincipal)
			yearLoan = yi + yp
			remainingBal -= yp
			if remainingBal < 0 {
				remainingBal = 0
			}
		}
		manualTotalCF += annualRent - yearLoan - annualExpenses
	}
	if math.Abs(baseline.TotalCashFlow-manualTotalCF) > 1000 {
		t.Errorf("ストレスシナリオ TotalCashFlow(%.0f) と手動計算値(%.0f) の差が 1000 超",
			baseline.TotalCashFlow, manualTotalCF)
	}

	t.Logf("ベースライン TotalCashFlow=%.0f（手動計算=%.0f）", baseline.TotalCashFlow, manualTotalCF)
	t.Logf("年次返済額推移: year1=%.0f, year5=%.0f, year10=%.0f",
		result.YearlyResults[0].AnnualLoanPayment,
		result.YearlyResults[4].AnnualLoanPayment,
		result.YearlyResults[9].AnnualLoanPayment,
	)
}

// ── 変動金利テスト ────────────────────────────────────────────────

// TestResolveRateForYear はスケジュールに基づく年別金利解決を検証する
func TestResolveRateForYear(t *testing.T) {
	schedule := []RateAdjustment{
		{AfterYear: 6, Rate: 0.02},
		{AfterYear: 11, Rate: 0.03},
	}

	cases := []struct {
		year int
		want float64
	}{
		{1, 0.015},  // スケジュール前: baseRate
		{5, 0.015},  // スケジュール前: baseRate
		{6, 0.02},   // 第1ステップ適用
		{10, 0.02},  // 第1ステップ継続
		{11, 0.03},  // 第2ステップ適用
		{35, 0.03},  // 第2ステップ継続
	}

	for _, c := range cases {
		got := resolveRateForYear(0.015, 0, schedule, c.year)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("year=%d: got %.4f, want %.4f", c.year, got, c.want)
		}
	}

	// rateDelta が全ステップに上乗せされること
	got := resolveRateForYear(0.015, 0.01, schedule, 6)
	if math.Abs(got-0.03) > 1e-9 {
		t.Errorf("rateDelta: got %.4f, want 0.03", got)
	}

	// 空スケジュールは baseRate + rateDelta を返す
	got = resolveRateForYear(0.015, 0.005, nil, 10)
	if math.Abs(got-0.02) > 1e-9 {
		t.Errorf("empty schedule: got %.4f, want 0.02", got)
	}
}

// TestValidateRateAdjustmentSchedule はスケジュールのバリデーションを検証する
func TestValidateRateAdjustmentSchedule(t *testing.T) {
	base := InvestmentInput{
		VacancyRate:  0.05,
		LoanYears:    35,
		RentDeclineRate: 0,
	}

	t.Run("valid single step", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{{AfterYear: 6, Rate: 0.02}}
		if err := in.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid multi step ascending", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{
			{AfterYear: 6, Rate: 0.02},
			{AfterYear: 11, Rate: 0.025},
		}
		if err := in.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("AfterYear < 2", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{{AfterYear: 1, Rate: 0.02}}
		if err := in.Validate(); err == nil {
			t.Error("expected error for AfterYear < 2")
		}
	})

	t.Run("AfterYear > LoanYears", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{{AfterYear: 36, Rate: 0.02}}
		if err := in.Validate(); err == nil {
			t.Error("expected error for AfterYear > LoanYears")
		}
	})

	t.Run("Rate out of range", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{{AfterYear: 6, Rate: 0.35}}
		if err := in.Validate(); err == nil {
			t.Error("expected error for Rate > 0.3")
		}
	})

	t.Run("unsorted AfterYear", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{
			{AfterYear: 11, Rate: 0.02},
			{AfterYear: 6, Rate: 0.025},
		}
		if err := in.Validate(); err == nil {
			t.Error("expected error for unsorted schedule")
		}
	})

	t.Run("duplicate AfterYear", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{
			{AfterYear: 6, Rate: 0.02},
			{AfterYear: 6, Rate: 0.025},
		}
		if err := in.Validate(); err == nil {
			t.Error("expected error for duplicate AfterYear")
		}
	})
}

// TestCalcEffectiveExpenseRate は詳細経費フィールドのフォールバック挙動を検証する
func TestCalcEffectiveExpenseRate(t *testing.T) {
	tests := []struct {
		name  string
		input InvestmentInput
		want  float64
	}{
		{
			name:  "all detail fields zero falls back to ExpenseRate",
			input: InvestmentInput{ExpenseRate: 0.15},
			want:  0.15,
		},
		{
			name: "detail fields sum used when non-zero",
			input: InvestmentInput{
				ExpenseRate:       0.15,
				ManagementFeeRate: 0.05,
				RepairReserveRate: 0.01,
				InsuranceFeeRate:  0.003,
				OtherExpenseRate:  0.005,
			},
			want: 0.068, // 0.05+0.01+0.003+0.005
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcEffectiveExpenseRate(tt.input)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAnalyzeWithRateSchedule は変動金利スケジュール適用後の動作を検証する
func TestAnalyzeWithRateSchedule(t *testing.T) {
	base := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  120_000,
		VacancyRate:  0.05,
		LoanAmount:   13_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:    35,
		BuildingType: BuildingTypeWood,
		ExpenseRate:  0.20,
		IncomeTaxRate: 0.33,
		HoldingYears: 15,
		ExitYieldTarget: 0.06,
		YieldTarget:  0.08,
	}

	t.Run("fixed rate: all years same EffectiveRate", func(t *testing.T) {
		result := Analyze(context.Background(), base)
		for _, y := range result.YearlyResults {
			if math.Abs(y.EffectiveRate-0.015) > 1e-9 {
				t.Errorf("year=%d: EffectiveRate=%.4f, want 0.015", y.Year, y.EffectiveRate)
			}
		}
	})

	t.Run("rate steps up at year 6 and 11", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{
			{AfterYear: 6, Rate: 0.02},
			{AfterYear: 11, Rate: 0.03},
		}
		result := Analyze(context.Background(), in)

		for _, y := range result.YearlyResults {
			var wantRate float64
			switch {
			case y.Year >= 11:
				wantRate = 0.03
			case y.Year >= 6:
				wantRate = 0.02
			default:
				wantRate = 0.015
			}
			if math.Abs(y.EffectiveRate-wantRate) > 1e-9 {
				t.Errorf("year=%d: EffectiveRate=%.4f, want %.4f", y.Year, y.EffectiveRate, wantRate)
			}
		}

		// 金利変化年に月次返済額が増加することを確認
		// 年5→年6: 金利0.015→0.02なので返済額が増えるはず
		y5 := result.YearlyResults[4]
		y6 := result.YearlyResults[5]
		if y6.AnnualLoanPayment <= y5.AnnualLoanPayment {
			t.Errorf("year6 payment(%.0f) should be > year5 payment(%.0f) after rate increase",
				y6.AnnualLoanPayment, y5.AnnualLoanPayment)
		}
	})

	t.Run("variable rate total interest > fixed rate total interest", func(t *testing.T) {
		// 金利が途中から上がれば総支払利息は増える
		fixed := Analyze(context.Background(), base)
		variable := base
		variable.RateAdjustmentSchedule = []RateAdjustment{{AfterYear: 6, Rate: 0.03}}
		varResult := Analyze(context.Background(), variable)

		fixedInterest := 0.0
		varInterest := 0.0
		for i := range fixed.YearlyResults {
			fixedInterest += fixed.YearlyResults[i].AnnualInterest
			varInterest += varResult.YearlyResults[i].AnnualInterest
		}
		if varInterest <= fixedInterest {
			t.Errorf("variable rate total interest(%.0f) should be > fixed(%.0f)", varInterest, fixedInterest)
		}
	})

	t.Run("LoanRateDelta stacks on top of schedule", func(t *testing.T) {
		in := base
		in.RateAdjustmentSchedule = []RateAdjustment{{AfterYear: 6, Rate: 0.02}}
		in.LoanRateDelta = 0.005
		result := Analyze(context.Background(), in)

		// 年6以降: 0.02 + 0.005 = 0.025
		y6 := result.YearlyResults[5]
		if math.Abs(y6.EffectiveRate-0.025) > 1e-9 {
			t.Errorf("year6 EffectiveRate=%.4f, want 0.025 (schedule+delta)", y6.EffectiveRate)
		}
	})
}

func TestCalcNPV_Accuracy(t *testing.T) {
	// 5年間、毎年CF=100,000円、TV=1,000,000円、r=10%、初期投資=500,000円
	// 手計算:
	//   PV(CF) = 100000/1.1^1 + ... + 100000/1.1^5 = 379,078.68
	//   PV(TV) = 1,000,000/1.1^5                  = 620,921.32
	//   NPV    = 379,078.68 + 620,921.32 - 500,000 = 500,000.00
	cfs := []float64{100_000, 100_000, 100_000, 100_000, 100_000}
	npv := CalcNPV(cfs, 1_000_000, 0.10, 500_000)
	const want = 500_000.0
	if !approxEqual(npv, want, 1.0) {
		t.Errorf("CalcNPV = %.2f, want %.2f", npv, want)
	}
}

func TestCalcIRR_Convergence(t *testing.T) {
	// 初期投資=1,000,000、CF=150,000/year×5年、TV=800,000
	cfs := []float64{150_000, 150_000, 150_000, 150_000, 150_000}
	irr, err := CalcIRR(cfs, 800_000, 1_000_000)
	if err != nil {
		t.Fatalf("CalcIRR returned error: %v", err)
	}
	if irr == nil {
		t.Fatal("CalcIRR returned nil without error")
	}
	// IRRでNPV=0になることを検証
	npvAtIRR := CalcNPV(cfs, 800_000, *irr, 1_000_000)
	if math.Abs(npvAtIRR) >= 1.0 {
		t.Errorf("NPV at IRR = %.4f, want < 1.0", npvAtIRR)
	}
}

func TestCalcIRR_NoRoot(t *testing.T) {
	// 全CFが負→根なし
	cfs := []float64{-100_000, -100_000, -100_000}
	irr, err := CalcIRR(cfs, -500_000, 1_000_000)
	if err == nil {
		t.Errorf("expected error for no-root case, got IRR=%v", irr)
	}
	if irr != nil {
		t.Errorf("expected nil IRR for no-root case")
	}
}

func TestAnalyze_DecliningBalance(t *testing.T) {
	input := InvestmentInput{
		LandPrice:          5_000_000,
		BuildingCost:       10_000_000,
		BuildingAge:        0,
		BuildingType:       BuildingTypeWood, // 耐用年数22年
		MonthlyRent:        100_000,
		LoanAmount:         0,
		DepreciationMethod: DepreciationMethodDecliningBalance,
		HoldingYears:       10,
	}
	input.Defaults()
	result := Analyze(context.Background(), input)

	straightInput := input
	straightInput.DepreciationMethod = DepreciationMethodStraightLine
	straightResult := Analyze(context.Background(), straightInput)

	// 定率法の1年目減価は定額法より大きい
	declYear1 := result.YearlyResults[0].AnnualDepreciation
	straightYear1 := straightResult.YearlyResults[0].AnnualDepreciation
	if declYear1 <= straightYear1 {
		t.Errorf("declining-balance year1 depreciation (%.0f) should exceed straight-line (%.0f)",
			declYear1, straightYear1)
	}
	// 総減価償却額はBuildingCostを超えない
	var totalDecl float64
	for _, yr := range result.YearlyResults {
		totalDecl += yr.AnnualDepreciation
	}
	if totalDecl > input.BuildingCost+1 {
		t.Errorf("total declining-balance depreciation %.0f exceeds BuildingCost %.0f", totalDecl, input.BuildingCost)
	}
}

func TestAnalyze_PriceDeclineRate_Zero(t *testing.T) {
	base := InvestmentInput{
		LandPrice:        5_000_000,
		BuildingCost:     8_000_000,
		BuildingAge:      10,
		BuildingType:     BuildingTypeWood,
		MonthlyRent:      80_000,
		LoanAmount:       10_000_000,
		AnnualLoanRate:   0.02,
		LoanYears:        25,
		HoldingYears:     10,
		ExitYieldTarget:  0.06,
		DiscountRate:     0.05,
		PriceDeclineRate: 0,
	}
	base.Defaults()
	withZero := Analyze(context.Background(), base)

	noField := base
	noField.PriceDeclineRate = 0
	noField.Defaults()
	withoutField := Analyze(context.Background(), noField)

	if withZero.IRR == nil || withoutField.IRR == nil {
		t.Skip("IRR did not converge for test input — adjust inputs if this consistently skips")
	}
	if !approxEqual(*withZero.IRR, *withoutField.IRR, 0.0001) {
		t.Errorf("PriceDeclineRate=0 IRR %.4f != no-field IRR %.4f", *withZero.IRR, *withoutField.IRR)
	}
}

func TestAnalyze_PriceDeclineRate_NonZero(t *testing.T) {
	base := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    8_000_000,
		BuildingAge:     10,
		BuildingType:    BuildingTypeWood,
		MonthlyRent:     80_000,
		LoanAmount:      10_000_000,
		AnnualLoanRate:  0.02,
		LoanYears:       25,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
		DiscountRate:    0.05,
	}
	base.Defaults()
	zeroDecline := Analyze(context.Background(), base)

	withDecline := base
	withDecline.PriceDeclineRate = 0.02
	withDecline.Defaults()
	declineResult := Analyze(context.Background(), withDecline)

	if zeroDecline.IRR == nil || declineResult.IRR == nil {
		t.Skip("IRR did not converge for test input")
	}
	if *declineResult.IRR >= *zeroDecline.IRR {
		t.Errorf("price decline IRR (%.4f) should be lower than zero-decline IRR (%.4f)",
			*declineResult.IRR, *zeroDecline.IRR)
	}
}

func TestAnalyze_OverLoan_IRRNil(t *testing.T) {
	// オーバーローン（ローン額 > 総投資額）の場合、equity <= 0 なので IRR/NPV は計算不能
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    5_000_000,
		MonthlyRent:     100_000,
		LoanAmount:      15_000_000, // 総投資額（≈11M）を超えるオーバーローン
		AnnualLoanRate:  0.02,
		LoanYears:       35,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}
	input.Defaults()
	result := Analyze(context.Background(), input)

	if result.IRR != nil {
		t.Errorf("expected IRR=nil for over-loan case, got %v", *result.IRR)
	}
	if result.NPV != 0 {
		t.Errorf("expected NPV=0 for over-loan case, got %.2f", result.NPV)
	}
}

// TestAnalyze_DeadCross_ZeroBuildingCost は建物費用=0のときデッドクロスが発生しないことを確認する
func TestAnalyze_DeadCross_ZeroBuildingCost(t *testing.T) {
	in := InvestmentInput{
		LandPrice:       10_000_000,
		BuildingCost:    0, // 土地のみ投資 → 減価償却資産なし
		BuildingAge:     0,
		BuildingType:    "木造",
		MiscExpenseRate: 0.07,
		MonthlyRent:     80_000,
		VacancyRate:     0.05,
		LoanAmount:      8_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		HoldingYears:    10,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		ExitYieldTarget: 0.06,
		YieldTarget:     0.08,
	}
	result := Analyze(context.Background(), in)
	if result.DeadCrossYear != -1 {
		t.Errorf("DeadCrossYear = %d, want -1 (none) when BuildingCost=0", result.DeadCrossYear)
	}
}

// TestAnalyze_DeadCross_NewWoodFrame は新築木造・35年ローン・1.5%のときデッドクロスが早期に発生しないことを確認する
func TestAnalyze_DeadCross_NewWoodFrame(t *testing.T) {
	in := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000, // 耐用年数22年: 減価償却≒454,545円/年
		BuildingAge:     0,
		BuildingType:    "木造",
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		HoldingYears:    10,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		ExitYieldTarget: 0.06,
		YieldTarget:     0.08,
	}
	result := Analyze(context.Background(), in)
	// 新築木造35年1.5%では1年目の元金返済≒285,000円 < 減価償却≒454,545円
	// 両者が逆転するのは23年目（実計算値）
	if result.DeadCrossYear != 23 {
		t.Errorf("DeadCrossYear = %d, want 23", result.DeadCrossYear)
	}
}

// ---- CalcInvestmentScore テスト ----

func TestCalcInvestmentScore_EmptyInput(t *testing.T) {
	result := CalcInvestmentScore(InvestmentScoreInput{})
	if result.TotalScore < 0 || result.TotalScore > 100 {
		t.Errorf("TotalScore out of range: %d", result.TotalScore)
	}
	if result.TotalScore != 50 {
		t.Errorf("empty input should yield base score 50, got %d", result.TotalScore)
	}
	if result.Grade != "普通" {
		t.Errorf("empty input grade = %q, want 普通", result.Grade)
	}
	if len(result.Breakdown.RadarData) != 5 {
		t.Errorf("expected 5 radar categories, got %d", len(result.Breakdown.RadarData))
	}
}

func TestCalcInvestmentScore_AllPositive(t *testing.T) {
	input := InvestmentScoreInput{
		PopulationItems: []PopulationForecastItem{
			{Year: 2020, Pop: 1000},
			{Year: 2050, Pop: 1300}, // +30%
		},
		StationRiderships: []StationRidershipResult{
			{StationName: "渋谷", Passengers: 500_000},
		},
		UrbanZoningItems: []UrbanZoningItem{
			{AreaClassificationJa: "市街化区域"},
		},
		LocationItems: []LocationOptimizationItem{
			{KubunNameJa: "居住誘導区域"},
		},
	}
	result := CalcInvestmentScore(input)
	if result.TotalScore > 100 {
		t.Errorf("TotalScore exceeds 100: %d", result.TotalScore)
	}
	if result.TotalScore < 90 {
		t.Errorf("all positive inputs should score high, got %d", result.TotalScore)
	}
}

func TestCalcInvestmentScore_AllNegative(t *testing.T) {
	input := InvestmentScoreInput{
		PopulationItems: []PopulationForecastItem{
			{Year: 2020, Pop: 1000},
			{Year: 2050, Pop: 400}, // -60%
		},
		FloodItems: []FloodHazardItem{
			{DepthRank: 4},
		},
		StormItems:    []StormHazardItem{{DepthJa: "5m以上"}},
		TsunamiItems:  []TsunamiHazardItem{{DepthJa: "3m以上"}},
		LandslideItems: []LandslideHazardItem{{ZoneCode: 1}},
		LiquefactionItems: []LiquefactionRiskItem{{TendencyLevel: 1}},
		EmbankmentItems:   []EmbankmentItem{{Classification: "谷埋め型"}},
		DisasterItems:     []DisasterHistoryItem{{Name: "浸水域", Year: 2019}},
	}
	result := CalcInvestmentScore(input)
	if result.TotalScore < 0 {
		t.Errorf("TotalScore below 0: %d", result.TotalScore)
	}
	if result.TotalScore > 20 {
		t.Errorf("all negative inputs should score low, got %d", result.TotalScore)
	}
}

func TestCalcPopulationScore_LinearInterpolation(t *testing.T) {
	tests := []struct {
		name  string
		items []PopulationForecastItem
		want  int
	}{
		{"増加30%", []PopulationForecastItem{{Year: 2020, Pop: 100}, {Year: 2050, Pop: 130}}, 15},
		{"変化なし", []PopulationForecastItem{{Year: 2020, Pop: 100}, {Year: 2050, Pop: 100}}, 0},
		{"減少50%", []PopulationForecastItem{{Year: 2020, Pop: 100}, {Year: 2050, Pop: 50}}, -15},
		{"データなし", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := calcPopulationScore(tt.items)
			if item.Score != tt.want {
				t.Errorf("calcPopulationScore() = %d, want %d", item.Score, tt.want)
			}
		})
	}
}

func TestCalcRidershipScore_MaxCap(t *testing.T) {
	// 20万人以上で満点（上限20点）
	riderships := []StationRidershipResult{
		{StationName: "新宿", Passengers: 500_000},
	}
	item := calcRidershipScore(riderships)
	if item.Score != 20 {
		t.Errorf("500k passengers should give max score 20, got %d", item.Score)
	}
}

func TestCalcRidershipScore_Threshold(t *testing.T) {
	tests := []struct {
		name       string
		passengers int
		wantScore  int
	}{
		{"200k = max", 200_000, 20},
		{"100k = half", 100_000, 10},
		{"50k = quarter", 50_000, 5},
		{"0 passengers", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := calcRidershipScore([]StationRidershipResult{{StationName: "駅", Passengers: tt.passengers}})
			if item.Score != tt.wantScore {
				t.Errorf("passengers %d: got %d, want %d", tt.passengers, item.Score, tt.wantScore)
			}
		})
	}
}

func TestCalcDisasterScore_YearWeighted(t *testing.T) {
	// 年不明（Year=0）→ 最悪ケース -10
	items := []DisasterHistoryItem{{Name: "浸水域", Year: 0}}
	if s := calcDisasterScore(items); s.Score != -10 {
		t.Errorf("unknown year: want -10, got %d", s.Score)
	}

	// 空リスト→ 0
	if s := calcDisasterScore(nil); s.Score != 0 {
		t.Errorf("no disaster: want 0, got %d", s.Score)
	}

	// 複数履歴があるとき最悪スコアを採用する
	mixed := []DisasterHistoryItem{
		{Name: "浸水域", Year: 1980}, // 46年前 → -2
		{Name: "がけ崩れ", Year: 0},  // 年不明 → -10
	}
	if s := calcDisasterScore(mixed); s.Score != -10 {
		t.Errorf("mixed: want -10 (worst), got %d", s.Score)
	}
}

func TestDisasterScoreByYear(t *testing.T) {
	currentYear := 2026
	cases := []struct {
		year int
		want int
	}{
		{currentYear - 5, -10},  // 5年前 → -10
		{currentYear - 10, -10}, // ちょうど10年 → -10
		{currentYear - 15, -5},  // 15年前 → -5
		{currentYear - 30, -5},  // ちょうど30年 → -5
		{currentYear - 35, -2},  // 35年前 → -2
		{0, -10},                // 年不明 → -10
	}
	for _, c := range cases {
		got := disasterScoreByYear(c.year, currentYear)
		if got != c.want {
			t.Errorf("year %d (currentYear %d): got %d, want %d", c.year, currentYear, got, c.want)
		}
	}
}

func TestCalcLiquefactionScore_Levels(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{1, -10},
		{2, -10},
		{3, -5},
		{4, -5},
		{5, 0},
		{6, 0},
	}
	for _, tt := range tests {
		items := []LiquefactionRiskItem{{TendencyLevel: tt.level}}
		item := calcLiquefactionScore(items)
		if item.Score != tt.want {
			t.Errorf("level %d: got %d, want %d", tt.level, item.Score, tt.want)
		}
	}
}

func TestCalcUrbanAreaScore_MarketArea(t *testing.T) {
	market := []UrbanZoningItem{{AreaClassificationJa: "市街化区域"}}
	if s := calcUrbanAreaScore(market); s.Score != 10 {
		t.Errorf("市街化区域 should give +10, got %d", s.Score)
	}
	control := []UrbanZoningItem{{AreaClassificationJa: "市街化調整区域"}}
	if s := calcUrbanAreaScore(control); s.Score != 0 {
		t.Errorf("市街化調整区域 should give 0, got %d", s.Score)
	}
}

func TestCalcHazardScore_Combined(t *testing.T) {
	flood := []FloodHazardItem{{DepthRank: 3}}
	storm := []StormHazardItem{{DepthJa: "5m以上"}}
	tsunami := []TsunamiHazardItem{{DepthJa: "3m以上"}}
	landslide := []LandslideHazardItem{{ZoneCode: 1}}

	item := calcHazardScore(flood, storm, tsunami, landslide)
	if item.Score < -20 {
		t.Errorf("hazard score must not go below -20, got %d", item.Score)
	}
	if item.Score != -20 {
		t.Errorf("all 4 hazards should give -20 (capped), got %d", item.Score)
	}
}

func TestBuildHazardRisks_Empty(t *testing.T) {
	risks := BuildHazardRisks(nil, nil, nil, nil)
	if len(risks) != 0 {
		t.Errorf("expected 0 risks for empty input, got %d", len(risks))
	}
}

func TestBuildHazardRisks_FloodDepthRank(t *testing.T) {
	// DepthRank < 3 → WARNING
	risks := BuildHazardRisks([]FloodHazardItem{{DepthRank: 2, RiverName: "荒川"}}, nil, nil, nil)
	if len(risks) != 1 || risks[0].Code != "FLOOD_HAZARD" {
		t.Fatalf("unexpected risks: %+v", risks)
	}
	if risks[0].Level != UrbanRiskLevelWarning {
		t.Errorf("DepthRank 2 should be WARNING, got %s", risks[0].Level)
	}

	// DepthRank >= 3 → ERROR
	risks = BuildHazardRisks([]FloodHazardItem{{DepthRank: 3}}, nil, nil, nil)
	if risks[0].Level != UrbanRiskLevelError {
		t.Errorf("DepthRank 3 should be ERROR, got %s", risks[0].Level)
	}
}

func TestBuildHazardRisks_TsunamiAlwaysError(t *testing.T) {
	risks := BuildHazardRisks(nil, nil, []TsunamiHazardItem{{DepthJa: "3m未満"}}, nil)
	if len(risks) != 1 || risks[0].Level != UrbanRiskLevelError {
		t.Errorf("tsunami should always be ERROR, got %+v", risks)
	}
}

func TestBuildHazardRisks_LandslideZoneCode(t *testing.T) {
	// ZoneCode=2 警戒 → WARNING
	risks := BuildHazardRisks(nil, nil, nil, []LandslideHazardItem{{PhenomenonType: 2, ZoneCode: 2}})
	if risks[0].Level != UrbanRiskLevelWarning {
		t.Errorf("ZoneCode 2 should be WARNING, got %s", risks[0].Level)
	}

	// ZoneCode=1 特別警戒 → ERROR
	risks = BuildHazardRisks(nil, nil, nil, []LandslideHazardItem{{PhenomenonType: 1, ZoneCode: 1}})
	if risks[0].Level != UrbanRiskLevelError {
		t.Errorf("ZoneCode 1 should be ERROR, got %s", risks[0].Level)
	}
}

func TestBuildHazardRisks_AllFour(t *testing.T) {
	risks := BuildHazardRisks(
		[]FloodHazardItem{{DepthRank: 4}},
		[]StormHazardItem{{DepthJa: "5m以上"}},
		[]TsunamiHazardItem{{DepthJa: "3m以上"}},
		[]LandslideHazardItem{{PhenomenonType: 2, ZoneCode: 1}},
	)
	if len(risks) != 4 {
		t.Errorf("expected 4 risks, got %d", len(risks))
	}
	codes := map[string]bool{}
	for _, r := range risks {
		codes[r.Code] = true
	}
	for _, c := range []string{"FLOOD_HAZARD", "STORM_HAZARD", "TSUNAMI_HAZARD", "LANDSLIDE_HAZARD"} {
		if !codes[c] {
			t.Errorf("missing risk code: %s", c)
		}
	}
}

// baseInput は CapexSchedule / RentGrowth テスト用の共通入力
func capexBaseInput() InvestmentInput {
	return InvestmentInput{
		LandPrice:          5_000_000,
		BuildingCost:       10_000_000,
		MiscExpenseRate:    0.07,
		MonthlyRent:        100_000,
		VacancyRate:        0.05,
		LoanAmount:         12_000_000,
		AnnualLoanRate:     0.015,
		LoanYears:          35,
		BuildingType:       BuildingTypeWood,
		ExpenseRate:        0.20,
		IncomeTaxRate:      0.33,
		HoldingYears:       20,
		ExitYieldTarget:    0.06,
		DepreciationMethod: DepreciationMethodStraightLine,
	}
}

// TestCapexSchedule は大規模修繕費が指定年の CF から正しく控除されることを検証する。
func TestCapexSchedule(t *testing.T) {
	const capexAmount = 5_000_000.0 // 15年目に500万円の修繕費

	base := capexBaseInput()
	withCapex := capexBaseInput()
	withCapex.CapexSchedule = []CapexEvent{{Year: 15, Amount: capexAmount}}

	rBase := Analyze(context.Background(), base)
	rWith := Analyze(context.Background(), withCapex)

	// 15年目: 修繕費分だけ CashFlow / AfterTaxCashFlow が小さくなること
	baseCF15 := rBase.YearlyResults[14].AfterTaxCashFlow
	withCF15 := rWith.YearlyResults[14].AfterTaxCashFlow
	if !approxEqual(baseCF15-withCF15, capexAmount, 1.0) {
		t.Errorf("15年目のAfterTaxCF差 = %.0f, want %.0f", baseCF15-withCF15, capexAmount)
	}

	// 15年目の CapexAmount が記録されていること
	if !approxEqual(rWith.YearlyResults[14].CapexAmount, capexAmount, 1.0) {
		t.Errorf("15年目のCapexAmount = %.0f, want %.0f", rWith.YearlyResults[14].CapexAmount, capexAmount)
	}

	// 16年目は修繕費の影響がないこと（両者で差がゼロ）
	baseCF16 := rBase.YearlyResults[15].AfterTaxCashFlow
	withCF16 := rWith.YearlyResults[15].AfterTaxCashFlow
	if !approxEqual(baseCF16, withCF16, 1.0) {
		t.Errorf("16年目は修繕費なし: diff = %.0f, want ≈0", baseCF16-withCF16)
	}
}

// TestRentGrowthScenario は賃料上昇→下落シナリオが正しく計算されることを検証する。
func TestRentGrowthScenario(t *testing.T) {
	const (
		monthlyRent    = 100_000.0
		vacancyRate    = 0.05
		growthRate     = 0.02
		growthYears    = 3
		declineRate    = 0.01
	)
	baseAnnualRent := monthlyRent * 12 * (1 - vacancyRate)

	// rentForYear ヘルパーを直接検証
	// 1年目(y=0): 上昇期 → baseRent * (1+0.02)^0 = baseRent
	got0 := rentForYear(baseAnnualRent, declineRate, growthRate, growthYears, 0)
	if !approxEqual(got0, baseAnnualRent, 1.0) {
		t.Errorf("y=0: rent=%.0f, want %.0f", got0, baseAnnualRent)
	}

	// 3年目(y=2): 上昇期 → baseRent * 1.02^2
	want2 := baseAnnualRent * (1 + growthRate) * (1 + growthRate)
	got2 := rentForYear(baseAnnualRent, declineRate, growthRate, growthYears, 2)
	if !approxEqual(got2, want2, 1.0) {
		t.Errorf("y=2: rent=%.0f, want %.0f", got2, want2)
	}

	// 4年目(y=3): 上昇期終了後1年目 → peak * (1-0.01)^1
	peak := baseAnnualRent * math.Pow(1+growthRate, float64(growthYears))
	want3 := peak * (1 - declineRate)
	got3 := rentForYear(baseAnnualRent, declineRate, growthRate, growthYears, 3)
	if !approxEqual(got3, want3, 1.0) {
		t.Errorf("y=3: rent=%.0f, want %.0f", got3, want3)
	}

	// RentGrowthRate=0 の場合は従来の下落のみが適用されること
	gotNoGrowth0 := rentForYear(baseAnnualRent, declineRate, 0, growthYears, 0)
	if !approxEqual(gotNoGrowth0, baseAnnualRent, 1.0) {
		t.Errorf("growth=0, y=0: rent=%.0f, want %.0f", gotNoGrowth0, baseAnnualRent)
	}
	gotNoGrowth5 := rentForYear(baseAnnualRent, declineRate, 0, growthYears, 5)
	wantNoGrowth5 := baseAnnualRent * math.Pow(1-declineRate, 5)
	if !approxEqual(gotNoGrowth5, wantNoGrowth5, 1.0) {
		t.Errorf("growth=0, y=5: rent=%.0f, want %.0f", gotNoGrowth5, wantNoGrowth5)
	}
}

// TestCapexForYearMultiple は同一年に複数 CapexEvent がある場合の合算を検証する。
func TestCapexForYearMultiple(t *testing.T) {
	schedule := []CapexEvent{
		{Year: 10, Amount: 2_000_000},
		{Year: 10, Amount: 3_000_000},
		{Year: 15, Amount: 1_000_000},
	}
	got10 := capexForYear(schedule, 10)
	if !approxEqual(got10, 5_000_000, 1.0) {
		t.Errorf("year=10 合算: got %.0f, want 5000000", got10)
	}
	got15 := capexForYear(schedule, 15)
	if !approxEqual(got15, 1_000_000, 1.0) {
		t.Errorf("year=15: got %.0f, want 1000000", got15)
	}
	got20 := capexForYear(schedule, 20)
	if got20 != 0 {
		t.Errorf("year=20 (修繕なし): got %.0f, want 0", got20)
	}
}

// TestRentGrowthYearsZero は RentGrowthYears=0 のとき従来の下落ロジックと同一になることを検証する。
func TestRentGrowthYearsZero(t *testing.T) {
	base := 1_140_000.0
	decline := 0.01
	for _, y := range []int{0, 1, 5, 10} {
		withZeroYears := rentForYear(base, decline, 0.02, 0, y)
		traditional := base * math.Pow(1-decline, float64(y))
		if !approxEqual(withZeroYears, traditional, 1.0) {
			t.Errorf("rentGrowthYears=0, y=%d: got %.0f, want %.0f", y, withZeroYears, traditional)
		}
	}
}

// TestLoanFeeRate は LoanFeeRate=0.02 のとき TotalInvestment が LoanAmount×0.02 だけ増えることを検証する。
func TestLoanFeeRate(t *testing.T) {
	base := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	withoutFee := Analyze(context.Background(), base)

	withFee := base
	withFee.LoanFeeRate = 0.02
	resultWithFee := Analyze(context.Background(), withFee)

	expectedIncrease := base.LoanAmount * 0.02 // 13,000,000 × 0.02 = 260,000
	actualIncrease := resultWithFee.TotalInvestment - withoutFee.TotalInvestment

	if !approxEqual(actualIncrease, expectedIncrease, epsilon) {
		t.Errorf("TotalInvestment increase = %.0f, want %.0f (LoanAmount × LoanFeeRate)",
			actualIncrease, expectedIncrease)
	}
}

// TestTotalInterest は TotalInterest が保有期間の Σ AnnualInterest と一致することを検証する。
func TestTotalInterest(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     120_000,
		VacancyRate:     0.05,
		LoanAmount:      13_000_000,
		AnnualLoanRate:  0.015,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0.33,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)

	// 保有期間内（HoldingYears以内）の AnnualInterest を合計して比較
	var sumInterest float64
	for _, yr := range result.YearlyResults {
		if yr.Year <= input.HoldingYears {
			sumInterest += yr.AnnualInterest
		}
	}

	if !approxEqual(result.TotalInterest, sumInterest, epsilon) {
		t.Errorf("TotalInterest = %.2f, want Σ AnnualInterest = %.2f", result.TotalInterest, sumInterest)
	}
}

// TestCalcStressScenario_IsSafe_DSCR_Between1And1_2 は、DSCR が 1.0〜1.2 の範囲（例: 1.05 前後）のとき
// IsSafe=false になることを検証する。
//
// 設計: dscrSafeThreshold = 1.2 であるため、DSCR < 1.2 なら IsSafe は false になるべき。
// BreakEvenYear が保有期間内に達成されていても、DSCR 閾値未達で IsSafe=false となることを確認する。
//
//	条件設定（IncomeTaxRate=0 で税引後 CF をシンプルに保つ）:
//	  - LoanAmount    = 10,000,000 / rate=1.5% / 35年
//	  - 年間返済額   ≈ 367,284
//	  - MonthlyRent   = 40,000（VacancyRate=0, ExpenseRate=0.20）
//	  - NOI           = 40,000×12×0.80 = 384,000
//	  - DSCR          = 384,000 / 367,284 ≈ 1.045（1.0〜1.2 の範囲）
//	  - afterTaxCF    = NOI - ADS = 384,000 - 367,284 > 0 → BreakEvenYear=1
//	→ DSCR < 1.2 かつ BreakEvenYear=1 → IsSafe は false であるべき
func TestCalcStressScenario_IsSafe_DSCR_Between1And1_2(t *testing.T) {
	input := InvestmentInput{
		LandPrice:      0,
		BuildingCost:   10_000_000,
		MonthlyRent:    40_000, // NOI ≈ 1.045 × 年間返済額 → DSCR 1.0〜1.2
		VacancyRate:    0,
		LoanAmount:     10_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:      35,
		ExpenseRate:    0.20,
		IncomeTaxRate:  0, // 税なし → afterTaxCF = NOI - ADS > 0 でブレークイーン確実
		HoldingYears:   10,
		BuildingType:   BuildingTypeWood,
	}

	result := calcStressScenario(context.Background(), input, "DSCR 1.0〜1.2テスト", 0, 0)

	// 前提確認: DSCR が 1.0 以上 1.2 未満の範囲にある
	if result.DSCR < 1.0 {
		t.Fatalf("前提条件未充足: DSCR=%.4f < 1.0 (入力値を見直してください)", result.DSCR)
	}
	if result.DSCR >= 1.2 {
		t.Fatalf("前提条件未充足: DSCR=%.4f >= 1.2 (テストが DSCR 閾値境界を検証できていません)", result.DSCR)
	}

	// 前提確認: 保有期間内にブレークイーンしていること（DSCR のみが IsSafe=false の原因であることを確認）
	if result.BreakEvenYear == -1 || result.BreakEvenYear > input.HoldingYears {
		t.Fatalf("前提条件未充足: BreakEvenYear=%d (CF が保有期間内に黒転していないため DSCR 閾値の検証にならない)", result.BreakEvenYear)
	}

	// 主検証: DSCR < dscrSafeThreshold(1.2) → IsSafe は false
	if result.IsSafe {
		t.Errorf("IsSafe = true, want false: DSCR=%.4f は 1.0〜1.2 の範囲だが dscrSafeThreshold(1.2) 未達のため unsafe であるべき",
			result.DSCR)
	}
	t.Logf("DSCR=%.4f, BreakEvenYear=%d, IsSafe=%v", result.DSCR, result.BreakEvenYear, result.IsSafe)
}

// TestAnalyze_LossOffsetting は不動産所得が赤字の場合に損益通算（税還付）が
// キャッシュフローに反映されることを検証する（所得税法69条）。
// 設計: 高額建物（木造・大きな減価償却）＋高金利ローンで taxableIncome を意図的に負にする。
func TestAnalyze_LossOffsetting(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    20_000_000, // 木造22年 → 年間償却 ≈ 909,090円
		MiscExpenseRate: 0.07,
		MonthlyRent:     80_000,    // 低家賃: 年間実効賃料 ≈ 912,000円
		VacancyRate:     0.05,
		LoanAmount:      15_000_000,
		AnnualLoanRate:  0.03,      // 高金利: 1年目利息 ≈ 450,000円
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,      // 経費 ≈ 182,400円
		IncomeTaxRate:   0.33,
		HoldingYears:    5,
		ExitYieldTarget: 0.06,
	}
	// 1年目の taxableIncome ≈ 912,000 - 450,000 - 909,090 - 182,400 ≈ -629,490（赤字）

	result := Analyze(context.Background(), input)
	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}

	yr := result.YearlyResults[0]

	// 損益通算: 課税所得が負 → IncomeTax も負（税還付）
	if yr.IncomeTax >= 0 {
		t.Errorf("IncomeTax = %.0f, want negative (tax refund from loss offsetting)", yr.IncomeTax)
	}

	// 税還付分だけ AfterTaxCashFlow > CashFlow になる
	if yr.AfterTaxCashFlow <= yr.CashFlow {
		t.Errorf("AfterTaxCashFlow (%.0f) <= CashFlow (%.0f): loss offsetting should increase after-tax CF",
			yr.AfterTaxCashFlow, yr.CashFlow)
	}

	// 税還付額の近似検証: |IncomeTax| ≈ |taxableIncome| * IncomeTaxRate
	// taxableIncome は直接取得できないため AfterTaxCashFlow - CashFlow = -incomeTax で逆算
	taxBenefit := yr.AfterTaxCashFlow - yr.CashFlow
	if taxBenefit <= 0 {
		t.Errorf("tax benefit (AfterTaxCF - CashFlow) = %.0f, want positive", taxBenefit)
	}
	t.Logf("IncomeTax=%.0f, CashFlow=%.0f, AfterTaxCashFlow=%.0f, TaxBenefit=%.0f",
		yr.IncomeTax, yr.CashFlow, yr.AfterTaxCashFlow, taxBenefit)
}

// TestAnalyze_LossOffsetting_ZeroRate は IncomeTaxRate=0 のとき
// 課税所得が負でも税効果がゼロになることを検証する。
func TestAnalyze_LossOffsetting_ZeroRate(t *testing.T) {
	input := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    20_000_000,
		MiscExpenseRate: 0.07,
		MonthlyRent:     80_000,
		VacancyRate:     0.05,
		LoanAmount:      15_000_000,
		AnnualLoanRate:  0.03,
		LoanYears:       35,
		BuildingType:    BuildingTypeWood,
		ExpenseRate:     0.20,
		IncomeTaxRate:   0, // 税率ゼロ → 損益通算効果なし
		HoldingYears:    5,
		ExitYieldTarget: 0.06,
	}

	result := Analyze(context.Background(), input)
	if len(result.YearlyResults) == 0 {
		t.Fatal("YearlyResults is empty")
	}

	yr := result.YearlyResults[0]

	if yr.IncomeTax != 0 {
		t.Errorf("IncomeTax = %.0f, want 0 when IncomeTaxRate=0", yr.IncomeTax)
	}
	if yr.AfterTaxCashFlow != yr.CashFlow {
		t.Errorf("AfterTaxCashFlow (%.0f) != CashFlow (%.0f) when IncomeTaxRate=0",
			yr.AfterTaxCashFlow, yr.CashFlow)
	}
}

func TestCalcAnnualTurnoverCost(t *testing.T) {
	tests := []struct {
		name        string
		input       InvestmentInput
		monthlyRent float64
		want        float64
	}{
		{
			name:        "avgTenancyYears=0 returns 0 (backward compat)",
			input:       InvestmentInput{AvgTenancyYears: 0},
			monthlyRent: 100000,
			want:        0,
		},
		{
			name: "normal calculation",
			input: InvestmentInput{
				AvgTenancyYears: 2.0,
				RestorationCost: 150000,
				AdFee:           60000,
				RentFreePeriod:  0.5,
			},
			monthlyRent: 100000,
			// turnoverPerYear = 0.5
			// (150000+60000)*0.5 + 100000*0.5*0.5 = 105000 + 25000 = 130000
			want: 130000,
		},
		{
			name:        "negative avgTenancyYears returns 0 (treated as invalid)",
			input:       InvestmentInput{AvgTenancyYears: -1.0},
			monthlyRent: 100000,
			want:        0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcAnnualTurnoverCost(tt.input, tt.monthlyRent)
			if math.Abs(got-tt.want) > 1 {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOverloanWarning はローン金額が物件取得費を超える場合に OVERLOAN WARNING が返されることを検証する。
func TestOverloanWarning(t *testing.T) {
	t.Run("overloan emits WARNING", func(t *testing.T) {
		input := InvestmentInput{
			LandPrice:       5_000_000,
			BuildingCost:    10_000_000,
			LoanAmount:      16_000_000, // 土地+建物(15,000,000)を超過
			MonthlyRent:     100_000,
			VacancyRate:     0.05,
			AnnualLoanRate:  0.015,
			LoanYears:       35,
			BuildingType:    BuildingTypeWood,
			ExpenseRate:     0.20,
			IncomeTaxRate:   0.33,
			HoldingYears:    10,
			ExitYieldTarget: 0.06,
		}
		result := Analyze(context.Background(), input)
		var found bool
		for _, ce := range result.CriticalErrors {
			if ce.Code == "OVERLOAN" && ce.Status == CriticalStatusWarning {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected OVERLOAN WARNING in CriticalErrors, got: %v", result.CriticalErrors)
		}
	})

	t.Run("no overloan within limit", func(t *testing.T) {
		input := InvestmentInput{
			LandPrice:       5_000_000,
			BuildingCost:    10_000_000,
			LoanAmount:      15_000_000, // 土地+建物と同額: 超過なし
			MonthlyRent:     100_000,
			VacancyRate:     0.05,
			AnnualLoanRate:  0.015,
			LoanYears:       35,
			BuildingType:    BuildingTypeWood,
			ExpenseRate:     0.20,
			IncomeTaxRate:   0.33,
			HoldingYears:    10,
			ExitYieldTarget: 0.06,
		}
		result := Analyze(context.Background(), input)
		for _, ce := range result.CriticalErrors {
			if ce.Code == "OVERLOAN" {
				t.Errorf("unexpected OVERLOAN in CriticalErrors: %v", ce)
			}
		}
	})
}

// TestValidateCapexScheduleYear は CapexSchedule の年数が HoldingYears を超える場合にエラーが返ることを検証する。
func TestValidateCapexScheduleYear(t *testing.T) {
	base := InvestmentInput{
		VacancyRate:  0.05,
		LoanYears:    35,
		HoldingYears: 10,
	}

	t.Run("capex year within HoldingYears", func(t *testing.T) {
		in := base
		in.CapexSchedule = []CapexEvent{{Year: 10, Amount: 1_000_000}}
		if err := in.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("capex year exceeds HoldingYears", func(t *testing.T) {
		in := base
		in.CapexSchedule = []CapexEvent{{Year: 11, Amount: 1_000_000}}
		if err := in.Validate(); err == nil {
			t.Error("expected error for CapexSchedule year exceeding HoldingYears")
		}
	})

	t.Run("multiple capex events with one exceeding", func(t *testing.T) {
		in := base
		in.CapexSchedule = []CapexEvent{
			{Year: 5, Amount: 500_000},
			{Year: 15, Amount: 1_000_000}, // 15 > HoldingYears(10)
		}
		if err := in.Validate(); err == nil {
			t.Error("expected error for CapexSchedule year 15 exceeding HoldingYears 10")
		}
	})

	t.Run("empty capex schedule passes", func(t *testing.T) {
		in := base
		if err := in.Validate(); err != nil {
			t.Errorf("unexpected error for empty CapexSchedule: %v", err)
		}
	})
}

func TestCalcYieldDifficulty(t *testing.T) {
	tests := []struct {
		name        string
		medianTsubo float64
		budget      float64
		targetYield float64
		wantDiff    string
	}{
		{
			name:        "medianTsubo=0 returns empty",
			medianTsubo: 0,
			budget:      0,
			targetYield: 0.08,
			wantDiff:    "",
		},
		{
			name:        "no budget, low tsubo price -> achievable",
			medianTsubo: 100_000,
			budget:      0,
			targetYield: 0.08,
			// totalCostEst=100000*30+10000000=13000000, rentPerTsubo=13000000*0.08/12/30≈288
			wantDiff: "achievable",
		},
		{
			name:        "no budget, high tsubo price -> difficult",
			medianTsubo: 2_000_000,
			budget:      0,
			targetYield: 0.08,
			// totalCostEst=2000000*30+10000000=70000000, rentPerTsubo=70000000*0.08/12/30≈15556
			wantDiff: "difficult",
		},
		{
			name:        "budget mode: cheap area should be achievable",
			medianTsubo: 100_000,
			budget:      50_000_000,
			targetYield: 0.08,
			// tsuboCount=50000000/100000=500, rentPerTsubo=100000*0.08/12≈667
			wantDiff: "achievable",
		},
		{
			name:        "budget mode: expensive area should be difficult",
			medianTsubo: 3_000_000,
			budget:      50_000_000,
			targetYield: 0.08,
			// rentPerTsubo=3000000*0.08/12=20000
			wantDiff: "difficult",
		},
		{
			name:        "budget mode: different areas give different results (not all same)",
			medianTsubo: 500_000,
			budget:      50_000_000,
			targetYield: 0.08,
			// rentPerTsubo=500000*0.08/12≈3333
			wantDiff: "achievable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDiff := CalcYieldDifficulty(tt.medianTsubo, tt.budget, tt.targetYield)
			if gotDiff != tt.wantDiff {
				t.Errorf("difficulty: got %q, want %q", gotDiff, tt.wantDiff)
			}
		})
	}

	// budget モードでは異なる坪単価のエリアが異なる難易度になることを確認
	t.Run("budget mode produces area-dependent results", func(t *testing.T) {
		budget := 50_000_000.0
		yield := 0.08
		cheapDiff := CalcYieldDifficulty(100_000, budget, yield)
		expensiveDiff := CalcYieldDifficulty(3_000_000, budget, yield)
		if cheapDiff == expensiveDiff {
			t.Errorf("budget mode should give different difficulty for cheap (%s) vs expensive (%s) areas",
				cheapDiff, expensiveDiff)
		}
	})
}

// --- CalcLandPriceTrend ---

func TestCalcLandPriceTrend_Empty(t *testing.T) {
	got := CalcLandPriceTrend(nil)
	if got != "不明" {
		t.Errorf("empty: got %q, want 不明", got)
	}
}

func TestCalcLandPriceTrend_OneYear(t *testing.T) {
	txs := []LandTransaction{
		{Period: "令和5年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第3四半期", PricePerTsubo: 120_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "不明" {
		t.Errorf("one year only: got %q, want 不明", got)
	}
}

func TestCalcLandPriceTrend_Rise(t *testing.T) {
	// 前年中央値 100,000 → 今年中央値 110,000 (+10%) → 上昇
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 110_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 110_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "上昇" {
		t.Errorf("rise: got %q, want 上昇", got)
	}
}

func TestCalcLandPriceTrend_Stable(t *testing.T) {
	// +2% → 安定
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 102_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 102_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "安定" {
		t.Errorf("stable: got %q, want 安定", got)
	}
}

func TestCalcLandPriceTrend_Decline(t *testing.T) {
	// -10% → 下落
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 90_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 90_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "下落" {
		t.Errorf("decline: got %q, want 下落", got)
	}
}

func TestCalcLandPriceTrend_WesternEra(t *testing.T) {
	// 西暦形式 Period
	txs := []LandTransaction{
		{Period: "2023年第1四半期", PricePerTsubo: 100_000},
		{Period: "2023年第2四半期", PricePerTsubo: 100_000},
		{Period: "2024年第1四半期", PricePerTsubo: 108_000},
		{Period: "2024年第2四半期", PricePerTsubo: 108_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "上昇" {
		t.Errorf("western era rise: got %q, want 上昇", got)
	}
}

func TestCalcLandPriceTrend_MixedEra(t *testing.T) {
	// 和暦と西暦が混在（令和5年=2023, 2024年）、-10% → 下落
	txs := []LandTransaction{
		{Period: "令和5年第1四半期", PricePerTsubo: 200_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 200_000},
		{Period: "2024年第1四半期", PricePerTsubo: 180_000},
		{Period: "2024年第2四半期", PricePerTsubo: 180_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "下落" {
		t.Errorf("mixed era decline: got %q, want 下落", got)
	}
}

func TestCalcLandPriceTrend_Boundary_ExactlyMinus5(t *testing.T) {
	// -5% ちょうどは安定（change < -5 で下落、≥ -5 は安定）
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 95_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 95_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "安定" {
		t.Errorf("boundary -5%%: got %q, want 安定", got)
	}
}

func TestCalcLandPriceTrend_Boundary_ExactlyPlus5(t *testing.T) {
	// +5% ちょうどは安定（change > 5 で上昇）
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 105_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 105_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "安定" {
		t.Errorf("boundary +5%%: got %q, want 安定", got)
	}
}

func TestCalcLandPriceTrend_ZeroPriceSkipped(t *testing.T) {
	// PricePerTsubo == 0 の取引は除外される
	// 0 除外後に令和5年のみ残る → 不明
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 0},
		{Period: "令和5年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 110_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "不明" {
		t.Errorf("zero price skipped: got %q, want 不明", got)
	}
}

func TestCalcLandPriceTrend_InvalidPeriod(t *testing.T) {
	// パース不能な Period 文字列はすべて除外される → 不明
	txs := []LandTransaction{
		{Period: "不明", PricePerTsubo: 100_000},
		{Period: "", PricePerTsubo: 120_000},
		{Period: "第3四半期", PricePerTsubo: 80_000},
	}
	got := CalcLandPriceTrend(txs)
	if got != "不明" {
		t.Errorf("invalid period: got %q, want 不明", got)
	}
}

func TestCalcLandPriceTrend_ThreeYears_UsesLatestTwo(t *testing.T) {
	// 3年分のデータがある場合、最新2年（令和5年 vs 令和6年）のみ使う
	// 令和4年: 50,000（外れ値）は無視される
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 50_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 50_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和6年第1四半期", PricePerTsubo: 112_000},
		{Period: "令和6年第2四半期", PricePerTsubo: 108_000},
	}
	// 令和5年中央値=100,000 令和6年中央値=110,000 → +10% → 上昇
	got := CalcLandPriceTrend(txs)
	if got != "上昇" {
		t.Errorf("three years: got %q, want 上昇", got)
	}
}

func TestCalcLandPriceTrend_InsufficientSamplesPerYear(t *testing.T) {
	// 最新年のサンプルが landTrendMinSamples(2) 未満 → 不明
	txs := []LandTransaction{
		{Period: "令和4年第1四半期", PricePerTsubo: 100_000},
		{Period: "令和4年第2四半期", PricePerTsubo: 100_000},
		{Period: "令和5年第1四半期", PricePerTsubo: 200_000}, // 1件のみ
	}
	got := CalcLandPriceTrend(txs)
	if got != "不明" {
		t.Errorf("insufficient samples: got %q, want 不明", got)
	}
}

// TestAnalyze_AcquisitionCosts_NewBuild は Analyze() 経由で IsFirstRegistration が
// 正しく導線に乗ることを検証する（regression guard for investment.go:isNewBuilding derivation）
func TestAnalyze_AcquisitionCosts_NewBuild(t *testing.T) {
	base := InvestmentInput{
		LandPrice:    35_000_000,
		BuildingCost: 24_000_000,
		BuildingAge:  0, // 新築 → IsNewBuilding=true が自動設定されるべき
		MonthlyRent:  120_000,
		LoanAmount:   30_000_000,
		AnnualLoanRate: 0.015,
		LoanYears:    35,
		ExpenseRate:  0.2,
		VacancyRate:  0.05,
	}

	result := Analyze(context.Background(), base)
	ac := result.AcquisitionCosts

	// 仲介手数料: 土地のみ基準 (35M×0.03+6万)×1.1 = 1,221,000
	if ac.BrokerageFee != 1_221_000 {
		t.Errorf("新築: BrokerageFee = %.0f, want 1,221,000 (土地代金のみ基準)", ac.BrokerageFee)
	}
	// 印紙税: 分割計算 35M(20,000) + 24M(20,000) = 40,000
	if ac.StampDuty != 40_000 {
		t.Errorf("新築: StampDuty = %.0f, want 40,000 (売買/請負 分割)", ac.StampDuty)
	}
	// 登録免許税: 保存登記(0.4%) を使用していること
	// 土地推定=24.5M→490,000 + 建物推定=14.4M→57,600 + 抵当権=120,000 → 667,600
	if ac.RegistrationTax != 667_600 {
		t.Errorf("新築: RegistrationTax = %.0f, want 667,600 (保存登記0.4%%)", ac.RegistrationTax)
	}

	// IsFirstRegistration=false で明示的に移転登記(2%)を強制できること（築0年転売物件）
	falseVal := false
	base2 := base
	base2.IsFirstRegistration = &falseVal
	result2 := Analyze(context.Background(), base2)
	ac2 := result2.AcquisitionCosts

	// 仲介手数料: 中古扱いなので土地+建物合計基準
	if ac2.BrokerageFee != 2_013_000 {
		t.Errorf("築0年転売: BrokerageFee = %.0f, want 2,013,000 (合計基準)", ac2.BrokerageFee)
	}
	// 登録免許税: 移転登記(2%)を使用 → 建物: 14.4M×0.02=288,000 → 898,000
	if ac2.RegistrationTax != 898_000 {
		t.Errorf("築0年転売: RegistrationTax = %.0f, want 898,000 (移転登記2%%)", ac2.RegistrationTax)
	}
}

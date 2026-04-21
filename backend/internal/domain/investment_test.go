package domain

import (
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

	result := Analyze(input)

	// 総投資額の検証
	// 5,000,000 + 10,000,000 + (15,000,000 * 0.07) = 16,050,000
	wantTotal := 5_000_000.0 + 10_000_000.0 + 15_000_000.0*0.07
	if !approxEqual(result.TotalInvestment, wantTotal, epsilon) {
		t.Errorf("TotalInvestment = %.0f, want %.0f", result.TotalInvestment, wantTotal)
	}

	// 表面利回り: (120,000 * 12) / 16,050,000 ≈ 0.0897
	wantGross := (120_000.0 * 12) / wantTotal
	if !approxEqual(result.GrossYield, wantGross, 0.0001) {
		t.Errorf("GrossYield = %.4f, want %.4f", result.GrossYield, wantGross)
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
	r1 := Analyze(highRent)
	if !r1.IsAbove8Percent {
		t.Errorf("高賃料ケース: IsAbove8Percent = false, want true (yield=%.2f%%)", r1.GrossYield*100)
	}

	// 低賃料 → 8%未満
	lowRent := InvestmentInput{
		LandPrice:    5_000_000,
		BuildingCost: 10_000_000,
		MonthlyRent:  80_000, // 低い賃料
	}
	lowRent.Defaults()
	r2 := Analyze(lowRent)
	if r2.IsAbove8Percent {
		t.Errorf("低賃料ケース: IsAbove8Percent = true, want false (yield=%.2f%%)", r2.GrossYield*100)
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

	result := Analyze(input)

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

	result := Analyze(input)

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

	baseResult := Analyze(base)
	stressResult := Analyze(stressed)

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

	stats := CalcLandPriceStats(transactions)

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

	result := Analyze(input)

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
	result := Analyze(input)
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

	result := Analyze(input)

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

			result := Analyze(input)

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

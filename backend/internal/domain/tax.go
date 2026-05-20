package domain

import "math"

// 所得税累進ブラケット（所得税法89条、2023年以降）
// rate: 国税所得税率, deduction: 控除額（円）
var incomeTaxBrackets = []struct {
	limit     float64
	rate      float64
	deduction float64
}{
	{1_950_000, 0.05, 0},
	{3_300_000, 0.10, 97_500},
	{6_950_000, 0.20, 427_500},
	{9_000_000, 0.23, 636_000},
	{18_000_000, 0.33, 1_536_000},
	{40_000_000, 0.40, 2_796_000},
	{math.MaxFloat64, 0.45, 4_796_000},
}

// corporateEffectiveRate は小規模法人（資本金1億円未満）の概算実効税率。
// 内訳: 法人税15%/23.2% + 地方法人税10.3% + 住民税7% + 事業税3.5% の合算概算。
const corporateEffectiveRate = 0.334

// calcIncomeTax は課税所得に対する国税所得税額を返す（住民税10%・復興特別税2.1%は含まない）。
// 課税所得が0以下のときは0を返す。
func calcIncomeTax(taxableIncome float64) float64 {
	if taxableIncome <= 0 {
		return 0
	}
	for _, b := range incomeTaxBrackets {
		if taxableIncome <= b.limit {
			return taxableIncome*b.rate - b.deduction
		}
	}
	return taxableIncome*0.45 - 4_796_000
}

// calcTotalTax は課税所得に対する合算税額（所得税＋住民税10%＋復興特別税2.1%）を返す。
// 損益通算で課税所得が負になった場合は0を返す（還付は行政手続きを経るが概算では0扱い）。
func calcTotalTax(taxableIncome float64) float64 {
	if taxableIncome <= 0 {
		return 0
	}
	incomeTax := calcIncomeTax(taxableIncome)
	// 復興特別所得税: 所得税 × 2.1%
	reconstructionTax := incomeTax * 0.021
	// 住民税: 課税所得 × 10%
	residentTax := taxableIncome * 0.10
	return incomeTax + reconstructionTax + residentTax
}

// calcSalaryDeduction は給与所得控除額を返す（所得税法28条、2020年以降）。
func calcSalaryDeduction(salary float64) float64 {
	switch {
	case salary <= 1_625_000:
		return 550_000
	case salary <= 1_800_000:
		return salary*0.40 - 100_000
	case salary <= 3_600_000:
		return salary*0.30 + 80_000
	case salary <= 6_600_000:
		return salary*0.20 + 440_000
	case salary <= 8_500_000:
		return salary*0.10 + 1_100_000
	default:
		return 1_950_000
	}
}

// calcSalaryTaxableIncome は給与年収から給与所得控除・基礎控除を差し引いた課税所得を返す。
// 基礎控除は48万円（所得税法86条、2020年以降・所得2,400万円以下の場合）。
func calcSalaryTaxableIncome(salary float64) float64 {
	deduction := calcSalaryDeduction(salary)
	const basicDeduction = 480_000
	taxable := salary - deduction - basicDeduction
	if taxable < 0 {
		return 0
	}
	return taxable
}

// CalcTaxSimulation は給与年収を元に損益通算と個人/法人比較を算出する。
// input.SalaryIncome == 0 のときは nil を返す。
// exitCapitalGain は売却時の譲渡所得（exit.go の calcExit から取得）。
func CalcTaxSimulation(input InvestmentInput, yearly []YearlyResult, exitCapitalGain float64) *TaxSimulationResult {
	if input.SalaryIncome <= 0 {
		return nil
	}

	holdingYears := input.HoldingYears
	if holdingYears <= 0 {
		holdingYears = 10
	}
	if holdingYears > len(yearly) {
		holdingYears = len(yearly)
	}

	salaryTaxable := calcSalaryTaxableIncome(input.SalaryIncome)
	baselineTax := calcTotalTax(salaryTaxable)

	// --- 損益通算シミュレーション (#399) ---
	rows := make([]TaxSimRow, holdingYears)
	var totalTaxSaving float64

	for i := 0; i < holdingYears; i++ {
		yr := yearly[i]
		// 合算課税所得: 給与課税所得 + 不動産課税所得（負なら損益通算で圧縮）
		combinedTaxable := salaryTaxable + yr.TaxableIncome
		if combinedTaxable < 0 {
			combinedTaxable = 0
		}
		combinedTax := calcTotalTax(combinedTaxable)
		diff := baselineTax - combinedTax // 正=節税、負=増税
		totalTaxSaving += diff

		rows[i] = TaxSimRow{
			Year:            yr.Year,
			RETaxableIncome: yr.TaxableIncome,
			CombinedIncome:  combinedTaxable,
			CombinedTax:     combinedTax,
			BaselineTax:     baselineTax,
			TaxDifference:   diff,
		}
	}

	// --- 個人 vs 法人比較 (#392) ---
	indivAnnual := make([]float64, holdingYears)
	corpAnnual := make([]float64, holdingYears)

	for i := 0; i < holdingYears; i++ {
		reTaxable := yearly[i].TaxableIncome

		// 個人: 給与+不動産の合算から「給与のみ」を引いた差分を不動産帰属税とみなす
		combinedTaxable := salaryTaxable + reTaxable
		if combinedTaxable < 0 {
			combinedTaxable = 0
		}
		indivAnnual[i] = calcTotalTax(combinedTaxable) - baselineTax

		// 法人: 不動産課税所得に法人実効税率（負なら0、繰越控除の簡易近似）
		if reTaxable > 0 {
			corpAnnual[i] = reTaxable * corporateEffectiveRate
		}
	}

	// 売却時の譲渡税
	indivTransferTax := exitCapitalGain * longTermTransferTaxRate
	if holdingYears <= 5 {
		indivTransferTax = exitCapitalGain * shortTermTransferTaxRate
	}
	corpTransferTax := exitCapitalGain * corporateEffectiveRate

	indivCumul := makeCumulative(indivAnnual, indivTransferTax)
	corpCumul := makeCumulative(corpAnnual, corpTransferTax)

	breakevenYear := -1
	for i := 0; i < holdingYears; i++ {
		if corpCumul[i] < indivCumul[i] {
			breakevenYear = i + 1
			break
		}
	}

	indivTotal := sum(indivAnnual) + indivTransferTax
	corpTotal := sum(corpAnnual) + corpTransferTax

	return &TaxSimulationResult{
		SalaryLossCarryover: SalaryLossCarryoverResult{
			SalaryIncomeYen:   input.SalaryIncome,
			BaselineSalaryTax: baselineTax,
			YearlyRows:        rows,
			TotalTaxSaving:    totalTaxSaving,
		},
		OwnershipComparison: OwnershipComparisonResult{
			Individual: OwnershipScenario{
				Label:            "個人",
				AnnualTax:        indivAnnual,
				TransferTax:      indivTransferTax,
				TotalTaxBurden:   indivTotal,
				CumulativeBurden: indivCumul,
			},
			Corporate: OwnershipScenario{
				Label:            "法人",
				AnnualTax:        corpAnnual,
				TransferTax:      corpTransferTax,
				TotalTaxBurden:   corpTotal,
				CumulativeBurden: corpCumul,
			},
			BreakevenYear: breakevenYear,
		},
	}
}

func makeCumulative(annual []float64, transferTax float64) []float64 {
	cumul := make([]float64, len(annual))
	running := 0.0
	for i, v := range annual {
		running += v
		cumul[i] = running
	}
	// 最終年に譲渡税を加算
	if len(cumul) > 0 {
		cumul[len(cumul)-1] += transferTax
	}
	return cumul
}

func sum(vals []float64) float64 {
	total := 0.0
	for _, v := range vals {
		total += v
	}
	return total
}

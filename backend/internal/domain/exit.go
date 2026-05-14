package domain

import (
	"fmt"
	"math"
)

// 譲渡所得税率（所得税＋復興特別所得税＋住民税）
// 根拠: 租税特別措置法31条・32条、復興財源確保法33条（2037年まで）
// 注意: 租税特別措置法31条の3の10年超軽減（14.21%）は「居住用財産」の特例であり
//
//	投資用物件には適用されない。投資用は保有年数に関わらず長期=20.315%。
const (
	shortTermTransferTaxRate = 0.3963  // 短期（5年以下）: 所得税30%+復興0.63%+住民税9%
	longTermTransferTaxRate  = 0.20315 // 長期（5年超）: 所得税15%+復興0.315%+住民税5%
)

// irrNPVResult は IRR / NPV 計算結果をまとめた内部 struct。
type irrNPVResult struct {
	irr *float64
	npv float64
}

// calcExit は出口戦略（売却）の試算を行う
//
// 売却価格: NOI（純収益）/ 目標利回り（実質ベース）で収益還元法により算出
// 取得費: 土地 + 建物簿価 + 取得時諸経費（税法上の取得費）
// 売却費用: 仲介手数料の上限額を概算控除（消費税込み）
// 税率: 保有5年超で長期、10年超で軽減税率を適用
func calcExit(input InvestmentInput, yearly []YearlyResult, accumulatedDepreciation float64, miscExpenses float64) (
	salePrice, capitalGain, transferTax, netProceeds, totalEquity float64,
) {
	if len(yearly) == 0 || input.HoldingYears <= 0 || input.ExitYieldTarget <= 0 {
		return
	}

	holdIdx := input.HoldingYears - 1
	if holdIdx >= len(yearly) {
		holdIdx = len(yearly) - 1
	}

	exitYear := yearly[holdIdx]

	// 収益還元法: 売却価格 = NOI / 目標利回り（実質ベース）
	// NOI = 実効賃料収入 - 運営経費（ローン利息は含まない）
	noi := exitYear.AnnualRent - exitYear.AnnualExpenses
	salePrice = noi / input.ExitYieldTarget

	// 売却費用（仲介手数料上限額の概算・消費税込み）
	// 根拠: 宅建業法46条 上限 = 売却価格×3%+6万円（+消費税10%）
	sellExpenses := (salePrice*0.03+60_000) * 1.10

	// 建物の税務上の簿価（定額法累計控除後）
	bookValueBuilding := input.BuildingCost - accumulatedDepreciation
	if bookValueBuilding < 0 {
		bookValueBuilding = 0
	}

	// 取得費 = 土地取得費 + 建物簿価 + 取得時諸経費
	// 根拠: 所得税法38条（取得費に含まれる付随費用）
	acquisitionCost := input.LandPrice + bookValueBuilding + miscExpenses

	// 譲渡所得 = 売却価格 - 売却費用 - 取得費
	capitalGain = salePrice - sellExpenses - acquisitionCost

	if capitalGain > 0 {
		// 投資用物件の譲渡所得税: 保有5年超=長期(20.315%)、5年以下=短期(39.63%)の2段階
		// 租税特別措置法31条の3の10年超軽減(14.21%)は居住用財産の特例のため対象外
		var taxRate float64
		if input.HoldingYears > 5 {
			taxRate = longTermTransferTaxRate
		} else {
			taxRate = shortTermTransferTaxRate
		}
		transferTax = capitalGain * taxRate
	}

	netProceeds = salePrice - sellExpenses - transferTax - exitYear.RemainingLoanBalance
	totalEquity = netProceeds + exitYear.CumulativeCashFlow
	return
}

// calcTerminalValueWithDecline は価格下落率を考慮した売却時ターミナルバリューを計算する
// 注意: 売却費用・譲渡税の計算ロジックは calcExit と重複している。
//
//	税率・仲介手数料率を変更する際は両方を更新すること。
func calcTerminalValueWithDecline(
	input InvestmentInput,
	yearly []YearlyResult,
	adjustedSalePrice float64,
	accumulatedDepreciation float64,
	miscExpenses float64,
) float64 {
	holdIdx := input.HoldingYears - 1
	if holdIdx >= len(yearly) {
		holdIdx = len(yearly) - 1
	}
	exitYear := yearly[holdIdx]
	sellExpenses := (adjustedSalePrice*0.03+60_000) * 1.10
	bookValueBuilding := math.Max(input.BuildingCost-accumulatedDepreciation, 0)
	acquisitionCost := input.LandPrice + bookValueBuilding + miscExpenses
	capGain := adjustedSalePrice - sellExpenses - acquisitionCost
	var transferTax float64
	if capGain > 0 {
		var taxRate float64
		if input.HoldingYears > 5 {
			taxRate = longTermTransferTaxRate
		} else {
			taxRate = shortTermTransferTaxRate
		}
		transferTax = capGain * taxRate
	}
	return adjustedSalePrice - sellExpenses - transferTax - exitYear.RemainingLoanBalance
}

// calcIRRNPV は IRR / NPV を計算して返す。
// equity がゼロ以下（オーバーローン）または HoldingYears=0 の場合は計算が成立しないためゼロ値を返す。
func calcIRRNPV(
	input InvestmentInput,
	yearlyResults []YearlyResult,
	equity float64,
	exitNet float64,
	exitSalePrice float64,
	accumulatedDepreciation float64,
	miscExpenses float64,
) irrNPVResult {
	if equity <= 0 || input.HoldingYears <= 0 {
		return irrNPVResult{}
	}
	irrCFs := make([]float64, input.HoldingYears)
	for i := 0; i < input.HoldingYears && i < len(yearlyResults); i++ {
		irrCFs[i] = yearlyResults[i].AfterTaxCashFlow
	}
	irrTerminalValue := exitNet
	if input.PriceDeclineRate > 0 {
		decayFactor := math.Pow(1-input.PriceDeclineRate, float64(input.HoldingYears))
		adjustedSalePrice := exitSalePrice * decayFactor
		irrTerminalValue = calcTerminalValueWithDecline(input, yearlyResults, adjustedSalePrice, accumulatedDepreciation, miscExpenses)
	}
	npv := CalcNPV(irrCFs, irrTerminalValue, input.DiscountRate, equity)
	irr, _ := CalcIRR(irrCFs, irrTerminalValue, equity)
	return irrNPVResult{irr: irr, npv: npv}
}

// calcMultiExitComparison は複数保有年数（デフォルト: 5/10/15/20年）の出口比較テーブルを生成する。
// ExitYears が空なら [5, 10, 15, 20] を使用し、データが存在しない年はスキップする。
func calcMultiExitComparison(input InvestmentInput, yearly []YearlyResult, accumulatedDepreciation float64, miscExpenses float64) []MultiExitRow {
	years := input.ExitYears
	if len(years) == 0 {
		years = []int{5, 10, 15, 20}
	}

	equity := input.LandPrice + input.BuildingCost + miscExpenses - input.LoanAmount

	var rows []MultiExitRow
	for _, yr := range years {
		if yr <= 0 || yr > len(yearly) {
			continue
		}
		idx := yr - 1
		exitYear := yearly[idx]

		// 累積CF（税引後）
		cumCF := exitYear.CumulativeCashFlow

		// ExitYieldTarget が 0 以下の場合は売却価格が未定義になるためスキップ
		if input.ExitYieldTarget <= 0 {
			continue
		}

		// 売却価格: NOI / exitYieldTarget（その年のNOIを使用）
		noi := exitYear.AnnualRent - exitYear.AnnualExpenses
		salePrice := noi / input.ExitYieldTarget

		// 売却費用（仲介手数料上限・消費税込み）
		sellExpenses := (salePrice*0.03 + 60_000) * 1.10

		// 建物の税務上の簿価: yearly[0..yr-1] の AnnualDepreciation を積算
		yearDepreciation := 0.0
		for i := 0; i < yr; i++ {
			yearDepreciation += yearly[i].AnnualDepreciation
		}
		bookValueBuilding := input.BuildingCost - yearDepreciation
		if bookValueBuilding < 0 {
			bookValueBuilding = 0
		}

		// 取得費 = 土地 + 建物簿価 + 諸経費
		acquisitionCost := input.LandPrice + bookValueBuilding + miscExpenses

		// 譲渡所得 = 売却価格 - 売却費用 - 取得費
		capitalGain := salePrice - sellExpenses - acquisitionCost

		// 譲渡税率: 5年以下=短期(39.63%)、5年超=長期(20.315%)
		var taxRate float64
		isShortTerm := yr <= 5
		if isShortTerm {
			taxRate = shortTermTransferTaxRate
		} else {
			taxRate = longTermTransferTaxRate
		}

		transferTax := 0.0
		if capitalGain > 0 {
			transferTax = capitalGain * taxRate
		}

		remainingLoan := exitYear.RemainingLoanBalance

		// 出口エクイティ = 売却価格 - 売却費用 - 残債 - 譲渡税 + 累積税引後CF
		exitEquity := salePrice - sellExpenses - remainingLoan - transferTax + cumCF

		// IRR 計算
		irrCFs := make([]float64, yr)
		for i := 0; i < yr; i++ {
			irrCFs[i] = yearly[i].AfterTaxCashFlow
		}
		netProceeds := salePrice - sellExpenses - transferTax - remainingLoan
		irr, _ := CalcIRR(irrCFs, netProceeds, equity)

		rows = append(rows, MultiExitRow{
			Year:            yr,
			SalePrice:       salePrice,
			TransferTaxRate: taxRate,
			TransferTax:     transferTax,
			RemainingLoan:   remainingLoan,
			CumulativeCF:    cumCF,
			ExitEquity:      exitEquity,
			IRR:             irr,
			IsShortTermWarn: isShortTerm,
		})
	}
	return rows
}

// CalcNPV は将来キャッシュフロー・ターミナルバリュー・割引率・初期投資から NPV を計算する
func CalcNPV(cfs []float64, terminalValue, discountRate, initialInvestment float64) float64 {
	pv := 0.0
	for t, cf := range cfs {
		pv += cf / math.Pow(1+discountRate, float64(t+1))
	}
	n := len(cfs)
	if n > 0 {
		pv += terminalValue / math.Pow(1+discountRate, float64(n))
	}
	return pv - initialInvestment
}

// CalcIRR は二分法で IRR を求める。収束しない場合は nil と error を返す。
func CalcIRR(cfs []float64, terminalValue, initialInvestment float64) (*float64, error) {
	const (
		lo      = -0.50
		hi      = 2.00
		maxIter = 200
		tol     = 1.0
	)
	npvLo := CalcNPV(cfs, terminalValue, lo, initialInvestment)
	npvHi := CalcNPV(cfs, terminalValue, hi, initialInvestment)
	if npvLo*npvHi > 0 {
		return nil, fmt.Errorf("IRR: no root in [%.0f%%, %.0f%%]", lo*100, hi*100)
	}
	low, high := lo, hi
	for i := 0; i < maxIter; i++ {
		mid := (low + high) / 2
		npvMid := CalcNPV(cfs, terminalValue, mid, initialInvestment)
		if math.Abs(npvMid) < tol {
			v := mid
			return &v, nil
		}
		if npvMid*npvLo < 0 {
			high = mid
		} else {
			low = mid
			npvLo = npvMid
		}
	}
	return nil, fmt.Errorf("IRR: did not converge after %d iterations", maxIter)
}

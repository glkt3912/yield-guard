package domain

import (
	"context"
	"math"
	"testing"
)

// TestCalcSellingExpenses は仲介手数料上限（消費税込み）の概算式を検証する。
func TestCalcSellingExpenses(t *testing.T) {
	// 売却価格 30,000,000 円: (30,000,000×0.03 + 60,000) × 1.10 = 960,000 × 1.10 = 1,056,000
	got := calcSellingExpenses(30_000_000)
	want := 1_056_000.0
	if !approxEqual(got, want, epsilon) {
		t.Errorf("calcSellingExpenses(30,000,000) = %.2f, want %.2f", got, want)
	}

	// 売却価格 0 円: (0 + 60,000) × 1.10 = 66,000
	if got := calcSellingExpenses(0); !approxEqual(got, 66_000, epsilon) {
		t.Errorf("calcSellingExpenses(0) = %.2f, want 66000", got)
	}
}

// TestTransferTaxRateForHolding は保有年数による短期/長期の税率切替（境界は5年）を検証する。
func TestTransferTaxRateForHolding(t *testing.T) {
	tests := []struct {
		holdingYears int
		want         float64
	}{
		{0, shortTermTransferTaxRate},
		{5, shortTermTransferTaxRate}, // 5年“以下”は短期（境界は短期側）
		{6, longTermTransferTaxRate},  // 5年“超”で長期
		{20, longTermTransferTaxRate},
	}
	for _, tt := range tests {
		if got := transferTaxRateForHolding(tt.holdingYears); got != tt.want {
			t.Errorf("transferTaxRateForHolding(%d) = %v, want %v", tt.holdingYears, got, tt.want)
		}
	}
}

// TestCalcTransferTax は譲渡所得・保有年数からの譲渡税額と、譲渡損益0以下の非課税を検証する。
func TestCalcTransferTax(t *testing.T) {
	tests := []struct {
		name         string
		capitalGain  float64
		holdingYears int
		want         float64
	}{
		{"譲渡益・長期", 10_000_000, 6, 10_000_000 * longTermTransferTaxRate},
		{"譲渡益・短期", 10_000_000, 5, 10_000_000 * shortTermTransferTaxRate},
		{"譲渡損は非課税", -5_000_000, 10, 0},
		{"譲渡益ゼロは非課税", 0, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcTransferTax(tt.capitalGain, tt.holdingYears); !approxEqual(got, tt.want, epsilon) {
				t.Errorf("calcTransferTax(%.0f, %d) = %.2f, want %.2f", tt.capitalGain, tt.holdingYears, got, tt.want)
			}
		})
	}
}

// TestCalcAcquisitionCostForTax は税法上の取得費（建物簿価＋諸経費、融資諸費用除外）と
// 減価償却累計が建物価格を超えた場合の簿価ゼロ下限を検証する。
func TestCalcAcquisitionCostForTax(t *testing.T) {
	in := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    10_000_000,
		MiscExpenseRate: 0.07,
		LoanAmount:      13_000_000,
		LoanFeeRate:     0.02, // 取得費には算入されないことを確認する
	}
	misc := (in.LandPrice + in.BuildingCost) * in.MiscExpenseRate // 1,050,000

	t.Run("償却途中（簿価あり）", func(t *testing.T) {
		accDep := 4_000_000.0
		want := in.LandPrice + (in.BuildingCost - accDep) + misc // 5,000,000 + 6,000,000 + 1,050,000
		if got := calcAcquisitionCostForTax(in, accDep); !approxEqual(got, want, epsilon) {
			t.Errorf("got %.0f, want %.0f", got, want)
		}
	})

	t.Run("償却累計が建物価格超過→簿価ゼロ下限", func(t *testing.T) {
		accDep := 12_000_000.0 // BuildingCost(10M) を超える
		want := in.LandPrice + 0 + misc
		if got := calcAcquisitionCostForTax(in, accDep); !approxEqual(got, want, epsilon) {
			t.Errorf("簿価ゼロ下限が効いていない: got %.0f, want %.0f", got, want)
		}
	})
}

// TestDecayedSalePrice は価格下落率の複利適用とゼロ時のパススルーを検証する。
func TestDecayedSalePrice(t *testing.T) {
	// 下落率 0 以下は減衰なし
	if got := decayedSalePrice(30_000_000, 0, 10); !approxEqual(got, 30_000_000, epsilon) {
		t.Errorf("rate=0: got %.2f, want 30000000", got)
	}
	// 2% × 10年: 30,000,000 × 0.98^10
	want := 30_000_000 * math.Pow(0.98, 10)
	if got := decayedSalePrice(30_000_000, 0.02, 10); !approxEqual(got, want, epsilon) {
		t.Errorf("rate=2%%/10y: got %.2f, want %.2f", got, want)
	}
}

// TestAnalyze_PriceDecline_HeadlineReflected は #774 の修正を検証する。
// PriceDeclineRate が出口ヘッドライン（売却価格・手残り・出口エクイティ）に反映され、
// かつ IRR の terminal value と二重適用されていないことを確認する。
func TestAnalyze_PriceDecline_HeadlineReflected(t *testing.T) {
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
	zero := Analyze(context.Background(), base)

	declined := base
	declined.PriceDeclineRate = 0.02
	declined.Defaults()
	got := Analyze(context.Background(), declined)

	// NOI は PriceDeclineRate に依存しないため、売却価格は zero × 0.98^10 に一致するはず
	decay := math.Pow(0.98, float64(base.HoldingYears))
	wantSalePrice := zero.ExitSalePrice * decay
	if !approxEqual(got.ExitSalePrice, wantSalePrice, 1) {
		t.Errorf("ExitSalePrice = %.0f, want %.0f (= %.0f × 0.98^10)。下落が反映されていない/二重適用の疑い",
			got.ExitSalePrice, wantSalePrice, zero.ExitSalePrice)
	}

	// 手残り・出口エクイティも下落分だけ減少する
	if !(got.ExitNetProceeds < zero.ExitNetProceeds) {
		t.Errorf("ExitNetProceeds が減少していない: zero=%.0f declined=%.0f", zero.ExitNetProceeds, got.ExitNetProceeds)
	}
	if !(got.ExitTotalEquity < zero.ExitTotalEquity) {
		t.Errorf("ExitTotalEquity が減少していない: zero=%.0f declined=%.0f", zero.ExitTotalEquity, got.ExitTotalEquity)
	}
}

// TestMultiExit_TransferTaxBoundary は短期/長期の境界（5年=短期 / 6年=長期）が
// 出口比較テーブルの税率・短期警告フラグに反映されることを検証する（#776）。
// 本ツールは保有年数による簡略判定のため、5年=短期・6年=長期となる。
func TestMultiExit_TransferTaxBoundary(t *testing.T) {
	in := InvestmentInput{
		LandPrice:       5_000_000,
		BuildingCost:    8_000_000,
		BuildingAge:     10,
		BuildingType:    BuildingTypeWood,
		MonthlyRent:     90_000,
		LoanAmount:      10_000_000,
		AnnualLoanRate:  0.02,
		LoanYears:       25,
		HoldingYears:    10,
		ExitYieldTarget: 0.06,
		ExitYears:       []int{5, 6},
	}
	in.Defaults()
	res := Analyze(context.Background(), in)

	rowByYear := map[int]MultiExitRow{}
	for _, r := range res.MultiExitComparison {
		rowByYear[r.Year] = r
	}

	r5, ok5 := rowByYear[5]
	r6, ok6 := rowByYear[6]
	if !ok5 || !ok6 {
		t.Fatalf("expected rows for year 5 and 6, got years %v", func() []int {
			var ys []int
			for _, r := range res.MultiExitComparison {
				ys = append(ys, r.Year)
			}
			return ys
		}())
	}

	if !r5.IsShortTermWarn {
		t.Errorf("5年目は短期判定（IsShortTermWarn=true）であるべき")
	}
	if !approxEqual(r5.TransferTaxRate, shortTermTransferTaxRate, epsilon) {
		t.Errorf("5年目税率 = %.5f, want 短期 %.5f", r5.TransferTaxRate, shortTermTransferTaxRate)
	}
	if r6.IsShortTermWarn {
		t.Errorf("6年目は長期判定（IsShortTermWarn=false）であるべき")
	}
	if !approxEqual(r6.TransferTaxRate, longTermTransferTaxRate, epsilon) {
		t.Errorf("6年目税率 = %.5f, want 長期 %.5f", r6.TransferTaxRate, longTermTransferTaxRate)
	}
}

package domain

import (
	"math"
	"math/rand"
	"sort"
)

// MonteCarloInput はモンテカルロシミュレーションの入力値
type MonteCarloInput struct {
	Base             InvestmentInput `json:"base"`
	Simulations      int             `json:"simulations"`      // 試行回数 (デフォルト 1000、最大 10000)
	VacancyRateSigma float64         `json:"vacancyRateSigma"` // 空室率の標準偏差 (デフォルト 0.05)
	LoanRateSigma    float64         `json:"loanRateSigma"`    // 金利の標準偏差 (デフォルト 0.005)
}

// MonteCarloResult はモンテカルロシミュレーションの結果
type MonteCarloResult struct {
	SimulationCount   int            `json:"simulationCount"`
	IRRPercentiles    Percentiles    `json:"irrPercentiles"`
	EquityPercentiles Percentiles    `json:"equityPercentiles"`
	DeadCrossRate     float64        `json:"deadCrossRate"` // デッドクロス発生率 (0〜1)
	IRRHistogram      []HistogramBin `json:"irrHistogram"`
	EquityHistogram   []HistogramBin `json:"equityHistogram"`
	SuccessRate       float64        `json:"successRate"` // IRR > 0 の割合
}

// Percentiles はパーセンタイル統計
type Percentiles struct {
	P10 float64 `json:"p10"`
	P25 float64 `json:"p25"`
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P90 float64 `json:"p90"`
}

// HistogramBin はヒストグラムの1区間
type HistogramBin struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

const (
	defaultSimulations      = 1000
	maxSimulations          = 10000
	defaultVacancyRateSigma = 0.05
	defaultLoanRateSigma    = 0.005
	histogramBins           = 20
)

// MonteCarloSimulate は正規分布による摂動を N 回繰り返し、IRR・最終純資産の分布を返す
func MonteCarloSimulate(input MonteCarloInput) MonteCarloResult {
	if input.Simulations <= 0 {
		input.Simulations = defaultSimulations
	}
	if input.Simulations > maxSimulations {
		input.Simulations = maxSimulations
	}
	if input.VacancyRateSigma <= 0 {
		input.VacancyRateSigma = defaultVacancyRateSigma
	}
	if input.LoanRateSigma <= 0 {
		input.LoanRateSigma = defaultLoanRateSigma
	}

	// 同一入力に対して決定的な結果を保証するため固定シードを使用。
	// 毎回異なる分布が必要な場合は rand.NewSource(time.Now().UnixNano()) に変更すること。
	rng := rand.New(rand.NewSource(42)) //nolint:gosec

	irrs := make([]float64, 0, input.Simulations)
	equities := make([]float64, 0, input.Simulations)
	deadCrossCount := 0

	for range input.Simulations {
		sim := input.Base
		sim.VacancyRate = clamp(sim.VacancyRate+normalSample(rng, 0, input.VacancyRateSigma), 0, 0.99)
		sim.AnnualLoanRate = clamp(sim.AnnualLoanRate+normalSample(rng, 0, input.LoanRateSigma), 0, 0.30)
		// ストレスオフセットはベース値に統合済みなのでリセット
		sim.VacancyRateDelta = 0
		sim.LoanRateDelta = 0

		result := Analyze(sim)

		irr := calcIRR(result, sim)
		if !math.IsNaN(irr) {
			irrs = append(irrs, irr)
		}
		equities = append(equities, result.ExitTotalEquity)

		if result.DeadCrossYear > 0 {
			deadCrossCount++
		}
	}

	sort.Float64s(irrs)
	sort.Float64s(equities)

	var irrPct Percentiles
	if len(irrs) > 0 {
		irrPct = buildPercentiles(irrs)
	}

	successCount := 0
	for _, v := range irrs {
		if v > 0 {
			successCount++
		}
	}
	successRate := 0.0
	if len(irrs) > 0 {
		successRate = float64(successCount) / float64(len(irrs))
	}

	return MonteCarloResult{
		SimulationCount:   input.Simulations,
		IRRPercentiles:    irrPct,
		EquityPercentiles: buildPercentiles(equities),
		DeadCrossRate:     float64(deadCrossCount) / float64(input.Simulations),
		IRRHistogram:      buildHistogram(irrs, histogramBins),
		EquityHistogram:   buildHistogram(equities, histogramBins),
		SuccessRate:       successRate,
	}
}

// calcIRR はキャッシュフロー系列から内部収益率を二分法で求める。
// 収束しない場合は math.NaN() を返す。
func calcIRR(result InvestmentResult, input InvestmentInput) float64 {
	holdN := input.HoldingYears
	if holdN <= 0 {
		holdN = 10
	}
	if holdN > len(result.YearlyResults) {
		holdN = len(result.YearlyResults)
	}

	// CF系列: year0 = -TotalInvestment, year1..holdN = AfterTaxCashFlow
	// 最終年に売却手取りを加算
	cfs := make([]float64, holdN+1)
	cfs[0] = -result.TotalInvestment
	for i := 1; i <= holdN; i++ {
		yr := result.YearlyResults[i-1]
		cfs[i] = yr.AfterTaxCashFlow
	}
	cfs[holdN] += result.ExitNetProceeds

	npv := func(r float64) float64 {
		sum := 0.0
		for t, cf := range cfs {
			sum += cf / math.Pow(1+r, float64(t))
		}
		return sum
	}

	// 二分法: [-99%, 1000%] の範囲で探索
	lo, hi := -0.99, 10.0
	if npv(lo)*npv(hi) > 0 {
		return math.NaN() // 解が存在しない
	}
	for range 100 {
		mid := (lo + hi) / 2
		if npv(mid) > 0 {
			lo = mid
		} else {
			hi = mid
		}
		if hi-lo < 1e-8 {
			break
		}
	}
	return (lo + hi) / 2
}

// normalSample は Box-Muller 変換で正規分布 N(mean, sigma) からサンプリングする
func normalSample(rng *rand.Rand, mean, sigma float64) float64 {
	u1 := rng.Float64()
	u2 := rng.Float64()
	// u1 == 0 の場合の log(0) を防ぐ
	for u1 == 0 {
		u1 = rng.Float64()
	}
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + sigma*z
}

// buildPercentiles はソート済みスライスからパーセンタイルを計算する
func buildPercentiles(sorted []float64) Percentiles {
	n := len(sorted)
	if n == 0 {
		return Percentiles{}
	}
	pct := func(p float64) float64 {
		idx := p * float64(n-1)
		lo := int(idx)
		hi := lo + 1
		if hi >= n {
			return sorted[n-1]
		}
		frac := idx - float64(lo)
		return sorted[lo]*(1-frac) + sorted[hi]*frac
	}
	return Percentiles{
		P10: pct(0.10),
		P25: pct(0.25),
		P50: pct(0.50),
		P75: pct(0.75),
		P90: pct(0.90),
	}
}

// buildHistogram はソート済みスライスを bins 個の区間に分割してヒストグラムを返す
func buildHistogram(sorted []float64, bins int) []HistogramBin {
	if len(sorted) == 0 || bins <= 0 {
		return nil
	}
	minV := sorted[0]
	maxV := sorted[len(sorted)-1]
	if minV == maxV {
		return []HistogramBin{{Min: minV, Max: maxV, Count: len(sorted)}}
	}
	width := (maxV - minV) / float64(bins)
	result := make([]HistogramBin, bins)
	for i := range bins {
		result[i] = HistogramBin{
			Min: minV + float64(i)*width,
			Max: minV + float64(i+1)*width,
		}
	}
	for _, v := range sorted {
		idx := int((v - minV) / width)
		if idx >= bins {
			idx = bins - 1
		}
		result[idx].Count++
	}
	return result
}

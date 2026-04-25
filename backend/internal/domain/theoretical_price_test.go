package domain

import (
	"context"
	"math"
	"testing"
	"time"
)

func makeStats(medianTsubo float64, transactions []LandTransaction) LandPriceStats {
	return LandPriceStats{
		Count:        len(transactions),
		MedianTsubo:  medianTsubo,
		Transactions: transactions,
	}
}

func txWithAge(buildingYear int) LandTransaction {
	return LandTransaction{BuildingYear: buildingYear, PricePerTsubo: 100_000}
}

func txWithAgeAndStation(buildingYear, stationMinutes int) LandTransaction {
	return LandTransaction{BuildingYear: buildingYear, StationMinutes: stationMinutes, PricePerTsubo: 100_000}
}

func approx(t *testing.T, got, want, tol float64, label string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %.4f, want %.4f (tol %.4f)", label, got, want, tol)
	}
}

func TestEstimateTheoreticalPrice_NoData(t *testing.T) {
	_, ok := EstimateTheoreticalPrice(context.Background(), LandPriceStats{}, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 100, BuildingAge: 10,
	})
	if ok {
		t.Error("空の stats では false を返すべき")
	}
}

func TestEstimateTheoreticalPrice_NoBuildingYearData(t *testing.T) {
	stats := makeStats(100_000, []LandTransaction{
		{PricePerTsubo: 100_000}, // BuildingYear なし
	})
	_, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 100, BuildingAge: 10,
	})
	if ok {
		t.Error("BuildingYear データなしでは false を返すべき")
	}
}

func TestEstimateTheoreticalPrice_NoAgeCorrection(t *testing.T) {
	// 中央値築年と物件築年が同じ → 補正なし
	currentYear := time.Now().Year()
	medianBuildYear := currentYear - 10 // 10年前建築が中央値
	txs := make([]LandTransaction, 10)
	for i := range txs {
		txs[i] = txWithAge(medianBuildYear)
	}
	stats := makeStats(300_000, txs)

	res, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 30, BuildingAge: 10,
	})
	if !ok {
		t.Fatal("ok should be true")
	}
	approx(t, res.AgeCorrection, 0.0, 0.001, "AgeCorrection")
	approx(t, res.StationCorrection, 0.0, 0.001, "StationCorrection")

	expected := 300_000 * (30 / SqmPerTsubo)
	approx(t, res.TheoreticalPriceJPY, expected, 1.0, "TheoreticalPriceJPY")
}

func TestEstimateTheoreticalPrice_AgeCorrectionPositive(t *testing.T) {
	// 物件が中央値より新しい(築年数が少ない) → プラス補正
	currentYear := time.Now().Year()
	medianBuildYear := currentYear - 25 // 中央値築25年
	txs := make([]LandTransaction, 10)
	for i := range txs {
		txs[i] = txWithAge(medianBuildYear)
	}
	stats := makeStats(300_000, txs)

	propAge := 10
	res, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 30, BuildingAge: propAge,
	})
	if !ok {
		t.Fatal("ok should be true")
	}
	// -0.02 × (10 - 25) = +0.30
	approx(t, res.AgeCorrection, 0.30, 0.001, "AgeCorrection")
}

func TestEstimateTheoreticalPrice_AgeCorrectionClamp(t *testing.T) {
	// 築年数差が大きい → ±30%でクランプ
	currentYear := time.Now().Year()
	txs := make([]LandTransaction, 10)
	for i := range txs {
		txs[i] = txWithAge(currentYear - 55) // 中央値築55年
	}
	stats := makeStats(300_000, txs)

	// 物件は新築(築0年) → -0.02×(0-55)=+1.1 → +0.30にクランプ
	res, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 30, BuildingAge: 0,
	})
	if !ok {
		t.Fatal("ok should be true")
	}
	approx(t, res.AgeCorrection, ageCorrectionMax, 0.001, "AgeCorrection clamped to +0.30")
}

func TestEstimateTheoreticalPrice_StationCorrection(t *testing.T) {
	// 中央値駅距離10分、物件は5分 → プラス補正
	currentYear := time.Now().Year()
	txs := make([]LandTransaction, 10)
	for i := range txs {
		txs[i] = txWithAgeAndStation(currentYear-15, 10)
	}
	stats := makeStats(300_000, txs)

	res, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 30, BuildingAge: 15, StationMinutes: 5,
	})
	if !ok {
		t.Fatal("ok should be true")
	}
	if !res.HasStationData {
		t.Error("HasStationData should be true")
	}
	// -0.01 × (5 - 10) = +0.05
	approx(t, res.StationCorrection, 0.05, 0.001, "StationCorrection")
}

func TestEstimateTheoreticalPrice_NoStationInput(t *testing.T) {
	// station_minutes=0 → 駅距離補正なし
	currentYear := time.Now().Year()
	txs := make([]LandTransaction, 10)
	for i := range txs {
		txs[i] = txWithAgeAndStation(currentYear-15, 10)
	}
	stats := makeStats(300_000, txs)

	res, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: 5_000_000, LandArea: 30, BuildingAge: 15, StationMinutes: 0,
	})
	if !ok {
		t.Fatal("ok should be true")
	}
	if res.HasStationData {
		t.Error("StationMinutes=0 なので HasStationData=false のはず")
	}
	approx(t, res.StationCorrection, 0.0, 0.001, "StationCorrection should be 0")
}

func TestEstimateTheoreticalPrice_DeviationPct(t *testing.T) {
	currentYear := time.Now().Year()
	txs := make([]LandTransaction, 10)
	for i := range txs {
		txs[i] = txWithAge(currentYear - 10)
	}
	stats := makeStats(300_000, txs)

	// 補正なし: theoretical = 300000 × (30/3.30578)
	// listingPrice > theoretical → DeviationPct > 0
	theoretical := 300_000 * (30 / SqmPerTsubo)
	listing := theoretical * 1.1 // 10%割高

	res, ok := EstimateTheoreticalPrice(context.Background(), stats, TheoreticalPriceInput{
		ListingPrice: listing, LandArea: 30, BuildingAge: 10,
	})
	if !ok {
		t.Fatal("ok should be true")
	}
	approx(t, res.DeviationPct, 10.0, 0.01, "DeviationPct should be ~10%")
}

func TestMedianInt(t *testing.T) {
	tests := []struct {
		vals []int
		want int
	}{
		{nil, 0},
		{[]int{5}, 5},
		{[]int{1, 3, 5}, 3},
		{[]int{1, 2, 3, 4}, 2}, // 偶数個: (2+3)/2=2
		{[]int{5, 1, 3}, 3},    // ソートされていない
	}
	for _, tt := range tests {
		if got := medianInt(tt.vals); got != tt.want {
			t.Errorf("medianInt(%v) = %d, want %d", tt.vals, got, tt.want)
		}
	}
}

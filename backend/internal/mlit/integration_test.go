//go:build integration

package mlit

import (
	"context"
	"testing"
	"time"
)

// TestFetchLandPrices_RealAPI は実際の国交省APIへの疎通を確認する統合テスト。
// 通常の go test ./... では実行されない。実行するには:
//
//	go test -tags=integration ./internal/mlit/... -v -timeout 60s
func TestFetchLandPrices_RealAPI(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 群馬県(area=10)の直近2年分を取得
	q := LandPriceQuery{
		Area:      "10",
		Year:      2024,
		Quarter:   1,
		ToYear:    2024,
		ToQuarter: 4,
	}

	transactions, err := client.FetchLandPrices(ctx, q)
	if err != nil {
		t.Fatalf("FetchLandPrices failed: %v", err)
	}

	if len(transactions) == 0 {
		t.Fatal("expected at least 1 transaction, got 0")
	}
	t.Logf("取得件数: %d 件", len(transactions))

	// 取得データの基本的な整合性を検証
	for i, tx := range transactions {
		if tx.TradePrice <= 0 {
			t.Errorf("transactions[%d]: TradePrice should be positive, got %f", i, tx.TradePrice)
		}
		if tx.PricePerSqm <= 0 && tx.Area > 0 {
			t.Errorf("transactions[%d]: PricePerSqm should be positive when Area > 0, got %f", i, tx.PricePerSqm)
		}
		if tx.PricePerTsubo <= 0 && tx.PricePerSqm > 0 {
			t.Errorf("transactions[%d]: PricePerTsubo should be positive when PricePerSqm > 0, got %f", i, tx.PricePerTsubo)
		}
	}
}

// TestFetchPopulationForecast_RealAPI は XKT013 将来推計人口APIへの疎通を確認する。
// 渋谷付近 (z=14, x=14547, y=6451) でテスト。
func TestFetchPopulationForecast_RealAPI(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 渋谷付近タイル座標 (z=14)
	z, x, y := 14, 14547, 6451

	items, err := client.FetchPopulationForecast(ctx, z, x, y)
	if err != nil {
		t.Fatalf("FetchPopulationForecast failed: %v", err)
	}

	t.Logf("取得年数: %d 年分", len(items))
	for _, it := range items {
		t.Logf("  %d年: %.1f人", it.Year, it.Pop)
	}

	if len(items) == 0 {
		t.Skip("フィーチャが0件のタイルでした（エリア外の可能性）")
	}

	// 返却された年のセットを検証
	years := make(map[int]float64, len(items))
	for _, it := range items {
		years[it.Year] = it.Pop
	}
	for _, wantYear := range []int{2020, 2025, 2030, 2035, 2040, 2045, 2050} {
		if _, ok := years[wantYear]; !ok {
			t.Errorf("year %d が結果に含まれていません", wantYear)
		}
	}

	// 2020年の基準人口が正 (> 0) であることを確認
	if pop2020 := years[2020]; pop2020 <= 0 {
		t.Errorf("PTN_2020 should be positive, got %f", pop2020)
	}
}

// TestFetchPopulationForecast_RealAPI_Rural は地方都市（前橋市付近）での疎通テスト。
// 都市部より人口減少が大きいエリアの確認用。
func TestFetchPopulationForecast_RealAPI_Rural(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 前橋市付近タイル座標 (z=14)
	z, x, y := 14, 14479, 6412

	items, err := client.FetchPopulationForecast(ctx, z, x, y)
	if err != nil {
		t.Fatalf("FetchPopulationForecast (rural) failed: %v", err)
	}

	t.Logf("前橋市付近 - 取得年数: %d 年分", len(items))
	for _, it := range items {
		t.Logf("  %d年: %.1f人", it.Year, it.Pop)
	}

	if len(items) == 0 {
		t.Skip("フィーチャが0件のタイルでした")
	}

	// 2020→2050 変化率をログ出力
	pop2020, pop2050 := 0.0, 0.0
	for _, it := range items {
		if it.Year == 2020 {
			pop2020 = it.Pop
		}
		if it.Year == 2050 {
			pop2050 = it.Pop
		}
	}
	if pop2020 > 0 {
		rate := (pop2050 - pop2020) / pop2020 * 100
		t.Logf("30年後変化率: %+.1f%%", rate)
	}
}

// TestFetchLandAppraisals は XCT001 地価公示APIへの疎通を確認する統合テスト。
// 東京都(area=13)・住宅地(division=00)・2024年のデータを取得してフィールドを検証する。
func TestFetchLandAppraisals(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	items, err := client.FetchLandAppraisals(ctx, "13", "", 2024, "00")
	if err != nil {
		t.Fatalf("FetchLandAppraisals failed: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("expected at least 1 appraisal item, got 0")
	}
	t.Logf("取得件数: %d 件", len(items))

	for i, item := range items[:min(5, len(items))] {
		t.Logf("  [%d] 地区=%s 価格=%v円/m² 変動率=%.2f%%", i, item.District, item.PricePerSqm, item.ChangeRate*100)
		if item.PricePerSqm <= 0 {
			t.Errorf("items[%d]: PricePerSqm should be positive", i)
		}
		if item.Year != 2024 {
			t.Errorf("items[%d]: Year = %d, want 2024", i, item.Year)
		}
	}
}

// TestFetchLandAppraisals_CityFilter は市区町村コードによるフィルタリングの疎通テスト。
func TestFetchLandAppraisals_CityFilter(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 千代田区(13101)に絞り込み
	items, err := client.FetchLandAppraisals(ctx, "13", "13101", 2024, "00")
	if err != nil {
		t.Fatalf("FetchLandAppraisals (city filter) failed: %v", err)
	}

	t.Logf("千代田区の取得件数: %d 件", len(items))
	for _, item := range items {
		t.Logf("  地区=%s 価格=%v円/m²", item.District, item.PricePerSqm)
	}
}

// TestFetchLandPrices_RealAPI_WithCity は市区町村コード絞り込みの疎通テスト。
func TestFetchLandPrices_RealAPI_WithCity(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 群馬県 前橋市(10201)に絞り込み
	q := LandPriceQuery{
		Area:      "10",
		City:      "10201",
		Year:      2024,
		Quarter:   1,
		ToYear:    2024,
		ToQuarter: 4,
	}

	transactions, err := client.FetchLandPrices(ctx, q)
	if err != nil {
		t.Fatalf("FetchLandPrices (with city) failed: %v", err)
	}

	t.Logf("前橋市の取得件数: %d 件", len(transactions))

	// 市区町村指定の場合は件数が少ない可能性があるため、エラーにはしない
	// ただし、結果が返った場合は坪単価換算が正しいことを確認する
	for i, tx := range transactions {
		if tx.PricePerSqm > 0 {
			expected := tx.PricePerSqm * 3.30578
			diff := tx.PricePerTsubo - expected
			if diff < -1 || diff > 1 {
				t.Errorf("transactions[%d]: PricePerTsubo conversion incorrect: got %f, want ~%f", i, tx.PricePerTsubo, expected)
			}
		}
	}
}

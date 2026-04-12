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

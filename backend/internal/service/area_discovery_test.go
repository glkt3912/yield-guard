package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// mockAreaClient は AreaMLITClient のテスト用モック
type mockAreaClient struct {
	muniFunc  func(ctx context.Context, area string) ([]mlit.Municipality, error)
	fetchFunc func(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
}

func (m *mockAreaClient) FetchMunicipalities(ctx context.Context, area string) ([]mlit.Municipality, error) {
	if m.muniFunc != nil {
		return m.muniFunc(ctx, area)
	}
	return nil, nil
}

func (m *mockAreaClient) FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, q)
	}
	return nil, nil
}

// stubSummarizer は ai.Summarizer のテスト用スタブ。
// summary / areaSummary を返し、GenerateAreaSummary に渡された item を記録する。
type stubSummarizer struct {
	summary     string
	areaSummary string
	gotItem     domain.AreaDiscoveryItem
}

func (s *stubSummarizer) GenerateSummary(_ context.Context, _ domain.InvestmentInput, _ domain.InvestmentResult) string {
	return s.summary
}

func (s *stubSummarizer) GenerateAreaSummary(_ context.Context, item domain.AreaDiscoveryItem) string {
	s.gotItem = item
	return s.areaSummary
}

func sampleTransactions(pricePerTsubo float64) []domain.LandTransaction {
	return []domain.LandTransaction{
		{Period: "2024年第1四半期", TradePrice: 10_000_000, Area: 100, PricePerSqm: 100_000, PricePerTsubo: pricePerTsubo},
		{Period: "2024年第2四半期", TradePrice: 11_000_000, Area: 110, PricePerSqm: 100_000, PricePerTsubo: pricePerTsubo},
		{Period: "2024年第3四半期", TradePrice: 12_000_000, Area: 120, PricePerSqm: 100_000, PricePerTsubo: pricePerTsubo},
	}
}

// ---- Discover ----

func TestDiscover_MunicipalityFetchError(t *testing.T) {
	client := &mockAreaClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return nil, errors.New("upstream error")
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	_, err := svc.Discover(context.Background(), "13", 0, 0.08)
	if err == nil {
		t.Fatal("expected error on municipality fetch failure, got nil")
	}
}

func TestDiscover_Success(t *testing.T) {
	client := &mockAreaClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{
				{ID: "13101", Name: "千代田区"},
				{ID: "13102", Name: "中央区"},
			}, nil
		},
		fetchFunc: func(_ context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return sampleTransactions(330_578), nil
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	resp, err := svc.Discover(context.Background(), "13", 0, 0.08)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Prefecture != "13" {
		t.Errorf("expected prefecture=13, got %q", resp.Prefecture)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	for _, item := range resp.Items {
		if item.MunicipalityCode != "13101" && item.MunicipalityCode != "13102" {
			t.Errorf("unexpected MunicipalityCode %q", item.MunicipalityCode)
		}
		if item.MedianTsubo == 0 {
			t.Errorf("expected non-zero MedianTsubo for %s", item.MunicipalityCode)
		}
		if !item.DataSufficient {
			t.Errorf("expected DataSufficient=true for %s (3 transactions)", item.MunicipalityCode)
		}
		if item.TransactionCount != 3 {
			t.Errorf("expected TransactionCount=3, got %d", item.TransactionCount)
		}
	}
}

// 個別市区町村の取得失敗は全体を失敗させず、データ不足として返す
func TestDiscover_PartialLandPriceFailure(t *testing.T) {
	client := &mockAreaClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{
				{ID: "13101", Name: "千代田区"},
				{ID: "13102", Name: "中央区"},
			}, nil
		},
		fetchFunc: func(_ context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			if q.City == "13101" {
				return nil, errors.New("upstream error")
			}
			return sampleTransactions(100_000), nil
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	resp, err := svc.Discover(context.Background(), "13", 0, 0.08)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	var failed *domain.AreaDiscoveryItem
	for i := range resp.Items {
		if resp.Items[i].MunicipalityCode == "13101" {
			failed = &resp.Items[i]
		}
	}
	if failed == nil {
		t.Fatal("expected item for failed municipality 13101")
	}
	if failed.DataSufficient {
		t.Error("expected DataSufficient=false for failed municipality")
	}
	if failed.YieldDifficulty != "unknown" || failed.YieldDifficultyLabel != "データ不足" {
		t.Errorf("expected unknown/データ不足, got %s/%s", failed.YieldDifficulty, failed.YieldDifficultyLabel)
	}
	if failed.LandPriceTrend != "不明" {
		t.Errorf("expected LandPriceTrend=不明, got %q", failed.LandPriceTrend)
	}
}

// 達成可能 → やや困難 → 困難 → 不明の順に並ぶ
func TestDiscover_SortsByDifficulty(t *testing.T) {
	client := &mockAreaClient{
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{
				{ID: "13101", Name: "千代田区"}, // 高額 → 困難系
				{ID: "13102", Name: "中央区"},   // データなし → unknown
				{ID: "13103", Name: "港区"},    // 安価 → achievable
			}, nil
		},
		fetchFunc: func(_ context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			switch q.City {
			case "13101":
				return sampleTransactions(3_000_000), nil
			case "13103":
				return sampleTransactions(100_000), nil
			default:
				return nil, nil
			}
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	resp, err := svc.Discover(context.Background(), "13", 50_000_000, 0.08)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}
	if resp.Items[0].MunicipalityCode != "13103" {
		t.Errorf("expected achievable municipality 13103 first, got %s (%s)", resp.Items[0].MunicipalityCode, resp.Items[0].YieldDifficulty)
	}
	if resp.Items[2].MunicipalityCode != "13102" {
		t.Errorf("expected unknown municipality 13102 last, got %s (%s)", resp.Items[2].MunicipalityCode, resp.Items[2].YieldDifficulty)
	}
}

// ---- Summarize ----

func TestSummarize_NoTransactions_Fallback(t *testing.T) {
	client := &mockAreaClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return []domain.LandTransaction{}, nil
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	// summarizer が "" を返す場合は YieldDifficultyLabel にフォールバック
	if got := svc.Summarize(context.Background(), "13", "13101", 0, 0.08); got != "データ不足" {
		t.Errorf("expected fallback summary=%q, got %q", "データ不足", got)
	}
}

func TestSummarize_Success(t *testing.T) {
	client := &mockAreaClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return sampleTransactions(100_000), nil
		},
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{{ID: "13101", Name: "千代田区"}}, nil
		},
	}
	sum := &stubSummarizer{}
	svc := NewAreaDiscoveryService(client, sum)

	// PricePerTsubo=100_000 → CalcYieldDifficulty → "達成可能"
	if got := svc.Summarize(context.Background(), "13", "13101", 0, 0.08); got != "達成可能" {
		t.Errorf("expected summary=%q, got %q", "達成可能", got)
	}
	// summarizer には市区町村名解決済みの item が渡る
	if sum.gotItem.MunicipalityName != "千代田区" {
		t.Errorf("expected resolved MunicipalityName=千代田区, got %q", sum.gotItem.MunicipalityName)
	}
	if !sum.gotItem.DataSufficient {
		t.Error("expected DataSufficient=true in item passed to summarizer")
	}
}

func TestSummarize_BudgetYieldParams(t *testing.T) {
	client := &mockAreaClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return sampleTransactions(3_000_000), nil
		},
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{{ID: "13101", Name: "千代田区"}}, nil
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	// budget=50000000, yield=0.08: medianTsubo=3000000 → rentPerTsubo=20000 → "困難"
	if got := svc.Summarize(context.Background(), "13", "13101", 50_000_000, 0.08); got != "困難" {
		t.Errorf("expected summary=%q, got %q", "困難", got)
	}
}

func TestSummarize_NoData_UnknownDifficulty(t *testing.T) {
	client := &mockAreaClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return nil, nil
		},
		muniFunc: func(_ context.Context, _ string) ([]mlit.Municipality, error) {
			return []mlit.Municipality{{ID: "13101", Name: "千代田区"}}, nil
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{})

	if got := svc.Summarize(context.Background(), "13", "13101", 0, 0.08); got != "データ不足" {
		t.Errorf("expected summary=%q, got %q", "データ不足", got)
	}
}

// AI が要約を返す場合はフォールバックせずそのまま返す
func TestSummarize_UsesSummarizerOutput(t *testing.T) {
	client := &mockAreaClient{
		fetchFunc: func(_ context.Context, _ mlit.LandPriceQuery) ([]domain.LandTransaction, error) {
			return sampleTransactions(100_000), nil
		},
	}
	svc := NewAreaDiscoveryService(client, &stubSummarizer{areaSummary: "AI要約テキスト"})

	if got := svc.Summarize(context.Background(), "13", "13101", 0, 0.08); got != "AI要約テキスト" {
		t.Errorf("expected AI summary, got %q", got)
	}
}

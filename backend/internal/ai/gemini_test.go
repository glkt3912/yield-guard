package ai

import (
	"context"
	"testing"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

// mockSummaryCache is a test double for summaryCache.
type mockSummaryCache struct {
	data     map[string]string
	getCalls int
	setCalls int
}

func newMockSummaryCache() *mockSummaryCache {
	return &mockSummaryCache{data: make(map[string]string)}
}

func (m *mockSummaryCache) get(_ context.Context, key string) (string, bool) {
	m.getCalls++
	v, ok := m.data[key]
	return v, ok
}

func (m *mockSummaryCache) set(_ context.Context, key, summary string) {
	m.setCalls++
	m.data[key] = summary
}

func TestInMemorySummaryCache_GetSet(t *testing.T) {
	c := newInMemorySummaryCache()
	ctx := context.Background()

	if _, ok := c.get(ctx, "k1"); ok {
		t.Fatal("expected cache miss on empty cache")
	}

	c.set(ctx, "k1", "summary-text")

	got, ok := c.get(ctx, "k1")
	if !ok {
		t.Fatal("expected cache hit after set")
	}
	if got != "summary-text" {
		t.Fatalf("got %q, want %q", got, "summary-text")
	}
}

func TestInMemorySummaryCache_Expiry(t *testing.T) {
	c := newInMemorySummaryCache()
	ctx := context.Background()

	c.mu.Lock()
	c.entries["expired"] = cacheEntry{summary: "old", expiresAt: time.Now().Add(-time.Second)}
	c.mu.Unlock()

	if _, ok := c.get(ctx, "expired"); ok {
		t.Fatal("expected expired entry to be a cache miss")
	}
}

func TestNewSummarizer_NoAPIKey_ReturnsNoop(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	s := NewSummarizer(nil)
	if _, ok := s.(noopSummarizer); !ok {
		t.Fatalf("expected noopSummarizer when GEMINI_API_KEY is unset, got %T", s)
	}
}

func TestNewSummarizer_WithAPIKey_NilFirestore_UsesInMemory(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "dummy-key-for-test")
	s := NewSummarizer(nil)
	gs, ok := s.(*GeminiSummarizer)
	if !ok {
		t.Fatalf("expected *GeminiSummarizer, got %T", s)
	}
	if _, ok := gs.cache.(*inMemorySummaryCache); !ok {
		t.Fatalf("expected *inMemorySummaryCache fallback, got %T", gs.cache)
	}
}

func TestGeminiSummarizer_GenerateSummary_CacheHit(t *testing.T) {
	mock := newMockSummaryCache()

	in := domain.InvestmentInput{}
	res := domain.InvestmentResult{}
	key := summaryKey(in, res)
	mock.data[key] = "cached-summary"

	s := &GeminiSummarizer{cache: mock}

	got := s.GenerateSummary(context.Background(), in, res)
	if got != "cached-summary" {
		t.Fatalf("expected %q from cache, got %q", "cached-summary", got)
	}
	if mock.getCalls != 1 {
		t.Fatalf("expected 1 cache get call, got %d", mock.getCalls)
	}
	if mock.setCalls != 0 {
		t.Fatalf("expected 0 cache set calls on hit, got %d", mock.setCalls)
	}
}

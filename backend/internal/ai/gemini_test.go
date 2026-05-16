package ai

import (
	"context"
	"sync"
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

// stubSummaryCache is a test double that allows injecting a fixed summary for get,
// and counts set calls. It is safe to use with the goroutine-based set path.
type stubSummaryCache struct {
	getResult string
	getHit    bool
	setCalls  int
	mu        sync.Mutex
}

func (s *stubSummaryCache) get(_ context.Context, _ string) (string, bool) {
	return s.getResult, s.getHit
}

func (s *stubSummaryCache) set(_ context.Context, _, _ string) {
	s.mu.Lock()
	s.setCalls++
	s.mu.Unlock()
}

// fakeGeminiSummarizer embeds GeminiSummarizer but overrides call() via a hook.
// We use a callFn to avoid needing a real Gemini client in unit tests.
type callableGeminiSummarizer struct {
	GeminiSummarizer
	callFn func() (string, error)
}

func (s *callableGeminiSummarizer) GenerateSummary(ctx context.Context, input domain.InvestmentInput, result domain.InvestmentResult) string {
	key := summaryKey(input, result)
	if cached, ok := s.cache.get(ctx, key); ok {
		return cached
	}
	summary, err := s.callFn()
	if err != nil {
		return ""
	}
	go func() {
		setCtx, setCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer setCancel()
		s.cache.set(setCtx, key, summary)
	}()
	return summary
}

func TestGeminiSummarizer_GenerateSummary_CacheMiss_CallsSetOnce(t *testing.T) {
	stub := &stubSummaryCache{getHit: false}

	in := domain.InvestmentInput{}
	res := domain.InvestmentResult{}

	s := &callableGeminiSummarizer{
		GeminiSummarizer: GeminiSummarizer{cache: stub},
		callFn: func() (string, error) {
			return "generated-summary", nil
		},
	}

	got := s.GenerateSummary(context.Background(), in, res)
	if got != "generated-summary" {
		t.Fatalf("expected %q, got %q", "generated-summary", got)
	}

	// Wait for the background goroutine to complete.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stub.mu.Lock()
		calls := stub.setCalls
		stub.mu.Unlock()
		if calls >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	stub.mu.Lock()
	setCalls := stub.setCalls
	stub.mu.Unlock()
	if setCalls != 1 {
		t.Fatalf("expected set to be called once on cache miss, got %d", setCalls)
	}
}

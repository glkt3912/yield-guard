package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/generative-ai-go/genai"
	"github.com/yield-guard/backend/internal/domain"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	aiCacheTTL               = 24 * time.Hour
	aiCallTimeout            = 5 * time.Second
	defaultGeminiModel       = "gemini-2.5-flash"
	aiSummaryCacheCollection = "ai_summary_cache"
)

const systemPrompt = `あなたは日本の不動産投資専門アドバイザーです。
以下の判断基準で分析してください：
- DSCR 1.2未満: 危険域（ローン返済に余裕がない）
- DSCR 1.2〜1.5: 要注意
- DSCR 1.5以上: 安全域
- デッドクロス10年以内: 税引後CFが悪化するため短期出口戦略が必須
- 表面利回りはエリア・建物種別・築年数で判断が変わる
- 木造（耐用年数22年）は減価償却が早くデッドクロスが来やすい
- RC造・SRC造（耐用年数47年）は長期保有向き
- 軽量鉄骨・重量鉄骨は耐用年数19〜34年で木造とRC造の中間的特性
回答は3〜4文の日本語のみで返すこと。`

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Summarizer generates an AI investment summary.
type Summarizer interface {
	GenerateSummary(ctx context.Context, input domain.InvestmentInput, result domain.InvestmentResult) string
}

type noopSummarizer struct{}

func (noopSummarizer) GenerateSummary(_ context.Context, _ domain.InvestmentInput, _ domain.InvestmentResult) string {
	return ""
}

type cacheEntry struct {
	summary   string
	expiresAt time.Time
}

// summaryCache is the interface for L2 Firestore and L1 in-memory cache operations.
type summaryCache interface {
	get(ctx context.Context, key string) (string, bool)
	set(ctx context.Context, key, summary string)
}

// inMemorySummaryCache is the fallback in-memory cache used when Firestore is unavailable.
type inMemorySummaryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func newInMemorySummaryCache() *inMemorySummaryCache {
	return &inMemorySummaryCache{entries: make(map[string]cacheEntry)}
}

func (c *inMemorySummaryCache) get(_ context.Context, key string) (string, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return "", false
	}
	return entry.summary, true
}

func (c *inMemorySummaryCache) set(_ context.Context, key, summary string) {
	c.mu.Lock()
	c.entries[key] = cacheEntry{summary: summary, expiresAt: time.Now().Add(aiCacheTTL)}
	c.mu.Unlock()
}

// firestoreSummaryCache stores AI summaries in Firestore with TTL.
type firestoreSummaryCache struct {
	client *firestore.Client
}

func (c *firestoreSummaryCache) get(ctx context.Context, key string) (string, bool) {
	doc, err := c.client.Collection(aiSummaryCacheCollection).Doc(key).Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			slog.WarnContext(ctx, "Firestore ai_summary_cache get failed", "key", key, "error", err)
		}
		return "", false
	}
	expiresAt, ok := doc.Data()["expiresAt"].(time.Time)
	if !ok {
		slog.WarnContext(ctx, "ai_summary_cache: unexpected expiresAt type", "key", key)
		return "", false
	}
	if time.Now().After(expiresAt) {
		return "", false
	}
	summary, ok := doc.Data()["summary"].(string)
	if !ok {
		return "", false
	}
	return summary, true
}

func (c *firestoreSummaryCache) set(ctx context.Context, key, summary string) {
	_, err := c.client.Collection(aiSummaryCacheCollection).Doc(key).Set(ctx, map[string]any{
		"summary":   summary,
		"expiresAt": time.Now().Add(aiCacheTTL),
	})
	if err != nil {
		slog.WarnContext(ctx, "Firestore ai_summary_cache set failed", "key", key, "error", err)
	}
}

// GeminiSummarizer calls Google AI Studio Gemini to generate a Japanese investment summary.
type GeminiSummarizer struct {
	client *genai.Client
	cache  summaryCache
}

// NewSummarizer returns a Gemini-backed Summarizer, or a no-op if GEMINI_API_KEY is unset.
// If fsClient is non-nil, Firestore is used as the L2 cache; otherwise falls back to in-memory.
func NewSummarizer(fsClient *firestore.Client) Summarizer {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return noopSummarizer{}
	}
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		slog.Warn("Gemini init failed, AI summary disabled", "error", err)
		return noopSummarizer{}
	}
	var cache summaryCache
	if fsClient != nil {
		cache = &firestoreSummaryCache{client: fsClient}
	} else {
		cache = newInMemorySummaryCache()
	}
	return &GeminiSummarizer{
		client: client,
		cache:  cache,
	}
}

func (s *GeminiSummarizer) GenerateSummary(ctx context.Context, input domain.InvestmentInput, result domain.InvestmentResult) string {
	key := summaryKey(input, result)

	if cached, ok := s.cache.get(ctx, key); ok {
		return cached
	}

	tctx, cancel := context.WithTimeout(ctx, aiCallTimeout)
	defer cancel()

	summary, err := s.call(tctx, input, result)
	if err != nil {
		slog.WarnContext(ctx, "Gemini summary failed", "error", err)
		return ""
	}

	go func() {
		setCtx, setCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer setCancel()
		s.cache.set(setCtx, key, summary)
	}()

	return summary
}

func (s *GeminiSummarizer) call(ctx context.Context, input domain.InvestmentInput, r domain.InvestmentResult) (string, error) {
	model := s.client.GenerativeModel(envOrDefault("GEMINI_MODEL", defaultGeminiModel))
	model.SetMaxOutputTokens(400)
	model.SetTemperature(0.3)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	prompt := buildPrompt(input, r)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}
	part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return "", fmt.Errorf("unexpected response type from Gemini")
	}
	return string(part), nil
}

func buildPrompt(input domain.InvestmentInput, r domain.InvestmentResult) string {
	deadCross := "なし"
	if r.DeadCrossYear > 0 {
		deadCross = fmt.Sprintf("%d年目", r.DeadCrossYear)
	}

	loanMethod := "元利均等"
	if input.LoanMethod == domain.LoanMethodEqualPrincipal {
		loanMethod = "元金均等"
	}

	buildingAge := "新築"
	if input.BuildingAge > 0 {
		buildingAge = fmt.Sprintf("%d年", input.BuildingAge)
	}

	return fmt.Sprintf(
		`以下の不動産投資シミュレーション結果を分析し、強み・リスク・推奨アクションを専門家が初心者に説明するスタイルで、数値を引用しながらまとめてください。

【物件・ローン条件】
- 建物種別: %s（築%s）
- ローン金額: %.0f万円 / 年利: %.2f%% / 期間: %d年 / 返済方式: %s
- 保有予定年数: %d年
- 空室率: %.1f%%

【シミュレーション結果】
- 表面利回り: %.1f%%
- 実質利回り: %.1f%%
- デッドクロス発生: %s
- DSCR（借入金償還余裕率）: %.2f
- 保有期間最終手残り: %.0f万円
- NPV（正味現在価値）: %.0f万円`,
		input.BuildingType,
		buildingAge,
		input.LoanAmount/10000,
		input.AnnualLoanRate*100,
		input.LoanYears,
		loanMethod,
		input.HoldingYears,
		input.VacancyRate*100,
		r.GrossYield*100,
		r.NetYield*100,
		deadCross,
		r.DSCR,
		r.ExitTotalEquity/10000,
		r.NPV/10000,
	)
}

func summaryKey(input domain.InvestmentInput, r domain.InvestmentResult) string {
	gross := math.Round(r.GrossYield*10000) / 10000
	net := math.Round(r.NetYield*10000) / 10000
	equity := math.Round(r.ExitTotalEquity / 1000)
	npv := math.Round(r.NPV / 1000)
	dscr := math.Round(r.DSCR*100) / 100
	loanAmt := math.Round(input.LoanAmount / 10000)
	loanRate := math.Round(input.AnnualLoanRate*10000) / 10000
	return fmt.Sprintf("%g:%g:%d:%g:%g:%g|%s:%d:%g:%g:%d:%d:%g",
		gross, net, r.DeadCrossYear, equity, npv, dscr,
		input.BuildingType, input.BuildingAge, loanAmt, loanRate,
		input.LoanYears, input.HoldingYears,
		math.Round(input.VacancyRate*1000)/1000,
	)
}

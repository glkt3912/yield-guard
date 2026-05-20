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
投資経験のない初心者にも分かるよう、専門用語を使わず平易な日本語で回答してください。

【用語の言い換えルール（回答中で必ず使用）】
- DSCR → 返済の余裕度
- デッドクロス → 税負担急増リスク
- NPV → 将来収益の現在価値
- IRR → 実質利回り

【判断基準（参考情報）】
- 返済の余裕度が1.2未満: ローン返済に余裕がない状態
- 返済の余裕度が1.5以上: 安全域
- 税負担急増リスクが10年以内に発生: 短期での売却戦略が重要
- 木造（耐用年数22年）は税負担急増が早く来やすい
- RC造・SRC造（耐用年数47年）は長期保有向き

【回答フォーマット（厳守）】
結論1文・理由2文・アクション1文の計4文で回答すること。
必ず「月々の返済額」と「10年後の手残り」の数値に言及すること。
回答は日本語のみ。`

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

	monthlyPaymentMan := 0.0
	if len(r.YearlyResults) > 0 {
		monthlyPaymentMan = r.YearlyResults[0].AnnualLoanPayment / 12 / 10000
	}

	found10 := false
	var equity10Man float64
	for _, row := range r.MultiExitComparison {
		if row.Year == 10 {
			equity10Man = row.ExitEquity / 10000
			found10 = true
			break
		}
	}

	equity10Line := ""
	if found10 {
		equity10Line = fmt.Sprintf("\n- 10年後の手残り: %.0f万円", equity10Man)
	}

	return fmt.Sprintf(
		`以下の不動産投資シミュレーション結果を分析し、強み・リスク・推奨アクションを初心者向けに数値を引用しながらまとめてください。

【物件・ローン条件】
- 建物種別: %s（築%s）
- ローン金額: %.0f万円 / 年利: %.2f%% / 期間: %d年 / 返済方式: %s
- 保有予定年数: %d年
- 空室率: %.1f%%

【シミュレーション結果】
- 月々の返済額: %.1f万円（1年目）
- 表面利回り: %.1f%%
- 実質利回り: %.1f%%
- 税負担急増リスク発生: %s
- 返済の余裕度: %.2f%s
- 保有期間最終手残り: %.0f万円
- 将来収益の現在価値: %.0f万円`,
		input.BuildingType,
		buildingAge,
		input.LoanAmount/10000,
		input.AnnualLoanRate*100,
		input.LoanYears,
		loanMethod,
		input.HoldingYears,
		input.VacancyRate*100,
		monthlyPaymentMan,
		r.GrossYield*100,
		r.NetYield*100,
		deadCross,
		r.DSCR,
		equity10Line,
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

package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/yield-guard/backend/internal/domain"
	"google.golang.org/api/option"
)

const (
	aiCacheTTL          = 24 * time.Hour
	aiCacheEvictInterval = time.Hour
	aiCallTimeout        = 5 * time.Second
	defaultGeminiModel   = "gemini-2.5-flash"
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

// GeminiSummarizer calls Google AI Studio Gemini to generate a Japanese investment summary.
type GeminiSummarizer struct {
	client  *genai.Client
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

// NewSummarizer returns a Gemini-backed Summarizer, or a no-op if GEMINI_API_KEY is unset.
func NewSummarizer() Summarizer {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return noopSummarizer{}
	}
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		slog.Warn("Gemini init failed, AI summary disabled", "error", err)
		return noopSummarizer{}
	}
	s := &GeminiSummarizer{
		client:  client,
		entries: make(map[string]cacheEntry),
	}
	go s.evictLoop()
	return s
}

func (s *GeminiSummarizer) evictLoop() {
	ticker := time.NewTicker(aiCacheEvictInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for k, e := range s.entries {
			if now.After(e.expiresAt) {
				delete(s.entries, k)
			}
		}
		s.mu.Unlock()
	}
}

func (s *GeminiSummarizer) GenerateSummary(ctx context.Context, input domain.InvestmentInput, result domain.InvestmentResult) string {
	key := summaryKey(input, result)

	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.summary
	}

	tctx, cancel := context.WithTimeout(ctx, aiCallTimeout)
	defer cancel()

	summary, err := s.call(tctx, input, result)
	if err != nil {
		slog.WarnContext(ctx, "Gemini summary failed", "error", err)
		return ""
	}

	s.mu.Lock()
	s.entries[key] = cacheEntry{summary: summary, expiresAt: time.Now().Add(aiCacheTTL)}
	s.mu.Unlock()

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

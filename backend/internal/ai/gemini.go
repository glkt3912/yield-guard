package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/yield-guard/backend/internal/domain"
)

const (
	aiCacheTTL         = 24 * time.Hour
	aiCallTimeout      = 3 * time.Second
	defaultGeminiModel = "gemini-2.5-flash-preview-04-17"
	defaultGCPLocation = "us-central1"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Summarizer generates an AI investment summary.
type Summarizer interface {
	GenerateSummary(ctx context.Context, result domain.InvestmentResult) string
}

type noopSummarizer struct{}

func (noopSummarizer) GenerateSummary(_ context.Context, _ domain.InvestmentResult) string {
	return ""
}

type cacheEntry struct {
	summary   string
	expiresAt time.Time
}

// GeminiSummarizer calls Vertex AI Gemini to generate a Japanese investment summary.
type GeminiSummarizer struct {
	client  *genai.Client
	project string
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

// NewSummarizer returns a Gemini-backed Summarizer, or a no-op if GOOGLE_CLOUD_PROJECT is unset.
func NewSummarizer() Summarizer {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		return noopSummarizer{}
	}
	client, err := genai.NewClient(context.Background(), project, envOrDefault("VERTEX_AI_LOCATION", defaultGCPLocation))
	if err != nil {
		slog.Warn("Gemini init failed, AI summary disabled", "error", err)
		return noopSummarizer{}
	}
	return &GeminiSummarizer{
		client:  client,
		project: project,
		entries: make(map[string]cacheEntry),
	}
}

func (s *GeminiSummarizer) GenerateSummary(ctx context.Context, result domain.InvestmentResult) string {
	key := summaryKey(result)

	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.summary
	}

	tctx, cancel := context.WithTimeout(ctx, aiCallTimeout)
	defer cancel()

	summary, err := s.call(tctx, result)
	if err != nil {
		slog.WarnContext(ctx, "Gemini summary failed", "error", err)
		return ""
	}

	s.mu.Lock()
	s.entries[key] = cacheEntry{summary: summary, expiresAt: time.Now().Add(aiCacheTTL)}
	s.mu.Unlock()

	return summary
}

func (s *GeminiSummarizer) call(ctx context.Context, r domain.InvestmentResult) (string, error) {
	model := s.client.GenerativeModel(envOrDefault("GEMINI_MODEL", defaultGeminiModel))
	model.SetMaxOutputTokens(300)
	model.SetTemperature(0.3)

	prompt := buildPrompt(r)
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

func buildPrompt(r domain.InvestmentResult) string {
	deadCross := "なし"
	if r.DeadCrossYear > 0 {
		deadCross = fmt.Sprintf("%d年目", r.DeadCrossYear)
	}
	return fmt.Sprintf(
		`あなたは不動産投資の専門アドバイザーです。以下のシミュレーション結果を分析し、強み・リスク・推奨アクションを含む3〜4文の日本語で簡潔にまとめてください。専門家が初心者に説明するスタイルで、数値を引用しながら答えてください。

- 表面利回り: %.1f%%
- 実質利回り: %.1f%%
- デッドクロス発生: %s
- DSCR（借入金償還余裕率）: %.2f
- 保有期間最終手残り: %.0f万円
- NPV（正味現在価値）: %.0f万円

回答は3〜4文の日本語のみで返してください。`,
		r.GrossYield*100,
		r.NetYield*100,
		deadCross,
		r.DSCR,
		r.ExitTotalEquity/10000,
		r.NPV/10000,
	)
}

func summaryKey(r domain.InvestmentResult) string {
	// round to 2 decimal places to avoid floating-point drift
	gross := math.Round(r.GrossYield*10000) / 10000
	net := math.Round(r.NetYield*10000) / 10000
	equity := math.Round(r.ExitTotalEquity / 1000) // nearest 1000 yen
	npv := math.Round(r.NPV / 1000)
	dscr := math.Round(r.DSCR*100) / 100
	return fmt.Sprintf("%g:%g:%d:%g:%g:%g", gross, net, r.DeadCrossYear, equity, npv, dscr)
}

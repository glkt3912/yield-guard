package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/yield-guard/backend/internal/ai"
	"github.com/yield-guard/backend/internal/concurrent"
	"github.com/yield-guard/backend/internal/domain"
)

// domainTracerName は domain 計算をラップする span のトレーサ名。
// 従来 domain パッケージ内で生成していた span を service 層から発行するために使う。
const domainTracerName = "domain"

// InvestmentMLITClient は投資分析に必要な国交省APIのサブセット
type InvestmentMLITClient interface {
	FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
}

// rentDeclineHintTimeout は賃料下落率参考値の地価公示取得全体のタイムアウト
const rentDeclineHintTimeout = 30 * time.Second

// xct001Years は XCT001（地価公示）の対応年
var xct001Years = []int{2022, 2023, 2024, 2025, 2026}

// InvestmentService は投資シミュレーションと関連参考値の算出を担う
type InvestmentService interface {
	// Analyze は投資シミュレーションを実行し、コンテキストの残余時間内で AI 要約を付与する。
	// AI 要約が間に合わない場合は要約なしで結果を返す。
	Analyze(ctx context.Context, input domain.InvestmentInput) domain.InvestmentResult
	// RentDeclineHint は地価公示5年分から賃料下落率参考値を返す。
	// 一部年の取得失敗は許容し、全年失敗した場合のみエラーを返す。
	RentDeclineHint(ctx context.Context, area, municipality string) (domain.RentDeclineHint, error)
}

// InvestmentAnalysisService は InvestmentService の実装
type InvestmentAnalysisService struct {
	client     InvestmentMLITClient
	summarizer ai.Summarizer
}

func NewInvestmentAnalysisService(client InvestmentMLITClient, summarizer ai.Summarizer) InvestmentService {
	return &InvestmentAnalysisService{client: client, summarizer: summarizer}
}

func (s *InvestmentAnalysisService) Analyze(ctx context.Context, input domain.InvestmentInput) domain.InvestmentResult {
	// span は domain.Analyze の計算のみを計測する。ctx を上書きすると span 終了後に走る
	// AI 要約 goroutine が「終了済み span」を親とする ctx を受け取ってしまうため、
	// 計算用には別の spanCtx を渡し、元の ctx は汚さない。
	spanCtx, span := otel.Tracer(domainTracerName).Start(ctx, "domain.Analyze")
	span.SetAttributes(
		attribute.Float64("domain.land_price", input.LandPrice),
		attribute.Float64("domain.building_cost", input.BuildingCost),
		attribute.Float64("domain.loan_amount", input.LoanAmount),
	)
	result := domain.Analyze(spanCtx, input)
	span.End()

	// run Gemini in background; collect result within remaining context budget
	aiCh := make(chan string, 1)
	go func() {
		aiCh <- s.summarizer.GenerateSummary(ctx, input, result)
	}()
	select {
	case result.AISummary = <-aiCh:
	case <-ctx.Done():
		// request cancelled or timed out upstream
	}
	return result
}

func (s *InvestmentAnalysisService) RentDeclineHint(ctx context.Context, area, municipality string) (domain.RentDeclineHint, error) {
	ctx, cancel := context.WithTimeout(ctx, rentDeclineHintTimeout)
	defer cancel()

	// XCT001 の対応年を並列取得
	type yearItems struct {
		year  int
		items []domain.LandAppraisalItem
	}
	chs := make([]<-chan concurrent.Result[yearItems], 0, len(xct001Years))
	for _, y := range xct001Years {
		chs = append(chs, concurrent.FanOut(func() (yearItems, error) {
			items, err := s.client.FetchLandAppraisals(ctx, area, municipality, y, "00")
			if err != nil {
				return yearItems{year: y}, err
			}
			return yearItems{year: y, items: items}, nil
		}))
	}

	itemsByYear := make(map[int][]domain.LandAppraisalItem, len(xct001Years))
	var fetchErr error
	for i, ch := range chs {
		r := <-ch
		if r.Err != nil {
			slog.WarnContext(ctx, "FetchLandAppraisals failed", "year", xct001Years[i], "area", area, "error", r.Err)
			fetchErr = r.Err
			continue
		}
		if len(r.Data.items) > 0 {
			itemsByYear[r.Data.year] = r.Data.items
		}
	}

	// 全年エラーの場合のみエラーを返す
	if len(itemsByYear) == 0 && fetchErr != nil {
		slog.ErrorContext(ctx, "FetchLandAppraisals failed for all years", "err", fetchErr)
		return domain.RentDeclineHint{}, fmt.Errorf("fetch land appraisals: %w", fetchErr)
	}

	return domain.CalcRentDeclineHint(itemsByYear), nil
}

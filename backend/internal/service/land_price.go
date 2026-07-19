package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// ErrEstimateDataInsufficient は理論価格推定に必要な建築年データが取引事例にない場合のエラー
var ErrEstimateDataInsufficient = errors.New("理論価格の推定に必要なデータが不足しています")

// ErrNoAppraisalData は指定エリアの地価公示データが存在しない場合のエラー
var ErrNoAppraisalData = errors.New("地価公示データが見つかりませんでした")

// LandMLITClient は土地価格分析に必要な国交省APIのサブセット
type LandMLITClient interface {
	FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
	FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
}

// LandPriceService は土地取引価格・地価公示の統計と比較を担う
type LandPriceService interface {
	// Stats は期間内の土地取引から統計を返す
	Stats(ctx context.Context, q mlit.LandPriceQuery) (domain.LandPriceStats, error)
	// Compare は検討中の土地価格と相場を比較する
	Compare(ctx context.Context, q mlit.LandPriceQuery, landPrice, areaSqm float64) (domain.LandPriceComparison, error)
	// Estimate は築年数・駅距離補正による理論価格と乖離率を返す。
	// 取引事例に建築年データがない場合は ErrEstimateDataInsufficient を返す。
	Estimate(ctx context.Context, q mlit.LandPriceQuery, in domain.TheoreticalPriceInput) (domain.TheoreticalPriceResult, error)
	// Appraisals は地価公示情報の比較統計を返す。
	// データが存在しない場合は ErrNoAppraisalData を返す。
	Appraisals(ctx context.Context, area, city string, year int, division string) (domain.AppraisalComparisonResult, error)
}

// LandPriceAnalysisService は LandPriceService の実装
type LandPriceAnalysisService struct {
	client LandMLITClient
}

func NewLandPriceAnalysisService(client LandMLITClient) LandPriceService {
	return &LandPriceAnalysisService{client: client}
}

func (s *LandPriceAnalysisService) fetchStats(ctx context.Context, q mlit.LandPriceQuery) (domain.LandPriceStats, error) {
	transactions, err := s.client.FetchLandPrices(ctx, q)
	if err != nil {
		slog.ErrorContext(ctx, "FetchLandPrices failed", "err", err)
		return domain.LandPriceStats{}, fmt.Errorf("fetch land prices: %w", err)
	}
	return domain.CalcLandPriceStats(ctx, transactions), nil
}

func (s *LandPriceAnalysisService) Stats(ctx context.Context, q mlit.LandPriceQuery) (domain.LandPriceStats, error) {
	return s.fetchStats(ctx, q)
}

func (s *LandPriceAnalysisService) Compare(ctx context.Context, q mlit.LandPriceQuery, landPrice, areaSqm float64) (domain.LandPriceComparison, error) {
	stats, err := s.fetchStats(ctx, q)
	if err != nil {
		return domain.LandPriceComparison{}, err
	}
	return domain.CompareLandPrice(stats, landPrice, areaSqm), nil
}

func (s *LandPriceAnalysisService) Estimate(ctx context.Context, q mlit.LandPriceQuery, in domain.TheoreticalPriceInput) (domain.TheoreticalPriceResult, error) {
	stats, err := s.fetchStats(ctx, q)
	if err != nil {
		return domain.TheoreticalPriceResult{}, err
	}
	spanCtx, span := otel.Tracer(domainTracerName).Start(ctx, "domain.EstimateTheoreticalPrice")
	defer span.End() // 純粋計算だが panic 時も確実に span を閉じる
	span.SetAttributes(attribute.Int("domain.transaction_count", stats.Count))
	result, ok := domain.EstimateTheoreticalPrice(spanCtx, stats, in)
	if !ok {
		return domain.TheoreticalPriceResult{}, ErrEstimateDataInsufficient
	}
	return result, nil
}

func (s *LandPriceAnalysisService) Appraisals(ctx context.Context, area, city string, year int, division string) (domain.AppraisalComparisonResult, error) {
	items, err := s.client.FetchLandAppraisals(ctx, area, city, year, division)
	if err != nil {
		slog.ErrorContext(ctx, "FetchLandAppraisals failed", "err", err)
		return domain.AppraisalComparisonResult{}, fmt.Errorf("fetch land appraisals: %w", err)
	}
	if len(items) == 0 {
		return domain.AppraisalComparisonResult{}, ErrNoAppraisalData
	}
	return domain.CalcAppraisalComparison(items), nil
}

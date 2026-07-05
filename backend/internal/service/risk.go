package service

import (
	"context"
	"log/slog"

	"github.com/yield-guard/backend/internal/concurrent"
	"github.com/yield-guard/backend/internal/domain"
)

// RiskMLITClient は都市計画リスク・ハザード評価に必要な国交省APIのサブセット
type RiskMLITClient interface {
	FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	FetchUrbanRoad(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error)
	FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
	FetchFloodHazard(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error)
	FetchStormHazard(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error)
	FetchTsunamiHazard(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error)
	FetchLandslideHazard(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error)
}

// RiskService はタイル単位の都市計画リスク・ハザード評価を担う。
// いずれの API が失敗しても警告ログのみで、取得できた結果からリスクを構築する（部分失敗許容）。
// 戻り値は常に非 nil スライス。
type RiskService interface {
	UrbanRisks(ctx context.Context, z, x, y int) []domain.UrbanRisk
	HazardRisks(ctx context.Context, z, x, y int) []domain.UrbanRisk
}

// RiskAssessmentService は RiskService の実装
type RiskAssessmentService struct {
	client RiskMLITClient
}

func NewRiskAssessmentService(client RiskMLITClient) RiskService {
	return &RiskAssessmentService{client: client}
}

// UrbanRisks は立地適正化・大規模盛土・都市計画道路・災害履歴から都市計画リスクを構築する
func (s *RiskAssessmentService) UrbanRisks(ctx context.Context, z, x, y int) []domain.UrbanRisk {
	// 4 API を並列取得。いずれか失敗してもログのみで他の結果は返す
	locCh := concurrent.FanOut(func() ([]domain.LocationOptimizationItem, error) {
		return s.client.FetchLocationOptimization(ctx, z, x, y)
	})
	embCh := concurrent.FanOut(func() ([]domain.EmbankmentItem, error) { return s.client.FetchEmbankment(ctx, z, x, y) })
	rdCh := concurrent.FanOut(func() ([]domain.UrbanRoadItem, error) { return s.client.FetchUrbanRoad(ctx, z, x, y) })
	disCh := concurrent.FanOut(func() ([]domain.DisasterHistoryItem, error) { return s.client.FetchDisasterHistory(ctx, z, x, y) })

	var location []domain.LocationOptimizationItem
	var embank []domain.EmbankmentItem
	var road []domain.UrbanRoadItem
	var disaster []domain.DisasterHistoryItem

	if r := <-locCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchLocationOptimization failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		location = r.Data
	}
	if r := <-embCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchEmbankment failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		embank = r.Data
	}
	if r := <-rdCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchUrbanRoad failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		road = r.Data
	}
	if r := <-disCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchDisasterHistory failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		disaster = r.Data
	}

	risks := domain.BuildUrbanRisksFromAPIs(location, embank, road, disaster)
	if risks == nil {
		risks = []domain.UrbanRisk{}
	}
	return risks
}

// HazardRisks は洪水・高潮・津波・土砂災害のハザード情報からリスクを構築する
func (s *RiskAssessmentService) HazardRisks(ctx context.Context, z, x, y int) []domain.UrbanRisk {
	floCh := concurrent.FanOut(func() ([]domain.FloodHazardItem, error) { return s.client.FetchFloodHazard(ctx, z, x, y) })
	stmCh := concurrent.FanOut(func() ([]domain.StormHazardItem, error) { return s.client.FetchStormHazard(ctx, z, x, y) })
	tsuCh := concurrent.FanOut(func() ([]domain.TsunamiHazardItem, error) { return s.client.FetchTsunamiHazard(ctx, z, x, y) })
	lsCh := concurrent.FanOut(func() ([]domain.LandslideHazardItem, error) { return s.client.FetchLandslideHazard(ctx, z, x, y) })

	var floods []domain.FloodHazardItem
	var storms []domain.StormHazardItem
	var tsunamis []domain.TsunamiHazardItem
	var landslides []domain.LandslideHazardItem

	if r := <-floCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchFloodHazard failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		floods = r.Data
	}
	if r := <-stmCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchStormHazard failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		storms = r.Data
	}
	if r := <-tsuCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchTsunamiHazard failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		tsunamis = r.Data
	}
	if r := <-lsCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchLandslideHazard failed", "z", z, "x", x, "y", y, "error", r.Err)
	} else {
		landslides = r.Data
	}

	risks := domain.BuildHazardRisks(floods, storms, tsunamis, landslides)
	if risks == nil {
		risks = []domain.UrbanRisk{}
	}
	return risks
}

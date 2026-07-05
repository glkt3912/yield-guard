package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yield-guard/backend/internal/domain"
)

// mockRiskClient は RiskMLITClient のテスト用モック。未設定の API は空結果を返す。
type mockRiskClient struct {
	locationFunc  func(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	embankFunc    func(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	roadFunc      func(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error)
	disasterFunc  func(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
	floodFunc     func(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error)
	stormFunc     func(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error)
	tsunamiFunc   func(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error)
	landslideFunc func(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error)
}

func (m *mockRiskClient) FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error) {
	if m.locationFunc != nil {
		return m.locationFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error) {
	if m.embankFunc != nil {
		return m.embankFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchUrbanRoad(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error) {
	if m.roadFunc != nil {
		return m.roadFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error) {
	if m.disasterFunc != nil {
		return m.disasterFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchFloodHazard(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error) {
	if m.floodFunc != nil {
		return m.floodFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchStormHazard(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error) {
	if m.stormFunc != nil {
		return m.stormFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchTsunamiHazard(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error) {
	if m.tsunamiFunc != nil {
		return m.tsunamiFunc(ctx, z, x, y)
	}
	return nil, nil
}

func (m *mockRiskClient) FetchLandslideHazard(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error) {
	if m.landslideFunc != nil {
		return m.landslideFunc(ctx, z, x, y)
	}
	return nil, nil
}

// ---- UrbanRisks ----

func TestUrbanRisks_WithRisks(t *testing.T) {
	client := &mockRiskClient{
		embankFunc: func(_ context.Context, _, _, _ int) ([]domain.EmbankmentItem, error) {
			return []domain.EmbankmentItem{{Classification: "谷埋め型"}}, nil
		},
		roadFunc: func(_ context.Context, _, _, _ int) ([]domain.UrbanRoadItem, error) {
			return []domain.UrbanRoadItem{{PlanningRoadJa: "都市計画道路A", KubunID: 3011}}, nil
		},
		disasterFunc: func(_ context.Context, _, _, _ int) ([]domain.DisasterHistoryItem, error) {
			return []domain.DisasterHistoryItem{{Name: "浸水域", Year: 2019}}, nil
		},
	}
	svc := NewRiskAssessmentService(client)

	risks := svc.UrbanRisks(context.Background(), 14, 100, 200)
	codes := make(map[string]bool)
	for _, r := range risks {
		codes[r.Code] = true
	}
	for _, want := range []string{"LARGE_EMBANKMENT", "URBAN_PLANNING_ROAD", "DISASTER_HISTORY"} {
		if !codes[want] {
			t.Errorf("expected risk code %s in result", want)
		}
	}
}

// 一部 API が失敗しても他の結果からリスクを構築する
func TestUrbanRisks_PartialAPIFailure(t *testing.T) {
	client := &mockRiskClient{
		embankFunc: func(_ context.Context, _, _, _ int) ([]domain.EmbankmentItem, error) {
			return nil, errors.New("API timeout")
		},
		disasterFunc: func(_ context.Context, _, _, _ int) ([]domain.DisasterHistoryItem, error) {
			return []domain.DisasterHistoryItem{{Name: "がけ崩れ", Year: 2011}}, nil
		},
	}
	svc := NewRiskAssessmentService(client)

	risks := svc.UrbanRisks(context.Background(), 14, 100, 200)
	found := false
	for _, r := range risks {
		if r.Code == "DISASTER_HISTORY" {
			found = true
		}
	}
	if !found {
		t.Error("expected DISASTER_HISTORY in result despite embankment API failure")
	}
}

func TestUrbanRisks_EmptyIsNonNil(t *testing.T) {
	svc := NewRiskAssessmentService(&mockRiskClient{})

	risks := svc.UrbanRisks(context.Background(), 14, 100, 200)
	if risks == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(risks) != 0 {
		t.Errorf("expected 0 risks, got %d", len(risks))
	}
}

// ---- HazardRisks ----

func TestHazardRisks_Success(t *testing.T) {
	client := &mockRiskClient{
		floodFunc: func(_ context.Context, _, _, _ int) ([]domain.FloodHazardItem, error) {
			return []domain.FloodHazardItem{{DepthRank: 4, RiverName: "荒川"}}, nil
		},
		tsunamiFunc: func(_ context.Context, _, _, _ int) ([]domain.TsunamiHazardItem, error) {
			return []domain.TsunamiHazardItem{{DepthJa: "3m以上"}}, nil
		},
	}
	svc := NewRiskAssessmentService(client)

	risks := svc.HazardRisks(context.Background(), 14, 100, 200)
	if len(risks) != 2 {
		t.Fatalf("expected 2 risks (flood + tsunami), got %d", len(risks))
	}
	codes := map[string]bool{}
	for _, r := range risks {
		codes[r.Code] = true
	}
	if !codes["FLOOD_HAZARD"] {
		t.Error("expected FLOOD_HAZARD in result")
	}
	if !codes["TSUNAMI_HAZARD"] {
		t.Error("expected TSUNAMI_HAZARD in result")
	}
}

func TestHazardRisks_PartialAPIFailure(t *testing.T) {
	client := &mockRiskClient{
		floodFunc: func(_ context.Context, _, _, _ int) ([]domain.FloodHazardItem, error) {
			return nil, errors.New("upstream error")
		},
		landslideFunc: func(_ context.Context, _, _, _ int) ([]domain.LandslideHazardItem, error) {
			return []domain.LandslideHazardItem{{PhenomenonType: 2, ZoneCode: 1}}, nil
		},
	}
	svc := NewRiskAssessmentService(client)

	// flood 失敗でも他のハザード結果は含まれること
	risks := svc.HazardRisks(context.Background(), 14, 100, 200)
	if len(risks) != 1 || risks[0].Code != "LANDSLIDE_HAZARD" {
		t.Errorf("expected only LANDSLIDE_HAZARD, got %+v", risks)
	}
}

func TestHazardRisks_EmptyIsNonNil(t *testing.T) {
	svc := NewRiskAssessmentService(&mockRiskClient{})

	risks := svc.HazardRisks(context.Background(), 14, 100, 200)
	if risks == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(risks) != 0 {
		t.Errorf("expected 0 risks, got %d", len(risks))
	}
}

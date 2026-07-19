package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/yield-guard/backend/internal/concurrent"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// MLITClient は投資スコア計算に必要な国交省APIのサブセット
type MLITClient interface {
	FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
	FetchStationRidership(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error)
	FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
	FetchUrbanZoning(ctx context.Context, z, x, y int) ([]domain.UrbanZoningItem, error)
	FetchLiquefaction(ctx context.Context, z, x, y int) ([]domain.LiquefactionRiskItem, error)
	FetchFloodHazard(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error)
	FetchStormHazard(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error)
	FetchTsunamiHazard(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error)
	FetchLandslideHazard(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error)
	FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
}

// LocationService はタイル単位の投資スコア計算を担う
type LocationService interface {
	CalcScoreForTile(ctx context.Context, z, x, y int) (domain.InvestmentScoreResult, error)
}

// InvestmentScoreService は LocationService の実装
type InvestmentScoreService struct {
	client MLITClient
}

func NewInvestmentScoreService(client MLITClient) LocationService {
	return &InvestmentScoreService{client: client}
}

// CalcScoreForTile は指定タイル座標に対して複数 API を並列取得し投資適地スコアを返す。
// 個別 API の失敗は警告ログのみでスキップする。コンテキストキャンセル時はエラーを返す。
func (s *InvestmentScoreService) CalcScoreForTile(ctx context.Context, z, x, y int) (domain.InvestmentScoreResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.InvestmentScoreResult{}, err
	}

	popCh := concurrent.FanOut(func() ([]domain.PopulationForecastItem, error) { return s.client.FetchPopulationForecast(ctx, z, x, y) })
	ridCh := concurrent.FanOut(func() ([]mlit.StationRidership, error) { return s.client.FetchStationRidership(ctx, z, x, y) })
	locCh := concurrent.FanOut(func() ([]domain.LocationOptimizationItem, error) { return s.client.FetchLocationOptimization(ctx, z, x, y) })
	embCh := concurrent.FanOut(func() ([]domain.EmbankmentItem, error) { return s.client.FetchEmbankment(ctx, z, x, y) })
	disCh := concurrent.FanOut(func() ([]domain.DisasterHistoryItem, error) { return s.client.FetchDisasterHistory(ctx, z, x, y) })
	zonCh := concurrent.FanOut(func() ([]domain.UrbanZoningItem, error) { return s.client.FetchUrbanZoning(ctx, z, x, y) })
	liqCh := concurrent.FanOut(func() ([]domain.LiquefactionRiskItem, error) { return s.client.FetchLiquefaction(ctx, z, x, y) })
	floCh := concurrent.FanOut(func() ([]domain.FloodHazardItem, error) { return s.client.FetchFloodHazard(ctx, z, x, y) })
	stoCh := concurrent.FanOut(func() ([]domain.StormHazardItem, error) { return s.client.FetchStormHazard(ctx, z, x, y) })
	tsuCh := concurrent.FanOut(func() ([]domain.TsunamiHazardItem, error) { return s.client.FetchTsunamiHazard(ctx, z, x, y) })
	lanCh := concurrent.FanOut(func() ([]domain.LandslideHazardItem, error) { return s.client.FetchLandslideHazard(ctx, z, x, y) })

	// 地価トレンド: タイル中心座標→都道府県コードを逆引きし、新旧2期間を並列取得する
	// 直近2年 vs その2年前の期間を動的に算出する
	centerLat, centerLng := mlit.TileToLatLng(x, y, z)
	prefCode := domain.PrefCodeFromLatLng(centerLat, centerLng)
	type landResult struct {
		stats domain.LandPriceStats
		err   error
	}
	recentLandCh := make(chan landResult, 1)
	oldLandCh := make(chan landResult, 1)
	if prefCode != "" {
		now := time.Now().Year()
		go func() {
			tx, e := s.client.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: now - 2, Quarter: 1, ToYear: now - 1, ToQuarter: 4})
			recentLandCh <- landResult{domain.CalcLandPriceStats(ctx, tx), e}
		}()
		go func() {
			tx, e := s.client.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: now - 4, Quarter: 1, ToYear: now - 3, ToQuarter: 4})
			oldLandCh <- landResult{domain.CalcLandPriceStats(ctx, tx), e}
		}()
	} else {
		recentLandCh <- landResult{}
		oldLandCh <- landResult{}
	}

	input := domain.InvestmentScoreInput{}

	if r := <-popCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchPopulationForecast failed", "error", r.Err)
	} else {
		input.PopulationItems = r.Data
	}
	if r := <-ridCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchStationRidership failed", "error", r.Err)
	} else {
		input.StationRiderships = make([]domain.StationRidershipResult, 0, len(r.Data))
		for _, st := range r.Data {
			score := domain.CalcRidershipDemandScore(st.Passengers)
			input.StationRiderships = append(input.StationRiderships, domain.StationRidershipResult{
				StationName: st.StationName,
				LineName:    st.LineName,
				Passengers:  st.Passengers,
				DemandScore: score,
				Correction:  domain.RidershipCorrectionFactor(score),
			})
		}
	}
	if r := <-locCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchLocationOptimization failed", "error", r.Err)
	} else {
		input.LocationItems = r.Data
	}
	if r := <-embCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchEmbankment failed", "error", r.Err)
	} else {
		input.EmbankmentItems = r.Data
	}
	if r := <-disCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchDisasterHistory failed", "error", r.Err)
	} else {
		input.DisasterItems = r.Data
	}
	if r := <-zonCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchUrbanZoning failed", "error", r.Err)
	} else {
		input.UrbanZoningItems = r.Data
	}
	if r := <-liqCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchLiquefaction failed", "error", r.Err)
	} else {
		input.LiquefactionItems = r.Data
	}
	if r := <-floCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchFloodHazard failed", "error", r.Err)
	} else {
		input.FloodItems = r.Data
	}
	if r := <-stoCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchStormHazard failed", "error", r.Err)
	} else {
		input.StormItems = r.Data
	}
	if r := <-tsuCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchTsunamiHazard failed", "error", r.Err)
	} else {
		input.TsunamiItems = r.Data
	}
	if r := <-lanCh; r.Err != nil {
		slog.WarnContext(ctx, "FetchLandslideHazard failed", "error", r.Err)
	} else {
		input.LandslideItems = r.Data
	}

	recentLand := <-recentLandCh
	oldLand := <-oldLandCh
	if recentLand.err != nil {
		slog.WarnContext(ctx, "FetchLandPrices (recent) failed", "error", recentLand.err)
	}
	if oldLand.err != nil {
		slog.WarnContext(ctx, "FetchLandPrices (old) failed", "error", oldLand.err)
	}
	if recentLand.err == nil && oldLand.err == nil &&
		recentLand.stats.MedianTsubo > 0 && oldLand.stats.MedianTsubo > 0 {
		input.LandPriceChangeRate = (recentLand.stats.MedianTsubo - oldLand.stats.MedianTsubo) / oldLand.stats.MedianTsubo
		input.HasLandPriceTrend = true
	}

	result := domain.CalcInvestmentScore(input)
	// domain はコード値を返す。API 契約（grade は日本語）を維持するため service で変換する。
	result.Grade = gradeLabelFor(result.Grade)
	return result, nil
}

// gradeLabelFor は domain.CalcInvestmentScore が返す評価コードを日本語表示ラベルへ変換する。
// 表示文言は domain（純粋計算層）から分離し service 層で管理する。
func gradeLabelFor(code string) string {
	switch code {
	case "excellent":
		return "優良"
	case "good":
		return "良好"
	case "average":
		return "普通"
	case "caution":
		return "注意"
	case "warning":
		return "要注意"
	default:
		return code
	}
}

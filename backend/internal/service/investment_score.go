package service

import (
	"context"
	"log/slog"

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

func NewInvestmentScoreService(client MLITClient) *InvestmentScoreService {
	return &InvestmentScoreService{client: client}
}

type apiResult[T any] struct {
	data T
	err  error
}

func fanOut[T any](fetch func() (T, error)) <-chan apiResult[T] {
	ch := make(chan apiResult[T], 1)
	go func() {
		d, e := fetch()
		ch <- apiResult[T]{d, e}
	}()
	return ch
}

// CalcScoreForTile は指定タイル座標に対して複数 API を並列取得し投資適地スコアを返す。
// 個別 API の失敗は警告ログのみでスキップする。コンテキストキャンセル時はエラーを返す。
func (s *InvestmentScoreService) CalcScoreForTile(ctx context.Context, z, x, y int) (domain.InvestmentScoreResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.InvestmentScoreResult{}, err
	}

	popCh := fanOut(func() ([]domain.PopulationForecastItem, error) { return s.client.FetchPopulationForecast(ctx, z, x, y) })
	ridCh := fanOut(func() ([]mlit.StationRidership, error) { return s.client.FetchStationRidership(ctx, z, x, y) })
	locCh := fanOut(func() ([]domain.LocationOptimizationItem, error) { return s.client.FetchLocationOptimization(ctx, z, x, y) })
	embCh := fanOut(func() ([]domain.EmbankmentItem, error) { return s.client.FetchEmbankment(ctx, z, x, y) })
	disCh := fanOut(func() ([]domain.DisasterHistoryItem, error) { return s.client.FetchDisasterHistory(ctx, z, x, y) })
	zonCh := fanOut(func() ([]domain.UrbanZoningItem, error) { return s.client.FetchUrbanZoning(ctx, z, x, y) })
	liqCh := fanOut(func() ([]domain.LiquefactionRiskItem, error) { return s.client.FetchLiquefaction(ctx, z, x, y) })
	floCh := fanOut(func() ([]domain.FloodHazardItem, error) { return s.client.FetchFloodHazard(ctx, z, x, y) })
	stoCh := fanOut(func() ([]domain.StormHazardItem, error) { return s.client.FetchStormHazard(ctx, z, x, y) })
	tsuCh := fanOut(func() ([]domain.TsunamiHazardItem, error) { return s.client.FetchTsunamiHazard(ctx, z, x, y) })
	lanCh := fanOut(func() ([]domain.LandslideHazardItem, error) { return s.client.FetchLandslideHazard(ctx, z, x, y) })

	// 地価トレンド: タイル中心座標→都道府県コードを逆引きし、新旧2期間を並列取得する
	centerLat, centerLng := mlit.TileToLatLng(x, y, z)
	prefCode := domain.PrefCodeFromLatLng(centerLat, centerLng)
	type landResult struct {
		stats domain.LandPriceStats
		err   error
	}
	recentLandCh := make(chan landResult, 1)
	oldLandCh := make(chan landResult, 1)
	if prefCode != "" {
		go func() {
			tx, e := s.client.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: 2023, Quarter: 1, ToYear: 2024, ToQuarter: 4})
			recentLandCh <- landResult{domain.CalcLandPriceStats(ctx, tx), e}
		}()
		go func() {
			tx, e := s.client.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: 2021, Quarter: 1, ToYear: 2022, ToQuarter: 4})
			oldLandCh <- landResult{domain.CalcLandPriceStats(ctx, tx), e}
		}()
	} else {
		recentLandCh <- landResult{}
		oldLandCh <- landResult{}
	}

	input := domain.InvestmentScoreInput{}

	if r := <-popCh; r.err != nil {
		slog.WarnContext(ctx, "FetchPopulationForecast failed", "error", r.err)
	} else {
		input.PopulationItems = r.data
	}
	if r := <-ridCh; r.err != nil {
		slog.WarnContext(ctx, "FetchStationRidership failed", "error", r.err)
	} else {
		input.StationRiderships = make([]domain.StationRidershipResult, 0, len(r.data))
		for _, st := range r.data {
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
	if r := <-locCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLocationOptimization failed", "error", r.err)
	} else {
		input.LocationItems = r.data
	}
	if r := <-embCh; r.err != nil {
		slog.WarnContext(ctx, "FetchEmbankment failed", "error", r.err)
	} else {
		input.EmbankmentItems = r.data
	}
	if r := <-disCh; r.err != nil {
		slog.WarnContext(ctx, "FetchDisasterHistory failed", "error", r.err)
	} else {
		input.DisasterItems = r.data
	}
	if r := <-zonCh; r.err != nil {
		slog.WarnContext(ctx, "FetchUrbanZoning failed", "error", r.err)
	} else {
		input.UrbanZoningItems = r.data
	}
	if r := <-liqCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLiquefaction failed", "error", r.err)
	} else {
		input.LiquefactionItems = r.data
	}
	if r := <-floCh; r.err != nil {
		slog.WarnContext(ctx, "FetchFloodHazard failed", "error", r.err)
	} else {
		input.FloodItems = r.data
	}
	if r := <-stoCh; r.err != nil {
		slog.WarnContext(ctx, "FetchStormHazard failed", "error", r.err)
	} else {
		input.StormItems = r.data
	}
	if r := <-tsuCh; r.err != nil {
		slog.WarnContext(ctx, "FetchTsunamiHazard failed", "error", r.err)
	} else {
		input.TsunamiItems = r.data
	}
	if r := <-lanCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLandslideHazard failed", "error", r.err)
	} else {
		input.LandslideItems = r.data
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

	return domain.CalcInvestmentScore(input), nil
}

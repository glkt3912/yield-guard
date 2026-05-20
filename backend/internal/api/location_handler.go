package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// GetStationRidership は物件の緯度経度から駅別乗降客数と需要スコアを返す（XKT015）
// @Summary     駅別乗降客数
// @Tags        location
// @Produce     json
// @Param       lat  query  number   true  "緯度"
// @Param       lng  query  number   true  "経度"
// @Param       z    query  integer  false "ズームレベル (11-15, デフォルト: 14)"
// @Success     200  {array}   domain.StationRidershipResult
// @Failure     400  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/station-ridership [get]
func (h *Handler) GetStationRidership(c *gin.Context) {
	lat, lng, ok := parseLatLng(c, coordsGlobal)
	if !ok {
		return
	}
	z, ok := parseZoom(c, 14)
	if !ok {
		return
	}

	tx, ty := mlit.LatLngToTile(lat, lng, z)

	stations, err := h.mlitClient.FetchStationRidership(c.Request.Context(), z, tx, ty)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchStationRidership failed", "err", err)
		badGateway(c, "駅別乗降客数の取得に失敗しました")
		return
	}

	results := make([]domain.StationRidershipResult, 0, len(stations))
	for _, s := range stations {
		score := domain.CalcRidershipDemandScore(s.Passengers)
		results = append(results, domain.StationRidershipResult{
			StationName: s.StationName,
			LineName:    s.LineName,
			Passengers:  s.Passengers,
			DemandScore: score,
			Correction:  domain.RidershipCorrectionFactor(score),
		})
	}

	c.JSON(http.StatusOK, results)
}

// GetPopulationForecast は物件の緯度経度から将来推計人口と人口減少シナリオを返す（XKT013）
// @Summary     将来推計人口
// @Tags        location
// @Produce     json
// @Param       lat  query  number   true  "緯度"
// @Param       lng  query  number   true  "経度"
// @Param       z    query  integer  false "ズームレベル (11-15, デフォルト: 14)"
// @Success     200  {object}  domain.PopulationForecastResult
// @Failure     400  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/population-forecast [get]
func (h *Handler) GetPopulationForecast(c *gin.Context) {
	lat, lng, ok := parseLatLng(c, coordsGlobal)
	if !ok {
		return
	}
	z, ok := parseZoom(c, 14)
	if !ok {
		return
	}

	tx, ty := mlit.LatLngToTile(lat, lng, z)

	items, err := h.mlitClient.FetchPopulationForecast(c.Request.Context(), z, tx, ty)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchPopulationForecast failed", "err", err)
		badGateway(c, "将来推計人口の取得に失敗しました")
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"snapshots": []struct{}{}, "changeRate30yr": 0, "vacancyRateDelta": 0, "trend": ""})
		return
	}

	result := domain.CalcPopulationForecast(items)
	c.JSON(http.StatusOK, result)
}

// GetUrbanRisks は緯度経度から都市計画リスクを一括取得する
// @Summary     都市計画リスク
// @Tags        location
// @Produce     json
// @Param       lat  query  number  true  "緯度 (日本国内 20-46)"
// @Param       lng  query  number  true  "経度 (日本国内 122-154)"
// @Success     200  {array}   domain.UrbanRisk
// @Failure     400  {object}  map[string]string
// @Router      /api/urban-risks [get]
func (h *Handler) GetUrbanRisks(c *gin.Context) {
	lat, lng, ok := parseLatLng(c, coordsJapanOnly)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	z := 14
	x, y := mlit.LatLngToTile(lat, lng, z)

	type result struct {
		location []domain.LocationOptimizationItem
		embank   []domain.EmbankmentItem
		road     []domain.UrbanRoadItem
		disaster []domain.DisasterHistoryItem
	}
	var res result

	locCh := fanOut(func() ([]domain.LocationOptimizationItem, error) { return h.mlitClient.FetchLocationOptimization(ctx, z, x, y) })
	embCh := fanOut(func() ([]domain.EmbankmentItem, error) { return h.mlitClient.FetchEmbankment(ctx, z, x, y) })
	rdCh := fanOut(func() ([]domain.UrbanRoadItem, error) { return h.mlitClient.FetchUrbanRoad(ctx, z, x, y) })
	disCh := fanOut(func() ([]domain.DisasterHistoryItem, error) { return h.mlitClient.FetchDisasterHistory(ctx, z, x, y) })

	if r := <-locCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLocationOptimization failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		res.location = r.data
	}
	if r := <-embCh; r.err != nil {
		slog.WarnContext(ctx, "FetchEmbankment failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		res.embank = r.data
	}
	if r := <-rdCh; r.err != nil {
		slog.WarnContext(ctx, "FetchUrbanRoad failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		res.road = r.data
	}
	if r := <-disCh; r.err != nil {
		slog.WarnContext(ctx, "FetchDisasterHistory failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		res.disaster = r.data
	}

	risks := domain.BuildUrbanRisksFromAPIs(res.location, res.embank, res.road, res.disaster)
	if risks == nil {
		risks = []domain.UrbanRisk{}
	}
	c.JSON(http.StatusOK, risks)
}

// GetHazardInfo は物件の緯度経度から洪水・高潮・津波・土砂災害のハザード情報を返す
// @Summary     ハザード情報
// @Tags        location
// @Produce     json
// @Param       lat  query  number  true  "緯度 (日本国内 20-46)"
// @Param       lng  query  number  true  "経度 (日本国内 122-154)"
// @Success     200  {array}   domain.UrbanRisk
// @Failure     400  {object}  map[string]string
// @Router      /api/hazard [get]
func (h *Handler) GetHazardInfo(c *gin.Context) {
	lat, lng, ok := parseLatLng(c, coordsJapanOnly)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	z := 14
	x, y := mlit.LatLngToTile(lat, lng, z)

	floCh := fanOut(func() ([]domain.FloodHazardItem, error) { return h.mlitClient.FetchFloodHazard(ctx, z, x, y) })
	stmCh := fanOut(func() ([]domain.StormHazardItem, error) { return h.mlitClient.FetchStormHazard(ctx, z, x, y) })
	tsuCh := fanOut(func() ([]domain.TsunamiHazardItem, error) { return h.mlitClient.FetchTsunamiHazard(ctx, z, x, y) })
	lsCh := fanOut(func() ([]domain.LandslideHazardItem, error) { return h.mlitClient.FetchLandslideHazard(ctx, z, x, y) })

	var floods []domain.FloodHazardItem
	var storms []domain.StormHazardItem
	var tsunamis []domain.TsunamiHazardItem
	var landslides []domain.LandslideHazardItem

	if r := <-floCh; r.err != nil {
		slog.WarnContext(ctx, "FetchFloodHazard failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		floods = r.data
	}
	if r := <-stmCh; r.err != nil {
		slog.WarnContext(ctx, "FetchStormHazard failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		storms = r.data
	}
	if r := <-tsuCh; r.err != nil {
		slog.WarnContext(ctx, "FetchTsunamiHazard failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		tsunamis = r.data
	}
	if r := <-lsCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLandslideHazard failed", "z", z, "x", x, "y", y, "error", r.err)
	} else {
		landslides = r.data
	}

	risks := domain.BuildHazardRisks(floods, storms, tsunamis, landslides)
	if risks == nil {
		risks = []domain.UrbanRisk{}
	}
	c.JSON(http.StatusOK, risks)
}

// calcScoreForTile は指定タイル座標に対して複数 API を並列取得し投資適地スコアを返す。
// 個別 API の失敗は警告ログのみでスキップする。コンテキストキャンセル時はエラーを返す。
func (h *Handler) calcScoreForTile(ctx context.Context, z, x, y int) (domain.InvestmentScoreResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.InvestmentScoreResult{}, err
	}
	popCh := fanOut(func() ([]domain.PopulationForecastItem, error) { return h.mlitClient.FetchPopulationForecast(ctx, z, x, y) })
	ridCh := fanOut(func() ([]mlit.StationRidership, error) { return h.mlitClient.FetchStationRidership(ctx, z, x, y) })
	locCh := fanOut(func() ([]domain.LocationOptimizationItem, error) { return h.mlitClient.FetchLocationOptimization(ctx, z, x, y) })
	embCh := fanOut(func() ([]domain.EmbankmentItem, error) { return h.mlitClient.FetchEmbankment(ctx, z, x, y) })
	disCh := fanOut(func() ([]domain.DisasterHistoryItem, error) { return h.mlitClient.FetchDisasterHistory(ctx, z, x, y) })
	zonCh := fanOut(func() ([]domain.UrbanZoningItem, error) { return h.mlitClient.FetchUrbanZoning(ctx, z, x, y) })
	liqCh := fanOut(func() ([]domain.LiquefactionRiskItem, error) { return h.mlitClient.FetchLiquefaction(ctx, z, x, y) })
	floCh := fanOut(func() ([]domain.FloodHazardItem, error) { return h.mlitClient.FetchFloodHazard(ctx, z, x, y) })
	stoCh := fanOut(func() ([]domain.StormHazardItem, error) { return h.mlitClient.FetchStormHazard(ctx, z, x, y) })
	tsuCh := fanOut(func() ([]domain.TsunamiHazardItem, error) { return h.mlitClient.FetchTsunamiHazard(ctx, z, x, y) })
	lanCh := fanOut(func() ([]domain.LandslideHazardItem, error) { return h.mlitClient.FetchLandslideHazard(ctx, z, x, y) })

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
			tx, e := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: 2023, Quarter: 1, ToYear: 2024, ToQuarter: 4})
			recentLandCh <- landResult{domain.CalcLandPriceStats(ctx, tx), e}
		}()
		go func() {
			tx, e := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: 2021, Quarter: 1, ToYear: 2022, ToQuarter: 4})
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
		for _, s := range r.data {
			score := domain.CalcRidershipDemandScore(s.Passengers)
			input.StationRiderships = append(input.StationRiderships, domain.StationRidershipResult{
				StationName: s.StationName,
				LineName:    s.LineName,
				Passengers:  s.Passengers,
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
	if recentLand.err != nil || oldLand.err != nil {
		if recentLand.err != nil {
			slog.WarnContext(ctx, "FetchLandPrices (recent) failed", "error", recentLand.err)
		}
		if oldLand.err != nil {
			slog.WarnContext(ctx, "FetchLandPrices (old) failed", "error", oldLand.err)
		}
	} else if recentLand.stats.MedianTsubo > 0 && oldLand.stats.MedianTsubo > 0 {
		input.LandPriceChangeRate = (recentLand.stats.MedianTsubo - oldLand.stats.MedianTsubo) / oldLand.stats.MedianTsubo
		input.HasLandPriceTrend = true
	}

	return domain.CalcInvestmentScore(input), nil
}

// GetInvestmentScore は物件の緯度経度から投資適地スコアを算出して返す
// @Summary     投資適地スコア
// @Tags        location
// @Produce     json
// @Param       lat  query  number  true  "緯度 (日本国内 20-46)"
// @Param       lng  query  number  true  "経度 (日本国内 122-154)"
// @Success     200  {object}  domain.InvestmentScoreResult
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /api/investment-score [get]
func (h *Handler) GetInvestmentScore(c *gin.Context) {
	lat, lng, ok := parseLatLng(c, coordsJapanOnly)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	z := 14
	x, y := mlit.LatLngToTile(lat, lng, z)

	score, err := h.calcScoreForTile(ctx, z, x, y)
	if err != nil {
		slog.ErrorContext(ctx, "calcScoreForTile failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "投資スコアの計算に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, score)
}

const maxHeatmapTiles = 50

// GetInvestmentScoreHeatmap はバウンディングボックス内の全タイルに対して投資スコアを並列計算して返す
// @Summary     投資スコアヒートマップ
// @Tags        location
// @Produce     json
// @Param       minLat  query  number   true  "南端緯度"
// @Param       maxLat  query  number   true  "北端緯度"
// @Param       minLng  query  number   true  "西端経度"
// @Param       maxLng  query  number   true  "東端経度"
// @Param       z       query  integer  false "ズームレベル (11-15, デフォルト: 13)"
// @Success     200  {object}  domain.HeatmapResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /api/investment-score-heatmap [get]
func (h *Handler) GetInvestmentScoreHeatmap(c *gin.Context) {
	minLatStr := c.Query("minLat")
	maxLatStr := c.Query("maxLat")
	minLngStr := c.Query("minLng")
	maxLngStr := c.Query("maxLng")
	if minLatStr == "" || maxLatStr == "" || minLngStr == "" || maxLngStr == "" {
		badRequest(c, "minLat, maxLat, minLng, maxLng は必須パラメータです")
		return
	}
	minLat, err := strconv.ParseFloat(minLatStr, 64)
	if err != nil {
		badRequest(c, "minLat の値が不正です")
		return
	}
	maxLat, err := strconv.ParseFloat(maxLatStr, 64)
	if err != nil {
		badRequest(c, "maxLat の値が不正です")
		return
	}
	minLng, err := strconv.ParseFloat(minLngStr, 64)
	if err != nil {
		badRequest(c, "minLng の値が不正です")
		return
	}
	maxLng, err := strconv.ParseFloat(maxLngStr, 64)
	if err != nil {
		badRequest(c, "maxLng の値が不正です")
		return
	}

	if minLat >= maxLat {
		badRequest(c, "minLat は maxLat より小さい値を指定してください")
		return
	}
	if minLng >= maxLng {
		badRequest(c, "minLng は maxLng より小さい値を指定してください")
		return
	}
	if minLat < 20 || maxLat > 46 {
		badRequest(c, "緯度は日本国内（20〜46）の範囲で指定してください")
		return
	}
	if minLng < 122 || maxLng > 154 {
		badRequest(c, "経度は日本国内（122〜154）の範囲で指定してください")
		return
	}

	z, ok := parseZoom(c, 13)
	if !ok {
		return
	}

	// parseZoom は z ∈ [11,15] を保証する
	var maxTiles int
	switch z {
	case 11, 12:
		maxTiles = 20
	case 13, 14:
		maxTiles = 30
	default: // z == 15
		maxTiles = maxHeatmapTiles
	}

	xMin, yMin := mlit.LatLngToTile(maxLat, minLng, z)
	xMax, yMax := mlit.LatLngToTile(minLat, maxLng, z)

	tileCount := (xMax - xMin + 1) * (yMax - yMin + 1)
	if tileCount > maxTiles {
		badRequest(c, fmt.Sprintf("too many tiles: max %d for zoom level %d", maxTiles, z))
		return
	}

	ctx := c.Request.Context()

	sem := make(chan struct{}, 5)
	type tileResult struct {
		tile domain.HeatmapTile
		err  error
	}
	results := make(chan tileResult, tileCount)

	for tx := xMin; tx <= xMax; tx++ {
		for ty := yMin; ty <= yMax; ty++ {
			go func(tx, ty int) {
				select {
				case <-ctx.Done():
					results <- tileResult{err: ctx.Err()}
					return
				case sem <- struct{}{}:
				}
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						results <- tileResult{err: fmt.Errorf("panic in calcScoreForTile: %v", r)}
					}
				}()
				score, err := h.calcScoreForTile(ctx, z, tx, ty)
				if err != nil {
					results <- tileResult{err: err}
					return
				}
				lat, lng := mlit.TileToLatLng(tx, ty, z)
				results <- tileResult{tile: domain.HeatmapTile{
					X:          tx,
					Y:          ty,
					Z:          z,
					CenterLat:  lat,
					CenterLng:  lng,
					TotalScore: score.TotalScore,
					Grade:      score.Grade,
				}}
			}(tx, ty)
		}
	}

	tiles := make([]domain.HeatmapTile, 0, tileCount)
	for i := 0; i < tileCount; i++ {
		r := <-results
		if r.err != nil {
			slog.WarnContext(ctx, "calcScoreForTile failed", "error", r.err)
			continue
		}
		tiles = append(tiles, r.tile)
	}

	c.JSON(http.StatusOK, domain.HeatmapResponse{
		Tiles:     tiles,
		TileCount: len(tiles),
	})
}

package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

const maxHeatmapTiles = 50

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

	score, err := h.locationSvc.CalcScoreForTile(ctx, z, x, y)
	if err != nil {
		slog.ErrorContext(ctx, "CalcScoreForTile failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "投資スコアの計算に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, score)
}

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

	// ズームレベルに応じてタイル数上限を決定
	// z=11-12: 20, z=13-14: 30, z=15: 50
	var maxTiles int
	switch {
	case z <= 12:
		maxTiles = 20
	case z <= 14:
		maxTiles = 30
	default:
		maxTiles = maxHeatmapTiles
	}

	// 高緯度 → 小さい y 値のため注意
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
				defer func() {
					if r := recover(); r != nil {
						results <- tileResult{err: fmt.Errorf("panic in CalcScoreForTile: %v", r)}
					}
				}()
				select {
				case <-ctx.Done():
					results <- tileResult{err: ctx.Err()}
					return
				case sem <- struct{}{}:
				}
				defer func() { <-sem }()
				score, err := h.locationSvc.CalcScoreForTile(ctx, z, tx, ty)
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
			slog.WarnContext(ctx, "CalcScoreForTile failed", "error", r.err)
			continue
		}
		tiles = append(tiles, r.tile)
	}

	c.JSON(http.StatusOK, domain.HeatmapResponse{
		Tiles:     tiles,
		TileCount: len(tiles),
	})
}

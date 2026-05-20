package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

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

	// 4 API を並列取得。いずれか失敗してもログのみで他の結果は返す
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

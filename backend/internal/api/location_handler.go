package api

import (
	"log/slog"
	"net/http"

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

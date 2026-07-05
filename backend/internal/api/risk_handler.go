package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

	z := 14
	x, y := mlit.LatLngToTile(lat, lng, z)

	c.JSON(http.StatusOK, h.riskSvc.UrbanRisks(c.Request.Context(), z, x, y))
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

	z := 14
	x, y := mlit.LatLngToTile(lat, lng, z)

	c.JSON(http.StatusOK, h.riskSvc.HazardRisks(c.Request.Context(), z, x, y))
}

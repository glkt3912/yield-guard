package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetRentStats は XIT001（賃貸）からエリア賃料相場を返す
// @Summary     賃料統計
// @Tags        land-prices
// @Produce     json
// @Param       area          query  string  true  "都道府県コード (例: 13)"
// @Param       municipality  query  string  false "市区町村コード"
// @Param       area_sqm      query  number  false "専有面積 (m²)"
// @Success     200  {object}  domain.RentStatsResult
// @Failure     400  {object}  map[string]string
// @Router      /api/rent-stats [get]
func (h *Handler) GetRentStats(c *gin.Context) {
	area := c.Query("area")
	if area == "" {
		badRequest(c, "area は必須パラメータです")
		return
	}

	municipality := c.Query("municipality")

	areaSqm := 0.0
	if s := c.Query("area_sqm"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v <= 0 {
			badRequest(c, "area_sqm は正の数値で指定してください")
			return
		}
		areaSqm = v
	}

	// データなし・取得失敗時は nil → JSON null（フロントはhide）
	c.JSON(http.StatusOK, h.rentSvc.Stats(c.Request.Context(), area, municipality, areaSqm))
}

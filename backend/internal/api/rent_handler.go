package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/mlit"
)

// GetRentStats は XIT001（賃貸）からエリア賃料相場を返す
// GET /api/rent-stats?area=13[&municipality=13101][&area_sqm=60]
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

	// 直近2年分のデータを取得する
	now := time.Now()
	toYear := now.Year()
	fromYear := toYear - 2
	currentQuarter := (int(now.Month())-1)/3 + 1
	toQuarter := currentQuarter - 1
	if toQuarter == 0 {
		toQuarter = 4
		toYear--
	}

	q := mlit.LandPriceQuery{
		Area:      area,
		City:      municipality,
		Year:      fromYear,
		Quarter:   1,
		ToYear:    toYear,
		ToQuarter: toQuarter,
	}

	result, err := h.mlitClient.FetchRentStats(c.Request.Context(), q, areaSqm)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "FetchRentStats failed", "err", err, "area", area)
		// データなしはサイレントに空レスポンスを返す（フロントはhide）
		c.JSON(http.StatusOK, nil)
		return
	}

	if result.Count == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}

	c.JSON(http.StatusOK, result)
}

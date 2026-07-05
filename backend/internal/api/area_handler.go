package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseBudgetYield は budget / yield クエリパラメータを解釈する。
// 未指定・不正値はデフォルト（budget=0, targetYield=0.08）に落とす。
func parseBudgetYield(c *gin.Context) (budget, targetYield float64) {
	budget = 0.0
	targetYield = 0.08
	if s := c.Query("budget"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			budget = v
		}
	}
	if s := c.Query("yield"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			targetYield = v
		}
	}
	return budget, targetYield
}

// HandleAreaDiscovery は都道府県内の市区町村を土地価格データで評価しランキング返却
// @Summary     投資エリア探索
// @Tags        area
// @Produce     json
// @Param       prefecture  query  string  true   "都道府県コード (例: 13)"
// @Param       budget      query  number  false  "予算 (円)"
// @Param       yield       query  number  false  "目標利回り (例: 0.07)"
// @Success     200  {object}  domain.AreaDiscoveryResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /api/area-discovery [get]
func (h *Handler) HandleAreaDiscovery(c *gin.Context) {
	prefecture := c.Query("prefecture")
	if prefecture == "" {
		badRequest(c, "prefecture は必須パラメータです")
		return
	}
	budget, targetYield := parseBudgetYield(c)

	resp, err := h.areaSvc.Discover(c.Request.Context(), prefecture, budget, targetYield)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "市区町村一覧の取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// HandleAreaSummary はエリア選択時に投資難易度を AI が 2 文で要約して返す
// @Summary     エリア AI サマリー
// @Tags        area
// @Produce     json
// @Param       area          query  string   true   "都道府県コード (例: 13)"
// @Param       municipality  query  string   true   "市区町村コード (例: 13101)"
// @Param       budget        query  number   false  "予算 (円, 例: 50000000)"
// @Param       yield         query  number   false  "目標利回り (小数, 例: 0.08)"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /api/area-discovery/summary [get]
func (h *Handler) HandleAreaSummary(c *gin.Context) {
	area := c.Query("area")
	municipality := c.Query("municipality")
	if area == "" || municipality == "" {
		badRequest(c, "area と municipality は必須パラメータです")
		return
	}
	budget, targetYield := parseBudgetYield(c)

	summary := h.areaSvc.Summarize(c.Request.Context(), area, municipality, budget, targetYield)
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

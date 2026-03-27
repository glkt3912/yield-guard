package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

type Handler struct {
	mlitClient *mlit.Client
}

func NewHandler(mlitClient *mlit.Client) *Handler {
	return &Handler{mlitClient: mlitClient}
}

// GetLandPrices は国交省APIから土地取引価格を取得して統計を返す
// GET /api/land-prices?area=10&city=10201&from=20231&to=20234
func (h *Handler) GetLandPrices(c *gin.Context) {
	area := c.Query("area")
	city := c.Query("city")
	from := c.Query("from")
	to := c.Query("to")

	if area == "" || from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "area, from, to は必須パラメータです"})
		return
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), mlit.LandPriceQuery{
		Area: area,
		City: city,
		From: from,
		To:   to,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "国交省APIからのデータ取得に失敗しました: " + err.Error()})
		return
	}

	stats := domain.CalcLandPriceStats(transactions)
	c.JSON(http.StatusOK, stats)
}

// CompareLandPrice は検討中の土地価格と相場を比較する
// GET /api/land-prices/compare?area=10&city=10201&from=20231&to=20234&price=5000000&area_sqm=100
func (h *Handler) CompareLandPrice(c *gin.Context) {
	area := c.Query("area")
	city := c.Query("city")
	from := c.Query("from")
	to := c.Query("to")
	priceStr := c.Query("price")
	areaSqmStr := c.Query("area_sqm")

	if area == "" || from == "" || to == "" || priceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "area, from, to, price は必須パラメータです"})
		return
	}

	landPrice, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price の値が不正です"})
		return
	}

	areaSqm := 0.0
	if areaSqmStr != "" {
		areaSqm, _ = strconv.ParseFloat(areaSqmStr, 64)
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), mlit.LandPriceQuery{
		Area: area,
		City: city,
		From: from,
		To:   to,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "国交省APIからのデータ取得に失敗しました: " + err.Error()})
		return
	}

	stats := domain.CalcLandPriceStats(transactions)
	comparison := domain.CompareLandPrice(stats, landPrice, areaSqm)
	c.JSON(http.StatusOK, comparison)
}

// Analyze は投資シミュレーションを実行する
// POST /api/analyze
func (h *Handler) Analyze(c *gin.Context) {
	var input domain.InvestmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が不正です: " + err.Error()})
		return
	}

	if input.LandPrice <= 0 || input.BuildingCost <= 0 || input.MonthlyRent <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "landPrice, buildingCost, monthlyRent は0より大きい値を指定してください"})
		return
	}

	result := domain.Analyze(input)
	c.JSON(http.StatusOK, result)
}

// GetPrefectures は都道府県一覧を返す
// GET /api/prefectures
func (h *Handler) GetPrefectures(c *gin.Context) {
	type Prefecture struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	prefectures := make([]Prefecture, 0, len(mlit.Prefectures))
	for code, name := range mlit.Prefectures {
		prefectures = append(prefectures, Prefecture{Code: code, Name: name})
	}
	c.JSON(http.StatusOK, prefectures)
}

// HealthCheck はサーバーの生存確認
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

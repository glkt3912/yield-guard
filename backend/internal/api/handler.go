package api

import (
	"errors"
	"net/http"
	"sort"
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
		Area: area, City: city, From: from, To: to,
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
	if err != nil || landPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price は正の数値で指定してください"})
		return
	}

	areaSqm := 0.0
	if areaSqmStr != "" {
		areaSqm, _ = strconv.ParseFloat(areaSqmStr, 64)
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), mlit.LandPriceQuery{
		Area: area, City: city, From: from, To: to,
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

	if err := validateInvestmentInput(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := domain.Analyze(input)
	c.JSON(http.StatusOK, result)
}

// validateInvestmentInput は投資入力値の範囲チェックを行う
func validateInvestmentInput(in domain.InvestmentInput) error {
	if in.LandPrice <= 0 || in.LandPrice > 10_000_000_000 {
		return errors.New("landPrice は 1〜100億円の範囲で指定してください")
	}
	if in.BuildingCost <= 0 || in.BuildingCost > 10_000_000_000 {
		return errors.New("buildingCost は 1〜100億円の範囲で指定してください")
	}
	if in.MonthlyRent <= 0 {
		return errors.New("monthlyRent は正の値を指定してください")
	}
	if in.VacancyRate < 0 || in.VacancyRate >= 1.0 {
		return errors.New("vacancyRate は 0.0〜0.99 の範囲で指定してください")
	}
	if in.LoanAmount < 0 {
		return errors.New("loanAmount は 0 以上を指定してください")
	}
	if in.AnnualLoanRate < 0 || in.AnnualLoanRate > 0.3 {
		return errors.New("annualLoanRate は 0〜30% の範囲で指定してください")
	}
	if in.LoanYears < 0 || in.LoanYears > 50 {
		return errors.New("loanYears は 0〜50 年の範囲で指定してください")
	}
	if in.MiscExpenseRate < 0 || in.MiscExpenseRate > 0.5 {
		return errors.New("miscExpenseRate は 0〜50% の範囲で指定してください")
	}
	if in.ExpenseRate < 0 || in.ExpenseRate > 0.9 {
		return errors.New("expenseRate は 0〜90% の範囲で指定してください")
	}
	if in.IncomeTaxRate < 0 || in.IncomeTaxRate > 0.6 {
		return errors.New("incomeTaxRate は 0〜60% の範囲で指定してください")
	}
	if in.ExitYieldTarget <= 0 || in.ExitYieldTarget > 0.5 {
		return errors.New("exitYieldTarget は 0%超〜50% の範囲で指定してください（ゼロ除算防止）")
	}
	if in.HoldingYears < 0 || in.HoldingYears > 50 {
		return errors.New("holdingYears は 0〜50 年の範囲で指定してください")
	}
	return nil
}

// GetPrefectures は都道府県一覧をコード順で返す
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
	// マップのイテレーション順は非決定的なので数値コード順にソート
	sort.Slice(prefectures, func(i, j int) bool {
		ci, _ := strconv.Atoi(prefectures[i].Code)
		cj, _ := strconv.Atoi(prefectures[j].Code)
		return ci < cj
	})
	c.JSON(http.StatusOK, prefectures)
}

// HealthCheck はサーバーの生存確認
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

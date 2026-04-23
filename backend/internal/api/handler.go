package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
	"github.com/yield-guard/backend/internal/telemetry"
)

// MLITClient は国交省APIクライアントのインターフェース（テスト時にモック注入可能）
type MLITClient interface {
	FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
	FetchMunicipalities(ctx context.Context, area string) ([]mlit.Municipality, error)
	FetchStationRidership(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error)
	FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
	FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
	FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	FetchUrbanRoad(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error)
	FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
}

type Handler struct {
	mlitClient MLITClient
}

func NewHandler(mlitClient MLITClient) *Handler {
	return &Handler{mlitClient: mlitClient}
}

// GetLandPrices は国交省APIから土地取引価格を取得して統計を返す
// GET /api/land-prices?area=10&city=10201&year=2024&quarter=1&to_year=2024&to_quarter=4
func (h *Handler) GetLandPrices(c *gin.Context) {
	q, err := parseLandPriceQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "国交省APIからのデータ取得に失敗しました: " + err.Error()})
		return
	}

	stats := domain.CalcLandPriceStats(transactions)
	c.JSON(http.StatusOK, stats)
}

// CompareLandPrice は検討中の土地価格と相場を比較する
// GET /api/land-prices/compare?area=10&city=10201&year=2024&quarter=1&to_year=2024&to_quarter=4&price=5000000&area_sqm=100
func (h *Handler) CompareLandPrice(c *gin.Context) {
	q, err := parseLandPriceQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priceStr := c.Query("price")
	areaSqmStr := c.Query("area_sqm")

	if priceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price は必須パラメータです"})
		return
	}

	landPrice, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || landPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price は正の数値で指定してください"})
		return
	}

	areaSqm := 0.0
	if areaSqmStr != "" {
		areaSqm, err = strconv.ParseFloat(areaSqmStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "area_sqm は数値で指定してください"})
			return
		}
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "国交省APIからのデータ取得に失敗しました: " + err.Error()})
		return
	}

	stats := domain.CalcLandPriceStats(transactions)
	comparison := domain.CompareLandPrice(stats, landPrice, areaSqm)
	c.JSON(http.StatusOK, comparison)
}

// parseLandPriceQuery はリクエストから LandPriceQuery を組み立てる
func parseLandPriceQuery(c *gin.Context) (mlit.LandPriceQuery, error) {
	area := c.Query("area")
	if area == "" {
		return mlit.LandPriceQuery{}, errors.New("area は必須パラメータです")
	}

	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2005 {
		return mlit.LandPriceQuery{}, errors.New("year は2005以降の整数で指定してください")
	}
	quarter, err := strconv.Atoi(c.Query("quarter"))
	if err != nil || quarter < 1 || quarter > 4 {
		return mlit.LandPriceQuery{}, errors.New("quarter は 1〜4 で指定してください")
	}
	toYear, err := strconv.Atoi(c.Query("to_year"))
	if err != nil || toYear < 2005 {
		return mlit.LandPriceQuery{}, errors.New("to_year は2005以降の整数で指定してください")
	}
	toQuarter, err := strconv.Atoi(c.Query("to_quarter"))
	if err != nil || toQuarter < 1 || toQuarter > 4 {
		return mlit.LandPriceQuery{}, errors.New("to_quarter は 1〜4 で指定してください")
	}

	return mlit.LandPriceQuery{
		Area:      area,
		City:      c.Query("city"),
		Year:      year,
		Quarter:   quarter,
		ToYear:    toYear,
		ToQuarter: toQuarter,
	}, nil
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

	input.Defaults()
	if err := input.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := domain.Analyze(input)
	telemetry.AnalyzeRequestsTotal.Add(c.Request.Context(), 1)
	c.JSON(http.StatusOK, result)
}

// MonteCarlo はモンテカルロシミュレーションを実行する
// POST /api/investment/simulate
func (h *Handler) MonteCarlo(c *gin.Context) {
	var input domain.MonteCarloInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が不正です: " + err.Error()})
		return
	}
	if err := validateInvestmentInput(input.Base); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Base.Defaults()
	if err := input.Base.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := domain.MonteCarloSimulate(input)
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
	if in.RentDeclineRate < 0 || in.RentDeclineRate > 0.2 {
		return errors.New("rentDeclineRate は 0.0〜0.2 の範囲で指定してください")
	}
	// DiscountRate == 0 は「未指定」扱い: Defaults() で 0.05 に補完される
	if in.DiscountRate < 0 || in.DiscountRate > 0.30 {
		return errors.New("discountRate は 0〜30% の範囲で指定してください")
	}
	if in.PriceDeclineRate < 0 || in.PriceDeclineRate > 0.10 {
		return errors.New("priceDeclineRate は 0〜10% の範囲で指定してください")
	}
	if in.DepreciationMethod != "" &&
		in.DepreciationMethod != domain.DepreciationMethodStraightLine &&
		in.DepreciationMethod != domain.DepreciationMethodDecliningBalance {
		return errors.New("depreciationMethod は \"straight-line\" または \"declining-balance\" を指定してください")
	}
	return nil
}

// EstimateLandPrice は築年数・駅距離補正による理論価格と乖離率を返す
// GET /api/land-prices/estimate?area=10&city=...&price=5000000&area_sqm=100&building_age=10&station_minutes=5
func (h *Handler) EstimateLandPrice(c *gin.Context) {
	q, err := parseLandPriceQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priceStr := c.Query("price")
	areaSqmStr := c.Query("area_sqm")
	buildingAgeStr := c.Query("building_age")

	if priceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price は必須パラメータです"})
		return
	}
	listingPrice, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || listingPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price は正の数値で指定してください"})
		return
	}

	areaSqm := 0.0
	if areaSqmStr != "" {
		areaSqm, err = strconv.ParseFloat(areaSqmStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "area_sqm は数値で指定してください"})
			return
		}
	}
	if areaSqm <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "area_sqm は正の数値で指定してください"})
		return
	}

	buildingAge := 0
	if buildingAgeStr != "" {
		buildingAge, err = strconv.Atoi(buildingAgeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "building_age は整数で指定してください"})
			return
		}
	}

	stationMinutes := 0
	if sm := c.Query("station_minutes"); sm != "" {
		stationMinutes, err = strconv.Atoi(sm)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "station_minutes は整数で指定してください"})
			return
		}
	}

	var ridershipScore domain.RidershipDemandScore
	if raw := c.Query("ridership_score"); raw != "" {
		score := domain.RidershipDemandScore(raw)
		if !score.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ridership_score は A〜E で指定してください"})
			return
		}
		ridershipScore = score
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "国交省APIからのデータ取得に失敗しました: " + err.Error()})
		return
	}

	stats := domain.CalcLandPriceStats(transactions)
	result, ok := domain.EstimateTheoreticalPrice(stats, domain.TheoreticalPriceInput{
		ListingPrice:   listingPrice,
		LandArea:       areaSqm,
		BuildingAge:    buildingAge,
		StationMinutes: stationMinutes,
		RidershipScore: ridershipScore,
	})
	if !ok {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "理論価格の推定に必要なデータが不足しています（取引事例に建築年データがありません）"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetStationRidership は物件の緯度経度からタイル座標を計算し、駅別乗降客数と需要スコアを返す（XKT015）
// GET /api/station-ridership?lat=35.6762&lng=139.6503[&z=14]
func (h *Handler) GetStationRidership(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat と lng は必須パラメータです"})
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat は -90〜90 の数値で指定してください"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng は -180〜180 の数値で指定してください"})
		return
	}

	z := 14
	if zStr := c.Query("z"); zStr != "" {
		zv, err := strconv.Atoi(zStr)
		if err != nil || zv < 11 || zv > 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "z は 11〜15 の整数で指定してください"})
			return
		}
		z = zv
	}

	tx, ty := mlit.LatLngToTile(lat, lng, z)

	stations, err := h.mlitClient.FetchStationRidership(c.Request.Context(), z, tx, ty)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "駅別乗降客数の取得に失敗しました: " + err.Error()})
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

// GetPopulationForecast は物件の緯度経度からタイル座標を計算し、将来推計人口と人口減少シナリオを返す（XKT013）
// GET /api/population-forecast?lat=35.6762&lng=139.6503[&z=14]
func (h *Handler) GetPopulationForecast(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat と lng は必須パラメータです"})
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat は -90〜90 の数値で指定してください"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng は -180〜180 の数値で指定してください"})
		return
	}

	z := 14
	if zStr := c.Query("z"); zStr != "" {
		zv, err := strconv.Atoi(zStr)
		if err != nil || zv < 11 || zv > 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "z は 11〜15 の整数で指定してください"})
			return
		}
		z = zv
	}

	tx, ty := mlit.LatLngToTile(lat, lng, z)

	items, err := h.mlitClient.FetchPopulationForecast(c.Request.Context(), z, tx, ty)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "将来推計人口の取得に失敗しました: " + err.Error()})
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"snapshots": []struct{}{}, "changeRate30yr": 0, "vacancyRateDelta": 0, "trend": ""})
		return
	}

	result := domain.CalcPopulationForecast(items)
	c.JSON(http.StatusOK, result)
}

// GetMunicipalities は指定都道府県の市区町村一覧を返す（XIT002）
// GET /api/municipalities?area=10
func (h *Handler) GetMunicipalities(c *gin.Context) {
	area := c.Query("area")
	if area == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "area は必須パラメータです"})
		return
	}

	municipalities, err := h.mlitClient.FetchMunicipalities(c.Request.Context(), area)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "市区町村一覧の取得に失敗しました: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, municipalities)
}

// GetLandAppraisals は XCT001 から地価公示情報を取得して比較統計を返す
// GET /api/land-appraisals?area=13&year=2024[&city=13101][&division=00]
// division: 00=住宅地(デフォルト), 05=商業地, 07=準工業地, 09=工業地
func (h *Handler) GetLandAppraisals(c *gin.Context) {
	area := c.Query("area")
	if area == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "area は必須パラメータです"})
		return
	}

	yearStr := c.Query("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2022 || year > 2030 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year は2022〜2030の整数で指定してください"})
		return
	}

	city := c.Query("city")
	division := c.DefaultQuery("division", "00")
	validDivisions := map[string]bool{"00": true, "05": true, "07": true, "09": true}
	if !validDivisions[division] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "division は 00/05/07/09 のいずれかを指定してください"})
		return
	}

	items, err := h.mlitClient.FetchLandAppraisals(c.Request.Context(), area, city, year, division)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "地価公示APIからのデータ取得に失敗しました: " + err.Error()})
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "指定エリアの地価公示データが見つかりませんでした"})
		return
	}

	result := domain.CalcAppraisalComparison(items)
	c.JSON(http.StatusOK, result)
}

// GetUrbanRisks は緯度経度から都市計画リスクを一括取得する
// GET /api/urban-risks?lat=35.68&lng=139.69
func (h *Handler) GetUrbanRisks(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat と lng は必須パラメータです"})
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < 20 || lat > 46 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat は日本国内の緯度（20〜46）で指定してください"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < 122 || lng > 154 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng は日本国内の経度（122〜154）で指定してください"})
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
	type apiResult[T any] struct {
		data []T
		err  error
	}
	locCh := make(chan apiResult[domain.LocationOptimizationItem], 1)
	embCh := make(chan apiResult[domain.EmbankmentItem], 1)
	rdCh := make(chan apiResult[domain.UrbanRoadItem], 1)
	disCh := make(chan apiResult[domain.DisasterHistoryItem], 1)

	go func() {
		d, e := h.mlitClient.FetchLocationOptimization(ctx, z, x, y)
		locCh <- apiResult[domain.LocationOptimizationItem]{d, e}
	}()
	go func() {
		d, e := h.mlitClient.FetchEmbankment(ctx, z, x, y)
		embCh <- apiResult[domain.EmbankmentItem]{d, e}
	}()
	go func() {
		d, e := h.mlitClient.FetchUrbanRoad(ctx, z, x, y)
		rdCh <- apiResult[domain.UrbanRoadItem]{d, e}
	}()
	go func() {
		d, e := h.mlitClient.FetchDisasterHistory(ctx, z, x, y)
		disCh <- apiResult[domain.DisasterHistoryItem]{d, e}
	}()

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

// HealthCheck はサーバーの生存確認
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

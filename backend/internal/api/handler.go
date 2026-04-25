package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

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
	FetchUrbanZoning(ctx context.Context, z, x, y int) ([]domain.UrbanZoningItem, error)
	FetchLiquefaction(ctx context.Context, z, x, y int) ([]domain.LiquefactionRiskItem, error)
	FetchFloodHazard(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error)
	FetchStormHazard(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error)
	FetchTsunamiHazard(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error)
	FetchLandslideHazard(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error)
}

type Handler struct {
	mlitClient    MLITClient
	geocodeClient GeocodeClient
}

func NewHandler(mlitClient MLITClient, geocodeClient GeocodeClient) *Handler {
	return &Handler{mlitClient: mlitClient, geocodeClient: geocodeClient}
}

// GetLandPrices は国交省APIから土地取引価格を取得して統計を返す
// GET /api/land-prices?area=10&city=10201&year=2024&quarter=1&to_year=2024&to_quarter=4
func (h *Handler) GetLandPrices(c *gin.Context) {
	q, err := parseLandPriceQuery(c)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), q)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchLandPrices failed", "err", err)
		badGateway(c, "国交省APIからのデータ取得に失敗しました")
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
		badRequest(c, err.Error())
		return
	}

	priceStr := c.Query("price")
	areaSqmStr := c.Query("area_sqm")

	if priceStr == "" {
		badRequest(c, "price は必須パラメータです")
		return
	}

	landPrice, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || landPrice <= 0 {
		badRequest(c, "price は正の数値で指定してください")
		return
	}

	areaSqm := 0.0
	if areaSqmStr != "" {
		areaSqm, err = strconv.ParseFloat(areaSqmStr, 64)
		if err != nil {
			badRequest(c, "area_sqm は数値で指定してください")
			return
		}
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), q)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchLandPrices failed", "err", err)
		badGateway(c, "国交省APIからのデータ取得に失敗しました")
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
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}

	if err := validateInvestmentInput(input); err != nil {
		badRequest(c, err.Error())
		return
	}

	input.Defaults()
	if err := input.Validate(); err != nil {
		badRequest(c, err.Error())
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
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}
	if err := validateInvestmentInput(input.Base); err != nil {
		badRequest(c, err.Error())
		return
	}
	input.Base.Defaults()
	if err := input.Base.Validate(); err != nil {
		badRequest(c, err.Error())
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
		badRequest(c, err.Error())
		return
	}

	priceStr := c.Query("price")
	areaSqmStr := c.Query("area_sqm")
	buildingAgeStr := c.Query("building_age")

	if priceStr == "" {
		badRequest(c, "price は必須パラメータです")
		return
	}
	listingPrice, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || listingPrice <= 0 {
		badRequest(c, "price は正の数値で指定してください")
		return
	}

	areaSqm := 0.0
	if areaSqmStr != "" {
		areaSqm, err = strconv.ParseFloat(areaSqmStr, 64)
		if err != nil {
			badRequest(c, "area_sqm は数値で指定してください")
			return
		}
	}
	if areaSqm <= 0 {
		badRequest(c, "area_sqm は正の数値で指定してください")
		return
	}

	buildingAge := 0
	if buildingAgeStr != "" {
		buildingAge, err = strconv.Atoi(buildingAgeStr)
		if err != nil {
			badRequest(c, "building_age は整数で指定してください")
			return
		}
	}

	stationMinutes := 0
	if sm := c.Query("station_minutes"); sm != "" {
		stationMinutes, err = strconv.Atoi(sm)
		if err != nil {
			badRequest(c, "station_minutes は整数で指定してください")
			return
		}
	}

	var ridershipScore domain.RidershipDemandScore
	if raw := c.Query("ridership_score"); raw != "" {
		score := domain.RidershipDemandScore(raw)
		if !score.IsValid() {
			badRequest(c, "ridership_score は A〜E で指定してください")
			return
		}
		ridershipScore = score
	}

	transactions, err := h.mlitClient.FetchLandPrices(c.Request.Context(), q)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchLandPrices failed", "err", err)
		badGateway(c, "国交省APIからのデータ取得に失敗しました")
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

// GetPopulationForecast は物件の緯度経度からタイル座標を計算し、将来推計人口と人口減少シナリオを返す（XKT013）
// GET /api/population-forecast?lat=35.6762&lng=139.6503[&z=14]
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

// GetMunicipalities は指定都道府県の市区町村一覧を返す（XIT002）
// GET /api/municipalities?area=10
func (h *Handler) GetMunicipalities(c *gin.Context) {
	area := c.Query("area")
	if area == "" {
		badRequest(c, "area は必須パラメータです")
		return
	}

	municipalities, err := h.mlitClient.FetchMunicipalities(c.Request.Context(), area)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchMunicipalities failed", "err", err)
		badGateway(c, "市区町村一覧の取得に失敗しました")
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
		badRequest(c, "area は必須パラメータです")
		return
	}

	yearStr := c.Query("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2022 || year > 2030 {
		badRequest(c, "year は2022〜2030の整数で指定してください")
		return
	}

	city := c.Query("city")
	division := c.DefaultQuery("division", "00")
	validDivisions := map[string]bool{"00": true, "05": true, "07": true, "09": true}
	if !validDivisions[division] {
		badRequest(c, "division は 00/05/07/09 のいずれかを指定してください")
		return
	}

	items, err := h.mlitClient.FetchLandAppraisals(c.Request.Context(), area, city, year, division)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "FetchLandAppraisals failed", "err", err)
		badGateway(c, "地価公示APIからのデータ取得に失敗しました")
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "指定エリアの地価公示データが見つかりませんでした"})
		return
	}

	result := domain.CalcAppraisalComparison(items)
	c.JSON(http.StatusOK, result)
}

// GetRentDeclineHint は XCT001 直近5年分から賃料下落率参考値を返す
// GET /api/investment/rent-decline-hint?area=13[&municipality=13101]
func (h *Handler) GetRentDeclineHint(c *gin.Context) {
	area := c.Query("area")
	if area == "" {
		badRequest(c, "area は必須パラメータです")
		return
	}

	municipality := c.Query("municipality")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// XCT001 の対応年（2022〜2026）を並列取得
	years := []int{2022, 2023, 2024, 2025, 2026}
	type yearResult struct {
		year  int
		items []domain.LandAppraisalItem
		err   error
	}
	ch := make(chan yearResult, len(years))
	for _, y := range years {
		go func() {
			items, err := h.mlitClient.FetchLandAppraisals(ctx, area, municipality, y, "00")
			ch <- yearResult{year: y, items: items, err: err}
		}()
	}

	itemsByYear := make(map[int][]domain.LandAppraisalItem, len(years))
	var fetchErr error
	for range years {
		r := <-ch
		if r.err != nil {
			slog.WarnContext(ctx, "FetchLandAppraisals failed", "year", r.year, "area", area, "error", r.err)
			fetchErr = r.err
			continue
		}
		if len(r.items) > 0 {
			itemsByYear[r.year] = r.items
		}
	}

	// 全年エラーの場合のみ502を返す
	if len(itemsByYear) == 0 && fetchErr != nil {
		badGateway(c, "地価公示APIからのデータ取得に失敗しました")
		return
	}

	hint := domain.CalcRentDeclineHint(itemsByYear)
	c.JSON(http.StatusOK, hint)
}

// GetUrbanRisks は緯度経度から都市計画リスクを一括取得する
// GET /api/urban-risks?lat=35.68&lng=139.69
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

// GetHazardInfo は物件の緯度経度から洪水・高潮・津波・土砂災害のハザード情報を返す。
// GET /api/hazard?lat=35.6895&lng=139.6917
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

// calcScoreForTile は指定タイル座標に対して複数 API を並列取得し投資適地スコアを返す。
// 個別 API の失敗は警告ログのみでスキップする。コンテキストキャンセル時はエラーを返す。
func (h *Handler) calcScoreForTile(ctx context.Context, z, x, y int) (domain.InvestmentScoreResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.InvestmentScoreResult{}, err
	}
	popCh := fanOut(func() ([]domain.PopulationForecastItem, error) { return h.mlitClient.FetchPopulationForecast(ctx, z, x, y) })
	ridCh := fanOut(func() ([]mlit.StationRidership, error) { return h.mlitClient.FetchStationRidership(ctx, z, x, y) })
	locCh := fanOut(func() ([]domain.LocationOptimizationItem, error) { return h.mlitClient.FetchLocationOptimization(ctx, z, x, y) })
	embCh := fanOut(func() ([]domain.EmbankmentItem, error) { return h.mlitClient.FetchEmbankment(ctx, z, x, y) })
	disCh := fanOut(func() ([]domain.DisasterHistoryItem, error) { return h.mlitClient.FetchDisasterHistory(ctx, z, x, y) })
	zonCh := fanOut(func() ([]domain.UrbanZoningItem, error) { return h.mlitClient.FetchUrbanZoning(ctx, z, x, y) })
	liqCh := fanOut(func() ([]domain.LiquefactionRiskItem, error) { return h.mlitClient.FetchLiquefaction(ctx, z, x, y) })
	floCh := fanOut(func() ([]domain.FloodHazardItem, error) { return h.mlitClient.FetchFloodHazard(ctx, z, x, y) })
	stoCh := fanOut(func() ([]domain.StormHazardItem, error) { return h.mlitClient.FetchStormHazard(ctx, z, x, y) })
	tsuCh := fanOut(func() ([]domain.TsunamiHazardItem, error) { return h.mlitClient.FetchTsunamiHazard(ctx, z, x, y) })
	lanCh := fanOut(func() ([]domain.LandslideHazardItem, error) { return h.mlitClient.FetchLandslideHazard(ctx, z, x, y) })

	// 地価トレンド: タイル中心座標→都道府県コードを逆引きし、新旧2期間を並列取得する
	centerLat, centerLng := mlit.TileToLatLng(x, y, z)
	prefCode := domain.PrefCodeFromLatLng(centerLat, centerLng)
	type landResult struct {
		stats domain.LandPriceStats
		err   error
	}
	recentLandCh := make(chan landResult, 1)
	oldLandCh := make(chan landResult, 1)
	if prefCode != "" {
		go func() {
			tx, e := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: 2023, Quarter: 1, ToYear: 2024, ToQuarter: 4})
			recentLandCh <- landResult{domain.CalcLandPriceStats(tx), e}
		}()
		go func() {
			tx, e := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{Area: prefCode, Year: 2021, Quarter: 1, ToYear: 2022, ToQuarter: 4})
			oldLandCh <- landResult{domain.CalcLandPriceStats(tx), e}
		}()
	} else {
		recentLandCh <- landResult{}
		oldLandCh <- landResult{}
	}

	input := domain.InvestmentScoreInput{}

	if r := <-popCh; r.err != nil {
		slog.WarnContext(ctx, "FetchPopulationForecast failed", "error", r.err)
	} else {
		input.PopulationItems = r.data
	}
	if r := <-ridCh; r.err != nil {
		slog.WarnContext(ctx, "FetchStationRidership failed", "error", r.err)
	} else {
		input.StationRiderships = make([]domain.StationRidershipResult, 0, len(r.data))
		for _, s := range r.data {
			score := domain.CalcRidershipDemandScore(s.Passengers)
			input.StationRiderships = append(input.StationRiderships, domain.StationRidershipResult{
				StationName: s.StationName,
				LineName:    s.LineName,
				Passengers:  s.Passengers,
				DemandScore: score,
				Correction:  domain.RidershipCorrectionFactor(score),
			})
		}
	}
	if r := <-locCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLocationOptimization failed", "error", r.err)
	} else {
		input.LocationItems = r.data
	}
	if r := <-embCh; r.err != nil {
		slog.WarnContext(ctx, "FetchEmbankment failed", "error", r.err)
	} else {
		input.EmbankmentItems = r.data
	}
	if r := <-disCh; r.err != nil {
		slog.WarnContext(ctx, "FetchDisasterHistory failed", "error", r.err)
	} else {
		input.DisasterItems = r.data
	}
	if r := <-zonCh; r.err != nil {
		slog.WarnContext(ctx, "FetchUrbanZoning failed", "error", r.err)
	} else {
		input.UrbanZoningItems = r.data
	}
	if r := <-liqCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLiquefaction failed", "error", r.err)
	} else {
		input.LiquefactionItems = r.data
	}
	if r := <-floCh; r.err != nil {
		slog.WarnContext(ctx, "FetchFloodHazard failed", "error", r.err)
	} else {
		input.FloodItems = r.data
	}
	if r := <-stoCh; r.err != nil {
		slog.WarnContext(ctx, "FetchStormHazard failed", "error", r.err)
	} else {
		input.StormItems = r.data
	}
	if r := <-tsuCh; r.err != nil {
		slog.WarnContext(ctx, "FetchTsunamiHazard failed", "error", r.err)
	} else {
		input.TsunamiItems = r.data
	}
	if r := <-lanCh; r.err != nil {
		slog.WarnContext(ctx, "FetchLandslideHazard failed", "error", r.err)
	} else {
		input.LandslideItems = r.data
	}

	recentLand := <-recentLandCh
	oldLand := <-oldLandCh
	if recentLand.err != nil || oldLand.err != nil {
		if recentLand.err != nil {
			slog.WarnContext(ctx, "FetchLandPrices (recent) failed", "error", recentLand.err)
		}
		if oldLand.err != nil {
			slog.WarnContext(ctx, "FetchLandPrices (old) failed", "error", oldLand.err)
		}
	} else if recentLand.stats.MedianTsubo > 0 && oldLand.stats.MedianTsubo > 0 {
		input.LandPriceChangeRate = (recentLand.stats.MedianTsubo - oldLand.stats.MedianTsubo) / oldLand.stats.MedianTsubo
		input.HasLandPriceTrend = true
	}

	return domain.CalcInvestmentScore(input), nil
}

// GetInvestmentScore は物件の緯度経度から複数 API を並列呼び出しし、投資適地スコアを算出して返す。
// GET /api/investment-score?lat=35.6762&lng=139.6503
func (h *Handler) GetInvestmentScore(c *gin.Context) {
	lat, lng, ok := parseLatLng(c, coordsJapanOnly)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	z := 14
	x, y := mlit.LatLngToTile(lat, lng, z)

	score, err := h.calcScoreForTile(ctx, z, x, y)
	if err != nil {
		slog.ErrorContext(ctx, "calcScoreForTile failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "投資スコアの計算に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, score)
}

const maxHeatmapTiles = 50

// GetInvestmentScoreHeatmap はバウンディングボックス内の全タイルに対して投資スコアを並列計算して返す。
// GET /api/investment-score-heatmap?minLat=35.6&maxLat=35.7&minLng=139.6&maxLng=139.7&z=13
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
						results <- tileResult{err: fmt.Errorf("panic in calcScoreForTile: %v", r)}
					}
				}()
				select {
				case <-ctx.Done():
					results <- tileResult{err: ctx.Err()}
					return
				case sem <- struct{}{}:
				}
				defer func() { <-sem }()
				score, err := h.calcScoreForTile(ctx, z, tx, ty)
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
			slog.WarnContext(ctx, "calcScoreForTile failed", "error", r.err)
			continue
		}
		tiles = append(tiles, r.tile)
	}

	c.JSON(http.StatusOK, domain.HeatmapResponse{
		Tiles:     tiles,
		TileCount: len(tiles),
	})
}

// HealthCheck はサーバーの生存確認
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func validateRenovationInput(in domain.RenovationInput) error {
	if in.PropertyPrice <= 0 {
		return errors.New("propertyPrice は正の値を指定してください")
	}
	if in.AnnualBaseRent < 0 {
		return errors.New("annualBaseRent は 0 以上を指定してください")
	}
	if in.EffectiveTaxRate < 0 || in.EffectiveTaxRate > 1 {
		return errors.New("effectiveTaxRate は 0.0〜1.0 の範囲で指定してください")
	}
	if in.SelfLaborRatePerHour < 0 {
		return errors.New("selfLaborRatePerHour は 0 以上を指定してください")
	}
	if len(in.Items) == 0 {
		return errors.New("items は 1 件以上指定してください")
	}
	for idx, item := range in.Items {
		if item.Cost <= 0 {
			return fmt.Errorf("items[%d].cost は正の値を指定してください", idx)
		}
	}
	return nil
}

// HandleAreaDiscovery は都道府県内の市区町村を土地価格データで評価しランキング返却
// GET /api/area-discovery?prefecture=13&budget=50000000&yield=0.07
func (h *Handler) HandleAreaDiscovery(c *gin.Context) {
	prefecture := c.Query("prefecture")
	if prefecture == "" {
		badRequest(c, "prefecture は必須パラメータです")
		return
	}
	budgetStr := c.Query("budget")
	yieldStr := c.Query("yield")

	budget := 0.0
	targetYield := 0.08
	if budgetStr != "" {
		if v, err := strconv.ParseFloat(budgetStr, 64); err == nil && v > 0 {
			budget = v
		}
	}
	if yieldStr != "" {
		if v, err := strconv.ParseFloat(yieldStr, 64); err == nil && v > 0 {
			targetYield = v
		}
	}

	ctx := c.Request.Context()

	// 市区町村一覧取得
	municipalities, err := h.mlitClient.FetchMunicipalities(ctx, prefecture)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "市区町村一覧の取得に失敗しました"})
		return
	}

	// 最新2年の土地取引データを並列取得
	now := time.Now()
	toYear := now.Year()
	fromYear := toYear - 2

	type result struct {
		item domain.AreaDiscoveryItem
		err  error
	}

	// 上位30市区町村に絞る（全件だとタイムアウトリスク）
	limit := 30
	if len(municipalities) < limit {
		limit = len(municipalities)
	}

	results := make([]result, limit)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // 並列5件上限

	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func(idx int, m mlit.Municipality) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			transactions, fetchErr := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{
				Area:      prefecture,
				City:      m.ID,
				Year:      fromYear,
				Quarter:   1,
				ToYear:    toYear,
				ToQuarter: 4,
			})

			item := domain.AreaDiscoveryItem{
				MunicipalityCode: m.ID,
				MunicipalityName: m.Name,
			}

			if fetchErr != nil || len(transactions) == 0 {
				item.DataSufficient = false
				item.LandPriceTrend = "不明"
				item.YieldDifficulty = "difficult"
				item.YieldDifficultyLabel = "データ不足"
				results[idx] = result{item: item}
				return
			}

			stats := domain.CalcLandPriceStats(transactions)

			item.MedianTsubo = stats.MedianTsubo
			item.TransactionCount = stats.Count
			item.DataSufficient = stats.Count >= 3

			// 利回り達成難易度: 予算（または中央坪単価×30坪+建物代）に対して目標利回りが必要とする月額家賃を試算し、
			// 1坪あたり月額賃料の現実性で判定する
			var totalCostEst float64
			if budget > 0 {
				totalCostEst = budget
			} else {
				totalCostEst = stats.MedianTsubo*30 + 10_000_000
			}
			annualRentNeeded := totalCostEst * targetYield

			// 1坪あたり月額賃料が現実的か判定（目安: 8,000円以下=達成可能, 15,000円超=困難）
			monthlyRentNeeded := annualRentNeeded / 12
			areaTsubo := 30.0
			rentPerTsubo := monthlyRentNeeded / areaTsubo
			if rentPerTsubo <= 8000 {
				item.YieldDifficulty = "achievable"
				item.YieldDifficultyLabel = "達成可能"
			} else if rentPerTsubo <= 15000 {
				item.YieldDifficulty = "slightly-difficult"
				item.YieldDifficultyLabel = "やや困難"
			} else {
				item.YieldDifficulty = "difficult"
				item.YieldDifficultyLabel = "困難"
			}

			item.LandPriceTrend = "データなし"
			results[idx] = result{item: item}
		}(i, municipalities[i])
	}
	wg.Wait()

	items := make([]domain.AreaDiscoveryItem, 0, limit)
	for _, r := range results {
		if r.err == nil {
			items = append(items, r.item)
		}
	}

	// 達成可能 → やや困難 → 困難 の順、同一難易度内は取引件数降順
	sort.Slice(items, func(i, j int) bool {
		difficultyOrder := map[string]int{"achievable": 0, "slightly-difficult": 1, "difficult": 2}
		di, dj := difficultyOrder[items[i].YieldDifficulty], difficultyOrder[items[j].YieldDifficulty]
		if di != dj {
			return di < dj
		}
		return items[i].TransactionCount > items[j].TransactionCount
	})

	c.JSON(http.StatusOK, domain.AreaDiscoveryResponse{
		Items:      items,
		Prefecture: prefecture,
	})
}

// HandleRenovationAnalyze はリフォームROIシミュレーションを実行する
func (h *Handler) HandleRenovationAnalyze(c *gin.Context) {
	var input domain.RenovationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}
	if err := validateRenovationInput(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := domain.CalcRenovationROI(input)
	c.JSON(http.StatusOK, result)
}

// GetGeocode は住所文字列から緯度・経度を返す
// GET /api/geocode?address=<住所>
// APIキーはサーバーサイドのみで保持し、フロントには露出しない
func (h *Handler) GetGeocode(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		badRequest(c, "address パラメータは必須です")
		return
	}

	// 住所はPIIになりうるため、accessLogMiddleware が記録するクエリ文字列をマスクする
	// nil ガードより先に実行することで、全パスでマスクが適用される
	masked := address
	if len([]rune(address)) > 10 {
		masked = string([]rune(address)[:10]) + "***"
	}
	c.Request.URL.RawQuery = "address=" + url.QueryEscape(masked)
	slog.InfoContext(c.Request.Context(), "geocode request", "address_masked", masked)

	if h.geocodeClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ジオコーディングが設定されていません"})
		return
	}

	result, err := h.geocodeClient.Geocode(c.Request.Context(), address)
	if err != nil {
		switch {
		case errors.Is(err, errGeocodeNotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ジオコーディングが設定されていません"})
		case errors.Is(err, errGeocodeNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": "住所が見つかりませんでした。丁目・番地まで入力してください"})
		default:
			slog.WarnContext(c.Request.Context(), "geocode upstream error", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "座標取得に失敗しました"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

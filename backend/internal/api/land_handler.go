package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

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

	stats := domain.CalcLandPriceStats(c.Request.Context(), transactions)
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

	stats := domain.CalcLandPriceStats(c.Request.Context(), transactions)
	comparison := domain.CompareLandPrice(stats, landPrice, areaSqm)
	c.JSON(http.StatusOK, comparison)
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

	stats := domain.CalcLandPriceStats(c.Request.Context(), transactions)
	result, ok := domain.EstimateTheoreticalPrice(c.Request.Context(), stats, domain.TheoreticalPriceInput{
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

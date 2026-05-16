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
// @Summary     土地取引価格統計
// @Tags        land-prices
// @Produce     json
// @Param       area        query  string   true  "都道府県コード (例: 13)"
// @Param       year        query  integer  true  "開始年 (2005以降)"
// @Param       quarter     query  integer  true  "開始四半期 (1-4)"
// @Param       to_year     query  integer  true  "終了年"
// @Param       to_quarter  query  integer  true  "終了四半期 (1-4)"
// @Param       city        query  string   false "市区町村コード"
// @Success     200  {object}  domain.LandPriceStats
// @Failure     400  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/land-prices/stats [get]
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
// @Summary     土地価格比較
// @Tags        land-prices
// @Produce     json
// @Param       area        query  string   true  "都道府県コード (例: 13)"
// @Param       year        query  integer  true  "開始年 (2005以降)"
// @Param       quarter     query  integer  true  "開始四半期 (1-4)"
// @Param       to_year     query  integer  true  "終了年"
// @Param       to_quarter  query  integer  true  "終了四半期 (1-4)"
// @Param       price       query  number   true  "検討物件価格 (円)"
// @Param       city        query  string   false "市区町村コード"
// @Param       area_sqm    query  number   false "土地面積 (m²)"
// @Success     200  {object}  domain.LandPriceComparison
// @Failure     400  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/land-prices/compare [get]
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
// @Summary     理論価格推定
// @Tags        land-prices
// @Produce     json
// @Param       area             query  string   true  "都道府県コード (例: 13)"
// @Param       year             query  integer  true  "開始年 (2005以降)"
// @Param       quarter          query  integer  true  "開始四半期 (1-4)"
// @Param       to_year          query  integer  true  "終了年"
// @Param       to_quarter       query  integer  true  "終了四半期 (1-4)"
// @Param       price            query  number   true  "物件価格 (円)"
// @Param       area_sqm         query  number   true  "土地面積 (m²)"
// @Param       city             query  string   false "市区町村コード"
// @Param       building_age     query  integer  false "築年数"
// @Param       station_minutes  query  integer  false "最寄り駅徒歩分"
// @Param       ridership_score  query  string   false "需要スコア (A-E)"
// @Success     200  {object}  domain.TheoreticalPriceResult
// @Failure     400  {object}  map[string]string
// @Failure     422  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/land-prices/estimate [get]
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
// @Summary     市区町村一覧
// @Tags        land-prices
// @Produce     json
// @Param       area  query  string  true  "都道府県コード (例: 13)"
// @Success     200  {array}   mlit.Municipality
// @Failure     400  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/municipalities [get]
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

// GetLandAppraisals は地価公示情報を取得して比較統計を返す
// @Summary     地価公示統計
// @Tags        land-prices
// @Produce     json
// @Param       area      query  string   true  "都道府県コード (例: 13)"
// @Param       year      query  integer  true  "公示年 (2022-2030)"
// @Param       city      query  string   false "市区町村コード"
// @Param       division  query  string   false "用途区分 (00=住宅地, 05=商業地, 07=準工業地, 09=工業地)"
// @Success     200  {object}  domain.AppraisalComparisonResult
// @Failure     400  {object}  map[string]string
// @Failure     422  {object}  map[string]string
// @Failure     502  {object}  map[string]string
// @Router      /api/land-appraisals [get]
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

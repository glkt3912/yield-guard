package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/ai"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
	"github.com/yield-guard/backend/internal/service"
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
	FetchRentStats(ctx context.Context, q mlit.LandPriceQuery, areaSqm float64) (domain.RentStatsResult, error)
}

type Handler struct {
	mlitClient    MLITClient
	geocodeClient GeocodeClient
	summarizer    ai.Summarizer
	locationSvc   service.LocationService
	areaSvc       service.AreaService
	riskSvc       service.RiskService
}

func NewHandler(mlitClient MLITClient, geocodeClient GeocodeClient, locationSvc service.LocationService, fsClient ...*firestore.Client) *Handler {
	var fs *firestore.Client
	if len(fsClient) > 0 {
		fs = fsClient[0]
	}
	summarizer := ai.NewSummarizer(fs)
	return &Handler{
		mlitClient:    mlitClient,
		geocodeClient: geocodeClient,
		summarizer:    summarizer,
		locationSvc:   locationSvc,
		areaSvc:       service.NewAreaDiscoveryService(mlitClient, summarizer),
		riskSvc:       service.NewRiskAssessmentService(mlitClient),
	}
}

// HealthCheck はサーバーの生存確認を行う
// @Summary     ヘルスチェック
// @Tags        system
// @Produce     json
// @Success     200  {object}  map[string]string
// @Router      /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetGeocode は住所文字列から緯度・経度を返す
// @Summary     ジオコーディング
// @Tags        location
// @Produce     json
// @Param       address  query  string  true  "住所文字列"
// @Success     200  {object}  api.GeocodeResult
// @Failure     400  {object}  map[string]string
// @Failure     503  {object}  map[string]string
// @Router      /api/geocode [get]
func (h *Handler) GetGeocode(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		badRequest(c, "address パラメータは必須です")
		return
	}

	// address は PII のため accessLogMiddleware の sanitizeQuery が [REDACTED] に置換する。
	// デバッグ用に先頭10文字のみ構造化ログに残す。
	masked := address
	if len([]rune(address)) > 10 {
		masked = string([]rune(address)[:10]) + "***"
	}
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

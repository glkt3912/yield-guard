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

// mlitFetcher は Handler が直接呼ぶ MLIT メソッドのサブセット（consumer = Handler）。
// station-ridership / population-forecast / municipalities の各エンドポイントで使う。
type mlitFetcher interface {
	FetchStationRidership(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error)
	FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
	FetchMunicipalities(ctx context.Context, area string) ([]mlit.Municipality, error)
}

// mlitProvider は NewHandler が全 service を組み立てるのに必要な合成インターフェース。
// 各要素は利用側（service / handler）が所有する小インターフェースで、fat interface を
// 再宣言せず埋め込みで束ねる。*mlit.Client（本番）とテストモックの両方が満たす。
type mlitProvider interface {
	mlitFetcher
	service.AreaMLITClient
	service.RiskMLITClient
	service.LandMLITClient
	service.RentMLITClient
	service.InvestmentMLITClient
}

type Handler struct {
	mlitClient    mlitFetcher
	geocodeClient GeocodeClient
	locationSvc   service.LocationService
	areaSvc       service.AreaService
	riskSvc       service.RiskService
	landSvc       service.LandPriceService
	rentSvc       service.RentService
	investmentSvc service.InvestmentService
}

func NewHandler(mlitClient mlitProvider, geocodeClient GeocodeClient, locationSvc service.LocationService, fsClient ...*firestore.Client) *Handler {
	var fs *firestore.Client
	if len(fsClient) > 0 {
		fs = fsClient[0]
	}
	summarizer := ai.NewSummarizer(fs)
	return &Handler{
		mlitClient:    mlitClient,
		geocodeClient: geocodeClient,
		locationSvc:   locationSvc,
		areaSvc:       service.NewAreaDiscoveryService(mlitClient, summarizer),
		riskSvc:       service.NewRiskAssessmentService(mlitClient),
		landSvc:       service.NewLandPriceAnalysisService(mlitClient),
		rentSvc:       service.NewRentStatsService(mlitClient),
		investmentSvc: service.NewInvestmentAnalysisService(mlitClient, summarizer),
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

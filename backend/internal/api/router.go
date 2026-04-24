package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/time/rate"
)

func internalKeyMiddleware() gin.HandlerFunc {
	key := os.Getenv("APP_INTERNAL_API_KEY")
	return func(c *gin.Context) {
		if key != "" && c.GetHeader("X-Internal-Key") != key {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func accessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// liveness probe のログノイズを抑制
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		latencyMS := time.Since(start).Milliseconds()
		ctx := c.Request.Context()
		args := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latencyMS,
			"client_ip", c.ClientIP(),
		}
		// NOTE: クエリ文字列を丸ごとログに記録する。現状のエンドポイントは
		// 都道府県コード・タイル座標等の公開情報のみだが、将来 PII を含む
		// パラメータが追加される場合はこのフィールドを除外すること。
		if q := c.Request.URL.RawQuery; q != "" {
			args = append(args, "query", q)
		}
		switch {
		case status >= 500:
			slog.ErrorContext(ctx, "access", args...)
		case status >= 400:
			slog.WarnContext(ctx, "access", args...)
		default:
			slog.InfoContext(ctx, "access", args...)
		}
	}
}

func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, err any) {
		slog.ErrorContext(c.Request.Context(), "panic recovered",
			"error", fmt.Sprintf("%v", err),
			"path", c.Request.URL.Path,
			"stack", string(debug.Stack()),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// NewRouter は Gin ルーターを初期化して返す
func NewRouter(h *Handler) *gin.Engine {
	r := gin.New()

	r.Use(otelgin.Middleware("yield-guard-backend"))
	r.Use(accessLogMiddleware())
	r.Use(recoveryMiddleware())

	// リクエストボディを 64KB に制限（大量 JSON による DoS を防止）
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		c.Next()
	})

	// CORS: 許可オリジンを環境変数 ALLOW_ORIGINS（カンマ区切り）から取得
	// 未設定の場合は localhost:3000 のみ許可
	allowOrigins := os.Getenv("ALLOW_ORIGINS")
	if allowOrigins == "" {
		allowOrigins = "http://localhost:3000"
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(allowOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Accept"},
		AllowCredentials: false,
	}))

	// IP ベースレートリミッター: 一般 API 60 req/min (1 token/s)、analyze 10 req/min (1 token/6s)
	generalRL := newRateLimiter(rate.Every(time.Second), 20)
	analyzeRL := newRateLimiter(rate.Every(6*time.Second), 5)

	r.GET("/health", h.HealthCheck)

	api := r.Group("/api")
	api.Use(internalKeyMiddleware())
	api.Use(generalRL.middleware())
	{
		api.GET("/land-prices/stats", h.GetLandPrices)
		api.GET("/land-prices/compare", h.CompareLandPrice)
		api.GET("/land-prices/estimate", h.EstimateLandPrice)
		// analyze / simulate は generalRL + analyzeRL の両方でトークンを消費する（意図的な二重制限）
		api.POST("/investment/analyze", analyzeRL.middleware(), h.Analyze)
		api.GET("/investment/rent-decline-hint", analyzeRL.middleware(), h.GetRentDeclineHint)
		api.POST("/renovation/analyze", h.HandleRenovationAnalyze)
		api.POST("/investment/simulate", analyzeRL.middleware(), h.MonteCarlo)
		api.GET("/municipalities", h.GetMunicipalities)
		api.GET("/station-ridership", h.GetStationRidership)
		api.GET("/population-forecast", h.GetPopulationForecast)
		api.GET("/land-appraisals", h.GetLandAppraisals)
		api.GET("/urban-risks", h.GetUrbanRisks)
		api.GET("/hazard", h.GetHazardInfo)
		api.GET("/investment-score", h.GetInvestmentScore)
		api.GET("/investment-score-heatmap", analyzeRL.middleware(), h.GetInvestmentScoreHeatmap)
		api.GET("/area-discovery", h.HandleAreaDiscovery)
	}

	return r
}

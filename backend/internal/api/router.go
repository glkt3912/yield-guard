package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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

	r.GET("/health", h.HealthCheck)

	api := r.Group("/api")
	api.Use(internalKeyMiddleware())
	{
		api.GET("/land-prices/stats", h.GetLandPrices)
		api.GET("/land-prices/compare", h.CompareLandPrice)
		api.GET("/land-prices/estimate", h.EstimateLandPrice)
		api.POST("/investment/analyze", h.Analyze)
		api.GET("/municipalities", h.GetMunicipalities)
		api.GET("/station-ridership", h.GetStationRidership)
		api.GET("/population-forecast", h.GetPopulationForecast)
		api.GET("/land-appraisals", h.GetLandAppraisals)
		api.GET("/urban-risks", h.GetUrbanRisks)
	}

	return r
}

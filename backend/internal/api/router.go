package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter は Gin ルーターを初期化して返す
func NewRouter(h *Handler) *gin.Engine {
	r := gin.Default()

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
	{
		api.GET("/land-prices", h.GetLandPrices)
		api.GET("/land-prices/compare", h.CompareLandPrice)
		api.POST("/analyze", h.Analyze)
	}

	return r
}

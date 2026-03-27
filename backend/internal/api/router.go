package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter は Gin ルーターを初期化して返す
func NewRouter(h *Handler) *gin.Engine {
	r := gin.Default()

	// CORS: フロントエンド (localhost:3000) からのアクセスを許可
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
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
		api.GET("/prefectures", h.GetPrefectures)
	}

	return r
}

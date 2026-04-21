package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yield-guard/backend/internal/api"
	"github.com/yield-guard/backend/internal/mlit"
	"github.com/yield-guard/backend/internal/telemetry"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if os.Getenv("MLIT_API_KEY") == "" {
		log.Fatal("MLIT_API_KEY is not set")
	}

	ctx := context.Background()
	otelShutdown, err := telemetry.Setup(ctx, "yield-guard-backend", "0.1.0")
	if err != nil {
		log.Fatalf("failed to initialise OpenTelemetry: %v", err)
	}

	mlitClient := mlit.NewClient()
	handler := api.NewHandler(mlitClient)
	router := api.NewRouter(handler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// バックグラウンドでサーバーを起動
	go func() {
		log.Printf("Yield-Guard backend starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// SIGINT / SIGTERM を待機してグレースフルシャットダウン
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 国交省APIの最大タイムアウト(30s) より長い猶予を確保
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	if err := otelShutdown(context.Background()); err != nil {
		log.Printf("OTel shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

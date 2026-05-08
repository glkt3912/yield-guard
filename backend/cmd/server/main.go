package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yield-guard/backend/internal/api"
	"github.com/yield-guard/backend/internal/logger"
	"github.com/yield-guard/backend/internal/mlit"
	"github.com/yield-guard/backend/internal/telemetry"
)

// readSecret reads a secret from a volume-mounted file, falling back to an env var for local dev.
func readSecret(filePath, envKey string) string {
	if data, err := os.ReadFile(filePath); err == nil {
		return strings.TrimSpace(string(data))
	}
	return os.Getenv(envKey)
}

func main() {
	logger.Init(os.Stderr)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mlitAPIKey := readSecret("/secrets/mlit-api-key", "MLIT_API_KEY")
	if mlitAPIKey == "" {
		slog.Error("MLIT_API_KEY is not set")
		os.Exit(1)
	}

	appInternalAPIKey := readSecret("/secrets/app-internal-api-key", "APP_INTERNAL_API_KEY")
	if os.Getenv("GIN_MODE") == "release" && appInternalAPIKey == "" {
		slog.Error("APP_INTERNAL_API_KEY must be set in production (GIN_MODE=release)")
		os.Exit(1)
	}

	ctx := context.Background()
	otelShutdown, err := telemetry.Setup(ctx, "yield-guard-backend", "0.1.0")
	if err != nil {
		slog.Error("failed to initialise OpenTelemetry", "error", err)
		os.Exit(1)
	}

	mlitClient := mlit.NewClient(mlitAPIKey)
	geocodeClient := api.NewGoogleGeocodeClient(readSecret("/secrets/google-maps-api-key", "GOOGLE_MAPS_API_KEY"))
	handler := api.NewHandler(mlitClient, geocodeClient)
	router := api.NewRouter(handler, appInternalAPIKey)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// バックグラウンドでサーバーを起動
	go func() {
		slog.Info("Yield-Guard backend starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// SIGINT / SIGTERM を待機してグレースフルシャットダウン
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	// 国交省APIの最大タイムアウト(30s) より長い猶予を確保
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}
	// OTel フラッシュに 5s のタイムアウトを設定。
	// ローリングデプロイ時に新旧インスタンスのメトリクス時系列が重複し
	// "Points must be written in order" エラーが発生するのは既知の挙動のため WARN に留める。
	otelCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer otelCancel()
	if err := otelShutdown(otelCtx); err != nil {
		slog.Warn("OTel shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

package api

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type warmupTarget struct {
	area string
	name string
}

var warmupAreas = []warmupTarget{
	{area: "13", name: "東京都"},
	{area: "27", name: "大阪府"},
	{area: "14", name: "神奈川県"},
}

// WarmCache は Cloud Scheduler から呼ばれるキャッシュウォームアップハンドラー
// goroutineで並列実行し、50秒のコンテキストタイムアウトを設定する
func (h *Handler) WarmCache(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Second)
	defer cancel()

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		success int
		failed  int
	)

	for _, target := range warmupAreas {
		wg.Add(1)
		go func(t warmupTarget) {
			defer wg.Done()
			if err := h.warmArea(ctx, t); err != nil {
				slog.Warn("warmup failed", "area", t.name, "err", err)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}
			slog.Info("warmup success", "area", t.name)
			mu.Lock()
			success++
			mu.Unlock()
		}(target)
	}

	wg.Wait()
	c.JSON(http.StatusOK, gin.H{"success": success, "failed": failed})
}

func (h *Handler) warmArea(ctx context.Context, t warmupTarget) error {
	// FetchMunicipalities でキャッシュをウォームアップ
	if _, err := h.mlitClient.FetchMunicipalities(ctx, t.area); err != nil {
		return err
	}
	return nil
}

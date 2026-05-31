package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
	"golang.org/x/sync/errgroup"
)

// areaDiscoveryLimit はエリア探索で並列取得する市区町村の上限数。
// 全市区町村を対象にするとタイムアウトリスクがあるため制限する（東京都は62市区町村）。
const areaDiscoveryLimit = 30

// HandleAreaDiscovery は都道府県内の市区町村を土地価格データで評価しランキング返却
// @Summary     投資エリア探索
// @Tags        area
// @Produce     json
// @Param       prefecture  query  string  true   "都道府県コード (例: 13)"
// @Param       budget      query  number  false  "予算 (円)"
// @Param       yield       query  number  false  "目標利回り (例: 0.07)"
// @Success     200  {object}  domain.AreaDiscoveryResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /api/area-discovery [get]
func (h *Handler) HandleAreaDiscovery(c *gin.Context) {
	prefecture := c.Query("prefecture")
	if prefecture == "" {
		badRequest(c, "prefecture は必須パラメータです")
		return
	}
	budgetStr := c.Query("budget")
	yieldStr := c.Query("yield")

	budget := 0.0
	targetYield := 0.08
	if budgetStr != "" {
		if v, err := strconv.ParseFloat(budgetStr, 64); err == nil && v > 0 {
			budget = v
		}
	}
	if yieldStr != "" {
		if v, err := strconv.ParseFloat(yieldStr, 64); err == nil && v > 0 {
			targetYield = v
		}
	}

	ctx := c.Request.Context()

	// 市区町村一覧取得
	municipalities, err := h.mlitClient.FetchMunicipalities(ctx, prefecture)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "市区町村一覧の取得に失敗しました"})
		return
	}

	// 最新2年の土地取引データを並列取得
	now := time.Now()
	toYear := now.Year()
	fromYear := toYear - 2

	limit := areaDiscoveryLimit
	if len(municipalities) < limit {
		limit = len(municipalities)
	}

	results := make([]domain.AreaDiscoveryItem, limit)
	sem := make(chan struct{}, 5) // 並列5件上限
	g, gctx := errgroup.WithContext(ctx)

	for i := 0; i < limit; i++ {
		idx, m := i, municipalities[i]
		g.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			transactions, fetchErr := h.mlitClient.FetchLandPrices(gctx, mlit.LandPriceQuery{
				Area:      prefecture,
				City:      m.ID,
				Year:      fromYear,
				Quarter:   1,
				ToYear:    toYear,
				ToQuarter: 4,
			})

			item := domain.AreaDiscoveryItem{
				MunicipalityCode: m.ID,
				MunicipalityName: m.Name,
			}
			if lat, lng, ok := domain.MunicipalityCenter(m.ID); ok {
				item.CenterLat = lat
				item.CenterLng = lng
			}

			if fetchErr != nil || len(transactions) == 0 {
				item.DataSufficient = false
				item.LandPriceTrend = "不明"
				item.YieldDifficulty = "unknown"
				item.YieldDifficultyLabel = "データ不足"
				results[idx] = item
				return nil
			}

			stats := domain.CalcLandPriceStats(gctx, transactions)

			item.MedianTsubo = stats.MedianTsubo
			item.TransactionCount = stats.Count
			item.DataSufficient = stats.Count >= 3
			item.YieldDifficulty, item.YieldDifficultyLabel = domain.CalcYieldDifficulty(stats.MedianTsubo, budget, targetYield)

			item.LandPriceTrend = domain.CalcLandPriceTrend(transactions)
			results[idx] = item
			return nil
		})
	}
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		slog.WarnContext(ctx, "area discovery partial failure", "err", err)
	}

	// MunicipalityCode が空のスロットはキャンセル等で goroutine が未実行のもの
	items := make([]domain.AreaDiscoveryItem, 0, limit)
	for _, item := range results {
		if item.MunicipalityCode != "" {
			items = append(items, item)
		}
	}

	// 達成可能 → やや困難 → 困難 → 不明の順、同一難易度内は取引件数降順
	sort.Slice(items, func(i, j int) bool {
		difficultyOrder := map[string]int{"achievable": 0, "slightly-difficult": 1, "difficult": 2, "unknown": 3}
		di, dj := difficultyOrder[items[i].YieldDifficulty], difficultyOrder[items[j].YieldDifficulty]
		if di != dj {
			return di < dj
		}
		return items[i].TransactionCount > items[j].TransactionCount
	})

	c.JSON(http.StatusOK, domain.AreaDiscoveryResponse{
		Items:      items,
		Prefecture: prefecture,
	})
}

// HandleAreaSummary はエリア選択時に投資難易度を AI が 2 文で要約して返す
// @Summary     エリア AI サマリー
// @Tags        area
// @Produce     json
// @Param       area          query  string   true   "都道府県コード (例: 13)"
// @Param       municipality  query  string   true   "市区町村コード (例: 13101)"
// @Param       budget        query  number   false  "予算 (円, 例: 50000000)"
// @Param       yield         query  number   false  "目標利回り (小数, 例: 0.08)"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /api/area-discovery/summary [get]
func (h *Handler) HandleAreaSummary(c *gin.Context) {
	area := c.Query("area")
	municipality := c.Query("municipality")
	if area == "" || municipality == "" {
		badRequest(c, "area と municipality は必須パラメータです")
		return
	}

	budget := 0.0
	targetYield := 0.08
	if budgetStr := c.Query("budget"); budgetStr != "" {
		if v, err := strconv.ParseFloat(budgetStr, 64); err == nil && v > 0 {
			budget = v
		}
	}
	if yieldStr := c.Query("yield"); yieldStr != "" {
		if v, err := strconv.ParseFloat(yieldStr, 64); err == nil && v > 0 {
			targetYield = v
		}
	}

	ctx := c.Request.Context()
	now := time.Now()
	toYear := now.Year()
	fromYear := toYear - 2

	transactions, err := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{
		Area:      area,
		City:      municipality,
		Year:      fromYear,
		Quarter:   1,
		ToYear:    toYear,
		ToQuarter: 4,
	})

	item := domain.AreaDiscoveryItem{
		MunicipalityCode: municipality,
		MunicipalityName: municipality,
	}

	if err != nil || len(transactions) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "area summary: land price fetch failed", "area", area, "municipality", municipality, "err", err)
		}
		item.DataSufficient = false
		item.YieldDifficulty = "unknown"
		item.YieldDifficultyLabel = "データ不足"
	} else {
		stats := domain.CalcLandPriceStats(ctx, transactions)
		item.MedianTsubo = stats.MedianTsubo
		item.TransactionCount = stats.Count
		item.DataSufficient = stats.Count >= 3
		item.YieldDifficulty, item.YieldDifficultyLabel = domain.CalcYieldDifficulty(stats.MedianTsubo, budget, targetYield)
		if item.YieldDifficultyLabel == "" {
			item.YieldDifficulty = "unknown"
			item.YieldDifficultyLabel = "データ不足"
		}
	}

	// 市区町村名を municipalities から取得する（ベストエフォート）
	if municipalities, mErr := h.mlitClient.FetchMunicipalities(ctx, area); mErr == nil {
		for _, m := range municipalities {
			if m.ID == municipality {
				item.MunicipalityName = m.Name
				break
			}
		}
	}

	summary := h.summarizer.GenerateAreaSummary(ctx, item)
	if summary == "" {
		summary = item.YieldDifficultyLabel
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

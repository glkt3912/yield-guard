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
			if gctx.Err() != nil {
				return gctx.Err()
			}
			sem <- struct{}{}
			defer func() { <-sem }()

			if gctx.Err() != nil {
				return gctx.Err()
			}

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

			if fetchErr != nil || len(transactions) == 0 {
				item.DataSufficient = false
				item.LandPriceTrend = "不明"
				item.YieldDifficulty = "difficult"
				item.YieldDifficultyLabel = "データ不足"
				results[idx] = item
				return nil
			}

			stats := domain.CalcLandPriceStats(gctx, transactions)

			item.MedianTsubo = stats.MedianTsubo
			item.TransactionCount = stats.Count
			item.DataSufficient = stats.Count >= 3

			// 利回り達成難易度: 予算（または中央坪単価×30坪+建物代）に対して目標利回りが必要とする月額家賃を試算し、
			// 1坪あたり月額賃料の現実性で判定する
			var totalCostEst float64
			if budget > 0 {
				totalCostEst = budget
			} else {
				totalCostEst = stats.MedianTsubo*30 + 10_000_000
			}
			annualRentNeeded := totalCostEst * targetYield

			// 1坪あたり月額賃料が現実的か判定（目安: 8,000円以下=達成可能, 15,000円超=困難）
			monthlyRentNeeded := annualRentNeeded / 12
			areaTsubo := 30.0
			rentPerTsubo := monthlyRentNeeded / areaTsubo
			if rentPerTsubo <= 8000 {
				item.YieldDifficulty = "achievable"
				item.YieldDifficultyLabel = "達成可能"
			} else if rentPerTsubo <= 15000 {
				item.YieldDifficulty = "slightly-difficult"
				item.YieldDifficultyLabel = "やや困難"
			} else {
				item.YieldDifficulty = "difficult"
				item.YieldDifficultyLabel = "困難"
			}

			item.LandPriceTrend = "データなし"
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

	// 達成可能 → やや困難 → 困難 の順、同一難易度内は取引件数降順
	sort.Slice(items, func(i, j int) bool {
		difficultyOrder := map[string]int{"achievable": 0, "slightly-difficult": 1, "difficult": 2}
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

package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/ai"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// MLITClient は国交省APIクライアントのインターフェース（テスト時にモック注入可能）
type MLITClient interface {
	FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
	FetchMunicipalities(ctx context.Context, area string) ([]mlit.Municipality, error)
	FetchStationRidership(ctx context.Context, z, x, y int) ([]mlit.StationRidership, error)
	FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
	FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
	FetchLocationOptimization(ctx context.Context, z, x, y int) ([]domain.LocationOptimizationItem, error)
	FetchEmbankment(ctx context.Context, z, x, y int) ([]domain.EmbankmentItem, error)
	FetchUrbanRoad(ctx context.Context, z, x, y int) ([]domain.UrbanRoadItem, error)
	FetchDisasterHistory(ctx context.Context, z, x, y int) ([]domain.DisasterHistoryItem, error)
	FetchUrbanZoning(ctx context.Context, z, x, y int) ([]domain.UrbanZoningItem, error)
	FetchLiquefaction(ctx context.Context, z, x, y int) ([]domain.LiquefactionRiskItem, error)
	FetchFloodHazard(ctx context.Context, z, x, y int) ([]domain.FloodHazardItem, error)
	FetchStormHazard(ctx context.Context, z, x, y int) ([]domain.StormHazardItem, error)
	FetchTsunamiHazard(ctx context.Context, z, x, y int) ([]domain.TsunamiHazardItem, error)
	FetchLandslideHazard(ctx context.Context, z, x, y int) ([]domain.LandslideHazardItem, error)
	FetchRentStats(ctx context.Context, q mlit.LandPriceQuery, areaSqm float64) (domain.RentStatsResult, error)
}

type Handler struct {
	mlitClient    MLITClient
	geocodeClient GeocodeClient
	summarizer    ai.Summarizer
}

func NewHandler(mlitClient MLITClient, geocodeClient GeocodeClient) *Handler {
	return &Handler{
		mlitClient:    mlitClient,
		geocodeClient: geocodeClient,
		summarizer:    ai.NewSummarizer(),
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

func validateRenovationInput(in domain.RenovationInput) error {
	if in.PropertyPrice <= 0 {
		return errors.New("propertyPrice は正の値を指定してください")
	}
	if in.AnnualBaseRent < 0 {
		return errors.New("annualBaseRent は 0 以上を指定してください")
	}
	if in.EffectiveTaxRate < 0 || in.EffectiveTaxRate > 1 {
		return errors.New("effectiveTaxRate は 0.0〜1.0 の範囲で指定してください")
	}
	if in.SelfLaborRatePerHour < 0 {
		return errors.New("selfLaborRatePerHour は 0 以上を指定してください")
	}
	if len(in.Items) == 0 {
		return errors.New("items は 1 件以上指定してください")
	}
	for idx, item := range in.Items {
		if item.Cost <= 0 {
			return fmt.Errorf("items[%d].cost は正の値を指定してください", idx)
		}
	}
	return nil
}

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

	type result struct {
		item domain.AreaDiscoveryItem
		err  error
	}

	// 上位30市区町村に絞る（全件だとタイムアウトリスク）
	limit := 30
	if len(municipalities) < limit {
		limit = len(municipalities)
	}

	results := make([]result, limit)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // 並列5件上限

	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func(idx int, m mlit.Municipality) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			transactions, fetchErr := h.mlitClient.FetchLandPrices(ctx, mlit.LandPriceQuery{
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
				results[idx] = result{item: item}
				return
			}

			stats := domain.CalcLandPriceStats(ctx, transactions)

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
			results[idx] = result{item: item}
		}(i, municipalities[i])
	}
	wg.Wait()

	items := make([]domain.AreaDiscoveryItem, 0, limit)
	for _, r := range results {
		if r.err == nil {
			items = append(items, r.item)
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

// HandleRenovationAnalyze はリフォームROIシミュレーションを実行する
// @Summary     リフォームROI分析
// @Tags        renovation
// @Accept      json
// @Produce     json
// @Param       body  body  domain.RenovationInput  true  "リフォーム分析リクエスト"
// @Success     200  {object}  domain.RenovationResult
// @Failure     400  {object}  map[string]string
// @Router      /api/renovation/analyze [post]
func (h *Handler) HandleRenovationAnalyze(c *gin.Context) {
	var input domain.RenovationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}
	if err := validateRenovationInput(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := domain.CalcRenovationROI(input)
	c.JSON(http.StatusOK, result)
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

	// 住所はPIIになりうるため、accessLogMiddleware が記録するクエリ文字列をマスクする
	// nil ガードより先に実行することで、全パスでマスクが適用される
	masked := address
	if len([]rune(address)) > 10 {
		masked = string([]rune(address)[:10]) + "***"
	}
	c.Request.URL.RawQuery = "address=" + url.QueryEscape(masked)
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

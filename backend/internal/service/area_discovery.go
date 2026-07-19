package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yield-guard/backend/internal/ai"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// areaDiscoveryLimit はエリア探索で並列取得する市区町村の上限数。
// 全市区町村を対象にするとタイムアウトリスクがあるため制限する（東京都は62市区町村）。
const areaDiscoveryLimit = 30

// yieldDifficultyLabelFor は domain.CalcYieldDifficulty のコード値を日本語表示ラベルへ変換する。
// 表示文言は domain（純粋計算層）から分離し service 層で管理する。
// 未知コード・空コード（データ不足）は "" を返し、呼び出し側のフォールバックに委ねる。
func yieldDifficultyLabelFor(code string) string {
	switch code {
	case "achievable":
		return "達成可能"
	case "slightly-difficult":
		return "やや困難"
	case "difficult":
		return "困難"
	default:
		return ""
	}
}

// AreaMLITClient はエリア探索に必要な国交省APIのサブセット
type AreaMLITClient interface {
	FetchMunicipalities(ctx context.Context, area string) ([]mlit.Municipality, error)
	FetchLandPrices(ctx context.Context, q mlit.LandPriceQuery) ([]domain.LandTransaction, error)
}

// AreaService は市区町村単位の投資エリア評価を担う
type AreaService interface {
	// Discover は都道府県内の市区町村を土地価格データで評価しランキングを返す。
	// 市区町村一覧の取得失敗のみエラーとし、個別市区町村のデータ取得失敗はデータ不足として扱う。
	Discover(ctx context.Context, prefecture string, budget, targetYield float64) (domain.AreaDiscoveryResponse, error)
	// Summarize は指定市区町村の投資難易度を AI が要約した文字列を返す。
	// AI が利用できない場合は難易度ラベルにフォールバックする。
	Summarize(ctx context.Context, area, municipality string, budget, targetYield float64) string
}

// AreaDiscoveryService は AreaService の実装
type AreaDiscoveryService struct {
	client     AreaMLITClient
	summarizer ai.Summarizer
}

func NewAreaDiscoveryService(client AreaMLITClient, summarizer ai.Summarizer) AreaService {
	return &AreaDiscoveryService{client: client, summarizer: summarizer}
}

// landPriceRange は直近2年の取引期間を返す
func landPriceRange() (fromYear, toYear int) {
	toYear = time.Now().Year()
	return toYear - 2, toYear
}

func (s *AreaDiscoveryService) Discover(ctx context.Context, prefecture string, budget, targetYield float64) (domain.AreaDiscoveryResponse, error) {
	municipalities, err := s.client.FetchMunicipalities(ctx, prefecture)
	if err != nil {
		return domain.AreaDiscoveryResponse{}, fmt.Errorf("fetch municipalities: %w", err)
	}

	// 最新2年の土地取引データを並列取得
	fromYear, toYear := landPriceRange()

	limit := areaDiscoveryLimit
	if len(municipalities) < limit {
		limit = len(municipalities)
	}

	results := make([]domain.AreaDiscoveryItem, limit)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5) // 並列5件上限

	for i := 0; i < limit; i++ {
		idx, m := i, municipalities[i]
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			transactions, fetchErr := s.client.FetchLandPrices(gctx, mlit.LandPriceQuery{
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
			item.YieldDifficulty = domain.CalcYieldDifficulty(stats.MedianTsubo, budget, targetYield)
			item.YieldDifficultyLabel = yieldDifficultyLabelFor(item.YieldDifficulty)

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

	return domain.AreaDiscoveryResponse{
		Items:      items,
		Prefecture: prefecture,
	}, nil
}

func (s *AreaDiscoveryService) Summarize(ctx context.Context, area, municipality string, budget, targetYield float64) string {
	fromYear, toYear := landPriceRange()

	transactions, err := s.client.FetchLandPrices(ctx, mlit.LandPriceQuery{
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
		item.YieldDifficulty = domain.CalcYieldDifficulty(stats.MedianTsubo, budget, targetYield)
		item.YieldDifficultyLabel = yieldDifficultyLabelFor(item.YieldDifficulty)
		if item.YieldDifficultyLabel == "" {
			item.YieldDifficulty = "unknown"
			item.YieldDifficultyLabel = "データ不足"
		}
	}

	// 市区町村名を municipalities から取得する（ベストエフォート）
	if municipalities, mErr := s.client.FetchMunicipalities(ctx, area); mErr == nil {
		for _, m := range municipalities {
			if m.ID == municipality {
				item.MunicipalityName = m.Name
				break
			}
		}
	}

	summary := s.summarizer.GenerateAreaSummary(ctx, item)
	if summary == "" {
		summary = item.YieldDifficultyLabel
	}
	return summary
}

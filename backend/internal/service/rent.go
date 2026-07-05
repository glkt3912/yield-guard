package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/mlit"
)

// RentMLITClient は賃料統計に必要な国交省APIのサブセット
type RentMLITClient interface {
	FetchRentStats(ctx context.Context, q mlit.LandPriceQuery, areaSqm float64) (domain.RentStatsResult, error)
}

// RentService はエリア賃料相場の算出を担う
type RentService interface {
	// Stats は直近2年分の賃貸データからエリア賃料相場を返す。
	// データなし・取得失敗時は nil を返す（フロントは非表示にする）。
	Stats(ctx context.Context, area, municipality string, areaSqm float64) *domain.RentStatsResult
}

// RentStatsService は RentService の実装
type RentStatsService struct {
	client RentMLITClient
}

func NewRentStatsService(client RentMLITClient) RentService {
	return &RentStatsService{client: client}
}

func (s *RentStatsService) Stats(ctx context.Context, area, municipality string, areaSqm float64) *domain.RentStatsResult {
	// 直近2年分のデータを取得する（当四半期は未確定のため直前の四半期まで）
	now := time.Now()
	toYear := now.Year()
	currentQuarter := (int(now.Month())-1)/3 + 1
	toQuarter := currentQuarter - 1
	if toQuarter == 0 {
		toQuarter = 4
		toYear--
	}
	fromYear := toYear - 2

	q := mlit.LandPriceQuery{
		Area:      area,
		City:      municipality,
		Year:      fromYear,
		Quarter:   1,
		ToYear:    toYear,
		ToQuarter: toQuarter,
	}

	result, err := s.client.FetchRentStats(ctx, q, areaSqm)
	if err != nil {
		slog.WarnContext(ctx, "FetchRentStats failed", "err", err, "area", area)
		// データなしはサイレントに空レスポンスを返す（フロントはhide）
		return nil
	}
	if result.Count == 0 {
		return nil
	}

	result.LowConfidence = result.Count < 3
	return &result
}

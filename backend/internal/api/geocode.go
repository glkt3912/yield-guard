package api

import (
	"context"

	"github.com/yield-guard/backend/internal/geocode"
)

// GeocodeResult は /api/geocode のレスポンス DTO。
// swagger 定義 api.GeocodeResult として TS 契約に対応する。
type GeocodeResult struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	LocationType string  `json:"locationType"`
}

// GeocodeClient は Handler が住所→座標変換に用いる consumer 定義インターフェース。
type GeocodeClient interface {
	Geocode(ctx context.Context, address string) (*geocode.Result, error)
}

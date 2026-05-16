package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

var (
	errGeocodeNotConfigured = errors.New("geocode: API key not configured")
	errGeocodeNotFound      = errors.New("geocode: address not found")
	errGeocodeUpstream      = errors.New("geocode: upstream error")
)

// GeocodeResult はジオコーディング結果を表す
type GeocodeResult struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	LocationType string  `json:"locationType"`
}

// GeocodeClient は住所→座標変換のインターフェース（テスト時にモック注入可能）
type GeocodeClient interface {
	Geocode(ctx context.Context, address string) (*GeocodeResult, error)
}

// nominatimGeocodeClient は Nominatim (OpenStreetMap) を使用するジオコーダー
type nominatimGeocodeClient struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	cache      GeocodeCache
}

// nominatimSearchResult は Nominatim /search レスポンスの各要素
type nominatimSearchResult struct {
	Lat     string `json:"lat"`
	Lon     string `json:"lon"`
	Class   string `json:"class"`
	Type    string `json:"type"`
	PlaceID int64  `json:"place_id"`
}

// NewNominatimGeocodeClient は Nominatim ジオコーダーを返す。
// cache が nil の場合はキャッシュなし（noopGeocodeCache）で動作する。
func NewNominatimGeocodeClient(cache GeocodeCache) GeocodeClient {
	if cache == nil {
		cache = &noopGeocodeCache{}
	}
	return &nominatimGeocodeClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		// rate.Limiter はプロセス内のみ有効。Cloud Run が複数インスタンスに
		// スケールした場合はインスタンスをまたいだレート制限にはならない。
		limiter: rate.NewLimiter(rate.Every(time.Second), 1),
		cache:   cache,
	}
}

func (c *nominatimGeocodeClient) Geocode(ctx context.Context, address string) (*GeocodeResult, error) {
	if cached, ok := c.cache.Get(ctx, address); ok {
		return cached, nil
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: rate limiter: %w", errGeocodeUpstream, err)
	}

	q := url.Values{}
	q.Set("q", address)
	q.Set("format", "json")
	q.Set("countrycodes", "jp")
	q.Set("limit", "1")
	endpoint := "https://nominatim.openstreetmap.org/search?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGeocodeUpstream, err)
	}
	req.Header.Set("User-Agent", "YieldGuard/1.0 (mole.gunma@gmail.com)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGeocodeUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", errGeocodeUpstream, resp.StatusCode)
	}

	var results []nominatimSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("%w: %w", errGeocodeUpstream, err)
	}

	if len(results) == 0 {
		return nil, errGeocodeNotFound
	}

	var lat, lng float64
	if _, err := fmt.Sscanf(results[0].Lat, "%f", &lat); err != nil {
		return nil, fmt.Errorf("%w: invalid lat: %w", errGeocodeUpstream, err)
	}
	if _, err := fmt.Sscanf(results[0].Lon, "%f", &lng); err != nil {
		return nil, fmt.Errorf("%w: invalid lon: %w", errGeocodeUpstream, err)
	}

	gr := &GeocodeResult{
		Lat:          lat,
		Lng:          lng,
		LocationType: results[0].Type,
	}
	c.cache.Set(ctx, address, gr)
	return gr, nil
}

// GeocodeCache はジオコード結果のキャッシュインターフェース
type GeocodeCache interface {
	Get(ctx context.Context, address string) (*GeocodeResult, bool)
	Set(ctx context.Context, address string, result *GeocodeResult)
}

// noopGeocodeCache はキャッシュなしのダミー実装
type noopGeocodeCache struct{}

func (n *noopGeocodeCache) Get(_ context.Context, _ string) (*GeocodeResult, bool) { return nil, false }
func (n *noopGeocodeCache) Set(_ context.Context, _ string, _ *GeocodeResult)      {}

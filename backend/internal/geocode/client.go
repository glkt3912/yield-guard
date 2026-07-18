package geocode

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
	ErrNotConfigured = errors.New("geocode: API key not configured")
	ErrNotFound      = errors.New("geocode: address not found")
	ErrUpstream      = errors.New("geocode: upstream error")
)

// Result はジオコーディング結果を表す。
// json タグは Firestore キャッシュの blob 形式のために付与している
// （HTTP レスポンスは api 層の DTO にマップされる）。
type Result struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	LocationType string  `json:"locationType"`
}

// Client は住所→座標変換のインターフェース（テスト時にモック注入可能）
type Client interface {
	Geocode(ctx context.Context, address string) (*Result, error)
}

// NominatimClient は Nominatim (OpenStreetMap) を使用するジオコーダー
type NominatimClient struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	cache      Cache
}

// nominatimSearchResult は Nominatim /search レスポンスの各要素
type nominatimSearchResult struct {
	Lat     string `json:"lat"`
	Lon     string `json:"lon"`
	Class   string `json:"class"`
	Type    string `json:"type"`
	PlaceID int64  `json:"place_id"`
}

// NewNominatimClient は Nominatim ジオコーダーを返す。
// cache が nil の場合はキャッシュなし（noopCache）で動作する。
func NewNominatimClient(cache Cache) Client {
	if cache == nil {
		cache = &noopCache{}
	}
	return &NominatimClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		// rate.Limiter はプロセス内のみ有効。Cloud Run が複数インスタンスに
		// スケールした場合はインスタンスをまたいだレート制限にはならない。
		limiter: rate.NewLimiter(rate.Every(time.Second), 1),
		cache:   cache,
	}
}

func (c *NominatimClient) Geocode(ctx context.Context, address string) (*Result, error) {
	if cached, ok := c.cache.Get(ctx, address); ok {
		return cached, nil
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: rate limiter: %w", ErrUpstream, err)
	}

	q := url.Values{}
	q.Set("q", address)
	q.Set("format", "json")
	q.Set("countrycodes", "jp")
	q.Set("limit", "1")
	endpoint := "https://nominatim.openstreetmap.org/search?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	req.Header.Set("User-Agent", "YieldGuard/1.0 (mole.gunma@gmail.com)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUpstream, resp.StatusCode)
	}

	var results []nominatimSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}

	if len(results) == 0 {
		return nil, ErrNotFound
	}

	var lat, lng float64
	if _, err := fmt.Sscanf(results[0].Lat, "%f", &lat); err != nil {
		return nil, fmt.Errorf("%w: invalid lat: %w", ErrUpstream, err)
	}
	if _, err := fmt.Sscanf(results[0].Lon, "%f", &lng); err != nil {
		return nil, fmt.Errorf("%w: invalid lon: %w", ErrUpstream, err)
	}

	gr := &Result{
		Lat:          lat,
		Lng:          lng,
		LocationType: results[0].Type,
	}
	c.cache.Set(ctx, address, gr)
	return gr, nil
}

// Cache はジオコード結果のキャッシュインターフェース
type Cache interface {
	Get(ctx context.Context, address string) (*Result, bool)
	Set(ctx context.Context, address string, result *Result)
}

// noopCache はキャッシュなしのダミー実装
type noopCache struct{}

func (n *noopCache) Get(_ context.Context, _ string) (*Result, bool) { return nil, false }
func (n *noopCache) Set(_ context.Context, _ string, _ *Result)      {}

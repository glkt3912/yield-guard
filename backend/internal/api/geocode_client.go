package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
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

type googleGeocodeClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewGoogleGeocodeClient は Google Maps Geocoding API クライアントを返す
func NewGoogleGeocodeClient(apiKey string) GeocodeClient {
	return &googleGeocodeClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// googleGeocodeResponse は Google Maps Geocoding API のレスポンス構造体
type googleGeocodeResponse struct {
	Status  string `json:"status"`
	Results []struct {
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
		} `json:"geometry"`
	} `json:"results"`
}

func (c *googleGeocodeClient) Geocode(ctx context.Context, address string) (*GeocodeResult, error) {
	if c.apiKey == "" {
		return nil, errGeocodeNotConfigured
	}

	q := url.Values{}
	q.Set("address", address)
	q.Set("language", "ja")
	q.Set("key", c.apiKey)
	endpoint := "https://maps.googleapis.com/maps/api/geocode/json?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGeocodeUpstream, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGeocodeUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", errGeocodeUpstream, resp.StatusCode)
	}

	var gr googleGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("%w: %w", errGeocodeUpstream, err)
	}

	switch gr.Status {
	case "OK":
		if len(gr.Results) == 0 {
			return nil, fmt.Errorf("%w: empty results", errGeocodeUpstream)
		}
		loc := gr.Results[0].Geometry
		return &GeocodeResult{
			Lat:          loc.Location.Lat,
			Lng:          loc.Location.Lng,
			LocationType: loc.LocationType,
		}, nil
	case "ZERO_RESULTS":
		return nil, errGeocodeNotFound
	default:
		return nil, fmt.Errorf("%w: status=%s", errGeocodeUpstream, gr.Status)
	}
}

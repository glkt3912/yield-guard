package mlit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/telemetry"
)

const (
	mlitTracerName = "github.com/yield-guard/backend/internal/mlit"

	// 不動産情報ライブラリ API ベースURL (2024年4月〜)
	mlitBaseURL = "https://www.reinfolib.mlit.go.jp/ex-api/external"

	// エンドポイントパス
	endpointLandPrices         = "/XIT001"
	endpointMunicipalities     = "/XIT002"
	endpointStationRidership   = "/XKT015"
	endpointPopulationForecast = "/XKT013"
	endpointLandAppraisals     = "/XCT001"

	requestTimeout = 30 * time.Second

	// リトライ設定: 国交省APIは一時的な障害が多いため指数バックオフで再試行する
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
)

// Client は国交省 不動産取引価格情報取得APIのクライアント
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	cache      *cache
}

// NewClient は新しい Client を返す。
// 環境変数 MLIT_API_KEY からAPIキーを読み込む。
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		baseURL:    mlitBaseURL,
		apiKey:     os.Getenv("MLIT_API_KEY"),
		cache:      newCache(),
	}
}

// FetchLandPrices は指定条件で土地取引価格を取得し、統計を返す。
// キャッシュヒット時はAPIコールをスキップする（TTL: 24時間）。
// 一時的なネットワーク障害や 5xx レスポンスに対して指数バックオフでリトライする（ISSUE-13）。
func (c *Client) FetchLandPrices(ctx context.Context, q LandPriceQuery) ([]domain.LandTransaction, error) {
	ctx, span := otel.Tracer(mlitTracerName).Start(ctx, "mlit.FetchLandPrices")
	defer span.End()

	span.SetAttributes(
		semconv.ServerAddress("www.reinfolib.mlit.go.jp"),
		semconv.ServerPort(443),
		semconv.URLScheme("https"),
		attribute.String("mlit.endpoint", "XIT001"),
		attribute.String("mlit.query.area", q.Area),
		attribute.String("mlit.query.city", q.City),
		attribute.Int("mlit.query.year", q.Year),
		attribute.Int("mlit.query.quarter", q.Quarter),
	)

	key := cacheKey(q)
	if cached, ok := c.cache.get(key); ok {
		span.SetAttributes(attribute.Bool("mlit.cache.hit", true))
		telemetry.MLITCacheHits.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XIT001")))
		return cached, nil
	}
	telemetry.MLITCacheMisses.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XIT001")))
	span.SetAttributes(attribute.Bool("mlit.cache.hit", false))

	apiURL, err := c.buildLandPricesURL(q)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数バックオフ: 1s, 2s, 4s ...
			delay := retryBaseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		start := time.Now()
		result, err := c.doRequest(ctx, apiURL)
		latencyMs := float64(time.Since(start).Milliseconds())
		telemetry.MLITAPILatencyHistogram.Record(ctx, latencyMs,
			metric.WithAttributes(attribute.String("mlit.endpoint", "XIT001")))

		if err == nil {
			span.SetAttributes(attribute.Int("mlit.retry.count", attempt))
			c.cache.set(key, result)
			return result, nil
		}
		lastErr = err

		// クライアントエラー (4xx) はリトライしない
		if isClientError(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	span.RecordError(lastErr)
	span.SetStatus(codes.Error, lastErr.Error())
	return nil, fmt.Errorf("API request failed after %d attempts: %w", maxRetries, lastErr)
}

// FetchMunicipalities は指定都道府県の市区町村一覧を取得する（XIT002）。
// キャッシュヒット時はAPIコールをスキップする（TTL: 24時間）。
func (c *Client) FetchMunicipalities(ctx context.Context, area string) ([]Municipality, error) {
	ctx, span := otel.Tracer(mlitTracerName).Start(ctx, "mlit.FetchMunicipalities")
	defer span.End()

	span.SetAttributes(
		semconv.ServerAddress("www.reinfolib.mlit.go.jp"),
		semconv.ServerPort(443),
		semconv.URLScheme("https"),
		attribute.String("mlit.endpoint", "XIT002"),
		attribute.String("mlit.query.area", area),
	)

	if area == "" {
		err := fmt.Errorf("area is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if cached, ok := c.cache.getMuni(area); ok {
		span.SetAttributes(attribute.Bool("mlit.cache.hit", true))
		telemetry.MLITCacheHits.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XIT002")))
		return cached, nil
	}
	telemetry.MLITCacheMisses.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XIT002")))
	span.SetAttributes(attribute.Bool("mlit.cache.hit", false))

	params := url.Values{}
	params.Set("area", area)
	apiURL := c.baseURL + endpointMunicipalities + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("request build error: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latencyMs := float64(time.Since(start).Milliseconds())
	telemetry.MLITAPILatencyHistogram.Record(ctx, latencyMs,
		metric.WithAttributes(attribute.String("mlit.endpoint", "XIT002")))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("municipalities API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		err := &clientError{code: resp.StatusCode}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("municipalities API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var apiResp MunicipalityResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("municipalities JSON decode error: %w", err)
	}
	if apiResp.Status != "OK" {
		err := fmt.Errorf("municipalities API status: %s", apiResp.Status)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("mlit.retry.count", 0))
	c.cache.setMuni(area, apiResp.Data)
	return apiResp.Data, nil
}

// LatLngToTile は緯度経度をWebMercatorタイル座標に変換する。
func LatLngToTile(lat, lng float64, z int) (x, y int) {
	n := math.Pow(2, float64(z))
	x = int(math.Floor((lng + 180.0) / 360.0 * n))
	latRad := lat * math.Pi / 180.0
	y = int(math.Floor((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n))
	return x, y
}

// FetchStationRidership はタイル座標で XKT015 を呼び出し駅別乗降客数を取得する。
// キャッシュヒット時はAPIコールをスキップする（TTL: 24時間）。
func (c *Client) FetchStationRidership(ctx context.Context, z, x, y int) ([]StationRidership, error) {
	ctx, span := otel.Tracer(mlitTracerName).Start(ctx, "mlit.FetchStationRidership")
	defer span.End()

	span.SetAttributes(
		semconv.ServerAddress("www.reinfolib.mlit.go.jp"),
		semconv.ServerPort(443),
		semconv.URLScheme("https"),
		attribute.String("mlit.endpoint", "XKT015"),
		attribute.Int("mlit.query.tile_z", z),
		attribute.Int("mlit.query.tile_x", x),
		attribute.Int("mlit.query.tile_y", y),
	)

	key := fmt.Sprintf("ridership:%d:%d:%d", z, x, y)
	if cached, ok := c.cache.getRidership(key); ok {
		span.SetAttributes(attribute.Bool("mlit.cache.hit", true))
		telemetry.MLITCacheHits.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XKT015")))
		return cached, nil
	}
	telemetry.MLITCacheMisses.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XKT015")))
	span.SetAttributes(attribute.Bool("mlit.cache.hit", false))

	params := url.Values{}
	params.Set("response_format", "geojson")
	params.Set("z", strconv.Itoa(z))
	params.Set("x", strconv.Itoa(x))
	params.Set("y", strconv.Itoa(y))
	apiURL := c.baseURL + endpointStationRidership + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("request build error: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latencyMs := float64(time.Since(start).Milliseconds())
	telemetry.MLITAPILatencyHistogram.Record(ctx, latencyMs,
		metric.WithAttributes(attribute.String("mlit.endpoint", "XKT015")))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("station ridership API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		err := &clientError{code: resp.StatusCode}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("station ridership API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var geoResp StationRidershipGeoJSON
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("station ridership GeoJSON decode error: %w", err)
	}

	span.SetAttributes(attribute.Int("mlit.retry.count", 0))
	result := parseStationRiderships(geoResp.Features)
	c.cache.setRidership(key, result)
	return result, nil
}

// parseStationRiderships は GeoJSON フィーチャを StationRidership スライスに変換する。
// 同一（駅名, 路線名）の重複フィーチャは乗降客数が最大のものを残す。
func parseStationRiderships(features []StationRidershipFeature) []StationRidership {
	type key struct{ station, line string }
	best := make(map[key]StationRidership, len(features))
	for _, f := range features {
		p := f.Properties.StationName
		l := f.Properties.LineName
		if p == "" {
			continue
		}
		k := key{p, l}
		s := StationRidership{
			StationName: p,
			LineName:    l,
			Passengers:  latestPassengers(f.Properties),
		}
		if prev, ok := best[k]; !ok || s.Passengers > prev.Passengers {
			best[k] = s
		}
	}
	result := make([]StationRidership, 0, len(best))
	for _, s := range best {
		result = append(result, s)
	}
	return result
}

// latestPassengers は年別乗降客数フィールドから最新の有効値（非ゼロ）を返す。
// 2023年（S12_057）から降順にスキャンし、最初に非ゼロの値を返す。全年ゼロなら 0 を返す。
func latestPassengers(p StationRidershipProperties) int {
	for _, v := range []int{p.P2023, p.P2022, p.P2021, p.P2020, p.P2019, p.P2018, p.P2017, p.P2016, p.P2015, p.P2014, p.P2013, p.P2012, p.P2011} {
		if v > 0 {
			return v
		}
	}
	return 0
}

// FetchPopulationForecast はタイル座標で XKT013 を呼び出し将来推計人口を取得する。
// キャッシュヒット時はAPIコールをスキップする（TTL: 24時間）。
// 複数メッシュヒット時は人口を合算して返す。
func (c *Client) FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error) {
	ctx, span := otel.Tracer(mlitTracerName).Start(ctx, "mlit.FetchPopulationForecast")
	defer span.End()

	span.SetAttributes(
		semconv.ServerAddress("www.reinfolib.mlit.go.jp"),
		semconv.ServerPort(443),
		semconv.URLScheme("https"),
		attribute.String("mlit.endpoint", "XKT013"),
		attribute.Int("mlit.query.tile_z", z),
		attribute.Int("mlit.query.tile_x", x),
		attribute.Int("mlit.query.tile_y", y),
	)

	key := fmt.Sprintf("population:%d:%d:%d", z, x, y)
	if cached, ok := c.cache.getPopulation(key); ok {
		span.SetAttributes(attribute.Bool("mlit.cache.hit", true))
		telemetry.MLITCacheHits.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XKT013")))
		return cached, nil
	}
	telemetry.MLITCacheMisses.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XKT013")))
	span.SetAttributes(attribute.Bool("mlit.cache.hit", false))

	params := url.Values{}
	params.Set("response_format", "geojson")
	params.Set("z", strconv.Itoa(z))
	params.Set("x", strconv.Itoa(x))
	params.Set("y", strconv.Itoa(y))
	apiURL := c.baseURL + endpointPopulationForecast + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("request build error: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latencyMs := float64(time.Since(start).Milliseconds())
	telemetry.MLITAPILatencyHistogram.Record(ctx, latencyMs,
		metric.WithAttributes(attribute.String("mlit.endpoint", "XKT013")))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("population forecast API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		err := &clientError{code: resp.StatusCode}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("population forecast API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var geoResp PopulationForecastGeoJSON
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("population forecast GeoJSON decode error: %w", err)
	}

	span.SetAttributes(attribute.Int("mlit.retry.count", 0))
	result := parsePopulationForecasts(geoResp.Features)
	c.cache.setPopulation(key, result)
	return result, nil
}

// parsePopulationForecasts は GeoJSON フィーチャを年別人口スライスに変換する。
// フィーチャが0件（タイル外・海上等）の場合は nil を返す。
// 複数メッシュの人口は合算する。
func parsePopulationForecasts(features []PopulationForecastFeature) []domain.PopulationForecastItem {
	if len(features) == 0 {
		return nil
	}
	type yearPop struct {
		year int
		pop  float64
	}
	totals := []yearPop{
		{2020, 0}, {2025, 0}, {2030, 0}, {2035, 0}, {2040, 0}, {2045, 0}, {2050, 0},
	}
	for _, f := range features {
		p := f.Properties
		totals[0].pop += p.PTN2020
		totals[1].pop += p.PTN2025
		totals[2].pop += p.PTN2030
		totals[3].pop += p.PTN2035
		totals[4].pop += p.PTN2040
		totals[5].pop += p.PTN2045
		totals[6].pop += p.PTN2050
	}
	result := make([]domain.PopulationForecastItem, 0, len(totals))
	for _, t := range totals {
		result = append(result, domain.PopulationForecastItem{Year: t.year, Pop: t.pop})
	}
	return result
}

// FetchLandAppraisals は XCT001 を呼び出し地価公示情報を取得する。
// city が指定された場合は都道府県コード+市区町村コードで絞り込む（クライアントサイドフィルタリング）。
// キャッシュヒット時はAPIコールをスキップする（TTL: 24時間）。
// 一時的なネットワーク障害や 5xx レスポンスに対して指数バックオフでリトライする。
func (c *Client) FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error) {
	ctx, span := otel.Tracer(mlitTracerName).Start(ctx, "mlit.FetchLandAppraisals")
	defer span.End()

	span.SetAttributes(
		semconv.ServerAddress("www.reinfolib.mlit.go.jp"),
		semconv.ServerPort(443),
		semconv.URLScheme("https"),
		attribute.String("mlit.endpoint", "XCT001"),
		attribute.String("mlit.query.area", area),
		attribute.Int("mlit.query.year", year),
	)

	key := fmt.Sprintf("appraisals:%s:%s:%d:%s", area, city, year, division)
	if cached, ok := c.cache.getAppraisals(key); ok {
		span.SetAttributes(attribute.Bool("mlit.cache.hit", true))
		telemetry.MLITCacheHits.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XCT001")))
		return cached, nil
	}
	telemetry.MLITCacheMisses.Add(ctx, 1, metric.WithAttributes(attribute.String("mlit.endpoint", "XCT001")))
	span.SetAttributes(attribute.Bool("mlit.cache.hit", false))

	params := url.Values{}
	params.Set("area", area)
	params.Set("year", strconv.Itoa(year))
	params.Set("division", division)
	apiURL := c.baseURL + endpointLandAppraisals + "?" + params.Encode()

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		start := time.Now()
		result, err := c.doAppraisalsRequest(ctx, apiURL, area, city)
		latencyMs := float64(time.Since(start).Milliseconds())
		telemetry.MLITAPILatencyHistogram.Record(ctx, latencyMs,
			metric.WithAttributes(attribute.String("mlit.endpoint", "XCT001")))

		if err == nil {
			span.SetAttributes(attribute.Int("mlit.retry.count", attempt))
			c.cache.setAppraisals(key, result)
			return result, nil
		}
		lastErr = err
		if isClientError(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	span.RecordError(lastErr)
	span.SetStatus(codes.Error, lastErr.Error())
	return nil, fmt.Errorf("land appraisals API request failed after %d attempts: %w", maxRetries, lastErr)
}

// doAppraisalsRequest は単一の XCT001 HTTPリクエストを実行してパースする。
func (c *Client) doAppraisalsRequest(ctx context.Context, apiURL, area, city string) ([]domain.LandAppraisalItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request build error: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("land appraisals API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, &clientError{code: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("land appraisals API returned status %d", resp.StatusCode)
	}

	var apiResp LandAppraisalResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("land appraisals JSON decode error: %w", err)
	}
	if apiResp.Status != "OK" {
		return nil, fmt.Errorf("land appraisals API status: %s", apiResp.Status)
	}

	return parseLandAppraisals(apiResp.Data, area, city), nil
}

// parseLandAppraisals は XCT001 レスポンスを domain.LandAppraisalItem スライスに変換する。
// city が指定された場合は都道府県コード+市区町村コードの一致するレコードのみを返す。
func parseLandAppraisals(raw []LandAppraisalRaw, prefArea, city string) []domain.LandAppraisalItem {
	result := make([]domain.LandAppraisalItem, 0, len(raw))
	for _, r := range raw {
		// city フィルタ: "13101" のような5桁コードを prefCode+cityCode と照合
		if city != "" && (r.PrefCode+r.CityCode) != city {
			continue
		}

		year, _ := strconv.Atoi(r.Year)
		pricePerSqm := parseFloat(r.PricePerSqm)
		if pricePerSqm == 0 {
			pricePerSqm = parseFloat(r.AnnouncedPrice)
		}
		if pricePerSqm == 0 {
			continue
		}

		// 変動率は百分率文字列（例: "3.5" → 3.5%）→ 小数に変換
		changeRate := parseFloat(r.ChangeRate) / 100

		result = append(result, domain.LandAppraisalItem{
			Year:        year,
			PricePerSqm: pricePerSqm,
			ChangeRate:  changeRate,
			District:    r.DistrictName,
		})
	}
	return result
}

// doRequest は単一のHTTPリクエストを実行し、レスポンスをパースして返す
func (c *Client) doRequest(ctx context.Context, apiURL string) ([]domain.LandTransaction, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request build error: %w", err)
	}

	// 不動産情報ライブラリ API は Ocp-Apim-Subscription-Key ヘッダーによる認証が必要
	if c.apiKey != "" {
		req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		var apiResp APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, fmt.Errorf("JSON decode error: %w", err)
		}
		if apiResp.Status != "OK" {
			return nil, fmt.Errorf("API status: %s", apiResp.Status)
		}
		return parseTransactions(apiResp.Data), nil
	}

	// 4xx はクライアントエラーとしてマーク（リトライ不要）
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, &clientError{code: resp.StatusCode}
	}
	return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
}

// clientError は 4xx クライアントエラーを表す（リトライ不要を示す）
type clientError struct{ code int }

func (e *clientError) Error() string { return fmt.Sprintf("client error: HTTP %d", e.code) }

// isClientError は err が clientError かどうかを判定する
func isClientError(err error) bool {
	_, ok := err.(*clientError)
	return ok
}

// buildLandPricesURL は XIT001 のクエリURLを生成する
func (c *Client) buildLandPricesURL(q LandPriceQuery) (string, error) {
	if q.Area == "" {
		return "", fmt.Errorf("area is required")
	}
	if q.Year == 0 || q.Quarter == 0 || q.ToYear == 0 || q.ToQuarter == 0 {
		return "", fmt.Errorf("year, quarter, to_year, to_quarter are required")
	}
	if q.Quarter < 1 || q.Quarter > 4 || q.ToQuarter < 1 || q.ToQuarter > 4 {
		return "", fmt.Errorf("quarter must be between 1 and 4")
	}

	params := url.Values{}
	params.Set("area", q.Area)
	params.Set("year", strconv.Itoa(q.Year))
	params.Set("quarter", strconv.Itoa(q.Quarter))
	params.Set("toYear", strconv.Itoa(q.ToYear))
	params.Set("toQuarter", strconv.Itoa(q.ToQuarter))
	// 取引価格情報（01）を取得
	params.Set("priceClassification", "01")
	if q.City != "" {
		params.Set("city", q.City)
	}

	return c.baseURL + endpointLandPrices + "?" + params.Encode(), nil
}

// parseTransactions はAPIレスポンスを domain.LandTransaction スライスに変換する
// 土地(宅地)のみを対象とし、坪単価を算出する
func parseTransactions(raw []Transaction) []domain.LandTransaction {
	result := make([]domain.LandTransaction, 0, len(raw))
	for _, t := range raw {
		if !isLandType(t.Type) {
			continue
		}

		tradePrice := parseFloat(t.TradePrice)
		areaSqm := parseFloat(t.Area)
		pricePerSqm := parseFloat(t.PricePerUnit)

		// 単価が取れない場合は総額と面積から算出
		if pricePerSqm == 0 && areaSqm > 0 && tradePrice > 0 {
			pricePerSqm = tradePrice / areaSqm
		}

		pricePerTsubo := pricePerSqm * domain.SqmPerTsubo // 円/m² → 円/坪

		result = append(result, domain.LandTransaction{
			Period:           t.Period,
			District:         t.DistrictName,
			TradePrice:       tradePrice,
			Area:             areaSqm,
			PricePerSqm:      pricePerSqm,
			PricePerTsubo:    pricePerTsubo,
			CityPlanning:     t.CityPlanning,
			BuildingCoverage: t.BuildingCoverage,
			FloorAreaRatio:   t.FloorAreaRatio,
			BuildingYear:     parseJapaneseYear(t.BuildingYear),
			StationMinutes:   int(parseFloat(t.TimeToNearestStation)),
		})
	}
	return result
}

// parseJapaneseYear は国交省APIの建築年文字列を西暦年に変換する
// 例: "昭和63年" → 1988, "平成15年" → 2003, "令和5年" → 2023
func parseJapaneseYear(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "－" {
		return 0
	}

	eraMap := []struct {
		prefix string
		base   int
	}{
		{"令和", 2018},
		{"平成", 1988},
		{"昭和", 1925},
		{"大正", 1911},
		{"明治", 1867},
	}

	for _, e := range eraMap {
		if strings.HasPrefix(s, e.prefix) {
			numStr := strings.TrimPrefix(s, e.prefix)
			numStr = strings.TrimSuffix(numStr, "年")
			numStr = strings.TrimSpace(numStr)
			n, err := strconv.Atoi(numStr)
			if err != nil || n <= 0 {
				return 0
			}
			return e.base + n
		}
	}

	// 西暦形式 ("2023年" or "2023")
	numStr := strings.TrimSuffix(s, "年")
	n, err := strconv.Atoi(strings.TrimSpace(numStr))
	if err != nil || n < 1900 || n > 2100 {
		return 0
	}
	return n
}

// isLandType は取引種別が宅地(土地)かどうかを判定する
func isLandType(t string) bool {
	return strings.Contains(t, "宅地") && strings.Contains(t, "土地")
}

// parseFloat は国交省APIの文字列数値をfloat64にパースする
// カンマ区切りや全角文字に対応
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "－" || s == "-" {
		return 0
	}
	// カンマ除去
	s = strings.ReplaceAll(s, ",", "")
	// 全角数字→半角 (簡易)
	s = strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		return r
	}, s)
	// 「以上」「未満」などの不要な文字を取り除く
	for _, suffix := range []string{"以上", "未満", "m²", "㎡", "坪", "円"} {
		s = strings.ReplaceAll(s, suffix, "")
	}
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

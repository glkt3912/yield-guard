package mlit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

const (
	// 不動産情報ライブラリ API (2024年4月〜)
	mlitBaseURL    = "https://www.reinfolib.mlit.go.jp/ex-api/external/XIT001"
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
}

// NewClient は新しい Client を返す。
// 環境変数 MLIT_API_KEY からAPIキーを読み込む。
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		baseURL:    mlitBaseURL,
		apiKey:     os.Getenv("MLIT_API_KEY"),
	}
}

// FetchLandPrices は指定条件で土地取引価格を取得し、統計を返す。
// 一時的なネットワーク障害や 5xx レスポンスに対して指数バックオフでリトライする（ISSUE-13）。
func (c *Client) FetchLandPrices(ctx context.Context, q LandPriceQuery) ([]domain.LandTransaction, error) {
	apiURL, err := c.buildURL(q)
	if err != nil {
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

		result, err := c.doRequest(ctx, apiURL)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// クライアントエラー (4xx) はリトライしない
		if isClientError(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("API request failed after %d attempts: %w", maxRetries, lastErr)
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
	defer resp.Body.Close()

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

// buildURL はAPIのクエリURLを生成する
func (c *Client) buildURL(q LandPriceQuery) (string, error) {
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

	return c.baseURL + "?" + params.Encode(), nil
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
		})
	}
	return result
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

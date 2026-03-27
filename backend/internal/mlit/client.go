package mlit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

const (
	baseURL        = "https://www.land.mlit.go.jp/webland/api/TradeListSearch"
	landTypeFilter = "宅地(土地)"
	sqmPerTsubo    = 3.30578
	requestTimeout = 30 * time.Second
)

// Client は国交省 不動産取引価格情報取得APIのクライアント
type Client struct {
	httpClient *http.Client
}

// NewClient は新しい Client を返す
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

// FetchLandPrices は指定条件で土地取引価格を取得し、統計を返す
func (c *Client) FetchLandPrices(ctx context.Context, q LandPriceQuery) ([]domain.LandTransaction, error) {
	apiURL, err := buildURL(q)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request build error: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("JSON decode error: %w", err)
	}

	if apiResp.Status != "OK" {
		return nil, fmt.Errorf("API status: %s", apiResp.Status)
	}

	return parseTransactions(apiResp.Data), nil
}

// buildURL はAPIのクエリURLを生成する
func buildURL(q LandPriceQuery) (string, error) {
	if q.Area == "" {
		return "", fmt.Errorf("area is required")
	}
	if q.From == "" || q.To == "" {
		return "", fmt.Errorf("from and to are required")
	}

	params := url.Values{}
	params.Set("from", q.From)
	params.Set("to", q.To)
	params.Set("area", q.Area)
	if q.City != "" {
		params.Set("city", q.City)
	}

	return baseURL + "?" + params.Encode(), nil
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

		pricePerTsubo := pricePerSqm * sqmPerTsubo

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

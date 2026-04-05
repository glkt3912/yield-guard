# 国交省不動産取引価格APIクライアント仕様

`backend/internal/mlit/client.go` / `client_test.go` / `types.go`

---

## 利用API概要

| 項目 | 値 |
|------|-----|
| 正式名称 | 国土交通省 不動産取引価格情報取得API |
| エンドポイント | `https://www.land.mlit.go.jp/webland/api/TradeListSearch` |
| 認証 | 不要（公式オープンAPI） |
| タイムアウト | 30秒（`requestTimeout = 30 * time.Second`） |

---

## クエリパラメータ仕様

`LandPriceQuery` 構造体にマップされる。

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `area` | string | 必須 | 都道府県コード（`"01"`〜`"47"`）|
| `from` | string | 必須 | 開始時期（YYYYQ形式、Q=1〜4）|
| `to` | string | 必須 | 終了時期（YYYYQ形式）|
| `city` | string | 任意 | 市区町村コード（省略時は都道府県全体）|

**YYYYQ形式の例**:
- `"20231"` → 2023年第1四半期（1〜3月）
- `"20234"` → 2023年第4四半期（10〜12月）
- 2年分を取得する場合: `from="20221"`, `to="20234"`

---

## レスポンス形式（APIResponse）

```json
{
  "status": "OK",
  "data": [
    {
      "Type": "宅地(土地)",
      "TradePrice": "8,500,000",
      "Area": "100",
      "PricePerUnit": "85,000",
      "DistrictName": "南青山",
      "Period": "令和5年第1四半期",
      "CityPlanning": "第一種住居地域",
      "BuildingCoverage": "60",
      "FloorAreaRatio": "200"
    }
  ]
}
```

- `status` が `"OK"` 以外の場合はエラーとして扱う
- 数値は文字列形式（カンマ区切りや全角含む）で返ってくる

---

## 宅地フィルタリング（`isLandType`）

```go
func isLandType(t string) bool {
    return strings.Contains(t, "宅地") && strings.Contains(t, "土地")
}
```

`Type` フィールドが `"宅地(土地)"` のみを対象とする。
`"中古マンション等"`, `"農地"`, `"林地"` 等は除外される。

---

## 坪単価の算出

```go
pricePerSqm := parseFloat(t.PricePerUnit)

// 単価が取れない場合は総額÷面積から算出
if pricePerSqm == 0 && areaSqm > 0 && tradePrice > 0 {
    pricePerSqm = tradePrice / areaSqm
}

pricePerTsubo := pricePerSqm * domain.SqmPerTsubo  // × 3.30578
```

`SqmPerTsubo = 3.30578`（1坪 = 3.30578m²）

> **根拠・出典**: 1坪 = 6尺 × 6尺（江戸間）= 1.818m × 1.818m = **3.305785…m²**。計量法（昭和26年法律第207号）の付則では尺貫法の取引使用は禁止されているが、不動産業界では単価表示に慣習的に「坪」を使用。本ツールは `3.30578`（小数第5位まで）を採用。国土交通省の不動産情報ライブラリおよび業界団体でも同値を使用している。

---

## parseFloat 正規化

国交省APIの数値は文字列で返るため、以下の変換を適用する。

```go
func parseFloat(s string) float64 {
    // 1. 空文字・ダッシュ → 0
    // 2. カンマ除去 ("8,500,000" → "8500000")
    // 3. 全角数字→半角（"１２３" → "123"）
    // 4. 接尾辞除去（"以上", "未満", "m²", "㎡", "坪", "円"）
    // 5. strconv.ParseFloat
}
```

変換失敗（パースエラー）は `0` を返す。

---

## Client 構造体

```go
type Client struct {
    httpClient *http.Client
    baseURL    string  // デフォルト: mlitBaseURL（テスト時にモックサーバURLを注入可能）
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{Timeout: requestTimeout},
        baseURL:    mlitBaseURL,
    }
}
```

`baseURL` をフィールドとして持つことで、`httptest.NewServer` で立てたモックサーバを差し込んでテストできる。
`buildURL` は `Client` のメソッドとして実装されており、`c.baseURL` を参照してURLを生成する。

---

## 指数バックオフリトライ

```go
const (
    maxRetries     = 3
    retryBaseDelay = 1 * time.Second
)

for attempt := 0; attempt < maxRetries; attempt++ {
    delay := retryBaseDelay * time.Duration(1 << uint(attempt-1))
    // 1回目: 遅延なし
    // 2回目: 1秒待機
    // 3回目: 2秒待機
}
```

| 試行 | 待機 |
|------|------|
| 1回目 | 即時 |
| 2回目 | 1秒 |
| 3回目 | 2秒 |

- **4xx クライアントエラーはリトライしない**（`clientError` 型でマーク）
- `context.Done()` チェック付き（タイムアウト・キャンセル対応）
- 3回失敗後: `"API request failed after 3 attempts: <error>"` を返す

---

## 統計算出（`CalcLandPriceStats`）

`backend/internal/domain/investment.go` に実装。

```
平均: sum / len(prices)
中央値: n が偶数なら (n/2-1 + n/2) / 2、奇数なら n/2
```

- `PricePerTsubo == 0` のデータは統計から除外
- **`lowDataWarning = true`**: 有効データが 10件未満のとき

---

## 相場判定ロジック（`CompareLandPrice`）

```go
assessment := "相場"
if diffFromMedian > stats.MedianTsubo * 0.10 {
    assessment = "割高"
} else if diffFromMedian < -stats.MedianTsubo * 0.10 {
    assessment = "割安"
}
```

- 判定基準: 検討地の坪単価 vs 中央値の **±10%**
- `diffFromAverage` / `diffFromMedian`: プラスは「相場より高い」、マイナスは「相場より安い」

> **±10%閾値の設計根拠**: 本ツール独自の判定基準。不動産鑑定実務では「比準価格の採用差異が10%以内であれば合理的」とされる（不動産鑑定評価基準 各論第1章）ことを参考に設定した。また、同一地域内の取引価格ばらつき（変動係数）が一般に10〜30%程度であることから、±10%を「統計的に見た有意差の最小単位」として採用している。より精緻な判定が必要な場合は Z スコアや四分位範囲（IQR）を用いることを推奨する。

### 中央値を採用する理由

> 不動産取引価格データは**外れ値**（超高額・超低額物件）が混在しやすく、算術平均は外れ値に引っ張られるバイアスを持つ。国土交通省「不動産価格指数」（IPRI）も価格集計に中央値・トリム平均を採用している。本ツールでは `MedianTsubo`（中央値）を相場比較の基準とし、算術平均は参考値として表示する。

---

## 都道府県コードマップ（`mlit.Prefectures`）

47都道府県が `map[string]string` で定義されている。

```
"01" = 北海道, "13" = 東京都, "27" = 大阪府, "47" = 沖縄県
```

`GET /api/prefectures` はこのマップをコード昇順にソートして返す。

---

## テスト (`client_test.go`)

`net/http/httptest` のモックサーバを使い、実ネットワークなしで全ロジックを検証する。

| テスト | 内容 |
|--------|------|
| `TestParseFloat` | 全角数字・カンマ・接尾辞・空文字・浮動小数点・負数 |
| `TestIsLandType` | 宅地(土地) / 非土地 / 空文字 |
| `TestBuildURL` | 必須パラメータ欠落エラー・正常生成・cityオプション |
| `TestParseTransactions` | フィルタリング・単価算出・PricePerTsubo換算・空スライス |
| `TestFetchLandPrices_InvalidQuery` | buildURL エラーで HTTP リクエストが発生しないこと |
| `TestFetchLandPrices_RetryOn5xx` | 5xx → リトライ → 成功（3回目） |
| `TestFetchLandPrices_AllAttemptsFailWith5xx` | 3回連続5xx → エラー返却 |
| `TestFetchLandPrices_NoRetryOn4xx` | 4xx → リトライなし即エラー |
| `TestFetchLandPrices_ContextTimeout` | コンテキストタイムアウトでリトライ待機を中断 |
| `TestFetchLandPrices_APIStatusNotOK` | status!=OK → 3回リトライ後エラー |

```bash
cd backend
go test -race ./internal/mlit/... -v
```

# 国交省不動産取引価格APIクライアント仕様

`backend/internal/mlit/client.go` / `client_test.go` / `types.go`

---

## 利用API概要

| 項目 | 値 |
|------|-----|
| 正式名称 | 国土交通省 不動産情報ライブラリ API |
| エンドポイント | `https://www.reinfolib.mlit.go.jp/ex-api/external/XIT001` |
| 認証 | **APIキー必須**（`Ocp-Apim-Subscription-Key` ヘッダー） |
| 申請先 | https://www.reinfolib.mlit.go.jp/api/request/（審査5営業日） |
| タイムアウト | 30秒（`requestTimeout = 30 * time.Second`） |

> **移行経緯**: 旧 WebLand API（`www.land.mlit.go.jp/webland/api/`）は2024年4月に不動産情報ライブラリへ統合・廃止された。旧ドメインは現在 NXDOMAIN。

### 環境変数

```bash
MLIT_API_KEY=your_api_key_here   # .env.example 参照
```

---

## Client 構造体

```go
type Client struct {
    httpClient *http.Client
    baseURL    string  // デフォルト: mlitBaseURL（テスト時にモックサーバURLを注入可能）
    apiKey     string  // 環境変数 MLIT_API_KEY から読み込む
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{Timeout: requestTimeout},
        baseURL:    mlitBaseURL,
        apiKey:     os.Getenv("MLIT_API_KEY"),
    }
}
```

`baseURL` をフィールドとして持つことで、`httptest.NewServer` で立てたモックサーバを差し込んでテストできる。
`apiKey` が空の場合はヘッダーを付与しない（ローカル開発・テスト用）。

---

## クエリパラメータ仕様

`LandPriceQuery` 構造体にマップされる。

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `area` | string | 必須 | 都道府県コード（`"01"`〜`"47"`）|
| `Year` | int | 必須 | 取得開始年（例: `2024`）|
| `Quarter` | int | 必須 | 取得開始四半期（`1`〜`4`）|
| `ToYear` | int | 必須 | 取得終了年（例: `2024`）|
| `ToQuarter` | int | 必須 | 取得終了四半期（`1`〜`4`）|
| `City` | string | 任意 | 市区町村コード（省略時は都道府県全体）|

APIリクエストに変換されるクエリ文字列:

```
year=2024&quarter=1&toYear=2024&toQuarter=4&area=10&priceClassification=01
```

- `priceClassification=01`: 取引価格情報（成約価格は `02`。本ツールは取引価格のみ使用）

> **旧 API との変更点**: 旧 API は `from=20241&to=20244`（YYYYQ文字列）だったが、新 API は `year`/`quarter`/`toYear`/`toQuarter` の4パラメータに分割された。

---

## HTTPリクエスト仕様

```go
req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
```

APIキーが未設定（空文字）の場合はヘッダーを付与しない。
ユニットテストはモックサーバを使うためAPIキー不要。

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
- トップレベル構造は旧 API と互換

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
    // 1. 空文字 ("") → 0
    //    全角ダッシュ ("－") → 0（MLIT APIが「データなし」を示す文字）
    //    半角ダッシュ単体 ("-") → 0
    //    ※ "-100" のような負数はそのまま解析される（早期returnの対象外）
    // 2. カンマ除去 ("8,500,000" → "8500000")
    // 3. 全角数字→半角（"１２３" → "123"）
    // 4. 接尾辞除去（"以上", "未満", "m²", "㎡", "坪", "円"）
    // 5. strconv.ParseFloat
}
```

変換失敗（パースエラー）は `0` を返す。

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
- **`status != "OK"`（HTTP 200 だがAPIレベルのエラー）はリトライされる**（`clientError` に該当しないため）
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

---

## テスト (`client_test.go`)

`net/http/httptest` のモックサーバを使い、実ネットワークなしで全ロジックを検証する。

| テスト | 内容 |
|--------|------|
| `TestParseFloat` | 全角数字・カンマ・接尾辞・空文字・浮動小数点・負数 |
| `TestIsLandType` | 宅地(土地) / 非土地 / 空文字 |
| `TestBuildURL` | 必須パラメータ欠落エラー・quarter範囲外エラー・正常URL生成・cityオプション |
| `TestParseTransactions` | フィルタリング・単価算出・PricePerTsubo換算・空スライス |
| `TestFetchLandPrices_InvalidQuery` | buildURL エラーで HTTP リクエストが発生しないこと |
| `TestFetchLandPrices_RetryOn5xx` | 5xx → リトライ → 成功（3回目） |
| `TestFetchLandPrices_AllAttemptsFailWith5xx` | 3回連続5xx → エラー返却 |
| `TestFetchLandPrices_NoRetryOn4xx` | 4xx → リトライなし即エラー |
| `TestFetchLandPrices_ContextTimeout` | コンテキストタイムアウトでリトライ待機を中断 |
| `TestFetchLandPrices_APIStatusNotOK` | status!=OK → 3回リトライ後エラー |

```bash
# ユニットテスト（モックサーバ使用・APIキー不要）
cd backend
go test -race ./internal/mlit/... -v

# 統合テスト（実API疎通・APIキー必要）
MLIT_API_KEY=your_key go test -tags=integration ./internal/mlit/... -v -timeout 60s
```

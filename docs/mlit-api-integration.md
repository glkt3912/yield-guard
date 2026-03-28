# 国交省不動産取引価格APIクライアント仕様

`backend/internal/mlit/client.go` / `types.go`

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

---

## 都道府県コードマップ（`mlit.Prefectures`）

47都道府県が `map[string]string` で定義されている。

```
"01" = 北海道, "13" = 東京都, "27" = 大阪府, "47" = 沖縄県
```

`GET /api/prefectures` はこのマップをコード昇順にソートして返す。

# APIリファレンス

バックエンド: `backend/internal/api/handler.go` / `router.go`

## 共通仕様

| 項目 | 値 |
|------|-----|
| ベースURL | `http://localhost:8080` |
| リクエストボディ上限 | 64KB |
| レスポンス形式 | `application/json` |
| CORS 許可オリジン | 環境変数 `ALLOW_ORIGINS`（カンマ区切り）。未設定時は `http://localhost:3000` のみ |

### エラーレスポンス形式

```json
{ "error": "エラーメッセージ" }
```

---

## GET /api/land-prices

国交省APIから土地取引価格を取得し、統計を返す。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード（例: `"13"` = 東京都） |
| `year` | 必須 | 取得開始年（例: `2024`） |
| `quarter` | 必須 | 取得開始四半期（`1`〜`4`） |
| `to_year` | 必須 | 取得終了年（例: `2024`） |
| `to_quarter` | 必須 | 取得終了四半期（`1`〜`4`） |
| `city` | 任意 | 市区町村コード（例: `"13103"` = 港区） |

### レスポンス: `LandPriceStats`

```json
{
  "count": 42,
  "averageTsubo": 280000,
  "medianTsubo": 260000,
  "minTsubo": 150000,
  "maxTsubo": 500000,
  "transactions": [
    {
      "period": "令和5年第1四半期",
      "district": "青山",
      "tradePrice": 85000000,
      "area": 100.0,
      "pricePerSqm": 850000,
      "pricePerTsubo": 2810000,
      "cityPlanning": "第一種住居地域",
      "buildingCoverage": "60",
      "floorAreaRatio": "200"
    }
  ],
  "lowDataWarning": false,
  "warningMessage": ""
}
```

- `lowDataWarning: true`: 取引件数 < 10件のとき統計の信頼性が低い
- `warningMessage`: 件数不足時に具体的なメッセージを付与

### エラー

| コード | 条件 |
|--------|------|
| 400 | `area`, `year`, `quarter`, `to_year`, `to_quarter` のいずれかが未指定または範囲外 |
| 502 | 国交省APIへのリクエスト失敗 |

---

## GET /api/land-prices/compare

検討中の土地価格と相場を比較する。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード |
| `year` | 必須 | 取得開始年 |
| `quarter` | 必須 | 取得開始四半期（`1`〜`4`） |
| `to_year` | 必須 | 取得終了年 |
| `to_quarter` | 必須 | 取得終了四半期（`1`〜`4`） |
| `price` | 必須 | 検討中の土地価格（円、正の数値） |
| `city` | 任意 | 市区町村コード |
| `area_sqm` | 任意 | 土地面積（m²）。省略時は坪単価を 0 で比較 |

### レスポンス: `LandPriceComparison`

```json
{
  "stats": { /* LandPriceStats と同じ */ },
  "inputLandPrice": 5000000,
  "inputArea": 100.0,
  "inputPricePerTsubo": 165289,
  "diffFromAverage": -114711,
  "diffFromMedian": -94711,
  "assessment": "割安"
}
```

- `assessment`: `"割安"` / `"相場"` / `"割高"`
  - 判定基準: `inputPricePerTsubo` と `medianTsubo` の差が ±10% 以内 → `"相場"`
  - `+10%` 超 → `"割高"`, `-10%` 超（マイナス方向）→ `"割安"`
- `diffFromAverage` / `diffFromMedian`: プラスは「相場より高い」

### エラー

| コード | 条件 |
|--------|------|
| 400 | 必須パラメータ不足、または `price` が正の数値でない |
| 502 | 国交省APIへのリクエスト失敗 |

---

## POST /api/analyze

投資シミュレーションを実行する。

### リクエストボディ: `InvestmentInput`（JSON）

```json
{
  "landPrice": 5000000,
  "landArea": 100,
  "buildingCost": 10000000,
  "buildingAge": 0,
  "miscExpenseRate": 0.07,
  "monthlyRent": 120000,
  "vacancyRate": 0.05,
  "loanAmount": 13000000,
  "annualLoanRate": 0.015,
  "loanYears": 35,
  "buildingType": "木造",
  "expenseRate": 0.20,
  "incomeTaxRate": 0.33,
  "holdingYears": 10,
  "exitYieldTarget": 0.06,
  "vacancyRateDelta": 0,
  "loanRateDelta": 0
}
```

### バリデーション範囲

| フィールド | 制約 |
|-----------|------|
| `landPrice` | 1〜100億円 |
| `buildingCost` | 1〜100億円 |
| `monthlyRent` | 正の値 |
| `vacancyRate` | 0.0〜0.99 |
| `loanAmount` | 0以上 |
| `annualLoanRate` | 0〜30% |
| `loanYears` | 0〜50年 |
| `miscExpenseRate` | 0〜50% |
| `expenseRate` | 0〜90% |
| `incomeTaxRate` | 0〜60% |
| `exitYieldTarget` | 0%超〜50%（ゼロ除算防止） |
| `holdingYears` | 0〜50年 |

`buildingType` の有効値: `"木造"` / `"軽量鉄骨(4mm以下)"` / `"軽量鉄骨(3mm以下)"` / `"重量鉄骨"` / `"RC造"` / `"SRC造"`

`Defaults()` が適用される省略可能フィールド:
- `miscExpenseRate` 省略時 → `0.07`
- `holdingYears` 省略時 → `10`
- `exitYieldTarget` 省略時 → `0.06`
- `loanYears` 省略時 → `35`
- `buildingType` 省略時 → `"木造"`

### レスポンス: `InvestmentResult`

```json
{
  "totalInvestment": 16050000,
  "miscExpenses": 1050000,
  "grossYield": 0.0897,
  "netYield": 0.0673,
  "isAbove8Percent": true,
  "requiredCostReduction": 0,
  "requiredMonthlyRent": 107000,
  "deadCrossYear": 12,
  "yearlyResults": [/* YearlyResult × max(loanYears, holdingYears, 35) 件 */],
  "exitSalePrice": 21200000,
  "exitCapitalGain": 3500000,
  "exitTransferTax": 711025,
  "exitNetProceeds": 9750000,
  "exitTotalEquity": 12500000
}
```

### エラー

| コード | 条件 |
|--------|------|
| 400 | JSONパースエラー、バリデーションエラー |

---

## GET /api/prefectures

都道府県一覧をコード順（昇順）で返す。

### レスポンス

```json
[
  { "code": "01", "name": "北海道" },
  { "code": "02", "name": "青森県" },
  ...
  { "code": "47", "name": "沖縄県" }
]
```

47都道府県すべてを含む。コードは2桁ゼロパディング。

---

## GET /health

サーバー生存確認。

### レスポンス

```json
{ "status": "ok" }
```

---

## CORS 設定

`backend/internal/api/router.go`

```go
allowOrigins := os.Getenv("ALLOW_ORIGINS")
if allowOrigins == "" {
    allowOrigins = "http://localhost:3000"
}
```

- 環境変数 `ALLOW_ORIGINS` にカンマ区切りで複数オリジンを指定可能
- 許可メソッド: `GET`, `POST`, `OPTIONS`
- 許可ヘッダー: `Content-Type`, `Accept`
- `AllowCredentials: false`

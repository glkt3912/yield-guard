# APIリファレンス

バックエンド: `backend/internal/api/handler.go` / `router.go`

## 共通仕様

| 項目 | 値 |
|------|-----|
| ベースURL | `http://localhost:8080` |
| リクエストボディ上限 | 64KB |
| レスポンス形式 | `application/json` |
| CORS 許可オリジン | 環境変数 `ALLOW_ORIGINS`（カンマ区切り）。未設定時は `http://localhost:3000` のみ |

### 内部通信認証（`/api/*`）

環境変数 `APP_INTERNAL_API_KEY` が設定されている場合、`/api/*` エンドポイントへのすべてのリクエストに `X-Internal-Key` ヘッダーが必要。

| ヘッダー | 説明 |
|----------|------|
| `X-Internal-Key` | `APP_INTERNAL_API_KEY` と同一の値。不一致または未送信の場合は `401 Unauthorized` |

- `/health` エンドポイントはこの認証をスキップする
- `APP_INTERNAL_API_KEY` が未設定（ローカル開発）の場合はヘッダー検証をスキップし、従来通り動作する
- Vercel フロントエンドは `frontend/src/middleware.ts` で自動的にヘッダーを付与するため、ブラウザからの通常利用では意識不要

### エラーレスポンス形式

```json
{ "error": "エラーメッセージ" }
```

---

## GET /api/land-prices/stats

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
      "floorAreaRatio": "200",
      "buildingYear": 2005,
      "stationMinutes": 8
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

## GET /api/land-prices/estimate

築年数・駅距離・需要スコア補正による理論価格と販売価格乖離率を返す。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード |
| `year` | 必須 | 取得開始年 |
| `quarter` | 必須 | 取得開始四半期（`1`〜`4`） |
| `to_year` | 必須 | 取得終了年 |
| `to_quarter` | 必須 | 取得終了四半期（`1`〜`4`） |
| `price` | 必須 | 販売価格（円、正の数値） |
| `area_sqm` | 必須 | 土地面積（m²、正の数値） |
| `building_age` | 任意 | 物件築年数（省略時は 0） |
| `station_minutes` | 任意 | 最寄り駅徒歩分（省略または 0 で駅距離補正なし） |
| `ridership_score` | 任意 | 需要スコア（`A`〜`E`。省略で補正なし。`GET /api/station-ridership` で取得） |
| `city` | 任意 | 市区町村コード |

### 補正式

```
AgeCorrection        = clamp(-0.02 × (buildingAge - medianAge),     -0.30, +0.30)
StationCorrection    = clamp(-0.01 × (stationMin  - medianStation), -0.20, +0.20)
RidershipCorrection  = RidershipCorrectionFactor(ridership_score)  ← 省略時 0
TheoreticalPrice     = medianTsubo × (1+AgeCorr) × (1+StationCorr) × (1+RidershipCorr) × (area_sqm / 3.30578)
DeviationPct         = (price - TheoreticalPrice) / TheoreticalPrice × 100
```

中央値築年数・中央値駅距離は取引事例データから算出。

### レスポンス: `TheoreticalPriceResult`

```json
{
  "theoreticalPriceJPY": 4850000,
  "deviationPct": 3.1,
  "ageCorrection": 0.10,
  "stationCorrection": 0.05,
  "ridershipCorrection": 0.08,
  "medianBuildingAge": 18,
  "medianStationMinutes": 10,
  "isLowDataWarning": false,
  "hasStationData": true,
  "ridershipScore": "B",
  "hasRidershipData": true
}
```

- `deviationPct`: 正＝割高、負＝割安。±20%超で `LandPriceAnalysis` が強調表示する
- `isLowDataWarning`: 築年数データが10件未満のとき `true`（参考値扱い）
- `hasStationData`: `station_minutes` が指定かつ取引事例に駅距離データがある場合 `true`
- `ridershipScore`: `ridership_score` 指定時のみ含まれる（省略時はフィールドなし）
- `hasRidershipData`: `ridership_score` が指定された場合 `true`

### エラー

| コード | 条件 |
|--------|------|
| 400 | 必須パラメータ不足、または `ridership_score` が `A`〜`E` 以外の値 |
| 422 | 取引事例に築年数データがなく推定不可 |
| 502 | 国交省APIへのリクエスト失敗 |

---

## GET /api/station-ridership

物件の緯度経度からタイル座標を算出し、周辺駅の乗降客数と賃貸需要スコアを返す（国交省 XKT015）。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `lat` | 必須 | 緯度（-90〜90） |
| `lng` | 必須 | 経度（-180〜180） |
| `z` | 任意 | ズームレベル（11〜15、デフォルト: 14）。値が大きいほど狭域・詳細 |

バックエンドで緯度経度を WebMercator タイル座標（x, y）に変換し、XKT015 に `response_format=geojson&z=Z&x=X&y=Y` 形式でリクエストする。

### レスポンス: `StationRidershipResult[]`

```json
[
  {
    "stationName": "方南町",
    "lineName": "4号線丸ノ内線分岐線",
    "passengers": 38148,
    "demandScore": "B",
    "correction": 0.08
  },
  {
    "stationName": "永福町",
    "lineName": "井の頭線",
    "passengers": 29479,
    "demandScore": "B",
    "correction": 0.08
  }
]
```

乗降客数は最新年（2023年, `S12_057`）を優先し、欠損時は2011年（`S12_009`）を使用する。

### 需要スコア変換基準

| スコア | 乗降客数/日 | 補正係数 | 説明 |
|--------|------------|---------|------|
| A | ≥ 100,000人 | +0.15 | 超大型駅（渋谷・新宿・池袋等） |
| B | ≥ 30,000人 | +0.08 | 大型駅（地方政令市主要駅等） |
| C | ≥ 10,000人 | 0.00 | 中型駅（基準値） |
| D | ≥ 2,000人 | -0.08 | 小型駅 |
| E | < 2,000人 | -0.15 | 極小駅（地方小規模駅） |

- 結果は TTL 24時間でインメモリキャッシュされる（キー: `ridership:z:x:y`）
- `correction` の値を `GET /api/land-prices/estimate` の `ridership_score` パラメータに渡すことで理論価格に反映できる

### エラー

| コード | 条件 |
|--------|------|
| 400 | `lat` または `lng` が未指定・範囲外 |
| 400 | `z` が 11〜15 の範囲外 |
| 502 | 国交省APIへのリクエスト失敗 |

---

## GET /api/urban-risks

物件の緯度経度から都市計画リスクを一括取得する。XKT003（立地適正化計画）・XKT020（大規模盛土造成地）・XKT030（都市計画道路）・XST001（災害履歴）の 4 API を並列実行し、`UrbanRisk[]` を返す。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `lat` | 必須 | 緯度（20〜46: 日本国内） |
| `lng` | 必須 | 経度（122〜154: 日本国内） |

### レスポンス: `UrbanRisk[]`

```json
[
  {
    "code": "OUTSIDE_RESIDENTIAL_GUIDANCE",
    "level": "WARNING",
    "title": "居住誘導区域外の可能性",
    "description": "立地適正化計画上、居住誘導区域外である可能性があります。将来的なインフラ縮退リスクを確認してください。"
  },
  {
    "code": "URBAN_PLANNING_ROAD",
    "level": "WARNING",
    "title": "都市計画道路の区域内",
    "description": "都市計画道路の区域内に含まれる可能性があります。将来の道路拡幅で建物が移転対象になる場合があります。"
  }
]
```

### 検出リスクコード

| コード | API | 条件 | レベル |
|--------|-----|------|--------|
| `OUTSIDE_RESIDENTIAL_GUIDANCE` | XKT003 | 立地適正化計画データあり かつ 居住誘導区域外 | WARNING |
| `LARGE_EMBANKMENT` | XKT020 | 大規模盛土造成地データあり | WARNING |
| `URBAN_PLANNING_ROAD` | XKT030 | 都市計画道路（kubun_id=3011）データあり | WARNING |
| `DISASTER_HISTORY` | XST001 | 災害履歴データあり | WARNING |

### 実装詳細

- バックエンドで lat/lng を WebMercator タイル座標（z=14）に変換
- 4 API を goroutine で並列実行。個別 API の失敗はログのみ（他 API 結果は返す）
- 各タイルデータは TTL 24時間でインメモリキャッシュ（キー: `"{endpoint}:{z}:{x}:{y}"`）
- `detectUrbanRisks()`（XIT001 取引テキストから検出）とは独立したエンドポイント。フロントエンドで code 重複排除してマージする

### エラー

| コード | 条件 |
|--------|------|
| 400 | `lat` または `lng` が未指定・日本国外（20-46 / 122-154 範囲外） |
| 502 | 国交省 API へのリクエスト失敗 |

---

## POST /api/investment/analyze

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

## GET /api/population-forecast

物件の緯度経度からタイル座標を算出し、XKT013（将来推計人口250mメッシュ）を呼び出して30年後の人口変化率・空室率増加推定・人口動態トレンドを返す。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `lat` | 必須 | 緯度（-90〜90） |
| `lng` | 必須 | 経度（-180〜180） |
| `z` | 任意 | ズームレベル（11〜15、デフォルト: 14）。z=14 推奨（約1.7km×1.7km） |

バックエンドで緯度経度を WebMercator タイル座標に変換し、XKT013 に `response_format=geojson&z=Z&x=X&y=Y` 形式でリクエストする。タイル内の複数メッシュ（250m単位）の人口は合算して返す。

### レスポンス: `PopulationForecastResult`

```json
{
  "snapshots": [
    { "year": 2020, "pop": 11047.2 },
    { "year": 2025, "pop": 10698.8 },
    { "year": 2030, "pop": 10328.8 },
    { "year": 2035, "pop": 9941.1 },
    { "year": 2040, "pop": 9530.2 },
    { "year": 2045, "pop": 9106.2 },
    { "year": 2050, "pop": 8673.1 }
  ],
  "changeRate30yr": -0.215,
  "vacancyRateDelta": 0.107,
  "trend": "急激な減少"
}
```

| フィールド | 説明 |
|-----------|------|
| `snapshots` | 2020〜2050年（5年刻み）の推計人口。2020年は国勢調査基準値、2025〜は推計 |
| `changeRate30yr` | 2020→2050年の人口変化率（例: `-0.215` = 21.5%減） |
| `vacancyRateDelta` | 人口減少から推定される空室率増加幅: `max(0, -changeRate × 0.5)` |
| `trend` | 人口動向の4段階分類（下記参照） |

### トレンド分類基準

| `trend` | 変化率 |
|---------|--------|
| `"増加"` | > 0% |
| `"現状維持"` | -5%〜0% |
| `"緩やかな減少"` | -20%〜-5% |
| `"急激な減少"` | < -20% |

### 空室率増加推定式

```
vacancyRateDelta = max(0, -changeRate30yr × 0.5)
```

例: 30年で人口-32% → `vacancyRateDelta = 0.32 × 0.5 = 0.16`（空室率+16%pt）

- 結果は TTL 24時間でインメモリキャッシュされる（キー: `population:z:x:y`）
- XKT013 のフィールドは `PTN_2020`〜`PTN_2070`（5年刻み）。本エンドポイントは2020〜2050の7点を返す

### エラー

| コード | 条件 |
|--------|------|
| 400 | `lat` または `lng` が未指定・範囲外 |
| 400 | `z` が 11〜15 の範囲外 |
| 502 | 国交省APIへのリクエスト失敗 |

---

## GET /api/land-appraisals

国交省 XCT001（鑑定評価書情報）から地価公示データを取得し、取引価格との2軸比較統計を返す。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード（例: `"13"` = 東京都） |
| `year` | 必須 | 価格時点（2022〜2030） |
| `city` | 任意 | 市区町村コード（例: `"13101"` = 千代田区）。指定時はクライアントサイドでフィルタリング |
| `division` | 任意 | 用途区分（デフォルト: `"00"` = 住宅地）。`"05"` = 商業地 / `"07"` = 準工業地 / `"09"` = 工業地 |

### レスポンス: `AppraisalComparisonResult`

```json
{
  "appraisalMedianPerSqm": 1050000,
  "appraisalCount": 312,
  "appraisalTrend": 0.035,
  "trendLabel": "上昇"
}
```

| フィールド | 説明 |
|-----------|------|
| `appraisalMedianPerSqm` | 公示価格中央値（円/m²） |
| `appraisalCount` | 対象標準地点数 |
| `appraisalTrend` | 平均変動率（小数: 0.035 = +3.5%）。各地点の`変動率`フィールドを平均 |
| `trendLabel` | `"上昇"` / `"安定"` / `"下落"` |

### トレンド分類基準

| `trendLabel` | 変動率 |
|-------------|--------|
| `"上昇"` | > +3% |
| `"安定"` | -3%〜+3% |
| `"下落"` | < -3% |

### エラー

| コード | 条件 |
|--------|------|
| 400 | `area` が未指定 |
| 400 | `year` が2022〜2030の範囲外 |
| 422 | 指定エリアに地価公示データが存在しない |
| 502 | 国交省APIへのリクエスト失敗 |

- 結果は TTL 24時間でインメモリキャッシュされる（キー: `appraisals:area:city:year:division`）

---

## GET /api/municipalities

指定都道府県の市区町村一覧を返す（国交省 XIT002）。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード（例: `"13"` = 東京都） |

### レスポンス

```json
[
  { "id": "13101", "name": "千代田区" },
  { "id": "13102", "name": "中央区" }
]
```

- レスポンスは XIT002 が返す順序そのまま
- 結果は TTL 24時間でインメモリキャッシュされる

### エラー

| コード | 条件 |
|--------|------|
| 400 | `area` が未指定 |
| 502 | 国交省APIへのリクエスト失敗 |

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

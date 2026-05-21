---
purpose: 全 HTTP エンドポイントの詳細仕様（リクエスト/レスポンス/認証）
triggers: [api, endpoint, handler, request, response, auth]
audience: backend-dev, frontend-dev
token_weight: heavy
reads_next: [docs/llm/backend.md]
---

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

## GET /api/hazard

物件の緯度経度から洪水・高潮・津波・土砂災害の4種類のハザード情報を並列取得し、`UrbanRisk[]` 形式で返す（XKT026/027/028/029）。

### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `lat` | float | ○ | 緯度（20〜46: 日本国内） |
| `lng` | float | ○ | 経度（122〜154: 日本国内） |

バックエンドで緯度経度を WebMercator タイル座標（z=14）に変換し、4 API を並列実行する。

### レスポンス: `UrbanRisk[]`

```json
[
  {
    "code": "FLOOD_HIGH",
    "level": "WARNING",
    "title": "洪水浸水リスク（高）",
    "description": "洪水浸水想定区域（浸水深ランク3以上）に該当します。"
  },
  {
    "code": "LANDSLIDE_SPECIAL",
    "level": "WARNING",
    "title": "土砂災害特別警戒区域",
    "description": "土砂災害特別警戒区域（急傾斜地崩壊）に指定されています。"
  }
]
```

ハザード該当なしの場合は空配列 `[]` を返す。

### 検出リスクコード

#### 洪水（XKT026: `FetchFloodHazard`）

| コード | 条件（浸水深ランク） | レベル |
|--------|---------------------|--------|
| `FLOOD_LOW` | ランク 1〜2 | WARNING |
| `FLOOD_HIGH` | ランク 3 以上 | WARNING |

#### 高潮（XKT027: `FetchStormHazard`）

| コード | 条件 | レベル |
|--------|------|--------|
| `STORM_SURGE` | 高潮浸水データあり | WARNING |

#### 津波（XKT028: `FetchTsunamiHazard`）

| コード | 条件 | レベル |
|--------|------|--------|
| `TSUNAMI` | 津波浸水データあり | WARNING |

#### 土砂災害（XKT029: `FetchLandslideHazard`）

| コード | 条件 | レベル |
|--------|------|--------|
| `LANDSLIDE_WARNING` | 警戒区域（zoneCode=2）あり | WARNING |
| `LANDSLIDE_SPECIAL` | 特別警戒区域（zoneCode=1）あり | WARNING |

### 実装詳細

- 4 API を goroutine で並列実行。個別 API の失敗はログのみ（他 API 結果は返す）
- 各タイルデータは TTL 24時間でインメモリキャッシュ
- レートリミットなし（`generalRL` 20 req/s のみ適用）

### エラー

| コード | 条件 |
|--------|------|
| 400 | `lat` または `lng` が未指定 |
| 400 | `lat` が 20〜46 の範囲外、または `lng` が 122〜154 の範囲外 |

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
  "loanRateDelta": 0,
  "rentDeclineRate": 0.01,
  "loanMethod": "equal-payment"
}
```

### バリデーション範囲

| フィールド | 制約 |
|-----------|------|
| `landPrice` | 1〜100億円 |
| `buildingCost` | 1〜100億円（新築は建設費、中古は建物に帰属する取得費） |
| `rentDeclineRate` | 0.0〜0.2（年間賃料下落率。省略時は 0 = 下落なし） |
| `loanMethod` | `"equal-payment"`（元利均等）または `"equal-principal"`（元金均等）。省略時は `"equal-payment"` |
| `discountRate` | float64 | ✗ | 割引率（0–0.30）。0 は未指定扱い → Defaults() で 0.05 に補完 | 0.05 |
| `priceDeclineRate` | float64 | ✗ | 物件価格年間下落率（0–0.10）。IRR/NPVのターミナルバリューに適用 | 0 |
| `depreciationMethod` | string | ✗ | `"straight-line"` または `"declining-balance"` | `"straight-line"` |
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
- `loanMethod` 省略時 → `"equal-payment"`

クロスフィールドバリデーション:
- `vacancyRate + vacancyRateDelta` の合計が `0.99` を超える場合はエラー

### バリデーションエラーレスポンス

バリデーション違反時は `400 Bad Request` を返す。

```json
{
  "error": "rentDeclineRate は 0.0〜0.2 の範囲で指定してください"
}
```

`loanMethod` に無効な値を指定した場合もバリデーションエラー:

```json
{ "error": "loanMethod は equal-payment または equal-principal を指定してください" }
```

`buildingType` にアローリスト外の値を指定した場合もバリデーションエラー（プロンプトインジェクション防止のため、定義済み6種別のみ受け付ける）:

```json
{ "error": "buildingType が不正な値です" }
```

### レスポンス: `InvestmentResult`

```json
{
  "totalInvestment": 16050000,
  "miscExpenses": 1050000,
  "grossYield": 0.0897,
  "netYield": 0.0673,
  "isAboveYieldTarget": true,
  "yieldTarget": 0.08,
  "requiredCostReduction": 0,
  "requiredMonthlyRent": 107000,
  "deadCrossYear": 12,
  "yearlyResults": [/* YearlyResult × max(loanYears, holdingYears, 35) 件 */],
  "exitSalePrice": 21200000,
  "exitCapitalGain": 3500000,
  "exitTransferTax": 711025,
  "exitNetProceeds": 9750000,
  "exitTotalEquity": 12500000,
  "stressScenarios": [
    {
      "label": "ベースライン",
      "interestRateDelta": 0,
      "vacancyRateDelta": 0,
      "totalCashFlow": 3200000,
      "dscr": 1.25,
      "breakEvenYear": 4,
      "isSafe": true
    }
  ],
  "yieldScenarios": {
    "optimistic":  { "annualRent": 1368000, "grossYield": 0.0897 },
    "standard":    { "annualRent": 1296000, "grossYield": 0.0897 },
    "pessimistic": { "annualRent": 1224000, "grossYield": 0.0897 }
  },
  "dscr": 1.25,
  "ltvSensitivity": [
    { "ltv": 0.50, "equity": 8025000, "loanAmount": 8025000,  "dscr": 2.10, "annualCF": 500000, "cfYield": 0.031 },
    { "ltv": 0.60, "equity": 6420000, "loanAmount": 9630000,  "dscr": 1.60, "annualCF": 300000, "cfYield": 0.019 },
    { "ltv": 0.70, "equity": 4815000, "loanAmount": 11235000, "dscr": 1.20, "annualCF": 100000, "cfYield": 0.006 },
    { "ltv": 0.80, "equity": 3210000, "loanAmount": 12840000, "dscr": 0.90, "annualCF": -100000,"cfYield": -0.006 },
    { "ltv": 0.90, "equity": 1605000, "loanAmount": 14445000, "dscr": 0.60, "annualCF": -300000,"cfYield": -0.019 }
  ],
  "aiSummary": "木造（築10年）のため減価償却は残り12年で終了し、デッドクロスが12年目に到来する点に注意が必要です。DSCR1.25は安全域（1.2以上）を確保しており返済余力はありますが、金利上昇リスクに備えて10年以内の出口戦略を検討してください。表面利回り9.0%はエリア相場水準と比較して高めですが、木造の耐久性を踏まえると長期保有より中期売却が適切です。"
}
```

#### `stressScenarios: StressScenarioResult[]`

`Analyze()` が自動生成する 6 つのデフォルトシナリオの結果配列。入力に `vacancyRateDelta` または `loanRateDelta` が指定されている場合はカスタムシナリオが 7 本目として追加される。銀行提出資料や投資判断の感度分析に活用できる。

##### 自動生成シナリオ一覧

| `label` | `interestRateDelta` | `vacancyRateDelta` | 想定用途 |
|---------|--------------------|--------------------|----------|
| `"ベースライン"` | 0 | 0 | 入力条件そのままの基準値。他シナリオとの比較基点 |
| `"金利+1%"` | +0.01 | 0 | 金利上昇局面の軽微ストレス（変動金利リスク確認） |
| `"金利+2%"` | +0.02 | 0 | 金利急騰時の重大ストレス（銀行提出資料の標準感度） |
| `"空室+10%"` | 0 | +0.10 | 需要低下・競合増加による空室悪化シナリオ |
| `"空室+20%"` | 0 | +0.20 | 人口減少地域や築古物件の空室リスク最大値 |
| `"複合ストレス"` | +0.02 | +0.10 | 金利上昇＋空室悪化の同時発生（最悪ケース試算） |
| `"カスタム"` | `loanRateDelta` | `vacancyRateDelta` | リクエスト指定値による任意ストレス（任意追加） |

> カスタムシナリオは `vacancyRateDelta` または `loanRateDelta` のいずれかが非ゼロのときのみ追加される。

##### `StressScenarioResult` の構造

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `label` | string | シナリオ名（上表参照） |
| `interestRateDelta` | float64 | 基準金利からの上昇幅（例: `0.02` = +2%） |
| `vacancyRateDelta` | float64 | 基準空室率からの上昇幅（例: `0.10` = +10%） |
| `totalCashFlow` | float64 | 保有期間の税引後累積キャッシュフロー（円）。減価償却は省略した保守的近似 |
| `dscr` | float64 | 保有期間内の最悪年 DSCR（賃料下落率を年次適用した yearNOI / 年間ローン返済額）。ローンなしの場合は `0` |
| `breakEvenYear` | int | 累積**税引後**CF黒字転換年（保有期間内に未達なら `-1`） |
| `isSafe` | bool | 安全判定フラグ（下記参照） |

**DSCR（負債返済カバレッジ比率）の計算式:**

```
NOI(y) = 年間実効賃料 × (1 − RentDeclineRate)^(y−1) − 経費 − 固定資産税
DSCR   = min{ NOI(y) / 年間ローン返済額(y) }  // 保有期間内の最悪年
```

DSCR が 1.0 以上であれば NOI だけでローン返済を賄える状態（銀行融資審査の最低ライン）。  
実務基準は 1.2 以上。UI バッジは `≥1.2`: 緑（安全）/ `1.0〜1.2`: 黄（注意）/ `<1.0`: 赤（危険）で表示。

**`isSafe` の判定ロジック:**

| 条件 | 判定 |
|------|------|
| ローンあり: `DSCR >= 1.2` かつ `breakEvenYear` が保有期間以内 | `true` |
| ローンなし（`loanAmount = 0`）: `breakEvenYear` が保有期間以内 | `true` |
| 上記以外 | `false` |

| `irr` | float64\|null | 内部収益率（equity ≤ 0 または HoldingYears = 0 のとき null） |
| `npv` | float64 | 正味現在価値（円） |

#### `dscr: float64`

1年目の DSCR（Debt Service Coverage Ratio: 借入金償還余裕率）。`NOI / 年間ローン返済額（1年目）`。ローンなしの場合は `0`。`>= 1.0` で安全圏。

#### `ltvSensitivity: LTVSensitivityRow[]`

LTV を 50/60/70/80/90% に変化させたときの感度分析。常に5行返す。ベースケース（ストレスなし）で試算。元金均等の場合は1年目（最大）返済額を使用。

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `ltv` | float64 | 借入比率（例: 0.70） |
| `equity` | float64 | 自己資金（円） |
| `loanAmount` | float64 | 借入額（円） |
| `dscr` | float64 | DSCR |
| `annualCF` | float64 | 年間キャッシュフロー（円） |
| `cfYield` | float64 | CF利回り（`annualCF / 総投資額`） |

#### `yieldScenarios: YieldScenarios`

楽観・標準・悲観の 3 シナリオにおける年間実効賃料と表面利回りをバックエンドで計算して返す。

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `optimistic` | YieldScenario | 楽観シナリオ（空室率 × 0.5） |
| `standard` | YieldScenario | 標準シナリオ（空室率 × 1.0） |
| `pessimistic` | YieldScenario | 悲観シナリオ（空室率 × 1.5、上限 0.99 キャップ） |

`YieldScenario` の構造:

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `annualRent` | float64 | 年間実効賃料収入（空室控除後、円） |
| `grossYield` | float64 | 表面利回り（満室想定年収 / 総投資額。全シナリオ共通値） |

**注意**: `grossYield` はシナリオによらず満室想定で計算した固定値。シナリオ間で `annualRent` のみが変化する。

### エラー

| コード | 条件 |
|--------|------|
| 400 | JSONパースエラー、バリデーションエラー |

---

## GET /api/investment/rent-decline-hint

XCT001（国土交通省 地価公示）直近5年分のデータから賃料下落率の参考値を算出して返す。地価の CAGR（年平均成長率）を賃料下落の代理指標として使用する。

**レート制限**: `generalRL`（20 req/s）+ `analyzeRL`（6秒間隔）の二重制限を適用。

### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `area` | string | ○ | 都道府県コード（例: `"13"` = 東京都） |
| `municipality` | string | - | 市区町村コード（例: `"13101"` = 千代田区）。省略時はエリア全体で集計 |

### レスポンス: `RentDeclineHint`

```json
{
  "hintRate": 0.008,
  "basis": "land_appraisal",
  "dataPointCount": 312,
  "fallbackUsed": false,
  "note": "地価公示データに基づく参考値です。賃料下落率と完全には一致しません"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `hintRate` | float64 | 参考下落率（小数: `0.008` = 0.8%/年）。地価が上昇・横ばいの場合は `0` |
| `basis` | string | 算出根拠: `"land_appraisal"`（地価公示ベース）または `"fallback"`（データ不足・地価上昇） |
| `dataPointCount` | int | 使用した地価公示データ件数 |
| `fallbackUsed` | bool | フォールバック値を使用した場合は `true` |
| `note` | string | 補足メッセージ（フォールバック理由や注意点） |

### 算出ロジック

1. XCT001（住宅地: `division=00`）を 2022〜2026年の5年分並列取得
2. 取得できた年のデータを使い、各年の㎡単価中央値を算出
3. 最古年〜最新年の間で CAGR を計算:  
   `CAGR = (last_median / first_median)^(1/N) − 1`
4. CAGR < 0（地価下落）の場合のみ `hintRate = |CAGR|`、`basis = "land_appraisal"` を返す
5. 総データ件数 < 5 件、または取得できた年次が1年分のみの場合は `fallbackUsed: true` を返す

> `hintRate` は地価の変動率であり、賃料下落率と完全には一致しない。参考値として `POST /api/investment/analyze` の `rentDeclineRate` に活用できる。

### フォールバック条件

| 条件 | `basis` | `hintRate` |
|------|---------|-----------|
| 総データ件数 < 5 | `"fallback"` | `0` |
| 取得年数 < 2年 | `"fallback"` | `0` |
| 地価 CAGR ≥ 0（上昇・横ばい） | `"fallback"` | `0` |
| 地価 CAGR < 0（下落） | `"land_appraisal"` | `\|CAGR\|` |

### エラー

| コード | 条件 |
|--------|------|
| 400 | `area` が未指定 |
| 502 | 全年分の地価公示APIリクエストが失敗（一部失敗はスキップして継続） |

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

## POST /api/investment/simulate

モンテカルロ法による確率的ストレステストを実行する。空室率・金利を正規分布でサンプリングした N 試行を実施し、IRR・最終純資産の分布と統計量を返す。

### リクエストボディ: `MonteCarloInput`（JSON）

```json
{
  "base": { /* InvestmentInput と同一構造 */ },
  "simulations": 1000,
  "vacancyRateSigma": 0.05,
  "loanRateSigma": 0.005
}
```

| フィールド | 型 | デフォルト | 説明 |
|---|---|---|---|
| `base` | `InvestmentInput` | 必須 | `POST /api/investment/analyze` と同一の入力構造 |
| `simulations` | `int` | `1000` | 試行回数（最大 `10000`。超過時は自動クランプ） |
| `vacancyRateSigma` | `float64` | `0.05` | 空室率の正規分布標準偏差 |
| `loanRateSigma` | `float64` | `0.005` | 金利の正規分布標準偏差 |

`base` フィールドのバリデーションは `POST /api/investment/analyze` と同一ルールを適用。

### レスポンス: `MonteCarloResult`

```json
{
  "simulationCount": 1000,
  "irrPercentiles": {
    "p10": -0.012,
    "p25": 0.021,
    "p50": 0.048,
    "p75": 0.073,
    "p90": 0.098
  },
  "equityPercentiles": {
    "p10": 3200000,
    "p25": 5800000,
    "p50": 8500000,
    "p75": 11200000,
    "p90": 14100000
  },
  "deadCrossRate": 0.312,
  "irrHistogram": [
    { "min": -0.05, "max": -0.04, "count": 12 }
  ],
  "equityHistogram": [
    { "min": 2000000, "max": 3000000, "count": 8 }
  ],
  "successRate": 0.847
}
```

| フィールド | 説明 |
|---|---|
| `simulationCount` | 実施した試行回数 |
| `irrPercentiles` | IRR（内部収益率）の P10/P25/P50/P75/P90 |
| `equityPercentiles` | 最終純資産（売却手取り＋累積CF）の P10/P25/P50/P75/P90 |
| `deadCrossRate` | デッドクロスが発生した試行の割合（0〜1） |
| `irrHistogram` | IRR の度数分布（20ビン）。全試行の IRR が NaN の場合は `null` |
| `equityHistogram` | 最終純資産の度数分布（20ビン）。空の場合は `null` |
| `successRate` | IRR > 0 だった試行の割合（NaN 試行を分母から除外） |

**IRR 計算仕様**:
- CF 系列: `[-TotalInvestment, AfterTaxCF_1, ..., AfterTaxCF_N + ExitNetProceeds]`
- 二分法（[-99%, 1000%] 範囲・最大100回反復）で `NPV(r) = 0` となる `r` を求める
- 解が存在しない場合（全年 CF がマイナス等）は `NaN` として集計から除外

**サンプリング仕様**:
- Box-Muller 変換で正規分布 N(0, σ) からノイズを生成し、ベース値に加算
- 固定シード（`42`）により同一入力では常に同一の分布を返す（再現性保証）
- 空室率: `[0, 0.99]`、金利: `[0, 0.30]` にクランプ

### レートリミット

`POST /api/investment/analyze` と同一の二重レートリミットを適用（`generalRL` 20 req/s + `analyzeRL` 6秒間隔）。

---

## GET /api/investment-score

物件の緯度経度からタイル座標を算出し、複数の MLIT API を並列呼び出しして投資適地スコア（0〜100）を返す。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `lat` | 必須 | 緯度（20〜46: 日本国内） |
| `lng` | 必須 | 経度（122〜154: 日本国内） |

バックエンドで緯度経度を WebMercator タイル座標（z=14）に変換する。

### レスポンス: `InvestmentScoreResult`

```json
{
  "totalScore": 72,
  "grade": "良好",
  "breakdown": {
    "population":            { "score": 8,  "label": "人口動態",    "description": "30年で-15%の緩やかな減少" },
    "ridership":             { "score": 15, "label": "交通利便性",   "description": "方南町(B): 38,148人/日" },
    "urbanArea":             { "score": 10, "label": "市街化区域",   "description": "市街化区域内" },
    "locationOptimization":  { "score": 10, "label": "立地適正化",   "description": "居住誘導区域内" },
    "hazardRisk":            { "score": -5, "label": "ハザード",     "description": "洪水リスクあり（深さランク2）" },
    "liquefactionRisk":      { "score": 0,  "label": "液状化",      "description": "データなし" },
    "embankment":            { "score": 0,  "label": "盛土",        "description": "大規模盛土なし" },
    "disasterHistory":       { "score": 0,  "label": "災害履歴",    "description": "災害履歴なし" },
    "landPriceTrend":        { "score": 5,  "label": "地価トレンド", "description": "坪単価上昇（約+7.2%）" },
    "radarData": [
      { "category": "人口動態",  "score": 73 },
      { "category": "交通利便性", "score": 75 },
      { "category": "都市計画",  "score": 100 },
      { "category": "ハザード",  "score": 75 },
      { "category": "地盤",     "score": 50 }
    ]
  }
}
```

### スコア計算ロジック（`domain.CalcInvestmentScore`）

| 要素 | 範囲 | 計算根拠 |
|------|------|---------|
| 基礎点 | 50 | 中立値 |
| 人口動態 | ±15 | 30年変化率を線形補間（+30%→+15, 0%→0, -50%→-15） |
| 駅乗降客数 | 0〜+20 | 20万人/日以上→+20 を上限に線形スケール |
| 市街化区域 | 0 or +10 | 市街化区域内→+10 |
| 立地適正化 | -5〜+10 | 居住誘導区域内→+10、区域外→-5 |
| ハザードリスク | 0〜-20 | 洪水-3/-5・高潮/津波-5・土砂-3/-5 の合計 |
| 液状化リスク | 0〜-10 | 傾向区分（6段階）による |
| 大規模盛土 | 0 or -5 | 盛土造成地データあり→-5 |
| 災害履歴 | -10〜-2 | 10年以内:-10、30年以内:-5、30年超:-2、年不明:-10 |
| 地価トレンド | -10〜+10 | 坪単価2年比較: +10%超→+10、±5%以内→0、-10%超→-10 |

最終スコアは `[0, 100]` にクランプ。

### グレード基準

| グレード | スコア |
|---------|--------|
| 優良 | 80 以上 |
| 良好 | 65〜79 |
| 普通 | 50〜64 |
| 注意 | 35〜49 |
| 要注意 | 34 以下 |

### 実装詳細

- 複数 API（MLIT 系 11 本 + 地価トレンド 2 本）を goroutine で並列実行。個別 API の失敗はログのみ（部分スコアで返す）
- 地価トレンドはタイル中心座標→都道府県コードを境界ボックスで逆引きし、2023〜2024年と2021〜2022年の坪単価中央値を比較して変化率を算出
- 各タイルデータは TTL 24時間でインメモリキャッシュ（同都道府県の複数タイルは実質1回のみ API 呼び出し）
- コンテキストキャンセル時はエラーを返す

### エラー

| コード | 条件 |
|--------|------|
| 400 | `lat` または `lng` が未指定・日本国外 |
| 500 | コンテキストキャンセル |

---

## GET /api/investment-score-heatmap

バウンディングボックス内の全タイルに対して投資スコアを一括算出して返す。地図 viewport の投資適性をエリアで面的に把握するためのエンドポイント。

### クエリパラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `minLat` | 必須 | 緯度下限（20〜46） |
| `maxLat` | 必須 | 緯度上限（20〜46、minLat より大きい値） |
| `minLng` | 必須 | 経度下限（122〜154） |
| `maxLng` | 必須 | 経度上限（122〜154、minLng より大きい値） |
| `z` | 任意 | ズームレベル（11〜15、デフォルト: 13）。タイル上限はズームに応じて変化: z≤12: 20、z≤14: 30、z=15: 50 |

### レスポンス: `HeatmapResponse`

```json
{
  "tiles": [
    {
      "x": 7274, "y": 3225, "z": 13,
      "centerLat": 35.6851, "centerLng": 139.7715,
      "totalScore": 72,
      "grade": "良好"
    }
  ],
  "tileCount": 24
}
```

| フィールド | 説明 |
|-----------|------|
| `x`, `y`, `z` | Web Mercator タイル座標 |
| `centerLat`, `centerLng` | タイル中心の緯度経度（x+0.5, y+0.5 で算出） |
| `totalScore` | 投資適地スコア（0〜100） |
| `grade` | グレード（優良/良好/普通/注意/要注意） |
| `tileCount` | 実際に取得できたタイル数（失敗タイルを除く） |

### 実装詳細

- バックエンドでバウンディングボックスをタイル座標に変換し、矩形内の全タイルを列挙
- 各タイルで `calcScoreForTile`（MLIT 系 11 本 + 地価トレンド 2 本 = 最大 13 API 並列）を実行
- タイル並列数はセマフォ（バッファ=5）で制御 → 最大同時 MLIT リクエスト数: 5 × 13 = 65
- 個別タイルの失敗はスキップしてログのみ（部分結果を返す）
- コンテキストキャンセル時は goroutine が即座に終了
- パニック発生時も `recover` でキャッチして results チャネルに送信（デッドロック防止）
- 24時間 TTL キャッシュが効くため、同一範囲の2回目以降・同一都道府県の複数タイルはほぼゼロコスト

### レートリミット

`analyzeRL`（10 req/min, burst 5）を適用。通常エンドポイントより厳しい制限。

### エラー

| コード | 条件 |
|--------|------|
| 400 | 必須パラメータ未指定・範囲外・minLat ≥ maxLat など |
| 400 | タイル数が上限（z≤12: 20、z≤14: 30、z=15: 50）を超える場合 |

---

## GET /health

サーバー生存確認。

### レスポンス

```json
{ "status": "ok" }
```

---

---

## POST /api/renovation/analyze

リフォームROIシミュレーションを実行する。

**レート制限**: `generalRL`（60 req/分）のみ。

### リクエストボディ（`RenovationInput`）

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `propertyPrice` | float64 | ✓ | 物件取得価格（円、正値必須） |
| `annualBaseRent` | float64 | ✗ | リフォーム前年間家賃（円、≥0） |
| `annualExpenses` | float64 | ✗ | 年間経費（円、絶対額） |
| `effectiveTaxRate` | float64 | ✗ | 実効税率（0.0〜1.0） |
| `selfLaborRatePerHour` | float64 | ✗ | セルフリフォーム時給（円/時間、≥0） |
| `items` | RenovationItem[] | ✓ | 工事項目（1件以上）。各 `cost` は正値必須 |

**`RenovationItem`**:

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `name` | string | 部位名 |
| `cost` | float64 | 工事費（円、正値必須） |
| `expectedMonthlyRentIncrease` | float64 | 期待月額賃料アップ（円） |
| `isSelfWork` | bool | セルフリフォームか |
| `selfLaborHours` | float64 | 工数（時間） |

### レスポンス（`RenovationResult`）

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `recoveryYears` | float64 | 修繕費回収期間（年）。家賃アップなしは `0` |
| `isRecoverable` | bool | 回収可能か |
| `taxSavings` | float64 | 節税効果（円） |
| `virtualLaborCost` | float64 | セルフリフォーム仮想人件費合計（円） |
| `capitalExpenditures` | float64 | 資本的支出合計（60万円超） |
| `repairExpenses` | float64 | 修繕費合計（60万円以下） |
| `actualYield` | float64 | 実質利回り |
| `totalRenovationCost` | float64 | リフォーム総費用 |
| `annualRentIncrease` | float64 | 年間家賃アップ額 |
| `classifiedItems` | ClassifiedRenovationItem[] | 分類済み工事項目 |

**`ClassifiedRenovationItem`**: `RenovationItem` の全フィールド + `isCapitalExpenditure bool` + `virtualLaborCost float64`

### バリデーションエラー（400）

- `propertyPrice` ≤ 0
- `effectiveTaxRate` が 0–1 範囲外
- `selfLaborRatePerHour` < 0
- `items` が空
- いずれかの `item.cost` ≤ 0

---

## GET /api/rent-stats

XIT001（国交省 不動産取引価格情報: 賃貸）から直近2年分のデータを取得し、エリアの賃料相場（中央値・平均値・件数）を返す。

**レート制限**: `generalRL`（20 req/s）+ `analyzeRL`（6秒間隔）の二重制限を適用。

### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `area` | string | ○ | 都道府県コード（例: `"13"` = 東京都） |
| `municipality` | string | - | 市区町村コード（例: `"13101"` = 千代田区）。省略時はエリア全体で集計 |
| `area_sqm` | float | - | 物件専有面積（m²、正の値）。指定時は類似面積帯（±30%）に絞り込んで集計。省略時は全面積帯 |

バックエンドは現在日時から「直近2年分」の取引期間を自動決定する（`fromYear=現在年-2, quarter=1` 〜 `toYear=現在年, toQuarter=直前四半期`）。

### レスポンス: `RentStatsResult`

```json
{
  "median": 85000,
  "average": 89500,
  "count": 47
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `median` | float64 | 月額賃料 中央値（円） |
| `average` | float64 | 月額賃料 平均値（円） |
| `count` | int | 集計に使用したサンプル数 |

データなし（`count == 0`）の場合は `null` を返す（フロントエンドは非表示として扱う）。

### エラー

| コード | 条件 |
|--------|------|
| 400 | `area` が未指定 |
| 400 | `area_sqm` が正の数値として解釈できない、または 0 以下 |

> 国交省 API のリクエスト失敗や空レスポンスの場合は `400/502` ではなく `null` を返す（サイレント縮退）。

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

---

## GET /api/area-discovery

都道府県内の市区町村を土地価格データで評価し、投資有望エリアをランキング形式で返す。

**レート制限**: `analyzeRL`（10 req/min, burst 5）を適用。

### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `prefecture` | string | ○ | 都道府県コード（例: `"13"` = 東京都） |
| `budget` | float | - | 物件取得予算（円）。省略時は坪単価中央値×30坪+建物代1,000万円で試算 |
| `yield` | float | - | 目標表面利回り（例: `0.07` = 7%）。省略時は `0.08`（8%） |

### レスポンス: `AreaDiscoveryResponse`

```json
{
  "items": [
    {
      "municipalityCode": "13101",
      "municipalityName": "千代田区",
      "medianTsubo": 2500000,
      "transactionCount": 18,
      "yieldDifficulty": "achievable",
      "yieldDifficultyLabel": "達成可能",
      "landPriceTrend": "データなし",
      "dataSufficient": true
    }
  ],
  "prefecture": "13"
}
```

#### `AreaDiscoveryItem` フィールド

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `municipalityCode` | string | 市区町村コード |
| `municipalityName` | string | 市区町村名 |
| `medianTsubo` | float64 | 坪単価中央値（円） |
| `transactionCount` | int | 直近2年間の取引件数 |
| `yieldDifficulty` | string | 利回り達成難易度（下記参照） |
| `yieldDifficultyLabel` | string | 難易度の日本語ラベル |
| `landPriceTrend` | string | 地価トレンド（現在は常に `"データなし"`) |
| `dataSufficient` | bool | 取引件数が3件以上の場合 `true` |

#### `yieldDifficulty` 判定基準

目標利回りを達成するために必要な1坪あたり月額賃料（推定値）で判定する。

| 値 | 日本語ラベル | 条件（坪単価月額） |
|----|------------|----------------|
| `"achievable"` | 達成可能 | ≤ 8,000円/坪 |
| `"slightly-difficult"` | やや困難 | 8,001〜15,000円/坪 |
| `"difficult"` | 困難 | > 15,000円/坪 またはデータ不足 |

ソート順: `achievable` → `slightly-difficult` → `difficult`。同一難易度内は取引件数降順。

### 実装詳細

- 対象は上位30市区町村に絞る（タイムアウト防止）
- 各市区町村の直近2年間（現在年の2年前〜現在年）の土地取引データを並列取得（最大同時5件）
- 結果は TTL 24時間でインメモリキャッシュされる

### エラー

| コード | 説明 |
|--------|------|
| `400` | `prefecture` パラメータが未指定 |
| `500` | 市区町村一覧の取得失敗 |

### 実装

`backend/internal/api/handler.go` の `HandleAreaDiscovery` 関数。

---

## GET /api/geocode

住所文字列を緯度・経度に変換する（Google Maps Geocoding API 経由）。APIキーはサーバーサイドのみで保持し、フロントには露出しない。

### クエリパラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `address` | string | ○ | 変換対象の住所（例: `東京都渋谷区道玄坂1-1`） |

### レスポンス: `GeocodeResult`

```json
{
  "lat": 35.6595,
  "lng": 139.6984,
  "locationType": "ROOFTOP"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `lat` | float64 | 緯度 |
| `lng` | float64 | 経度 |
| `locationType` | string | Google Maps の精度種別（例: `"ROOFTOP"` / `"RANGE_INTERPOLATED"` / `"GEOMETRIC_CENTER"` / `"APPROXIMATE"`） |

### プライバシー保護

住所はPIIになりうるため、アクセスログには先頭10文字のみを記録し、残りは `***` でマスクする。

### エラー

| コード | 説明 |
|--------|------|
| `400` | `address` パラメータが空 |
| `400` | 該当住所なし（Google Maps から `ZERO_RESULTS`） |
| `503` | `GOOGLE_MAPS_API_KEY` が未設定（ジオコーディング無効） |
| `502` | Google Maps API へのリクエスト失敗 |

### 実装

`backend/internal/api/handler.go` の `GetGeocode` 関数 / `backend/internal/api/geocode_client.go` の `GoogleGeocodeClient`。

# 国交省不動産取引価格APIクライアント仕様

`backend/internal/mlit/client.go` / `client_test.go` / `types.go`

---

## 利用API概要

| 項目 | 値 |
|------|-----|
| 正式名称 | 国土交通省 不動産情報ライブラリ API |
| ベースURL | `https://www.reinfolib.mlit.go.jp/ex-api/external` |
| 認証 | **APIキー必須**（`Ocp-Apim-Subscription-Key` ヘッダー） |
| 申請先 | https://www.reinfolib.mlit.go.jp/api/request/（審査5営業日） |
| タイムアウト | 30秒（`requestTimeout = 30 * time.Second`） |
| APIキー状態 | **取得済み**（2026年4月申請・承認）|

> **移行経緯**: 旧 WebLand API（`www.land.mlit.go.jp/webland/api/`）は2024年4月に不動産情報ライブラリへ統合・廃止された。旧ドメインは現在 NXDOMAIN。

### 環境変数

```bash
MLIT_API_KEY=your_api_key_here   # .env / .env.example 参照（git管理外）
```

---

## 利用規約・クレジット表記（必須）

不動産情報ライブラリ利用規約（PDL1.0）に基づき、以下の表記が**義務付けられている**。

### データ出典表記

データを画面表示・帳票出力する際は必ず記載すること：

```
出典：国土交通省　不動産情報ライブラリ
```

加工・編集したデータを表示する場合は追加で作成者情報も明記する。

### APIサービス利用時の免責表記

APIを使用するサービスには以下の文言を表示すること：

```
このサービスは、国土交通省の不動産情報ライブラリのAPI機能を使用していますが、
提供情報の最新性、正確性、完全性等が保証されたものではありません
```

### データの性質・注意点

- 地価公示価格は基準日（1月1日）時点の価格であり、取引時点による変動は含まない
- 取引価格データは成約済み情報であり、すべての土地の現在価値を表すものではない
- 著作権は国土交通省に帰属する

---

## レート制限

明確な上限値は公開されていないが、以下の制約が存在する：

- 同一APIキーで基準期間内に多数のリクエストがあった場合、**アクセス制限が設けられる**
- 連続実行は避けることが推奨されている
- 対策として本プロジェクトでは指数バックオフリトライとインメモリキャッシュ（TTL 24時間）を実装済み

---

## 利用可能なAPIエンドポイント全一覧

本プロジェクトで現在使用しているのは **XIT001, XIT002, XKT015**。全31エンドポイントの活用方針を記載する。

### 価格情報

| API ID | 名称 | 本PJ活用方針 | issue |
|--------|------|------------|-------|
| **XIT001** | 不動産価格（取引価格・成約価格）情報取得 | ✅ 使用中（土地取引価格・相場判定） | — |
| **XIT002** | 都道府県内市区町村一覧取得 | ✅ 使用中（市区町村コードの動的取得） | #58 |
| **XCT001** | 鑑定評価書情報（地価公示・直近5年分） | 国の公式評価との2軸比較・トレンド算出 | #59 |
| XPT001 | 不動産価格ポイントAPI | 地図上への取引価格プロット表示 | 未定 |
| XPT002 | 地価公示・地価調査ポイントAPI（1995年〜） | 長期トレンド可視化・出口価格補正 | #59 |

### 都市計画情報

| API ID | 名称 | 本PJ活用方針 | issue |
|--------|------|------------|-------|
| **XKT001** | 都市計画区域/区域区分 | 市街化調整区域の警告表示 | #63 |
| **XKT002** | 用途地域 | 建ぺい率・容積率の自動入力、用途不整合警告 | #61 |
| **XKT003** | 立地適正化計画 | 居住誘導区域外の将来リスク警告 | ✅ #66 |
| XKT014 | 防火・準防火地域 | 建築制限の参考情報 | 未定 |
| XKT023 | 地区計画 | 建築制限の詳細確認 | 未定 |
| XKT024 | 高度利用地区 | 容積率ボーナスエリアの判定 | 未定 |
| **XKT030** | 都市計画道路 | 道路収用リスク・開発ポテンシャルの表示 | ✅ #66 |

### 周辺施設情報

| API ID | 名称 | 本PJ活用方針 | issue |
|--------|------|------------|-------|
| XKT004 | 小学校区 | ファミリー向け賃貸需要の参考指標 | 未定 |
| XKT005 | 中学校区 | 同上 | 未定 |
| XKT006 | 学校 | 周辺施設スコアの構成要素 | 未定 |
| XKT007 | 保育園・幼稚園等 | 周辺施設スコアの構成要素 | 未定 |
| XKT010 | 医療機関 | 高齢者向け施設周辺の賃貸需要判断 | 未定 |
| XKT011 | 福祉施設 | 同上 | 未定 |
| XKT017 | 図書館 | 周辺施設スコアの構成要素 | 未定 |
| XKT018 | 市区町村役場・集会施設等 | 生活利便性スコア | 未定 |
| XKT019 | 自然公園地域 | 建築制限エリアの参考情報 | 未定 |

### 人口情報

| API ID | 名称 | 本PJ活用方針 | issue |
|--------|------|------------|-------|
| **XKT013** | 将来推計人口（250mメッシュ） | ✅ 使用中（人口減少シナリオによるストレステスト自動生成・人口動態インジケーター） | #63 |
| **XKT015** | 駅別乗降客数 | ✅ 使用中（駅規模による賃貸需要スコア・理論価格補正） | #64 |
| XKT031 | 人口集中地区 | 都市度の判定・賃貸需要の参考指標 | 未定 |

### 防災情報

| API ID | 名称 | 本PJ活用方針 | issue |
|--------|------|------------|-------|
| **XKT016** | 災害危険区域 | 投資適地スコアのリスク要素 | #60 |
| **XKT020** | 大規模盛土造成地マップ | 地盤リスク（地震時の沈下・崩壊） | ✅ #66 |
| **XKT021** | 地すべり防止地区 | 地盤リスク | #60 |
| **XKT022** | 急傾斜地崩壊危険区域 | 地盤リスク | #60 |
| **XKT025** | 液状化発生傾向図 | 地盤リスク（#60のハザード警告に統合） | #60 |
| **XKT026** | 洪水浸水想定区域（想定最大規模） | 割安物件のリスク警告 | #60 |
| **XKT027** | 高潮浸水想定区域 | 同上 | #60 |
| **XKT028** | 津波浸水想定 | 同上 | #60 |
| **XKT029** | 土砂災害警戒区域 | 同上 | #60 |
| **XGT001** | 指定緊急避難場所 | 防災スコアの参考情報 | 未定 |
| **XST001** | 災害履歴 | 過去の実被害記録（ハザードマップと組み合わせ） | ✅ #66 |

### 投資適地スコア（複数API統合）

上記APIを組み合わせて物件の投資適性を数値化する差別化機能。

| 指標 | API | 配点 |
|---|---|---|
| 30年後人口変化率 | XKT013 | ±15点 |
| 駅乗降客数 | XKT015 | 0〜+20点 |
| 市街化区域・誘導区域内 | XKT001/XKT003 | +10点 or -5点 |
| ハザードリスク（洪水・津波・土砂・高潮） | XKT026〜029 | -0〜-20点 |
| 液状化・地盤リスク | XKT025/XKT020 | -0〜-10点 |
| 災害履歴あり | XST001 | -10点 or 0点 |

→ issue #63 で実装予定

---

## Client 構造体

```go
const (
    mlitBaseURL                = "https://www.reinfolib.mlit.go.jp/ex-api/external"
    endpointLandPrices         = "/XIT001"
    endpointMunicipalities     = "/XIT002"
    endpointStationRidership   = "/XKT015"
    endpointPopulationForecast = "/XKT013"
    endpointLandAppraisals     = "/XCT001"
)

type Client struct {
    httpClient *http.Client
    baseURL    string  // デフォルト: mlitBaseURL（テスト時にモックサーバURLを注入可能）
    apiKey     string  // 環境変数 MLIT_API_KEY から読み込む
    cache      *cache
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{Timeout: requestTimeout},
        baseURL:    mlitBaseURL,
        apiKey:     os.Getenv("MLIT_API_KEY"),
        cache:      newCache(),
    }
}
```

`baseURL` をフィールドとして持つことで、`httptest.NewServer` で立てたモックサーバを差し込んでテストできる。
`apiKey` が空の場合はヘッダーを付与しない（ローカル開発・テスト用）。

---

## XCT001 地価公示情報API

### エンドポイント

```
GET /ex-api/external/XCT001?area={都道府県コード}&year={年}&division={用途区分}
```

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード（例: `"13"` = 東京都）。カンマ区切りで複数指定可 |
| `year` | 必須 | 価格時点（2022〜2026） |
| `division` | 必須 | 用途区分。`00`=住宅地, `05`=商業地, `07`=準工業地, `09`=工業地 |

> XIT001/XIT002 と同形式のパラメータだが `city` パラメータは存在しない。市区町村絞り込みはクライアントサイドで行う。

### レスポンス形式

```json
{
  "status": "OK",
  "data": [
    {
      "価格時点": "2024",
      "標準地番号 市区町村コード 県コード": "13",
      "標準地番号 市区町村コード 市区町村コード": "101",
      "標準地番号 地域名": "千代田",
      "標準地番号 用途区分": "住宅地",
      "1㎡当たりの価格": "1200000",
      "公示価格": "1200000",
      "変動率": "3.5",
      "位置座標 緯度": "35.68950000",
      "位置座標 経度": "139.69050000"
    }
  ]
}
```

フィールド名はすべて日本語文字列（スペース含む）。型はすべて文字列形式で返る。

### 主要フィールド

| フィールド | 説明 |
|-----------|------|
| `"1㎡当たりの価格"` | 1㎡当たりの公示価格（円/m²）← 主要データ |
| `"公示価格"` | 正常価格（`1㎡当たりの価格`と通常同一） |
| `"変動率"` | 前年比変動率（百分率文字列: `"3.5"` = +3.5%） |
| `"標準地番号 市区町村コード 県コード"` | 都道府県コード（2桁） |
| `"標準地番号 市区町村コード 市区町村コード"` | 市区町村コード（3桁）。結合で5桁になる: `"13" + "101" = "13101"` |

### 型定義

```go
type LandAppraisalResponse struct {
    Status string             `json:"status"`
    Data   []LandAppraisalRaw `json:"data"`
}

type LandAppraisalRaw struct {
    Year           string `json:"価格時点"`
    PrefCode       string `json:"標準地番号 市区町村コード 県コード"`
    CityCode       string `json:"標準地番号 市区町村コード 市区町村コード"`
    DistrictName   string `json:"標準地番号 地域名"`
    UsageType      string `json:"標準地番号 用途区分"`
    PricePerSqm    string `json:"1㎡当たりの価格"`
    AnnouncedPrice string `json:"公示価格"`
    ChangeRate     string `json:"変動率"`
}
```

`domain.LandAppraisalItem`（`{Year int, PricePerSqm float64, ChangeRate float64, District string}`）はドメイン層で定義。

### FetchLandAppraisals シグネチャ

```go
func (c *Client) FetchLandAppraisals(ctx context.Context, area, city string, year int, division string) ([]domain.LandAppraisalItem, error)
```

- city が空でない場合: `prefCode + cityCode == city` の条件でクライアントサイドフィルタリング
- `変動率` 文字列は `parseFloat(s) / 100` で小数に変換（例: `"3.5"` → `0.035`）
- キャッシュキー: `"appraisals:{area}:{city}:{year}:{division}"`（TTL 24時間）

フロントエンドには `GET /api/land-appraisals?area=13&year=2024[&city=13101][&division=00]` として公開されている。

### ドメインロジック（`domain/land_appraisal.go`）

```go
func CalcAppraisalComparison(items []LandAppraisalItem) AppraisalComparisonResult
```

| 処理 | 計算式 |
|------|--------|
| 公示価格中央値 | `PricePerSqm` の中央値（ゼロ値を除外） |
| 平均変動率 | 全地点の `ChangeRate` の算術平均 |
| トレンド分類 | `> 3%`: 上昇 / `-3%〜3%`: 安定 / `< -3%`: 下落 |

**実測値（東京都・住宅地・2024年）**:

| area | division | 件数 | 公示価格中央値 |
|------|----------|------|-------------|
| 13（東京都） | 00（住宅地） | 3,394件 | ※実APIで確認 |

---

## XIT002 市区町村一覧API

### エンドポイント

```
GET /ex-api/external/XIT002?area={都道府県コード}
```

### パラメータ

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `area` | 必須 | 都道府県コード（`"01"`〜`"47"`） |

### レスポンス

```json
{
  "status": "OK",
  "data": [
    { "id": "13101", "name": "千代田区" },
    { "id": "13102", "name": "中央区" }
  ]
}
```

### バックエンドでの利用

`FetchMunicipalities(ctx, area)` を呼び出すと、XIT002 から市区町村一覧を取得して返す。
キャッシュ（TTL 24時間）に保存するため、同じ都道府県への2回目以降のリクエストはAPIコールをスキップする。

```go
func (c *Client) FetchMunicipalities(ctx context.Context, area string) ([]Municipality, error)
```

フロントエンドには `GET /api/municipalities?area=13` として公開されている（`handler.go` `GetMunicipalities`）。

---

## XKT015 駅別乗降客数API

### エンドポイント（タイル座標形式）

```
GET /ex-api/external/XKT015?response_format=geojson&z={z}&x={x}&y={y}
```

| パラメータ | 説明 |
|-----------|------|
| `response_format` | `geojson` 固定 |
| `z` | ズームレベル（11〜15）。z=14 を推奨（約1.7km×1.7km の範囲） |
| `x` / `y` | WebMercator タイル座標 |

> XIT001/XIT002 の `area`/`city` パラメータ形式とは異なる。PR #101 では誤って `area`/`city` 形式で実装していたが、PR #103 で修正した。

### 緯度経度→タイル座標変換（`LatLngToTile`）

```go
func LatLngToTile(lat, lng float64, z int) (x, y int) {
    n := math.Pow(2, float64(z))
    x = int(math.Floor((lng + 180.0) / 360.0 * n))
    latRad := lat * math.Pi / 180.0
    y = int(math.Floor((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n))
    return x, y
}
```

WebMercator（EPSG:3857）標準の変換式。`mlit` パッケージで公開されており、ハンドラ層から呼び出される。

### GeoJSONレスポンス形式

XKT015 は `FeatureCollection` を返す。各フィーチャのフィールド：

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `S12_001_ja` | string | 駅名 |
| `S12_002_ja` | string | 運営会社名 |
| `S12_003_ja` | string | 路線名 |
| `S12_001c` | string | 駅コード |
| `S12_009` | int | 乗降客数/日（2011年） |
| `S12_057` | int | 乗降客数/日（2023年・最新） |

年別乗降客数は4フィールド1組の構造（乗降客数+フラグ×3）になっており、`S12_009`=2011年、4ずつ増加して `S12_057`=2023年。`latestPassengers` で2023年を優先し、欠損時は2011年にフォールバックする。

`geometry.type` は `LineString`（線形フィーチャ）で返るため座標は使用しない。

### FetchStationRidership シグネチャ

```go
func (c *Client) FetchStationRidership(ctx context.Context, z, x, y int) ([]StationRidership, error)
```

フロントエンドには `GET /api/station-ridership?lat=35.6762&lng=139.6503[&z=14]` として公開されている。ハンドラ内で `LatLngToTile` によりタイル座標に変換してから呼び出す。

---

## XKT013 将来推計人口API

### エンドポイント（タイル座標形式）

```
GET /ex-api/external/XKT013?response_format=geojson&z={z}&x={x}&y={y}
```

XKT015 と同じタイル座標形式。z=14 で約1.7km×1.7km の範囲のメッシュデータを返す。

### GeoJSONレスポンス形式

1タイルに複数（数十〜100件以上）の250mメッシュフィーチャが含まれる。主要フィールド：

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `MESH_ID` | string | 地域メッシュコード（250mメッシュ） |
| `SHICODE` | string | 行政区域コード |
| `PTN_2020` | float | 2020年総人口（国勢調査基準値） |
| `PTN_2025` | float | 2025年推計総人口 |
| `PTN_2030` | float | 2030年推計総人口 |
| `PTN_2035` | float | 2035年推計総人口 |
| `PTN_2040` | float | 2040年推計総人口 |
| `PTN_2045` | float | 2045年推計総人口 |
| `PTN_2050` | float | 2050年推計総人口 |

実際には `PTN_2055`〜`PTN_2070` も存在するが、30年シナリオ（2020→2050）のみを使用する。また年齢別人口（`PT00_YYYY`〜`PT20_YYYY`）・年齢区分別（`PTA_YYYY`〜`PTE_YYYY`）・比率（`RTA_YYYY`〜`RTE_YYYY`）・非住宅割合（`HITOKU_YYYY`）なども含まれる。

出典: 国土数値情報「250mメッシュ別将来推計人口データ（R6国政局推計）」

### 型定義

```go
type PopulationForecastGeoJSON struct {
    Type     string                      `json:"type"`
    Features []PopulationForecastFeature `json:"features"`
}

type PopulationForecastProperties struct {
    MeshID  string  `json:"MESH_ID"`
    PTN2020 float64 `json:"PTN_2020"`
    PTN2025 float64 `json:"PTN_2025"`
    PTN2030 float64 `json:"PTN_2030"`
    PTN2035 float64 `json:"PTN_2035"`
    PTN2040 float64 `json:"PTN_2040"`
    PTN2045 float64 `json:"PTN_2045"`
    PTN2050 float64 `json:"PTN_2050"`
}
```

`domain.PopulationForecastItem`（`{Year int, Pop float64}`）はドメイン層で定義し、`mlit` → `domain` の循環インポートを防いでいる。

### 複数メッシュの集計

タイル内の全フィーチャ人口を年ごとに合算する（`parsePopulationForecasts`）。

```go
func (c *Client) FetchPopulationForecast(ctx context.Context, z, x, y int) ([]domain.PopulationForecastItem, error)
```

フロントエンドには `GET /api/population-forecast?lat=36.3906&lng=139.0608[&z=14]` として公開されている。

### ドメインロジック（`domain/population.go`）

```go
func CalcPopulationForecast(items []PopulationForecastItem) PopulationForecastResult
```

| 処理 | 計算式 |
|------|--------|
| 30年変化率 | `(PTN_2050 - PTN_2020) / PTN_2020` |
| 空室率増加推定 | `max(0, -changeRate × 0.5)` |
| トレンド分類 | `> 0`: 増加 / `-5%〜0`: 現状維持 / `-20%〜-5%`: 緩やかな減少 / `< -20%`: 急激な減少 |

**実測値（2026-04-19 統合テスト）**:

| エリア | 2020年人口 | 2050年人口 | 30年変化率 | 空室率増加推定 |
|--------|-----------|-----------|-----------|-------------|
| 渋谷付近（z=14, x=14547, y=6451） | 79,785人 | 84,132人 | **+5%** | 0% |
| 前橋市付近（z=14, x=14479, y=6412） | 1,898人 | 1,404人 | **-26%** | +13%pt |

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

## インメモリキャッシュ（`cache.go`）

`Client` はインスタンスごとに TTL 付きインメモリキャッシュを持ち、同一クエリの繰り返しリクエストでAPIコールをスキップする。

```go
const cacheTTL = 24 * time.Hour

type cacheEntry struct {
    data      []domain.LandTransaction
    expiresAt time.Time
}

type muniCacheEntry struct {
    data      []Municipality
    expiresAt time.Time
}

type cache struct {
    mu                 sync.RWMutex
    entries            map[string]cacheEntry            // XIT001 土地価格キャッシュ
    muniEntries        map[string]muniCacheEntry        // XIT002 市区町村キャッシュ（キー: 都道府県コード）
    ridershipEntries   map[string]ridershipCacheEntry   // XKT015 乗降客数キャッシュ（キー: "ridership:z:x:y"）
    populationEntries  map[string]populationCacheEntry  // XKT013 将来推計人口キャッシュ（キー: "population:z:x:y"）
}
```

### キャッシュキー

| キャッシュ種別 | キー形式 | 例 |
|---|---|---|
| 土地価格（XIT001） | `area:city:year:quarter:toYear:toQuarter` | `"13::2024:1:2024:4"` |
| 市区町村（XIT002） | 都道府県コード | `"13"` |
| 駅別乗降客数（XKT015） | `ridership:z:x:y` | `"ridership:14:14547:6451"` |
| 将来推計人口（XKT013） | `population:z:x:y` | `"population:14:14547:6451"` |

### 動作フロー

```
FetchLandPrices() 呼び出し
  ├─ キャッシュヒット（TTL内）→ コピーを返す（APIコールなし）
  └─ キャッシュミス / TTL切れ → API呼び出し → 成功時にキャッシュ保存

FetchMunicipalities() 呼び出し
  ├─ キャッシュヒット（TTL内）→ コピーを返す（APIコールなし）
  └─ キャッシュミス / TTL切れ → XIT002 呼び出し → 成功時にキャッシュ保存

FetchStationRidership() 呼び出し
  ├─ キャッシュヒット（TTL内）→ コピーを返す（APIコールなし）
  └─ キャッシュミス / TTL切れ → XKT015 呼び出し → 成功時にキャッシュ保存

FetchPopulationForecast() 呼び出し
  ├─ キャッシュヒット（TTL内）→ コピーを返す（APIコールなし）
  └─ キャッシュミス / TTL切れ → XKT013 呼び出し → 成功時にキャッシュ保存
```

### 設計上の判断

| 項目 | 内容 |
|------|------|
| **TTL: 24時間** | 土地価格データは四半期単位でしか更新されないため |
| **返却値はコピー** | 呼び出し元によるスライス変更でキャッシュが汚染されるバグを防ぐ |
| **Lazy Eviction** | TTL切れエントリは `get` アクセス時に削除。バックグラウンドGCゴルーチンは持たない |
| **サーバー再起動でリセット** | インメモリのため。永続化の複雑さを避けた割り切り |
| **API障害時の耐障害性** | TTL内であれば過去データを返し続けられる。ただし初回リクエスト時のAPI障害はカバーしない |

---

## 統計算出（`CalcLandPriceStats`）

`backend/internal/domain/investment.go` に実装。

```
平均: sum / len(prices)
中央値: n が偶数なら (n/2-1 + n/2) / 2、奇数なら n/2
```

- `PricePerTsubo == 0` のデータは統計から除外
- **`lowDataWarning = true`**: 有効データが 10件未満のとき

**用途地域サマリー抽出（`calcZoningSummary`）**:

`CalcLandPriceStats` 内で呼び出され、取引データから最頻の用途地域情報を抽出して `LandPriceStats.Zoning` に格納する。

```go
type ZoningSummary struct {
    CityPlanning     string  // 最頻の用途地域（例: 第一種住居地域）
    BuildingCoverage string  // 最頻の建ぺい率（例: 60%）
    FloorAreaRatio   string  // 最頻の容積率（例: 200%）
}
```

- `modalString()` で各フィールドの最頻値（空文字除外）を取得
- 3フィールドすべて空の場合は `nil` を返す（`json:"zoning,omitempty"` で出力省略）
- 同数タイの場合は Go の map イテレーション順に依存（非決定論的）

**都市計画リスク検出（`detectUrbanRisks`）**:

`CalcLandPriceStats` 内で `calcZoningSummary` の結果を受けて呼び出される。`LandPriceStats.UrbanRisks` に格納。

| コード | レベル | 検出条件 |
|--------|--------|---------|
| `URBANIZATION_CONTROL_ZONE` | ERROR | 最頻 CityPlanning が「市街化調整区域」を含む |
| `UNZONED_AREA` | WARNING | 最頻 CityPlanning が「非線引」含む または「都市計画区域外」 |
| `MIXED_ZONE_CAUTION` | WARNING | 最頻値が調整区域でなく、かつ取引の30%以上が市街化調整区域 |

- `zoning == nil`（取引データなし）の場合は空スライスを返す
- `UrbanRisks` は `[]UrbanRisk` 型。nil スライスは `json:"urbanRisks,omitempty"` で出力省略

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
| `TestBuildLandPricesURL` | 必須パラメータ欠落エラー・quarter範囲外エラー・正常URL生成・cityオプション |
| `TestParseTransactions` | フィルタリング・単価算出・PricePerTsubo換算・空スライス |
| `TestFetchLandPrices_InvalidQuery` | buildLandPricesURL エラーで HTTP リクエストが発生しないこと |
| `TestFetchLandPrices_RetryOn5xx` | 5xx → リトライ → 成功（3回目） |
| `TestFetchLandPrices_AllAttemptsFailWith5xx` | 3回連続5xx → エラー返却 |
| `TestFetchLandPrices_NoRetryOn4xx` | 4xx → リトライなし即エラー |
| `TestFetchLandPrices_ContextTimeout` | コンテキストタイムアウトでリトライ待機を中断 |
| `TestFetchLandPrices_APIStatusNotOK` | status!=OK → 3回リトライ後エラー |
| `TestFetchMunicipalities_Success` | XIT002 正常取得・パス/パラメータ検証 |
| `TestFetchMunicipalities_EmptyArea` | area 空文字でエラー（HTTPリクエストなし） |
| `TestFetchMunicipalities_CacheHit` | 2回目呼び出しがAPIコールなしでキャッシュから返る |
| `TestFetchMunicipalities_4xxNoRetry` | 4xx でエラー返却（リトライなし） |
| `TestLatLngToTile` | WebMercator 変換の期待タイル座標（渋谷付近・赤道・東経180度） |

```bash
# ユニットテスト（モックサーバ使用・APIキー不要）
cd backend
go test -race ./internal/mlit/... -v

# 統合テスト（実API疎通・APIキー必要）
MLIT_API_KEY=your_key go test -tags=integration ./internal/mlit/... -v -timeout 60s
```

---

## XKT003 立地適正化計画API

### エンドポイント（タイル座標形式）

```
GET /ex-api/external/XKT003?response_format=geojson&z={z}&x={x}&y={y}
```

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `response_format` | ○ | `"geojson"` 固定 |
| `z` | ○ | ズームレベル 11〜15。z=14 推奨（約1.7km×1.7km） |
| `x` / `y` | ○ | WebMercator タイル座標 |

### GeoJSONレスポンス フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| `prefecture` | string | 都道府県名 |
| `city_code` | string | 行政区域コード（5桁） |
| `city_name` | string | 市区町村名 |
| `decision_date` | string | 区域設定年月日 |
| `kubun_name_ja` | string | 区域名（例: 居住誘導区域、都市機能誘導区域） |
| `area_classification_ja` | string | 区域区分 |
| `notice_number` | string | 告示番号 |

### リスク判定方針

- タイル内フィーチャに `kubun_name_ja` が含まれる場合 → 立地適正化計画エリア内
- `kubun_name_ja` に「居住誘導区域」が含まれない、かつフィーチャが存在しない場合 → 居住誘導区域外（WARNING）
- フィーチャが0件 = 立地適正化計画未策定自治体（警告なし）

### 型定義

```go
type LocationOptimizationFeatureProps struct {
    Prefecture           string `json:"prefecture"`
    CityCode             string `json:"city_code"`
    CityName             string `json:"city_name"`
    DecisionDate         string `json:"decision_date"`
    KubunNameJa          string `json:"kubun_name_ja"`
    AreaClassificationJa string `json:"area_classification_ja"`
    NoticeNumber         string `json:"notice_number"`
}
```

---

## XKT020 大規模盛土造成地マップAPI

### エンドポイント（タイル座標形式）

```
GET /ex-api/external/XKT020?response_format=geojson&z={z}&x={x}&y={y}
```

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `response_format` | ○ | `"geojson"` 固定 |
| `z` | ○ | ズームレベル 11〜15 |
| `x` / `y` | ○ | WebMercator タイル座標 |

### GeoJSONレスポンス フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| `embankment_classification` | string | 盛土区分（例: 谷埋め型、腹付け型） |
| `prefecture_code` | string | 都道府県コード（例: 28） |
| `prefecture_name` | string | 都道府県名（例: 兵庫県） |
| `city_code` | string | 市区町村コード（例: 28215） |
| `city_name` | string | 市区町村名（例: 三木市） |
| `embankment_number` | string | 盛土番号（例: 三木08-03） |

### リスク判定方針

- タイル内フィーチャが1件以上 → 大規模盛土造成地エリア（WARNING）

### 型定義

```go
type EmbankmentFeatureProps struct {
    EmbankmentClassification string `json:"embankment_classification"`
    PrefectureCode           string `json:"prefecture_code"`
    PrefectureName           string `json:"prefecture_name"`
    CityCode                 string `json:"city_code"`
    CityName                 string `json:"city_name"`
    EmbankmentNumber         string `json:"embankment_number"`
}
```

---

## XKT030 都市計画道路API

### エンドポイント（タイル座標形式）

```
GET /ex-api/external/XKT030?response_format=geojson&z={z}&x={x}&y={y}
```

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `response_format` | ○ | `"geojson"` 固定 |
| `z` | ○ | ズームレベル 11〜15 |
| `x` / `y` | ○ | WebMercator タイル座標 |

### GeoJSONレスポンス フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| `planning_road_ja` | string | 都市計画道路種類名 |
| `kubun_id` | int | 区分コード（3011=都市計画道路、3023=広場） |
| `prefecture` | string | 都道府県名 |
| `city_code` | string | 市区町村コード |
| `city_name` | string | 市区町村名 |
| `first_decision_date` | string | 当初決定日 |
| `decision_date` | string | 設定年月日 |
| `decision_type_ja` | string | 設定区分名 |
| `decision_maker` | string | 設定者名 |
| `notice_number_s` | string | 告示番号（当初） |
| `notice_number` | string | 告示番号（最終） |

### リスク判定方針

- タイル内に `kubun_id == 3011`（都市計画道路）のフィーチャが1件以上 → 道路収用リスク（WARNING）

### 型定義

```go
type UrbanRoadFeatureProps struct {
    PlanningRoadJa    string `json:"planning_road_ja"`
    KubunID           int    `json:"kubun_id"`
    Prefecture        string `json:"prefecture"`
    CityCode          string `json:"city_code"`
    CityName          string `json:"city_name"`
    FirstDecisionDate string `json:"first_decision_date"`
    DecisionDate      string `json:"decision_date"`
    DecisionTypeJa    string `json:"decision_type_ja"`
    DecisionMaker     string `json:"decision_maker"`
    NoticeNumberS     string `json:"notice_number_s"`
    NoticeNumber      string `json:"notice_number"`
}
```

---

## XST001 国土調査（災害履歴）API

### エンドポイント（タイル座標形式）

```
GET /ex-api/external/XST001?response_format=geojson&z={z}&x={x}&y={y}[&disastertype_code={codes}]
```

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `response_format` | ○ | `"geojson"` 固定 |
| `z` | ○ | ズームレベル 9〜15 |
| `x` / `y` | ○ | WebMercator タイル座標 |
| `disastertype_code` | 任意 | 災害分類コード（カンマ区切りで複数指定可） |

### 災害分類コード一覧

| コード | 説明 |
|--------|------|
| 11 | 浸水域等 |
| 12 | 堤防決壊箇所等 |
| 13 | 高潮浸水域等 |
| 14 | 高潮破堤箇所等 |
| 21 | がけ崩れ等 |
| 22 | 地すべり等 |
| 23 | 河道閉塞箇所等 |
| 24 | 土石流等 |
| 33 | 液状化 |
| 34 | 地震土砂災害 |
| 37 | 津波高 |
| 38 | 津波浸水域 |

### GeoJSONレスポンス フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| `disastertype_code` | string | 災害分類コード（上表参照） |
| `disaster_name_ja` | string | 分類の呼称（例: 浸水域） |
| `disaster_date` | string | 西暦年月日（8桁）。不明部分は `0`（例: `19591100`） |
| `disaster_source` | string | 資料名（発行者） |

### リスク判定方針

- タイル内フィーチャが1件以上 → 災害履歴あり（WARNING）
- `disaster_name_ja` を列挙してリスク説明に含める
- `disaster_date` の上4桁（年）を取り出して表示

### 型定義

```go
type DisasterHistoryFeatureProps struct {
    DisastertypeCode string `json:"disastertype_code"`
    DisasterNameJa   string `json:"disaster_name_ja"`
    DisasterDate     string `json:"disaster_date"`
    DisasterSource   string `json:"disaster_source"`
}
```

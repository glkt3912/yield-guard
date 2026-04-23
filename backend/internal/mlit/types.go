package mlit

// APIResponse は国交省 不動産取引価格情報取得APIのレスポンス
type APIResponse struct {
	Status string        `json:"status"`
	Data   []Transaction `json:"data"`
}

// Transaction は個別の取引データ
// フィールド名は国交省API仕様に準拠
type Transaction struct {
	Type             string `json:"Type"`             // 取引種別 (宅地(土地), 中古マンション等...)
	Region           string `json:"Region"`           // 地域 (住宅地, 商業地...)
	MunicipalityCode string `json:"MunicipalityCode"` // 市区町村コード
	Prefecture       string `json:"Prefecture"`       // 都道府県名
	Municipality     string `json:"Municipality"`     // 市区町村名
	DistrictName     string `json:"DistrictName"`     // 地区名
	TradePrice       string `json:"TradePrice"`       // 取引価格（総額）
	PricePerUnit     string `json:"PricePerUnit"`     // 単価 (円/m²)
	FloorPlan        string `json:"FloorPlan"`        // 間取り
	Area             string `json:"Area"`             // 面積 (m²)
	UnitPrice        string `json:"UnitPrice"`        // 坪単価 (円/坪) ※APIによってはPricePerUnitと同一
	LandShape        string `json:"LandShape"`        // 土地の形状
	Frontage         string `json:"Frontage"`         // 間口 (m)
	TotalFloorArea   string `json:"TotalFloorArea"`   // 延床面積
	BuildingYear         string `json:"BuildingYear"`         // 建築年
	TimeToNearestStation string `json:"TimeToNearestStation"` // 最寄り駅距離 (分)
	Structure        string `json:"Structure"`        // 建物構造
	Use              string `json:"Use"`              // 用途
	Purpose          string `json:"Purpose"`          // 今後の利用目的
	Direction        string `json:"Direction"`        // 前面道路：方位
	Classification   string `json:"Classification"`   // 前面道路：種類
	Breadth          string `json:"Breadth"`          // 前面道路：幅員(m)
	CityPlanning     string `json:"CityPlanning"`     // 都市計画
	BuildingCoverage string `json:"BuildingCoverage"` // 建ぺい率 (%)
	FloorAreaRatio   string `json:"FloorAreaRatio"`   // 容積率 (%)
	Period           string `json:"Period"`           // 取引時期 (例: 令和5年第3四半期)
	Renovation       string `json:"Renovation"`       // 改装
	Remarks          string `json:"Remarks"`          // 取引の事情等
}

// MunicipalityResponse は XIT002 市区町村一覧APIのレスポンス
type MunicipalityResponse struct {
	Status string         `json:"status"`
	Data   []Municipality `json:"data"`
}

// Municipality は市区町村の1エントリ
type Municipality struct {
	ID   string `json:"id"`   // 市区町村コード (例: "13101")
	Name string `json:"name"` // 市区町村名 (例: "千代田区")
}

// LandPriceQuery は土地価格取得APIのクエリパラメータ
type LandPriceQuery struct {
	Area         string // 都道府県コード (例: "10" = 群馬県)
	City         string // 市区町村コード (例: "10201" = 前橋市)
	Year         int    // 取得年 (例: 2024)
	Quarter      int    // 取得四半期 (1〜4)
	ToYear       int    // 取得終了年 (例: 2024)
	ToQuarter    int    // 取得終了四半期 (1〜4)
}

// StationRidershipGeoJSON は XKT015 駅別乗降客数APIのGeoJSONレスポンス
type StationRidershipGeoJSON struct {
	Type     string                   `json:"type"`
	Features []StationRidershipFeature `json:"features"`
}

// StationRidershipFeature は GeoJSON の1フィーチャ（駅1件）
type StationRidershipFeature struct {
	Type       string                     `json:"type"`
	Properties StationRidershipProperties `json:"properties"`
	Geometry   StationRidershipGeometry   `json:"geometry"`
}

// StationRidershipProperties は駅別乗降客数の属性。フィールド名は国交省 XKT015 仕様に準拠（S12_XXX 形式）。
// 年別乗降客数フィールドは4フィールド1組: S12_009=2011年、S12_013=2012年、…、S12_057=2023年（最新）。
type StationRidershipProperties struct {
	StationCode  string `json:"S12_001c"`   // 駅コード
	StationName  string `json:"S12_001_ja"` // 駅名
	OperatorName string `json:"S12_002_ja"` // 運営会社名
	LineName     string `json:"S12_003_ja"` // 路線名
	// 年別乗降客数（整数値）。フィールドは +4 刻み: S12_009=2011〜S12_057=2023。
	P2011 int `json:"S12_009"`
	P2012 int `json:"S12_013"`
	P2013 int `json:"S12_017"`
	P2014 int `json:"S12_021"`
	P2015 int `json:"S12_025"`
	P2016 int `json:"S12_029"`
	P2017 int `json:"S12_033"`
	P2018 int `json:"S12_037"`
	P2019 int `json:"S12_041"`
	P2020 int `json:"S12_045"`
	P2021 int `json:"S12_049"`
	P2022 int `json:"S12_053"`
	P2023 int `json:"S12_057"`
}

// StationRidershipGeometry は GeoJSON Geometry（実際は LineString 形式で返る）。座標は使用しないため省略。
type StationRidershipGeometry struct {
	Type string `json:"type"`
}

// StationRidership はハンドラ層との橋渡し用の処理済みレコード
type StationRidership struct {
	StationName string `json:"stationName"`
	LineName    string `json:"lineName"`
	Passengers  int    `json:"passengers"` // 乗降客数/日
}

// PopulationForecastGeoJSON は XKT013 将来推計人口APIのGeoJSONレスポンス
type PopulationForecastGeoJSON struct {
	Type     string                      `json:"type"`
	Features []PopulationForecastFeature `json:"features"`
}

// PopulationForecastFeature は GeoJSON の1フィーチャ（250mメッシュ1件）
type PopulationForecastFeature struct {
	Properties PopulationForecastProperties `json:"properties"`
}

// PopulationForecastProperties は将来推計人口の属性。フィールド名は国交省 XKT013 仕様に準拠。
// PTN_YYYY は総人口（5年刻み: 2020〜2050）。
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

// LandAppraisalResponse は XCT001 鑑定評価書情報APIのレスポンス
type LandAppraisalResponse struct {
	Status string             `json:"status"`
	Data   []LandAppraisalRaw `json:"data"`
}

// LandAppraisalRaw は XCT001 の1レコード。フィールド名は国交省 XCT001 仕様に準拠（日本語キー）。
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

// LocationOptimizationGeoJSON は XKT003 立地適正化計画APIのGeoJSONレスポンス
type LocationOptimizationGeoJSON struct {
	Type     string                          `json:"type"`
	Features []LocationOptimizationFeature   `json:"features"`
}

// LocationOptimizationFeature は GeoJSON の1フィーチャ
type LocationOptimizationFeature struct {
	Properties LocationOptimizationProperties `json:"properties"`
}

// LocationOptimizationProperties は立地適正化計画の属性。フィールド名は国交省 XKT003 仕様に準拠。
type LocationOptimizationProperties struct {
	Prefecture           string `json:"prefecture"`
	CityCode             string `json:"city_code"`
	CityName             string `json:"city_name"`
	DecisionDate         string `json:"decision_date"`
	KubunNameJa          string `json:"kubun_name_ja"`           // 区域名（例: 居住誘導区域、都市機能誘導区域）
	AreaClassificationJa string `json:"area_classification_ja"`
	NoticeNumber         string `json:"notice_number"`
}

// EmbankmentGeoJSON は XKT020 大規模盛土造成地マップAPIのGeoJSONレスポンス
type EmbankmentGeoJSON struct {
	Type     string             `json:"type"`
	Features []EmbankmentFeature `json:"features"`
}

// EmbankmentFeature は GeoJSON の1フィーチャ
type EmbankmentFeature struct {
	Properties EmbankmentProperties `json:"properties"`
}

// EmbankmentProperties は大規模盛土造成地の属性。フィールド名は国交省 XKT020 仕様に準拠。
type EmbankmentProperties struct {
	EmbankmentClassification string `json:"embankment_classification"` // 盛土区分（例: 谷埋め型）
	PrefectureCode           string `json:"prefecture_code"`
	PrefectureName           string `json:"prefecture_name"`
	CityCode                 string `json:"city_code"`
	CityName                 string `json:"city_name"`
	EmbankmentNumber         string `json:"embankment_number"`
}

// UrbanRoadGeoJSON は XKT030 都市計画道路APIのGeoJSONレスポンス
type UrbanRoadGeoJSON struct {
	Type     string            `json:"type"`
	Features []UrbanRoadFeature `json:"features"`
}

// UrbanRoadFeature は GeoJSON の1フィーチャ
type UrbanRoadFeature struct {
	Properties UrbanRoadProperties `json:"properties"`
}

// UrbanRoadProperties は都市計画道路の属性。フィールド名は国交省 XKT030 仕様に準拠。
type UrbanRoadProperties struct {
	PlanningRoadJa    string `json:"planning_road_ja"`
	KubunID           int    `json:"kubun_id"`              // 3011=都市計画道路、3023=広場
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

// DisasterHistoryGeoJSON は XST001 国土調査（災害履歴）APIのGeoJSONレスポンス
type DisasterHistoryGeoJSON struct {
	Type     string                 `json:"type"`
	Features []DisasterHistoryFeature `json:"features"`
}

// DisasterHistoryFeature は GeoJSON の1フィーチャ
type DisasterHistoryFeature struct {
	Properties DisasterHistoryProperties `json:"properties"`
}

// DisasterHistoryProperties は災害履歴の属性。フィールド名は国交省 XST001 仕様に準拠。
type DisasterHistoryProperties struct {
	DisastertypeCode string `json:"disastertype_code"` // 11=浸水域, 21=がけ崩れ 等
	DisasterNameJa   string `json:"disaster_name_ja"`
	DisasterDate     string `json:"disaster_date"`     // 8桁 YYYYMMDD。不明部分は0
	DisasterSource   string `json:"disaster_source"`
}

// UrbanZoningGeoJSON は XKT001 都市計画区域/区域区分APIのGeoJSONレスポンス
type UrbanZoningGeoJSON struct {
	Type     string              `json:"type"`
	Features []UrbanZoningFeature `json:"features"`
}

// UrbanZoningFeature は GeoJSON の1フィーチャ
type UrbanZoningFeature struct {
	Properties UrbanZoningProperties `json:"properties"`
}

// UrbanZoningProperties は都市計画区域/区域区分の属性。フィールド名は国交省 XKT001 仕様に準拠。
type UrbanZoningProperties struct {
	AreaClassificationJa string `json:"area_classification_ja"` // 例: "市街化区域"、"市街化調整区域"
	KubunID              int    `json:"kubun_id"`
	Prefecture           string `json:"prefecture"`
	CityCode             string `json:"city_code"`
	CityName             string `json:"city_name"`
}

// LiquefactionGeoJSON は XKT025 液状化発生傾向図APIのGeoJSONレスポンス
type LiquefactionGeoJSON struct {
	Type     string               `json:"type"`
	Features []LiquefactionFeature `json:"features"`
}

// LiquefactionFeature は GeoJSON の1フィーチャ
type LiquefactionFeature struct {
	Properties LiquefactionProperties `json:"properties"`
}

// LiquefactionProperties は液状化発生傾向図の属性。フィールド名は国交省 XKT025 仕様に準拠。
type LiquefactionProperties struct {
	LiquefactionTendencyLevel int    `json:"liquefaction_tendency_level"` // 6段階（低値ほど液状化リスク高）
	Note                      string `json:"note"`                        // 例: "液状化しにくい"
	MeshCode                  string `json:"mesh_code"`
}

// FloodHazardGeoJSON は XKT026 洪水浸水想定区域APIのGeoJSONレスポンス
type FloodHazardGeoJSON struct {
	Type     string             `json:"type"`
	Features []FloodHazardFeature `json:"features"`
}

// FloodHazardFeature は GeoJSON の1フィーチャ
type FloodHazardFeature struct {
	Properties FloodHazardProperties `json:"properties"`
}

// FloodHazardProperties は洪水浸水想定区域の属性。フィールド名は国交省 XKT026 仕様に準拠。
type FloodHazardProperties struct {
	DepthRank    int    `json:"A31a_205"` // 浸水深ランク（高いほど深い）
	RiverName    string `json:"A31a_202"` // 河川名
	RiverManager string `json:"A31a_204"` // 河川管理者
}

// StormHazardGeoJSON は XKT027 高潮浸水想定区域APIのGeoJSONレスポンス
type StormHazardGeoJSON struct {
	Type     string              `json:"type"`
	Features []StormHazardFeature `json:"features"`
}

// StormHazardFeature は GeoJSON の1フィーチャ
type StormHazardFeature struct {
	Properties StormHazardProperties `json:"properties"`
}

// StormHazardProperties は高潮浸水想定区域の属性。フィールド名は国交省 XKT027 仕様に準拠。
type StormHazardProperties struct {
	DepthJa    string `json:"A49_003"`    // 浸水深区分（例: "5m以上10m未満"）
	Prefecture string `json:"A49_001"`    // 都道府県名
	TargetYear int    `json:"target_year"`
}

// TsunamiHazardGeoJSON は XKT028 津波浸水想定APIのGeoJSONレスポンス
type TsunamiHazardGeoJSON struct {
	Type     string               `json:"type"`
	Features []TsunamiHazardFeature `json:"features"`
}

// TsunamiHazardFeature は GeoJSON の1フィーチャ
type TsunamiHazardFeature struct {
	Properties TsunamiHazardProperties `json:"properties"`
}

// TsunamiHazardProperties は津波浸水想定の属性。フィールド名は国交省 XKT028 仕様に準拠。
type TsunamiHazardProperties struct {
	DepthJa    string `json:"A40_003"`    // 津波浸水深区分（例: "3m以上～5m未満"）
	Prefecture string `json:"A40_001"`    // 都道府県名
	TargetYear int    `json:"target_year"`
}

// LandslideHazardGeoJSON は XKT029 土砂災害警戒区域APIのGeoJSONレスポンス
type LandslideHazardGeoJSON struct {
	Type     string                 `json:"type"`
	Features []LandslideHazardFeature `json:"features"`
}

// LandslideHazardFeature は GeoJSON の1フィーチャ
type LandslideHazardFeature struct {
	Properties LandslideHazardProperties `json:"properties"`
}

// LandslideHazardProperties は土砂災害警戒区域の属性。フィールド名は国交省 XKT029 仕様に準拠。
type LandslideHazardProperties struct {
	PhenomenonType  int    `json:"A33_001"` // 現象種類（1=急傾斜地崩壊, 2=土石流, 3=地すべり）
	ZoneCode        int    `json:"A33_002"` // 区域区分（1=特別警戒区域, 2=警戒区域）
	PrefectureCode  string `json:"A33_003"` // 都道府県コード
	ZoneNumber      string `json:"A33_004"` // 区域番号
	SpecialZoneFlag int    `json:"A33_008"` // 特別警戒未指定フラグ
}

// Prefectures は都道府県コードマップ
var Prefectures = map[string]string{
	"01": "北海道", "02": "青森県", "03": "岩手県", "04": "宮城県",
	"05": "秋田県", "06": "山形県", "07": "福島県", "08": "茨城県",
	"09": "栃木県", "10": "群馬県", "11": "埼玉県", "12": "千葉県",
	"13": "東京都", "14": "神奈川県", "15": "新潟県", "16": "富山県",
	"17": "石川県", "18": "福井県", "19": "山梨県", "20": "長野県",
	"21": "岐阜県", "22": "静岡県", "23": "愛知県", "24": "三重県",
	"25": "滋賀県", "26": "京都府", "27": "大阪府", "28": "兵庫県",
	"29": "奈良県", "30": "和歌山県", "31": "鳥取県", "32": "島根県",
	"33": "岡山県", "34": "広島県", "35": "山口県", "36": "徳島県",
	"37": "香川県", "38": "愛媛県", "39": "高知県", "40": "福岡県",
	"41": "佐賀県", "42": "長崎県", "43": "熊本県", "44": "大分県",
	"45": "宮崎県", "46": "鹿児島県", "47": "沖縄県",
}

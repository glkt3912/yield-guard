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
	BuildingYear     string `json:"BuildingYear"`     // 建築年
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

// LandPriceQuery は土地価格取得APIのクエリパラメータ
type LandPriceQuery struct {
	Area         string // 都道府県コード (例: "10" = 群馬県)
	City         string // 市区町村コード (例: "10201" = 前橋市)
	Year         int    // 取得年 (例: 2024)
	Quarter      int    // 取得四半期 (1〜4)
	ToYear       int    // 取得終了年 (例: 2024)
	ToQuarter    int    // 取得終了四半期 (1〜4)
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

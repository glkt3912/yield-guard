package domain

// ZoningType は用途地域の種別（都市計画法第8条）
type ZoningType string

const (
	ZoningFirstLowRise        ZoningType = "第一種低層住居専用地域"
	ZoningSecondLowRise       ZoningType = "第二種低層住居専用地域"
	ZoningFirstMidHighRise    ZoningType = "第一種中高層住居専用地域"
	ZoningSecondMidHighRise   ZoningType = "第二種中高層住居専用地域"
	ZoningFirstResidential    ZoningType = "第一種住居地域"
	ZoningSecondResidential   ZoningType = "第二種住居地域"
	ZoningQuasiResidential    ZoningType = "準住居地域"
	ZoningNeighborhoodCommercial ZoningType = "近隣商業地域"
	ZoningCommercial          ZoningType = "商業地域"
	ZoningQuasiIndustrial     ZoningType = "準工業地域"
	ZoningIndustrial          ZoningType = "工業地域"
	ZoningExclusiveIndustrial ZoningType = "工業専用地域"
	ZoningGardenCity          ZoningType = "田園住居地域"
	ZoningUnknown             ZoningType = ""
)

// ZoningRiskLevel は用途地域の賃貸住宅投資リスクレベル
type ZoningRiskLevel int

const (
	ZoningRiskNone    ZoningRiskLevel = 0 // 問題なし
	ZoningRiskCaution ZoningRiskLevel = 1 // 注意（用途制限あり）
	ZoningRiskHigh    ZoningRiskLevel = 2 // 高リスク（住居不適）
)

// ZoningMeta は用途地域のメタデータ
type ZoningMeta struct {
	DefaultBuildingCoverage int             // デフォルト建ぺい率（%）
	DefaultFloorAreaRatio   int             // デフォルト容積率（%）
	RiskLevel               ZoningRiskLevel // 賃貸住宅投資リスク
	RiskMessage             string          // リスク説明
}

// zoningMetaMap は各用途地域のデフォルトメタデータ
// 建ぺい率・容積率は都市計画法の標準値（実際は自治体により異なる）
var zoningMetaMap = map[ZoningType]ZoningMeta{
	ZoningFirstLowRise:        {60, 100, ZoningRiskNone, ""},
	ZoningSecondLowRise:       {60, 150, ZoningRiskNone, ""},
	ZoningFirstMidHighRise:    {60, 200, ZoningRiskNone, ""},
	ZoningSecondMidHighRise:   {60, 200, ZoningRiskNone, ""},
	ZoningFirstResidential:    {60, 200, ZoningRiskNone, ""},
	ZoningSecondResidential:   {60, 200, ZoningRiskNone, ""},
	ZoningQuasiResidential:    {60, 200, ZoningRiskNone, ""},
	ZoningNeighborhoodCommercial: {80, 300, ZoningRiskCaution, "近隣商業地域: 日用品以外の店舗・工場が混在する場合があります"},
	ZoningCommercial:          {80, 400, ZoningRiskCaution, "商業地域: 風俗施設の立地が可能なため騒音・治安リスクがあります"},
	ZoningQuasiIndustrial:     {60, 200, ZoningRiskCaution, "準工業地域: 工場・倉庫が混在し騒音・臭気リスクがあります"},
	ZoningIndustrial:          {60, 200, ZoningRiskHigh, "工業地域: 住宅建設は可能ですが工場が多く賃貸需要が限定的です"},
	ZoningExclusiveIndustrial: {60, 200, ZoningRiskHigh, "工業専用地域: 住宅建設が法律上禁止されています"},
	ZoningGardenCity:          {50, 100, ZoningRiskCaution, "田園住居地域: 農地転用規制があり開発・増築に制約があります"},
}

// Meta は用途地域のメタデータを返す。不明な場合はゼロ値を返す
func (z ZoningType) Meta() ZoningMeta {
	return zoningMetaMap[z]
}

// IsValid は認識できる用途地域かどうかを返す
func (z ZoningType) IsValid() bool {
	if z == ZoningUnknown {
		return false
	}
	_, ok := zoningMetaMap[z]
	return ok
}

// ParseZoningType はMLIT APIが返す文字列をZoningTypeに変換する
// 認識できない場合はZoningUnknownを返す
func ParseZoningType(s string) ZoningType {
	z := ZoningType(s)
	if z.IsValid() {
		return z
	}
	return ZoningUnknown
}

// CityPlanningArea は都市計画区域の種別
type CityPlanningArea string

const (
	CityPlanningUrbanized        CityPlanningArea = "市街化区域"
	CityPlanningUrbanizationControlled CityPlanningArea = "市街化調整区域"
	CityPlanningUnzoned          CityPlanningArea = "非線引き区域"
	CityPlanningOutside          CityPlanningArea = "都市計画区域外"
	CityPlanningAreaUnknown      CityPlanningArea = ""
)

// ParseCityPlanningArea はMLIT APIが返す文字列をCityPlanningAreaに変換する
func ParseCityPlanningArea(s string) CityPlanningArea {
	switch CityPlanningArea(s) {
	case CityPlanningUrbanized, CityPlanningUrbanizationControlled, CityPlanningUnzoned, CityPlanningOutside:
		return CityPlanningArea(s)
	default:
		return CityPlanningAreaUnknown
	}
}

// IsHighRisk は市街化調整区域かどうかを返す（開発制限が厳しい）
func (c CityPlanningArea) IsHighRisk() bool {
	return c == CityPlanningUrbanizationControlled
}

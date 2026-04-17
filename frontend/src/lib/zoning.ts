export type ZoningRiskLevel = 0 | 1 | 2;

export interface ZoningMeta {
  defaultBuildingCoverage: number;
  defaultFloorAreaRatio: number;
  riskLevel: ZoningRiskLevel;
  riskMessage: string;
}

export const ZONING_TYPES = [
  "第一種低層住居専用地域",
  "第二種低層住居専用地域",
  "第一種中高層住居専用地域",
  "第二種中高層住居専用地域",
  "第一種住居地域",
  "第二種住居地域",
  "準住居地域",
  "近隣商業地域",
  "商業地域",
  "準工業地域",
  "工業地域",
  "工業専用地域",
  "田園住居地域",
] as const;

export type ZoningType = (typeof ZONING_TYPES)[number] | "";

export const ZONING_META: Record<string, ZoningMeta> = {
  第一種低層住居専用地域:    { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 100, riskLevel: 0, riskMessage: "" },
  第二種低層住居専用地域:    { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 150, riskLevel: 0, riskMessage: "" },
  第一種中高層住居専用地域:  { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 0, riskMessage: "" },
  第二種中高層住居専用地域:  { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 0, riskMessage: "" },
  第一種住居地域:            { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 0, riskMessage: "" },
  第二種住居地域:            { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 0, riskMessage: "" },
  準住居地域:                { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 0, riskMessage: "" },
  近隣商業地域:              { defaultBuildingCoverage: 80, defaultFloorAreaRatio: 300, riskLevel: 1, riskMessage: "日用品以外の店舗・工場が混在する場合があります" },
  商業地域:                  { defaultBuildingCoverage: 80, defaultFloorAreaRatio: 400, riskLevel: 1, riskMessage: "風俗施設の立地が可能なため騒音・治安リスクがあります" },
  準工業地域:                { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 1, riskMessage: "工場・倉庫が混在し騒音・臭気リスクがあります" },
  工業地域:                  { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 2, riskMessage: "工場が多く賃貸需要が限定的です。住宅建設は可能ですが将来空室リスクが高い" },
  工業専用地域:              { defaultBuildingCoverage: 60, defaultFloorAreaRatio: 200, riskLevel: 2, riskMessage: "住宅建設が法律上禁止されています" },
  田園住居地域:              { defaultBuildingCoverage: 50, defaultFloorAreaRatio: 100, riskLevel: 1, riskMessage: "農地転用規制があり開発・増築に制約があります" },
};

export type ZoningRiskLevel = 0 | 1 | 2 | 3;

export interface ZoningMeta {
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
  "市街化調整区域",
] as const;

export type ZoningType = (typeof ZONING_TYPES)[number] | "";

export const ZONING_META: Record<string, ZoningMeta> = {
  第一種低層住居専用地域: { riskLevel: 0, riskMessage: "" },
  第二種低層住居専用地域: { riskLevel: 0, riskMessage: "" },
  第一種中高層住居専用地域: { riskLevel: 0, riskMessage: "" },
  第二種中高層住居専用地域: { riskLevel: 0, riskMessage: "" },
  第一種住居地域: { riskLevel: 0, riskMessage: "" },
  第二種住居地域: { riskLevel: 0, riskMessage: "" },
  準住居地域: { riskLevel: 0, riskMessage: "" },
  田園住居地域: { riskLevel: 0, riskMessage: "" },
  近隣商業地域: { riskLevel: 1, riskMessage: "利便性は高いですが、騒音・治安リスクがあります" },
  商業地域: {
    riskLevel: 1,
    riskMessage: "利便性は高いですが、風俗施設の立地が可能なため騒音・治安リスクがあります",
  },
  準工業地域: { riskLevel: 2, riskMessage: "工場・倉庫が混在し、環境悪化の可能性があります" },
  工業地域: {
    riskLevel: 3,
    riskMessage: "周辺環境（騒音・臭気・トラック）リスクが高く、住宅需要の低下が見込まれます",
  },
  工業専用地域: {
    riskLevel: 3,
    riskMessage: "住宅建設が法律上禁止されており、賃貸アパートの建築は不可です",
  },
  市街化調整区域: {
    riskLevel: 3,
    riskMessage: "建築制限が厳しく、将来の再建築不可リスクがあります",
  },
};

import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceStats,
  LandPriceComparison,
  TheoreticalPriceResult,
  StationRidershipResult,
  RidershipDemandScore,
} from "@/types/investment";

const BASE = "/api";

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? "APIエラーが発生しました");
  }
  return res.json() as Promise<T>;
}

/** 土地取引価格の統計を取得 */
export async function fetchLandPrices(params: {
  area: string;
  city?: string;
  year: number;
  quarter: number;
  toYear: number;
  toQuarter: number;
}): Promise<LandPriceStats> {
  const q = new URLSearchParams({
    area: params.area,
    year: String(params.year),
    quarter: String(params.quarter),
    to_year: String(params.toYear),
    to_quarter: String(params.toQuarter),
  });
  if (params.city) q.set("city", params.city);
  const res = await fetch(`${BASE}/land-prices/stats?${q}`);
  return handleResponse<LandPriceStats>(res);
}

/** 検討中の土地価格と相場を比較 */
export async function compareLandPrice(params: {
  area: string;
  city?: string;
  year: number;
  quarter: number;
  toYear: number;
  toQuarter: number;
  price: number;
  areaSqm?: number;
}): Promise<LandPriceComparison> {
  const q = new URLSearchParams({
    area: params.area,
    year: String(params.year),
    quarter: String(params.quarter),
    to_year: String(params.toYear),
    to_quarter: String(params.toQuarter),
    price: String(params.price),
  });
  if (params.city) q.set("city", params.city);
  if (params.areaSqm) q.set("area_sqm", String(params.areaSqm));
  const res = await fetch(`${BASE}/land-prices/compare?${q}`);
  return handleResponse<LandPriceComparison>(res);
}

export interface Municipality {
  id: string;
  name: string;
}

/** 都道府県コードから市区町村一覧を取得（XIT002） */
export async function fetchMunicipalities(area: string): Promise<Municipality[]> {
  const res = await fetch(`${BASE}/municipalities?area=${encodeURIComponent(area)}`);
  return handleResponse<Municipality[]>(res);
}

/** 築年数・駅距離・乗降客数補正による理論価格推定 */
export async function estimateLandPrice(params: {
  area: string;
  city?: string;
  year: number;
  quarter: number;
  toYear: number;
  toQuarter: number;
  price: number;
  areaSqm: number;
  buildingAge: number;
  stationMinutes?: number;
  ridershipScore?: RidershipDemandScore;
}): Promise<TheoreticalPriceResult> {
  const q = new URLSearchParams({
    area: params.area,
    year: String(params.year),
    quarter: String(params.quarter),
    to_year: String(params.toYear),
    to_quarter: String(params.toQuarter),
    price: String(params.price),
    area_sqm: String(params.areaSqm),
    building_age: String(params.buildingAge),
  });
  if (params.city) q.set("city", params.city);
  if (params.stationMinutes) q.set("station_minutes", String(params.stationMinutes));
  if (params.ridershipScore) q.set("ridership_score", params.ridershipScore);
  const res = await fetch(`${BASE}/land-prices/estimate?${q}`);
  return handleResponse<TheoreticalPriceResult>(res);
}

/** 物件の緯度経度から周辺駅の乗降客数と需要スコアを取得（XKT015） */
export async function fetchStationRidership(params: {
  lat: number;
  lng: number;
  z?: number;
}): Promise<StationRidershipResult[]> {
  const q = new URLSearchParams({
    lat: String(params.lat),
    lng: String(params.lng),
  });
  if (params.z !== undefined) q.set("z", String(params.z));
  const res = await fetch(`${BASE}/station-ridership?${q}`);
  return handleResponse<StationRidershipResult[]>(res);
}

/** 投資シミュレーションを実行 */
export async function analyze(input: InvestmentInput): Promise<InvestmentResult> {
  const res = await fetch(`${BASE}/investment/analyze`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return handleResponse<InvestmentResult>(res);
}

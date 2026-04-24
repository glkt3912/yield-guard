import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceStats,
  LandPriceComparison,
  TheoreticalPriceResult,
  StationRidershipResult,
  RidershipDemandScore,
  PopulationForecastResult,
  AppraisalComparisonResult,
  UrbanRisk,
  RenovationInput,
  RenovationResult,
  MonteCarloInput,
  MonteCarloResult,
  InvestmentScoreResult,
  RentDeclineHint,
  HeatmapResponse,
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

/** 物件の緯度経度から将来推計人口と人口減少シナリオを取得（XKT013） */
export async function fetchPopulationForecast(params: {
  lat: number;
  lng: number;
  z?: number;
}): Promise<PopulationForecastResult> {
  const q = new URLSearchParams({
    lat: String(params.lat),
    lng: String(params.lng),
  });
  if (params.z !== undefined) q.set("z", String(params.z));
  const res = await fetch(`${BASE}/population-forecast?${q}`);
  return handleResponse<PopulationForecastResult>(res);
}

/** 地価公示情報を取得して取引価格との2軸比較統計を返す（XCT001） */
export async function fetchLandAppraisals(params: {
  area: string;
  year: number;
  city?: string;
  division?: string;
}): Promise<AppraisalComparisonResult> {
  const q = new URLSearchParams({
    area: params.area,
    year: String(params.year),
    division: params.division ?? "00",
  });
  if (params.city) q.set("city", params.city);
  const res = await fetch(`${BASE}/land-appraisals?${q}`);
  return handleResponse<AppraisalComparisonResult>(res);
}

/** 地価公示データから賃料下落率参考値を取得 */
export async function fetchRentDeclineHint(params: {
  area: string;
  municipality?: string;
}): Promise<RentDeclineHint> {
  const q = new URLSearchParams({ area: params.area });
  if (params.municipality) q.set("municipality", params.municipality);
  const res = await fetch(`${BASE}/investment/rent-decline-hint?${q}`);
  return handleResponse<RentDeclineHint>(res);
}

/** 緯度経度から都市計画リスクを一括取得 */
export async function fetchUrbanRisks(lat: number, lng: number): Promise<UrbanRisk[]> {
  const q = new URLSearchParams({ lat: String(lat), lng: String(lng) });
  const res = await fetch(`${BASE}/urban-risks?${q}`);
  return handleResponse<UrbanRisk[]>(res);
}

/** 緯度経度からハザード情報（洪水・高潮・津波・土砂災害）を取得 */
export async function fetchHazardInfo(lat: number, lng: number): Promise<UrbanRisk[]> {
  const q = new URLSearchParams({ lat: String(lat), lng: String(lng) });
  const res = await fetch(`${BASE}/hazard?${q}`);
  return handleResponse<UrbanRisk[]>(res);
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

/** リフォームROIシミュレーションを実行 */
export async function analyzeRenovation(input: RenovationInput): Promise<RenovationResult> {
  const res = await fetch(`${BASE}/renovation/analyze`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return handleResponse<RenovationResult>(res);
}

/** 投資適地スコアを取得 */
export async function fetchInvestmentScore(params: {
  lat: number;
  lng: number;
}): Promise<InvestmentScoreResult> {
  const p = new URLSearchParams({
    lat: String(params.lat),
    lng: String(params.lng),
  });
  const res = await fetch(`${BASE}/investment-score?${p}`);
  return handleResponse(res);
}

/** エリア内全タイルの投資スコアをまとめて取得 */
export async function fetchInvestmentScoreHeatmap(params: {
  minLat: number;
  maxLat: number;
  minLng: number;
  maxLng: number;
  z: number;
}): Promise<HeatmapResponse> {
  const p = new URLSearchParams({
    minLat: String(params.minLat),
    maxLat: String(params.maxLat),
    minLng: String(params.minLng),
    maxLng: String(params.maxLng),
    z: String(params.z),
  });
  const res = await fetch(`${BASE}/investment-score-heatmap?${p}`);
  return handleResponse(res);
}

export interface AreaDiscoveryItem {
  municipalityCode: string;
  municipalityName: string;
  medianTsubo: number;
  transactionCount: number;
  yieldDifficulty: "achievable" | "slightly-difficult" | "difficult";
  yieldDifficultyLabel: string;
  landPriceTrend: string;
  dataSufficient: boolean;
}

export interface AreaDiscoveryResponse {
  items: AreaDiscoveryItem[];
  prefecture: string;
}

/** エリア発見モード — 予算・目標利回りから候補エリアをランキング取得 */
export async function fetchAreaDiscovery(params: {
  prefecture: string;
  budget?: number;
  yield?: number;
}): Promise<AreaDiscoveryResponse> {
  const q = new URLSearchParams({ prefecture: params.prefecture });
  if (params.budget) q.set("budget", String(params.budget));
  if (params.yield) q.set("yield", String(params.yield));
  const res = await fetch(`${BASE}/area-discovery?${q}`);
  return handleResponse<AreaDiscoveryResponse>(res);
}

/** モンテカルロシミュレーションを実行 */
export async function simulate(input: MonteCarloInput): Promise<MonteCarloResult> {
  const res = await fetch(`${BASE}/investment/simulate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return handleResponse<MonteCarloResult>(res);
}

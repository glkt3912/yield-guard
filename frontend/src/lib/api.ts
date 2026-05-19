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
  GeocodeResult,
  Municipality,
} from "@/types/investment";
import { cachedFetch } from "./apiCache";

export type { Municipality };

const BASE = "/api";

const TTL_MARKET = 10 * 60 * 1000;
const TTL_RENT = 5 * 60 * 1000;

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? "APIエラーが発生しました");
  }
  return res.json() as Promise<T>;
}

async function postJson<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<T>(res);
}

function buildParams(
  required: Record<string, string | number>,
  optional?: Record<string, string | number | undefined>
): URLSearchParams {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(required)) q.set(k, String(v));
  if (optional) {
    for (const [k, v] of Object.entries(optional)) {
      if (v !== undefined) q.set(k, String(v));
    }
  }
  return q;
}

/** 土地取引価格の統計を取得 */
export function fetchLandPrices(params: {
  area: string;
  city?: string;
  year: number;
  quarter: number;
  toYear: number;
  toQuarter: number;
}): Promise<LandPriceStats> {
  const q = buildParams(
    {
      area: params.area,
      year: params.year,
      quarter: params.quarter,
      to_year: params.toYear,
      to_quarter: params.toQuarter,
    },
    { city: params.city }
  );
  return cachedFetch(`landPrices:${q}`, TTL_MARKET, async () => {
    const res = await fetch(`${BASE}/land-prices/stats?${q}`);
    return handleResponse<LandPriceStats>(res);
  });
}

/** 検討中の土地価格と相場を比較 */
export function compareLandPrice(params: {
  area: string;
  city?: string;
  year: number;
  quarter: number;
  toYear: number;
  toQuarter: number;
  price: number;
  areaSqm?: number;
}): Promise<LandPriceComparison> {
  const q = buildParams(
    {
      area: params.area,
      year: params.year,
      quarter: params.quarter,
      to_year: params.toYear,
      to_quarter: params.toQuarter,
      price: params.price,
    },
    { city: params.city, area_sqm: params.areaSqm }
  );
  return cachedFetch(`compareLandPrice:${q}`, TTL_RENT, async () => {
    const res = await fetch(`${BASE}/land-prices/compare?${q}`);
    return handleResponse<LandPriceComparison>(res);
  });
}

/** 都道府県コードから市区町村一覧を取得（XIT002） */
export function fetchMunicipalities(area: string): Promise<Municipality[]> {
  return cachedFetch(`municipalities:${area}`, TTL_MARKET, async () => {
    const res = await fetch(`${BASE}/municipalities?area=${encodeURIComponent(area)}`);
    return handleResponse<Municipality[]>(res);
  });
}

/** 築年数・駅距離・乗降客数補正による理論価格推定 */
export function estimateLandPrice(params: {
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
  const q = buildParams(
    {
      area: params.area,
      year: params.year,
      quarter: params.quarter,
      to_year: params.toYear,
      to_quarter: params.toQuarter,
      price: params.price,
      area_sqm: params.areaSqm,
      building_age: params.buildingAge,
    },
    {
      city: params.city,
      station_minutes: params.stationMinutes,
      ridership_score: params.ridershipScore,
    }
  );
  return cachedFetch(`estimateLandPrice:${q}`, TTL_RENT, async () => {
    const res = await fetch(`${BASE}/land-prices/estimate?${q}`);
    return handleResponse<TheoreticalPriceResult>(res);
  });
}

/** 物件の緯度経度から周辺駅の乗降客数と需要スコアを取得（XKT015） */
export function fetchStationRidership(params: {
  lat: number;
  lng: number;
  z?: number;
}): Promise<StationRidershipResult[]> {
  const z = params.z ?? "";
  return cachedFetch(`ridership:${params.lat}:${params.lng}:${z}`, TTL_MARKET, async () => {
    const q = new URLSearchParams({
      lat: String(params.lat),
      lng: String(params.lng),
    });
    if (params.z !== undefined) q.set("z", String(params.z));
    const res = await fetch(`${BASE}/station-ridership?${q}`);
    return handleResponse<StationRidershipResult[]>(res);
  });
}

/** 物件の緯度経度から将来推計人口と人口減少シナリオを取得（XKT013） */
export function fetchPopulationForecast(params: {
  lat: number;
  lng: number;
  z?: number;
}): Promise<PopulationForecastResult> {
  const z = params.z ?? "";
  return cachedFetch(`population:${params.lat}:${params.lng}:${z}`, TTL_MARKET, async () => {
    const q = new URLSearchParams({
      lat: String(params.lat),
      lng: String(params.lng),
    });
    if (params.z !== undefined) q.set("z", String(params.z));
    const res = await fetch(`${BASE}/population-forecast?${q}`);
    return handleResponse<PopulationForecastResult>(res);
  });
}

/** 地価公示情報を取得して取引価格との2軸比較統計を返す（XCT001） */
export function fetchLandAppraisals(params: {
  area: string;
  year: number;
  city?: string;
  division?: string;
}): Promise<AppraisalComparisonResult> {
  const city = params.city ?? "";
  const division = params.division ?? "00";
  return cachedFetch(
    `appraisals:${params.area}:${params.year}:${city}:${division}`,
    TTL_MARKET,
    async () => {
      const q = new URLSearchParams({
        area: params.area,
        year: String(params.year),
        division,
      });
      if (params.city) q.set("city", params.city);
      const res = await fetch(`${BASE}/land-appraisals?${q}`);
      return handleResponse<AppraisalComparisonResult>(res);
    }
  );
}

/** 地価公示データから賃料下落率参考値を取得 */
export function fetchRentDeclineHint(params: {
  area: string;
  municipality?: string;
}): Promise<RentDeclineHint> {
  const municipality = params.municipality ?? "";
  return cachedFetch(`rentDecline:${params.area}:${municipality}`, TTL_MARKET, async () => {
    const q = new URLSearchParams({ area: params.area });
    if (params.municipality) q.set("municipality", params.municipality);
    const res = await fetch(`${BASE}/investment/rent-decline-hint?${q}`);
    return handleResponse<RentDeclineHint>(res);
  });
}

/** 緯度経度から都市計画リスクを一括取得 */
export function fetchUrbanRisks(lat: number, lng: number): Promise<UrbanRisk[]> {
  return cachedFetch(`urbanRisks:${lat}:${lng}`, TTL_MARKET, async () => {
    const q = new URLSearchParams({ lat: String(lat), lng: String(lng) });
    const res = await fetch(`${BASE}/urban-risks?${q}`);
    return handleResponse<UrbanRisk[]>(res);
  });
}

/** 緯度経度からハザード情報（洪水・高潮・津波・土砂災害）を取得 */
export function fetchHazardInfo(lat: number, lng: number): Promise<UrbanRisk[]> {
  return cachedFetch(`hazard:${lat}:${lng}`, TTL_MARKET, async () => {
    const q = new URLSearchParams({ lat: String(lat), lng: String(lng) });
    const res = await fetch(`${BASE}/hazard?${q}`);
    return handleResponse<UrbanRisk[]>(res);
  });
}

/** 投資シミュレーションを実行 */
export async function analyze(input: InvestmentInput): Promise<InvestmentResult> {
  return postJson<InvestmentResult>("/investment/analyze", input);
}

/** リフォームROIシミュレーションを実行 */
export async function analyzeRenovation(input: RenovationInput): Promise<RenovationResult> {
  return postJson<RenovationResult>("/renovation/analyze", input);
}

/** 投資適地スコアを取得 */
export function fetchInvestmentScore(params: {
  lat: number;
  lng: number;
}): Promise<InvestmentScoreResult> {
  return cachedFetch(`score:${params.lat}:${params.lng}`, TTL_RENT, async () => {
    const p = new URLSearchParams({
      lat: String(params.lat),
      lng: String(params.lng),
    });
    const res = await fetch(`${BASE}/investment-score?${p}`);
    return handleResponse(res);
  });
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
export function fetchAreaDiscovery(params: {
  prefecture: string;
  budget?: number;
  yield?: number;
}): Promise<AreaDiscoveryResponse> {
  const budget = params.budget ?? "";
  const yieldRate = params.yield ?? "";
  return cachedFetch(
    `areaDiscovery:${params.prefecture}:${budget}:${yieldRate}`,
    TTL_MARKET,
    async () => {
      const q = new URLSearchParams({ prefecture: params.prefecture });
      if (params.budget) q.set("budget", String(params.budget));
      if (params.yield) q.set("yield", String(params.yield));
      const res = await fetch(`${BASE}/area-discovery?${q}`);
      return handleResponse<AreaDiscoveryResponse>(res);
    }
  );
}

/** モンテカルロシミュレーションを実行 */
export async function simulate(input: MonteCarloInput): Promise<MonteCarloResult> {
  return postJson<MonteCarloResult>("/investment/simulate", input);
}

/** 住所文字列から緯度・経度を取得（バックエンド経由でGoogle Maps Geocoding API を呼び出す） */
export function fetchGeocode(address: string): Promise<GeocodeResult> {
  return cachedFetch(`geocode:${address}`, TTL_MARKET, async () => {
    const q = new URLSearchParams({ address });
    const res = await fetch(`${BASE}/geocode?${q}`);
    return handleResponse<GeocodeResult>(res);
  });
}

export interface RentStatsResult {
  median: number;
  average: number;
  count: number;
  low_confidence?: boolean;
}

/** エリア賃料相場（中央値・平均値・件数）を取得（XIT001 賃貸） */
export function fetchRentStats(params: {
  area: string;
  municipality?: string;
  areaSqm?: number;
}): Promise<RentStatsResult | null> {
  const municipality = params.municipality ?? "";
  const areaSqm = params.areaSqm ?? "";
  return cachedFetch(`rentStats:${params.area}:${municipality}:${areaSqm}`, TTL_RENT, async () => {
    const q = new URLSearchParams({ area: params.area });
    if (params.municipality) q.set("municipality", params.municipality);
    if (params.areaSqm && params.areaSqm > 0) q.set("area_sqm", String(params.areaSqm));
    const res = await fetch(`${BASE}/rent-stats?${q}`);
    if (!res.ok) return null;
    const data = await res.json();
    if (!data || data.count === 0) return null;
    return data as RentStatsResult;
  });
}

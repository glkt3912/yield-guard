import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceStats,
  LandPriceComparison,
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
  from: string;
  to: string;
}): Promise<LandPriceStats> {
  const q = new URLSearchParams({ area: params.area, from: params.from, to: params.to });
  if (params.city) q.set("city", params.city);
  const res = await fetch(`${BASE}/land-prices?${q}`);
  return handleResponse<LandPriceStats>(res);
}

/** 検討中の土地価格と相場を比較 */
export async function compareLandPrice(params: {
  area: string;
  city?: string;
  from: string;
  to: string;
  price: number;
  areaSqm?: number;
}): Promise<LandPriceComparison> {
  const q = new URLSearchParams({
    area: params.area,
    from: params.from,
    to: params.to,
    price: String(params.price),
  });
  if (params.city) q.set("city", params.city);
  if (params.areaSqm) q.set("area_sqm", String(params.areaSqm));
  const res = await fetch(`${BASE}/land-prices/compare?${q}`);
  return handleResponse<LandPriceComparison>(res);
}

/** 投資シミュレーションを実行 */
export async function analyze(input: InvestmentInput): Promise<InvestmentResult> {
  const res = await fetch(`${BASE}/analyze`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return handleResponse<InvestmentResult>(res);
}

/** 都道府県一覧を取得 */
export async function fetchPrefectures(): Promise<{ code: string; name: string }[]> {
  const res = await fetch(`${BASE}/prefectures`);
  return handleResponse<{ code: string; name: string }[]>(res);
}

import type { Page } from "@playwright/test";

import analyzeFixture from "../fixtures/analyze-response.json";
import municipalitiesFixture from "../fixtures/municipalities-tokyo.json";
import landPriceStatsFixture from "../fixtures/land-price-stats.json";
import landPriceCompareFixture from "../fixtures/land-price-compare.json";
import landPriceEstimateFixture from "../fixtures/land-price-estimate.json";
import landAppraisalsFixture from "../fixtures/land-appraisals.json";
import urbanRisksFixture from "../fixtures/urban-risks-empty.json";
import hazardFixture from "../fixtures/hazard-empty.json";
import investmentScoreFixture from "../fixtures/investment-score.json";
import populationForecastFixture from "../fixtures/population-forecast.json";
import stationRidershipFixture from "../fixtures/station-ridership.json";
import rentDeclineHintFixture from "../fixtures/rent-decline-hint.json";
import rentStatsFixture from "../fixtures/rent-stats.json";
import areaDiscoveryFixture from "../fixtures/area-discovery.json";

type Override = {
  status: number;
  body: unknown;
  /** POST リクエストボディをキャプチャするコールバック */
  onRequest?: (body: unknown) => void;
};
export type ApiOverrides = Partial<{
  analyze: Override;
  simulate: Override;
  municipalities: Override;
  landPriceStats: Override;
  landPriceCompare: Override;
  landPriceEstimate: Override;
  landAppraisals: Override;
  urbanRisks: Override;
  hazard: Override;
  investmentScore: Override;
  populationForecast: Override;
  stationRidership: Override;
  rentDeclineHint: Override;
  rentStats: Override;
  areaDiscovery: Override;
  renovation: Override;
}>;

function fulfill(override: Override | undefined, defaultBody: unknown) {
  return {
    status: override?.status ?? 200,
    contentType: "application/json",
    body: JSON.stringify(override?.body ?? defaultBody),
  };
}

export async function setupApiMocks(page: Page, overrides: ApiOverrides = {}): Promise<void> {
  await page.route("**/api/investment/analyze", async (route) => {
    if (route.request().method() === "POST") {
      overrides.analyze?.onRequest?.(route.request().postDataJSON());
      await route.fulfill(fulfill(overrides.analyze, analyzeFixture));
    } else {
      await route.continue();
    }
  });

  await page.route("**/api/investment/simulate", async (route) => {
    await route.fulfill(fulfill(overrides.simulate, { iterations: [], percentiles: {} }));
  });

  await page.route("**/api/investment/rent-decline-hint**", async (route) => {
    await route.fulfill(fulfill(overrides.rentDeclineHint, rentDeclineHintFixture));
  });

  await page.route("**/api/renovation/analyze", async (route) => {
    await route.fulfill(fulfill(overrides.renovation, { totalCost: 0, roi: 0, paybackYears: 0 }));
  });

  await page.route("**/api/municipalities**", async (route) => {
    await route.fulfill(fulfill(overrides.municipalities, municipalitiesFixture));
  });

  await page.route("**/api/land-prices/stats**", async (route) => {
    await route.fulfill(fulfill(overrides.landPriceStats, landPriceStatsFixture));
  });

  await page.route("**/api/land-prices/compare**", async (route) => {
    await route.fulfill(fulfill(overrides.landPriceCompare, landPriceCompareFixture));
  });

  await page.route("**/api/land-prices/estimate**", async (route) => {
    await route.fulfill(fulfill(overrides.landPriceEstimate, landPriceEstimateFixture));
  });

  await page.route("**/api/land-appraisals**", async (route) => {
    await route.fulfill(fulfill(overrides.landAppraisals, landAppraisalsFixture));
  });

  await page.route("**/api/urban-risks**", async (route) => {
    await route.fulfill(fulfill(overrides.urbanRisks, urbanRisksFixture));
  });

  await page.route("**/api/hazard**", async (route) => {
    await route.fulfill(fulfill(overrides.hazard, hazardFixture));
  });

  await page.route("**/api/investment-score**", async (route) => {
    await route.fulfill(fulfill(overrides.investmentScore, investmentScoreFixture));
  });

  await page.route("**/api/investment-score-heatmap**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: "[]",
    });
  });

  await page.route("**/api/population-forecast**", async (route) => {
    await route.fulfill(fulfill(overrides.populationForecast, populationForecastFixture));
  });

  await page.route("**/api/station-ridership**", async (route) => {
    await route.fulfill(fulfill(overrides.stationRidership, stationRidershipFixture));
  });

  await page.route("**/api/rent-stats**", async (route) => {
    await route.fulfill(fulfill(overrides.rentStats, rentStatsFixture));
  });

  await page.route("**/api/area-discovery**", async (route) => {
    await route.fulfill(fulfill(overrides.areaDiscovery, areaDiscoveryFixture));
  });

  // geocode は住所入力テストが追加されるまで空レスポンスを返す
  await page.route("**/api/geocode**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ lat: 35.658, lng: 139.701, locationType: "APPROXIMATE" }),
    });
  });

  // 未登録エンドポイントの検知
  await page.route("**/api/**", async (route) => {
    console.warn(`[E2E] Unexpected API call: ${route.request().url()}`);
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        error: `Unexpected API call in test: ${route.request().url()}`,
      }),
    });
  });
}

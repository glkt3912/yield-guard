import { test, expect } from "@playwright/test";
import { analyzeFixture } from "./fixtures/analyze-response";
import { SimulationPage } from "./pages/simulation.page";

test.describe("Full mode ハッピーパス", () => {
  test("@p1 利回りとチャートが描画される", async ({ page }) => {
    const sim = new SimulationPage(page);
    await sim.setup();
    await sim.runFullSimulation({
      landPrice: "1200",
      buildingCost: "800",
      buildingAge: "20",
      monthlyRent: "15",
      loanAmount: "1600",
    });

    await expect(sim.grossYield()).toHaveText("10.58");
    await expect(sim.yieldBadge()).toBeVisible({ timeout: 5_000 });

    await expect(sim.cashflowChartHeading()).toBeVisible({ timeout: 5_000 });
    await expect(sim.deadCrossChartHeading()).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Full mode リクエストボディ検証", () => {
  test("@p1 送信時にリクエストボディが正しく構成される", async ({ page }) => {
    let capturedBody: Record<string, unknown> | null = null;

    const sim = new SimulationPage(page);
    await sim.setup({
      analyze: {
        status: 200,
        body: analyzeFixture,
        onRequest: (body) => {
          capturedBody = body as Record<string, unknown>;
        },
      },
    });

    await sim.runFullSimulation({
      landPrice: "1200",
      buildingCost: "800",
      buildingAge: "20",
      monthlyRent: "15",
    });

    await expect(sim.grossYield()).toHaveText("10.58");

    expect(capturedBody).not.toBeNull();
    expect(capturedBody!["landPrice"]).toBe(12000000);
    expect(capturedBody!["buildingCost"]).toBe(8000000);
    expect(capturedBody!["monthlyRent"]).toBe(150000);
    expect(capturedBody!["buildingAge"]).toBe(20);
  });
});

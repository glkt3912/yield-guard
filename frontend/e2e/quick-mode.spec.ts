import { test, expect } from "@playwright/test";
import { analyzeFixture } from "./fixtures/analyze-response";
import { SimulationPage } from "./pages/simulation.page";

test.describe("Quick mode ハッピーパス", () => {
  test("@p1 利回りバッジが緑でDSCRが表示される", async ({ page }) => {
    const sim = new SimulationPage(page);
    await sim.setup();
    await sim.runQuickSimulation("1700", "15");

    await expect(sim.grossYield()).toHaveText("10.58");
    await expect(sim.yieldBadge()).toBeVisible({ timeout: 5_000 });
    await expect(sim.dscrValue().first()).toBeVisible();
  });
});

test.describe("Quick mode リクエストボディ検証", () => {
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

    await sim.runQuickSimulation("1700", "150000");

    await expect(sim.grossYield()).toHaveText("10.58");

    expect(capturedBody).not.toBeNull();
    expect(capturedBody!["monthlyRent"]).toBe(150000);
    // Quick mode: landPrice = 総額 × 0.7
    expect(capturedBody!["landPrice"]).toBe(11900000);
    expect(capturedBody!["buildingCost"]).toBe(5100000);
  });
});

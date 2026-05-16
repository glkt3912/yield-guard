import { test, expect } from "@playwright/test";
import analyzeFixture from "./fixtures/analyze-response.json";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.describe("Full mode ハッピーパス", () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
    await page.goto("/");
    await page.getByRole("radio", { name: "詳細" }).click();
  });

  test("@p1 利回りとチャートが描画される", async ({ page }) => {
    await page.getByLabel("土地取得価格").fill("1200");
    await page.getByLabel("建物価格").fill("800");
    await page.getByLabel("築年数").fill("20");
    await page.getByLabel("想定月額賃料").fill("15");
    await page.getByLabel("ローン金額").fill("1600");
    await page.getByText("シミュレーション実行").click();

    await expect(page.getByTestId("gross-yield-value")).toHaveText("9.89", { timeout: 10_000 });
    await expect(page.getByTestId("yield-threshold-badge")).toBeVisible({ timeout: 5_000 });

    await expect(page.getByTestId("cashflow-chart-heading")).toBeVisible({ timeout: 5_000 });
    await expect(page.getByTestId("dead-cross-chart-heading")).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Full mode リクエストボディ検証", () => {
  test("@p1 送信時にリクエストボディが正しく構成される", async ({ page }) => {
    let capturedBody: Record<string, unknown> | null = null;

    await setupApiMocks(page, {
      analyze: {
        status: 200,
        body: analyzeFixture,
        onRequest: (body) => {
          capturedBody = body as Record<string, unknown>;
        },
      },
    });
    await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
    await page.goto("/");
    await page.getByRole("radio", { name: "詳細" }).click();

    await page.getByLabel("土地取得価格").fill("1200");
    await page.getByLabel("建物価格").fill("800");
    await page.getByLabel("築年数").fill("20");
    await page.getByLabel("想定月額賃料").fill("150000");
    await page.getByText("シミュレーション実行").click();

    await expect(page.getByTestId("gross-yield-value")).toHaveText("9.89", { timeout: 10_000 });

    expect(capturedBody).not.toBeNull();
    expect(capturedBody!["landPrice"]).toBe(12000000);
    expect(capturedBody!["buildingCost"]).toBe(8000000);
    expect(capturedBody!["monthlyRent"]).toBe(150000);
    expect(capturedBody!["buildingAge"]).toBe(20);
  });
});

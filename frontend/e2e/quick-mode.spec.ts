import { test, expect } from "@playwright/test";
import analyzeFixture from "./fixtures/analyze-response.json";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.describe("Quick mode ハッピーパス", () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
    await page.goto("/");
  });

  test("@p1 利回りバッジが緑でDSCRが表示される", async ({ page }) => {
    await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
    await page.getByLabel("想定月額賃料").fill("15");
    await page.getByText("シミュレーション実行").click();

    await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText("8.0%超え ✓")).toBeVisible();
    await expect(page.getByRole("cell", { name: "2.71" }).first()).toBeVisible();
  });
});

test.describe("Quick mode リクエストボディ検証", () => {
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

    await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
    await page.getByLabel("想定月額賃料").fill("15");
    await page.getByText("シミュレーション実行").click();

    await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });

    expect(capturedBody).not.toBeNull();
    expect(capturedBody!["monthlyRent"]).toBe(150000);
    // Quick mode: landPrice = 総額 × 0.7
    expect(capturedBody!["landPrice"]).toBe(11900000);
    expect(capturedBody!["buildingCost"]).toBe(5100000);
  });
});

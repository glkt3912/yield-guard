import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";

const ONBOARDING_KEY = "yield-guard:onboarded";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
  // 詳細モードに切り替え
  await page.getByRole("radio", { name: "詳細" }).click();
});

test("@p1 Full mode ハッピーパス：利回りとチャートが描画される", async ({ page }) => {
  await page.getByLabel("土地取得価格").fill("1200");
  await page.getByLabel("建物価格").fill("800");
  await page.getByLabel("築年数").fill("20");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByLabel("ローン金額").fill("1600");
  await page.getByText("シミュレーション実行").click();

  await expect(page.getByText("9.89%")).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText("8.0%超え ✓")).toBeVisible();

  // キャッシュフローチャートセクションが描画される
  await expect(page.getByText("キャッシュフロー推移")).toBeVisible();
  // デッドクロスチャートセクションが描画される
  await expect(page.getByText("デッドクロス")).toBeVisible();
});

test("@p1 Full mode：送信時にリクエストボディが正しく構成される", async ({ page }) => {
  let capturedBody: Record<string, unknown> | null = null;

  await page.route("**/api/investment/analyze", async (route) => {
    if (route.request().method() === "POST") {
      capturedBody = route.request().postDataJSON() as Record<string, unknown>;
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify((await import("./fixtures/analyze-response.json")).default),
    });
  });

  await page.getByLabel("土地取得価格").fill("1200");
  await page.getByLabel("建物価格").fill("800");
  await page.getByLabel("築年数").fill("20");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();

  await expect(page.getByText("9.89%")).toBeVisible({ timeout: 10_000 });

  expect(capturedBody).not.toBeNull();
  expect(capturedBody!["landPrice"]).toBe(12000000);
  expect(capturedBody!["buildingCost"]).toBe(8000000);
  expect(capturedBody!["monthlyRent"]).toBe(150000);
  expect(capturedBody!["buildingAge"]).toBe(20);
});

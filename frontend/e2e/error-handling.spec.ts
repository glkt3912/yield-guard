import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";
import { analyzeCriticalFixture } from "./fixtures/analyze-response-critical";
import { hazardAlertFixture } from "./fixtures/hazard-alert";

test.beforeEach(async ({ page }) => {
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
});

test("@p1 バックエンド503エラー：エラーバナーが表示されクラッシュしない", async ({ page }) => {
  await setupApiMocks(page, {
    analyze: { status: 503, body: { error: "Service Unavailable" } },
  });
  await page.goto("/");

  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();

  // ⚠ 付きエラーバナーが表示される
  await expect(page.locator("text=⚠").first()).toBeVisible({ timeout: 10_000 });
  // ページがクラッシュしていない（再実行ボタンが操作可能）
  await expect(page.getByText("シミュレーション実行")).toBeEnabled();
});

test("@p2 バックエンド429レートリミット：エラーバナーが表示される", async ({ page }) => {
  await setupApiMocks(page, {
    analyze: { status: 429, body: { error: "Too Many Requests" } },
  });
  await page.goto("/");

  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();

  await expect(page.locator("text=⚠").first()).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText("シミュレーション実行")).toBeEnabled();
});

test("@p3 criticalErrors バナー：⛔一発退場が表示される", async ({ page }) => {
  await setupApiMocks(page, {
    analyze: { status: 200, body: analyzeCriticalFixture },
  });
  await page.goto("/");

  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("5");
  await page.getByText("シミュレーション実行").click();

  const banner = page.getByRole("alert", { name: "重大なリスク" });
  await expect(banner).toBeVisible({ timeout: 10_000 });
  await expect(banner).toContainText("⛔");
  await expect(banner).toContainText("一発退場");
});

test("@p3 ハザードアラートバナー：洪水リスクが表示される", async ({ page }) => {
  await setupApiMocks(page, {
    hazard: { status: 200, body: hazardAlertFixture },
  });
  await page.goto("/");

  await page.getByRole("radio", { name: "詳細" }).click();
  await page.getByLabel("土地取得価格").fill("1200");
  await page.getByLabel("建物価格").fill("800");
  await page.getByLabel("築年数").fill("20");
  await page.getByLabel("想定月額賃料").fill("15");

  // 住所から座標を取得してハザード情報フェッチをトリガー
  // （hazard API は座標設定済みの「相場データを取得」経由でのみ呼ばれる）
  await page.getByLabel("物件住所").fill("東京都渋谷区");
  await page.getByRole("button", { name: "座標を取得" }).click();
  await expect(page.getByText("近似位置で取得（精度低）")).toBeVisible({ timeout: 10_000 });

  const hazardResponse = page.waitForResponse("**/api/hazard**");
  await page.getByRole("button", { name: "相場データを取得" }).click();
  await hazardResponse;

  const simulateBtn = page.getByText("シミュレーション実行");
  await expect(simulateBtn).toBeVisible({ timeout: 10_000 });
  await simulateBtn.click();

  const hazardBanner = page.getByRole("alert", { name: "ハザード警告" });
  await expect(hazardBanner).toBeVisible({ timeout: 10_000 });
  await expect(hazardBanner).toContainText("洪水");
});

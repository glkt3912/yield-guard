import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
});

test("@p3 エリア探索タブ：市区町村ランキングが表示される", async ({ page }) => {
  // タブを切り替え（最初の「エリアを探す」はタブボタン）
  await page.getByRole("button", { name: "エリアを探す" }).first().click();

  // AreaDiscovery 内の検索ボタンをクリックしてデータを取得
  await page.getByRole("button", { name: "エリアを探す" }).last().click();

  // フィクスチャの市区町村データが表示される
  await expect(page.getByText("前橋市")).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText("達成しやすい")).toBeVisible();
  await expect(page.getByText("高崎市")).toBeVisible();
  await expect(page.getByText("やや難しい")).toBeVisible();
});

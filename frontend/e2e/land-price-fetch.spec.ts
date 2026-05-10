import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";

const ONBOARDING_KEY = "yield-guard:onboarded";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
  await page.getByRole("radio", { name: "詳細" }).click();
});

test("@p2 地価データ取得：市区町村を選択して相場取得ボタンを押すと比較結果が表示される", async ({
  page,
}) => {
  // 都道府県を選択（東京都）
  await page.getByLabel("都道府県").selectOption("13");
  // 市区町村ドロップダウンが埋まるのを待つ
  await expect(page.getByText("渋谷区")).toBeVisible({ timeout: 5_000 });

  // 「相場データを取得」ボタンをクリック
  await page.getByText("相場データを取得").click();

  // land-price-compare フィクスチャの assessment が表示される
  await expect(page.getByText("割高")).toBeVisible({ timeout: 10_000 });
});

test("@p2 地価データ取得：市区町村ドロップダウンに選択肢が表示される", async ({ page }) => {
  await page.getByLabel("都道府県").selectOption("13");

  await expect(page.getByText("千代田区")).toBeVisible({ timeout: 5_000 });
  await expect(page.getByText("渋谷区")).toBeVisible();
});

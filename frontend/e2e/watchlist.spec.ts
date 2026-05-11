import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
});

async function runQuickAnalysis(page: Page) {
  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });
}

test("@p2 ウォッチリスト：追加するとトーストが表示される", async ({ page }) => {
  await runQuickAnalysis(page);

  await page.getByRole("textbox", { name: "物件名" }).fill("渋谷区テスト物件");
  await page.locator("div.flex.gap-2").filter({ has: page.locator('[aria-label="物件名"]') }).getByRole("button", { name: "追加" }).click();

  await expect(page.getByText("「渋谷区テスト物件」をウォッチリストに追加しました")).toBeVisible({
    timeout: 5_000,
  });
  await expect(page.getByText("渋谷区テスト物件")).toBeVisible();
});

test("@p2 ウォッチリスト：ページリロード後もエントリが残る", async ({ page }) => {
  await runQuickAnalysis(page);

  await page.getByRole("textbox", { name: "物件名" }).fill("永続化テスト物件");
  await page.locator("div.flex.gap-2").filter({ has: page.locator('[aria-label="物件名"]') }).getByRole("button", { name: "追加" }).click();
  await expect(page.getByText("永続化テスト物件")).toBeVisible({ timeout: 5_000 });

  // リロード後も localStorage から復元される（addInitScript は beforeEach で登録済み）
  await page.reload();
  await expect(page.getByText("永続化テスト物件")).toBeVisible({ timeout: 5_000 });
});

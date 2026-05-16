import { test, expect } from "@playwright/test";
import { SimulationPage } from "./pages/simulation.page";

test.beforeEach(async ({ page }) => {
  const sim = new SimulationPage(page);
  await sim.setup();
});

test("@p2 ウォッチリスト：追加するとトーストが表示される", async ({ page }) => {
  const sim = new SimulationPage(page);
  await sim.runQuickSimulation("1700", "15");
  await page.getByRole("tab", { name: "詳細・アクション" }).click();

  await page.getByRole("textbox", { name: "物件名" }).fill("渋谷区テスト物件");
  await page.getByTestId("watchlist-add-button").click();

  await expect(page.getByText("「渋谷区テスト物件」をウォッチリストに追加しました")).toBeVisible({
    timeout: 5_000,
  });
  await expect(page.getByTestId("watchlist-item-name").filter({ hasText: "渋谷区テスト物件" })).toBeVisible({ timeout: 5_000 });
});

test("@p2 ウォッチリスト：ページリロード後もエントリが残る", async ({ page }) => {
  const sim = new SimulationPage(page);
  await sim.runQuickSimulation("1700", "15");
  await page.getByRole("tab", { name: "詳細・アクション" }).click();

  await page.getByRole("textbox", { name: "物件名" }).fill("永続化テスト物件");
  await page.getByTestId("watchlist-add-button").click();
  await expect(page.getByTestId("watchlist-item-name").filter({ hasText: "永続化テスト物件" })).toBeVisible({ timeout: 5_000 });

  // リロード後も localStorage から復元される（addInitScript は beforeEach で登録済み）
  await page.reload();
  await expect(page.getByTestId("watchlist-item-name").filter({ hasText: "永続化テスト物件" })).toBeVisible({ timeout: 5_000 });
});

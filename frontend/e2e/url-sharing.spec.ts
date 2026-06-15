import { test, expect } from "@playwright/test";
import { SimulationPage } from "./pages/simulation.page";

test("@p2 URL共有：分析後のURLを開くとフォームが復元され結果が表示される", async ({
  page,
  context,
}) => {
  const sim = new SimulationPage(page);
  await sim.setup();
  await sim.runQuickSimulation("1700", "15");
  await expect(sim.grossYield()).toHaveText("10.58");

  // Clipboard 権限を付与してシェアボタンをクリック
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.getByRole("button", { name: "この条件をクリップボードにコピー" }).click();
  await expect(page.getByText("コピーしました")).toBeVisible({ timeout: 5_000 });

  // コピーされた URL を取得
  const sharedUrl = await page.evaluate(() => navigator.clipboard.readText());
  expect(sharedUrl).toContain("localhost:3000");

  // 新しいページで URL を開く
  const newPage = await context.newPage();
  const newSim = new SimulationPage(newPage);
  await newSim.setupMocksOnly();
  await newPage.goto(sharedUrl);

  // フォームが復元されている（quickTotalPriceMan パラメータ経由）
  await expect(newPage.getByLabel("物件価格（土地＋建物の総額）")).toHaveValue("1700", {
    timeout: 5_000,
  });
  // 再度シミュレーション実行して結果を表示
  await newPage.getByText("シミュレーション実行").click();
  await expect(newSim.grossYield()).toHaveText("10.58", { timeout: 10_000 });
});

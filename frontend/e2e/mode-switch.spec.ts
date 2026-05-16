import { test, expect } from "@playwright/test";
import { SimulationPage } from "./pages/simulation.page";

test.beforeEach(async ({ page }) => {
  const sim = new SimulationPage(page);
  await sim.setup();
});

test("@p2 モード切替：Quick分析後に詳細へ切り替えると結果がクリアされバナーが表示される", async ({
  page,
}) => {
  const sim = new SimulationPage(page);
  await sim.runQuickSimulation("1700", "15");
  await expect(sim.grossYield()).toHaveText("9.89");

  // 詳細モードに切り替え
  await page.getByRole("radio", { name: "詳細" }).click();

  await expect(
    page.getByText(
      "モードを切り替えたため、結果をクリアしました。再度シミュレーションを実行してください。"
    )
  ).toBeVisible({ timeout: 3_000 });
  // 結果値が消えている
  await expect(sim.grossYield()).toBeHidden();
});

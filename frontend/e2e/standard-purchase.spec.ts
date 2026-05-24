import { test, expect } from "@playwright/test";
import { QUICK_PRICE_MAN, QUICK_RENT_YEN } from "./helpers/constants";
import { SimulationPage } from "./pages/simulation.page";

test.describe("標準購入シナリオ ハッピーパス", () => {
  test("@p1 KpiStrip・StatusSummary・10年後手残りが表示される", async ({
    page,
  }) => {
    const sim = new SimulationPage(page);
    await sim.setup();
    await sim.runQuickSimulation(QUICK_PRICE_MAN, QUICK_RENT_YEN);

    await expect(sim.kpiStrip().getByText("月々の返済額")).toBeVisible();
    await expect(sim.statusSummaryBadge()).toHaveText(/投資適格|要交渉|見送り推奨/);
    await expect(sim.tenYearEquityCard()).toBeVisible();
  });
});

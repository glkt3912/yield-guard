import { test, expect } from "@playwright/test";
import { SimulationPage } from "./pages/simulation.page";

test.describe("標準購入シナリオ ハッピーパス", () => {
  test("@p1 KpiStrip・StatusSummary・10年後手残りが表示される", async ({
    page,
  }) => {
    const sim = new SimulationPage(page);
    await sim.setup();
    await sim.runQuickSimulation("1700", "150000");

    await expect(sim.kpiStrip().getByText("月々の返済額")).toBeVisible();
    await expect(sim.statusSummaryBadge()).toBeVisible();
    await expect(sim.tenYearEquityCard()).toBeVisible();
  });
});

import type { Page } from "@playwright/test";
import { expect } from "@playwright/test";
import { setupApiMocks, type ApiOverrides } from "../helpers/routes";
import { ONBOARDING_KEY } from "../helpers/constants";

export class SimulationPage {
  constructor(private page: Page) {}

  async setup(overrides: ApiOverrides = {}) {
    await setupApiMocks(this.page, overrides);
    await this.page.addInitScript(
      (key) => localStorage.setItem(key, "1"),
      ONBOARDING_KEY
    );
    await this.page.goto("/");
  }

  /** API モックと onboarding フラグだけ設定し、navigate しない（別の URL に goto する直前に使う） */
  async setupMocksOnly(overrides: ApiOverrides = {}) {
    await setupApiMocks(this.page, overrides);
    await this.page.addInitScript(
      (key) => localStorage.setItem(key, "1"),
      ONBOARDING_KEY
    );
  }

  async runQuickSimulation(price: string, rent: string) {
    await this.page.getByLabel("物件価格（土地＋建物の総額）").fill(price);
    await this.page.getByLabel("想定月額賃料").fill(rent);
    await this.page.getByText("シミュレーション実行").click();
    await expect(this.page.getByTestId("gross-yield-value")).toHaveText(/.+/, {
      timeout: 10_000,
    });
  }

  async runFullSimulation(params: {
    landPrice: string;
    buildingCost: string;
    buildingAge: string;
    monthlyRent: string;
    loanAmount?: string;
  }) {
    await this.page.getByRole("radio", { name: "くわしく分析" }).click();
    await this.page.getByLabel("土地取得価格").fill(params.landPrice);
    await this.page.getByLabel("建物価格").fill(params.buildingCost);
    await this.page.getByLabel("築年数").fill(params.buildingAge);
    await this.page.getByLabel("想定月額賃料").fill(params.monthlyRent);
    if (params.loanAmount) {
      await this.page.getByLabel("ローン金額").fill(params.loanAmount);
    }
    await this.page.getByText("シミュレーション実行").click();
    await expect(this.page.getByTestId("gross-yield-value")).toHaveText(/.+/, {
      timeout: 10_000,
    });
  }

  grossYield() {
    return this.page.getByTestId("gross-yield-value");
  }

  yieldBadge() {
    return this.page.getByTestId("yield-threshold-badge");
  }

  dscrValue() {
    return this.page.getByTestId("dscr-value");
  }

  cashflowChartHeading() {
    return this.page.getByTestId("cashflow-chart-heading");
  }

  deadCrossChartHeading() {
    return this.page.getByTestId("dead-cross-chart-heading");
  }

  kpiStrip() {
    return this.page.getByLabel("KPIサマリ");
  }

  statusSummaryBadge() {
    return this.page.getByTestId("status-summary-badge");
  }

  tenYearEquityCard() {
    return this.page.getByTestId("ten-year-equity-card");
  }
}

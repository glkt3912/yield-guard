import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import CostBreakdown from "@/components/CostBreakdown";
import { makeInput, makeResult, makeYearlyResult } from "./helpers";
import type { AcquisitionCostBreakdown } from "@/types/investment";

function makeAcquisitionCosts(overrides: Partial<AcquisitionCostBreakdown> = {}): AcquisitionCostBreakdown {
  return {
    brokerageFee: 561_000,
    stampDuty: 20_000,
    registrationTax: 420_000,
    realEstateAcquisitionTax: 315_000,
    propertyTaxProration: 0,
    total: 1_316_000,
    ...overrides,
  };
}

describe("CostBreakdown", () => {
  it("取得時諸経費の各項目ラベルが表示される", () => {
    const input = makeInput();
    const costs = makeAcquisitionCosts();
    const yearlyResults = [makeYearlyResult(1)];

    const { container } = render(<CostBreakdown input={input} acquisitionCosts={costs} yearlyResults={yearlyResults} />);

    // Labels appear in both the pie chart legend and the detail table;
    // scope assertions to the table to avoid "multiple elements found" errors.
    const table = container.querySelector("table")!;
    expect(within(table).getByText("仲介手数料（税込）")).toBeInTheDocument();
    expect(within(table).getByText("印紙税")).toBeInTheDocument();
    expect(within(table).getByText("登録免許税")).toBeInTheDocument();
    expect(within(table).getByText("不動産取得税（概算）")).toBeInTheDocument();
    expect(within(table).getByText("合計")).toBeInTheDocument();
  });

  it("コスト内訳セクションタイトルが表示される", () => {
    render(
      <CostBreakdown
        input={makeInput()}
        acquisitionCosts={makeAcquisitionCosts()}
        yearlyResults={[makeYearlyResult(1)]}
      />
    );
    expect(screen.getByText("コスト内訳")).toBeInTheDocument();
    expect(screen.getByText("取得時諸経費の明細")).toBeInTheDocument();
  });

  it("固定資産税日割りがゼロのとき行が表示されない", () => {
    render(
      <CostBreakdown
        input={makeInput()}
        acquisitionCosts={makeAcquisitionCosts({ propertyTaxProration: 0 })}
        yearlyResults={[makeYearlyResult(1)]}
      />
    );
    expect(screen.queryByText("固定資産税日割り精算")).not.toBeInTheDocument();
  });

  it("固定資産税日割りが正値のとき行が表示される", () => {
    render(
      <CostBreakdown
        input={makeInput()}
        acquisitionCosts={makeAcquisitionCosts({ propertyTaxProration: 50_000 })}
        yearlyResults={[makeYearlyResult(1)]}
      />
    );
    expect(screen.getByText("固定資産税日割り精算")).toBeInTheDocument();
  });

  it("1年目の年間費用内訳（ローン返済・運営経費・所得税）が表示される", () => {
    const year1 = makeYearlyResult(1, {
      annualLoanPayment: 600_000,
      annualExpenses: 240_000,
      incomeTax: 85_800,
    });

    render(
      <CostBreakdown
        input={makeInput()}
        acquisitionCosts={makeAcquisitionCosts()}
        yearlyResults={[year1]}
      />
    );

    expect(screen.getByText("年間費用の内訳（1年目）")).toBeInTheDocument();
    expect(screen.getByText("ローン返済")).toBeInTheDocument();
    expect(screen.getByText("運営経費")).toBeInTheDocument();
    expect(screen.getByText("所得税")).toBeInTheDocument();
  });

  it("ローンゼロのとき1年目年間費用にローン返済が表示されない", () => {
    const year1 = makeYearlyResult(1, {
      annualLoanPayment: 0,
      annualExpenses: 240_000,
      incomeTax: 0,
    });

    render(
      <CostBreakdown
        input={makeInput()}
        acquisitionCosts={makeAcquisitionCosts()}
        yearlyResults={[year1]}
      />
    );

    // ローン返済が0なので表示されない（filter(item => item.value > 0)）
    expect(screen.queryByText("ローン返済")).not.toBeInTheDocument();
  });

  it("yearlyResults が空のとき年間費用セクションが表示されない", () => {
    render(
      <CostBreakdown
        input={makeInput()}
        acquisitionCosts={makeAcquisitionCosts()}
        yearlyResults={[]}
      />
    );
    expect(screen.queryByText("年間費用の内訳（1年目）")).not.toBeInTheDocument();
  });

  it("土地・建物ラベルが初期投資内訳に表示される", () => {
    render(
      <CostBreakdown
        input={makeInput({ landPrice: 10_000_000, buildingCost: 5_000_000 })}
        acquisitionCosts={makeAcquisitionCosts()}
        yearlyResults={[makeYearlyResult(1)]}
      />
    );
    expect(screen.getByText("土地")).toBeInTheDocument();
    expect(screen.getByText("建物")).toBeInTheDocument();
  });
});

import { render, screen } from "@testing-library/react";
import { CashFlowChart } from "@/components/CashFlowChart";
import { makeResult } from "./helpers";
import type { YearlyResult } from "@/types/investment";

describe("CashFlowChart", () => {
  it("自己資金回収年を表示する", () => {
    // 年間税引後CF 274,200円。equityInvested=1,000,000 → 4年目で累積CF=1,096,800 > 1,000,000
    const yearlyResults: YearlyResult[] = Array.from({ length: 35 }, (_, i) => ({
      year: i + 1,
      annualRent: 1_200_000,
      annualLoanPayment: 600_000,
      annualInterest: 100_000,
      annualPrincipal: 500_000,
      annualDepreciation: 600_000,
      annualExpenses: 240_000,
      taxableIncome: 260_000,
      incomeTax: 85_800,
      cashFlow: 360_000,
      afterTaxCashFlow: 274_200,
      remainingLoanBalance: 12_000_000,
      cumulativeCashFlow: 274_200 * (i + 1),
      isDeadCrossYear: false,
      isInDeadCrossZone: false,
    }));
    const result = makeResult({ yearlyResults });
    render(<CashFlowChart result={result} equityInvested={1_000_000} />);
    expect(screen.getByText(/4年目/)).toBeInTheDocument();
  });

  it("35年以内に回収できない場合は該当メッセージを表示する", () => {
    // afterTaxCashFlow が負 → 永遠に回収不可
    const yearlyResults: YearlyResult[] = Array.from({ length: 35 }, (_, i) => ({
      year: i + 1,
      annualRent: 500_000,
      annualLoanPayment: 800_000,
      annualInterest: 200_000,
      annualPrincipal: 600_000,
      annualDepreciation: 300_000,
      annualExpenses: 100_000,
      taxableIncome: -100_000,
      incomeTax: 0,
      cashFlow: -300_000,
      afterTaxCashFlow: -300_000,
      remainingLoanBalance: 10_000_000,
      cumulativeCashFlow: -300_000 * (i + 1),
      isDeadCrossYear: false,
      isInDeadCrossZone: false,
    }));
    const result = makeResult({ yearlyResults });
    render(<CashFlowChart result={result} equityInvested={5_000_000} />);
    expect(screen.getByText(/35年以内に自己資金を回収できない/)).toBeInTheDocument();
  });

  it("exitTotalEquity が正のとき緑色で表示する", () => {
    const result = makeResult({ exitTotalEquity: 3_000_000 });
    render(<CashFlowChart result={result} equityInvested={2_000_000} />);
    const el = screen.getByText(/300\.0万円/);
    expect(el).toHaveClass("text-green-600");
  });

  it("exitTotalEquity が負のとき赤色で表示する", () => {
    const result = makeResult({ exitTotalEquity: -1_000_000 });
    render(<CashFlowChart result={result} equityInvested={2_000_000} />);
    const el = screen.getByText(/-100\.0万円/);
    expect(el).toHaveClass("text-red-600");
  });
});

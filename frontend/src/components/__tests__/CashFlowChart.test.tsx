import { render, screen } from "@testing-library/react";
import { CashFlowChart } from "@/components/CashFlowChart";
import { makeResult, makeYearlyResult } from "./helpers";

describe("CashFlowChart", () => {
  it("自己資金回収年を表示する", () => {
    // 年間税引後CF 274,200円。equityInvested=1,000,000 → 4年目で累積CF=1,096,800 > 1,000,000
    const yearlyResults = Array.from({ length: 35 }, (_, i) =>
      makeYearlyResult(i + 1, { cumulativeCashFlow: 274_200 * (i + 1) })
    );
    const result = makeResult({ yearlyResults });
    render(<CashFlowChart result={result} equityInvested={1_000_000} />);
    expect(screen.getByText(/4年目/)).toBeInTheDocument();
  });

  it("35年以内に回収できない場合は該当メッセージを表示する", () => {
    // cumulativeCashFlow が常に負 → 永遠に回収不可
    const yearlyResults = Array.from({ length: 35 }, (_, i) =>
      makeYearlyResult(i + 1, { cumulativeCashFlow: -300_000 * (i + 1) })
    );
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

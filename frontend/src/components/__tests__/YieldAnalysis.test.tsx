import { render, screen } from "@testing-library/react";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import { makeResult } from "./helpers";

describe("YieldAnalysis", () => {
  it("表面利回りの数値を表示する", () => {
    const result = makeResult({ grossYield: 0.09 });
    render(<YieldAnalysis result={result} />);
    expect(screen.getByText("9.00")).toBeInTheDocument();
  });

  it("8%超えのとき緑のバッジと成功アイコンを表示する", () => {
    const result = makeResult({ isAbove8Percent: true, grossYield: 0.09 });
    render(<YieldAnalysis result={result} />);
    expect(screen.getByText("8%超え ✓")).toBeInTheDocument();
  });

  it("8%未満のとき赤のバッジを表示する", () => {
    const result = makeResult({ isAbove8Percent: false, grossYield: 0.07 });
    render(<YieldAnalysis result={result} />);
    expect(screen.getByText("8%未満 ✗")).toBeInTheDocument();
  });

  it("8%未満のとき改善カードを表示し、余裕度カードを非表示にする", () => {
    const result = makeResult({
      isAbove8Percent: false,
      grossYield: 0.07,
      requiredCostReduction: 500_000,
      requiredMonthlyRent: 140_000,
    });
    render(<YieldAnalysis result={result} />);
    expect(screen.getByText("8%達成のために必要な改善（いずれか一方）")).toBeInTheDocument();
    expect(screen.queryByText("8%超え達成！余裕度")).not.toBeInTheDocument();
  });

  it("8%以上のとき余裕度カードを表示し、改善カードを非表示にする", () => {
    const result = makeResult({ isAbove8Percent: true, grossYield: 0.09 });
    render(<YieldAnalysis result={result} />);
    expect(screen.getByText("8%超え達成！余裕度")).toBeInTheDocument();
    expect(screen.queryByText("8%達成のために必要な改善（いずれか一方）")).not.toBeInTheDocument();
  });
});

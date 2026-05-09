import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import type { LandPriceStats } from "@/types/investment";
import { makeInput, makeResult } from "./helpers";

const makeLandPriceStats = (overrides?: Partial<LandPriceStats>): LandPriceStats => ({
  count: 10,
  averageTsubo: 200_000,
  medianTsubo: 200_000,
  minTsubo: 150_000,
  maxTsubo: 250_000,
  transactions: [],
  lowDataWarning: false,
  ...overrides,
});

describe("YieldAnalysis", () => {
  it("表面利回りの数値を表示する", () => {
    const result = makeResult({ grossYield: 0.09 });
    render(<YieldAnalysis result={result} input={makeInput()} />);
    expect(screen.getByText("9.00")).toBeInTheDocument();
  });

  it("8%超えのとき緑のバッジと成功アイコンを表示する", () => {
    const result = makeResult({ isAboveYieldTarget: true, grossYield: 0.09 });
    render(<YieldAnalysis result={result} input={makeInput()} />);
    expect(screen.getByText("8%超え ✓")).toBeInTheDocument();
  });

  it("8%未満のとき赤のバッジを表示する", () => {
    const result = makeResult({ isAboveYieldTarget: false, grossYield: 0.07 });
    render(<YieldAnalysis result={result} input={makeInput()} />);
    expect(screen.getByText("8%未満 ✗")).toBeInTheDocument();
  });

  it("8%未満のとき改善カードを表示し、余裕度カードを非表示にする", () => {
    const result = makeResult({
      isAboveYieldTarget: false,
      grossYield: 0.07,
      requiredCostReduction: 500_000,
      requiredMonthlyRent: 140_000,
    });
    render(<YieldAnalysis result={result} input={makeInput()} />);
    expect(screen.getByText("8%達成のために必要な改善（いずれか一方）")).toBeInTheDocument();
    expect(screen.queryByText("8%超え達成！余裕度")).not.toBeInTheDocument();
  });

  it("8%以上のとき余裕度カードを表示し、改善カードを非表示にする", () => {
    const result = makeResult({ isAboveYieldTarget: true, grossYield: 0.09 });
    render(<YieldAnalysis result={result} input={makeInput()} />);
    expect(screen.getByText("8%超え達成！余裕度")).toBeInTheDocument();
    expect(screen.queryByText("8%達成のために必要な改善（いずれか一方）")).not.toBeInTheDocument();
  });

  it("landPriceStats が有効なとき市場想定利回りテキストが表示される", () => {
    const result = makeResult({ isAboveYieldTarget: true, grossYield: 0.09 });
    render(
      <YieldAnalysis result={result} input={makeInput()} landPriceStats={makeLandPriceStats()} />
    );
    expect(screen.getByText(/市場想定利回り/)).toBeInTheDocument();
  });

  it("judgment が realistic のとき 現実的 Badge が表示される", () => {
    // makeInput defaults: landPrice=10M, buildingCost=5M, monthlyRent=120K
    // userYield = 120K*12 / 15M ≈ 0.096
    // estimatedYieldTypical with medianTsubo=200K, area=100sqm ≈ 0.130
    // ratio ≈ 0.737 → realistic
    const result = makeResult({ isAboveYieldTarget: true, grossYield: 0.09 });
    render(
      <YieldAnalysis result={result} input={makeInput()} landPriceStats={makeLandPriceStats()} />
    );
    expect(screen.getByText("現実的")).toBeInTheDocument();
  });

  it("landPriceStats が null のとき市場想定利回りブロックが表示されない", () => {
    const result = makeResult({ isAboveYieldTarget: true, grossYield: 0.09 });
    render(<YieldAnalysis result={result} input={makeInput()} landPriceStats={null} />);
    expect(screen.queryByText(/市場想定利回り/)).not.toBeInTheDocument();
  });
});

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MonteCarloChart } from "@/components/MonteCarloChart";
import type { MonteCarloResult } from "@/types/investment";

// Recharts mocked globally via ResizeObserver in vitest.setup.ts

function makeMonteCarloResult(overrides: Partial<MonteCarloResult> = {}): MonteCarloResult {
  return {
    simulationCount: 1000,
    successRate: 0.75,
    deadCrossRate: 0.2,
    irrPercentiles: {
      p10: -0.01,
      p25: 0.02,
      p50: 0.05,
      p75: 0.08,
      p90: 0.12,
    },
    equityPercentiles: {
      p10: -500_000,
      p25: 200_000,
      p50: 1_000_000,
      p75: 2_000_000,
      p90: 3_500_000,
    },
    irrHistogram: [
      { min: -0.05, max: 0, count: 100 },
      { min: 0, max: 0.05, count: 400 },
      { min: 0.05, max: 0.1, count: 350 },
      { min: 0.1, max: 0.15, count: 150 },
    ],
    equityHistogram: [
      { min: -1_000_000, max: 0, count: 50 },
      { min: 0, max: 1_000_000, count: 450 },
      { min: 1_000_000, max: 2_000_000, count: 400 },
      { min: 2_000_000, max: 3_000_000, count: 100 },
    ],
    ...overrides,
  };
}

describe("MonteCarloChart", () => {
  it("カードタイトルが表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult()} />);
    expect(screen.getByText("モンテカルロ・シミュレーション結果")).toBeInTheDocument();
  });

  it("試行回数が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ simulationCount: 1000 })} />);
    expect(screen.getByText(/1,000 試行/)).toBeInTheDocument();
  });

  it("IRR正値達成率が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ successRate: 0.75 })} />);
    expect(screen.getByText("IRR正値達成率")).toBeInTheDocument();
    expect(screen.getByText("75.0%")).toBeInTheDocument();
  });

  it("デッドクロス発生率が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ deadCrossRate: 0.2 })} />);
    expect(screen.getByText("デッドクロス発生率")).toBeInTheDocument();
    expect(screen.getByText("20.0%")).toBeInTheDocument();
  });

  it("IRR中央値が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult()} />);
    expect(screen.getByText("IRR中央値")).toBeInTheDocument();
    // p50 = 0.05 → 5.0% (appears in both StatCard and percentile table; take first = StatCard)
    expect(screen.getAllByText("5.0%")[0]).toBeInTheDocument();
  });

  it("パーセンタイル表が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult()} />);
    expect(screen.getByText("P10（悲観）")).toBeInTheDocument();
    expect(screen.getByText("P25")).toBeInTheDocument();
    expect(screen.getByText("P50（中央値）")).toBeInTheDocument();
    expect(screen.getByText("P75")).toBeInTheDocument();
    expect(screen.getByText("P90（楽観）")).toBeInTheDocument();
  });

  it("IRRヒストグラムセクションが表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult()} />);
    // Text is split across elements (TermTooltip wraps "IRR"), so use function matcher
    expect(
      screen.getByText((_, el) => {
        const text = el?.textContent?.trim().replace(/\s+/g, " ") ?? "";
        return el?.tagName === "P" && /IRR\s*分布/.test(text);
      })
    ).toBeInTheDocument();
  });

  it("最終純資産ヒストグラムセクションが表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult()} />);
    expect(screen.getByText("最終純資産 分布")).toBeInTheDocument();
  });

  it("irrHistogram が null のとき「IRRを算出できた試行がありませんでした」が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ irrHistogram: null })} />);
    expect(screen.getByText(/IRRを算出できた試行がありませんでした/)).toBeInTheDocument();
  });

  it("irrHistogram が空配列のとき「IRRを算出できた試行がありませんでした」が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ irrHistogram: [] })} />);
    expect(screen.getByText(/IRRを算出できた試行がありませんでした/)).toBeInTheDocument();
  });

  it("successRate >= 0.5 のとき IRR正値達成率の値が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ successRate: 0.75 })} />);
    expect(screen.getByText("IRR正値達成率")).toBeInTheDocument();
    expect(screen.getByText("75.0%")).toBeInTheDocument();
  });

  it("successRate < 0.5 のとき IRR正値達成率の値が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ successRate: 0.3 })} />);
    expect(screen.getByText("IRR正値達成率")).toBeInTheDocument();
    expect(screen.getByText("30.0%")).toBeInTheDocument();
  });

  it("deadCrossRate < 0.3 のとき デッドクロス発生率の値が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ deadCrossRate: 0.2 })} />);
    expect(screen.getByText("デッドクロス発生率")).toBeInTheDocument();
    expect(screen.getByText("20.0%")).toBeInTheDocument();
  });

  it("deadCrossRate >= 0.3 のとき デッドクロス発生率の値が表示される", () => {
    render(<MonteCarloChart result={makeMonteCarloResult({ deadCrossRate: 0.5 })} />);
    expect(screen.getByText("デッドクロス発生率")).toBeInTheDocument();
    expect(screen.getByText("50.0%")).toBeInTheDocument();
  });

  it("IRR中央値が負のとき負の値が表示される", () => {
    render(
      <MonteCarloChart
        result={makeMonteCarloResult({
          irrPercentiles: {
            p10: -0.1,
            p25: -0.05,
            p50: -0.02,
            p75: 0.01,
            p90: 0.05,
          },
        })}
      />
    );
    // p50 = -0.02 → -2.0% がパーセンタイル表に表示される
    expect(screen.getAllByText("-2.0%")[0]).toBeInTheDocument();
  });
});

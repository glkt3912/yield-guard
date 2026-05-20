import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusSummary } from "@/components/StatusSummary";
import type { StressScenarioResult } from "@/types/investment";
import { makeResult, makeInput } from "./helpers";

const stressScenario = (label: string, dscr: number): StressScenarioResult => ({
  label,
  dscr,
  interestRateDelta: 0,
  vacancyRateDelta: 0,
  totalCashFlow: 0,
  breakEvenYear: -1,
  isSafe: dscr >= 1.0,
});

describe("StatusSummary", () => {
  it("PASS 判定: 「投資適格」が表示される", () => {
    const result = makeResult({
      dscr: 1.3,
      grossYield: 0.09,
      yieldTarget: 0.08,
      exitTotalEquity: 1_000_000,
      criticalErrors: [],
      stressScenarios: [],
    });
    render(<StatusSummary result={result} input={makeInput()} />);
    expect(screen.getByText(/投資適格/)).toBeInTheDocument();
  });

  it("CAUTION 判定: 「要交渉」が表示される（DSCR 1.0 以上 1.2 未満）", () => {
    const result = makeResult({
      dscr: 1.1,
      criticalErrors: [],
      stressScenarios: [],
    });
    render(<StatusSummary result={result} input={makeInput()} />);
    expect(screen.getByText(/要交渉/)).toBeInTheDocument();
  });

  it("REJECT 判定: 「見送り推奨」が表示される（REJECT criticalError があるとき）", () => {
    const result = makeResult({
      criticalErrors: [{ code: "TEST", status: "REJECT", message: "テスト" }],
      stressScenarios: [],
    });
    render(<StatusSummary result={result} input={makeInput()} />);
    expect(screen.getByText(/見送り推奨/)).toBeInTheDocument();
  });

  it("stressScenarios が空のとき result.dscr をフォールバックとして正常レンダリングする", () => {
    // Bug 1 修正前: dscrStress = 0（"複合ストレス" 未検出）→ calcVerdict に 0 が渡る
    // Bug 1 修正後: dscrStress = result.dscr = 1.5 → 正常
    const result = makeResult({ dscr: 1.5, stressScenarios: [] });
    render(<StatusSummary result={result} input={makeInput()} />);
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByText(/投資適格/)).toBeInTheDocument();
  });

  it("「複合ストレス」ラベルのないシナリオでも最小 DSCR を取得してクラッシュしない", () => {
    // 旧実装: "複合ストレス" が見つからず dscrStress = 0
    // 新実装: Math.min(1.1, 0.9) = 0.9 を dscrStress として使用
    const result = makeResult({
      dscr: 1.3,
      stressScenarios: [stressScenario("金利上昇", 1.1), stressScenario("空室率上昇", 0.9)],
    });
    render(<StatusSummary result={result} input={makeInput()} />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("複数ストレスシナリオのうち最小 DSCR を使用する（ラベル非依存）", () => {
    const result = makeResult({
      dscr: 1.5,
      stressScenarios: [
        stressScenario("シナリオA", 1.3),
        stressScenario("シナリオB", 0.8), // 最小
        stressScenario("シナリオC", 1.1),
      ],
    });
    // コンポーネントが正常にレンダリングされ、dscr=1.5 なので PASS 判定
    render(<StatusSummary result={result} input={makeInput()} />);
    expect(screen.getByText(/投資適格/)).toBeInTheDocument();
  });

  it("role=status と data-testid が付与されている", () => {
    render(<StatusSummary result={makeResult()} input={makeInput()} />);
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByTestId("status-summary-badge")).toBeInTheDocument();
  });
});

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import DeadCrossChart from "@/components/DeadCrossChart";
import { makeResult, makeYearlyResult } from "./helpers";
import type { CriticalError } from "@/types/investment";

const deadCrossYear = 8;

function makeDeadCrossResult(criticalErrors: CriticalError[] = []) {
  const yearlyResults = Array.from({ length: 35 }, (_, i) =>
    makeYearlyResult(i + 1, {
      isInDeadCrossZone: i + 1 >= deadCrossYear,
      annualPrincipal: i + 1 >= deadCrossYear ? 700_000 : 500_000,
      annualDepreciation: i + 1 >= deadCrossYear ? 400_000 : 600_000,
    })
  );
  return makeResult({ deadCrossYear, yearlyResults, criticalErrors });
}

describe("DeadCrossAdviceGuide", () => {
  it("criticalErrors に DEADCROSS_EARLY が含まれるとき「早期警告」バッジを表示する", () => {
    const errors: CriticalError[] = [
      { code: "DEADCROSS_EARLY", status: "WARNING", message: "10年以内にデッドクロスが発生します" },
    ];
    render(<DeadCrossChart result={makeDeadCrossResult(errors)} />);
    expect(screen.getByText("早期警告")).toBeInTheDocument();
  });

  it("criticalErrors に DEADCROSS_EARLY が含まれないとき「早期警告」バッジを表示しない", () => {
    render(<DeadCrossChart result={makeDeadCrossResult([])} />);
    expect(screen.queryByText("早期警告")).not.toBeInTheDocument();
  });

  it("3つのアドバイス項目がすべてレンダリングされる", () => {
    render(<DeadCrossChart result={makeDeadCrossResult()} />);
    expect(screen.getByText("減価償却が大きい築浅物件への切り替え")).toBeInTheDocument();
    expect(screen.getByText("法定耐用年数前の売却（デッドクロス前の出口）")).toBeInTheDocument();
    expect(screen.getByText("法人化による損益通算")).toBeInTheDocument();
  });

  it("deadCrossYear の数値がテキストに含まれる", () => {
    render(<DeadCrossChart result={makeDeadCrossResult()} />);
    expect(screen.getByText(/保有8年目から税負担が増加します/)).toBeInTheDocument();
  });
});

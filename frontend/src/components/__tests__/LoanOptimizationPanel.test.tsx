import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LoanOptimizationPanel } from "@/components/LoanOptimizationPanel";
import { makeResult } from "./helpers";
import type { LTVSensitivityRow } from "@/types/investment";

const makeLTVRows = (): LTVSensitivityRow[] => [
  { ltv: 0.5, equity: 8_025_000, loanAmount: 8_025_000, dscr: 2.1, annualCF: 500_000, cfYield: 0.031 },
  { ltv: 0.6, equity: 6_420_000, loanAmount: 9_630_000, dscr: 1.6, annualCF: 300_000, cfYield: 0.019 },
  { ltv: 0.7, equity: 4_815_000, loanAmount: 11_235_000, dscr: 1.2, annualCF: 100_000, cfYield: 0.006 },
  { ltv: 0.8, equity: 3_210_000, loanAmount: 12_840_000, dscr: 0.9, annualCF: -100_000, cfYield: -0.006 },
  { ltv: 0.9, equity: 1_605_000, loanAmount: 14_445_000, dscr: 0.6, annualCF: -300_000, cfYield: -0.019 },
];

const WITH_LOAN = 13_000_000;

describe("LoanOptimizationPanel", () => {
  it("DSCR >= 1.0 のとき緑バッジ（安全）を表示する", () => {
    const result = makeResult({ dscr: 1.2, ltvSensitivity: [] });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={WITH_LOAN} />
    );
    expect(screen.getByText("安全（≥ 1.0）")).toBeInTheDocument();
  });

  it("DSCR < 1.0 のとき赤バッジ（危険）を表示する", () => {
    const result = makeResult({ dscr: 0.8, ltvSensitivity: [] });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={WITH_LOAN} />
    );
    expect(screen.getByText("危険（< 1.0）")).toBeInTheDocument();
  });

  it("DSCR = 1.0 ちょうどのとき安全バッジを表示する", () => {
    const result = makeResult({ dscr: 1.0, ltvSensitivity: [] });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={WITH_LOAN} />
    );
    expect(screen.getByText("安全（≥ 1.0）")).toBeInTheDocument();
  });

  it("LTV 感度テーブルの5行が描画される", () => {
    const result = makeResult({ dscr: 1.2, ltvSensitivity: makeLTVRows() });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={WITH_LOAN} />
    );
    // LTV列の値が表示されていること
    expect(screen.getByText("50.00%")).toBeInTheDocument();
    expect(screen.getByText("90.00%")).toBeInTheDocument();
  });

  it("LTV テーブルで DSCR < 1.0 の行は赤文字、>= 1.0 は緑文字になる", () => {
    const result = makeResult({ dscr: 1.2, ltvSensitivity: makeLTVRows() });
    const { container } = render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={WITH_LOAN} />
    );
    const greenDscr = container.querySelectorAll(".text-green-600");
    const redDscr = container.querySelectorAll(".text-red-600");
    // LTV 50/60/70% → DSCR >= 1.0 → 緑3件、LTV 80/90% → DSCR < 1.0 → 赤
    expect(greenDscr.length).toBeGreaterThanOrEqual(3);
    expect(redDscr.length).toBeGreaterThanOrEqual(1);
  });

  it("返済方式セレクタが初期値を表示し、変更時に onLoanMethodChange を呼ぶ", () => {
    const onChange = vi.fn();
    const result = makeResult({ dscr: 1.2, ltvSensitivity: [] });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={onChange} loanAmount={WITH_LOAN} />
    );
    const select = screen.getByRole("combobox") as HTMLSelectElement;
    expect(select.value).toBe("equal-payment");

    fireEvent.change(select, { target: { value: "equal-principal" } });
    expect(onChange).toHaveBeenCalledWith("equal-principal");
  });

  it("ltvSensitivity が空のとき LTV テーブルを表示しない", () => {
    const result = makeResult({ dscr: 1.2, ltvSensitivity: [] });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={WITH_LOAN} />
    );
    expect(screen.queryByText("LTV 感度分析")).not.toBeInTheDocument();
  });

  it("loanAmount=0（現金購入）のとき DSCR=0 でも赤バッジではなく非適用バッジを表示する", () => {
    const result = makeResult({ dscr: 0, ltvSensitivity: [] });
    render(
      <LoanOptimizationPanel result={result} loanMethod="equal-payment" onLoanMethodChange={vi.fn()} loanAmount={0} />
    );
    expect(screen.getByText("非適用（ローンなし）")).toBeInTheDocument();
    expect(screen.queryByText("危険（< 1.0）")).not.toBeInTheDocument();
  });
});

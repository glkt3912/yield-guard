import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MobileSummaryCard } from "@/components/MobileSummaryCard";
import { makeInput, makeResult } from "./helpers";

vi.mock("@/lib/generatePdf", () => ({
  downloadReportPDF: vi.fn(),
}));

import * as pdfLib from "@/lib/generatePdf";

describe("MobileSummaryCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("表面利回りと実質利回りが表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ grossYield: 0.09, netYield: 0.065 })}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByText("9.00%")).toBeInTheDocument();
    expect(screen.getByText("6.50%")).toBeInTheDocument();
  });

  it("目標利回り達成のとき成功バッジが表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ isAboveYieldTarget: true })}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByText("目標利回り達成")).toBeInTheDocument();
  });

  it("目標利回り未達のとき警告バッジが表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ isAboveYieldTarget: false })}
        input={makeInput()}
        yieldPct={5.0}
        netYieldPct={3.0}
      />
    );
    expect(screen.getByText("目標利回り未達")).toBeInTheDocument();
  });

  it("DSCR が表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ dscr: 1.35 })}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByText("1.35")).toBeInTheDocument();
  });

  it("デッドクロスなしのとき「なし」が表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ deadCrossYear: 0 })}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByText("なし")).toBeInTheDocument();
  });

  it("デッドクロスありのとき「N年目〜」が表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ deadCrossYear: 15 })}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByText("15年目〜")).toBeInTheDocument();
  });

  it("IRRが表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult({ irr: 0.07 })}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByText("7.00%")).toBeInTheDocument();
  });

  it("IRRがnullのとき「―」が表示される", () => {
    const result = makeResult();
    (result as { irr: null }).irr = null;
    render(
      <MobileSummaryCard result={result} input={makeInput()} yieldPct={9.0} netYieldPct={6.5} />
    );
    expect(screen.getByText("―")).toBeInTheDocument();
  });

  it("「詳細PDFを出力」ボタンが表示される", () => {
    render(
      <MobileSummaryCard
        result={makeResult()}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    expect(screen.getByRole("button", { name: /詳細PDFを出力/ })).toBeInTheDocument();
  });

  it("PDFボタンをクリックするとdownloadReportPDFが呼ばれる", async () => {
    vi.mocked(pdfLib.downloadReportPDF).mockResolvedValue(undefined);
    const input = makeInput();
    const result = makeResult();
    render(<MobileSummaryCard result={result} input={input} yieldPct={9.0} netYieldPct={6.5} />);
    await userEvent.click(screen.getByRole("button", { name: /詳細PDFを出力/ }));
    await waitFor(() => {
      expect(pdfLib.downloadReportPDF).toHaveBeenCalledWith(input, result);
    });
  });

  it("PDF生成中はボタンが「生成中...」になり無効化される", async () => {
    let resolve: (v: undefined) => void;
    const pending = new Promise<undefined>((r) => {
      resolve = r;
    });
    vi.mocked(pdfLib.downloadReportPDF).mockReturnValue(pending);
    render(
      <MobileSummaryCard
        result={makeResult()}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    const pdfButton = screen.getByRole("button", { name: /詳細PDFを出力/ });
    await userEvent.click(pdfButton);
    expect(screen.getByRole("button", { name: /生成中/ })).toBeDisabled();
    act(() => {
      resolve!(undefined);
    });
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /詳細PDFを出力/ })).not.toBeDisabled();
    });
  });

  it("PDF生成エラー時にエラーメッセージが表示される", async () => {
    vi.mocked(pdfLib.downloadReportPDF).mockRejectedValue(new Error("PDF error"));
    render(
      <MobileSummaryCard
        result={makeResult()}
        input={makeInput()}
        yieldPct={9.0}
        netYieldPct={6.5}
      />
    );
    await userEvent.click(screen.getByRole("button", { name: /詳細PDFを出力/ }));
    await waitFor(() => {
      expect(screen.getByText(/PDF生成に失敗しました/)).toBeInTheDocument();
    });
  });

  it("必要頭金が表示される", () => {
    // landPrice + buildingCost - loanAmount = 10M + 5M - 13M = 2M
    const input = makeInput({
      landPrice: 10_000_000,
      buildingCost: 5_000_000,
      loanAmount: 13_000_000,
    });
    render(
      <MobileSummaryCard result={makeResult()} input={input} yieldPct={9.0} netYieldPct={6.5} />
    );
    expect(screen.getByText("必要頭金")).toBeInTheDocument();
  });
});

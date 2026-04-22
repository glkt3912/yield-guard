import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Dashboard } from "@/components/Dashboard";
import { makeResult } from "./helpers";

// Mock API module
vi.mock("@/lib/api", () => ({
  analyze: vi.fn(),
  compareLandPrice: vi.fn(),
  estimateLandPrice: vi.fn(),
  fetchStationRidership: vi.fn(),
  fetchPopulationForecast: vi.fn(),
  fetchLandAppraisals: vi.fn(),
}));

import * as api from "@/lib/api";

describe("Dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("初期状態でプレースホルダーが表示される", () => {
    render(<Dashboard />);
    expect(screen.getByText(/左のフォームから条件を入力して/)).toBeInTheDocument();
  });

  it("analyzeAPIの応答後にYieldAnalysisが表示される（クイックモード）", async () => {
    const mockResult = makeResult({ grossYield: 0.09, isAbove8Percent: true });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      // YieldAnalysis が表示される（表面利回り数値）
      expect(screen.getByText("9.00")).toBeInTheDocument();
    });

    // クイックモードではCashFlowChartは非表示
    expect(screen.queryByText(/キャッシュフロー推移/)).not.toBeInTheDocument();
  });

  it("詳細モードでanalyzeAPIの応答後にCashFlowChartが表示される", async () => {
    const mockResult = makeResult({ grossYield: 0.09, isAbove8Percent: true });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    render(<Dashboard />);

    // 詳細モードに切り替え
    await userEvent.click(screen.getByRole("radio", { name: /詳細/ }));

    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      expect(screen.getByText("9.00")).toBeInTheDocument();
    });

    // CashFlowChart が表示される（CardTitle のテキスト）
    expect(screen.getByText(/キャッシュフロー推移/)).toBeInTheDocument();
  });

  it("API呼び出し中はローディング状態でボタンが無効化される", async () => {
    // analyzeが解決されない promise を返す
    let resolve: (v: ReturnType<typeof makeResult>) => void;
    const pending = new Promise<ReturnType<typeof makeResult>>((r) => { resolve = r; });
    vi.mocked(api.analyze).mockReturnValue(pending);

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    // ボタンが無効化されていること
    expect(screen.getByRole("button", { name: /シミュレーション実行/ })).toBeDisabled();

    // 後始末
    resolve!(makeResult());
  });

  it("APIエラー時にエラーメッセージが表示される", async () => {
    vi.mocked(api.analyze).mockRejectedValue(new Error("サーバーエラー"));

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      expect(screen.getByText(/サーバーエラー/)).toBeInTheDocument();
    });
  });

  it("APIエラーがError以外の場合はフォールバックメッセージが表示される", async () => {
    vi.mocked(api.analyze).mockRejectedValue("unknown error");

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      expect(screen.getByText(/シミュレーションに失敗しました/)).toBeInTheDocument();
    });
  });
});

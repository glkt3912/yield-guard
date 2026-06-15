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
  fetchUrbanRisks: vi.fn(),
  fetchHazardInfo: vi.fn(),
  fetchInvestmentScore: vi.fn(),
  simulate: vi.fn(),
  fetchMunicipalities: vi.fn().mockResolvedValue([]),
  fetchGeocode: vi.fn(),
  fetchRentDeclineHint: vi.fn(),
}));

// PDFDownloadLink is a browser-only API; stub it out in jsdom
vi.mock("next/dynamic", () => ({
  default: () => () => null,
}));

// WatchlistPanel and AreaDiscovery use localStorage/external APIs not needed for Dashboard unit tests
vi.mock("@/components/WatchlistPanel", () => ({ default: () => null }));
vi.mock("@/components/AreaDiscovery", () => ({ AreaDiscovery: () => null }));

// Mock next/navigation for useRouter
const mockReplace = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mockReplace }),
}));

import * as api from "@/lib/api";

describe("Dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockReplace.mockClear();
  });

  it("初期状態でプレースホルダーが表示される", () => {
    render(<Dashboard />);
    expect(
      screen.getByText(/条件を入力してシミュレーションを実行してください/)
    ).toBeInTheDocument();
  });

  it("analyzeAPIの応答後にYieldAnalysisが表示される（クイックモード）", async () => {
    const mockResult = makeResult({
      grossYield: 0.09,
      marketGrossYield: 0.09,
      isAboveYieldTarget: true,
    });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    // 結果タブが出るまで待つ
    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "財務分析" })).toBeInTheDocument();
    });

    // 財務分析タブに切り替えてYieldAnalysisを確認
    await userEvent.click(screen.getByRole("tab", { name: "財務分析" }));

    await waitFor(() => {
      // YieldAnalysis が表示される（表面利回り数値）
      expect(screen.getByText("9.00")).toBeInTheDocument();
    });

    // クイックモードではCashFlowChartは非表示
    expect(screen.queryByText(/キャッシュフロー推移/)).not.toBeInTheDocument();
  });

  it("詳細モードでanalyzeAPIの応答後にCashFlowChartが表示される", async () => {
    const mockResult = makeResult({
      grossYield: 0.09,
      marketGrossYield: 0.09,
      isAboveYieldTarget: true,
    });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    render(<Dashboard />);

    // 詳細モードに切り替え（Dashboard は複数箇所にトグルを持つため最初の要素を使う）
    await userEvent.click(screen.getAllByRole("radio", { name: /くわしく分析/ })[0]);

    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    // 結果タブが出るまで待つ
    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "財務分析" })).toBeInTheDocument();
    });

    // 財務分析タブに切り替えてYieldAnalysis・CashFlowChartを確認
    await userEvent.click(screen.getByRole("tab", { name: "財務分析" }));

    await waitFor(() => {
      expect(screen.getByText("9.00")).toBeInTheDocument();
    });

    // CashFlowChart が表示される（CardTitle のテキスト）
    expect(screen.getByText(/キャッシュフロー推移/)).toBeInTheDocument();
  });

  it("API呼び出し中はローディング状態でボタンが無効化される", async () => {
    // analyzeが解決されない promise を返す
    let resolve: (v: ReturnType<typeof makeResult>) => void;
    const pending = new Promise<ReturnType<typeof makeResult>>((r) => {
      resolve = r;
    });
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

  it("シミュレーション実行後にrouter.replaceでURLが更新される", async () => {
    const mockResult = makeResult({
      grossYield: 0.09,
      marketGrossYield: 0.09,
      isAboveYieldTarget: true,
    });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      expect(mockReplace).toHaveBeenCalled();
    });

    const calledWith = mockReplace.mock.calls[0][0] as string;
    // URLにmodeパラメータが含まれること
    expect(calledWith).toContain("mode=quick");
    // クイックモードのtotalPriceが含まれること
    expect(calledWith).toContain("totalPrice=1500");
  });

  it("initialParamsでクイックモードが復元される", () => {
    const params = new URLSearchParams("mode=quick&totalPrice=2000&rent=12");
    render(<Dashboard initialParams={params} />);
    // クイックモードのラジオが選択されていること
    expect(screen.getAllByRole("radio", { name: /かんたん判定/ })[0]).toHaveAttribute(
      "aria-checked",
      "true"
    );
    // 物件価格フィールドが復元されていること
    expect(screen.getByLabelText(/物件価格（土地＋建物の総額）/)).toHaveValue("2000");
  });

  it("initialParamsでフルモードが復元される", () => {
    const params = new URLSearchParams("mode=full&landPrice=1000&buildingCost=500");
    render(<Dashboard initialParams={params} />);
    // フルモードのラジオが選択されていること
    expect(screen.getAllByRole("radio", { name: /くわしく分析/ })[0]).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  it("シミュレーション結果表示後にURL共有ボタンが表示される", async () => {
    const mockResult = makeResult({
      grossYield: 0.09,
      marketGrossYield: 0.09,
      isAboveYieldTarget: true,
    });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /この条件をクリップボードにコピー/ })
      ).toBeInTheDocument();
    });
  });

  it("URL共有ボタンを押すとクリップボードにURLがコピーされる", async () => {
    const mockResult = makeResult({
      grossYield: 0.09,
      marketGrossYield: 0.09,
      isAboveYieldTarget: true,
    });
    vi.mocked(api.analyze).mockResolvedValue(mockResult);

    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });

    render(<Dashboard />);

    await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /この条件をクリップボードにコピー/ })
      ).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: /この条件をクリップボードにコピー/ }));

    expect(writeText).toHaveBeenCalledTimes(1);
    expect(writeText).toHaveBeenCalledWith(expect.any(String));
  });
});

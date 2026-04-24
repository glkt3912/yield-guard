import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InvestmentForm } from "@/components/InvestmentForm";
import { DEFAULT_INPUT, QUICK_MODE_DEFAULTS } from "@/types/investment";
import type { SimulationMode } from "@/types/investment";

vi.mock("@/lib/api", () => ({
  fetchMunicipalities: vi.fn().mockResolvedValue([]),
  fetchRentDeclineHint: vi.fn(),
}));

import * as api from "@/lib/api";

function renderForm(mode: SimulationMode = "full") {
  const onAnalyze = vi.fn().mockResolvedValue(undefined);
  const onFetchLandPrices = vi.fn().mockResolvedValue(undefined);
  const onModeChange = vi.fn();
  render(
    <InvestmentForm
      onAnalyze={onAnalyze}
      onFetchLandPrices={onFetchLandPrices}
      loading={false}
      simulationMode={mode}
      onModeChange={onModeChange}
    />
  );
  return { onAnalyze, onFetchLandPrices, onModeChange };
}

describe("InvestmentForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchMunicipalities).mockResolvedValue([]);
  });

  it("シミュレーション実行ボタンをクリックすると onAnalyze が呼ばれる", async () => {
    const onAnalyze = vi.fn().mockResolvedValue(undefined);
    const onFetchLandPrices = vi.fn().mockResolvedValue(undefined);
    const onModeChange = vi.fn();
    render(<InvestmentForm onAnalyze={onAnalyze} onFetchLandPrices={onFetchLandPrices} loading={false} simulationMode="full" onModeChange={onModeChange} />);

    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));
    expect(onAnalyze).toHaveBeenCalledTimes(1);
    expect(onAnalyze).toHaveBeenCalledWith(expect.objectContaining({
      landPrice: DEFAULT_INPUT.landPrice,
      buildingType: DEFAULT_INPUT.buildingType,
    }));
  });

  it("相場データを取得ボタンをクリックすると onFetchLandPrices が呼ばれる", async () => {
    const onAnalyze = vi.fn().mockResolvedValue(undefined);
    const onFetchLandPrices = vi.fn().mockResolvedValue(undefined);
    const onModeChange = vi.fn();
    render(<InvestmentForm onAnalyze={onAnalyze} onFetchLandPrices={onFetchLandPrices} loading={false} simulationMode="full" onModeChange={onModeChange} />);

    await userEvent.click(screen.getByRole("button", { name: /相場データを取得/ }));
    expect(onFetchLandPrices).toHaveBeenCalledTimes(1);
  });

  it("loading=true のとき操作ボタンが無効化される", () => {
    render(
      <InvestmentForm
        onAnalyze={vi.fn()}
        onFetchLandPrices={vi.fn()}
        loading={true}
        simulationMode="full"
        onModeChange={vi.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /シミュレーション実行/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /相場データを取得/ })).toBeDisabled();
  });

  describe("用途地域リスク警告", () => {
    function renderZoningForm() {
      render(<InvestmentForm onAnalyze={vi.fn()} onFetchLandPrices={vi.fn()} loading={false} simulationMode="full" onModeChange={vi.fn()} />);
      return screen.getByLabelText("用途地域（任意）");
    }

    it("未選択時はリスクバナーが表示されない", () => {
      renderZoningForm();
      expect(screen.queryByText(/リスク/)).not.toBeInTheDocument();
      expect(screen.queryByText(/良好な住環境/)).not.toBeInTheDocument();
    });

    it("第一種低層住居専用地域を選択すると良好な住環境バナーが表示される", async () => {
      const select = renderZoningForm();
      await userEvent.selectOptions(select, "第一種低層住居専用地域");
      expect(screen.getByText("良好な住環境です")).toBeInTheDocument();
    });

    it("近隣商業地域を選択すると低リスクバナーが表示される", async () => {
      const select = renderZoningForm();
      await userEvent.selectOptions(select, "近隣商業地域");
      expect(screen.getByText(/低リスク/)).toBeInTheDocument();
    });

    it("準工業地域を選択すると中リスクバナーが表示される", async () => {
      const select = renderZoningForm();
      await userEvent.selectOptions(select, "準工業地域");
      expect(screen.getByText(/中リスク/)).toBeInTheDocument();
    });

    it("工業地域を選択すると高リスクバナーが表示される", async () => {
      const select = renderZoningForm();
      await userEvent.selectOptions(select, "工業地域");
      expect(screen.getByText(/高リスク/)).toBeInTheDocument();
      expect(screen.getByText(/住宅需要の低下/)).toBeInTheDocument();
    });

    it("工業専用地域を選択すると高リスクバナーが表示される", async () => {
      const select = renderZoningForm();
      await userEvent.selectOptions(select, "工業専用地域");
      expect(screen.getByText(/高リスク/)).toBeInTheDocument();
      expect(screen.getByText(/住宅建設が法律上禁止/)).toBeInTheDocument();
    });

    it("市街化調整区域を選択すると高リスクバナーが表示される", async () => {
      const select = renderZoningForm();
      await userEvent.selectOptions(select, "市街化調整区域");
      expect(screen.getByText(/高リスク/)).toBeInTheDocument();
      expect(screen.getByText(/再建築不可リスク/)).toBeInTheDocument();
    });
  });

  it("詳細モードでは諸経費率フィールドが常時表示される", () => {
    render(
      <InvestmentForm onAnalyze={vi.fn()} onFetchLandPrices={vi.fn()} loading={false} simulationMode="full" onModeChange={vi.fn()} />
    );
    expect(screen.getByLabelText(/諸経費率/)).toBeInTheDocument();
  });

  describe("クイックシミュレーションモード", () => {
    it("クイックモードでは土地面積フィールドが非表示", () => {
      renderForm("quick");
      expect(screen.queryByLabelText(/土地面積/)).not.toBeInTheDocument();
    });

    it("クイックモードでは築年数フィールドが非表示", () => {
      renderForm("quick");
      expect(screen.queryByLabelText(/築年数/)).not.toBeInTheDocument();
    });

    it("クイックモードでは建物構造フィールドが非表示", () => {
      renderForm("quick");
      expect(screen.queryByLabelText(/建物構造/)).not.toBeInTheDocument();
    });

    it("クイックモードでは詳細設定トグルが非表示", () => {
      renderForm("quick");
      expect(screen.queryByText(/詳細設定（諸経費率・空室率など）/)).not.toBeInTheDocument();
    });

    it("クイックモードではストレステストセクションが非表示", () => {
      renderForm("quick");
      expect(screen.queryByText("ストレステスト")).not.toBeInTheDocument();
    });

    it("詳細モードでは土地面積フィールドが表示される", () => {
      renderForm("full");
      expect(screen.getByLabelText(/土地面積/)).toBeInTheDocument();
    });

    it("クイックモードでシミュレーション実行すると QUICK_MODE_DEFAULTS がマージされる", async () => {
      const { onAnalyze } = renderForm("quick");
      await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
      await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));
      expect(onAnalyze).toHaveBeenCalledTimes(1);
      expect(onAnalyze).toHaveBeenCalledWith(
        expect.objectContaining({
          vacancyRate: QUICK_MODE_DEFAULTS.vacancyRate,
          expenseRate: QUICK_MODE_DEFAULTS.expenseRate,
        }),
        "1500"
      );
    });

    it("詳細ボタンをクリックすると onModeChange('full') が呼ばれる", async () => {
      const { onModeChange } = renderForm("quick");
      await userEvent.click(screen.getByRole("radio", { name: /詳細/ }));
      expect(onModeChange).toHaveBeenCalledWith("full");
    });

    it("クイックモードのラジオボタンは aria-checked=true", () => {
      renderForm("quick");
      const quickBtn = screen.getByRole("radio", { name: /クイック/ });
      expect(quickBtn).toHaveAttribute("aria-checked", "true");
    });

    it("クイックモードではインフォバナーが表示される", () => {
      renderForm("quick");
      expect(screen.getByText(/自動設定されるデフォルト値/)).toBeInTheDocument();
    });

    it("詳細モードではクイックモードのインフォバナーが非表示", () => {
      renderForm("full");
      expect(screen.queryByText(/クイックモード:/)).not.toBeInTheDocument();
    });
  });

  describe("現金購入チェックボックス（クイックモード）", () => {
    it("クイックモードでは現金購入チェックボックスが表示される", () => {
      renderForm("quick");
      expect(screen.getByLabelText(/現金購入（ローンなし）/)).toBeInTheDocument();
    });

    it("詳細モードでも現金購入チェックボックスが表示される", () => {
      renderForm("full");
      expect(screen.getByLabelText(/現金購入（ローンなし）/)).toBeInTheDocument();
    });

    it("詳細モードで現金購入チェックを入れるとローン関連フィールドが無効化される", async () => {
      renderForm("full");
      const checkbox = screen.getByLabelText(/現金購入（ローンなし）/);
      await userEvent.click(checkbox);
      expect(screen.getByLabelText(/ローン金額/)).toBeDisabled();
      expect(screen.getByLabelText(/返済期間/)).toBeDisabled();
    });

    it("現金購入チェックを入れるとローン80%自動適用メッセージが非表示になる", async () => {
      renderForm("quick");
      expect(screen.getByText(/ローン 80% 自動適用中/)).toBeInTheDocument();
      const checkbox = screen.getByLabelText(/現金購入（ローンなし）/);
      await userEvent.click(checkbox);
      expect(screen.queryByText(/ローン 80% 自動適用中/)).not.toBeInTheDocument();
    });

    it("カスタム設定リンクをクリックするとローン金額入力欄が表示される", async () => {
      renderForm("quick");
      expect(screen.queryByLabelText(/ローン金額/)).not.toBeInTheDocument();
      await userEvent.click(screen.getByText(/カスタム設定/));
      expect(screen.getByLabelText(/ローン金額/)).toBeInTheDocument();
    });

    it("カスタム設定後に80%自動計算に戻すリンクをクリックするとローン金額入力欄が非表示になる", async () => {
      renderForm("quick");
      await userEvent.click(screen.getByText(/カスタム設定/));
      expect(screen.getByLabelText(/ローン金額/)).toBeInTheDocument();
      await userEvent.click(screen.getByText(/80% 自動計算に戻す/));
      expect(screen.queryByLabelText(/ローン金額/)).not.toBeInTheDocument();
    });

    it("現金購入チェック時にシミュレーションを実行するとloanAmountが0で送信される", async () => {
      const { onAnalyze } = renderForm("quick");
      await userEvent.type(screen.getByLabelText(/物件価格（土地＋建物の総額）/), "1500");
      await userEvent.click(screen.getByLabelText(/現金購入（ローンなし）/));
      await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));
      expect(onAnalyze).toHaveBeenCalledWith(
        expect.objectContaining({
          loanAmount: 0,
          loanYears: 0,
        }),
        "1500"
      );
    });

    it("現金購入チェック時にインフォバナーが「現金購入（ローンなし）」に変わる", async () => {
      renderForm("quick");
      const checkbox = screen.getByLabelText(/現金購入（ローンなし）/);
      await userEvent.click(checkbox);
      expect(screen.getAllByText(/現金購入（ローンなし）/).length).toBeGreaterThan(0);
    });

    it("現金購入チェックオン時に showCustomLoan がリセットされる", async () => {
      renderForm("quick");
      await userEvent.click(screen.getByText(/カスタム設定/));
      expect(screen.getByLabelText(/ローン金額/)).toBeInTheDocument();
      await userEvent.click(screen.getByLabelText(/現金購入（ローンなし）/));
      await userEvent.click(screen.getByLabelText(/現金購入（ローンなし）/));
      expect(screen.queryByLabelText(/ローン金額/)).not.toBeInTheDocument();
      expect(screen.getByText(/ローン 80% 自動適用中/)).toBeInTheDocument();
    });

    it("前回の入力から開始ボタンで直前の入力が復元される", async () => {
      const store: Record<string, string> = {
        "yield-guard:quick-history": JSON.stringify([
          { totalPriceMan: "3000", monthlyRentYen: "150000", ts: Date.now() },
        ]),
      };
      vi.stubGlobal("localStorage", {
        getItem: (key: string) => store[key] ?? null,
        setItem: (key: string, value: string) => { store[key] = value; },
        removeItem: (key: string) => { delete store[key]; },
        clear: () => { Object.keys(store).forEach((k) => delete store[k]); },
      });
      renderForm("quick");
      await userEvent.click(screen.getByText(/前回の入力から開始/));
      expect(screen.getByLabelText(/物件価格（土地＋建物の総額）/)).toHaveValue(3000);
      expect(screen.getByLabelText(/想定月額賃料/)).toHaveValue(150000);
      vi.unstubAllGlobals();
    });
  });

  describe("市区町村フェッチエラー", () => {
    it("詳細モードで市区町村の取得に失敗するとエラーメッセージが表示される", async () => {
      vi.mocked(api.fetchMunicipalities).mockRejectedValue(new Error("ネットワークエラー"));
      renderForm("full");
      await waitFor(() => {
        expect(screen.getByText("ネットワークエラー")).toBeInTheDocument();
      });
    });

    it("fetchMunicipalities がError以外を投げた場合もフォールバックメッセージが表示される", async () => {
      vi.mocked(api.fetchMunicipalities).mockRejectedValue("unknown");
      renderForm("full");
      await waitFor(() => {
        expect(screen.getByText("市区町村の取得に失敗しました")).toBeInTheDocument();
      });
    });

    it("都道府県を変更すると前のエラーがクリアされる", async () => {
      vi.mocked(api.fetchMunicipalities)
        .mockRejectedValueOnce(new Error("ネットワークエラー"))
        .mockResolvedValueOnce([]);
      renderForm("full");
      await waitFor(() => {
        expect(screen.getByText("ネットワークエラー")).toBeInTheDocument();
      });
      await userEvent.selectOptions(screen.getByLabelText("都道府県"), "13");
      await waitFor(() => {
        expect(screen.queryByText("ネットワークエラー")).not.toBeInTheDocument();
      });
    });
  });
});

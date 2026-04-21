import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InvestmentForm } from "@/components/InvestmentForm";
import { DEFAULT_INPUT, QUICK_MODE_DEFAULTS } from "@/types/investment";
import type { SimulationMode } from "@/types/investment";

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

  it("詳細設定を開く/閉じるトグルが機能する", async () => {
    render(
      <InvestmentForm onAnalyze={vi.fn()} onFetchLandPrices={vi.fn()} loading={false} simulationMode="full" onModeChange={vi.fn()} />
    );
    expect(screen.queryByText("諸経費率")).not.toBeInTheDocument();
    await userEvent.click(screen.getByText(/詳細設定（諸経費率・空室率など）/));
    expect(screen.getByText(/諸経費率/)).toBeInTheDocument();
    await userEvent.click(screen.getByText(/詳細設定を閉じる/));
    expect(screen.queryByText("諸経費率")).not.toBeInTheDocument();
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
      await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));
      expect(onAnalyze).toHaveBeenCalledTimes(1);
      expect(onAnalyze).toHaveBeenCalledWith(expect.objectContaining({
        vacancyRate: QUICK_MODE_DEFAULTS.vacancyRate,
        expenseRate: QUICK_MODE_DEFAULTS.expenseRate,
      }));
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
      expect(screen.getByText(/クイックモード:/)).toBeInTheDocument();
    });

    it("詳細モードではクイックモードのインフォバナーが非表示", () => {
      renderForm("full");
      expect(screen.queryByText(/クイックモード:/)).not.toBeInTheDocument();
    });
  });
});

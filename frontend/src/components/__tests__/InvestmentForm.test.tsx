import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InvestmentForm } from "@/components/InvestmentForm";
import { DEFAULT_INPUT } from "@/types/investment";

describe("InvestmentForm", () => {
  it("シミュレーション実行ボタンをクリックすると onAnalyze が呼ばれる", async () => {
    const onAnalyze = vi.fn().mockResolvedValue(undefined);
    const onFetchLandPrices = vi.fn().mockResolvedValue(undefined);
    render(<InvestmentForm onAnalyze={onAnalyze} onFetchLandPrices={onFetchLandPrices} loading={false} />);

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
    render(<InvestmentForm onAnalyze={onAnalyze} onFetchLandPrices={onFetchLandPrices} loading={false} />);

    await userEvent.click(screen.getByRole("button", { name: /相場データを取得/ }));
    expect(onFetchLandPrices).toHaveBeenCalledTimes(1);
  });

  it("loading=true のとき操作ボタンが無効化される", () => {
    render(
      <InvestmentForm
        onAnalyze={vi.fn()}
        onFetchLandPrices={vi.fn()}
        loading={true}
      />
    );
    expect(screen.getByRole("button", { name: /シミュレーション実行/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /相場データを取得/ })).toBeDisabled();
  });

  describe("用途地域リスク警告", () => {
    function renderForm() {
      render(<InvestmentForm onAnalyze={vi.fn()} onFetchLandPrices={vi.fn()} loading={false} />);
      return screen.getByLabelText("用途地域（任意）");
    }

    it("未選択時はリスクバナーが表示されない", () => {
      renderForm();
      expect(screen.queryByText(/リスク/)).not.toBeInTheDocument();
      expect(screen.queryByText(/良好な住環境/)).not.toBeInTheDocument();
    });

    it("第一種低層住居専用地域を選択すると良好な住環境バナーが表示される", async () => {
      const select = renderForm();
      await userEvent.selectOptions(select, "第一種低層住居専用地域");
      expect(screen.getByText("良好な住環境です")).toBeInTheDocument();
    });

    it("近隣商業地域を選択すると低リスクバナーが表示される", async () => {
      const select = renderForm();
      await userEvent.selectOptions(select, "近隣商業地域");
      expect(screen.getByText(/低リスク/)).toBeInTheDocument();
    });

    it("準工業地域を選択すると中リスクバナーが表示される", async () => {
      const select = renderForm();
      await userEvent.selectOptions(select, "準工業地域");
      expect(screen.getByText(/中リスク/)).toBeInTheDocument();
    });

    it("工業地域を選択すると高リスクバナーが表示される", async () => {
      const select = renderForm();
      await userEvent.selectOptions(select, "工業地域");
      expect(screen.getByText(/高リスク/)).toBeInTheDocument();
      expect(screen.getByText(/住宅需要の低下/)).toBeInTheDocument();
    });

    it("工業専用地域を選択すると高リスクバナーが表示される", async () => {
      const select = renderForm();
      await userEvent.selectOptions(select, "工業専用地域");
      expect(screen.getByText(/高リスク/)).toBeInTheDocument();
      expect(screen.getByText(/住宅建設が法律上禁止/)).toBeInTheDocument();
    });

    it("市街化調整区域を選択すると高リスクバナーが表示される", async () => {
      const select = renderForm();
      await userEvent.selectOptions(select, "市街化調整区域");
      expect(screen.getByText(/高リスク/)).toBeInTheDocument();
      expect(screen.getByText(/再建築不可リスク/)).toBeInTheDocument();
    });
  });

  it("詳細設定を開く/閉じるトグルが機能する", async () => {
    render(
      <InvestmentForm onAnalyze={vi.fn()} onFetchLandPrices={vi.fn()} loading={false} />
    );
    expect(screen.queryByText("諸経費率")).not.toBeInTheDocument();
    await userEvent.click(screen.getByText(/詳細設定（諸経費率・空室率など）/));
    expect(screen.getByText(/諸経費率/)).toBeInTheDocument();
    await userEvent.click(screen.getByText(/詳細設定を閉じる/));
    expect(screen.queryByText("諸経費率")).not.toBeInTheDocument();
  });
});

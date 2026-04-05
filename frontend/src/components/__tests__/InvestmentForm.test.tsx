import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InvestmentForm } from "@/components/InvestmentForm";
import { vi } from "vitest";

describe("InvestmentForm", () => {
  it("シミュレーション実行ボタンをクリックすると onAnalyze が呼ばれる", async () => {
    const onAnalyze = vi.fn().mockResolvedValue(undefined);
    const onFetchLandPrices = vi.fn().mockResolvedValue(undefined);
    render(<InvestmentForm onAnalyze={onAnalyze} onFetchLandPrices={onFetchLandPrices} loading={false} />);

    await userEvent.click(screen.getByRole("button", { name: /シミュレーション実行/ }));
    expect(onAnalyze).toHaveBeenCalledTimes(1);
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

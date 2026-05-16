import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import RenovationPanel from "@/components/RenovationPanel";
import type { RenovationResult } from "@/types/investment";

vi.mock("@/lib/api", () => ({
  analyzeRenovation: vi.fn(),
}));

import * as api from "@/lib/api";

function makeRenovationResult(overrides: Partial<RenovationResult> = {}): RenovationResult {
  return {
    recoveryYears: 5.5,
    isRecoverable: true,
    taxSavings: 150_000,
    virtualLaborCost: 80_000,
    capitalExpenditures: 800_000,
    repairExpenses: 200_000,
    actualYield: 0.09,
    totalRenovationCost: 1_000_000,
    annualRentIncrease: 180_000,
    classifiedItems: [
      {
        name: "内装",
        cost: 500_000,
        expectedMonthlyRentIncrease: 10_000,
        isSelfWork: false,
        selfLaborHours: 0,
        isCapitalExpenditure: true,
        virtualLaborCost: 0,
      },
      {
        name: "外壁",
        cost: 500_000,
        expectedMonthlyRentIncrease: 5_000,
        isSelfWork: false,
        selfLaborHours: 0,
        isCapitalExpenditure: false,
        virtualLaborCost: 0,
      },
    ],
    ...overrides,
  };
}

describe("RenovationPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("カードタイトルが表示される", () => {
    render(<RenovationPanel />);
    expect(screen.getByText("修繕費回収期間シミュレーション")).toBeInTheDocument();
  });

  it("初期状態で工事項目が1行表示される", () => {
    render(<RenovationPanel />);
    // 1行目のname inputがある
    expect(screen.getByPlaceholderText("例：内装")).toBeInTheDocument();
  });

  it("「＋ 行を追加」ボタンをクリックすると工事項目が増える", async () => {
    render(<RenovationPanel />);
    const inputs = screen.getAllByPlaceholderText("例：内装");
    expect(inputs).toHaveLength(1);

    await userEvent.click(screen.getByRole("button", { name: "工事項目を追加" }));

    expect(screen.getAllByPlaceholderText("例：内装")).toHaveLength(2);
  });

  it("工事項目が2件以上のとき削除ボタンが表示される", async () => {
    render(<RenovationPanel />);
    await userEvent.click(screen.getByRole("button", { name: "工事項目を追加" }));

    expect(screen.getAllByRole("button", { name: /工事項目.*を削除/ })).toHaveLength(2);
  });

  it("削除ボタンをクリックすると工事項目が減る", async () => {
    render(<RenovationPanel />);
    await userEvent.click(screen.getByRole("button", { name: "工事項目を追加" }));
    expect(screen.getAllByPlaceholderText("例：内装")).toHaveLength(2);

    await userEvent.click(screen.getAllByRole("button", { name: /工事項目.*を削除/ })[0]);
    expect(screen.getAllByPlaceholderText("例：内装")).toHaveLength(1);
  });

  it("工事項目が1件のときは削除ボタンが表示されない", () => {
    render(<RenovationPanel />);
    expect(screen.queryByRole("button", { name: /工事項目.*を削除/ })).not.toBeInTheDocument();
  });

  it("セルフチェックボックスをオンにすると工数フィールドが表示される", async () => {
    render(<RenovationPanel />);
    const checkbox = screen.getAllByRole("checkbox")[0];
    expect(checkbox).not.toBeChecked();

    await userEvent.click(checkbox);
    expect(checkbox).toBeChecked();
    // 工数入力が出現する
    expect(screen.getAllByDisplayValue("0").length).toBeGreaterThan(0);
  });

  it("工事費が0のままで実行するとバリデーションエラーが表示される", async () => {
    render(<RenovationPanel />);
    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    expect(screen.getByText("工事費は正の値を入力してください")).toBeInTheDocument();
    expect(api.analyzeRenovation).not.toHaveBeenCalled();
  });

  it("工事項目が0件で実行するとバリデーションエラーが表示される", async () => {
    render(<RenovationPanel />);
    // 全項目削除
    await userEvent.click(screen.getByRole("button", { name: "工事項目を追加" }));
    const deleteButtons = screen.getAllByRole("button", { name: /工事項目.*を削除/ });
    await userEvent.click(deleteButtons[1]);
    await userEvent.click(screen.getAllByRole("button", { name: /工事項目.*を削除/ })[0]);

    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    expect(screen.getByText("工事項目を1件以上追加してください")).toBeInTheDocument();
    expect(api.analyzeRenovation).not.toHaveBeenCalled();
  });

  it("正常な入力で実行するとAPIが呼ばれ結果が表示される", async () => {
    vi.mocked(api.analyzeRenovation).mockResolvedValue(makeRenovationResult());
    render(<RenovationPanel />);

    // 工事費を入力（100万円）
    const costInputs = screen.getAllByRole("spinbutton");
    // 工事費フィールドは2番目のspinbutton（名前欄はtextarea）
    // items[0].cost フィールド — 0 から 100 万に変更
    await userEvent.clear(
      costInputs.find((el) => (el as HTMLInputElement).min === "0" && el.closest("td"))!
    );
    // 最初の工事費セルに入力
    const costCell = screen
      .getAllByRole("spinbutton")
      .find((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    if (costCell) {
      await userEvent.clear(costCell);
      await userEvent.type(costCell, "100");
    }

    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    await waitFor(() => {
      expect(api.analyzeRenovation).toHaveBeenCalledTimes(1);
    });
  });

  it("API成功後に修繕費回収期間が表示される", async () => {
    vi.mocked(api.analyzeRenovation).mockResolvedValue(
      makeRenovationResult({ recoveryYears: 5.5 })
    );
    render(<RenovationPanel />);

    // 工事費を入力
    const costInputs = screen
      .getAllByRole("spinbutton")
      .filter((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    await userEvent.clear(costInputs[0]);
    await userEvent.type(costInputs[0], "100");

    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    await waitFor(() => {
      expect(screen.getByText("5.5年")).toBeInTheDocument();
    });
    expect(screen.getByText("修繕費回収期間")).toBeInTheDocument();
  });

  it("API成功後に工事分類結果テーブルが表示される", async () => {
    vi.mocked(api.analyzeRenovation).mockResolvedValue(makeRenovationResult());
    render(<RenovationPanel />);

    const costInputs = screen
      .getAllByRole("spinbutton")
      .filter((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    await userEvent.clear(costInputs[0]);
    await userEvent.type(costInputs[0], "100");
    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    await waitFor(() => {
      expect(screen.getByText("工事分類結果")).toBeInTheDocument();
    });
    expect(screen.getByText("資本的支出")).toBeInTheDocument();
    expect(screen.getByText("修繕費")).toBeInTheDocument();
  });

  it("isRecoverable=false のとき「回収不可」が表示される", async () => {
    vi.mocked(api.analyzeRenovation).mockResolvedValue(
      makeRenovationResult({ isRecoverable: false, recoveryYears: 0 })
    );
    render(<RenovationPanel />);

    const costInputs = screen
      .getAllByRole("spinbutton")
      .filter((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    await userEvent.clear(costInputs[0]);
    await userEvent.type(costInputs[0], "100");
    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    await waitFor(() => {
      expect(screen.getByText("回収不可")).toBeInTheDocument();
    });
  });

  it("APIエラー時にエラーメッセージが表示される", async () => {
    vi.mocked(api.analyzeRenovation).mockRejectedValue(new Error("計算エラー"));
    render(<RenovationPanel />);

    const costInputs = screen
      .getAllByRole("spinbutton")
      .filter((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    await userEvent.clear(costInputs[0]);
    await userEvent.type(costInputs[0], "100");
    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    await waitFor(() => {
      expect(screen.getByText("計算エラー")).toBeInTheDocument();
    });
  });

  it("APIがError以外を投げた場合はフォールバックメッセージが表示される", async () => {
    vi.mocked(api.analyzeRenovation).mockRejectedValue("unknown");
    render(<RenovationPanel />);

    const costInputs = screen
      .getAllByRole("spinbutton")
      .filter((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    await userEvent.clear(costInputs[0]);
    await userEvent.type(costInputs[0], "100");
    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    await waitFor(() => {
      expect(screen.getByText("エラーが発生しました")).toBeInTheDocument();
    });
  });

  it("計算中はボタンが無効化される", async () => {
    let resolve: (v: RenovationResult) => void;
    const pending = new Promise<RenovationResult>((r) => {
      resolve = r;
    });
    vi.mocked(api.analyzeRenovation).mockReturnValue(pending);
    render(<RenovationPanel />);

    const costInputs = screen
      .getAllByRole("spinbutton")
      .filter((el) => el.closest("td") !== null && (el as HTMLInputElement).min === "0");
    await userEvent.clear(costInputs[0]);
    await userEvent.type(costInputs[0], "100");
    await userEvent.click(screen.getByRole("button", { name: "リフォーム分析を実行" }));

    expect(screen.getByRole("button", { name: "計算中..." })).toBeDisabled();

    await waitFor(async () => {
      resolve!(makeRenovationResult());
    });
  });
});

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LoanComparePanel } from "@/components/LoanComparePanel";
import { makeInput, makeResult } from "./helpers";

vi.mock("@/lib/api", () => ({
  analyze: vi.fn(),
}));

import * as api from "@/lib/api";

describe("LoanComparePanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("カードタイトルが表示される", () => {
    render(<LoanComparePanel baseInput={makeInput()} />);
    expect(screen.getByText("複数融資条件の横並び比較")).toBeInTheDocument();
  });

  it("初期状態で2つのシナリオが表示される", () => {
    render(<LoanComparePanel baseInput={makeInput()} />);
    expect(screen.getByDisplayValue("シナリオ 1")).toBeInTheDocument();
    expect(screen.getByDisplayValue("シナリオ 2")).toBeInTheDocument();
  });

  it("「シナリオ追加」ボタンをクリックするとシナリオが増える", async () => {
    render(<LoanComparePanel baseInput={makeInput()} />);
    await userEvent.click(screen.getByRole("button", { name: /シナリオ追加/ }));
    expect(screen.getByDisplayValue("シナリオ 3")).toBeInTheDocument();
  });

  it("シナリオが4件になると「シナリオ追加」ボタンが無効化される", async () => {
    render(<LoanComparePanel baseInput={makeInput()} />);
    await userEvent.click(screen.getByRole("button", { name: /シナリオ追加/ }));
    await userEvent.click(screen.getByRole("button", { name: /シナリオ追加/ }));
    expect(screen.getByRole("button", { name: /シナリオ追加/ })).toBeDisabled();
  });

  it("削除ボタンをクリックするとシナリオが減る", async () => {
    render(<LoanComparePanel baseInput={makeInput()} />);
    // aria-label="削除" ボタンが2つある（各シナリオに1つ）
    const deleteButtons = screen.getAllByRole("button", { name: "削除" });
    await userEvent.click(deleteButtons[0]);
    expect(screen.queryByDisplayValue("シナリオ 1")).not.toBeInTheDocument();
  });

  it("「比較実行」ボタンをクリックするとAPIが呼ばれる", async () => {
    vi.mocked(api.analyze).mockResolvedValue(makeResult());
    render(<LoanComparePanel baseInput={makeInput()} />);
    await userEvent.click(screen.getByRole("button", { name: /比較実行/ }));
    await waitFor(() => {
      expect(api.analyze).toHaveBeenCalledTimes(2); // 2 scenarios
    });
  });

  it("比較結果が返ると表が表示される", async () => {
    vi.mocked(api.analyze).mockResolvedValue(makeResult());
    render(<LoanComparePanel baseInput={makeInput()} />);
    await userEvent.click(screen.getByRole("button", { name: /比較実行/ }));
    await waitFor(() => {
      expect(screen.getByText("月返済額")).toBeInTheDocument();
    });
  });

  it("APIエラー時にエラーメッセージが表示される", async () => {
    vi.mocked(api.analyze).mockRejectedValue(new Error("比較エラー"));
    render(<LoanComparePanel baseInput={makeInput()} />);
    await userEvent.click(screen.getByRole("button", { name: /比較実行/ }));
    await waitFor(() => {
      expect(screen.getByText("比較エラー")).toBeInTheDocument();
    });
  });

  it("APIがError以外を投げた場合はフォールバックメッセージが表示される", async () => {
    vi.mocked(api.analyze).mockRejectedValue("unknown");
    render(<LoanComparePanel baseInput={makeInput()} />);
    await userEvent.click(screen.getByRole("button", { name: /比較実行/ }));
    await waitFor(() => {
      expect(screen.getByText("比較に失敗しました")).toBeInTheDocument();
    });
  });

  it("シナリオ名を変更できる", async () => {
    render(<LoanComparePanel baseInput={makeInput()} />);
    const nameInput = screen.getByDisplayValue("シナリオ 1");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "変動金利");
    expect(screen.getByDisplayValue("変動金利")).toBeInTheDocument();
  });
});

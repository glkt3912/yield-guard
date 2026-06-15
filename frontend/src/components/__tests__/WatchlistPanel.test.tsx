import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import WatchlistPanel from "@/components/WatchlistPanel";
import { makeResult } from "./helpers";

// WatchlistCompareTable is a child component; stub it to simplify
vi.mock("@/components/WatchlistCompareTable", () => ({ default: () => null }));
vi.mock("@/components/RootLayoutClient", () => ({
  useAuthContext: () => ({ user: { uid: "test-uid" }, loading: false }),
}));
vi.mock("firebase/firestore", () => ({
  collection: vi.fn(() => null),
  doc: vi.fn(() => null),
  addDoc: vi.fn(),
  updateDoc: vi.fn(),
  deleteDoc: vi.fn(),
  query: vi.fn(),
  orderBy: vi.fn(),
  onSnapshot: vi.fn(() => () => {}),
  serverTimestamp: vi.fn(() => null),
}));

vi.mock("@/components/ui/toast", () => ({
  useToast: () => ({ toast: vi.fn() }),
  ToastProvider: ({ children }: { children: React.ReactNode }) => children,
}));

const STORAGE_KEY = "yg_watchlist";

function setupStorage(items: object[] = []) {
  const store: Record<string, string> = {
    [STORAGE_KEY]: JSON.stringify(items),
  };
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      Object.keys(store).forEach((k) => delete store[k]);
    },
  });
  return store;
}

describe("WatchlistPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupStorage([]);
  });

  it("カードタイトルが表示される", () => {
    render(<WatchlistPanel />);
    expect(screen.getByText("物件候補ウォッチリスト")).toBeInTheDocument();
  });

  it("物件名フィールドとメモフィールドが表示される", () => {
    render(<WatchlistPanel />);
    expect(screen.getByLabelText("物件名")).toBeInTheDocument();
    expect(screen.getByLabelText("メモ")).toBeInTheDocument();
  });

  it("物件名なしで追加ボタンを押すとバリデーションエラーが表示される", async () => {
    render(<WatchlistPanel />);
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    expect(screen.getByText("物件名を入力してください")).toBeInTheDocument();
  });

  it("物件名を入力して追加するとリストに表示される", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "テスト物件A");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    expect(screen.getByText("テスト物件A")).toBeInTheDocument();
  });

  it("追加後に物件名フィールドがクリアされる", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "テスト物件B");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    expect(screen.getByLabelText("物件名")).toHaveValue("");
  });

  it("Enterキーでも物件が追加される", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "物件Enter{Enter}");
    expect(screen.getByText("物件Enter")).toBeInTheDocument();
  });

  it("currentResultがある場合は指標が追加される（追加した物件に表面利回りが保存される）", async () => {
    // 表面利回り表示は物件価格ベースの marketGrossYield を使う（#773）
    const result = makeResult({ marketGrossYield: 0.09 });
    render(<WatchlistPanel currentResult={result} />);
    await userEvent.type(screen.getByLabelText("物件名"), "指標付き物件");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    // marketGrossYield 9.0% が表示される（小数1桁）
    await waitFor(() => {
      expect(screen.getByText(/9\.0%/)).toBeInTheDocument();
    });
  });

  it("物件が2件あると比較表示ボタンが表示される", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "物件1");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    await userEvent.type(screen.getByLabelText("物件名"), "物件2");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    expect(screen.getByRole("button", { name: /比較表示/ })).toBeInTheDocument();
  });

  it("削除ボタンをクリックすると確認ダイアログが表示される", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "削除テスト物件");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));

    const deleteButtons = screen.getAllByRole("button", { name: /削除/ });
    // Find the trash button (not the "比較表示" confirm button)
    const trashBtn = deleteButtons.find((btn) => btn.querySelector("svg") !== null);
    if (trashBtn) await userEvent.click(trashBtn);

    // Confirmation modal should appear
    await waitFor(() => {
      expect(screen.getByText(/削除しますか/)).toBeInTheDocument();
    });
  });

  it("localStorage からアイテムが復元される", () => {
    setupStorage([
      {
        id: "abc-123",
        name: "復元物件",
        memo: "",
        status: "検討中",
        addedAt: new Date().toISOString(),
      },
    ]);
    render(<WatchlistPanel />);
    expect(screen.getByText("復元物件")).toBeInTheDocument();
  });

  it("ステータスセレクトで状態を変更できる", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "ステータス変更物件");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));

    const select = screen.getByRole("combobox");
    await userEvent.selectOptions(select, "見送り");
    expect((select as HTMLSelectElement).value).toBe("見送り");
  });

  it("物件が1件の場合、比較表示ボタンが表示されない", async () => {
    render(<WatchlistPanel />);
    await userEvent.type(screen.getByLabelText("物件名"), "物件X");
    await userEvent.click(screen.getByRole("button", { name: "追加" }));
    expect(screen.queryByRole("button", { name: /比較表示/ })).not.toBeInTheDocument();
  });
});

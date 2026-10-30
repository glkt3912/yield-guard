import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import WatchlistPanel from "@/components/WatchlistPanel";

// Firebase 未設定（ログインユーザーなし）時は localStorage だけで動くこと
vi.mock("@/components/WatchlistCompareTable", () => ({ default: () => null }));
vi.mock("@/components/RootLayoutClient", () => ({
  useAuthContext: () => ({ user: null, loading: false }),
}));
vi.mock("@/lib/firebase", () => ({ getFirebaseDb: () => null }));
vi.mock("@/components/ui/toast", () => ({
  useToast: () => ({ toast: vi.fn() }),
  ToastProvider: ({ children }: { children: React.ReactNode }) => children,
}));

const STORAGE_KEY = "yg_watchlist";

let store: Record<string, string>;

function stored() {
  return JSON.parse(store[STORAGE_KEY] ?? "[]") as { name: string; status: string }[];
}

describe("WatchlistPanel（Firebase 未設定）", () => {
  beforeEach(() => {
    store = {
      [STORAGE_KEY]: JSON.stringify([
        {
          id: "local_1",
          name: "物件A",
          memo: "",
          status: "検討中",
          addedAt: "2026-10-01T00:00:00Z",
        },
      ]),
    };
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => {
        store[key] = value;
      },
      removeItem: (key: string) => {
        delete store[key];
      },
    });
  });

  it("削除すると一覧と localStorage から消える", async () => {
    render(<WatchlistPanel />);
    await userEvent.click(screen.getByRole("button", { name: "物件A を削除" }));
    const dialog = screen.getByRole("alertdialog");
    await userEvent.click(within(dialog).getByRole("button", { name: "削除" }));

    expect(screen.queryByText("物件A")).not.toBeInTheDocument();
    expect(stored()).toEqual([]);
  });

  it("ステータス変更が localStorage に保存される", async () => {
    render(<WatchlistPanel />);
    await userEvent.selectOptions(screen.getByLabelText("物件A のステータスを変更"), "見送り");

    expect(stored()[0].status).toBe("見送り");
  });
});

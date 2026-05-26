import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { act } from "react";
import { AreaDiscovery } from "@/components/AreaDiscovery";
import type { AreaDiscoveryItem } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  fetchAreaDiscovery: vi.fn(),
}));

import * as api from "@/lib/api";

function makeItem(overrides: Partial<AreaDiscoveryItem> = {}): AreaDiscoveryItem {
  return {
    municipalityCode: "13101",
    municipalityName: "千代田区",
    medianTsubo: 5_000_000,
    transactionCount: 42,
    yieldDifficulty: "difficult",
    yieldDifficultyLabel: "難しい",
    landPriceTrend: "上昇",
    dataSufficient: true,
    centerLat: 35.694,
    centerLng: 139.753,
    ...overrides,
  };
}

describe("AreaDiscovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("初期状態でフォームが表示される", () => {
    render(<AreaDiscovery />);
    expect(screen.getByRole("heading", { name: "エリアを探す" })).toBeInTheDocument();
    expect(screen.getByText(/予算と目標利回りを入力/)).toBeInTheDocument();
  });

  it("初期状態で東京都がデフォルト選択されている", () => {
    render(<AreaDiscovery />);
    const select = screen.getByRole("combobox");
    expect((select as HTMLSelectElement).value).toBe("13");
  });

  it("初期状態で目標利回りが8%にセットされている", () => {
    render(<AreaDiscovery />);
    const yieldInput = screen.getByPlaceholderText("例: 8");
    expect((yieldInput as HTMLInputElement).value).toBe("8");
  });

  it("検索ボタンをクリックすると fetchAreaDiscovery が呼ばれる", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({ items: [], prefecture: "13" });
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));
    await waitFor(() => {
      expect(api.fetchAreaDiscovery).toHaveBeenCalledTimes(1);
    });
  });

  it("都道府県・予算・利回りを指定して検索するとパラメータが正しく渡される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({ items: [], prefecture: "27" });
    render(<AreaDiscovery />);

    // 大阪府を選択
    await userEvent.selectOptions(screen.getByRole("combobox"), "27");
    // 予算入力
    await userEvent.type(screen.getByPlaceholderText("例: 3000"), "3000");
    // 利回り変更
    await userEvent.clear(screen.getByPlaceholderText("例: 8"));
    await userEvent.type(screen.getByPlaceholderText("例: 8"), "10");

    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(api.fetchAreaDiscovery).toHaveBeenCalledWith({
        prefecture: "27",
        budget: 30_000_000,
        yield: 0.1,
      });
    });
  });

  it("予算が空の場合はbudgetパラメータが渡されない", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({ items: [], prefecture: "13" });
    render(<AreaDiscovery />);

    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(api.fetchAreaDiscovery).toHaveBeenCalledWith(
        expect.not.objectContaining({ budget: expect.anything() })
      );
    });
  });

  it("検索中はボタンが「検索中...」に変わり無効化される", async () => {
    let resolve: (v: { items: AreaDiscoveryItem[]; prefecture: string }) => void;
    const pending = new Promise<{ items: AreaDiscoveryItem[]; prefecture: string }>((r) => {
      resolve = r;
    });
    vi.mocked(api.fetchAreaDiscovery).mockReturnValue(pending);

    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    expect(screen.getByRole("button", { name: "検索中..." })).toBeDisabled();

    await act(async () => {
      resolve!({ items: [], prefecture: "13" });
    });
  });

  it("検索結果が0件のとき「データが見つかりませんでした」が表示される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({ items: [], prefecture: "13" });
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText(/該当するエリアのデータが見つかりませんでした/)).toBeInTheDocument();
    });
  });

  it("検索結果が返るとテーブルが表示される", async () => {
    const items = [
      makeItem({ municipalityName: "千代田区", transactionCount: 42 }),
      makeItem({
        municipalityCode: "13102",
        municipalityName: "中央区",
        transactionCount: 30,
        yieldDifficulty: "achievable",
        yieldDifficultyLabel: "達成可能",
      }),
    ];
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({ items, prefecture: "13" });

    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText("千代田区")).toBeInTheDocument();
      expect(screen.getByText("中央区")).toBeInTheDocument();
    });
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("30")).toBeInTheDocument();
    expect(screen.getByText("難しい")).toBeInTheDocument();
    expect(screen.getByText("達成可能")).toBeInTheDocument();
  });

  it("坪単価が0の場合はダッシュ（—）が表示される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({
      items: [makeItem({ medianTsubo: 0 })],
      prefecture: "13",
    });
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText("—")).toBeInTheDocument();
    });
  });

  it("dataSufficient=false のとき「（データ少）」が表示される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({
      items: [makeItem({ dataSufficient: false })],
      prefecture: "13",
    });
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText("（データ少）")).toBeInTheDocument();
    });
  });

  it("APIエラー時にエラーメッセージが表示される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockRejectedValue(new Error("サーバーエラー"));
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText("サーバーエラー")).toBeInTheDocument();
    });
  });

  it("APIがError以外を投げた場合はフォールバックエラーが表示される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockRejectedValue("unknown");
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText("エリアデータの取得に失敗しました")).toBeInTheDocument();
    });
  });

  it("市区町村行をクリックすると onMunicipalitySelect が呼ばれる", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({
      items: [makeItem()],
      prefecture: "13",
    });
    const onMunicipalitySelect = vi.fn();
    render(<AreaDiscovery onMunicipalitySelect={onMunicipalitySelect} />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText("千代田区")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText("千代田区"));
    expect(onMunicipalitySelect).toHaveBeenCalledWith("13101", "千代田区", "13", 35.694, 139.753);
  });

  it("結果が表示されると地図に関するヒントが表示される", async () => {
    vi.mocked(api.fetchAreaDiscovery).mockResolvedValue({
      items: [makeItem()],
      prefecture: "13",
    });
    render(<AreaDiscovery />);
    await userEvent.click(screen.getByRole("button", { name: "エリアを探す" }));

    await waitFor(() => {
      expect(screen.getByText(/市区町村を選択すると地図が表示されます/)).toBeInTheDocument();
    });
  });
});

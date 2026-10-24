import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import dynamic from "@/lib/dynamic";

describe("dynamic", () => {
  const reload = vi.fn();
  beforeEach(() => {
    sessionStorage.clear();
    reload.mockClear();
    vi.stubGlobal("location", { ...window.location, reload });
    vi.spyOn(console, "error").mockImplementation(() => {});
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("読み込み成功時はコンポーネントを描画する", async () => {
    const Comp = dynamic(async () => ({ default: () => <p>loaded</p> }));
    render(<Comp />);
    expect(await screen.findByText("loaded")).toBeInTheDocument();
  });

  it("チャンク読み込み失敗時は再読み込みする", async () => {
    const Comp = dynamic(() => Promise.reject(new Error("chunk")));
    render(<Comp />);
    await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1));
  });

  it("直前に再読み込み済みなら再読み込みせずエラー表示にする（無限リロード防止）", async () => {
    sessionStorage.setItem("chunk-reload-at", String(Date.now()));
    const A = dynamic(() => Promise.reject(new Error("a")));
    const B = dynamic(() => Promise.reject(new Error("b")));
    render(
      <>
        <A />
        <B />
      </>
    );
    expect(await screen.findAllByText(/読み込めませんでした/)).toHaveLength(2);
    expect(reload).not.toHaveBeenCalled();
  });
});

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useResponsiveChart, useChartHeight } from "@/lib/useChartHeight";

describe("useResponsiveChart", () => {
  const originalInnerWidth = window.innerWidth;

  beforeEach(() => {
    vi.spyOn(window, "addEventListener");
    vi.spyOn(window, "removeEventListener");
  });

  afterEach(() => {
    Object.defineProperty(window, "innerWidth", {
      writable: true,
      value: originalInnerWidth,
    });
    vi.restoreAllMocks();
  });

  function setWidth(width: number) {
    Object.defineProperty(window, "innerWidth", { writable: true, value: width });
    window.dispatchEvent(new Event("resize"));
  }

  it("初期値はデスクトップの高さを返す", () => {
    Object.defineProperty(window, "innerWidth", { writable: true, value: 1200 });
    const { result } = renderHook(() => useResponsiveChart(160, 180, 200));
    // After mount effect, desktop width → height=200
    act(() => setWidth(1200));
    expect(result.current.height).toBe(200);
    expect(result.current.isMobile).toBe(false);
  });

  it("モバイル幅(< 640)のとき mobile の高さを返す", () => {
    const { result } = renderHook(() => useResponsiveChart(160, 180, 200));
    act(() => setWidth(375));
    expect(result.current.height).toBe(160);
    expect(result.current.isMobile).toBe(true);
  });

  it("タブレット幅(640〜1023)のとき tablet の高さを返す", () => {
    const { result } = renderHook(() => useResponsiveChart(160, 180, 200));
    act(() => setWidth(768));
    expect(result.current.height).toBe(180);
    expect(result.current.isMobile).toBe(false);
  });

  it("デスクトップ幅(>= 1024)のとき desktop の高さを返す", () => {
    const { result } = renderHook(() => useResponsiveChart(160, 180, 200));
    act(() => setWidth(1440));
    expect(result.current.height).toBe(200);
    expect(result.current.isMobile).toBe(false);
  });

  it("resize イベントリスナーが登録される", () => {
    renderHook(() => useResponsiveChart(160, 180, 200));
    expect(window.addEventListener).toHaveBeenCalledWith("resize", expect.any(Function));
  });

  it("アンマウント時に resize イベントリスナーが解除される", () => {
    const { unmount } = renderHook(() => useResponsiveChart(160, 180, 200));
    unmount();
    expect(window.removeEventListener).toHaveBeenCalledWith("resize", expect.any(Function));
  });

  it("ウィンドウサイズ変更でリアクティブに更新される", () => {
    const { result } = renderHook(() => useResponsiveChart(160, 180, 200));
    act(() => setWidth(375));
    expect(result.current.height).toBe(160);
    act(() => setWidth(1440));
    expect(result.current.height).toBe(200);
  });
});

describe("useChartHeight", () => {
  afterEach(() => {
    Object.defineProperty(window, "innerWidth", {
      writable: true,
      value: 1024,
    });
  });

  it("モバイル幅のとき mobile の高さ（数値）を返す", () => {
    const { result } = renderHook(() => useChartHeight(160, 180, 200));
    act(() => {
      Object.defineProperty(window, "innerWidth", { writable: true, value: 375 });
      window.dispatchEvent(new Event("resize"));
    });
    expect(result.current).toBe(160);
  });

  it("デスクトップ幅のとき desktop の高さ（数値）を返す", () => {
    const { result } = renderHook(() => useChartHeight(160, 180, 200));
    act(() => {
      Object.defineProperty(window, "innerWidth", { writable: true, value: 1440 });
      window.dispatchEvent(new Event("resize"));
    });
    expect(result.current).toBe(200);
  });
});

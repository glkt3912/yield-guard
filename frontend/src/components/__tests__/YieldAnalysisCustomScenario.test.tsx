import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { YieldAnalysis, getDscrColorClass } from "@/components/YieldAnalysis";
import { makeInput, makeResult } from "./helpers";
import type { StressScenarioResult } from "@/types/investment";

vi.mock("@/lib/api", () => ({
  analyze: vi.fn(),
}));

import * as api from "@/lib/api";

function makeStressScenario(overrides: Partial<StressScenarioResult> = {}): StressScenarioResult {
  return {
    label: "カスタム",
    dscr: 1.1,
    breakEvenYear: 5,
    interestRateDelta: 0.01,
    vacancyRateDelta: 0.1,
    totalCashFlow: 500_000,
    isSafe: true,
    ...overrides,
  };
}

describe("getDscrColorClass", () => {
  it("dscr=1.3 のとき text-green-600 を返す", () => {
    expect(getDscrColorClass(1.3)).toBe("text-green-600");
  });

  it("dscr=1.2 のとき text-green-600 を返す（境界値）", () => {
    expect(getDscrColorClass(1.2)).toBe("text-green-600");
  });

  it("dscr=1.1 のとき text-yellow-600 を返す", () => {
    expect(getDscrColorClass(1.1)).toBe("text-yellow-600");
  });

  it("dscr=1.0 のとき text-yellow-600 を返す（境界値）", () => {
    expect(getDscrColorClass(1.0)).toBe("text-yellow-600");
  });

  it("dscr=0.9 のとき text-red-600 を返す", () => {
    expect(getDscrColorClass(0.9)).toBe("text-red-600");
  });
});

describe("YieldAnalysis — カスタムストレステスト", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("スライダーが両方 0 のときカスタム行は表示されない", () => {
    render(<YieldAnalysis result={makeResult()} input={makeInput()} />);
    // Both sliders start at 0, so no custom scenario row should appear.
    // The "カスタム（計算中）" loading row and custom result row should both be absent.
    expect(screen.queryByText(/計算中/)).not.toBeInTheDocument();
    expect(screen.queryByText(/カスタム（計算中）/)).not.toBeInTheDocument();
    expect(vi.mocked(api.analyze)).not.toHaveBeenCalled();
  });

  it("金利スライダーを変更すると debounce 後に analyze が正しい loanRateDelta で呼ばれる", async () => {
    vi.mocked(api.analyze).mockResolvedValue(
      makeResult({
        stressScenarios: [makeStressScenario({ label: "カスタム", dscr: 1.15 })],
      })
    );

    render(<YieldAnalysis result={makeResult()} input={makeInput()} />);

    // The loan rate slider (金利オフセット) — value range 0-3
    const sliders = screen.getAllByRole("slider");
    const loanSlider = sliders[0];

    act(() => {
      fireEvent.change(loanSlider, { target: { value: "1" } });
    });

    // analyze should NOT have been called yet (debounce pending)
    expect(vi.mocked(api.analyze)).not.toHaveBeenCalled();

    // Advance timers past the 400ms debounce, flush async microtasks
    await act(async () => {
      await vi.advanceTimersByTimeAsync(400);
    });

    expect(vi.mocked(api.analyze)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(api.analyze)).toHaveBeenCalledWith(
      expect.objectContaining({
        loanRateDelta: 0.01,
        vacancyRateDelta: 0,
      })
    );
  });

  it("空室率スライダーを変更すると debounce 後に analyze が正しい vacancyRateDelta で呼ばれる", async () => {
    vi.mocked(api.analyze).mockResolvedValue(
      makeResult({
        stressScenarios: [makeStressScenario({ label: "カスタム", dscr: 0.95 })],
      })
    );

    render(<YieldAnalysis result={makeResult()} input={makeInput()} />);

    const sliders = screen.getAllByRole("slider");
    const vacancySlider = sliders[1];

    act(() => {
      fireEvent.change(vacancySlider, { target: { value: "10" } });
    });

    expect(vi.mocked(api.analyze)).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(400);
    });

    expect(vi.mocked(api.analyze)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(api.analyze)).toHaveBeenCalledWith(
      expect.objectContaining({
        loanRateDelta: 0,
        vacancyRateDelta: 0.1,
      })
    );
  });

  it("analyze が「カスタム」シナリオを返すとカスタム行がテーブルに表示される", async () => {
    vi.mocked(api.analyze).mockResolvedValue(
      makeResult({
        stressScenarios: [makeStressScenario({ label: "カスタム", dscr: 1.25 })],
      })
    );

    render(<YieldAnalysis result={makeResult()} input={makeInput()} />);

    const sliders = screen.getAllByRole("slider");
    act(() => {
      fireEvent.change(sliders[0], { target: { value: "1" } });
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(400);
    });

    // Custom scenario row appears — the DSCR value should be visible
    expect(screen.getAllByText("1.25").length).toBeGreaterThan(0);
  });

  it("フェッチ中はローディングスピナーが表示される", async () => {
    let resolveAnalyze!: (v: ReturnType<typeof makeResult>) => void;
    const pending = new Promise<ReturnType<typeof makeResult>>((r) => {
      resolveAnalyze = r;
    });
    vi.mocked(api.analyze).mockReturnValue(pending);

    render(<YieldAnalysis result={makeResult()} input={makeInput()} />);

    const sliders = screen.getAllByRole("slider");
    act(() => {
      fireEvent.change(sliders[0], { target: { value: "2" } });
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(400);
    });

    // Loading spinner text should appear (e.g. "カスタム（計算中）")
    expect(screen.getAllByText(/計算中/).length).toBeGreaterThan(0);

    // Resolve to clean up
    await act(async () => {
      resolveAnalyze(makeResult());
    });
  });
});

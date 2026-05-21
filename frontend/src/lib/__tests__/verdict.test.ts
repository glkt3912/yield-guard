import { describe, it, expect } from "vitest";
import { calcVerdict } from "../pdf/verdict";
import { makeInput, makeResult } from "@/components/__tests__/helpers";

describe("calcVerdict", () => {
  it("returns PASS when all conditions are met", () => {
    const input = makeInput();
    const result = makeResult({
      grossYield: 0.09, // >= yieldTarget 0.08
      yieldTarget: 0.08,
      exitTotalEquity: 1_000_000,
      criticalErrors: [],
    });
    const v = calcVerdict(input, result, 1.3, 1.1);
    expect(v.level).toBe("PASS");
    expect(v.label).toBe("投資適格");
    expect(v.color).toBe("#16a34a");
    expect(v.reasons).toHaveLength(3);
  });

  it("returns CAUTION when DSCR is 1.0–1.19", () => {
    const input = makeInput();
    const result = makeResult({
      grossYield: 0.06, // below yieldTarget → not PASS
      yieldTarget: 0.08,
      exitTotalEquity: 500_000,
      criticalErrors: [],
    });
    const v = calcVerdict(input, result, 1.1, 0.9);
    expect(v.level).toBe("CAUTION");
    expect(v.label).toBe("要交渉");
    expect(v.color).toBe("#d97706");
  });

  it("returns REJECT when DSCR < 1.0", () => {
    const input = makeInput();
    const result = makeResult({ criticalErrors: [] });
    const v = calcVerdict(input, result, 0.8, 0.6);
    expect(v.level).toBe("REJECT");
    expect(v.label).toBe("見送り推奨");
    expect(v.color).toBe("#dc2626");
  });

  it("returns REJECT when criticalErrors has a REJECT status", () => {
    const input = makeInput();
    const result = makeResult({
      grossYield: 0.09,
      exitTotalEquity: 1_000_000,
      criticalErrors: [
        { code: "DEADCROSS_EARLY", status: "REJECT", message: "デッドクロス早期発生" },
      ],
    });
    const v = calcVerdict(input, result, 1.5, 1.2);
    expect(v.level).toBe("REJECT");
  });

  it("returns CAUTION when exitTotalEquity is negative but DSCR >= 1.0 with no REJECT errors", () => {
    const input = makeInput();
    const result = makeResult({
      grossYield: 0.09,
      exitTotalEquity: -500_000,
      criticalErrors: [],
    });
    // PASS requires exitTotalEquity >= 0; CAUTION only needs DSCR >= 1.0 and no REJECT errors
    const v = calcVerdict(input, result, 1.5, 1.2);
    expect(v.level).toBe("CAUTION");
  });

  it("returns CAUTION (not REJECT) for WARNING-only criticalErrors with DSCR >= 1.0", () => {
    const input = makeInput();
    const result = makeResult({
      grossYield: 0.07,
      yieldTarget: 0.08,
      exitTotalEquity: 500_000,
      criticalErrors: [
        { code: "LAND_VALUE_GUARD", status: "WARNING" as never, message: "警告のみ" },
      ],
    });
    const v = calcVerdict(input, result, 1.05, 0.95);
    expect(v.level).toBe("CAUTION");
  });

  it("includes deadCrossYear info in autoComment when present", () => {
    const input = makeInput();
    const result = makeResult({ deadCrossYear: 8, criticalErrors: [] });
    const v = calcVerdict(input, result, 0.9, 0.7);
    expect(v.autoComment).toContain("8年目");
  });

  it("mentions stress DSCR drop in autoComment", () => {
    const input = makeInput();
    const result = makeResult({ criticalErrors: [] });
    const v = calcVerdict(input, result, 1.4, 0.9);
    expect(v.autoComment).toContain("0.90");
  });
});

describe("calcVerdict – DSCR 境界値", () => {
  it("DSCR 1.19 → CAUTION（1.20 未満）", () => {
    const v = calcVerdict(makeInput(), makeResult({ criticalErrors: [] }), 1.19, 1.0);
    expect(v.level).toBe("CAUTION");
  });

  it("DSCR 1.20 → PASS（閾値ちょうど、他条件 OK）", () => {
    const v = calcVerdict(makeInput(), makeResult({ criticalErrors: [] }), 1.2, 1.0);
    expect(v.level).toBe("PASS");
  });

  it("DSCR 0 → REJECT（1.00 未満）", () => {
    const v = calcVerdict(makeInput(), makeResult({ criticalErrors: [] }), 0, 0);
    expect(v.level).toBe("REJECT");
  });

  it("DSCR 1.20 でも criticalErrors に REJECT があれば REJECT", () => {
    const result = makeResult({
      criticalErrors: [{ code: "NEGATIVE_CF", status: "REJECT", message: "キャッシュフロー赤字" }],
    });
    const v = calcVerdict(makeInput(), result, 1.2, 1.0);
    expect(v.level).toBe("REJECT");
  });
});

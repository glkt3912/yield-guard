import { describe, it, expect } from "vitest";
import { fmtYen, fmtPct, fmtPct1, sanitize } from "../pdf/format";

describe("fmtYen", () => {
  it("formats zero as 0円", () => {
    expect(fmtYen(0)).toBe("0円");
  });

  it("rounds fractional yen", () => {
    expect(fmtYen(587_874.124)).toBe("587,874円");
  });

  it("formats values under 1,000,000 in yen", () => {
    expect(fmtYen(9_999)).toBe("9,999円");
  });

  it("formats 999,999 in yen", () => {
    expect(fmtYen(999_999)).toBe("999,999円");
  });

  it("formats 150,000 in yen", () => {
    expect(fmtYen(150_000)).toBe("150,000円");
  });

  it("formats 1,000,000 as 1.0百万円", () => {
    expect(fmtYen(1_000_000)).toBe("1.0百万円");
  });

  it("formats values in 百万円", () => {
    expect(fmtYen(1_500_000)).toBe("1.5百万円");
  });

  it("formats 99,999,999 in 百万円", () => {
    expect(fmtYen(99_999_999)).toBe("100.0百万円");
  });

  it("formats 100,000,000 as 1.0億円", () => {
    expect(fmtYen(100_000_000)).toBe("1.0億円");
  });

  it("formats negative values", () => {
    expect(fmtYen(-2_000_000)).toBe("-2.0百万円");
  });

  it("formats negative under 10,000", () => {
    expect(fmtYen(-5_000)).toBe("-5,000円");
  });
});

describe("fmtPct", () => {
  it("formats 0 as 0.00%", () => {
    expect(fmtPct(0)).toBe("0.00%");
  });

  it("formats 0.08 as 8.00%", () => {
    expect(fmtPct(0.08)).toBe("8.00%");
  });

  it("formats 0.0825 as 8.25%", () => {
    expect(fmtPct(0.0825)).toBe("8.25%");
  });

  it("formats 0.015 as 1.50%", () => {
    expect(fmtPct(0.015)).toBe("1.50%");
  });
});

describe("fmtPct1", () => {
  it("formats 0.083 as 8.3%", () => {
    expect(fmtPct1(0.083)).toBe("8.3%");
  });
});

describe("sanitize", () => {
  it("removes < and >", () => {
    expect(sanitize("<script>")).toBe("script");
  });

  it("removes &", () => {
    expect(sanitize("A&B")).toBe("AB");
  });

  it("returns empty string for undefined", () => {
    expect(sanitize(undefined)).toBe("");
  });

  it("converts number to string", () => {
    expect(sanitize(123)).toBe("123");
  });

  it("passes through normal Japanese text", () => {
    expect(sanitize("木造")).toBe("木造");
  });
});

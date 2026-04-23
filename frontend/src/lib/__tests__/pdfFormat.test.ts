import { describe, it, expect } from "vitest";
import { fmtYen, fmtPct, sanitize } from "../pdf/format";

describe("fmtYen", () => {
  it("formats zero as 0円", () => {
    expect(fmtYen(0)).toBe("0円");
  });

  it("rounds fractional yen and converts to 万円", () => {
    expect(fmtYen(587_874.124)).toBe("59万円");
  });

  it("formats values under 10,000 in yen", () => {
    expect(fmtYen(9_999)).toBe("9,999円");
  });

  it("formats 10,000 as 1万円 (boundary)", () => {
    expect(fmtYen(10_000)).toBe("1万円");
  });

  it("formats 150,000 as 15万円", () => {
    expect(fmtYen(150_000)).toBe("15万円");
  });

  it("formats 999,999 as 100万円 (rounds to nearest 万)", () => {
    expect(fmtYen(999_999)).toBe("100万円");
  });

  it("formats 1,000,000 as 100万円", () => {
    expect(fmtYen(1_000_000)).toBe("100万円");
  });

  it("formats 1,500,000 as 150万円", () => {
    expect(fmtYen(1_500_000)).toBe("150万円");
  });

  it("formats 99,999,999 as 1.0億円 (rounds up to 億円)", () => {
    expect(fmtYen(99_999_999)).toBe("1.0億円");
  });

  it("formats 95,000,000 as 9500万円 (stays in 万円)", () => {
    expect(fmtYen(95_000_000)).toBe("9500万円");
  });

  it("formats 100,000,000 as 1.0億円", () => {
    expect(fmtYen(100_000_000)).toBe("1.0億円");
  });

  it("formats negative 万円 values", () => {
    expect(fmtYen(-2_000_000)).toBe("-200万円");
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

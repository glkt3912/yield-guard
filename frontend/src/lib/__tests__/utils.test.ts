import { describe, expect, it } from "vitest";
import { toMan, toManFloat, fromMan } from "../utils";

describe("toMan", () => {
  it("returns empty string for 0", () => {
    expect(toMan(0)).toBe("");
  });

  it("converts 10,000 yen to '1'", () => {
    expect(toMan(10_000)).toBe("1");
  });

  it("converts 50,000,000 yen to '5000'", () => {
    expect(toMan(50_000_000)).toBe("5000");
  });

  it("rounds 15,000 yen to '2' (Math.round: 1.5 → 2)", () => {
    expect(toMan(15_000)).toBe("2");
  });

  it("truncates 14,999 yen to '1' (Math.round: 1.4999 → 1)", () => {
    expect(toMan(14_999)).toBe("1");
  });
});

describe("toManFloat", () => {
  it("returns empty string for 0", () => {
    expect(toManFloat(0)).toBe("");
  });

  it("returns integer string when no decimal part", () => {
    expect(toManFloat(100_000)).toBe("10");
  });

  it("returns one decimal place for fractional value", () => {
    expect(toManFloat(85_000)).toBe("8.5");
  });

  // Known precision loss: toFixed(1) rounds 8.55 → "8.6", so fromMan gives 86,000 (500 yen drift).
  // Monthly rent is typically in 1,000-yen increments so practical impact is negligible.
  it("rounds to one decimal place (known precision loss for 85,500)", () => {
    expect(toManFloat(85_500)).toBe("8.6");
  });
});

describe("fromMan", () => {
  it("returns 0 for empty string", () => {
    expect(fromMan("")).toBe(0);
  });

  it("returns 0 for non-numeric string", () => {
    expect(fromMan("abc")).toBe(0);
  });

  it("converts '1' to 10,000 yen", () => {
    expect(fromMan("1")).toBe(10_000);
  });

  it("converts '5000' to 50,000,000 yen", () => {
    expect(fromMan("5000")).toBe(50_000_000);
  });
});

describe("fromMan + toManFloat round-trip", () => {
  it("round-trips 100,000 yen exactly", () => {
    expect(fromMan(toManFloat(100_000))).toBe(100_000);
  });

  it("round-trips 85,000 yen exactly", () => {
    expect(fromMan(toManFloat(85_000))).toBe(85_000);
  });

  // Known precision loss: 85,500 → "8.6" → 86,000 (500 yen drift due to toFixed(1))
  it("round-trip of 85,500 results in 86,000 due to known toFixed(1) precision loss", () => {
    expect(fromMan(toManFloat(85_500))).toBe(86_000);
  });
});

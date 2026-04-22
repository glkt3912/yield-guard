import { describe, it, expect } from "vitest";
import { encodeUrlParams, decodeUrlParams } from "@/lib/urlParams";
import { DEFAULT_INPUT } from "@/types/investment";

describe("encodeUrlParams", () => {
  it("mode=quick is always included", () => {
    const params = encodeUrlParams("quick", DEFAULT_INPUT, "");
    expect(params.get("mode")).toBe("quick");
  });

  it("mode=full is included", () => {
    const params = encodeUrlParams("full", DEFAULT_INPUT);
    expect(params.get("mode")).toBe("full");
  });

  it("default values are omitted from URL (quick mode)", () => {
    const params = encodeUrlParams("quick", DEFAULT_INPUT, "");
    // All defaults should be omitted
    expect(params.get("totalPrice")).toBeNull();
    expect(params.get("rent")).toBeNull();
    expect(params.get("loanAmount")).toBeNull();
    expect(params.get("loanRate")).toBeNull();
    expect(params.get("loanYears")).toBeNull();
    expect(params.get("holdingYears")).toBeNull();
    expect(params.get("vacancy")).toBeNull();
    expect(params.get("expenseRate")).toBeNull();
  });

  it("totalPrice is included in quick mode when provided", () => {
    const params = encodeUrlParams("quick", DEFAULT_INPUT, "2000");
    expect(params.get("totalPrice")).toBe("2000");
  });

  it("non-default rent is included", () => {
    const input = { ...DEFAULT_INPUT, monthlyRent: 80_000 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("rent")).toBe("8");
  });

  it("non-default loanAmount is included in 万円", () => {
    const input = { ...DEFAULT_INPUT, loanAmount: 10_000_000 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("loanAmount")).toBe("1000");
  });

  it("non-default loanRate is included as percent", () => {
    const input = { ...DEFAULT_INPUT, annualLoanRate: 0.02 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("loanRate")).toBe("2");
  });

  it("non-default loanYears is included", () => {
    const input = { ...DEFAULT_INPUT, loanYears: 25 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("loanYears")).toBe("25");
  });

  it("non-default holdingYears is included", () => {
    const input = { ...DEFAULT_INPUT, holdingYears: 20 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("holdingYears")).toBe("20");
  });

  it("non-default vacancyRate is included as percent", () => {
    const input = { ...DEFAULT_INPUT, vacancyRate: 0.1 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("vacancy")).toBe("10");
  });

  it("non-default expenseRate is included as percent", () => {
    const input = { ...DEFAULT_INPUT, expenseRate: 0.25 };
    const params = encodeUrlParams("quick", input, "");
    expect(params.get("expenseRate")).toBe("25");
  });

  it("full mode includes landPrice and buildingCost when non-default", () => {
    const input = { ...DEFAULT_INPUT, landPrice: 20_000_000, buildingCost: 30_000_000 };
    const params = encodeUrlParams("full", input);
    expect(params.get("landPrice")).toBe("2000");
    expect(params.get("buildingCost")).toBe("3000");
  });

  it("full mode omits landPrice/buildingCost when default", () => {
    const params = encodeUrlParams("full", DEFAULT_INPUT);
    expect(params.get("landPrice")).toBeNull();
    expect(params.get("buildingCost")).toBeNull();
  });
});

describe("decodeUrlParams", () => {
  it("returns mode=quick when no mode param", () => {
    const params = new URLSearchParams("");
    const { mode } = decodeUrlParams(params);
    expect(mode).toBe("quick");
  });

  it("decodes mode=full", () => {
    const params = new URLSearchParams("mode=full");
    const { mode } = decodeUrlParams(params);
    expect(mode).toBe("full");
  });

  it("decodes totalPrice into quickTotalPriceMan", () => {
    const params = new URLSearchParams("mode=quick&totalPrice=2000");
    const { quickTotalPriceMan } = decodeUrlParams(params);
    expect(quickTotalPriceMan).toBe("2000");
  });

  it("decodes rent into monthlyRent (yen)", () => {
    const params = new URLSearchParams("rent=8");
    const { input } = decodeUrlParams(params);
    expect(input.monthlyRent).toBe(80_000);
  });

  it("decodes loanAmount into yen", () => {
    const params = new URLSearchParams("loanAmount=1000");
    const { input } = decodeUrlParams(params);
    expect(input.loanAmount).toBe(10_000_000);
  });

  it("decodes loanRate from percent to decimal", () => {
    const params = new URLSearchParams("loanRate=2");
    const { input } = decodeUrlParams(params);
    expect(input.annualLoanRate).toBeCloseTo(0.02);
  });

  it("decodes loanYears", () => {
    const params = new URLSearchParams("loanYears=25");
    const { input } = decodeUrlParams(params);
    expect(input.loanYears).toBe(25);
  });

  it("decodes holdingYears", () => {
    const params = new URLSearchParams("holdingYears=20");
    const { input } = decodeUrlParams(params);
    expect(input.holdingYears).toBe(20);
  });

  it("decodes vacancy from percent to decimal", () => {
    const params = new URLSearchParams("vacancy=10");
    const { input } = decodeUrlParams(params);
    expect(input.vacancyRate).toBeCloseTo(0.1);
  });

  it("decodes expenseRate from percent to decimal", () => {
    const params = new URLSearchParams("expenseRate=25");
    const { input } = decodeUrlParams(params);
    expect(input.expenseRate).toBeCloseTo(0.25);
  });

  it("decodes landPrice into yen (full mode)", () => {
    const params = new URLSearchParams("mode=full&landPrice=2000");
    const { input } = decodeUrlParams(params);
    expect(input.landPrice).toBe(20_000_000);
  });

  it("decodes buildingCost into yen (full mode)", () => {
    const params = new URLSearchParams("mode=full&buildingCost=3000");
    const { input } = decodeUrlParams(params);
    expect(input.buildingCost).toBe(30_000_000);
  });

  it("returns empty input for unknown/invalid params", () => {
    const params = new URLSearchParams("foo=bar&loanRate=notanumber");
    const { input } = decodeUrlParams(params);
    expect(input.annualLoanRate).toBeUndefined();
  });

  it("round-trips encode->decode for quick mode", () => {
    const original = {
      ...DEFAULT_INPUT,
      monthlyRent: 95_000,
      loanAmount: 12_000_000,
      annualLoanRate: 0.018,
      loanYears: 30,
      holdingYears: 15,
      vacancyRate: 0.08,
      expenseRate: 0.22,
    };
    const encoded = encodeUrlParams("quick", original, "1800");
    const { mode, input, quickTotalPriceMan } = decodeUrlParams(encoded);

    expect(mode).toBe("quick");
    expect(quickTotalPriceMan).toBe("1800");
    expect(input.monthlyRent).toBe(95_000);
    expect(input.loanAmount).toBeCloseTo(12_000_000, -1);
    expect(input.annualLoanRate).toBeCloseTo(0.018, 5);
    expect(input.loanYears).toBe(30);
    expect(input.holdingYears).toBe(15);
    expect(input.vacancyRate).toBeCloseTo(0.08, 3);
    expect(input.expenseRate).toBeCloseTo(0.22, 3);
  });

  it("round-trips encode->decode for full mode", () => {
    const original = {
      ...DEFAULT_INPUT,
      landPrice: 20_000_000,
      buildingCost: 30_000_000,
    };
    const encoded = encodeUrlParams("full", original);
    const { mode, input } = decodeUrlParams(encoded);

    expect(mode).toBe("full");
    expect(input.landPrice).toBe(20_000_000);
    expect(input.buildingCost).toBe(30_000_000);
  });
});

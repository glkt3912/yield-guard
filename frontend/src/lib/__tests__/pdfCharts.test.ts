import { describe, it, expect } from "vitest";
import { buildCfBarChartSvg, buildDeadCrossLineSvg, buildCostDonutSvg } from "../pdf/charts";
import { makeYearlyResult } from "@/components/__tests__/helpers";

const sampleYearly = Array.from({ length: 10 }, (_, i) => makeYearlyResult(i + 1));

describe("buildCfBarChartSvg", () => {
  it("returns a valid SVG string", () => {
    const svg = buildCfBarChartSvg(sampleYearly);
    expect(svg).toMatch(/^<svg /);
    expect(svg).toMatch(/<\/svg>$/);
  });

  it("contains rect elements for bars", () => {
    const svg = buildCfBarChartSvg(sampleYearly);
    expect(svg).toContain("<rect ");
  });

  it("returns empty SVG for empty input", () => {
    const svg = buildCfBarChartSvg([]);
    expect(svg).toMatch(/^<svg /);
    expect(svg).not.toContain("<rect ");
  });

  it("uses red fill for dead cross zone bars", () => {
    const dcYear = makeYearlyResult(3, { isInDeadCrossZone: true, afterTaxCashFlow: -100_000 });
    const yearly = [
      makeYearlyResult(1),
      makeYearlyResult(2),
      dcYear,
      ...Array.from({ length: 7 }, (_, i) => makeYearlyResult(i + 4)),
    ];
    const svg = buildCfBarChartSvg(yearly);
    expect(svg).toContain("#fca5a5");
  });
});

describe("buildDeadCrossLineSvg", () => {
  it("returns a valid SVG string", () => {
    const yearly = Array.from({ length: 35 }, (_, i) => makeYearlyResult(i + 1));
    const svg = buildDeadCrossLineSvg(yearly, 23);
    expect(svg).toMatch(/^<svg /);
    expect(svg).toMatch(/<\/svg>$/);
  });

  it("contains polyline elements for principal and depreciation", () => {
    const yearly = Array.from({ length: 35 }, (_, i) => makeYearlyResult(i + 1));
    const svg = buildDeadCrossLineSvg(yearly, -1);
    const matches = svg.match(/<polyline /g);
    expect(matches).toHaveLength(2);
  });

  it("includes dead cross year vertical marker when deadCrossYear > 0", () => {
    const yearly = Array.from({ length: 35 }, (_, i) => makeYearlyResult(i + 1));
    const svg = buildDeadCrossLineSvg(yearly, 10);
    expect(svg).toContain("10年目");
    expect(svg).toContain("stroke-dasharray");
  });

  it("omits dead cross marker when deadCrossYear is -1", () => {
    const yearly = Array.from({ length: 35 }, (_, i) => makeYearlyResult(i + 1));
    const svg = buildDeadCrossLineSvg(yearly, -1);
    expect(svg).not.toContain("年目");
  });
});

describe("buildCostDonutSvg", () => {
  const costs = {
    brokerageFee: 561_000,
    stampDuty: 20_000,
    registrationTax: 420_000,
    realEstateAcquisitionTax: 315_000,
    propertyTaxProration: 0,
    total: 1_316_000,
  };

  it("returns a valid SVG string", () => {
    const svg = buildCostDonutSvg(10_000_000, 5_000_000, costs);
    expect(svg).toMatch(/^<svg /);
    expect(svg).toMatch(/<\/svg>$/);
  });

  it("contains path elements for segments", () => {
    const svg = buildCostDonutSvg(10_000_000, 5_000_000, costs);
    expect(svg).toContain("<path ");
  });

  it("returns empty SVG when all values are zero", () => {
    const zeroCosts = { ...costs, total: 0 };
    const svg = buildCostDonutSvg(0, 0, zeroCosts);
    expect(svg).not.toContain("<path ");
  });

  it("excludes zero-value segments", () => {
    const svg = buildCostDonutSvg(10_000_000, 0, costs);
    expect(svg).not.toContain("建物");
  });
});

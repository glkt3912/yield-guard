import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { ReportPDF } from "@/components/ReportPDF";
import { makeInput, makeResult, makeYearlyResult } from "./helpers";

// @react-pdf/renderer renders to PDF (canvas/SVG), not DOM.
// Mock it so vitest (jsdom) can run the component tree without errors.
vi.mock("@react-pdf/renderer", () => {
  const React = require("react");

  const noop = ({ children }: { children?: React.ReactNode }) =>
    React.createElement(React.Fragment, null, children);

  return {
    Document: noop,
    Page: noop,
    View: noop,
    Text: ({ children }: { children?: React.ReactNode }) =>
      React.createElement("span", null, children),
    StyleSheet: {
      create: (styles: Record<string, unknown>) => styles,
    },
    Font: {
      register: vi.fn(),
    },
  };
});

describe("ReportPDF", () => {
  it("renders without throwing an exception", () => {
    const input = makeInput();
    const result = makeResult({
      stressScenarios: [
        {
          label: "ベースケース",
          interestRateDelta: 0,
          vacancyRateDelta: 0,
          totalCashFlow: 1_000_000,
          dscr: 1.2,
          breakEvenYear: 5,
          isSafe: true,
        },
        {
          label: "金利+1%",
          interestRateDelta: 0.01,
          vacancyRateDelta: 0,
          totalCashFlow: -500_000,
          dscr: 0.9,
          breakEvenYear: -1,
          isSafe: false,
        },
      ],
    });

    expect(() => render(<ReportPDF input={input} result={result} />)).not.toThrow();
  });

  it("renders with minimal result (empty stressScenarios)", () => {
    const input = makeInput();
    const result = makeResult({ stressScenarios: [] });

    expect(() => render(<ReportPDF input={input} result={result} />)).not.toThrow();
  });

  it("renders correctly with propertyTaxProration > 0", () => {
    const input = makeInput();
    const result = makeResult({
      acquisitionCosts: {
        brokerageFee: 561_000,
        stampDuty: 20_000,
        registrationTax: 420_000,
        realEstateAcquisitionTax: 315_000,
        propertyTaxProration: 50_000,
        total: 1_366_000,
      },
    });

    expect(() => render(<ReportPDF input={input} result={result} />)).not.toThrow();
  });

  it("renders correctly for newly built property (buildingAge = 0)", () => {
    const input = makeInput({ buildingAge: 0 });
    const result = makeResult();

    expect(() => render(<ReportPDF input={input} result={result} />)).not.toThrow();
  });

  it("renders correctly when deadCrossYear is set", () => {
    const input = makeInput();
    const yearlyResults = Array.from({ length: 10 }, (_, i) =>
      makeYearlyResult(i + 1, { isDeadCrossYear: i === 7, isInDeadCrossZone: i >= 7 })
    );
    const result = makeResult({ deadCrossYear: 8, yearlyResults });

    expect(() => render(<ReportPDF input={input} result={result} />)).not.toThrow();
  });
});

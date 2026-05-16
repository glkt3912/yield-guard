import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import NegotiationPanel from "@/components/NegotiationPanel";
import { makeInput, makeResult, makeComparison } from "./helpers";
import type { TheoreticalPriceResult } from "@/types/investment";

function makeTheoreticalPrice(
  overrides: Partial<TheoreticalPriceResult> = {}
): TheoreticalPriceResult {
  return {
    theoreticalPriceJPY: 12_000_000,
    deviationPct: 5,
    ageCorrection: 1.0,
    stationCorrection: 1.0,
    ridershipCorrection: 1.0,
    medianBuildingAge: 10,
    medianStationMinutes: 5,
    isLowDataWarning: false,
    hasStationData: true,
    hasRidershipData: false,
    ...overrides,
  };
}

describe("NegotiationPanel", () => {
  it("カードタイトルが表示される", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText("逆算・交渉シミュレーション")).toBeInTheDocument();
  });

  it("逆算モードセクションが表示される", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText("最大買付可能価格")).toBeInTheDocument();
    expect(screen.getByText("売出価格との差額")).toBeInTheDocument();
    expect(screen.getByText("売出価格での表面利回り")).toBeInTheDocument();
  });

  it("目標利回りが表示される", () => {
    const input = makeInput({ yieldTarget: 0.08 });
    render(
      <NegotiationPanel
        result={makeResult()}
        input={input}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText(/8.00%/)).toBeInTheDocument();
  });

  it("comparisonがnullのとき交渉シミュレーションセクションが非表示", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.queryByText("交渉シミュレーション")).not.toBeInTheDocument();
  });

  it("theoreticalPriceがnullのとき交渉シミュレーションセクションが非表示", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.queryByText("交渉シミュレーション")).not.toBeInTheDocument();
  });

  it("comparisonが存在すると交渉シミュレーションセクションが表示される", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={makeComparison()}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText("交渉シミュレーション")).toBeInTheDocument();
    expect(screen.getByText("市場取引実勢")).toBeInTheDocument();
  });

  it("theoreticalPriceが存在すると交渉シミュレーションに理論価格が表示される", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={makeTheoreticalPrice({ theoreticalPriceJPY: 11_000_000 })}
      />
    );
    expect(screen.getByText("交渉シミュレーション")).toBeInTheDocument();
    expect(screen.getByText("理論価格")).toBeInTheDocument();
  });

  it("isAboveYieldTarget=trueのとき表面利回りの値が表示される", () => {
    const result = makeResult({ grossYield: 0.09, isAboveYieldTarget: true });
    render(
      <NegotiationPanel
        result={result}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText("売出価格での表面利回り")).toBeInTheDocument();
    expect(screen.getByText("9.00%")).toBeInTheDocument();
  });

  it("isAboveYieldTarget=falseのとき表面利回りの値が表示される", () => {
    const result = makeResult({ grossYield: 0.05, isAboveYieldTarget: false });
    render(
      <NegotiationPanel
        result={result}
        input={makeInput()}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText("売出価格での表面利回り")).toBeInTheDocument();
    expect(screen.getByText("5.00%")).toBeInTheDocument();
  });

  it("comparison と theoreticalPrice が両方存在すると交渉推奨価格レンジが表示される", () => {
    render(
      <NegotiationPanel
        result={makeResult()}
        input={makeInput()}
        comparison={makeComparison()}
        theoreticalPrice={makeTheoreticalPrice({ theoreticalPriceJPY: 11_000_000 })}
      />
    );
    expect(screen.getByText("交渉推奨価格レンジ")).toBeInTheDocument();
  });

  it("売出価格がゼロの場合はダッシュ（—）が表示される", () => {
    const input = makeInput({ landPrice: 0, buildingCost: 0 });
    render(
      <NegotiationPanel
        result={makeResult()}
        input={input}
        comparison={null}
        theoreticalPrice={null}
      />
    );
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

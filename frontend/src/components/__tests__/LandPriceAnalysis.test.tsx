import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import { makeComparison, ZERO_STATS } from "./helpers";

describe("LandPriceAnalysis", () => {
  describe("count === 0 のとき", () => {
    it("「取引データが見つかりませんでした」を表示する", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: ZERO_STATS })} />);
      expect(screen.getByText("取引データが見つかりませんでした")).toBeInTheDocument();
    });

    it("判定バッジを表示しない", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: ZERO_STATS })} />);
      expect(screen.queryByText("割高")).not.toBeInTheDocument();
      expect(screen.queryByText("割安")).not.toBeInTheDocument();
      expect(screen.queryByText("相場")).not.toBeInTheDocument();
    });

    it("統計グリッドを表示しない", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: ZERO_STATS })} />);
      expect(screen.queryByText("最安値")).not.toBeInTheDocument();
      expect(screen.queryByText("中央値")).not.toBeInTheDocument();
    });
  });

  describe("lowDataWarning === true のとき", () => {
    const lowDataStats = {
      count: 3,
      averageTsubo: 300_000,
      medianTsubo: 290_000,
      minTsubo: 200_000,
      maxTsubo: 400_000,
      transactions: [],
      lowDataWarning: true,
      warningMessage: "取引件数が3件と少ないため統計の信頼性が低い可能性があります",
    };

    it("「統計データが不足しています」警告を表示する", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: lowDataStats })} />);
      expect(screen.getByText("統計データが不足しています")).toBeInTheDocument();
    });

    it("判定バッジに「(参考値)」を付与する", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: lowDataStats, assessment: "割高" })} />);
      expect(screen.getByText("割高（参考値）")).toBeInTheDocument();
    });

    it("比較エリアに「※ データ件数不足のため参考値」を表示する", () => {
      render(
        <LandPriceAnalysis
          comparison={makeComparison({ stats: lowDataStats, inputPricePerTsubo: 330_000 })}
        />
      );
      expect(screen.getByText("※ データ件数不足のため参考値")).toBeInTheDocument();
    });

    it("統計グリッドは表示する", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: lowDataStats })} />);
      expect(screen.getByText("最安値")).toBeInTheDocument();
      expect(screen.getByText("中央値")).toBeInTheDocument();
    });
  });

  describe("通常時 (count >= 10, lowDataWarning === false)", () => {
    it("判定バッジを「(参考値)」なしで表示する", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ assessment: "割安" })} />);
      expect(screen.getByText("割安")).toBeInTheDocument();
      expect(screen.queryByText(/参考値/)).not.toBeInTheDocument();
    });

    it("警告バナーを表示しない", () => {
      render(<LandPriceAnalysis comparison={makeComparison()} />);
      expect(screen.queryByText("統計データが不足しています")).not.toBeInTheDocument();
    });

    it("「取引データが見つかりませんでした」を表示しない", () => {
      render(<LandPriceAnalysis comparison={makeComparison()} />);
      expect(screen.queryByText("取引データが見つかりませんでした")).not.toBeInTheDocument();
    });
  });
});

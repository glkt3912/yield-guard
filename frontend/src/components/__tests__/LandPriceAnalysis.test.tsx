import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import { makeComparison, makeInput, ZERO_STATS } from "./helpers";

describe("LandPriceAnalysis", () => {
  describe("count === 0 のとき", () => {
    it("「このエリアの取引データは未収録です」を表示する", () => {
      render(<LandPriceAnalysis comparison={makeComparison({ stats: ZERO_STATS })} />);
      expect(screen.getByText("このエリアの取引データは未収録です")).toBeInTheDocument();
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
      render(
        <LandPriceAnalysis
          comparison={makeComparison({ stats: lowDataStats, assessment: "割高" })}
        />
      );
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

  describe("土地値割れ判定", () => {
    it("総取得価格 < 土地概算価値 のとき「土地値割れ」を表示する", () => {
      // medianTsubo=300_000, inputArea=100m² → 推定土地価値 ≒ 9,074,000円
      // landPrice+buildingCost = 5_000_000+5_000_000 = 10_000_000 → 土地値超になるためlandPriceを下げる
      const input = makeInput({ landPrice: 4_000_000, buildingCost: 2_000_000 }); // 計6,000,000 < 9,074,000
      render(<LandPriceAnalysis comparison={makeComparison({ inputArea: 100 })} input={input} />);
      expect(screen.getByText("土地値割れ")).toBeInTheDocument();
    });

    it("総取得価格 > 土地概算価値 × 1.5 のとき「土地値超」を表示する", () => {
      // medianTsubo=300_000, inputArea=100m² → 推定土地価値 ≒ 9,074,000円 × 1.5 = 13,611,000円
      const input = makeInput({ landPrice: 10_000_000, buildingCost: 10_000_000 }); // 計20,000,000 > 13,611,000
      render(<LandPriceAnalysis comparison={makeComparison({ inputArea: 100 })} input={input} />);
      expect(screen.getByText("土地値超")).toBeInTheDocument();
    });

    it("input が null のとき土地値判定セクションを表示しない", () => {
      render(<LandPriceAnalysis comparison={makeComparison()} input={null} />);
      expect(screen.queryByText("土地値割れ判定")).not.toBeInTheDocument();
    });

    it("stats.count === 0 のとき土地値判定セクションを表示しない", () => {
      render(
        <LandPriceAnalysis comparison={makeComparison({ stats: ZERO_STATS })} input={makeInput()} />
      );
      expect(screen.queryByText("土地値割れ判定")).not.toBeInTheDocument();
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

    it("「このエリアの取引データは未収録です」を表示しない", () => {
      render(<LandPriceAnalysis comparison={makeComparison()} />);
      expect(screen.queryByText("このエリアの取引データは未収録です")).not.toBeInTheDocument();
    });
  });
});

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InvestmentScoreCard } from "@/components/InvestmentScoreCard";
import type { InvestmentScoreResult, ScoreItem } from "@/types/investment";

function makeScoreItem(label: string, score: number): ScoreItem {
  return { label, score, description: `${label}の説明` };
}

function makeScore(totalScore: number, grade: string): InvestmentScoreResult {
  return {
    totalScore,
    grade,
    breakdown: {
      population: makeScoreItem("人口動態", 5),
      ridership: makeScoreItem("乗降客数", 10),
      urbanArea: makeScoreItem("市街化区域", 3),
      locationOptimization: makeScoreItem("立地最適化", 2),
      hazardRisk: makeScoreItem("ハザード", -5),
      liquefactionRisk: makeScoreItem("液状化", 0),
      embankment: makeScoreItem("盛土", 0),
      disasterHistory: makeScoreItem("災害履歴", 0),
      radarData: [],
    },
  };
}

describe("InvestmentScoreCard", () => {
  it("totalScore と grade が表示される", () => {
    render(<InvestmentScoreCard score={makeScore(78, "良好")} />);
    expect(screen.getByText("78")).toBeTruthy();
    expect(screen.getByText("良好")).toBeTruthy();
  });

  it("優良グレードで Badge が表示される", () => {
    render(<InvestmentScoreCard score={makeScore(90, "優良")} />);
    expect(screen.getByText("優良")).toBeTruthy();
  });

  it("要注意グレードで Badge が表示される", () => {
    render(<InvestmentScoreCard score={makeScore(30, "要注意")} />);
    expect(screen.getByText("要注意")).toBeTruthy();
  });

  it("各スコア項目のラベルが表示される", () => {
    render(<InvestmentScoreCard score={makeScore(60, "普通")} />);
    expect(screen.getByText("人口動態")).toBeTruthy();
    expect(screen.getByText("乗降客数")).toBeTruthy();
    expect(screen.getByText("ハザード")).toBeTruthy();
  });

  it("onApplyRecommend が渡された場合、推奨値適用ボタンが表示される", async () => {
    const onApply = vi.fn();
    render(<InvestmentScoreCard score={makeScore(70, "良好")} onApplyRecommend={onApply} />);
    const btn = screen.getByRole("button", { name: /推奨空室率/ });
    expect(btn).toBeTruthy();
    await userEvent.click(btn);
    expect(onApply).toHaveBeenCalledOnce();
  });

  it("onApplyRecommend が渡されない場合、推奨値適用ボタンは表示されない", () => {
    render(<InvestmentScoreCard score={makeScore(60, "普通")} />);
    expect(screen.queryByRole("button", { name: /推奨空室率/ })).toBeNull();
  });
});

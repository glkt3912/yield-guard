import type { components } from "@/types/api.generated";
type InvestmentScoreResult = components["schemas"]["domain.InvestmentScoreResult"];

export const investmentScoreFixture = {
  totalScore: 72,
  grade: "良好",
  breakdown: {
    population: { score: 70, label: "人口動態", description: "緩やかな減少傾向" },
    ridership: { score: 80, label: "乗降客数", description: "主要駅あり" },
    urbanArea: { score: 75, label: "市街化区域", description: "市街化区域内" },
    locationOptimization: { score: 60, label: "立地適正化", description: "対象外" },
    hazardRisk: { score: 60, label: "ハザードリスク", description: "低リスク" },
    liquefactionRisk: { score: 65, label: "液状化リスク", description: "低リスク" },
    embankment: { score: 80, label: "盛土", description: "盛土なし" },
    disasterHistory: { score: 90, label: "災害履歴", description: "履歴なし" },
    landPriceTrend: { score: 65, label: "地価トレンド", description: "安定" },
    radarData: [
      { category: "人口動態", score: 70 },
      { category: "乗降客数", score: 80 },
      { category: "市街化区域", score: 75 },
      { category: "立地適正化", score: 60 },
      { category: "ハザードリスク", score: 60 },
      { category: "液状化リスク", score: 65 },
      { category: "盛土", score: 80 },
      { category: "災害履歴", score: 90 },
      { category: "地価トレンド", score: 65 },
    ],
  },
} satisfies InvestmentScoreResult;

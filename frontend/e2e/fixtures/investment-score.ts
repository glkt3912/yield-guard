// Note: This fixture shape does not match the InvestmentScoreResult type from @/types/investment.
// The backend returns a flat score/rank/label/components structure; typed as const for now.
export const investmentScoreFixture = {
  score: 72,
  rank: "B",
  label: "良好",
  components: {
    yieldScore: 85,
    populationScore: 70,
    ridershipScore: 80,
    hazardScore: 60,
    landPriceScore: 65,
  },
} as const satisfies Record<string, unknown>;

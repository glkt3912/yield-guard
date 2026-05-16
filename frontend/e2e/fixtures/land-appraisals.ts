import type { components } from "@/types/api.generated";
type AppraisalComparisonResult = components["schemas"]["domain.AppraisalComparisonResult"];

export const landAppraisalsFixture = {
  appraisalMedianPerSqm: 980000,
  appraisalCount: 5,
  appraisalTrend: 0.02,
  trendLabel: "上昇",
} satisfies AppraisalComparisonResult;

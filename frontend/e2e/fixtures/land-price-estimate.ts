import type { components } from "@/types/api.generated";
type TheoreticalPriceResult = components["schemas"]["domain.TheoreticalPriceResult"];

export const landPriceEstimateFixture = {
  theoreticalPriceJPY: 11200000,
  deviationPct: 7.1,
  ageCorrection: -0.05,
  stationCorrection: -0.03,
  ridershipCorrection: 0.02,
  medianBuildingAge: 18,
  medianStationMinutes: 8,
  isLowDataWarning: false,
  hasStationData: true,
  ridershipScore: "B" as const,
  hasRidershipData: true,
} satisfies TheoreticalPriceResult;

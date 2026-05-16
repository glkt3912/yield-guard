import type { components } from "@/types/api.generated";
type LandPriceStats = components["schemas"]["domain.LandPriceStats"];

export const landPriceStatsFixture = {
  count: 23,
  averageTsubo: 480000,
  medianTsubo: 460000,
  minTsubo: 310000,
  maxTsubo: 750000,
  transactions: [],
  lowDataWarning: false,
} satisfies LandPriceStats;

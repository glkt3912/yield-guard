import type { components } from "@/types/api.generated";
type LandPriceComparison = components["schemas"]["domain.LandPriceComparison"];

export const landPriceCompareFixture = {
  stats: {
    count: 23,
    averageTsubo: 480000,
    medianTsubo: 460000,
    minTsubo: 310000,
    maxTsubo: 750000,
    transactions: [],
    lowDataWarning: false,
  },
  inputLandPrice: 12000000,
  inputArea: 80,
  inputPricePerTsubo: 495000,
  diffFromAverage: 15000,
  diffFromMedian: 35000,
  assessment: "割高" as const,
} satisfies LandPriceComparison;

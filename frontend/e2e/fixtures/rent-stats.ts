import type { components } from "@/types/api.generated";
type RentStatsResult = components["schemas"]["domain.RentStatsResult"];

export const rentStatsFixture = {
  median: 150000,
  average: 158000,
  count: 42,
} satisfies RentStatsResult;

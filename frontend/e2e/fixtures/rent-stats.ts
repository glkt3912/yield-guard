// Note: RentStatsResult from @/lib/api does not include lowDataWarning.
// The actual API response includes it; typed as const to preserve the full shape.
export const rentStatsFixture = {
  median: 150000,
  average: 158000,
  count: 42,
  lowDataWarning: false,
} as const satisfies Record<string, unknown>;

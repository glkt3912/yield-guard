// Note: This fixture shape does not match the AppraisalComparisonResult type from @/types/investment.
// The backend returns appraisals[]/averagePricePerSqm/inputPricePerSqm; typed as const for now.
export const landAppraisalsFixture = {
  appraisals: [
    {
      name: "渋谷5-1",
      pricePerSqm: 1520000,
      year: 2024,
      use: "商業地",
    },
    {
      name: "渋谷住宅-3",
      pricePerSqm: 980000,
      year: 2024,
      use: "住宅地",
    },
  ],
  averagePricePerSqm: 1250000,
  inputPricePerSqm: 150000,
} as const satisfies Record<string, unknown>;

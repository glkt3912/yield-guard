// Note: This fixture shape does not match the RentDeclineHint type from @/types/investment.
// The backend returns annualDeclineRate/dataPoints/lowDataWarning; typed as const for now.
export const rentDeclineHintFixture = {
  annualDeclineRate: 0.005,
  dataPoints: 18,
  lowDataWarning: false,
} as const satisfies Record<string, unknown>;

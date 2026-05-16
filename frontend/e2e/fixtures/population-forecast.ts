// Note: This fixture shape does not match the PopulationForecastResult type from @/types/investment.
// The backend returns a flat current/forecast/changeRate/trend structure; typed as const for now.
export const populationForecastFixture = {
  current: 42000,
  forecast2030: 39500,
  forecast2040: 36000,
  changeRate2030: -0.06,
  changeRate2040: -0.143,
  trend: "減少",
} as const satisfies Record<string, unknown>;

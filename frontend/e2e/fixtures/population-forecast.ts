import type { components } from "@/types/api.generated";
type PopulationForecastResult = components["schemas"]["domain.PopulationForecastResult"];

export const populationForecastFixture = {
  snapshots: [
    { year: 2020, pop: 42000 },
    { year: 2025, pop: 41000 },
    { year: 2030, pop: 39500 },
    { year: 2040, pop: 36000 },
    { year: 2050, pop: 32000 },
  ],
  changeRate30yr: -0.238,
  vacancyRateDelta: 0.04,
  trend: "緩やかな減少" as const,
} satisfies PopulationForecastResult;

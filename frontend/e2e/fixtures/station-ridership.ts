import type { components } from "@/types/api.generated";
type StationRidershipResult = components["schemas"]["domain.StationRidershipResult"];

export const stationRidershipFixture = [
  {
    stationName: "渋谷",
    lineName: "JR山手線",
    passengers: 420000,
    demandScore: "A" as const,
    correction: 0.1,
  },
] satisfies StationRidershipResult[];

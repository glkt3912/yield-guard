import type { components } from "@/types/api.generated";
type UrbanRisk = components["schemas"]["domain.UrbanRisk"];

export const hazardAlertFixture = [
  {
    code: "flood",
    level: "ERROR" as const,
    title: "洪水（計画規模）",
    description: "この地域は計画規模の洪水で3m以上の浸水が想定されます。",
  },
] satisfies UrbanRisk[];

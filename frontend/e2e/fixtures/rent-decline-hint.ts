import type { components } from "@/types/api.generated";
type RentDeclineHint = components["schemas"]["domain.RentDeclineHint"];

export const rentDeclineHintFixture = {
  hintRate: 0.005,
  basis: "land_appraisal",
  dataPointCount: 18,
  fallbackUsed: false,
  note: "直近3年の地価公示CAGRから推計",
} satisfies RentDeclineHint;

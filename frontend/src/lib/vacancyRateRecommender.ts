/**
 * vacancyRateRecommender.ts
 *
 * Derives vacancy rate and rent decline rate recommendations from
 * ridership demand score (A–E) and population 30-year change rate.
 */

export type RidershipGrade = "A" | "B" | "C" | "D" | "E";

export interface VacancyRecommendation {
  vacancyRate: number;
  rentDeclineRate: number;
  grade: RidershipGrade;
}

/**
 * Mapping from ridership demand score (0–20, linear from backend) to grade.
 * Backend formula: score = maxPassengers / 200_000 * 20
 *   A: 20万人+ → score ≥ 20 (clamped to 20)
 *   B: 10–20万 → score ≥ 10
 *   C:  5–10万 → score ≥ 5
 *   D:  1– 5万 → score ≥ 1
 *   E:    <1万 → score < 1
 */
// Backend clamps ridership score to [0, 20] via math.Min, so >= 20 safely captures the top grade.
export function ridershipGradeFromScore(score: number): RidershipGrade {
  if (score >= 20) return "A";
  if (score >= 10) return "B";
  if (score >= 5) return "C";
  if (score >= 1) return "D";
  return "E";
}

const GRADE_RECOMMEND: Record<RidershipGrade, { vacancyRate: number; rentDeclineRate: number }> = {
  A: { vacancyRate: 0.03, rentDeclineRate: 0.005 },
  B: { vacancyRate: 0.05, rentDeclineRate: 0.01 },
  C: { vacancyRate: 0.07, rentDeclineRate: 0.015 },
  D: { vacancyRate: 0.1, rentDeclineRate: 0.02 },
  E: { vacancyRate: 0.15, rentDeclineRate: 0.03 },
};

/**
 * Returns recommended vacancy rate and rent decline rate for a given
 * ridership demand score (0–20 as returned by the investment-score API).
 */
export function getRidershipRecommend(ridershipScore: number): VacancyRecommendation {
  const grade = ridershipGradeFromScore(ridershipScore);
  return { grade, ...GRADE_RECOMMEND[grade] };
}

const POPULATION_DECLINE_THRESHOLD = -0.3;
const POPULATION_VACANCY_PENALTY = 0.03;

/**
 * Returns the vacancy rate delta to add when population 30-year change rate
 * is −30% or worse (i.e. changeRate30yr ≤ −0.30).
 */
export function getPopulationVacancyDelta(changeRate30yr: number): number {
  return changeRate30yr <= POPULATION_DECLINE_THRESHOLD ? POPULATION_VACANCY_PENALTY : 0;
}

import type { InvestmentInput } from "@/types/investment";

export const SAMPLE_PROPERTY: InvestmentInput = {
  landPrice: 20_000_000,
  landArea: 100,
  buildingCost: 15_000_000,
  buildingAge: 15,
  stationMinutes: 8,
  miscExpenseRate: 0.07,
  monthlyRent: 250_000,
  vacancyRate: 0.05,
  actualVacancyRate: 0,
  loanAmount: 24_000_000,
  annualLoanRate: 0.018,
  loanYears: 30,
  buildingType: "木造",
  expenseRate: 0.2,
  incomeTaxRate: 0.33,
  holdingYears: 10,
  exitYieldTarget: 0.06,
  vacancyRateDelta: 0,
  loanRateDelta: 0,
  annualPropertyTax: 0,
  rentDeclineRate: 0.005,
  yieldTarget: 0.08,
  loanMethod: "equal-payment",
  rateAdjustmentSchedule: [],
  discountRate: 0.05,
  priceDeclineRate: 0.02,
  depreciationMethod: "straight-line" as const,
};

export const ONBOARDING_KEY = "yield-guard:onboarded";

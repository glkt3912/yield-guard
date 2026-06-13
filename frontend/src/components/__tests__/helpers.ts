import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceComparison,
  LandPriceStats,
  YearlyResult,
} from "@/types/investment";

export function makeInput(overrides: Partial<InvestmentInput> = {}): InvestmentInput {
  return {
    landPrice: 10_000_000,
    landArea: 100,
    buildingCost: 5_000_000,
    buildingAge: 10,
    stationMinutes: 0,
    miscExpenseRate: 0.07,
    monthlyRent: 120_000,
    vacancyRate: 0.05,
    actualVacancyRate: 0,
    loanAmount: 13_000_000,
    annualLoanRate: 0.015,
    loanYears: 35,
    buildingType: "木造",
    expenseRate: 0.2,
    incomeTaxRate: 0.33,
    holdingYears: 10,
    exitYieldTarget: 0.06,
    vacancyRateDelta: 0,
    loanRateDelta: 0,
    annualPropertyTax: 0,
    rentDeclineRate: 0,
    yieldTarget: 0.08,
    rateAdjustmentSchedule: [],
    ...overrides,
  };
}

export const ZERO_STATS: LandPriceStats = {
  count: 0,
  averageTsubo: 0,
  medianTsubo: 0,
  minTsubo: 0,
  maxTsubo: 0,
  transactions: [],
  lowDataWarning: false,
};

export function makeComparison(overrides: Partial<LandPriceComparison> = {}): LandPriceComparison {
  return {
    stats: {
      count: 15,
      averageTsubo: 300_000,
      medianTsubo: 290_000,
      minTsubo: 200_000,
      maxTsubo: 400_000,
      transactions: [],
      lowDataWarning: false,
    },
    inputLandPrice: 10_000_000,
    inputArea: 100,
    inputPricePerTsubo: 330_000,
    diffFromAverage: 30_000,
    diffFromMedian: 40_000,
    assessment: "割高",
    ...overrides,
  };
}

export function makeYearlyResult(
  year: number,
  overrides: Partial<YearlyResult> = {}
): YearlyResult {
  return {
    year,
    annualRent: 1_200_000,
    annualLoanPayment: 600_000,
    annualInterest: 100_000,
    annualPrincipal: 500_000,
    annualDepreciation: 600_000,
    annualExpenses: 240_000,
    taxableIncome: 260_000,
    incomeTax: 85_800,
    cashFlow: 360_000,
    afterTaxCashFlow: 274_200,
    remainingLoanBalance: 12_000_000,
    cumulativeCashFlow: 274_200 * year,
    isDeadCrossYear: false,
    isInDeadCrossZone: false,
    effectiveRate: 0.015,
    capexAmount: 0,
    ...overrides,
  };
}

export function makeResult(overrides: Partial<InvestmentResult> = {}): InvestmentResult {
  const yearlyResults = Array.from({ length: 35 }, (_, i) => makeYearlyResult(i + 1));
  return {
    totalInvestment: 16_050_000,
    miscExpenses: 1_050_000,
    grossYield: 0.09,
    grossYieldOnTotalInvestment: 0.084,
    netYield: 0.065,
    isAboveYieldTarget: true,
    yieldTarget: 0.08,
    requiredCostReduction: 0,
    requiredMonthlyRent: 120_000,
    deadCrossYear: 0,
    yearlyResults,
    criticalErrors: [],
    acquisitionCosts: {
      brokerageFee: 561_000,
      stampDuty: 20_000,
      registrationTax: 420_000,
      realEstateAcquisitionTax: 315_000,
      propertyTaxProration: 0,
      total: 1_316_000,
    },
    exitSalePrice: 12_000_000,
    exitCapitalGain: 2_000_000,
    exitTransferTax: 400_000,
    exitNetProceeds: 11_600_000,
    exitTotalEquity: 3_000_000,
    stressScenarios: [],
    yieldScenarios: {
      optimistic: { annualRent: 1_368_000, grossYield: 0.09 },
      standard: { annualRent: 1_296_000, grossYield: 0.09 },
      pessimistic: { annualRent: 1_224_000, grossYield: 0.09 },
    },
    dscr: 1.2,
    ltvSensitivity: [],
    irr: 0.07,
    npv: 1_200_000,
    totalInterest: 3_500_000,
    ...overrides,
  };
}

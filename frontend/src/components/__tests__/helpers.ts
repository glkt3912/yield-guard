import type { InvestmentResult, YearlyResult } from "@/types/investment";

function makeYearlyResult(year: number, overrides: Partial<YearlyResult> = {}): YearlyResult {
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
    ...overrides,
  };
}

export function makeResult(overrides: Partial<InvestmentResult> = {}): InvestmentResult {
  const yearlyResults = Array.from({ length: 35 }, (_, i) => makeYearlyResult(i + 1));
  return {
    totalInvestment: 16_050_000,
    miscExpenses: 1_050_000,
    grossYield: 0.09,
    netYield: 0.065,
    isAbove8Percent: true,
    requiredCostReduction: 0,
    requiredMonthlyRent: 120_000,
    deadCrossYear: 0,
    yearlyResults,
    exitSalePrice: 12_000_000,
    exitCapitalGain: 2_000_000,
    exitTransferTax: 400_000,
    exitNetProceeds: 11_600_000,
    exitTotalEquity: 3_000_000,
    ...overrides,
  };
}

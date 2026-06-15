import type { components } from "@/types/api.generated";
// IRR は Go の *float64 (nullable) だが生成型では irr?: number のため Omit で再定義して null を許容する
type NullableMultiExitRow = Omit<components["schemas"]["domain.MultiExitRow"], "irr"> & {
  irr?: number | null;
};
type InvestmentResult = Omit<
  components["schemas"]["domain.InvestmentResult"],
  "irr" | "multiExitComparison"
> & {
  irr?: number | null;
  multiExitComparison?: NullableMultiExitRow[];
};

export const analyzeCriticalFixture = {
  totalInvestment: 18190000,
  miscExpenses: 1190000,
  grossYield: 0.033,
  marketGrossYield: 0.0353, // 物件価格(17,000,000)ベースの表面利回り（#773）
  netYield: 0.012,
  isAboveYieldTarget: false,
  yieldTarget: 0.08,
  requiredCostReduction: 8500000,
  requiredMonthlyRent: 210000,
  deadCrossYear: 3,
  dscr: 0.72,
  irr: null,
  npv: -4200000,
  totalInterest: 2100000,
  criticalErrors: [
    {
      code: "NEGATIVE_CF",
      status: "REJECT" as const,
      message: "初年度キャッシュフローが大幅マイナスです。投資を見直してください。",
    },
  ],
  acquisitionCosts: {
    brokerageFee: 627000,
    stampDuty: 20000,
    registrationTax: 480000,
    realEstateAcquisitionTax: 0,
    propertyTaxProration: 63000,
    total: 1190000,
  },
  exitSalePrice: 9000000,
  exitCapitalGain: -3200000,
  exitTransferTax: 0,
  exitNetProceeds: 9000000,
  exitTotalEquity: -2500000,
  stressScenarios: [
    {
      label: "ベースライン",
      interestRateDelta: 0,
      vacancyRateDelta: 0,
      totalCashFlow: -1200000,
      dscr: 0.72,
      breakEvenYear: -1,
      isSafe: false,
    },
  ],
  yieldScenarios: {
    optimistic: { annualRent: 720000, grossYield: 0.0396 },
    standard: { annualRent: 600000, grossYield: 0.033 },
    pessimistic: { annualRent: 480000, grossYield: 0.0264 },
  },
  ltvSensitivity: [],
  yearlyResults: [
    {
      year: 1,
      annualRent: 570000,
      annualLoanPayment: 630432,
      annualInterest: 204000,
      annualPrincipal: 426432,
      annualDepreciation: 145000,
      annualExpenses: 57000,
      taxableIncome: 164000,
      incomeTax: 32800,
      capexAmount: 0,
      cashFlow: -60432,
      afterTaxCashFlow: -93232,
      remainingLoanBalance: 13189568,
      cumulativeCashFlow: -93232,
      isDeadCrossYear: false,
      isInDeadCrossZone: false,
      effectiveRate: 0.015,
    },
  ],
  aiSummary: "",
  multiExitComparison: [
    {
      year: 5,
      salePrice: 9000000,
      transferTaxRate: 0.3963,
      transferTax: 0,
      remainingLoan: 11421312,
      cumulativeCf: -466160,
      exitEquity: -3320000,
      irr: null,
      isShortTermWarn: true,
    },
    {
      year: 10,
      salePrice: 8000000,
      transferTaxRate: 0.20315,
      transferTax: 0,
      remainingLoan: 9850000,
      cumulativeCf: -932320,
      exitEquity: -3600000,
      irr: null,
      isShortTermWarn: false,
    },
  ],
} satisfies InvestmentResult;

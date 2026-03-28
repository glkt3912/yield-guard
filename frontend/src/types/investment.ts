export type BuildingType =
  | "木造"
  | "軽量鉄骨(4mm以下)"
  | "軽量鉄骨(3mm以下)"
  | "重量鉄骨"
  | "RC造"
  | "SRC造";

export interface InvestmentInput {
  landPrice: number;
  landArea: number;       // 土地面積 (m²)
  buildingCost: number;
  buildingAge: number;    // 築年数 (0=新築)
  miscExpenseRate: number;
  monthlyRent: number;
  vacancyRate: number;
  loanAmount: number;
  annualLoanRate: number;
  loanYears: number;
  buildingType: BuildingType;
  expenseRate: number;
  incomeTaxRate: number;
  holdingYears: number;
  exitYieldTarget: number;
  vacancyRateDelta: number;
  loanRateDelta: number;
}

export interface YearlyResult {
  year: number;
  annualRent: number;           // 実効賃料収入（空室控除後）
  annualLoanPayment: number;
  annualInterest: number;
  annualPrincipal: number;
  annualDepreciation: number;
  annualExpenses: number;
  taxableIncome: number;
  incomeTax: number;
  cashFlow: number;
  afterTaxCashFlow: number;
  remainingLoanBalance: number;
  cumulativeCashFlow: number;
  isDeadCrossYear: boolean;
  isInDeadCrossZone: boolean;   // デッドクロス継続ゾーン
}

export interface InvestmentResult {
  totalInvestment: number;
  miscExpenses: number;
  grossYield: number;
  netYield: number;
  isAbove8Percent: boolean;
  requiredCostReduction: number;  // 土地または建築費いずれか一方の削減必要額
  requiredMonthlyRent: number;
  deadCrossYear: number;
  yearlyResults: YearlyResult[];
  exitSalePrice: number;
  exitCapitalGain: number;
  exitTransferTax: number;
  exitNetProceeds: number;
  exitTotalEquity: number;
}

export interface LandTransaction {
  period: string;
  district: string;
  tradePrice: number;
  area: number;
  pricePerSqm: number;
  pricePerTsubo: number;
  cityPlanning: string;
  buildingCoverage: string;
  floorAreaRatio: string;
}

export interface LandPriceStats {
  count: number;
  averageTsubo: number;
  medianTsubo: number;
  minTsubo: number;
  maxTsubo: number;
  transactions: LandTransaction[];
  lowDataWarning: boolean;
  warningMessage?: string;
}

export interface LandPriceComparison {
  stats: LandPriceStats;
  inputLandPrice: number;
  inputArea: number;
  inputPricePerTsubo: number;
  diffFromAverage: number;
  diffFromMedian: number;
  assessment: "割安" | "相場" | "割高";
}

export const DEFAULT_INPUT: InvestmentInput = {
  landPrice: 5_000_000,
  landArea: 100,
  buildingCost: 10_000_000,
  buildingAge: 0,
  miscExpenseRate: 0.07,
  monthlyRent: 120_000,
  vacancyRate: 0.05,
  loanAmount: 13_000_000,
  annualLoanRate: 0.015,
  loanYears: 35,
  buildingType: "木造",
  expenseRate: 0.20,
  incomeTaxRate: 0.33,
  holdingYears: 10,
  exitYieldTarget: 0.06,
  vacancyRateDelta: 0,
  loanRateDelta: 0,
};

export const BUILDING_USEFUL_LIFE: Record<BuildingType, number> = {
  "木造": 22,
  "軽量鉄骨(3mm以下)": 19,
  "軽量鉄骨(4mm以下)": 27,
  "重量鉄骨": 34,
  "RC造": 47,
  "SRC造": 47,
};

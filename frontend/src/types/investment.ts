export interface CriticalError {
  code: string;
  status: "REJECT" | "WARNING";
  message: string;
}

export interface AcquisitionCostBreakdown {
  brokerageFee: number;
  stampDuty: number;
  registrationTax: number;
  realEstateAcquisitionTax: number;
  propertyTaxProration: number;
  total: number;
}

export type BuildingType =
  | "木造"
  | "軽量鉄骨(4mm以下)"
  | "軽量鉄骨(3mm以下)"
  | "重量鉄骨"
  | "RC造"
  | "SRC造";

export type RidershipDemandScore = "A" | "B" | "C" | "D" | "E";

export interface TheoreticalPriceResult {
  theoreticalPriceJPY: number;
  deviationPct: number;
  ageCorrection: number;
  stationCorrection: number;
  ridershipCorrection: number;
  medianBuildingAge: number;
  medianStationMinutes: number;
  isLowDataWarning: boolean;
  hasStationData: boolean;
  ridershipScore?: RidershipDemandScore;
  hasRidershipData: boolean;
}

export interface StationRidershipResult {
  stationName: string;
  lineName: string;
  passengers: number;
  demandScore: RidershipDemandScore;
  correction: number;
}

export interface InvestmentInput {
  landPrice: number;
  landArea: number;       // 土地面積 (m²)
  buildingCost: number;
  buildingAge: number;    // 築年数 (0=新築)
  stationMinutes: number; // 最寄り駅徒歩分 (0=未入力)
  miscExpenseRate: number;
  monthlyRent: number;
  vacancyRate: number;        // 想定空室率（長期シミュレーション用）
  actualVacancyRate: number;  // 現況空室率（現時点の実態）
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
  annualPropertyTax: number;  // 固定資産税・都市計画税（年間）。0 = ExpenseRateに含む。
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
  requiredCostReduction: number;
  requiredMonthlyRent: number;
  deadCrossYear: number;
  yearlyResults: YearlyResult[];
  criticalErrors: CriticalError[];
  acquisitionCosts: AcquisitionCostBreakdown;
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
  zoning?: ZoningSummary;
  urbanRisks?: UrbanRisk[];
}

export interface ZoningSummary {
  cityPlanning: string;
  buildingCoverage: string;
  floorAreaRatio: string;
}

export type UrbanRiskLevel = "ERROR" | "WARNING" | "INFO";

export interface UrbanRisk {
  code: string;
  level: UrbanRiskLevel;
  title: string;
  description: string;
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

export interface PopulationSnapshot {
  year: number;
  pop: number;
}

export interface PopulationForecastResult {
  snapshots: PopulationSnapshot[];
  changeRate30yr: number;
  vacancyRateDelta: number;
  trend: "増加" | "現状維持" | "緩やかな減少" | "急激な減少";
}

export interface AppraisalComparisonResult {
  appraisalMedianPerSqm: number; // 公示価格中央値（円/m²）
  appraisalCount: number;         // 標準地点数
  appraisalTrend: number;         // 平均変動率（小数: 0.05 = +5%）
  trendLabel: "上昇" | "安定" | "下落";
}

export const DEFAULT_INPUT: InvestmentInput = {
  landPrice: 5_000_000,
  landArea: 100,
  buildingCost: 10_000_000,
  buildingAge: 0,
  stationMinutes: 0,
  miscExpenseRate: 0.07,
  monthlyRent: 120_000,
  vacancyRate: 0.05,
  actualVacancyRate: 0,
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
  annualPropertyTax: 0,
};

export const BUILDING_USEFUL_LIFE: Record<BuildingType, number> = {
  "木造": 22,
  "軽量鉄骨(3mm以下)": 19,
  "軽量鉄骨(4mm以下)": 27,
  "重量鉄骨": 34,
  "RC造": 47,
  "SRC造": 47,
};

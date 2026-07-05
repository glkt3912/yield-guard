/**
 * バックエンド API 契約に対応する型定義。
 *
 * バックエンドと厳密に一致する型は生成型 (api.generated.ts) の再エクスポート。
 * フィールドコメントは Go 側のコメントが JSDoc として生成型に引き継がれている。
 * 生成型との整合は types/__tests__/api-contract.ts で `tsc --noEmit` により検証される。
 *
 * 手書きのまま残している型:
 * - バックエンドより厳しい絞り込みを持つ型（生成型は string のためユニオンで補強）:
 *   LoanMethod / GeocodeResult.locationType / RentDeclineHint.basis /
 *   AppraisalComparisonResult.trendLabel / LandPriceComparison.assessment
 * - InvestmentInput: フロント固有フィールド stationMinutes（理論価格推定 API の
 *   クエリ用、simulate 契約外）を含むため
 * - フロント固有型（バックエンド契約に対応スキーマがなく、契約検証の対象外）:
 *   SimulationMode / WatchlistStatus / WatchlistMetrics / WatchlistItem
 */
import type { components } from "@/types/api.generated";

type Schemas = components["schemas"];

// ==== バックエンド生成型の再エクスポート ====

export type Municipality = Schemas["mlit.Municipality"];
export type CriticalError = Schemas["domain.CriticalError"];
export type AcquisitionCostBreakdown = Schemas["domain.AcquisitionCostBreakdown"];
export type BuildingType = Schemas["domain.BuildingType"];
export type RidershipDemandScore = Schemas["domain.RidershipDemandScore"];
export type TheoreticalPriceResult = Schemas["domain.TheoreticalPriceResult"];
export type StationRidershipResult = Schemas["domain.StationRidershipResult"];
export type RateAdjustment = Schemas["domain.RateAdjustment"];
export type MultiExitRow = Schemas["domain.MultiExitRow"];
export type CapexEvent = Schemas["domain.CapexEvent"];
export type YearlyResult = Schemas["domain.YearlyResult"];
export type StressScenarioResult = Schemas["domain.StressScenarioResult"];
export type YieldScenario = Schemas["domain.YieldScenario"];
export type YieldScenarios = Schemas["domain.YieldScenarios"];
export type LTVSensitivityRow = Schemas["domain.LTVSensitivityRow"];
export type InvestmentResult = Schemas["domain.InvestmentResult"];
export type TaxSimRow = Schemas["domain.TaxSimRow"];
export type SalaryLossCarryoverResult = Schemas["domain.SalaryLossCarryoverResult"];
export type OwnershipScenario = Schemas["domain.OwnershipScenario"];
export type OwnershipComparisonResult = Schemas["domain.OwnershipComparisonResult"];
export type TaxSimulationResult = Schemas["domain.TaxSimulationResult"];
export type LandTransaction = Schemas["domain.LandTransaction"];
export type LandPriceStats = Schemas["domain.LandPriceStats"];
export type ZoningSummary = Schemas["domain.ZoningSummary"];
export type UrbanRiskLevel = Schemas["domain.UrbanRiskLevel"];
export type UrbanRisk = Schemas["domain.UrbanRisk"];
export type PopulationSnapshot = Schemas["domain.PopulationSnapshot"];
export type PopulationForecastResult = Schemas["domain.PopulationForecastResult"];
export type Percentiles = Schemas["domain.Percentiles"];
export type HistogramBin = Schemas["domain.HistogramBin"];
export type MonteCarloResult = Schemas["domain.MonteCarloResult"];
export type RenovationItem = Schemas["domain.RenovationItem"];
export type ClassifiedRenovationItem = Schemas["domain.ClassifiedRenovationItem"];
export type RenovationInput = Schemas["domain.RenovationInput"];
export type RenovationResult = Schemas["domain.RenovationResult"];
export type ScoreItem = Schemas["domain.ScoreItem"];
export type RadarPoint = Schemas["domain.RadarPoint"];
export type ScoreBreakdown = Schemas["domain.ScoreBreakdown"];
export type InvestmentScoreResult = Schemas["domain.InvestmentScoreResult"];
export type HeatmapTile = Schemas["domain.HeatmapTile"];
export type HeatmapResponse = Schemas["domain.HeatmapResponse"];

// ==== 手書き型（フロント側でユニオンに絞り込み） ====

export type LoanMethod = "equal-payment" | "equal-principal";

export interface InvestmentInput {
  landPrice: number;
  landArea: number; // 土地面積 (m²)
  buildingCost: number;
  buildingAge: number; // 築年数 (0=新築)
  stationMinutes: number; // 最寄り駅徒歩分 (0=未入力)。フロント固有: 理論価格推定APIに渡す入力で、simulate契約には含まれない（api-contract.ts で除外）
  miscExpenseRate: number;
  monthlyRent: number;
  vacancyRate: number; // 想定空室率（長期シミュレーション用）
  actualVacancyRate: number; // 現況空室率（現時点の実態）
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
  annualPropertyTax: number; // 固定資産税・都市計画税（年間）。0 = ExpenseRateに含む。
  rentDeclineRate: number; // 年間賃料下落率（例: 0.01 = 1%/年）
  yieldTarget: number; // 目標表面利回り（例: 0.08 = 8%）
  loanMethod?: LoanMethod; // 返済方式（省略時 = equal-payment）
  rateAdjustmentSchedule: RateAdjustment[]; // 変動金利スケジュール（空=固定金利）
  discountRate?: number; // 割引率（NPV/IRR計算用、例: 0.05 = 5%）
  priceDeclineRate?: number; // 物件価格下落率（例: 0.02 = 年2%下落）
  depreciationMethod?: "straight-line" | "declining-balance"; // 減価償却方式
  capexSchedule?: CapexEvent[]; // 大規模修繕費スケジュール（最大5件）
  rentGrowthRate?: number; // 年間賃料上昇率（新築・リノベ向け、例: 0.02 = 2%）
  rentGrowthYears?: number; // 賃料上昇が続く年数
  loanFeeRate?: number; // 融資諸費用率（保証料・登記費用等の合算）
  exitYears?: number[]; // 複数保有年数の出口比較（省略時 = [5, 10, 15, 20]）
  // 詳細経費内訳（全てoptional・後方互換）。合計 > 0 の場合は expenseRate より優先される。
  managementFeeRate?: number; // 管理委託費率 (例: 0.05 = 5%)
  repairReserveRate?: number; // 修繕積立費率 (例: 0.01 = 1%)
  insuranceFeeRate?: number; // 保険料率 (例: 0.003 = 0.3%)
  otherExpenseRate?: number; // その他経費率 (例: 0.005 = 0.5%)
  expenseInflationRate?: number; // 経費インフレ率/年 (例: 0.01 = 1%)

  // 入退去コスト（全て optional。0 または未入力の場合はターンオーバーコスト = 0）
  avgTenancyYears?: number; // 平均入居期間（年）例: 2.0
  restorationCost?: number; // 原状回復費（円/回）例: 150000
  adFee?: number; // 入居者募集 AD（円/回）例: 家賃1ヶ月分
  rentFreePeriod?: number; // フリーレント（ヶ月）例: 0.5

  // 税務シミュレーション用（任意）。0 の場合は損益通算・法人比較を計算しない。
  salaryIncome?: number; // 給与年収（円）

  // 所有権保存登記（新築直売）か移転登記（中古・転売）か。省略時はバックエンドが buildingAge==0 から自動判定。
  isFirstRegistration?: boolean;
}

export interface MonteCarloInput {
  base: InvestmentInput;
  simulations?: number;
  vacancyRateSigma?: number;
  loanRateSigma?: number;
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

export interface AppraisalComparisonResult {
  appraisalMedianPerSqm: number; // 公示価格中央値（円/m²）
  appraisalCount: number; // 標準地点数
  appraisalTrend: number; // 平均変動率（小数: 0.05 = +5%）
  trendLabel: "上昇" | "安定" | "下落";
}

export interface RentDeclineHint {
  hintRate: number;
  basis: "land_appraisal" | "fallback";
  dataPointCount: number;
  fallbackUsed: boolean;
  note: string;
}

export interface GeocodeResult {
  lat: number;
  lng: number;
  locationType: "ROOFTOP" | "RANGE_INTERPOLATED" | "GEOMETRIC_CENTER" | "APPROXIMATE";
}

// ==== フロント固有型・定数 ====

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
  expenseRate: 0.2,
  incomeTaxRate: 0.33,
  holdingYears: 10,
  exitYieldTarget: 0.06,
  vacancyRateDelta: 0,
  loanRateDelta: 0,
  annualPropertyTax: 0,
  rentDeclineRate: 0,
  yieldTarget: 0.08,
  loanMethod: "equal-payment",
  rateAdjustmentSchedule: [],
  discountRate: 0.05,
  priceDeclineRate: 0,
  depreciationMethod: "straight-line" as const,
};

export const BUILDING_USEFUL_LIFE: Record<BuildingType, number> = {
  木造: 22,
  "軽量鉄骨(3mm以下)": 19,
  "軽量鉄骨(4mm以下)": 27,
  重量鉄骨: 34,
  RC造: 47,
  SRC造: 47,
};

export type SimulationMode = "quick" | "full";

export const QUICK_MODE_DEFAULTS: Partial<InvestmentInput> = {
  landArea: 100,
  buildingAge: 0,
  stationMinutes: 0,
  miscExpenseRate: 0.07,
  vacancyRate: 0.05,
  actualVacancyRate: 0,
  expenseRate: 0.2,
  incomeTaxRate: 0.33,
  exitYieldTarget: 0.06,
  vacancyRateDelta: 0,
  loanRateDelta: 0,
  annualPropertyTax: 0,
  buildingType: "木造" as BuildingType,
  annualLoanRate: 0.015,
  rentDeclineRate: 0.01,
  yieldTarget: 0.08,
  loanMethod: "equal-payment",
  rateAdjustmentSchedule: [],
};

export const DEFAULT_RENOVATION_INPUT: RenovationInput = {
  propertyPrice: 10_000_000,
  annualBaseRent: 1_200_000,
  annualExpenses: 240_000,
  effectiveTaxRate: 0.3,
  selfLaborRatePerHour: 0,
  items: [],
};

export type WatchlistStatus = "検討中" | "見送り" | "購入済み";

export interface WatchlistMetrics {
  grossYield: number; // 総投資利回り（満室想定年収/総投資額）
  marketGrossYield?: number; // 表面利回り（物件価格ベース）。旧データには存在しないため optional
  netYield: number;
  dscr: number;
  irr: number | null;
  totalInvestment: number;
  exitTotalEquity: number;
  deadCrossYear?: number; // -1 = none (best)
  npv?: number;
  monthlyPayment?: number; // 月々の返済額（円）
}

export interface WatchlistItem {
  id: string;
  name: string;
  memo: string;
  status: WatchlistStatus;
  addedAt: string; // ISO 8601
  metrics?: WatchlistMetrics;
}

/**
 * API 契約テスト（コンパイル時のみ・ランタイムでは実行されない）
 *
 * 手書き型 (types/investment.ts, lib/api.ts) と swagger.json 由来の生成型
 * (types/api.generated.ts) のドリフトを `tsc --noEmit` で検出する。
 * バックエンドの Go 型を変更したら `make swagger` → `npm run generate:types`
 * を実行すると、このファイルがコンパイルエラーになって不整合を知らせる。
 *
 * オブジェクト型の検証は CheckContract で2段構え:
 * 1. キー集合一致 — フィールドの追加・削除・改名を検出
 *    （生成型は swag の制約で全フィールド optional のため、
 *      代入可能性チェックだけではフィールド欠落を検出できない）
 * 2. フィールド単位の代入可能性 — 同名フィールドの型不一致を検出
 *    （ネストした型も構造的に検証される）
 * ユニオン型（enum 由来）は Equals でメンバー集合の完全一致を検証する。
 *
 * フロント固有型はバックエンド契約に対応がないため対象外
 * （investment.ts 冒頭のコメント参照）。
 */
import type { components } from "@/types/api.generated";
import type { AreaDiscoveryItem, AreaDiscoveryResponse, RentStatsResult } from "@/lib/api";
import type {
  AcquisitionCostBreakdown,
  AppraisalComparisonResult,
  BuildingType,
  CapexEvent,
  ClassifiedRenovationItem,
  CriticalError,
  GeocodeResult,
  HeatmapResponse,
  HeatmapTile,
  HistogramBin,
  InvestmentInput,
  InvestmentResult,
  InvestmentScoreResult,
  LandPriceComparison,
  LandPriceStats,
  LandTransaction,
  LTVSensitivityRow,
  MonteCarloInput,
  MonteCarloResult,
  MultiExitRow,
  Municipality,
  OwnershipComparisonResult,
  OwnershipScenario,
  Percentiles,
  PopulationForecastResult,
  PopulationSnapshot,
  RadarPoint,
  RateAdjustment,
  RenovationInput,
  RenovationItem,
  RenovationResult,
  RentDeclineHint,
  RidershipDemandScore,
  SalaryLossCarryoverResult,
  ScoreBreakdown,
  ScoreItem,
  StationRidershipResult,
  StressScenarioResult,
  TaxSimRow,
  TaxSimulationResult,
  TheoreticalPriceResult,
  UrbanRisk,
  UrbanRiskLevel,
  YearlyResult,
  YieldScenario,
  YieldScenarios,
  ZoningSummary,
} from "@/types/investment";

type Schemas = components["schemas"];

/** 同名フィールドのうち、手書き型が生成型に代入できないフィールド名を列挙する */
type BadFields<Hand, Gen> = {
  [K in keyof Hand & keyof Gen]: [Hand[K]] extends [Gen[K]] ? never : K;
}[keyof Hand & keyof Gen];

/**
 * オブジェクト型の契約検証。合格なら true、不合格なら原因を示す
 * オブジェクト型（missingInHandwritten / unknownToBackend / fieldTypeMismatch）
 * に解決され、AssertAll の制約違反としてエラーになる。
 */
type CheckContract<Hand, Gen> = [
  Exclude<keyof Gen, keyof Hand>,
  Exclude<keyof Hand, keyof Gen>,
] extends [never, never]
  ? [BadFields<Hand, Gen>] extends [never]
    ? true
    : { fieldTypeMismatch: BadFields<Hand, Gen> }
  : {
      missingInHandwritten: Exclude<keyof Gen, keyof Hand>;
      unknownToBackend: Exclude<keyof Hand, keyof Gen>;
    };

/** ユニオン型のメンバー集合が完全一致するかを検証する */
type Equals<Hand, Gen> = [Hand] extends [Gen]
  ? [Gen] extends [Hand]
    ? true
    : { genOnlyMembers: Exclude<Gen, Hand> }
  : { handOnlyMembers: Exclude<Hand, Gen> };

type AssertAll<Checks extends { [K in keyof Checks]: true }> = Checks;

/**
 * フロント固有フィールド（バックエンドの simulate 契約に含まれないもの）。
 * ここに追加する場合は investment.ts 側のコメントにも明記すること。
 * - stationMinutes: 理論価格推定 API (/land-prices/estimate) のクエリ用
 */
type FrontendOnlyInputFields = "stationMinutes";
type ContractInvestmentInput = Omit<InvestmentInput, FrontendOnlyInputFields>;

/** スキーマ名 → 対応する手書き型。バックエンド由来のオブジェクト型はここに全て列挙する */
type HandwrittenBySchema = {
  "api.GeocodeResult": GeocodeResult;
  "domain.AcquisitionCostBreakdown": AcquisitionCostBreakdown;
  "domain.AppraisalComparisonResult": AppraisalComparisonResult;
  "domain.AreaDiscoveryItem": AreaDiscoveryItem;
  "domain.AreaDiscoveryResponse": AreaDiscoveryResponse;
  "domain.CapexEvent": CapexEvent;
  "domain.ClassifiedRenovationItem": ClassifiedRenovationItem;
  "domain.CriticalError": CriticalError;
  "domain.HeatmapResponse": HeatmapResponse;
  "domain.HeatmapTile": HeatmapTile;
  "domain.HistogramBin": HistogramBin;
  "domain.InvestmentInput": ContractInvestmentInput;
  "domain.InvestmentResult": InvestmentResult;
  "domain.InvestmentScoreResult": InvestmentScoreResult;
  "domain.LTVSensitivityRow": LTVSensitivityRow;
  "domain.LandPriceComparison": LandPriceComparison;
  "domain.LandPriceStats": LandPriceStats;
  "domain.LandTransaction": LandTransaction;
  "domain.MonteCarloInput": MonteCarloInput;
  "domain.MonteCarloResult": MonteCarloResult;
  "domain.MultiExitRow": MultiExitRow;
  "domain.OwnershipComparisonResult": OwnershipComparisonResult;
  "domain.OwnershipScenario": OwnershipScenario;
  "domain.Percentiles": Percentiles;
  "domain.PopulationForecastResult": PopulationForecastResult;
  "domain.PopulationSnapshot": PopulationSnapshot;
  "domain.RadarPoint": RadarPoint;
  "domain.RateAdjustment": RateAdjustment;
  "domain.RenovationInput": RenovationInput;
  "domain.RenovationItem": RenovationItem;
  "domain.RenovationResult": RenovationResult;
  "domain.RentDeclineHint": RentDeclineHint;
  "domain.RentStatsResult": RentStatsResult;
  "domain.SalaryLossCarryoverResult": SalaryLossCarryoverResult;
  "domain.ScoreBreakdown": ScoreBreakdown;
  "domain.ScoreItem": ScoreItem;
  "domain.StationRidershipResult": StationRidershipResult;
  "domain.StressScenarioResult": StressScenarioResult;
  "domain.TaxSimRow": TaxSimRow;
  "domain.TaxSimulationResult": TaxSimulationResult;
  "domain.TheoreticalPriceResult": TheoreticalPriceResult;
  "domain.UrbanRisk": UrbanRisk;
  "domain.YearlyResult": YearlyResult;
  "domain.YieldScenario": YieldScenario;
  "domain.YieldScenarios": YieldScenarios;
  "domain.ZoningSummary": ZoningSummary;
  "mlit.Municipality": Municipality;
};

/** 全オブジェクト型の一括検証。不合格の型はスキーマ名付きでエラーに現れる */
export type ContractChecks = AssertAll<{
  [K in keyof HandwrittenBySchema]: CheckContract<HandwrittenBySchema[K], Schemas[K]>;
}>;

/** enum 由来のユニオン型はメンバー集合の完全一致を検証する */
export type UnionChecks = AssertAll<{
  buildingType: Equals<BuildingType, Schemas["domain.BuildingType"]>;
  criticalErrorStatus: Equals<CriticalError["status"], Schemas["domain.CriticalErrorStatus"]>;
  populationTrend: Equals<PopulationForecastResult["trend"], Schemas["domain.PopulationTrend"]>;
  ridershipDemandScore: Equals<RidershipDemandScore, Schemas["domain.RidershipDemandScore"]>;
  urbanRiskLevel: Equals<UrbanRiskLevel, Schemas["domain.UrbanRiskLevel"]>;
}>;

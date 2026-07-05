/**
 * API 契約テスト（コンパイル時のみ・ランタイムでは実行されない）
 *
 * 手書き型 (types/investment.ts) と swagger.json 由来の生成型
 * (types/api.generated.ts) のドリフトを `tsc --noEmit` で検出する。
 * バックエンドの Go 型を変更したら `make swagger` → `npm run generate:types`
 * を実行すると、このファイルがコンパイルエラーになって不整合を知らせる。
 *
 * 検証は2段構え:
 * 1. AssertSameKeys — フィールドの追加・削除・改名を検出
 *    （生成型は swag の制約で全フィールド optional のため、
 *      代入可能性チェックだけではフィールド欠落を検出できない）
 * 2. AssertAssignable — 同名フィールドの型不一致を検出
 *    （ネストした型も構造的に検証される）
 *
 * 対象: バックエンド由来の主要3型（Issue #811 PR-1）。PR-2 で全型へ拡大予定。
 */
import type { components } from "@/types/api.generated";
import type { InvestmentInput, InvestmentResult, LandPriceStats } from "@/types/investment";

type Schemas = components["schemas"];

/** キー集合が一致しない場合、差分キーを含むオブジェクト型に解決されてエラーになる */
type AssertSameKeys<Hand, Gen> = [
  Exclude<keyof Gen, keyof Hand>,
  Exclude<keyof Hand, keyof Gen>,
] extends [never, never]
  ? true
  : {
      missingInHandwritten: Exclude<keyof Gen, keyof Hand>;
      unknownToBackend: Exclude<keyof Hand, keyof Gen>;
    };

/** Hand が Gen に代入不可能な場合、制約違反としてエラーになる */
type AssertAssignable<Hand extends Gen, Gen> = Hand;

type Expect<T extends true> = T;

// ---- domain.InvestmentInput ----
/**
 * フロント固有フィールド（バックエンドの simulate 契約に含まれないもの）。
 * ここに追加する場合は investment.ts 側のコメントにも明記すること。
 * - stationMinutes: 理論価格推定 API (/land-prices/estimate) のクエリ用
 */
type FrontendOnlyInputFields = "stationMinutes";
type ContractInvestmentInput = Omit<InvestmentInput, FrontendOnlyInputFields>;

export type CheckInvestmentInputKeys = Expect<
  AssertSameKeys<ContractInvestmentInput, Schemas["domain.InvestmentInput"]>
>;
export type CheckInvestmentInputFields = AssertAssignable<
  ContractInvestmentInput,
  Schemas["domain.InvestmentInput"]
>;

// ---- domain.InvestmentResult ----
export type CheckInvestmentResultKeys = Expect<
  AssertSameKeys<InvestmentResult, Schemas["domain.InvestmentResult"]>
>;
export type CheckInvestmentResultFields = AssertAssignable<
  InvestmentResult,
  Schemas["domain.InvestmentResult"]
>;

// ---- domain.LandPriceStats ----
export type CheckLandPriceStatsKeys = Expect<
  AssertSameKeys<LandPriceStats, Schemas["domain.LandPriceStats"]>
>;
export type CheckLandPriceStatsFields = AssertAssignable<
  LandPriceStats,
  Schemas["domain.LandPriceStats"]
>;

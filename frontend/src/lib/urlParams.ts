/**
 * URL parameter encoding/decoding for sharing investment simulation conditions.
 * PII (address, property name) and geo data (area, city) are excluded.
 */

import type { InvestmentInput, SimulationMode } from "@/types/investment";
import { DEFAULT_INPUT } from "@/types/investment";

/** Parameters that are safe to include in the URL (no PII, no geo data) */
export interface UrlShareParams {
  mode: SimulationMode;
  // Quick mode
  totalPrice?: number; // 万円
  // Full mode extras
  landPrice?: number; // 万円
  buildingCost?: number; // 万円
  // Common
  rent?: number; // 万円/月 (stored as float with 4 decimal places)
  loanAmount?: number; // 万円
  loanRate?: number; // % (e.g. 1.5 for 1.5%)
  loanYears?: number;
  holdingYears?: number;
  vacancy?: number; // % (e.g. 5 for 5%)
  expenseRate?: number; // % (e.g. 20 for 20%)
}

function roundTo(n: number, decimals: number): number {
  const factor = Math.pow(10, decimals);
  return Math.round(n * factor) / factor;
}

/**
 * Encode simulation conditions into URLSearchParams.
 * Only includes values that differ from defaults to keep URLs short.
 */
export function encodeUrlParams(
  mode: SimulationMode,
  input: InvestmentInput,
  quickTotalPriceMan?: string
): URLSearchParams {
  const params = new URLSearchParams();

  params.set("mode", mode);

  if (mode === "quick") {
    const total = parseFloat(quickTotalPriceMan ?? "") || 0;
    if (total > 0) params.set("totalPrice", String(total));
  } else {
    // full mode
    const landMan = roundTo(input.landPrice / 10_000, 2);
    if (landMan !== roundTo(DEFAULT_INPUT.landPrice / 10_000, 2)) {
      params.set("landPrice", String(landMan));
    }
    const buildMan = roundTo(input.buildingCost / 10_000, 2);
    if (buildMan !== roundTo(DEFAULT_INPUT.buildingCost / 10_000, 2)) {
      params.set("buildingCost", String(buildMan));
    }
  }

  // rent (万円/月, 4 decimal places for precision)
  const rentMan = roundTo(input.monthlyRent / 10_000, 4);
  const defaultRentMan = roundTo(DEFAULT_INPUT.monthlyRent / 10_000, 4);
  if (rentMan !== defaultRentMan) {
    params.set("rent", String(rentMan));
  }

  // loanAmount (万円)
  const loanMan = roundTo(input.loanAmount / 10_000, 2);
  const defaultLoanMan = roundTo(DEFAULT_INPUT.loanAmount / 10_000, 2);
  if (loanMan !== defaultLoanMan) {
    params.set("loanAmount", String(loanMan));
  }

  // loanRate (%)
  const loanRatePct = roundTo(input.annualLoanRate * 100, 3);
  const defaultLoanRatePct = roundTo(DEFAULT_INPUT.annualLoanRate * 100, 3);
  if (loanRatePct !== defaultLoanRatePct) {
    params.set("loanRate", String(loanRatePct));
  }

  // loanYears
  if (input.loanYears !== DEFAULT_INPUT.loanYears) {
    params.set("loanYears", String(input.loanYears));
  }

  // holdingYears
  if (input.holdingYears !== DEFAULT_INPUT.holdingYears) {
    params.set("holdingYears", String(input.holdingYears));
  }

  // vacancyRate (%)
  const vacancyPct = roundTo(input.vacancyRate * 100, 1);
  const defaultVacancyPct = roundTo(DEFAULT_INPUT.vacancyRate * 100, 1);
  if (vacancyPct !== defaultVacancyPct) {
    params.set("vacancy", String(vacancyPct));
  }

  // expenseRate (%)
  const expensePct = roundTo(input.expenseRate * 100, 1);
  const defaultExpensePct = roundTo(DEFAULT_INPUT.expenseRate * 100, 1);
  if (expensePct !== defaultExpensePct) {
    params.set("expenseRate", String(expensePct));
  }

  return params;
}

export interface DecodedParams {
  mode: SimulationMode;
  input: Partial<InvestmentInput>;
  quickTotalPriceMan: string; // empty string if not set
}

/**
 * Decode URLSearchParams into initial state values for Dashboard/InvestmentForm.
 * Returns only the fields that are present in the URL.
 */
export function decodeUrlParams(params: URLSearchParams): DecodedParams {
  const rawMode = params.get("mode");
  const mode: SimulationMode = rawMode === "full" ? "full" : "quick";

  const input: Partial<InvestmentInput> = {};
  let quickTotalPriceMan = "";

  // totalPrice (quick mode)
  const totalPrice = params.get("totalPrice");
  if (totalPrice !== null) {
    const v = parseFloat(totalPrice);
    if (!isNaN(v) && v > 0) quickTotalPriceMan = String(v);
  }

  // landPrice (full mode)
  const landPrice = params.get("landPrice");
  if (landPrice !== null) {
    const v = parseFloat(landPrice);
    if (!isNaN(v) && v > 0) input.landPrice = v * 10_000;
  }

  // buildingCost (full mode)
  const buildingCost = params.get("buildingCost");
  if (buildingCost !== null) {
    const v = parseFloat(buildingCost);
    if (!isNaN(v) && v > 0) input.buildingCost = v * 10_000;
  }

  // rent
  const rent = params.get("rent");
  if (rent !== null) {
    const v = parseFloat(rent);
    if (!isNaN(v) && v > 0) input.monthlyRent = Math.round(v * 10_000);
  }

  // loanAmount
  const loanAmount = params.get("loanAmount");
  if (loanAmount !== null) {
    const v = parseFloat(loanAmount);
    if (!isNaN(v) && v >= 0) input.loanAmount = v * 10_000;
  }

  // loanRate
  const loanRate = params.get("loanRate");
  if (loanRate !== null) {
    const v = parseFloat(loanRate);
    if (!isNaN(v) && v >= 0) input.annualLoanRate = v / 100;
  }

  // loanYears
  const loanYears = params.get("loanYears");
  if (loanYears !== null) {
    const v = parseInt(loanYears, 10);
    if (!isNaN(v) && v >= 0) input.loanYears = v;
  }

  // holdingYears
  const holdingYears = params.get("holdingYears");
  if (holdingYears !== null) {
    const v = parseInt(holdingYears, 10);
    if (!isNaN(v) && v >= 0) input.holdingYears = v;
  }

  // vacancy
  const vacancy = params.get("vacancy");
  if (vacancy !== null) {
    const v = parseFloat(vacancy);
    if (!isNaN(v) && v >= 0) input.vacancyRate = v / 100;
  }

  // expenseRate
  const expenseRate = params.get("expenseRate");
  if (expenseRate !== null) {
    const v = parseFloat(expenseRate);
    if (!isNaN(v) && v >= 0) input.expenseRate = v / 100;
  }

  return { mode, input, quickTotalPriceMan };
}

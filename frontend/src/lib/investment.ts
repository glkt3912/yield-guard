/**
 * investment.ts — client-side port of backend/internal/domain/investment.go
 *
 * Used as offline fallback when the backend /api/investment/analyze endpoint
 * is unavailable. Results must match the backend calculation exactly.
 */

import type {
  InvestmentInput,
  InvestmentResult,
  YearlyResult,
  CriticalError,
  AcquisitionCostBreakdown,
  StressScenarioResult,
  YieldScenarios,
  YieldScenario,
  LTVSensitivityRow,
  BuildingType,
} from "@/types/investment";

// ─── Constants ────────────────────────────────────────────────────────────────

const LOAN_METHOD_EQUAL_PAYMENT = "equal-payment";
const LOAN_METHOD_EQUAL_PRINCIPAL = "equal-principal";
const DEPRECIATION_STRAIGHT_LINE = "straight-line";
const DEPRECIATION_DECLINING_BALANCE = "declining-balance";

/** 長期譲渡所得税率（5年超） */
const LONG_TERM_TRANSFER_TAX_RATE = 0.20315;
/** 短期譲渡所得税率（5年以下） */
const SHORT_TERM_TRANSFER_TAX_RATE = 0.3963;

// ─── Building useful life ─────────────────────────────────────────────────────

const BUILDING_USEFUL_LIFE_MAP: Record<BuildingType, number> = {
  木造: 22,
  "軽量鉄骨(3mm以下)": 19,
  "軽量鉄骨(4mm以下)": 27,
  重量鉄骨: 34,
  RC造: 47,
  SRC造: 47,
};

function getUsefulLife(buildingType: BuildingType): number {
  return BUILDING_USEFUL_LIFE_MAP[buildingType] ?? 22;
}

/**
 * calcResidualUsefulLife — 中古物件の簡便法耐用年数
 * 根拠: 耐用年数の適用等に関する取扱通達 1-5-3
 */
export function calcResidualUsefulLife(buildingType: BuildingType, buildingAge: number): number {
  const legal = getUsefulLife(buildingType);
  if (buildingAge <= 0) return legal;

  let residual: number;
  if (buildingAge >= legal) {
    residual = Math.floor(legal * 0.2);
  } else {
    residual = legal - buildingAge + Math.floor(buildingAge * 0.2);
  }
  return Math.max(residual, 2);
}

// ─── Defaults ─────────────────────────────────────────────────────────────────

function applyDefaults(input: InvestmentInput): InvestmentInput {
  const i = { ...input };
  if (i.miscExpenseRate == null || i.miscExpenseRate === 0) i.miscExpenseRate = 0.07;
  if (i.holdingYears == null || i.holdingYears === 0) i.holdingYears = 10;
  if (i.exitYieldTarget == null || i.exitYieldTarget === 0) i.exitYieldTarget = 0.06;
  if (i.yieldTarget == null || i.yieldTarget === 0) i.yieldTarget = 0.08;
  if (i.loanYears == null || i.loanYears === 0) i.loanYears = 35;
  if (!i.buildingType) i.buildingType = "木造";
  if (!i.loanMethod) i.loanMethod = LOAN_METHOD_EQUAL_PAYMENT;
  if (i.discountRate == null || i.discountRate === 0) i.discountRate = 0.05;
  if (!i.depreciationMethod) i.depreciationMethod = DEPRECIATION_STRAIGHT_LINE;
  if (!i.rateAdjustmentSchedule) i.rateAdjustmentSchedule = [];
  return i;
}

// ─── Loan calculations ────────────────────────────────────────────────────────

/**
 * calcMonthlyPayment — 元利均等返済の月次返済額
 */
export function calcMonthlyPayment(principal: number, annualRate: number, years: number): number {
  if (principal <= 0 || years <= 0) return 0;
  if (annualRate === 0) return principal / (years * 12);
  const r = annualRate / 12;
  const n = years * 12;
  return (principal * r * Math.pow(1 + r, n)) / (Math.pow(1 + r, n) - 1);
}

/**
 * calcYearlyLoanComponents — 元利均等返済: 1年分の利息・元金返済額
 */
function calcYearlyLoanComponents(
  balance: number,
  annualRate: number,
  monthlyPayment: number
): { interest: number; principal: number } {
  if (annualRate === 0) {
    if (monthlyPayment * 12 > balance) return { interest: 0, principal: balance };
    return { interest: 0, principal: monthlyPayment * 12 };
  }
  const r = annualRate / 12;
  let remaining = balance;
  let interest = 0;
  let principal = 0;
  for (let m = 0; m < 12; m++) {
    if (remaining <= 0) break;
    const monthInterest = remaining * r;
    let monthPrincipal = monthlyPayment - monthInterest;
    if (monthPrincipal > remaining) monthPrincipal = remaining;
    interest += monthInterest;
    principal += monthPrincipal;
    remaining -= monthPrincipal;
  }
  return { interest, principal };
}

/**
 * calcYearlyLoanComponentsEqualPrincipal — 元金均等返済: 1年分の利息・元金返済額
 */
function calcYearlyLoanComponentsEqualPrincipal(
  balance: number,
  annualRate: number,
  monthlyPrincipal: number
): { interest: number; principal: number } {
  const r = annualRate / 12;
  let remaining = balance;
  let interest = 0;
  let principal = 0;
  for (let m = 0; m < 12; m++) {
    if (remaining <= 0) break;
    const mp = Math.min(monthlyPrincipal, remaining);
    interest += remaining * r;
    principal += mp;
    remaining -= mp;
  }
  return { interest, principal };
}

/**
 * resolveRateForYear — スケジュールと基準金利から指定年の適用金利を返す
 */
function resolveRateForYear(
  baseRate: number,
  rateDelta: number,
  schedule: InvestmentInput["rateAdjustmentSchedule"],
  year: number
): number {
  let rate = baseRate;
  for (const adj of schedule) {
    if (year >= adj.afterYear) rate = adj.rate;
  }
  return rate + rateDelta;
}

// ─── CalcDSCR ─────────────────────────────────────────────────────────────────

export function calcDSCR(noi: number, annualDebtService: number): number {
  if (annualDebtService <= 0) return 0;
  return noi / annualDebtService;
}

// ─── Acquisition costs ────────────────────────────────────────────────────────

function calcBrokerageFee(price: number, multiplier: number): number {
  if (multiplier <= 0 || price <= 0) return 0;
  let base: number;
  if (price <= 2_000_000) {
    base = price * 0.05;
  } else if (price <= 4_000_000) {
    base = price * 0.04 + 20_000;
  } else {
    base = price * 0.03 + 60_000;
  }
  return Math.round(base * 1.1 * multiplier);
}

function calcStampDuty(price: number): number {
  if (price <= 100_000) return 200;
  if (price <= 500_000) return 400;
  if (price <= 1_000_000) return 1_000;
  if (price <= 5_000_000) return 2_000;
  if (price <= 10_000_000) return 10_000;
  if (price <= 50_000_000) return 20_000;
  if (price <= 100_000_000) return 60_000;
  if (price <= 500_000_000) return 100_000;
  if (price <= 1_000_000_000) return 200_000;
  return 600_000;
}

function calcRegistrationTax(
  landAssessed: number,
  buildingAssessed: number,
  loanAmount: number,
  isNewBuilding: boolean
): number {
  const landTransfer = landAssessed * 0.02;
  const buildingTransfer = isNewBuilding ? buildingAssessed * 0.0015 : buildingAssessed * 0.02;
  const mortgage = loanAmount * 0.004;
  return Math.round(landTransfer + buildingTransfer + mortgage);
}

function calcRealEstateAcquisitionTax(landAssessed: number, buildingAssessed: number): number {
  const taxRate = 0.03;
  const landTax = landAssessed * 0.5 * taxRate;
  const buildingTax = buildingAssessed * taxRate;
  return Math.round(landTax + buildingTax);
}

function calcAcquisitionCosts(
  landPrice: number,
  buildingCost: number,
  loanAmount: number
): AcquisitionCostBreakdown {
  const totalPrice = landPrice + buildingCost;
  const brokerage = calcBrokerageFee(totalPrice, 1.0);
  const stamp = calcStampDuty(totalPrice);

  const assessedLand = landPrice * 0.7;
  const assessedBuilding = buildingCost * 0.6;

  const regTax = calcRegistrationTax(assessedLand, assessedBuilding, loanAmount, false);
  const acqTax = calcRealEstateAcquisitionTax(assessedLand, assessedBuilding);

  const total = brokerage + stamp + regTax + acqTax;
  return {
    brokerageFee: brokerage,
    stampDuty: stamp,
    registrationTax: regTax,
    realEstateAcquisitionTax: acqTax,
    propertyTaxProration: 0,
    total,
  };
}

// ─── Required-for-target ──────────────────────────────────────────────────────

function calcRequiredForTarget(
  input: InvestmentInput,
  totalInvestment: number
): { requiredRent: number; costReduction: number } {
  const target = input.yieldTarget;
  const requiredAnnualRent = totalInvestment * target;
  const requiredRent = requiredAnnualRent / 12;

  const currentAnnualRent = input.monthlyRent * 12;
  const requiredTotalInvestment = currentAnnualRent / target;

  const excess = totalInvestment - requiredTotalInvestment;
  const costReduction = excess > 0 ? excess : 0;
  return { requiredRent, costReduction };
}

// ─── Exit strategy ────────────────────────────────────────────────────────────

function calcExit(
  input: InvestmentInput,
  yearly: YearlyResult[],
  accumulatedDepreciation: number,
  miscExpenses: number
): {
  salePrice: number;
  capitalGain: number;
  transferTax: number;
  netProceeds: number;
  totalEquity: number;
} {
  if (yearly.length === 0 || input.holdingYears <= 0 || input.exitYieldTarget <= 0) {
    return { salePrice: 0, capitalGain: 0, transferTax: 0, netProceeds: 0, totalEquity: 0 };
  }

  const holdIdx = Math.min(input.holdingYears - 1, yearly.length - 1);
  const exitYear = yearly[holdIdx];

  const noi = exitYear.annualRent - exitYear.annualExpenses;
  const salePrice = noi / input.exitYieldTarget;

  const sellExpenses = (salePrice * 0.03 + 60_000) * 1.1;

  const bookValueBuilding = Math.max(input.buildingCost - accumulatedDepreciation, 0);
  const acquisitionCost = input.landPrice + bookValueBuilding + miscExpenses;

  const capitalGain = salePrice - sellExpenses - acquisitionCost;

  let transferTax = 0;
  if (capitalGain > 0) {
    const taxRate =
      input.holdingYears > 5 ? LONG_TERM_TRANSFER_TAX_RATE : SHORT_TERM_TRANSFER_TAX_RATE;
    transferTax = capitalGain * taxRate;
  }

  const netProceeds = salePrice - sellExpenses - transferTax - exitYear.remainingLoanBalance;
  const totalEquity = netProceeds + exitYear.cumulativeCashFlow;

  return { salePrice, capitalGain, transferTax, netProceeds, totalEquity };
}

// ─── Critical errors ──────────────────────────────────────────────────────────

function calcCriticalErrors(
  input: InvestmentInput,
  deadCrossYear: number,
  usefulLife: number
): CriticalError[] {
  const errors: CriticalError[] = [];

  const totalPurchase = input.landPrice + input.buildingCost;
  const residualLife = calcResidualUsefulLife(input.buildingType, input.buildingAge);
  const buildingAppraisedValue = input.buildingCost * (residualLife / usefulLife);
  const appraisedValue = input.landPrice + buildingAppraisedValue;

  if (totalPurchase > 0 && appraisedValue < totalPurchase * 0.5) {
    errors.push({
      code: "LAND_VALUE_GUARD",
      status: "REJECT",
      message: `積算評価額（${Math.round(appraisedValue / 10000)}万円）が購入総額（${Math.round(totalPurchase / 10000)}万円）の50%未満です。銀行の担保評価が低くなり次の買主がローンを組めない可能性があります。`,
    });
  }

  if (deadCrossYear > 0 && deadCrossYear <= 10) {
    errors.push({
      code: "DEADCROSS_EARLY",
      status: "REJECT",
      message: `${deadCrossYear}年目にデッドクロスが発生します。帳簿上は黒字でもキャッシュ不足に陥るリスクがあります。`,
    });
  }

  return errors;
}

// ─── Yield scenarios ──────────────────────────────────────────────────────────

function calcYieldScenarios(input: InvestmentInput, totalInvestment: number): YieldScenarios {
  const grossYield = totalInvestment > 0 ? (input.monthlyRent * 12) / totalInvestment : 0;

  const calcScenario = (vacancyMultiplier: number): YieldScenario => {
    const effectiveVacancy = Math.min(input.vacancyRate * vacancyMultiplier, 0.99);
    const annualRent = input.monthlyRent * 12 * (1 - effectiveVacancy);
    return { annualRent, grossYield };
  };

  return {
    optimistic: calcScenario(0.5),
    standard: calcScenario(1.0),
    pessimistic: calcScenario(1.5),
  };
}

// ─── Stress scenarios ─────────────────────────────────────────────────────────

function calcStressScenario(
  base: InvestmentInput,
  label: string,
  rateDelta: number,
  vacDelta: number
): StressScenarioResult {
  const inInput = applyDefaults(base);

  let effectiveVacancy = inInput.vacancyRate + vacDelta;
  if (effectiveVacancy > 1) effectiveVacancy = 1;

  // 初年度賃料（空室率調整済み）— 賃料下落はループ内で年次適用
  const annualRent = inInput.monthlyRent * 12 * (1 - effectiveVacancy);

  const initRate = resolveRateForYear(
    inInput.annualLoanRate,
    rateDelta,
    inInput.rateAdjustmentSchedule,
    1
  );

  let monthlyPayment = 0;
  let monthlyPrincipalStress = 0;

  if (inInput.loanMethod === LOAN_METHOD_EQUAL_PRINCIPAL && inInput.loanYears > 0) {
    const totalMonths = inInput.loanYears * 12;
    monthlyPrincipalStress = inInput.loanAmount / totalMonths;
  } else {
    monthlyPayment = calcMonthlyPayment(inInput.loanAmount, initRate, inInput.loanYears);
  }

  // DSCR は各年返済額から算出した保有期間内最悪値（変動金利上昇ケースで正確なリスク評価を行うため）
  let holdingYears = inInput.holdingYears;
  if (holdingYears <= 0) holdingYears = 10;

  let totalCF = 0;
  let breakEvenYear = -1;
  let cumCF = 0;
  let remainingBalance = inInput.loanAmount;
  let currentRate = initRate;
  let curMonthlyPayment = monthlyPayment;
  let minDSCR = Infinity;
  let hasLoanYear = false;

  for (let y = 1; y <= holdingYears; y++) {
    if (y > 1 && inInput.rateAdjustmentSchedule.length > 0) {
      const newRate = resolveRateForYear(
        inInput.annualLoanRate,
        rateDelta,
        inInput.rateAdjustmentSchedule,
        y
      );
      if (newRate !== currentRate && remainingBalance > 0 && y <= inInput.loanYears) {
        if (inInput.loanMethod !== LOAN_METHOD_EQUAL_PRINCIPAL) {
          const remainingYears = inInput.loanYears - (y - 1);
          curMonthlyPayment = calcMonthlyPayment(remainingBalance, newRate, remainingYears);
        }
        currentRate = newRate;
      }
    }

    let yearLoan = 0;
    let yearInterest = 0;
    if (remainingBalance > 0 && y <= inInput.loanYears) {
      if (inInput.loanMethod === LOAN_METHOD_EQUAL_PRINCIPAL) {
        const { interest: yi, principal: yp } = calcYearlyLoanComponentsEqualPrincipal(
          remainingBalance,
          currentRate,
          monthlyPrincipalStress
        );
        yearLoan = yi + yp;
        yearInterest = yi;
        remainingBalance -= yp;
      } else {
        const { interest: annInterest, principal: annPrincipal } = calcYearlyLoanComponents(
          remainingBalance,
          currentRate,
          curMonthlyPayment
        );
        yearLoan = curMonthlyPayment * 12;
        yearInterest = annInterest;
        remainingBalance -= annPrincipal;
      }
      if (remainingBalance < 0) remainingBalance = 0;
    }

    // 賃料下落率を年次適用（Analyze() の declineFactor と同じロジック、1-indexed なので y-1 を指数に使用）
    const declineFactor = Math.pow(1 - inInput.rentDeclineRate, y - 1);
    const yearRent = annualRent * declineFactor;
    const yearExpenses = yearRent * inInput.expenseRate + inInput.annualPropertyTax;
    const yearNOI = yearRent - yearExpenses;

    if (yearLoan > 0) {
      hasLoanYear = true;
      const yearDSCR = yearNOI / yearLoan;
      if (yearDSCR < minDSCR) minDSCR = yearDSCR;
    }

    const cf = yearNOI - yearLoan;
    // 減価償却は省略した保守的近似（簡略ストレス計算のため過大に税を見積もる）
    const taxableIncome = yearNOI - yearInterest;
    const incomeTax = taxableIncome > 0 ? taxableIncome * inInput.incomeTaxRate : 0;
    const afterTaxCF = cf - incomeTax;
    totalCF += afterTaxCF;
    cumCF += afterTaxCF;
    if (breakEvenYear === -1 && cumCF > 0) breakEvenYear = y;
  }

  const dscr = hasLoanYear ? minDSCR : 0;

  let isSafe: boolean;
  if (!hasLoanYear) {
    // 保有期間内に返済が発生しない場合（無借金物件等）はブレークイーン達成のみで安全と判定
    isSafe = breakEvenYear !== -1 && breakEvenYear <= holdingYears;
  } else {
    isSafe = dscr >= 1.0 && breakEvenYear !== -1 && breakEvenYear <= holdingYears;
  }

  return {
    label,
    interestRateDelta: rateDelta,
    vacancyRateDelta: vacDelta,
    totalCashFlow: totalCF,
    dscr,
    breakEvenYear,
    isSafe,
  };
}

// ─── LTV sensitivity ──────────────────────────────────────────────────────────

function calcLTVSensitivity(input: InvestmentInput): LTVSensitivityRow[] {
  const ltvRange = [0.5, 0.6, 0.7, 0.8, 0.9];
  const miscExpenses = (input.landPrice + input.buildingCost) * input.miscExpenseRate;
  const totalInvestment = input.landPrice + input.buildingCost + miscExpenses;
  if (totalInvestment <= 0) return [];

  const effectiveVacancy = Math.min(input.vacancyRate, 0.99);
  const annualRent = input.monthlyRent * 12 * (1 - effectiveVacancy);
  const annualExpenses = annualRent * input.expenseRate + input.annualPropertyTax;
  const noi = annualRent - annualExpenses;

  return ltvRange.map((ltv) => {
    const loanAmount = totalInvestment * ltv;
    const equity = totalInvestment * (1 - ltv);

    let annualDebtService: number;
    if (input.loanMethod === LOAN_METHOD_EQUAL_PRINCIPAL && input.loanYears > 0) {
      const totalMonths = input.loanYears * 12;
      const monthlyPrincipal = loanAmount / totalMonths;
      const { interest: yi, principal: yp } = calcYearlyLoanComponentsEqualPrincipal(
        loanAmount,
        input.annualLoanRate,
        monthlyPrincipal
      );
      annualDebtService = yi + yp;
    } else {
      const mp = calcMonthlyPayment(loanAmount, input.annualLoanRate, input.loanYears);
      annualDebtService = mp * 12;
    }

    const dscr = calcDSCR(noi, annualDebtService);
    const annualCF = noi - annualDebtService;
    const cfYield = annualCF / totalInvestment;

    return { ltv, equity, loanAmount, dscr, annualCF, cfYield };
  });
}

// ─── NPV / IRR ────────────────────────────────────────────────────────────────

function calcNPV(
  cfs: number[],
  terminalValue: number,
  discountRate: number,
  initialInvestment: number
): number {
  let pv = 0;
  for (let t = 0; t < cfs.length; t++) {
    pv += cfs[t] / Math.pow(1 + discountRate, t + 1);
  }
  if (cfs.length > 0) {
    pv += terminalValue / Math.pow(1 + discountRate, cfs.length);
  }
  return pv - initialInvestment;
}

function calcIRR(cfs: number[], terminalValue: number, initialInvestment: number): number | null {
  const LO = -0.5;
  const HI = 2.0;
  const MAX_ITER = 200;
  const TOL = 1.0;

  const npvLo = calcNPV(cfs, terminalValue, LO, initialInvestment);
  const npvHi = calcNPV(cfs, terminalValue, HI, initialInvestment);
  if (npvLo * npvHi > 0) return null;

  let low = LO,
    high = HI;
  let currentNpvLo = npvLo;
  for (let i = 0; i < MAX_ITER; i++) {
    const mid = (low + high) / 2;
    const npvMid = calcNPV(cfs, terminalValue, mid, initialInvestment);
    if (Math.abs(npvMid) < TOL) return mid;
    if (npvMid * currentNpvLo < 0) {
      high = mid;
    } else {
      low = mid;
      currentNpvLo = npvMid;
    }
  }
  return null;
}

// ─── Terminal value with price decline ────────────────────────────────────────

function calcTerminalValueWithDecline(
  input: InvestmentInput,
  yearly: YearlyResult[],
  adjustedSalePrice: number,
  accumulatedDepreciation: number,
  miscExpenses: number
): number {
  const holdIdx = Math.min(input.holdingYears - 1, yearly.length - 1);
  const exitYear = yearly[holdIdx];
  const sellExpenses = (adjustedSalePrice * 0.03 + 60_000) * 1.1;
  const bookValueBuilding = Math.max(input.buildingCost - accumulatedDepreciation, 0);
  const acquisitionCost = input.landPrice + bookValueBuilding + miscExpenses;
  const capGain = adjustedSalePrice - sellExpenses - acquisitionCost;
  let transferTax = 0;
  if (capGain > 0) {
    const taxRate =
      input.holdingYears > 5 ? LONG_TERM_TRANSFER_TAX_RATE : SHORT_TERM_TRANSFER_TAX_RATE;
    transferTax = capGain * taxRate;
  }
  return adjustedSalePrice - sellExpenses - transferTax - exitYear.remainingLoanBalance;
}

// ─── Main Analyze function ────────────────────────────────────────────────────

/**
 * analyze — client-side equivalent of backend Analyze()
 *
 * Performs the full investment simulation without any network calls.
 * Used as an offline fallback.
 */
export function analyze(inputRaw: InvestmentInput): InvestmentResult {
  const input = applyDefaults(inputRaw);

  const effectiveVacancy = Math.min(input.vacancyRate + input.vacancyRateDelta, 0.99);

  const miscExpenses = (input.landPrice + input.buildingCost) * input.miscExpenseRate;
  const totalInvestment = input.landPrice + input.buildingCost + miscExpenses;

  const annualRent = input.monthlyRent * 12 * (1 - effectiveVacancy);
  const grossYield = totalInvestment > 0 ? (input.monthlyRent * 12) / totalInvestment : 0;

  const annualExpenses = annualRent * input.expenseRate;
  const netYield = totalInvestment > 0 ? (annualRent - annualExpenses) / totalInvestment : 0;

  const { requiredRent, costReduction } = calcRequiredForTarget(input, totalInvestment);

  const usefulLife = calcResidualUsefulLife(input.buildingType, input.buildingAge);
  const annualDepreciation = input.buildingCost / usefulLife;

  let currentRate = resolveRateForYear(
    input.annualLoanRate,
    input.loanRateDelta,
    input.rateAdjustmentSchedule,
    1
  );
  let monthlyPayment = 0;
  let monthlyPrincipalFixed = 0;

  if (input.loanMethod === LOAN_METHOD_EQUAL_PRINCIPAL && input.loanYears > 0) {
    monthlyPrincipalFixed = input.loanAmount / (input.loanYears * 12);
  } else {
    monthlyPayment = calcMonthlyPayment(input.loanAmount, currentRate, input.loanYears);
  }

  let bookValue = 0;
  let decliningRate = 0;
  if (input.depreciationMethod === DEPRECIATION_DECLINING_BALANCE) {
    bookValue = input.buildingCost;
    decliningRate = 1.5 / usefulLife;
  }

  const years = Math.max(input.loanYears, input.holdingYears, 35);
  const yearlyResults: YearlyResult[] = [];
  let remainingBalance = input.loanAmount;
  let cumulativeCF = 0;
  let deadCrossYear = -1;
  let accumulatedDepreciation = 0;

  for (let y = 0; y < years; y++) {
    const year = y + 1;

    if (year > 1 && input.rateAdjustmentSchedule.length > 0) {
      const newRate = resolveRateForYear(
        input.annualLoanRate,
        input.loanRateDelta,
        input.rateAdjustmentSchedule,
        year
      );
      if (newRate !== currentRate && remainingBalance > 0 && year <= input.loanYears) {
        if (input.loanMethod !== LOAN_METHOD_EQUAL_PRINCIPAL) {
          const remainingYears = input.loanYears - y;
          monthlyPayment = calcMonthlyPayment(remainingBalance, newRate, remainingYears);
        }
        currentRate = newRate;
      }
    }

    let annualInterest = 0;
    let annualPrincipal = 0;
    let annualLoanPayment = 0;

    if (remainingBalance > 0 && year <= input.loanYears) {
      if (input.loanMethod === LOAN_METHOD_EQUAL_PRINCIPAL) {
        const { interest, principal } = calcYearlyLoanComponentsEqualPrincipal(
          remainingBalance,
          currentRate,
          monthlyPrincipalFixed
        );
        annualInterest = interest;
        annualPrincipal = principal;
        annualLoanPayment = annualInterest + annualPrincipal;
      } else {
        const { interest, principal } = calcYearlyLoanComponents(
          remainingBalance,
          currentRate,
          monthlyPayment
        );
        annualInterest = interest;
        annualPrincipal = principal;
        annualLoanPayment = monthlyPayment * 12;
      }
      remainingBalance -= annualPrincipal;
      if (remainingBalance < 0) remainingBalance = 0;
    }

    const declineFactor = Math.pow(1 - input.rentDeclineRate, y);
    const yearAnnualRent = annualRent * declineFactor;
    const yearExpenses = yearAnnualRent * input.expenseRate + input.annualPropertyTax;

    let yearDepreciation = 0;
    if (input.depreciationMethod === DEPRECIATION_DECLINING_BALANCE) {
      if (bookValue > 1.0) {
        yearDepreciation = bookValue * decliningRate;
        if (bookValue - yearDepreciation < 1.0) {
          yearDepreciation = bookValue - 1.0;
        }
        bookValue -= yearDepreciation;
      }
    } else {
      if (year <= usefulLife) {
        yearDepreciation = annualDepreciation;
      }
    }
    accumulatedDepreciation += yearDepreciation;

    const taxableIncome = yearAnnualRent - annualInterest - yearDepreciation - yearExpenses;
    const incomeTax = taxableIncome > 0 ? taxableIncome * input.incomeTaxRate : 0;

    const cashFlow = yearAnnualRent - annualLoanPayment - yearExpenses;
    const afterTaxCF = cashFlow - incomeTax;
    cumulativeCF += afterTaxCF;

    const inDeadCrossZone =
      input.buildingCost > 0 && annualPrincipal > 0 && annualPrincipal > yearDepreciation;

    let isDeadCrossYear = false;
    if (deadCrossYear === -1 && inDeadCrossZone) {
      deadCrossYear = year;
      isDeadCrossYear = true;
    }

    yearlyResults.push({
      year,
      annualRent: yearAnnualRent,
      annualLoanPayment,
      annualInterest,
      annualPrincipal,
      annualDepreciation: yearDepreciation,
      annualExpenses: yearExpenses,
      taxableIncome,
      incomeTax,
      cashFlow,
      afterTaxCashFlow: afterTaxCF,
      remainingLoanBalance: remainingBalance,
      cumulativeCashFlow: cumulativeCF,
      isDeadCrossYear,
      isInDeadCrossZone: inDeadCrossZone,
      effectiveRate: currentRate,
    });
  }

  // DSCR: 1年目の NOI / 年間返済額
  let dscr = 0;
  if (yearlyResults.length > 0) {
    const noi = yearlyResults[0].annualRent - yearlyResults[0].annualExpenses;
    dscr = calcDSCR(noi, yearlyResults[0].annualLoanPayment);
  }

  const { salePrice, capitalGain, transferTax, netProceeds, totalEquity } = calcExit(
    input,
    yearlyResults,
    accumulatedDepreciation,
    miscExpenses
  );

  const criticalErrors = calcCriticalErrors(input, deadCrossYear, usefulLife);

  const defaultScenarios: { label: string; rateDelta: number; vacDelta: number }[] = [
    { label: "ベースライン", rateDelta: 0, vacDelta: 0 },
    { label: "金利+1%", rateDelta: 0.01, vacDelta: 0 },
    { label: "金利+2%", rateDelta: 0.02, vacDelta: 0 },
    { label: "空室+10%", rateDelta: 0, vacDelta: 0.1 },
    { label: "空室+20%", rateDelta: 0, vacDelta: 0.2 },
    { label: "複合ストレス", rateDelta: 0.02, vacDelta: 0.1 },
  ];

  const stressScenarios: StressScenarioResult[] = defaultScenarios.map((sc) =>
    calcStressScenario(input, sc.label, sc.rateDelta, sc.vacDelta)
  );

  if (input.loanRateDelta !== 0 || input.vacancyRateDelta !== 0) {
    stressScenarios.push(
      calcStressScenario(input, "カスタム", input.loanRateDelta, input.vacancyRateDelta)
    );
  }

  const acquisitionCosts = calcAcquisitionCosts(
    input.landPrice,
    input.buildingCost,
    input.loanAmount
  );
  const yieldScenarios = calcYieldScenarios(input, totalInvestment);
  const ltvSensitivity = calcLTVSensitivity(input);

  // IRR / NPV
  const equity = totalInvestment - input.loanAmount;
  let irr: number | null = null;
  let npv = 0;
  if (equity > 0 && input.holdingYears > 0) {
    const irrCFs: number[] = [];
    for (let i = 0; i < input.holdingYears && i < yearlyResults.length; i++) {
      irrCFs.push(yearlyResults[i].afterTaxCashFlow);
    }

    let irrTerminalValue = netProceeds;
    const priceDeclineRate = input.priceDeclineRate ?? 0;
    if (priceDeclineRate > 0 && input.holdingYears > 0) {
      const decayFactor = Math.pow(1 - priceDeclineRate, input.holdingYears);
      const adjustedSalePrice = salePrice * decayFactor;
      irrTerminalValue = calcTerminalValueWithDecline(
        input,
        yearlyResults,
        adjustedSalePrice,
        accumulatedDepreciation,
        miscExpenses
      );
    }

    const discountRate = input.discountRate ?? 0.05;
    npv = calcNPV(irrCFs, irrTerminalValue, discountRate, equity);
    irr = calcIRR(irrCFs, irrTerminalValue, equity);
  }

  return {
    totalInvestment,
    miscExpenses,
    grossYield,
    netYield,
    isAboveYieldTarget: grossYield >= input.yieldTarget,
    yieldTarget: input.yieldTarget,
    requiredCostReduction: costReduction,
    requiredMonthlyRent: requiredRent,
    deadCrossYear,
    yearlyResults,
    criticalErrors,
    acquisitionCosts,
    exitSalePrice: salePrice,
    exitCapitalGain: capitalGain,
    exitTransferTax: transferTax,
    exitNetProceeds: netProceeds,
    exitTotalEquity: totalEquity,
    stressScenarios,
    yieldScenarios,
    dscr,
    ltvSensitivity,
    irr,
    npv,
  };
}

import { describe, it, expect } from "vitest";
import { analyze, calcMonthlyPayment, calcResidualUsefulLife } from "@/lib/investment";
import type { InvestmentInput } from "@/types/investment";

const EPSILON = 0.01; // 1円以内の誤差を許容

function approxEqual(a: number, b: number, eps = EPSILON): boolean {
  return Math.abs(a - b) <= eps;
}

// ─── Base test input (matches backend TestAnalyze_GrossYield) ──────────────

const BASE_INPUT: InvestmentInput = {
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
  rentDeclineRate: 0,
  yieldTarget: 0.08,
  loanMethod: "equal-payment",
  rateAdjustmentSchedule: [],
  discountRate: 0.05,
  priceDeclineRate: 0,
  depreciationMethod: "straight-line",
};

// ─── calcMonthlyPayment ────────────────────────────────────────────────────────

describe("calcMonthlyPayment", () => {
  it("1000万 年利1.5% 35年 ≈ 30,607円", () => {
    const got = calcMonthlyPayment(10_000_000, 0.015, 35);
    expect(approxEqual(got, 30_607, 500)).toBe(true);
  });

  it("3000万 年利2.0% 30年 ≈ 110,879円", () => {
    const got = calcMonthlyPayment(30_000_000, 0.020, 30);
    expect(approxEqual(got, 110_879, 500)).toBe(true);
  });

  it("金利ゼロ: 元金のみ均等返済", () => {
    const got = calcMonthlyPayment(12_000_000, 0, 10);
    expect(approxEqual(got, 100_000, 1)).toBe(true);
  });

  it("元金ゼロ → 0を返す", () => {
    expect(calcMonthlyPayment(0, 0.015, 35)).toBe(0);
  });

  it("期間ゼロ → 0を返す", () => {
    expect(calcMonthlyPayment(10_000_000, 0.015, 0)).toBe(0);
  });
});

// ─── calcResidualUsefulLife ───────────────────────────────────────────────────

describe("calcResidualUsefulLife", () => {
  it("新築木造 → 法定耐用年数22年", () => {
    expect(calcResidualUsefulLife("木造", 0)).toBe(22);
  });

  it("築10年木造: (22-10) + floor(10*0.2) = 14年", () => {
    expect(calcResidualUsefulLife("木造", 10)).toBe(14);
  });

  it("築30年木造 (法定超過): floor(22*0.2)=4年", () => {
    expect(calcResidualUsefulLife("木造", 30)).toBe(4);
  });

  it("築100年木造 (大幅超過): floor(22*0.2)=4年", () => {
    // 法定耐用年数を大幅超過した場合も floor(22*0.2)=4年
    expect(calcResidualUsefulLife("木造", 100)).toBe(4);
  });

  it("新築RC造 → 47年", () => {
    expect(calcResidualUsefulLife("RC造", 0)).toBe(47);
  });

  it("築50年RC造 (法定超過): floor(47*0.2)=9年", () => {
    expect(calcResidualUsefulLife("RC造", 50)).toBe(9);
  });
});

// ─── analyze: 総投資額と表面利回り ────────────────────────────────────────────

describe("analyze - TotalInvestment and GrossYield", () => {
  it("総投資額 = 土地 + 建物 + 諸経費(7%)", () => {
    const result = analyze(BASE_INPUT);
    // 5,000,000 + 10,000,000 + 15,000,000*0.07 = 16,050,000
    const wantTotal = 5_000_000 + 10_000_000 + 15_000_000 * 0.07;
    expect(approxEqual(result.totalInvestment, wantTotal, 1)).toBe(true);
  });

  it("表面利回り = (月額賃料*12) / 総投資額", () => {
    const result = analyze(BASE_INPUT);
    const wantTotal = 5_000_000 + 10_000_000 + 15_000_000 * 0.07;
    const wantGross = (120_000 * 12) / wantTotal;
    expect(approxEqual(result.grossYield, wantGross, 0.0001)).toBe(true);
  });
});

// ─── analyze: 8%判定 ──────────────────────────────────────────────────────────

describe("analyze - isAboveYieldTarget", () => {
  it("高賃料(20万) → 8%超でtrueになる", () => {
    const r = analyze({ ...BASE_INPUT, monthlyRent: 200_000 });
    expect(r.isAboveYieldTarget).toBe(true);
  });

  it("低賃料(5万) → 8%未満でfalseになる", () => {
    const r = analyze({ ...BASE_INPUT, monthlyRent: 50_000 });
    expect(r.isAboveYieldTarget).toBe(false);
  });
});

// ─── analyze: dead cross ──────────────────────────────────────────────────────

describe("analyze - dead cross detection", () => {
  it("新築木造、短期ローン → デッドクロス発生（元金 > 減価償却）", () => {
    // 築0年木造: 耐用年数22年、短いローンだと元金返済が早く depreciation を超える
    const r = analyze({ ...BASE_INPUT, loanYears: 10, loanAmount: 10_000_000 });
    // deadCrossYear は -1 でないはず（元金返済がすぐ減価償却を超える）
    expect(r.deadCrossYear).toBeGreaterThan(-1);
  });

  it("建物費用ゼロ → デッドクロスなし(-1)", () => {
    const r = analyze({ ...BASE_INPUT, buildingCost: 0, loanAmount: 5_000_000 });
    expect(r.deadCrossYear).toBe(-1);
  });
});

// ─── analyze: yearly results length ──────────────────────────────────────────

describe("analyze - yearlyResults length", () => {
  it("最低35年分の結果を返す", () => {
    const r = analyze(BASE_INPUT);
    expect(r.yearlyResults.length).toBeGreaterThanOrEqual(35);
  });

  it("LoanYears=40の場合は40年分以上返す", () => {
    const r = analyze({ ...BASE_INPUT, loanYears: 40 });
    expect(r.yearlyResults.length).toBeGreaterThanOrEqual(40);
  });
});

// ─── analyze: remaining balance ───────────────────────────────────────────────

describe("analyze - remaining loan balance", () => {
  it("ローン完済後の残高は0", () => {
    const r = analyze(BASE_INPUT);
    // 35年後（インデックス34）は残高0であるべき
    const lastLoanYear = r.yearlyResults[BASE_INPUT.loanYears - 1];
    expect(lastLoanYear.remainingLoanBalance).toBeLessThanOrEqual(1); // 1円未満の丸め誤差を許容
  });
});

// ─── analyze: stress scenarios ────────────────────────────────────────────────

describe("analyze - stress scenarios", () => {
  it("デフォルトで6つのストレスシナリオを返す", () => {
    const r = analyze(BASE_INPUT);
    expect(r.stressScenarios.length).toBe(6);
  });

  it("カスタムストレスがある場合は7つ返す", () => {
    const r = analyze({ ...BASE_INPUT, loanRateDelta: 0.01 });
    expect(r.stressScenarios.length).toBe(7);
  });

  it("ベースラインシナリオのDSCRは正の値", () => {
    const r = analyze(BASE_INPUT);
    const baseline = r.stressScenarios.find((s) => s.label === "ベースライン");
    expect(baseline).toBeDefined();
    expect(baseline!.dscr).toBeGreaterThan(0);
  });

  it("変動金利で後年金利が急上昇するとき最悪年DSCRを返す（初年度より低い）", () => {
    // 1-4年: 1.5%、5年目以降: 4.0% に急上昇するスケジュール
    const input: InvestmentInput = {
      ...BASE_INPUT,
      annualLoanRate: 0.015,
      loanAmount: 18_000_000,
      loanYears: 25,
      rateAdjustmentSchedule: [{ afterYear: 5, rate: 0.04 }],
    };

    // year1 相当の DSCR を手計算（参照値）
    const annualRent = input.monthlyRent * 12 * (1 - input.vacancyRate);
    const annualExpenses = annualRent * input.expenseRate + input.annualPropertyTax;
    const noi = annualRent - annualExpenses;
    const mr = input.annualLoanRate / 12;
    const n = input.loanYears * 12;
    const monthlyY1 = (input.loanAmount * mr * Math.pow(1 + mr, n)) / (Math.pow(1 + mr, n) - 1);
    const dscrYear1 = noi / (monthlyY1 * 12);

    const baseline = analyze(input).stressScenarios.find((s) => s.label === "ベースライン")!;

    // 最悪年 DSCR は初年度ベースより低い（5年目以降の高金利を反映）
    expect(baseline.dscr).toBeLessThan(dscrYear1);
    expect(baseline.dscr).toBeGreaterThan(0);
  });

  it("変動金利上昇で初年度DSCR>=1.0でもisSafeがfalseになる", () => {
    // 初期金利 0.5%（DSCR >= 1.0）→ 5年目に 5.0% 急上昇（最悪年 DSCR < 1.0）
    const input: InvestmentInput = {
      ...BASE_INPUT,
      monthlyRent: 90_000,
      vacancyRate: 0.05,
      loanAmount: 16_000_000,
      annualLoanRate: 0.005,
      loanYears: 25,
      expenseRate: 0.10,
      holdingYears: 10,
      rateAdjustmentSchedule: [{ afterYear: 5, rate: 0.05 }],
    };

    const baseline = analyze(input).stressScenarios.find((s) => s.label === "ベースライン")!;

    // 最悪年 DSCR < 1.0 → isSafe = false（旧実装では初年度 DSCR >= 1.0 で true になっていたバグ）
    expect(baseline.dscr).toBeLessThan(1.0);
    expect(baseline.isSafe).toBe(false);
  });
});

// ─── analyze: yield scenarios ────────────────────────────────────────────────

describe("analyze - yield scenarios", () => {
  it("楽観シナリオの年間賃料 > 標準 > 悲観", () => {
    const r = analyze(BASE_INPUT);
    expect(r.yieldScenarios.optimistic.annualRent).toBeGreaterThan(r.yieldScenarios.standard.annualRent);
    expect(r.yieldScenarios.standard.annualRent).toBeGreaterThan(r.yieldScenarios.pessimistic.annualRent);
  });

  it("全シナリオで表面利回りは同じ（空室率に依存しない）", () => {
    const r = analyze(BASE_INPUT);
    expect(r.yieldScenarios.optimistic.grossYield).toBe(r.yieldScenarios.standard.grossYield);
    expect(r.yieldScenarios.standard.grossYield).toBe(r.yieldScenarios.pessimistic.grossYield);
  });
});

// ─── analyze: exit strategy ──────────────────────────────────────────────────

describe("analyze - exit strategy", () => {
  it("売却価格は正の値", () => {
    const r = analyze(BASE_INPUT);
    expect(r.exitSalePrice).toBeGreaterThan(0);
  });

  it("保有5年超 → 長期税率(20.315%)が適用される", () => {
    // 保有10年の場合、capitalgain > 0 なら tax = gain * 0.20315
    const r = analyze(BASE_INPUT);
    if (r.exitCapitalGain > 0) {
      expect(approxEqual(r.exitTransferTax, r.exitCapitalGain * 0.20315, r.exitCapitalGain * 0.001)).toBe(true);
    }
  });

  it("保有5年以下 → 短期税率(39.63%)が適用される", () => {
    const r = analyze({ ...BASE_INPUT, holdingYears: 3 });
    if (r.exitCapitalGain > 0) {
      expect(approxEqual(r.exitTransferTax, r.exitCapitalGain * 0.3963, r.exitCapitalGain * 0.001)).toBe(true);
    }
  });
});

// ─── analyze: LTV sensitivity ────────────────────────────────────────────────

describe("analyze - LTV sensitivity", () => {
  it("5つのLTV水準(50%〜90%)のデータを返す", () => {
    const r = analyze(BASE_INPUT);
    expect(r.ltvSensitivity.length).toBe(5);
    expect(r.ltvSensitivity[0].ltv).toBe(0.5);
    expect(r.ltvSensitivity[4].ltv).toBe(0.9);
  });

  it("高LTVほどDSCRが低い", () => {
    const r = analyze(BASE_INPUT);
    const rows = r.ltvSensitivity;
    expect(rows[0].dscr).toBeGreaterThan(rows[4].dscr);
  });
});

// ─── analyze: critical errors ────────────────────────────────────────────────

describe("analyze - critical errors", () => {
  it("正常な物件ではcriticalErrorsが空", () => {
    const r = analyze(BASE_INPUT);
    expect(r.criticalErrors).toEqual([]);
  });

  it("DEADCROSS_EARLY: 10年以内にデッドクロス発生で REJECT エラー", () => {
    // 短い耐用年数の建物 + 長いローンで早期デッドクロスを誘発
    // 築21年木造（簡便法耐用年数 = (22-21)+floor(21*0.2) = 1+4 = 5年）
    const r = analyze({
      ...BASE_INPUT,
      buildingAge: 21,
      loanYears: 35,
      loanAmount: 10_000_000,
    });
    if (r.deadCrossYear > 0 && r.deadCrossYear <= 10) {
      const err = r.criticalErrors.find((e) => e.code === "DEADCROSS_EARLY");
      expect(err).toBeDefined();
      expect(err!.status).toBe("REJECT");
    }
  });
});

// ─── analyze: rent decline rate ──────────────────────────────────────────────

describe("analyze - rent decline rate", () => {
  it("年1%下落の場合、10年目の賃料は初年度より低い", () => {
    const r = analyze({ ...BASE_INPUT, rentDeclineRate: 0.01 });
    expect(r.yearlyResults[9].annualRent).toBeLessThan(r.yearlyResults[0].annualRent);
  });

  it("賃料下落なしの場合、毎年の賃料は一定", () => {
    const r = analyze({ ...BASE_INPUT, rentDeclineRate: 0 });
    expect(approxEqual(r.yearlyResults[0].annualRent, r.yearlyResults[9].annualRent, 1)).toBe(true);
  });
});

// ─── analyze: equal-principal loan method ────────────────────────────────────

describe("analyze - equal-principal loan", () => {
  it("元金均等返済: 初年度の元金返済額が後年より多い（逓減型）", () => {
    const r = analyze({ ...BASE_INPUT, loanMethod: "equal-principal" });
    expect(r.yearlyResults[0].annualPrincipal).toBeGreaterThanOrEqual(
      r.yearlyResults[9].annualPrincipal,
    );
  });

  it("元金均等返済でも35年分の結果が返る", () => {
    const r = analyze({ ...BASE_INPUT, loanMethod: "equal-principal" });
    expect(r.yearlyResults.length).toBeGreaterThanOrEqual(35);
  });
});

// ─── analyze: NPV / IRR ──────────────────────────────────────────────────────

describe("analyze - NPV and IRR", () => {
  it("エクイティが正でholdingYears>0ならNPVが計算される", () => {
    const r = analyze(BASE_INPUT);
    // equity = totalInvestment - loanAmount > 0
    expect(typeof r.npv).toBe("number");
    expect(isFinite(r.npv)).toBe(true);
  });

  it("オーバーローン(equity<=0)の場合はIRRがnull", () => {
    // loanAmount >= totalInvestment でオーバーローン
    const r = analyze({ ...BASE_INPUT, loanAmount: 20_000_000 });
    expect(r.irr).toBeNull();
  });
});

// ─── analyze: DSCR ───────────────────────────────────────────────────────────

describe("analyze - DSCR", () => {
  it("DSCRは正の値(NOI/年間返済額)", () => {
    const r = analyze(BASE_INPUT);
    expect(r.dscr).toBeGreaterThan(0);
  });
});

// ─── analyze: acquisition costs ──────────────────────────────────────────────

describe("analyze - acquisition costs", () => {
  it("仲介手数料が正の値", () => {
    const r = analyze(BASE_INPUT);
    expect(r.acquisitionCosts.brokerageFee).toBeGreaterThan(0);
  });

  it("合計 = 各費用の合計", () => {
    const r = analyze(BASE_INPUT);
    const c = r.acquisitionCosts;
    const sum = c.brokerageFee + c.stampDuty + c.registrationTax + c.realEstateAcquisitionTax + c.propertyTaxProration;
    expect(approxEqual(c.total, sum, 1)).toBe(true);
  });
});

// ─── analyze: declining balance depreciation ─────────────────────────────────

describe("analyze - declining balance depreciation", () => {
  it("定率法では初年度の減価償却費が定額法より大きい（通常）", () => {
    const straight = analyze({ ...BASE_INPUT, depreciationMethod: "straight-line" });
    const declining = analyze({ ...BASE_INPUT, depreciationMethod: "declining-balance" });
    // 定率法の初年度: buildingCost * (1.5/usefulLife)
    // 定額法の初年度: buildingCost / usefulLife
    // 1.5/L > 1/L なので定率法の方が大きい
    expect(declining.yearlyResults[0].annualDepreciation).toBeGreaterThan(
      straight.yearlyResults[0].annualDepreciation,
    );
  });
});

// ─── analyze: vacancy rate stress ────────────────────────────────────────────

describe("analyze - vacancy rate stress test", () => {
  it("vacancyRateDeltaが設定されると実効空室率に反映される", () => {
    const base = analyze(BASE_INPUT);
    const stressed = analyze({ ...BASE_INPUT, vacancyRateDelta: 0.10 });
    // 空室増加 → 年間賃料減少
    expect(stressed.yearlyResults[0].annualRent).toBeLessThan(base.yearlyResults[0].annualRent);
  });
});

// ─── stress scenario: rent decline and after-tax CF (#311, #312) ──────────────

describe("calcStressScenario - rent decline and after-tax CF", () => {
  it("RentDeclineRate > 0 のとき DSCR が下落率なしより低くなる (#311)", () => {
    const base: InvestmentInput = {
      ...BASE_INPUT,
      monthlyRent: 130_000,
      loanAmount: 18_000_000,
      loanYears: 25,
      expenseRate: 0.20,
      incomeTaxRate: 0.33,
      holdingYears: 10,
      rateAdjustmentSchedule: [{ afterYear: 5, rate: 0.03 }],
    };
    const r0 = analyze({ ...base, rentDeclineRate: 0 });
    const r1 = analyze({ ...base, rentDeclineRate: 0.03 });

    const baseline0 = r0.stressScenarios.find((s) => s.label === "ベースライン")!;
    const baseline1 = r1.stressScenarios.find((s) => s.label === "ベースライン")!;

    expect(baseline1.dscr).toBeLessThan(baseline0.dscr);
  });

  it("IncomeTaxRate > 0 のとき税引後CF基準の黒転が税なしより悪化（遅延またはなし）する (#312)", () => {
    // yearNOI がローン返済額をわずかに上回る（CF > 0）が、
    // incomeTax(40%) がそのCFを超えるよう設定 → afterTaxCF < 0 → 黒転なし
    const base: InvestmentInput = {
      ...BASE_INPUT,
      monthlyRent: 77_000,
      vacancyRate: 0.05,
      loanAmount: 15_000_000,
      annualLoanRate: 0.02,
      loanYears: 30,
      expenseRate: 0.15,
      holdingYears: 15,
      rentDeclineRate: 0,
    };
    const r0 = analyze({ ...base, incomeTaxRate: 0 });
    const r1 = analyze({ ...base, incomeTaxRate: 0.40 });

    const s0 = r0.stressScenarios.find((s) => s.label === "ベースライン")!;
    const s1 = r1.stressScenarios.find((s) => s.label === "ベースライン")!;

    // 税なしは年1から黒転
    expect(s0.breakEvenYear).toBe(1);
    // 税ありは afterTaxCF < 0 → 黒転なし（-1）または遅延
    if (s1.breakEvenYear !== -1) {
      expect(s1.breakEvenYear).toBeGreaterThan(s0.breakEvenYear);
    }
  });
});

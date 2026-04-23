import type { InvestmentInput, InvestmentResult } from "@/types/investment";
import { fmtYen, fmtPct } from "./format";

export type VerdictLevel = "PASS" | "CAUTION" | "REJECT";

export interface VerdictResult {
  level: VerdictLevel;
  label: string;
  color: string;
  reasons: string[];
  autoComment: string;
}

function buildAutoComment(
  level: VerdictLevel,
  dscr: number,
  dscrStress: number,
  result: InvestmentResult,
): string {
  const parts: string[] = [];

  if (level === "PASS") {
    parts.push(
      `表面利回り${fmtPct(result.grossYield)}・DSCR${dscr.toFixed(2)}ともに基準を上回り、投資適格と判定されました。`,
    );
  } else if (level === "CAUTION") {
    parts.push(
      `DSCR${dscr.toFixed(2)}は最低水準を満たしていますが、余裕が少なく要注意です。`,
    );
  } else {
    parts.push(`複数の重大リスクが検出されたため、現時点では見送りを推奨します。`);
  }

  if (dscrStress > 0 && dscrStress < dscr) {
    const drop = ((dscr - dscrStress) / dscr) * 100;
    parts.push(
      `複合ストレス時のDSCRは${dscrStress.toFixed(2)}（${drop.toFixed(0)}%低下）です。`,
    );
  }

  if (result.deadCrossYear > 0) {
    parts.push(`${result.deadCrossYear}年目にデッドクロスが発生する点に注意してください。`);
  }

  return parts.join("　");
}

export function calcVerdict(
  _input: InvestmentInput,
  result: InvestmentResult,
  dscr: number,
  dscrStress: number,
): VerdictResult {
  const hasREJECT = result.criticalErrors.some((e) => e.status === "REJECT");

  const isPass =
    !hasREJECT &&
    dscr >= 1.2 &&
    result.grossYield >= result.yieldTarget &&
    result.exitTotalEquity >= 0;

  const isCaution = !isPass && !hasREJECT && dscr >= 1.0;

  const level: VerdictLevel = isPass ? "PASS" : isCaution ? "CAUTION" : "REJECT";

  const levelMeta = {
    PASS: { label: "投資適格", color: "#16a34a" },
    CAUTION: { label: "要交渉", color: "#d97706" },
    REJECT: { label: "見送り推奨", color: "#dc2626" },
  } as const;

  const reasons: string[] = [];

  if (dscr >= 1.2) {
    reasons.push(`DSCR ${dscr.toFixed(2)}（基準1.20達成）`);
  } else if (dscr >= 1.0) {
    reasons.push(`DSCR ${dscr.toFixed(2)}（基準1.20未達、1.00は超過）`);
  } else {
    reasons.push(`DSCR ${dscr.toFixed(2)}（基準1.00未達）`);
  }

  if (result.grossYield >= result.yieldTarget) {
    reasons.push(
      `表面利回り${fmtPct(result.grossYield)}（目標${fmtPct(result.yieldTarget)}達成）`,
    );
  } else {
    reasons.push(
      `表面利回り${fmtPct(result.grossYield)}（目標${fmtPct(result.yieldTarget)}未達）`,
    );
  }

  if (result.exitTotalEquity >= 0) {
    reasons.push(`出口Equity ${fmtYen(result.exitTotalEquity)}（プラス）`);
  } else {
    reasons.push(`出口Equity ${fmtYen(result.exitTotalEquity)}（マイナス）`);
  }

  const autoComment = buildAutoComment(level, dscr, dscrStress, result);

  return {
    level,
    label: levelMeta[level].label,
    color: levelMeta[level].color,
    reasons: reasons.slice(0, 3),
    autoComment,
  };
}

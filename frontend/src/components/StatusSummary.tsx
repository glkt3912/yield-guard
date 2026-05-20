import React from "react";
import { CheckCircle2, AlertTriangle, XOctagon } from "lucide-react";
import type { InvestmentInput, InvestmentResult } from "@/types/investment";
import { calcVerdict } from "@/lib/pdf/verdict";

interface StatusSummaryProps {
  result: InvestmentResult;
  input: InvestmentInput;
}

const LEVEL_CONFIG = {
  PASS: {
    label: "投資適格",
    icon: <CheckCircle2 className="h-4 w-4 shrink-0 text-green-600" aria-hidden />,
    className: "border-green-300 bg-green-50 text-green-900",
  },
  CAUTION: {
    label: "要交渉",
    icon: <AlertTriangle className="h-4 w-4 shrink-0 text-yellow-600" aria-hidden />,
    className: "border-yellow-300 bg-yellow-50 text-yellow-900",
  },
  REJECT: {
    label: "見送り推奨",
    icon: <XOctagon className="h-4 w-4 shrink-0 text-red-600" aria-hidden />,
    className: "border-red-300 bg-red-50 text-red-900",
  },
} as const;

export function StatusSummary({ result, input }: StatusSummaryProps) {
  const dscrStress = result.stressScenarios.find((sc) => sc.label === "複合ストレス")?.dscr ?? 0;
  const verdict = calcVerdict(input, result, result.dscr, dscrStress);
  const { label, icon, className } = LEVEL_CONFIG[verdict.level];

  const grossYieldPct = (result.grossYield * 100).toFixed(2);
  const dscrVal = result.dscr.toFixed(2);
  const dcYear = result.deadCrossYear > 0 ? `${result.deadCrossYear}年目` : "なし";

  return (
    <div
      role="status"
      aria-label={`投資判定: ${label}`}
      data-testid="status-summary-badge"
      className={`flex items-center gap-2 rounded-lg border px-4 py-2.5 text-sm font-medium ${className}`}
    >
      {icon}
      <span className="font-semibold">[{label}]</span>
      <span className="hidden sm:inline text-muted-foreground">|</span>
      <span>
        利回り {grossYieldPct}% / 返済の余裕度 {dscrVal} / 税負担急増リスク {dcYear}
      </span>
    </div>
  );
}

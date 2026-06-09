import React from "react";
import { TrendingUp, ShieldCheck, AlertCircle, Banknote, CreditCard } from "lucide-react";
import type { InvestmentResult } from "@/types/investment";
import { TermTooltip } from "@/components/ui/TermTooltip";

interface KpiStripProps {
  result: InvestmentResult;
  yieldTarget?: number;
  holdingYears?: number;
}

interface KpiCellProps {
  icon: React.ReactNode;
  label: React.ReactNode;
  value: string;
  sub: string;
  subPositive?: boolean;
}

function KpiCell({ icon, label, value, sub, subPositive }: KpiCellProps) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border bg-card p-3 shadow-sm">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        {icon}
        <span>{label}</span>
      </div>
      <p className="text-xl font-bold leading-none">{value}</p>
      <p
        className={`text-xs font-medium ${
          subPositive === undefined
            ? "text-muted-foreground"
            : subPositive
              ? "text-green-600"
              : "text-red-600"
        }`}
      >
        {sub}
      </p>
    </div>
  );
}

export function KpiStrip({ result, yieldTarget = 0.08, holdingYears = 30 }: KpiStripProps) {
  const monthlyPaymentMan = Math.floor(
    (result.yearlyResults[0]?.annualLoanPayment ?? 0) / 12 / 10_000
  );

  const grossYieldPct = result.marketGrossYield * 100;
  const yieldDiff = grossYieldPct - yieldTarget * 100;
  const yieldDiffStr = (yieldDiff >= 0 ? "+" : "") + yieldDiff.toFixed(2) + "pp vs 目標";

  const dscrDiff = result.dscr - 1.0;
  const dscrDiffStr = (dscrDiff >= 0 ? "+" : "") + dscrDiff.toFixed(2) + " vs 1.0";

  // deadCrossYear: -1 or 0 = なし、正の数 = 発生年
  const dcYear = result.deadCrossYear > 0 ? `${result.deadCrossYear}年目` : "なし";
  const dcSub =
    result.deadCrossYear > 0
      ? result.deadCrossYear > holdingYears
        ? "保有期間外（安全）"
        : "保有期間内に発生"
      : "デッドクロスなし";
  const dcPositive = result.deadCrossYear <= 0 || result.deadCrossYear > holdingYears;

  const equityMan = Math.round(result.exitTotalEquity / 10_000);
  const equityStr =
    Math.abs(equityMan) >= 10_000
      ? `${(equityMan / 10_000).toFixed(1)}億円`
      : `${equityMan.toLocaleString()}万円`;
  const equityPositive = result.exitTotalEquity >= 0;
  const equitySub = equityPositive ? "出口時プラス" : "出口時マイナス";

  return (
    <div
      role="region"
      aria-label="KPIサマリ"
      data-testid="kpi-strip"
      className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5"
    >
      <KpiCell
        icon={<CreditCard className="h-3.5 w-3.5" aria-hidden />}
        label="月々の返済額"
        value={`${monthlyPaymentMan.toLocaleString()}万円`}
        sub="元利合計（1年目）"
      />
      <KpiCell
        icon={<TrendingUp className="h-3.5 w-3.5" aria-hidden />}
        label={<TermTooltip term="grossYield">表面利回り</TermTooltip>}
        value={`${grossYieldPct.toFixed(2)}%`}
        sub={yieldDiffStr}
        subPositive={yieldDiff >= 0}
      />
      <KpiCell
        icon={<ShieldCheck className="h-3.5 w-3.5" aria-hidden />}
        label={<TermTooltip term="dscr">DSCR（1年目）</TermTooltip>}
        value={result.dscr.toFixed(2)}
        sub={dscrDiffStr}
        subPositive={dscrDiff >= 0}
      />
      <KpiCell
        icon={<AlertCircle className="h-3.5 w-3.5" aria-hidden />}
        label={<TermTooltip term="deadCross">デッドクロス</TermTooltip>}
        value={dcYear}
        sub={dcSub}
        subPositive={dcPositive}
      />
      <KpiCell
        icon={<Banknote className="h-3.5 w-3.5" aria-hidden />}
        label={<TermTooltip term="exitEquity">出口 Equity</TermTooltip>}
        value={equityStr}
        sub={equitySub}
        subPositive={equityPositive}
      />
    </div>
  );
}

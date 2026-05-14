import React from "react";
import { TrendingUp, ShieldCheck, AlertCircle, Banknote } from "lucide-react";
import type { InvestmentResult } from "@/types/investment";

interface KpiStripProps {
  result: InvestmentResult;
  yieldTarget?: number;
}

interface KpiCellProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub: string;
  subPositive?: boolean;
}

function KpiCell({ icon, label, value, sub, subPositive }: KpiCellProps) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border bg-white p-3 shadow-sm">
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

export function KpiStrip({ result, yieldTarget = 0.08 }: KpiStripProps) {
  const grossYieldPct = result.grossYield * 100;
  const yieldDiff = grossYieldPct - yieldTarget * 100;
  const yieldDiffStr = (yieldDiff >= 0 ? "+" : "") + yieldDiff.toFixed(2) + "pp vs 目標";

  const dscrDiff = result.dscr - 1.0;
  const dscrDiffStr = (dscrDiff >= 0 ? "+" : "") + dscrDiff.toFixed(2) + " vs 1.0";

  // deadCrossYear: -1 or 0 = なし、正の数 = 発生年
  const dcYear = result.deadCrossYear > 0 ? `${result.deadCrossYear}年目` : "なし";
  const dcSub =
    result.deadCrossYear > 0
      ? result.deadCrossYear > 10
        ? "保有期間内は安全圏"
        : "保有期間内に発生"
      : "デッドクロスなし";
  const dcPositive = result.deadCrossYear <= 0 || result.deadCrossYear > 10;

  const equityMan = Math.round(result.exitTotalEquity / 10_000);
  const equityStr =
    Math.abs(equityMan) >= 10_000
      ? `${(equityMan / 10_000).toFixed(1)}億円`
      : `${equityMan.toLocaleString()}万円`;
  const equityPositive = result.exitTotalEquity >= 0;
  const equitySub = equityPositive ? "出口時プラス" : "出口時マイナス";

  return (
    <div aria-label="KPIサマリ" className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <KpiCell
        icon={<TrendingUp className="h-3.5 w-3.5" aria-hidden />}
        label="表面利回り"
        value={`${grossYieldPct.toFixed(2)}%`}
        sub={yieldDiffStr}
        subPositive={yieldDiff >= 0}
      />
      <KpiCell
        icon={<ShieldCheck className="h-3.5 w-3.5" aria-hidden />}
        label="DSCR（1年目）"
        value={result.dscr.toFixed(2)}
        sub={dscrDiffStr}
        subPositive={dscrDiff >= 0}
      />
      <KpiCell
        icon={<AlertCircle className="h-3.5 w-3.5" aria-hidden />}
        label="デッドクロス"
        value={dcYear}
        sub={dcSub}
        subPositive={dcPositive}
      />
      <KpiCell
        icon={<Banknote className="h-3.5 w-3.5" aria-hidden />}
        label="出口 Equity"
        value={equityStr}
        sub={equitySub}
        subPositive={equityPositive}
      />
    </div>
  );
}

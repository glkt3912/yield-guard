"use client";
import React, { useState } from "react";
import { Badge } from "@/components/ui/badge";
import type { InvestmentInput, InvestmentResult } from "@/types/investment";
import { formatMan, formatPct } from "@/lib/utils";
import { CheckCircle, AlertTriangle, FileText } from "lucide-react";
import { downloadReportPDF } from "@/lib/generatePdf";

interface Props {
  result: InvestmentResult;
  input: InvestmentInput;
  yieldPct: number;
  netYieldPct: number;
}

/**
 * Agent-facing one-glance card shown only on mobile.
 * Shows the 8 most decision-critical metrics and a PDF export button.
 */
export function MobileSummaryCard({ result, input, yieldPct, netYieldPct }: Props) {
  const [pdfLoading, setPdfLoading] = useState(false);

  const dscr = result.dscr;
  const deadCrossYear = result.deadCrossYear;
  const hasDeadCross = deadCrossYear > 0 && deadCrossYear <= 35;
  const deadCrossEarly = hasDeadCross && deadCrossYear <= 10;
  const downPayment = Math.max(0, input.landPrice + input.buildingCost - input.loanAmount);

  const dscrColor =
    dscr >= 1.2 ? "text-green-700" : dscr >= 1.0 ? "text-yellow-700" : "text-red-700";
  const dscrBg =
    dscr >= 1.2 ? "bg-green-50" : dscr >= 1.0 ? "bg-yellow-50" : "bg-red-50";

  async function handlePdf() {
    setPdfLoading(true);
    try {
      await downloadReportPDF(input, result);
    } finally {
      setPdfLoading(false);
    }
  }

  return (
    <div className="rounded-2xl border-2 border-primary/30 bg-card p-4 shadow-sm lg:hidden">
      <div className="mb-3 flex items-center justify-between">
        <p className="text-sm font-semibold text-foreground">物件サマリー</p>
        {result.isAboveYieldTarget ? (
          <Badge variant="success" className="flex items-center gap-1">
            <CheckCircle className="h-3 w-3" />
            目標利回り達成
          </Badge>
        ) : (
          <Badge variant="danger" className="flex items-center gap-1">
            <AlertTriangle className="h-3 w-3" />
            目標利回り未達
          </Badge>
        )}
      </div>

      <div className="grid grid-cols-2 gap-2 text-sm">
        <MetricRow label="表面利回り" value={`${yieldPct.toFixed(2)}%`} />
        <MetricRow label="実質利回り" value={`${netYieldPct.toFixed(2)}%`} />
        <div className={`col-span-1 rounded-lg px-2 py-1.5 ${dscrBg}`}>
          <p className="text-xs text-muted-foreground">DSCR</p>
          <p className={`font-bold ${dscrColor}`}>{dscr.toFixed(2)}</p>
        </div>
        <div className={`col-span-1 rounded-lg px-2 py-1.5 ${hasDeadCross ? (deadCrossEarly ? "bg-red-50" : "bg-yellow-50") : "bg-green-50"}`}>
          <p className="text-xs text-muted-foreground">デッドクロス</p>
          <p className={`font-bold ${hasDeadCross ? (deadCrossEarly ? "text-red-700" : "text-yellow-700") : "text-green-700"}`}>
            {hasDeadCross ? `${deadCrossYear}年目〜` : "なし"}
          </p>
        </div>
        <MetricRow label="必要頭金" value={formatMan(downPayment)} />
        <MetricRow label="出口手取り" value={formatMan(result.exitNetProceeds)} />
        <MetricRow
          label="IRR"
          value={result.irr != null ? `${(result.irr * 100).toFixed(2)}%` : "―"}
          valueClass={result.irr != null && result.irr >= 0 ? "text-green-700" : "text-red-700"}
        />
        <MetricRow
          label="最終手残り"
          value={formatMan(result.exitTotalEquity)}
          valueClass={result.exitTotalEquity >= 0 ? "text-green-700" : "text-red-700"}
        />
      </div>

      <button
        type="button"
        onClick={handlePdf}
        disabled={pdfLoading}
        className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg border border-primary/40 bg-primary/5 px-3 py-2.5 text-sm font-medium text-primary transition-colors hover:bg-primary/10 disabled:opacity-60 min-h-[44px]"
      >
        <FileText className="h-4 w-4" />
        {pdfLoading ? "生成中..." : "詳細PDFを出力"}
      </button>
    </div>
  );
}

function MetricRow({
  label,
  value,
  valueClass = "text-foreground",
}: {
  label: string;
  value: string;
  valueClass?: string;
}) {
  return (
    <div className="rounded-lg bg-muted/40 px-2 py-1.5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className={`font-bold ${valueClass}`}>{value}</p>
    </div>
  );
}

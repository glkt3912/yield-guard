"use client";
import React from "react";
import { AlertTriangle } from "lucide-react";
import type { MultiExitRow } from "@/types/investment";

interface MultiExitCompareTableProps {
  rows: MultiExitRow[];
}

function formatYen(value: number): string {
  const abs = Math.abs(value);
  if (abs >= 1_0000_0000) {
    return `${(value / 1_0000_0000).toFixed(2)}億円`;
  }
  if (abs >= 1_0000) {
    return `${Math.round(value / 1_0000)}万円`;
  }
  return `${Math.round(value).toLocaleString()}円`;
}

function formatPct(value: number): string {
  return `${(value * 100).toFixed(2)}%`;
}

export default function MultiExitCompareTable({ rows }: MultiExitCompareTableProps) {
  if (!rows || rows.length === 0) return null;

  // 出口エクイティが最大の列インデックスを特定
  const maxEquity = Math.max(...rows.map((r) => r.exitEquity));
  const maxEquityIdx = rows.findIndex((r) => r.exitEquity === maxEquity);

  const labels = [
    "想定売却価格",
    "譲渡税率",
    "譲渡税額",
    "残債残高",
    "累積税引後CF",
    "出口エクイティ合計",
    "IRR",
  ];

  return (
    <div className="rounded-xl border bg-white p-5 shadow-sm">
      <h3 className="text-base font-semibold text-foreground mb-4">複数保有年数 出口比較</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr>
              <th className="text-left py-2 pr-4 font-medium text-muted-foreground whitespace-nowrap border-b">
                項目
              </th>
              {rows.map((row, i) => (
                <th
                  key={row.year}
                  className={`text-right py-2 px-3 font-medium border-b whitespace-nowrap ${
                    i === maxEquityIdx ? "bg-green-50" : ""
                  }`}
                >
                  <div className="flex flex-col items-end gap-1">
                    <span>{row.year}年</span>
                    {row.isShortTermWarn && (
                      <span className="inline-flex items-center gap-0.5 rounded-full bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-800">
                        <AlertTriangle className="h-3 w-3" aria-hidden="true" />
                        短期譲渡税
                      </span>
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {labels.map((label, labelIdx) => (
              <tr key={label} className={labelIdx % 2 === 0 ? "bg-muted/20" : ""}>
                <td className="py-2 pr-4 text-muted-foreground whitespace-nowrap">{label}</td>
                {rows.map((row, colIdx) => {
                  let value: string;
                  let className = "text-right py-2 px-3 tabular-nums";
                  if (colIdx === maxEquityIdx) {
                    className += " bg-green-50";
                  }
                  if (label === "想定売却価格") {
                    value = formatYen(row.salePrice);
                  } else if (label === "譲渡税率") {
                    value = formatPct(row.transferTaxRate);
                  } else if (label === "譲渡税額") {
                    value = formatYen(row.transferTax);
                  } else if (label === "残債残高") {
                    value = formatYen(row.remainingLoan);
                  } else if (label === "累積税引後CF") {
                    value = formatYen(row.cumulativeCf);
                  } else if (label === "出口エクイティ合計") {
                    value = formatYen(row.exitEquity);
                    if (colIdx === maxEquityIdx) {
                      className += " font-semibold text-green-700";
                    }
                  } else if (label === "IRR") {
                    value = row.irr != null ? formatPct(row.irr) : "-";
                  } else {
                    value = "-";
                  }
                  return (
                    <td key={row.year} className={className}>
                      {value}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="mt-3 text-xs text-muted-foreground">
        ※ 緑ハイライト列が出口エクイティ最大。短期譲渡税（5年以下）は税率39.63%が適用されます。
      </p>
    </div>
  );
}

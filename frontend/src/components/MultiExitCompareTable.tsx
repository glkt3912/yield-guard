"use client";

import React from "react";
import { AlertTriangle } from "lucide-react";
import type { MultiExitRow } from "@/types/investment";
import { TermTooltip } from "@/components/ui/TermTooltip";

interface MultiExitCompareTableProps {
  rows: MultiExitRow[];
}

function formatYen(value: number): string {
  const abs = Math.abs(value);
  if (abs >= 100_000_000) {
    return `${(value / 100_000_000).toFixed(2)}億円`;
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

  const labelRows: { id: string; label: React.ReactNode }[] = [
    { id: "想定売却価格", label: "想定売却価格" },
    { id: "譲渡税率", label: <TermTooltip term="transferTaxRate">譲渡税率</TermTooltip> },
    { id: "譲渡税額", label: "譲渡税額" },
    { id: "残債残高", label: "残債残高" },
    { id: "累積税引後CF", label: "累積税引後CF" },
    {
      id: "出口エクイティ合計",
      label: <TermTooltip term="exitEquity">出口エクイティ合計</TermTooltip>,
    },
    { id: "IRR", label: <TermTooltip term="irr">IRR</TermTooltip> },
  ];

  return (
    <div className="rounded-xl border bg-card p-5 shadow-sm">
      <h3 className="text-base font-semibold text-foreground mb-1">複数保有年数 出口比較</h3>
      <p className="text-sm text-muted-foreground mb-4">
        保有年数ごとの出口エクイティとIRRを比較し、最も有利な売却タイミングを把握できます。
      </p>
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
            {labelRows.map(({ id, label }, labelIdx) => (
              <tr key={id} className={labelIdx % 2 === 0 ? "bg-muted/20" : ""}>
                <td className="py-2 pr-4 text-muted-foreground whitespace-nowrap">{label}</td>
                {rows.map((row, colIdx) => {
                  let value: string;
                  let className = "text-right py-2 px-3 tabular-nums";
                  if (colIdx === maxEquityIdx) {
                    className += " bg-green-50";
                  }
                  if (id === "想定売却価格") {
                    value = formatYen(row.salePrice);
                  } else if (id === "譲渡税率") {
                    value = formatPct(row.transferTaxRate);
                  } else if (id === "譲渡税額") {
                    value = formatYen(row.transferTax);
                  } else if (id === "残債残高") {
                    value = formatYen(row.remainingLoan);
                  } else if (id === "累積税引後CF") {
                    value = formatYen(row.cumulativeCf);
                  } else if (id === "出口エクイティ合計") {
                    value = formatYen(row.exitEquity);
                    if (colIdx === maxEquityIdx) {
                      className += " font-semibold text-green-700";
                    }
                  } else if (id === "IRR") {
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
        ※
        緑ハイライト列が出口エクイティ最大。短期譲渡税（保有5年以下）は税率39.63%、長期（5年超）は20.315%が適用されます。
      </p>
      <p className="mt-1 text-xs text-muted-foreground">
        ※
        長期/短期は税法上「譲渡した年の1月1日時点で所有期間5年超」で判定します（本ツールは保有年数で簡略判定）。
        取得時期によっては実質的に取得から約6年の保有が必要になるため、5〜6年目の売却は税率を保守的にご確認ください。
      </p>
    </div>
  );
}

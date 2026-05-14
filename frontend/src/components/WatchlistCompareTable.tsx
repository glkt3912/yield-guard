"use client";

import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { BarChart2, TrendingUp, TrendingDown, Minus } from "lucide-react";
import type { WatchlistItem, WatchlistMetrics } from "@/types/investment";

interface WatchlistCompareTableProps {
  items: WatchlistItem[]; // 2–4 items already selected
}

type MetricKey = keyof WatchlistMetrics;

interface MetricRow {
  label: string;
  key: MetricKey;
  format: (v: number | null | undefined) => string;
  best: "max" | "min";
}

const METRIC_ROWS: MetricRow[] = [
  {
    label: "表面利回り",
    key: "grossYield",
    format: (v) => (v != null ? `${(Number(v) * 100).toFixed(1)}%` : "-"),
    best: "max",
  },
  {
    label: "実質利回り",
    key: "netYield",
    format: (v) => (v != null ? `${(Number(v) * 100).toFixed(1)}%` : "-"),
    best: "max",
  },
  {
    label: "DSCR",
    key: "dscr",
    format: (v) => (v != null ? Number(v).toFixed(2) : "-"),
    best: "max",
  },
  {
    label: "IRR",
    key: "irr",
    format: (v) => (v != null ? `${(Number(v) * 100).toFixed(1)}%` : "-"),
    best: "max",
  },
  {
    label: "総投資額",
    key: "totalInvestment",
    format: (v) =>
      v != null
        ? `${(Number(v) / 10_000).toLocaleString("ja-JP", { maximumFractionDigits: 0 })}万円`
        : "-",
    best: "min",
  },
  {
    label: "出口エクイティ",
    key: "exitTotalEquity",
    format: (v) =>
      v != null
        ? `${(Number(v) / 10_000).toLocaleString("ja-JP", { maximumFractionDigits: 0 })}万円`
        : "-",
    best: "max",
  },
  {
    label: "デッドクロス年",
    key: "deadCrossYear",
    format: (v) => {
      if (v == null) return "-";
      return Number(v) === -1 ? "なし（最良）" : `${Number(v)}年目`;
    },
    best: "max", // -1 (none) is best, larger deadCrossYear means it arrives later = better
  },
  {
    label: "NPV",
    key: "npv",
    format: (v) =>
      v != null
        ? `${(Number(v) / 10_000).toLocaleString("ja-JP", { maximumFractionDigits: 0 })}万円`
        : "-",
    best: "max",
  },
];

function getValue(metrics: WatchlistMetrics | undefined, key: MetricKey): number | null {
  if (!metrics) return null;
  const v = metrics[key];
  if (v == null) return null;
  // deadCrossYear: -1 means "no dead cross" which is the best possible value
  if (key === "deadCrossYear" && Number(v) === -1) return Infinity;
  return Number(v);
}

/**
 * Find the index of the best value among items for a given metric row.
 * Returns -1 if no valid values exist or if all values are equal.
 */
function findBestIndex(items: WatchlistItem[], row: MetricRow): number {
  const values = items.map((item) => getValue(item.metrics, row.key));
  const validValues = values.filter((v): v is number => v !== null);
  if (validValues.length === 0) return -1;

  const bestValue = row.best === "max" ? Math.max(...validValues) : Math.min(...validValues);

  // Only highlight if there's a unique best (no tie)
  const bestCount = validValues.filter((v) => v === bestValue).length;
  if (bestCount > 1) return -1;

  return values.indexOf(bestValue);
}

function DscrTrendIcon({ dscr }: { dscr: number | null }) {
  if (dscr === null) return <Minus className="h-3 w-3 text-muted-foreground" aria-hidden="true" />;
  if (dscr >= 1.2) return <TrendingUp className="h-3 w-3 text-green-600" aria-hidden="true" />;
  if (dscr >= 1.0) return <Minus className="h-3 w-3 text-yellow-600" aria-hidden="true" />;
  return <TrendingDown className="h-3 w-3 text-red-600" aria-hidden="true" />;
}

export default function WatchlistCompareTable({ items }: WatchlistCompareTableProps) {
  return (
    <Card className="rounded-xl shadow-sm">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base font-semibold">
          <BarChart2 className="h-4 w-4 text-primary" aria-hidden="true" />
          物件比較表（{items.length}件）
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[480px] text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="py-2 pr-3 text-left text-xs font-medium text-muted-foreground w-32">
                  指標
                </th>
                {items.map((item) => (
                  <th
                    key={item.id}
                    className="py-2 px-2 text-center text-xs font-medium max-w-[120px]"
                  >
                    <span className="block truncate" title={item.name}>
                      {item.name}
                    </span>
                    <span
                      className={`mt-0.5 inline-block rounded px-1.5 py-0.5 text-xs font-medium ${
                        item.status === "検討中"
                          ? "bg-blue-100 text-blue-800"
                          : item.status === "見送り"
                            ? "bg-gray-100 text-gray-600"
                            : "bg-emerald-100 text-emerald-800"
                      }`}
                    >
                      {item.status}
                    </span>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {METRIC_ROWS.map((row) => {
                const bestIdx = findBestIndex(items, row);
                return (
                  <tr key={row.key} className="border-b border-border/60 last:border-0">
                    <td className="py-2 pr-3 text-xs text-muted-foreground whitespace-nowrap">
                      {row.label}
                    </td>
                    {items.map((item, colIdx) => {
                      // raw: comparison value (Infinity for deadCrossYear=-1)
                      const raw = getValue(item.metrics, row.key);
                      // displayValue: original metric value, used for formatting only
                      const metricVal = item.metrics?.[row.key];
                      const displayValue = metricVal != null ? Number(metricVal) : null;
                      const isBest = bestIdx !== -1 && colIdx === bestIdx;
                      const isUnavailable = raw === null;
                      return (
                        <td
                          key={item.id}
                          className={`py-2 px-2 text-center text-xs ${
                            isBest
                              ? "bg-green-50 font-bold text-green-800 rounded"
                              : isUnavailable
                                ? "text-muted-foreground"
                                : ""
                          }`}
                          aria-label={
                            isBest
                              ? `${item.name} ${row.label}: ${row.format(displayValue)} (最優位)`
                              : undefined
                          }
                        >
                          {/* DSCR行はアイコンを併記してWCAG 1.4.1準拠 */}
                          {row.key === "dscr" ? (
                            <span className="inline-flex items-center justify-center gap-1">
                              <DscrTrendIcon dscr={displayValue} />
                              <span>{row.format(displayValue)}</span>
                              {isBest && <span className="sr-only">（最優位）</span>}
                            </span>
                          ) : (
                            <span className="inline-flex items-center justify-center gap-1">
                              {row.format(displayValue)}
                              {isBest && (
                                <span className="text-green-600" title="最優位" aria-hidden="true">
                                  ★
                                </span>
                              )}
                              {isBest && <span className="sr-only">（最優位）</span>}
                            </span>
                          )}
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          ★ は各指標の最優位物件を示します。同値の場合はハイライトなし。
        </p>
      </CardContent>
    </Card>
  );
}

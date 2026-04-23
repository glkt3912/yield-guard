"use client";
import React from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type { MonteCarloResult } from "@/types/investment";
import { formatMan, formatPct } from "@/lib/utils";
import { Sigma } from "lucide-react";

interface Props {
  result: MonteCarloResult;
}

export function MonteCarloChart({ result }: Props) {
  const {
    simulationCount,
    irrPercentiles,
    equityPercentiles,
    deadCrossRate,
    irrHistogram,
    equityHistogram,
    successRate,
  } = result;

  // null guard: 全試行のIRRがNaNの場合バックエンドからnullが返る
  const irrData = (irrHistogram ?? []).map((b) => ({
    label: formatPct(b.min, 1),
    count: b.count,
    positive: b.min >= 0,
  }));

  const equityData = (equityHistogram ?? []).map((b) => ({
    label: formatMan(b.min, 0),
    count: b.count,
    positive: b.min >= 0,
  }));

  const pctRows = [
    { label: "P10（悲観）", irr: irrPercentiles.p10, equity: equityPercentiles.p10 },
    { label: "P25", irr: irrPercentiles.p25, equity: equityPercentiles.p25 },
    { label: "P50（中央値）", irr: irrPercentiles.p50, equity: equityPercentiles.p50 },
    { label: "P75", irr: irrPercentiles.p75, equity: equityPercentiles.p75 },
    { label: "P90（楽観）", irr: irrPercentiles.p90, equity: equityPercentiles.p90 },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Sigma className="h-5 w-5 text-primary" />
          モンテカルロ・シミュレーション結果
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          空室率・金利を正規分布でサンプリングした {simulationCount.toLocaleString()} 試行の確率分布
        </p>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* サマリーバッジ */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <StatCard label="IRR正値達成率" value={formatPct(successRate, 1)} positive={successRate >= 0.5} />
          <StatCard label="デッドクロス発生率" value={formatPct(deadCrossRate, 1)} positive={deadCrossRate < 0.3} invert />
          <StatCard label="IRR中央値" value={formatPct(irrPercentiles.p50, 1)} positive={irrPercentiles.p50 >= 0} />
        </div>

        {/* IRRヒストグラム */}
        <div>
          <p className="mb-2 text-sm font-medium">IRR 分布</p>
          {irrData.length === 0 ? (
            <p className="text-xs text-muted-foreground">IRRを算出できた試行がありませんでした。</p>
          ) : (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={irrData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="label" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tick={{ fontSize: 10 }} />
                <Tooltip formatter={(v: number) => [`${v} 件`, "頻度"]} />
                <Bar dataKey="count" maxBarSize={24} radius={[2, 2, 0, 0]}>
                  {irrData.map((entry, i) => (
                    <Cell key={i} fill={entry.positive ? "#60a5fa" : "#fca5a5"} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* 最終純資産ヒストグラム */}
        <div>
          <p className="mb-2 text-sm font-medium">最終純資産 分布</p>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={equityData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis dataKey="label" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
              <YAxis tick={{ fontSize: 10 }} />
              <Tooltip formatter={(v: number) => [`${v} 件`, "頻度"]} />
              <Bar dataKey="count" maxBarSize={24} radius={[2, 2, 0, 0]}>
                {equityData.map((entry, i) => (
                  <Cell key={i} fill={entry.positive ? "#34d399" : "#fca5a5"} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* パーセンタイル表 */}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-xs text-muted-foreground">
                <th className="pb-1 text-left">シナリオ</th>
                <th className="pb-1 text-right">IRR</th>
                <th className="pb-1 text-right">最終純資産</th>
              </tr>
            </thead>
            <tbody>
              {pctRows.map((r) => (
                <tr key={r.label} className="border-b last:border-0">
                  <td className="py-1">{r.label}</td>
                  <td className={`py-1 text-right font-mono ${r.irr >= 0 ? "text-blue-600" : "text-red-500"}`}>
                    {formatPct(r.irr, 1)}
                  </td>
                  <td className={`py-1 text-right font-mono ${r.equity >= 0 ? "text-emerald-600" : "text-red-500"}`}>
                    {formatMan(r.equity, 0)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

function StatCard({
  label,
  value,
  positive,
  invert = false,
}: {
  label: string;
  value: string;
  positive: boolean;
  invert?: boolean;
}) {
  const good = invert ? !positive : positive;
  return (
    <div className={`rounded-lg border p-3 ${good ? "border-emerald-200 bg-emerald-50" : "border-red-200 bg-red-50"}`}>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className={`text-lg font-semibold ${good ? "text-emerald-700" : "text-red-600"}`}>{value}</p>
    </div>
  );
}

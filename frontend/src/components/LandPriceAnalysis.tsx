"use client";
import React from "react";
import {
  ScatterChart,
  Scatter,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { LandPriceComparison } from "@/types/investment";
import { formatTsubo, formatMan } from "@/lib/utils";
import { MapPin } from "lucide-react";

interface Props {
  comparison: LandPriceComparison;
}

const ASSESSMENT_BADGE: Record<string, "success" | "warning" | "danger"> = {
  割安: "success",
  相場: "warning",
  割高: "danger",
};

export function LandPriceAnalysis({ comparison }: Props) {
  const { stats, assessment, inputPricePerTsubo, diffFromMedian } = comparison;

  const scatterData = stats.transactions
    .filter((t) => t.pricePerTsubo > 0 && t.area > 0)
    .map((t) => ({
      area: Math.round(t.area),
      tsubo: Math.round(t.pricePerTsubo / 10_000), // 万円/坪
    }));

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5 text-primary" />
            土地価格相場分析
          </CardTitle>
          <Badge variant={ASSESSMENT_BADGE[assessment] ?? "outline"}>{assessment}</Badge>
        </div>
        <p className="text-xs text-muted-foreground">
          取引件数: {stats.count}件 　平均坪単価: {formatTsubo(stats.averageTsubo)} 　中央値: {formatTsubo(stats.medianTsubo)}
        </p>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* 統計サマリー */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {[
            { label: "最安値", value: formatTsubo(stats.minTsubo) },
            { label: "中央値", value: formatTsubo(stats.medianTsubo) },
            { label: "平均値", value: formatTsubo(stats.averageTsubo) },
            { label: "最高値", value: formatTsubo(stats.maxTsubo) },
          ].map(({ label, value }) => (
            <div key={label} className="rounded-md border bg-muted/30 p-2 text-center">
              <p className="text-xs text-muted-foreground">{label}</p>
              <p className="text-sm font-semibold">{value}</p>
            </div>
          ))}
        </div>

        {/* 検討中価格 vs 相場 */}
        {inputPricePerTsubo > 0 && (
          <div
            className={`rounded-md border p-3 ${
              assessment === "割高"
                ? "border-red-200 bg-red-50"
                : assessment === "割安"
                ? "border-green-200 bg-green-50"
                : "border-yellow-200 bg-yellow-50"
            }`}
          >
            <p className="text-sm font-semibold">
              検討中の坪単価: {formatTsubo(inputPricePerTsubo)}
            </p>
            <p className="text-xs mt-1">
              中央値との差:{" "}
              <span className={diffFromMedian > 0 ? "text-red-600 font-bold" : "text-green-600 font-bold"}>
                {diffFromMedian > 0 ? "+" : ""}{formatTsubo(Math.abs(diffFromMedian))}
                （{diffFromMedian > 0 ? "割高" : "割安"}）
              </span>
            </p>
          </div>
        )}

        {/* 散布図: 面積 vs 坪単価 */}
        {scatterData.length > 0 && (
          <>
            <p className="text-xs font-medium text-muted-foreground">取引データ分布（面積 vs 坪単価）</p>
            <ResponsiveContainer width="100%" height={220}>
              <ScatterChart margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis
                  dataKey="area"
                  name="面積"
                  unit="m²"
                  tick={{ fontSize: 10 }}
                  label={{ value: "面積(m²)", position: "insideBottom", offset: -4, fontSize: 10 }}
                />
                <YAxis
                  dataKey="tsubo"
                  name="坪単価"
                  unit="万円"
                  tick={{ fontSize: 10 }}
                  width={50}
                />
                <Tooltip
                  formatter={(v: number, name: string) => [
                    name === "tsubo" ? `${v}万円/坪` : `${v}m²`,
                    name === "tsubo" ? "坪単価" : "面積",
                  ]}
                />
                <Scatter data={scatterData} fill="#60a5fa" opacity={0.6} />
                {inputPricePerTsubo > 0 && (
                  <ReferenceLine
                    y={Math.round(inputPricePerTsubo / 10_000)}
                    stroke="#ef4444"
                    strokeWidth={2}
                    strokeDasharray="6 3"
                    label={{
                      value: "検討中",
                      position: "right",
                      fontSize: 10,
                      fill: "#ef4444",
                    }}
                  />
                )}
                <ReferenceLine
                  y={Math.round(stats.medianTsubo / 10_000)}
                  stroke="#f59e0b"
                  strokeWidth={1.5}
                  strokeDasharray="4 4"
                  label={{
                    value: "中央値",
                    position: "right",
                    fontSize: 10,
                    fill: "#f59e0b",
                  }}
                />
              </ScatterChart>
            </ResponsiveContainer>
          </>
        )}
      </CardContent>
    </Card>
  );
}

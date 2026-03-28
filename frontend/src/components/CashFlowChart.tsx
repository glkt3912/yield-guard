"use client";
import React from "react";
import {
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ReferenceLine,
  ResponsiveContainer,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type { InvestmentResult } from "@/types/investment";
import { formatMan } from "@/lib/utils";
import { TrendingUp } from "lucide-react";

interface Props {
  result: InvestmentResult;
}

export function CashFlowChart({ result }: Props) {
  const { yearlyResults, exitTotalEquity, exitSalePrice, exitNetProceeds } = result;

  const data = yearlyResults.slice(0, 35).map((y) => ({
    year: `${y.year}年`,
    税引後CF: Math.round(y.afterTaxCashFlow / 10_000),
    累積CF: Math.round(y.cumulativeCashFlow / 10_000),
    isDeadCross: y.isDeadCrossYear,
  }));

  // 累積CFが正に転じる年
  const breakEvenYear = yearlyResults.find((y) => y.cumulativeCashFlow >= 0)?.year ?? null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5 text-primary" />
          キャッシュフロー推移（35年）
        </CardTitle>
        {breakEvenYear && (
          <p className="text-xs text-muted-foreground">
            累積CFが黒字転換：<span className="font-semibold text-green-600">{breakEvenYear}年目</span>
          </p>
        )}
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <ComposedChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="year" tick={{ fontSize: 11 }} interval={4} />
            <YAxis
              yAxisId="left"
              tickFormatter={(v) => `${v}万`}
              tick={{ fontSize: 11 }}
              width={55}
            />
            <YAxis
              yAxisId="right"
              orientation="right"
              tickFormatter={(v) => `${v}万`}
              tick={{ fontSize: 11 }}
              width={55}
            />
            <Tooltip
              formatter={(value: number, name: string) => [`${value}万円`, name]}
              labelStyle={{ fontWeight: "bold" }}
            />
            <Legend />
            <ReferenceLine yAxisId="left" y={0} stroke="#9ca3af" />
            {breakEvenYear && (
              <ReferenceLine
                yAxisId="right"
                x={`${breakEvenYear}年`}
                stroke="#22c55e"
                strokeDasharray="4 4"
                label={{ value: "黒字転換", position: "top", fontSize: 10, fill: "#22c55e" }}
              />
            )}
            <Bar
              yAxisId="left"
              dataKey="税引後CF"
              fill="#60a5fa"
              radius={[2, 2, 0, 0]}
              maxBarSize={20}
            />
            <Line
              yAxisId="right"
              type="monotone"
              dataKey="累積CF"
              stroke="#f59e0b"
              strokeWidth={2}
              dot={false}
            />
          </ComposedChart>
        </ResponsiveContainer>

        {/* 出口戦略サマリー */}
        <div className="mt-4 grid grid-cols-3 gap-3 rounded-md border bg-muted/30 p-3">
          <div className="text-center">
            <p className="text-xs text-muted-foreground">売却価格</p>
            <p className="font-bold text-sm">{formatMan(exitSalePrice)}</p>
          </div>
          <div className="text-center">
            <p className="text-xs text-muted-foreground">売却手取り</p>
            <p className="font-bold text-sm">{formatMan(exitNetProceeds)}</p>
          </div>
          <div className="text-center">
            <p className="text-xs text-muted-foreground">最終手残り（Equity）</p>
            <p className={`font-bold text-sm ${exitTotalEquity >= 0 ? "text-green-600" : "text-red-600"}`}>
              {formatMan(exitTotalEquity)}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

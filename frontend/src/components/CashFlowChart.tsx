"use client";
import React from "react";
import {
  ComposedChart, Bar, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ReferenceLine, ResponsiveContainer, Cell,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type { InvestmentResult } from "@/types/investment";
import { formatMan } from "@/lib/utils";
import { TrendingUp } from "lucide-react";

interface Props {
  result: InvestmentResult;
  /** 自己資金（総投資額 - ローン金額）。投資回収年の計算に使用 */
  equityInvested: number;
}

export function CashFlowChart({ result, equityInvested }: Props) {
  const { yearlyResults, exitTotalEquity, exitSalePrice, exitNetProceeds } = result;

  const data = yearlyResults.slice(0, 35).map((y) => ({
    year: `${y.year}年`,
    税引後CF: Math.round(y.afterTaxCashFlow / 10_000),
    // 自己資金を初期コストとして加算した累積CF（ISSUE-22）
    累積CF: Math.round((y.cumulativeCashFlow - equityInvested) / 10_000),
    isDeadCrossZone: y.isInDeadCrossZone,
  }));

  // 自己資金を回収した年（累積CF - 自己資金 >= 0）
  const breakEvenYear = yearlyResults.find(
    (y) => y.cumulativeCashFlow - equityInvested >= 0
  )?.year ?? null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5 text-primary" />
          キャッシュフロー推移（35年）
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          累積CFは自己資金（{formatMan(equityInvested)}）を初期コストとして控除しています。
          {breakEvenYear
            ? <> 自己資金回収：<span className="font-semibold text-green-600">{breakEvenYear}年目</span></>
            : " 35年以内に自己資金を回収できない見込みです"}
        </p>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <ComposedChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="year" tick={{ fontSize: 11 }} interval={4} />
            <YAxis yAxisId="left" tickFormatter={(v) => `${v}万`} tick={{ fontSize: 11 }} width={55} />
            <YAxis yAxisId="right" orientation="right" tickFormatter={(v) => `${v}万`} tick={{ fontSize: 11 }} width={55} />
            <Tooltip formatter={(value: number, name: string) => [`${value}万円`, name]} labelStyle={{ fontWeight: "bold" }} />
            <Legend />
            <ReferenceLine yAxisId="left" y={0} stroke="#9ca3af" />
            <ReferenceLine yAxisId="right" y={0} stroke="#9ca3af" strokeDasharray="4 2" />
            {breakEvenYear && (
              <ReferenceLine yAxisId="right" x={`${breakEvenYear}年`} stroke="#22c55e" strokeDasharray="4 4"
                label={{ value: "回収", position: "top", fontSize: 10, fill: "#22c55e" }} />
            )}
            <Bar yAxisId="left" dataKey="税引後CF" maxBarSize={20} radius={[2, 2, 0, 0]}>
              {data.map((entry, index) => (
                <Cell key={index} fill={entry.isDeadCrossZone ? "#fca5a5" : "#60a5fa"} />
              ))}
            </Bar>
            <Line yAxisId="right" type="monotone" dataKey="累積CF" stroke="#f59e0b" strokeWidth={2} dot={false} />
          </ComposedChart>
        </ResponsiveContainer>
        <p className="mt-1 text-xs text-muted-foreground text-right">
          ※赤色の棒はデッドクロスゾーン（元金返済 &gt; 減価償却費）
        </p>

        {/* 出口戦略サマリー */}
        <div className="mt-4 grid grid-cols-3 gap-3 rounded-md border bg-muted/30 p-3">
          <div className="text-center">
            <p className="text-xs text-muted-foreground">売却価格（NOI基準）</p>
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

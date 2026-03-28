"use client";
import React from "react";
import {
  LineChart,
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
import { Badge } from "@/components/ui/badge";
import type { InvestmentResult } from "@/types/investment";
import { formatMan } from "@/lib/utils";
import { Skull, ShieldCheck } from "lucide-react";

interface Props {
  result: InvestmentResult;
}

export function DeadCrossChart({ result }: Props) {
  const { deadCrossYear, yearlyResults } = result;

  // 35年分に絞ってグラフ表示
  const data = yearlyResults.slice(0, 35).map((y) => ({
    year: `${y.year}年`,
    元金返済: Math.round(y.annualPrincipal / 10_000),
    減価償却費: Math.round(y.annualDepreciation / 10_000),
  }));

  const hasDeadCross = deadCrossYear > 0;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            {hasDeadCross ? (
              <Skull className="h-5 w-5 text-red-500" />
            ) : (
              <ShieldCheck className="h-5 w-5 text-green-500" />
            )}
            デッドクロス予測
          </CardTitle>
          {hasDeadCross ? (
            <Badge variant="danger">{deadCrossYear}年目にデッドクロス</Badge>
          ) : (
            <Badge variant="success">デッドクロスなし</Badge>
          )}
        </div>
        <p className="text-xs text-muted-foreground mt-1">
          元金返済額が減価償却費を上回ると「帳簿上は黒字でも手元資金が枯渇」するリスクが高まります
        </p>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={280}>
          <LineChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis
              dataKey="year"
              tick={{ fontSize: 11 }}
              interval={4}
            />
            <YAxis
              tickFormatter={(v) => `${v}万`}
              tick={{ fontSize: 11 }}
              width={50}
            />
            <Tooltip
              formatter={(value: number, name: string) => [`${value}万円`, name]}
              labelStyle={{ fontWeight: "bold" }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="元金返済"
              stroke="#ef4444"
              strokeWidth={2}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="減価償却費"
              stroke="#3b82f6"
              strokeWidth={2}
              dot={false}
              strokeDasharray="5 5"
            />
            {hasDeadCross && (
              <ReferenceLine
                x={`${deadCrossYear}年`}
                stroke="#f97316"
                strokeWidth={2}
                label={{
                  value: `デッドクロス (${deadCrossYear}年目)`,
                  position: "top",
                  fontSize: 11,
                  fill: "#f97316",
                }}
              />
            )}
          </LineChart>
        </ResponsiveContainer>

        {hasDeadCross && (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm">
            <p className="font-semibold text-red-800">
              ⚠ {deadCrossYear}年目以降、所得税の負担が実態より重くなります
            </p>
            <p className="mt-1 text-red-700 text-xs">
              {deadCrossYear}年目の元金返済：
              <strong>{formatMan(yearlyResults[deadCrossYear - 1]?.annualPrincipal ?? 0)}</strong>
              　減価償却費：
              <strong>{formatMan(yearlyResults[deadCrossYear - 1]?.annualDepreciation ?? 0)}</strong>
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

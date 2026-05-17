"use client";
import React, { useMemo } from "react";
import {
  ComposedChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ReferenceLine,
  ReferenceArea,
  ResponsiveContainer,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { CriticalError, InvestmentResult } from "@/types/investment";
import { formatMan } from "@/lib/utils";
import { chartColors } from "@/lib/chartColors";
import { Skull, ShieldCheck } from "lucide-react";
import { TermTooltip } from "@/components/ui/TermTooltip";
import { useResponsiveChart } from "@/lib/useChartHeight";

const DEADCROSS_EARLY_CODE = "DEADCROSS_EARLY" as const;

const ADVICE_ITEMS = [
  "減価償却が大きい築浅物件への切り替え",
  "法定耐用年数前の売却（デッドクロス前の出口）",
  "法人化による損益通算",
] as const;

interface AdviceGuideProps {
  deadCrossYear: number;
  criticalErrors: CriticalError[];
}

function DeadCrossAdviceGuide({ deadCrossYear, criticalErrors }: AdviceGuideProps) {
  const isEarlyWarning = criticalErrors.some((e) => e.code === DEADCROSS_EARLY_CODE);

  return (
    <div
      role="note"
      aria-label="デッドクロス対策ガイド"
      className="mt-4 rounded-md border border-orange-200 bg-orange-50 p-4"
    >
      <div className="flex items-center gap-2 mb-2">
        <p className="text-sm font-semibold text-orange-900">対策ガイド</p>
        {isEarlyWarning && <Badge variant="danger">早期警告</Badge>}
      </div>
      <p className="text-sm text-orange-800 mb-3">
        保有{deadCrossYear}年目から税負担が増加します。以下の対策を検討してください：
      </p>
      <ul className="space-y-1">
        {ADVICE_ITEMS.map((item, i) => (
          <li key={i} className="flex items-start gap-2 text-sm text-orange-700">
            <span className="mt-0.5 shrink-0">•</span>
            <span>{item}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

interface Props {
  result: InvestmentResult;
}

function DeadCrossChart({ result }: Props) {
  const { deadCrossYear, yearlyResults } = result;
  const { height: chartHeight, isMobile } = useResponsiveChart(200, 240, 280);

  const data = useMemo(
    () =>
      yearlyResults.slice(0, 35).map((y) => ({
        year: `${y.year}年`,
        元金返済: Math.round(y.annualPrincipal / 10_000),
        減価償却費: Math.round(y.annualDepreciation / 10_000),
        isDeadCrossZone: y.isInDeadCrossZone,
      })),
    [yearlyResults]
  );

  const hasDeadCross = deadCrossYear > 0 && deadCrossYear <= 35;

  // デッドクロスゾーンの終了年（ローン完済またはデータ終端）
  const deadCrossEndYear = useMemo(
    () =>
      hasDeadCross
        ? ([...yearlyResults.slice(0, 35)].reverse().find((y) => y.isInDeadCrossZone)?.year ??
          deadCrossYear)
        : null,
    [hasDeadCross, yearlyResults, deadCrossYear]
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle data-testid="dead-cross-chart-heading" className="flex items-center gap-2">
            {hasDeadCross ? (
              <Skull className="h-5 w-5 text-red-500" />
            ) : (
              <ShieldCheck className="h-5 w-5 text-green-500" />
            )}
            <TermTooltip term="deadCross">デッドクロス</TermTooltip>予測
          </CardTitle>
          {hasDeadCross ? (
            <Badge variant="danger">
              {deadCrossYear}年目〜<TermTooltip term="deadCross">デッドクロス</TermTooltip>ゾーン
            </Badge>
          ) : (
            <Badge variant="success">
              <TermTooltip term="deadCross">デッドクロス</TermTooltip>なし（35年以内）
            </Badge>
          )}
        </div>
        <p className="text-xs text-muted-foreground mt-1">
          元金返済額 &gt;
          減価償却費となるゾーンでは、帳簿上黒字でも実質的な税負担が増加します（黒字倒産リスク）
        </p>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={chartHeight}>
          <ComposedChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="year" tick={{ fontSize: 11 }} interval={4} />
            <YAxis
              tickFormatter={(v) => `${v}万`}
              tick={{ fontSize: 11 }}
              width={isMobile ? 32 : 50}
            />
            <Tooltip
              formatter={(value, name) => [`${value}万円`, name]}
              labelStyle={{ fontWeight: "bold" }}
            />
            <Legend />

            {/* デッドクロスゾーン全体をハイライト（ISSUE-07） */}
            {hasDeadCross && deadCrossEndYear && (
              <ReferenceArea
                x1={`${deadCrossYear}年`}
                x2={`${deadCrossEndYear}年`}
                fill={chartColors.danger}
                fillOpacity={0.15}
              />
            )}

            <Line
              type="monotone"
              dataKey="元金返済"
              stroke={chartColors.danger}
              strokeWidth={2}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="減価償却費"
              stroke={chartColors.primary}
              strokeWidth={2}
              dot={false}
              strokeDasharray="5 5"
            />

            {hasDeadCross && (
              <ReferenceLine
                x={`${deadCrossYear}年`}
                stroke={chartColors.warning}
                strokeWidth={2}
                label={{
                  value: `開始(${deadCrossYear}年)`,
                  position: "top",
                  fontSize: 11,
                  fill: chartColors.warning,
                }}
              />
            )}
          </ComposedChart>
        </ResponsiveContainer>

        {hasDeadCross && (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm">
            <p className="font-semibold text-red-800">
              ⚠ {deadCrossYear}年目以降、所得税の実質負担が増加します
            </p>
            <p className="mt-1 text-red-700 text-xs">
              {deadCrossYear}年目 — 元金返済：
              <strong>{formatMan(yearlyResults[deadCrossYear - 1]?.annualPrincipal ?? 0)}</strong>
              ／減価償却費：
              <strong>
                {formatMan(yearlyResults[deadCrossYear - 1]?.annualDepreciation ?? 0)}
              </strong>
            </p>
          </div>
        )}

        {hasDeadCross && (
          <DeadCrossAdviceGuide
            deadCrossYear={deadCrossYear}
            criticalErrors={result.criticalErrors}
          />
        )}
      </CardContent>
    </Card>
  );
}

export default React.memo(DeadCrossChart);

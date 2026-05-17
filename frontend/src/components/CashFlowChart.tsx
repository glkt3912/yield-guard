"use client";
import React, { useMemo } from "react";
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
  Cell,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type { InvestmentResult } from "@/types/investment";
import { formatMan } from "@/lib/utils";
import { chartColors } from "@/lib/chartColors";
import { TrendingUp } from "lucide-react";
import { TermTooltip } from "@/components/ui/TermTooltip";
import { useResponsiveChart } from "@/lib/useChartHeight";

interface Props {
  result: InvestmentResult;
  /** 自己資金（総投資額 - ローン金額）。投資回収年の計算に使用 */
  equityInvested: number;
}

function CashFlowChart({ result, equityInvested }: Props) {
  const { yearlyResults, exitTotalEquity, exitSalePrice, exitNetProceeds } = result;
  const { height: chartHeight, isMobile } = useResponsiveChart(220, 260, 300);

  const data = useMemo(
    () =>
      yearlyResults.slice(0, 35).map((y) => ({
        year: `${y.year}年`,
        税引後CF: Math.round(y.afterTaxCashFlow / 10_000),
        // 自己資金を初期コストとして加算した累積CF（ISSUE-22）
        累積CF: Math.round((y.cumulativeCashFlow - equityInvested) / 10_000),
        isDeadCrossZone: y.isInDeadCrossZone,
        effectiveRate: y.effectiveRate,
        // capex は NOI から除外（バックエンドの calcDSCR と同定義: NOI = annualRent - annualExpenses）
        DSCR:
          y.annualLoanPayment > 0
            ? Math.round(((y.annualRent - y.annualExpenses) / y.annualLoanPayment) * 100) / 100
            : null,
      })),
    [yearlyResults, equityInvested]
  );

  // 自己資金を回収した年（累積CF - 自己資金 >= 0）
  const breakEvenYear = useMemo(
    () => yearlyResults.find((y) => y.cumulativeCashFlow - equityInvested >= 0)?.year ?? null,
    [yearlyResults, equityInvested]
  );

  // 金利が変化した年のリスト（2年目以降で前年から変化した年）
  const rateChangeYears = useMemo(() => {
    const changes: { year: string; rate: number }[] = [];
    for (let i = 1; i < Math.min(yearlyResults.length, 35); i++) {
      if (yearlyResults[i].effectiveRate !== yearlyResults[i - 1].effectiveRate) {
        changes.push({
          year: `${yearlyResults[i].year}年`,
          rate: yearlyResults[i].effectiveRate,
        });
      }
    }
    return changes;
  }, [yearlyResults]);

  return (
    <Card>
      <CardHeader>
        <CardTitle data-testid="cashflow-chart-heading" className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5 text-primary" />
          キャッシュフロー推移（35年）
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          累積CFは自己資金（{formatMan(equityInvested)}）を初期コストとして控除しています。
          {breakEvenYear ? (
            <>
              {" "}
              自己資金回収：
              <span className="font-semibold text-green-600">{breakEvenYear}年目</span>
            </>
          ) : (
            " 35年以内に自己資金を回収できない見込みです"
          )}
        </p>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={chartHeight}>
          <ComposedChart
            data={data}
            margin={{ top: 8, right: isMobile ? 60 : 80, left: 0, bottom: 0 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="year" tick={{ fontSize: 11 }} interval={4} />
            <YAxis
              yAxisId="left"
              tickFormatter={(v) => `${v}万`}
              tick={{ fontSize: 11 }}
              width={isMobile ? 36 : 55}
            />
            <YAxis
              yAxisId="right"
              orientation="right"
              tickFormatter={(v) => `${v}万`}
              tick={{ fontSize: 11 }}
              width={isMobile ? 36 : 55}
            />
            <YAxis
              yAxisId="dscr"
              orientation="right"
              domain={[0, 3]}
              tickFormatter={(v: number) => v.toFixed(1)}
              tick={{ fontSize: 10, fill: chartColors.secondary }}
              width={isMobile ? 28 : 36}
              dx={isMobile ? 28 : 40}
              label={
                isMobile
                  ? undefined
                  : {
                      value: "DSCR",
                      angle: 90,
                      position: "insideRight",
                      fontSize: 10,
                      fill: chartColors.secondary,
                      dx: 12,
                    }
              }
            />
            <Tooltip
              formatter={(value, name) => {
                if (name === "DSCR") {
                  const n = typeof value === "number" ? value : null;
                  return [n == null ? "―" : n.toFixed(2), name];
                }
                return [`${value}万円`, name];
              }}
              labelStyle={{ fontWeight: "bold" }}
            />
            <Legend />
            <ReferenceLine yAxisId="left" y={0} stroke={chartColors.muted} />
            <ReferenceLine yAxisId="right" y={0} stroke={chartColors.muted} strokeDasharray="4 2" />
            {breakEvenYear && (
              <ReferenceLine
                yAxisId="right"
                x={`${breakEvenYear}年`}
                stroke={chartColors.success}
                strokeDasharray="4 4"
                label={{ value: "回収", position: "top", fontSize: 10, fill: chartColors.success }}
              />
            )}
            {rateChangeYears.map(({ year, rate }) => (
              <ReferenceLine
                key={year}
                yAxisId="left"
                x={year}
                stroke={chartColors.warning}
                strokeDasharray="4 3"
                label={{
                  value: `${(rate * 100).toFixed(2)}%`,
                  position: "insideTopRight",
                  fontSize: 9,
                  fill: chartColors.warning,
                }}
              />
            ))}
            <Bar yAxisId="left" dataKey="税引後CF" maxBarSize={20} radius={[2, 2, 0, 0]}>
              {data.map((entry, index) => (
                <Cell key={index} fill={entry.isDeadCrossZone ? chartColors.danger : chartColors.primary} />
              ))}
            </Bar>
            <Line
              yAxisId="right"
              type="monotone"
              dataKey="累積CF"
              stroke={chartColors.warning}
              strokeWidth={2}
              dot={false}
            />
            <ReferenceLine
              yAxisId="dscr"
              y={1.0}
              stroke={chartColors.danger}
              strokeDasharray="3 3"
              label={{
                value: "危険",
                position: "insideTopRight",
                fontSize: 9,
                fill: chartColors.danger,
              }}
            />
            <ReferenceLine
              yAxisId="dscr"
              y={1.2}
              stroke={chartColors.success}
              strokeDasharray="3 3"
              label={{
                value: "安全",
                position: "insideTopRight",
                fontSize: 9,
                fill: chartColors.success,
              }}
            />
            <Line
              yAxisId="dscr"
              type="monotone"
              dataKey="DSCR"
              stroke={chartColors.secondary}
              strokeWidth={2}
              dot={false}
              connectNulls
            />
          </ComposedChart>
        </ResponsiveContainer>
        <p className="mt-1 text-xs text-muted-foreground text-right">
          ※赤色の棒はデッドクロスゾーン（元金返済 &gt; 減価償却費）
          {rateChangeYears.length > 0 && "　※橙色の縦線は金利変更年"}
        </p>
        {isMobile ? (
          <details className="mt-2">
            <summary className="cursor-pointer text-xs text-purple-600 font-medium select-none">
              DSCR（借入金償還余裕率）の説明
            </summary>
            <p className="mt-1 text-xs text-muted-foreground">
              ※紫線はDSCR（借入金償還余裕率）。1.0未満は危険、1.2以上が安全とされる
            </p>
          </details>
        ) : (
          <p className="mt-1 text-xs text-muted-foreground">
            ※紫線はDSCR（借入金償還余裕率）。1.0未満は危険、1.2以上が安全とされる
          </p>
        )}

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
            <p
              className={`font-bold text-sm ${exitTotalEquity >= 0 ? "text-green-600" : "text-red-600"}`}
            >
              {formatMan(exitTotalEquity)}
            </p>
          </div>
        </div>

        {/* IRR / NPV */}
        <div className="mt-3 grid grid-cols-2 gap-3 rounded-md border bg-muted/30 p-3">
          <div className="text-center">
            <p className="text-xs text-muted-foreground">
              <TermTooltip term="irr">IRR（内部収益率）</TermTooltip>
            </p>
            <p
              className={`text-sm font-bold ${result.irr != null ? (result.irr >= 0 ? "text-green-600" : "text-red-600") : "text-muted-foreground"}`}
            >
              {result.irr != null ? `${(result.irr * 100).toFixed(2)}%` : "―"}
            </p>
          </div>
          <div className="text-center">
            <p className="text-xs text-muted-foreground">
              <TermTooltip term="npv">NPV（正味現在価値）</TermTooltip>
            </p>
            <p
              className={`text-sm font-bold ${result.npv >= 0 ? "text-green-600" : "text-red-600"}`}
            >
              {formatMan(result.npv)}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default React.memo(CashFlowChart);

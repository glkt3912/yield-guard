"use client";
import React, { useState } from "react";
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import type { InvestmentResult, InvestmentInput, TaxSimRow } from "@/types/investment";
import { formatMan } from "@/lib/utils";
import { chartColors } from "@/lib/chartColors";
import { TrendingDown, TrendingUp, Building2, User } from "lucide-react";

interface Props {
  result: InvestmentResult;
  input: InvestmentInput;
}

type PanelTab = "carryover" | "ownership";

export function TaxSimulationPanel({ result, input }: Props) {
  const [tab, setTab] = useState<PanelTab>("carryover");

  if (!result.taxSimulation) return null;
  const { salaryLossCarryover, ownershipComparison } = result.taxSimulation;

  const totalSaving = salaryLossCarryover.totalTaxSaving;
  const isSaving = totalSaving > 0;

  const carryoverChartData = salaryLossCarryover.yearlyRows.map((row: TaxSimRow) => ({
    year: `${row.year}年`,
    節税額: row.taxDifference > 0 ? Math.round(row.taxDifference / 10_000) : 0,
    増税額: row.taxDifference < 0 ? Math.round(-row.taxDifference / 10_000) : 0,
  }));

  const { individual, corporate, breakevenYear } = ownershipComparison;
  const holdingYears = input.holdingYears || 10;
  const safeLen = Math.min(
    holdingYears,
    individual.cumulativeBurden.length,
    corporate.cumulativeBurden.length
  );
  const ownershipChartData = Array.from({ length: safeLen }, (_, i) => ({
    year: `${i + 1}年`,
    個人: Math.round(individual.cumulativeBurden[i] / 10_000),
    法人: Math.round(corporate.cumulativeBurden[i] / 10_000),
  }));

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base flex items-center gap-2">
          税務シミュレーション
          <Badge variant="outline" className="text-xs font-normal">
            給与年収 {formatMan(salaryLossCarryover.salaryIncomeYen)}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* タブ切り替え */}
        <div className="flex gap-1 border rounded-lg p-1 bg-muted/30">
          <button
            className={`flex-1 text-xs py-1.5 px-2 rounded transition-colors ${
              tab === "carryover"
                ? "bg-background shadow-sm font-medium"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setTab("carryover")}
          >
            損益通算（節税効果）
          </button>
          <button
            className={`flex-1 text-xs py-1.5 px-2 rounded transition-colors ${
              tab === "ownership"
                ? "bg-background shadow-sm font-medium"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setTab("ownership")}
          >
            個人 vs 法人
          </button>
        </div>

        {tab === "carryover" && (
          <div className="space-y-4">
            {/* サマリー */}
            <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/30">
              {isSaving ? (
                <TrendingDown className="h-5 w-5 text-chart-success shrink-0" />
              ) : (
                <TrendingUp className="h-5 w-5 text-destructive shrink-0" />
              )}
              <div>
                <p className="text-xs text-muted-foreground">
                  {holdingYears}年間の損益通算
                  {isSaving ? "節税" : "増税"}合計
                </p>
                <p
                  className={`text-lg font-bold ${isSaving ? "text-chart-success" : "text-destructive"}`}
                >
                  {isSaving ? "▼" : "▲"} {formatMan(Math.abs(totalSaving))}
                </p>
              </div>
              <div className="ml-auto text-right">
                <p className="text-xs text-muted-foreground">給与のみの税額（年）</p>
                <p className="text-sm font-medium">
                  {formatMan(salaryLossCarryover.baselineSalaryTax)}
                </p>
              </div>
            </div>

            {/* 棒グラフ */}
            <div>
              <p className="text-xs text-muted-foreground mb-2">年別の節税額／増税額（万円）</p>
              <ResponsiveContainer width="100%" height={180}>
                <BarChart
                  data={carryoverChartData}
                  margin={{ top: 4, right: 8, left: 0, bottom: 0 }}
                >
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="year" tick={{ fontSize: 10 }} />
                  <YAxis tick={{ fontSize: 10 }} unit="万" />
                  <Tooltip formatter={(v, name) => [`${v}万円`, name]} />
                  <Legend iconSize={10} wrapperStyle={{ fontSize: 11 }} />
                  <ReferenceLine y={0} stroke="hsl(var(--border))" />
                  <Bar dataKey="節税額" fill={chartColors.success} radius={[2, 2, 0, 0]} />
                  <Bar dataKey="増税額" fill={chartColors.danger} radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* 年別テーブル（折りたたみ） */}
            <details className="text-xs">
              <summary className="cursor-pointer text-muted-foreground hover:text-foreground py-1">
                年別明細を表示
              </summary>
              <div className="mt-2 overflow-x-auto">
                <table className="w-full text-xs border-collapse">
                  <thead>
                    <tr className="border-b text-muted-foreground">
                      <th className="text-left py-1 pr-3">年</th>
                      <th className="text-right py-1 pr-3">不動産課税所得</th>
                      <th className="text-right py-1 pr-3">合算課税所得</th>
                      <th className="text-right py-1">節税/増税額</th>
                    </tr>
                  </thead>
                  <tbody>
                    {salaryLossCarryover.yearlyRows.map((row: TaxSimRow) => (
                      <tr key={row.year} className="border-b border-border/40">
                        <td className="py-1 pr-3">{row.year}年目</td>
                        <td
                          className={`text-right py-1 pr-3 ${row.reTaxableIncome < 0 ? "text-chart-success" : ""}`}
                        >
                          {formatMan(row.reTaxableIncome)}
                        </td>
                        <td className="text-right py-1 pr-3">{formatMan(row.combinedIncome)}</td>
                        <td
                          className={`text-right py-1 font-medium ${
                            row.taxDifference > 0
                              ? "text-chart-success"
                              : row.taxDifference < 0
                                ? "text-destructive"
                                : ""
                          }`}
                        >
                          {row.taxDifference > 0 ? "▼" : row.taxDifference < 0 ? "▲" : "―"}
                          {formatMan(Math.abs(row.taxDifference))}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </details>
          </div>
        )}

        {tab === "ownership" && (
          <div className="space-y-4">
            {/* ブレークイーヤーバッジ */}
            <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/30">
              <Building2 className="h-5 w-5 text-primary shrink-0" />
              <div>
                <p className="text-xs text-muted-foreground">法人保有が有利になる年</p>
                {breakevenYear === -1 ? (
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">保有期間内に逆転なし</Badge>
                    <span className="text-xs text-muted-foreground">個人保有が有利</span>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <Badge className="bg-primary text-primary-foreground">
                      {breakevenYear}年目〜
                    </Badge>
                    <span className="text-xs text-muted-foreground">以降は法人が有利</span>
                  </div>
                )}
              </div>
            </div>

            {/* 累積税負担グラフ */}
            <div>
              <p className="text-xs text-muted-foreground mb-2">
                累積税負担の推移（万円・譲渡税含む）
              </p>
              <ResponsiveContainer width="100%" height={200}>
                <LineChart
                  data={ownershipChartData}
                  margin={{ top: 4, right: 8, left: 0, bottom: 0 }}
                >
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="year" tick={{ fontSize: 10 }} />
                  <YAxis tick={{ fontSize: 10 }} unit="万" />
                  <Tooltip formatter={(v, name) => [`${v}万円`, name]} />
                  <Legend iconSize={10} wrapperStyle={{ fontSize: 11 }} />
                  <Line
                    type="monotone"
                    dataKey="個人"
                    stroke={chartColors.primary}
                    strokeWidth={2}
                    dot={false}
                  />
                  <Line
                    type="monotone"
                    dataKey="法人"
                    stroke={chartColors.secondary}
                    strokeWidth={2}
                    dot={false}
                    strokeDasharray="5 5"
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>

            {/* サマリー表 */}
            <div className="grid grid-cols-2 gap-3">
              {[
                { scenario: individual, icon: <User className="h-4 w-4" /> },
                { scenario: corporate, icon: <Building2 className="h-4 w-4" /> },
              ].map(({ scenario, icon }) => (
                <div key={scenario.label} className="rounded-lg border p-3 space-y-1.5">
                  <div className="flex items-center gap-1.5 text-sm font-medium">
                    {icon}
                    {scenario.label}
                  </div>
                  <div className="text-xs space-y-0.5">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">保有中の税合計</span>
                      <span>{formatMan(scenario.totalTaxBurden - scenario.transferTax)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">売却時の税</span>
                      <span>{formatMan(scenario.transferTax)}</span>
                    </div>
                    <div className="flex justify-between font-medium border-t pt-1 mt-1">
                      <span>合計税負担</span>
                      <span>{formatMan(scenario.totalTaxBurden)}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        <Alert>
          <AlertDescription className="text-xs text-muted-foreground">
            概算値です。個人事業・法人設立コスト・青色申告・消費税は含みません。
            実際の申告・意思決定は税理士にご相談ください。
          </AlertDescription>
        </Alert>
      </CardContent>
    </Card>
  );
}

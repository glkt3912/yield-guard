"use client";
import React, { useState } from "react";
import {
  LineChart,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  YAxis,
  Tooltip,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { analyze } from "@/lib/api";
import type { InvestmentInput, InvestmentResult } from "@/types/investment";
import { toMan, toPct, fromPct } from "@/lib/utils";
import { Plus, Trash2, BarChart3 } from "lucide-react";

interface LoanScenario {
  name: string;
  annualLoanRate: number; // 年利（小数）
  loanYears: number;
  ltv: number; // LTV（小数）
  loanFeeRate: number; // 諸費用率（小数）
}

const DEFAULT_SCENARIO = (index: number): LoanScenario => ({
  name: `シナリオ ${index + 1}`,
  annualLoanRate: 0.015,
  loanYears: 35,
  ltv: 0.7,
  loanFeeRate: 0.02,
});

interface Props {
  baseInput: InvestmentInput;
}

export function LoanComparePanel({ baseInput }: Props) {
  const [scenarios, setScenarios] = useState<LoanScenario[]>([
    DEFAULT_SCENARIO(0),
    DEFAULT_SCENARIO(1),
  ]);
  const [results, setResults] = useState<(InvestmentResult | null)[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const addScenario = () => {
    if (scenarios.length >= 4) return;
    setScenarios((prev) => [...prev, DEFAULT_SCENARIO(prev.length)]);
  };

  const removeScenario = (i: number) => {
    if (scenarios.length <= 1) return;
    setScenarios((prev) => prev.filter((_, idx) => idx !== i));
    setResults((prev) => prev.filter((_, idx) => idx !== i));
  };

  const updateScenario = (i: number, patch: Partial<LoanScenario>) => {
    setScenarios((prev) =>
      prev.map((s, idx) => (idx === i ? { ...s, ...patch } : s))
    );
  };

  const handleCompare = async () => {
    setLoading(true);
    setError(null);
    try {
      const propertyValue = baseInput.landPrice + baseInput.buildingCost;
      const inputs: InvestmentInput[] = scenarios.map((sc) => ({
        ...baseInput,
        loanAmount: propertyValue * sc.ltv,
        annualLoanRate: sc.annualLoanRate,
        loanYears: sc.loanYears,
        loanFeeRate: sc.loanFeeRate,
      }));
      const fetched = await Promise.all(inputs.map((inp) => analyze(inp)));
      setResults(fetched);
    } catch (e) {
      setError(e instanceof Error ? e.message : "比較に失敗しました");
    } finally {
      setLoading(false);
    }
  };

  // DSCR推移データ（年次結果から HoldingYears 分）
  const dscrSeriesForResult = (res: InvestmentResult): { year: number; dscr: number }[] => {
    const holdingYears = baseInput.holdingYears ?? 10;
    return res.yearlyResults
      .filter((yr) => yr.year <= holdingYears)
      .map((yr) => {
        const noi = yr.annualRent - yr.annualExpenses;
        const dscr = yr.annualLoanPayment > 0 ? noi / yr.annualLoanPayment : 0;
        return { year: yr.year, dscr: parseFloat(dscr.toFixed(3)) };
      });
  };

  const propertyValue = baseInput.landPrice + baseInput.buildingCost;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BarChart3 className="h-5 w-5 text-primary" />
          複数融資条件の横並び比較
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* シナリオ入力フォーム */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {scenarios.map((sc, i) => (
            <div key={i} className="rounded-lg border bg-muted/20 p-3 space-y-2">
              <div className="flex items-center justify-between">
                <Input
                  label="シナリオ名"
                  type="text"
                  value={sc.name}
                  onChange={(e) => updateScenario(i, { name: e.target.value })}
                />
                <button
                  onClick={() => removeScenario(i)}
                  disabled={scenarios.length <= 1}
                  className="ml-2 mt-5 text-muted-foreground hover:text-destructive disabled:opacity-30 disabled:cursor-not-allowed"
                  aria-label="削除"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
              <Input
                label="金利"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="0.1"
                value={toPct(sc.annualLoanRate, 1)}
                onChange={(e) =>
                  updateScenario(i, { annualLoanRate: fromPct(e.target.value) })
                }
              />
              <Input
                label="融資期間"
                type="number"
                inputMode="numeric"
                suffix="年"
                step="1"
                value={sc.loanYears}
                onChange={(e) =>
                  updateScenario(i, { loanYears: parseInt(e.target.value, 10) || 35 })
                }
              />
              <Input
                label="LTV"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="5"
                value={toPct(sc.ltv, 0)}
                onChange={(e) =>
                  updateScenario(i, { ltv: fromPct(e.target.value) })
                }
              />
              <Input
                label="諸費用率"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="0.5"
                value={toPct(sc.loanFeeRate, 1)}
                onChange={(e) =>
                  updateScenario(i, { loanFeeRate: fromPct(e.target.value) })
                }
              />
            </div>
          ))}
        </div>

        {/* シナリオ追加 / 比較実行 */}
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={addScenario}
            disabled={scenarios.length >= 4}
          >
            <Plus className="mr-1 h-4 w-4" />
            シナリオ追加
          </Button>
          <Button size="sm" onClick={handleCompare} loading={loading}>
            比較実行
          </Button>
        </div>

        {error && (
          <p className="text-sm text-destructive">{error}</p>
        )}

        {/* 比較結果テーブル */}
        {results.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr className="border-b bg-muted/40">
                  <th className="px-3 py-2 text-left font-medium">指標</th>
                  {scenarios.slice(0, results.length).map((sc, i) => (
                    <th key={i} className="px-3 py-2 text-center font-medium">
                      {sc.name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y">
                {/* 月返済額 */}
                <tr>
                  <td className="px-3 py-2 text-muted-foreground">月返済額</td>
                  {results.map((res, i) => {
                    if (!res) return <td key={i} className="px-3 py-2 text-center">—</td>;
                    const monthlyPayment =
                      res.yearlyResults[0]?.annualLoanPayment != null
                        ? res.yearlyResults[0].annualLoanPayment / 12
                        : 0;
                    return (
                      <td key={i} className="px-3 py-2 text-center">
                        {toMan(monthlyPayment)}万円
                      </td>
                    );
                  })}
                </tr>
                {/* DSCR（初年度） */}
                <tr>
                  <td className="px-3 py-2 text-muted-foreground">
                    DSCR（初年度）
                    <span className="ml-1 text-xs text-muted-foreground">参考値</span>
                  </td>
                  {results.map((res, i) => {
                    if (!res) return <td key={i} className="px-3 py-2 text-center">—</td>;
                    const dscr = res.dscr;
                    const colorClass =
                      dscr >= 1.2
                        ? "text-green-600 font-semibold"
                        : "text-red-600 font-semibold";
                    return (
                      <td key={i} className="px-3 py-2 text-center">
                        <span className={colorClass}>{dscr.toFixed(2)}</span>
                        {dscr < 1.0 && (
                          <div className="text-xs text-amber-600 mt-0.5">
                            ⚠️ 参考: DSCR &lt; 1.0 は審査通過を保証しません
                          </div>
                        )}
                      </td>
                    );
                  })}
                </tr>
                {/* DSCR推移ミニグラフ */}
                <tr>
                  <td className="px-3 py-2 text-muted-foreground">DSCR推移</td>
                  {results.map((res, i) => {
                    if (!res) return <td key={i} className="px-3 py-2 text-center">—</td>;
                    const data = dscrSeriesForResult(res);
                    return (
                      <td key={i} className="px-3 py-2">
                        <ResponsiveContainer width="100%" height={80}>
                          <LineChart data={data} margin={{ top: 4, right: 4, bottom: 4, left: 0 }}>
                            <YAxis domain={["auto", "auto"]} hide />
                            <Tooltip
                              formatter={(v: number) => [v.toFixed(2), "DSCR"]}
                              labelFormatter={(l) => `${l}年目`}
                            />
                            <ReferenceLine y={1.2} stroke="#ef4444" strokeDasharray="3 3" />
                            <Line
                              type="monotone"
                              dataKey="dscr"
                              dot={false}
                              stroke="#3b82f6"
                              strokeWidth={1.5}
                            />
                          </LineChart>
                        </ResponsiveContainer>
                      </td>
                    );
                  })}
                </tr>
                {/* デッドクロス年 */}
                <tr>
                  <td className="px-3 py-2 text-muted-foreground">デッドクロス年</td>
                  {results.map((res, i) => {
                    if (!res) return <td key={i} className="px-3 py-2 text-center">—</td>;
                    return (
                      <td key={i} className="px-3 py-2 text-center">
                        {res.deadCrossYear > 0 ? `${res.deadCrossYear}年目` : "なし"}
                      </td>
                    );
                  })}
                </tr>
                {/* 自己資金必要額 */}
                <tr>
                  <td className="px-3 py-2 text-muted-foreground">自己資金必要額</td>
                  {scenarios.slice(0, results.length).map((sc, i) => {
                    const equity = propertyValue * (1 - sc.ltv);
                    return (
                      <td key={i} className="px-3 py-2 text-center">
                        {toMan(equity)}万円
                      </td>
                    );
                  })}
                </tr>
                {/* 総支払利息 */}
                <tr>
                  <td className="px-3 py-2 text-muted-foreground">総支払利息</td>
                  {results.map((res, i) => {
                    if (!res) return <td key={i} className="px-3 py-2 text-center">—</td>;
                    return (
                      <td key={i} className="px-3 py-2 text-center">
                        {toMan(res.totalInterest)}万円
                      </td>
                    );
                  })}
                </tr>
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

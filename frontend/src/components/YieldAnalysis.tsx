"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { InvestmentResult } from "@/types/investment";
import { formatMan, formatPct, formatYen } from "@/lib/utils";
import { TrendingUp, TrendingDown, AlertTriangle, CheckCircle } from "lucide-react";

interface Props {
  result: InvestmentResult;
}

export function YieldAnalysis({ result }: Props) {
  const yieldPct = result.grossYield * 100;
  const netYieldPct = result.netYield * 100;
  const isGood = result.isAbove8Percent;

  return (
    <div className="space-y-4">
      {/* メイン利回り表示 */}
      <Card className={`border-2 ${isGood ? "border-green-400" : "border-red-400"}`}>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">表面利回り</p>
              <div className="flex items-end gap-2">
                <span
                  className={`text-5xl font-bold ${isGood ? "text-green-600" : "text-red-600"}`}
                >
                  {yieldPct.toFixed(2)}
                </span>
                <span className="mb-2 text-2xl font-semibold text-muted-foreground">%</span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                実質利回り：{netYieldPct.toFixed(2)}%
              </p>
            </div>
            <div className="flex flex-col items-center gap-2">
              {isGood ? (
                <>
                  <CheckCircle className="h-12 w-12 text-green-500" />
                  <Badge variant="success">8%超え ✓</Badge>
                </>
              ) : (
                <>
                  <AlertTriangle className="h-12 w-12 text-red-500" />
                  <Badge variant="danger">8%未満 ✗</Badge>
                </>
              )}
            </div>
          </div>

          {/* 8%境界線インジケーター */}
          <div className="mt-4">
            <div className="flex justify-between text-xs text-muted-foreground mb-1">
              <span>0%</span>
              <span className="font-semibold text-orange-500">目標 8%</span>
              <span>16%+</span>
            </div>
            <div className="relative h-3 rounded-full bg-muted overflow-hidden">
              <div
                className="absolute inset-y-0 left-0 rounded-full bg-gradient-to-r from-red-400 via-yellow-400 to-green-400"
                style={{ width: "100%" }}
              />
              {/* 現在値マーカー */}
              <div
                className="absolute top-0 h-full w-1 bg-foreground/80 rounded"
                style={{ left: `${Math.min(yieldPct / 16, 1) * 100}%` }}
              />
              {/* 8%ライン */}
              <div
                className="absolute top-0 h-full w-0.5 bg-orange-500"
                style={{ left: "50%" }}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 総投資額サマリー */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">投資サマリー</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="space-y-2">
            {[
              { label: "総投資額", value: formatMan(result.totalInvestment), highlight: true },
              { label: "　うち諸経費", value: formatYen(result.miscExpenses) },
              { label: "年間賃料収入（想定）", value: formatYen(result.yearlyResults[0]?.annualRent ?? 0) },
              { label: "年間ローン返済", value: formatYen(result.yearlyResults[0]?.annualLoanPayment ?? 0) },
              { label: "1年目税引後CF", value: formatYen(result.yearlyResults[0]?.afterTaxCashFlow ?? 0) },
            ].map(({ label, value, highlight }) => (
              <div key={label} className="flex justify-between text-sm">
                <dt className={`text-muted-foreground ${highlight ? "font-semibold text-foreground" : ""}`}>
                  {label}
                </dt>
                <dd className={`font-medium ${highlight ? "text-primary" : ""}`}>{value}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      {/* 8%未達の場合: 逆算カード */}
      {!isGood && (
        <Card className="border-orange-200 bg-orange-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-orange-800">
              <TrendingDown className="h-5 w-5" />
              8%達成のために必要な改善
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="rounded-md bg-white/70 p-3">
              <p className="text-xs text-muted-foreground">土地価格の削減が必要な額</p>
              <p className="text-xl font-bold text-orange-700">
                ▼ {formatMan(result.requiredLandPriceDrop)}
              </p>
            </div>
            <div className="rounded-md bg-white/70 p-3">
              <p className="text-xs text-muted-foreground">建築費の削減が必要な額</p>
              <p className="text-xl font-bold text-orange-700">
                ▼ {formatMan(result.requiredBuildingCostDrop)}
              </p>
            </div>
            <div className="rounded-md bg-white/70 p-3">
              <p className="text-xs text-muted-foreground">8%達成に必要な月額賃料</p>
              <p className="text-xl font-bold text-orange-700">
                ▲ {formatYen(result.requiredMonthlyRent)}/月
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 8%以上の場合: 余裕度表示 */}
      {isGood && (
        <Card className="border-green-200 bg-green-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-green-800">
              <TrendingUp className="h-5 w-5" />
              8%超え達成！余裕度
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-green-700">
              目標の8%に対して{" "}
              <span className="font-bold">
                +{formatPct(result.grossYield - 0.08)}
              </span>{" "}
              の余裕があります。
            </p>
            <p className="mt-2 text-xs text-muted-foreground">
              賃料が{formatPct((result.grossYield - 0.08) / result.grossYield)}下落しても8%を維持できます。
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

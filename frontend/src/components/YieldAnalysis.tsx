"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { InvestmentInput, InvestmentResult, PopulationForecastResult, StressScenarioResult, YieldScenarios } from "@/types/investment";
import { formatMan, formatPct, formatYen } from "@/lib/utils";
import { TrendingUp, TrendingDown, AlertTriangle, CheckCircle, Users } from "lucide-react";

interface Props {
  result: InvestmentResult;
  input: InvestmentInput;
  populationForecast?: PopulationForecastResult | null;
}

const MAX_YIELD_PCT = 16; // ゲージ上限（8%が中央に来る設計）
const TARGET_PCT = 8;

export function YieldAnalysis({ result, input, populationForecast }: Props) {
  const yieldPct = result.grossYield * 100;
  const netYieldPct = result.netYield * 100;
  const isGood = result.isAbove8Percent;

  // 人口シナリオ用（populationForecast表示のために残す）
  const actualV = input.actualVacancyRate > 0 ? input.actualVacancyRate : input.vacancyRate;

  const stressScenarios: StressScenarioResult[] = result.stressScenarios ?? [];

  const gaugePosition = Math.min(yieldPct / MAX_YIELD_PCT, 1) * 100;
  const targetPosition = (TARGET_PCT / MAX_YIELD_PCT) * 100; // = 50%

  return (
    <div className="space-y-4">
      {/* メイン利回り表示 */}
      <Card className={`border-2 ${isGood ? "border-green-400" : "border-red-400"}`}>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">表面利回り（満室想定年収 / 総投資額）</p>
              <div className="flex items-end gap-2">
                <span className={`text-5xl font-bold ${isGood ? "text-green-600" : "text-red-600"}`}>
                  {yieldPct.toFixed(2)}
                </span>
                <span className="mb-2 text-2xl font-semibold text-muted-foreground">%</span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                実質利回り（空室・経費控除後）：{netYieldPct.toFixed(2)}%
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

          {/* 8%境界線ゲージ（上限16%、8%が中央） */}
          <div className="mt-4">
            <div className="flex justify-between text-xs text-muted-foreground mb-1">
              <span>0%</span>
              <span className="font-semibold text-orange-500">目標 {TARGET_PCT}%</span>
              <span>{MAX_YIELD_PCT}%+</span>
            </div>
            <div className="relative h-3 rounded-full bg-muted overflow-hidden">
              <div className="absolute inset-y-0 left-0 rounded-full bg-gradient-to-r from-red-400 via-yellow-400 to-green-400 w-full" />
              {/* 現在値マーカー */}
              <div className="absolute top-0 h-full w-1 bg-foreground/80 rounded"
                style={{ left: `${gaugePosition}%` }} />
              {/* 8%ライン（スケールと連動） */}
              <div className="absolute top-0 h-full w-0.5 bg-orange-500"
                style={{ left: `${targetPosition}%` }} />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* ストレステストシナリオ比較 */}
      <Card>
        <CardHeader><CardTitle className="text-base">ストレステストシナリオ（銀行融資審査用）</CardTitle></CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="pb-2 text-left font-medium">シナリオ</th>
                  <th className="pb-2 text-right font-medium">金利△</th>
                  <th className="pb-2 text-right font-medium">空室△</th>
                  <th className="pb-2 text-right font-medium">DSCR</th>
                  <th className="pb-2 text-right font-medium">黒転年</th>
                  <th className="pb-2 text-right font-medium">判定</th>
                </tr>
              </thead>
              <tbody>
                {stressScenarios.map((s) => {
                  const isCompound = s.label === "複合ストレス";
                  const rowBg = isCompound ? "bg-orange-50" : "";
                  const safeBadge = s.isSafe
                    ? <Badge variant="success">安全</Badge>
                    : s.dscr >= 1.0
                      ? <Badge variant="warning">注意</Badge>
                      : <Badge variant="danger">危険</Badge>;
                  return (
                    <tr key={s.label} className={`border-b last:border-0 ${rowBg}`}>
                      <td className="py-2 font-medium">
                        {s.label}
                        {isCompound && <span className="ml-1 text-xs text-orange-600">★</span>}
                      </td>
                      <td className="py-2 text-right">
                        {s.interestRateDelta !== 0 ? `+${(s.interestRateDelta * 100).toFixed(1)}%` : "±0"}
                      </td>
                      <td className="py-2 text-right">
                        {s.vacancyRateDelta !== 0 ? `+${(s.vacancyRateDelta * 100).toFixed(0)}%` : "±0"}
                      </td>
                      <td className={`py-2 text-right font-medium ${s.dscr >= 1.0 ? "text-green-600" : "text-red-600"}`}>
                        {s.dscr.toFixed(2)}
                      </td>
                      <td className="py-2 text-right text-muted-foreground">
                        {s.breakEvenYear === -1 ? "なし" : `${s.breakEvenYear}年目`}
                      </td>
                      <td className="py-2 text-right">{safeBadge}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">
            ※ DSCR（借債返済カバー率）= NOI / 年間ローン返済額。1.0以上が銀行審査の目安。黒転年は保有期間内での累積CF黒転年度。
          </p>

          {/* 人口減少シナリオ */}
          {populationForecast && populationForecast.snapshots.length > 0 && (() => {
            const popV = Math.min(actualV + populationForecast.vacancyRateDelta, 0.99);
            const popNetYield = result.grossYield * (1 - popV) * (1 - input.expenseRate);
            const popAnnualRent = result.grossYield * result.totalInvestment * (1 - popV);
            const popCF = popAnnualRent - (result.yearlyResults[0]?.annualLoanPayment ?? 0) - (result.yearlyResults[0]?.annualExpenses ?? 0);
            const isDeficit = popCF < 0;
            return (
              <div className={`mt-4 rounded-md border p-3 ${isDeficit ? "border-red-300 bg-red-50" : "border-yellow-200 bg-yellow-50"}`}>
                <p className="flex items-center gap-1 text-sm font-semibold mb-2">
                  <Users className="h-4 w-4" />
                  人口減少シナリオ（30年後推計）
                  {isDeficit && <Badge variant="danger" className="ml-2">赤字転落 ⚠️</Badge>}
                </p>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                  <div className="text-muted-foreground">想定空室率</div>
                  <div className="text-right font-medium">{(popV * 100).toFixed(0)}%</div>
                  <div className="text-muted-foreground">表面利回り</div>
                  <div className={`text-right font-medium ${popNetYield * 100 >= 8 ? "text-green-600" : "text-red-600"}`}>
                    {(result.grossYield * (1 - popV) * 100).toFixed(2)}%
                  </div>
                  <div className="text-muted-foreground">実質利回り</div>
                  <div className="text-right text-muted-foreground">{(popNetYield * 100).toFixed(2)}%</div>
                  <div className="text-muted-foreground">年間CF（概算）</div>
                  <div className={`text-right font-bold ${isDeficit ? "text-red-700" : "text-green-700"}`}>
                    {formatMan(popCF)}
                  </div>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  ※ このエリアの30年後人口推計: {(populationForecast.changeRate30yr * 100).toFixed(0)}%（現在比）／トレンド: {populationForecast.trend}
                </p>
              </div>
            );
          })()}
        </CardContent>
      </Card>

      {/* 空室シナリオ比較 */}
      {result.yieldScenarios && (
        <VacancyScenarioCard yieldScenarios={result.yieldScenarios} vacancyRate={input.vacancyRate} />
      )}

      {/* 総投資額サマリー */}
      <Card>
        <CardHeader><CardTitle className="text-base">投資サマリー</CardTitle></CardHeader>
        <CardContent>
          <dl className="space-y-2">
            {[
              { label: "総投資額", value: formatMan(result.totalInvestment), highlight: true },
              { label: "　うち諸経費", value: formatYen(result.miscExpenses) },
              { label: "年間実効賃料収入（空室控除後）", value: formatYen(result.yearlyResults[0]?.annualRent ?? 0) },
              { label: "年間ローン返済", value: formatYen(result.yearlyResults[0]?.annualLoanPayment ?? 0) },
              { label: "1年目税引後CF", value: formatYen(result.yearlyResults[0]?.afterTaxCashFlow ?? 0) },
            ].map(({ label, value, highlight }) => (
              <div key={label} className="flex justify-between text-sm">
                <dt className={`text-muted-foreground ${highlight ? "font-semibold text-foreground" : ""}`}>{label}</dt>
                <dd className={`font-medium ${highlight ? "text-primary" : ""}`}>{value}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      {/* 8%未達の場合: 逆算カード（ISSUE-16: either-or を明示） */}
      {!isGood && (
        <Card className="border-orange-200 bg-orange-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-orange-800">
              <TrendingDown className="h-5 w-5" />
              8%達成のために必要な改善（いずれか一方）
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="rounded-md bg-white/70 p-3">
              <p className="text-xs text-muted-foreground">
                土地価格 <strong>または</strong> 建築費のどちらか一方を削減する必要がある額
              </p>
              <p className="text-xl font-bold text-orange-700">
                ▼ {formatMan(result.requiredCostReduction)}
              </p>
            </div>
            <div className="rounded-md bg-white/70 p-3">
              <p className="text-xs text-muted-foreground">または、必要な月額賃料（満室想定）</p>
              <p className="text-xl font-bold text-orange-700">
                ▲ {formatYen(result.requiredMonthlyRent)}/月
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 8%以上の場合: 余裕度 */}
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
              <span className="font-bold">+{formatPct(result.grossYield - 0.08)}</span>{" "}
              の余裕があります。
            </p>
            <p className="mt-2 text-xs text-muted-foreground">
              ※満室想定賃料が{formatPct((result.grossYield - 0.08) / result.grossYield)}下落すると8%を下回ります
              （空室変動の影響は別途考慮してください）
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

interface VacancyScenarioCardProps {
  yieldScenarios: YieldScenarios;
  vacancyRate: number;
}

function VacancyScenarioCard({ yieldScenarios, vacancyRate }: VacancyScenarioCardProps) {
  const scenarios = [
    {
      label: "楽観",
      vacancyPct: (vacancyRate * 0.5 * 100).toFixed(0),
      annualRent: yieldScenarios.optimistic.annualRent,
      grossYield: yieldScenarios.optimistic.grossYield,
      colorClass: "text-green-600",
    },
    {
      label: "標準",
      vacancyPct: (vacancyRate * 1.0 * 100).toFixed(0),
      annualRent: yieldScenarios.standard.annualRent,
      grossYield: yieldScenarios.standard.grossYield,
      colorClass: "text-blue-600",
    },
    {
      label: "悲観",
      vacancyPct: (vacancyRate * 1.5 * 100).toFixed(0),
      annualRent: yieldScenarios.pessimistic.annualRent,
      grossYield: yieldScenarios.pessimistic.grossYield,
      colorClass: "text-red-600",
    },
  ];

  return (
    <Card>
      <CardHeader><CardTitle className="text-base">空室シナリオ別 年間賃料収入</CardTitle></CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-muted-foreground">
                <th className="pb-2 text-left font-medium">シナリオ</th>
                <th className="pb-2 text-right font-medium">想定空室率</th>
                <th className="pb-2 text-right font-medium">年間実効賃料</th>
                <th className="pb-2 text-right font-medium">表面利回り</th>
              </tr>
            </thead>
            <tbody>
              {scenarios.map((s) => (
                <tr key={s.label} className="border-b last:border-0">
                  <td className={`py-2 font-medium ${s.colorClass}`}>{s.label}</td>
                  <td className="py-2 text-right text-muted-foreground">{s.vacancyPct}%</td>
                  <td className="py-2 text-right font-medium">{formatYen(s.annualRent)}</td>
                  <td className={`py-2 text-right font-medium ${s.colorClass}`}>
                    {(s.grossYield * 100).toFixed(2)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          ※ 楽観: 想定空室率×0.5、標準: 想定空室率×1.0、悲観: 想定空室率×1.5。表面利回りは満室想定年収/総投資額。
        </p>
      </CardContent>
    </Card>
  );
}

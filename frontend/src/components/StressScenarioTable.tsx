"use client";
import React, { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { AlertTriangle, ChevronDown, ChevronUp, Loader2, Users } from "lucide-react";
import { TermTooltip } from "@/components/ui/TermTooltip";
import { formatMan, formatYen } from "@/lib/utils";
import type {
  InvestmentInput,
  InvestmentResult,
  PopulationForecastResult,
  StressScenarioResult,
} from "@/types/investment";
import { getDscrBadge, getDscrColorClass, getDscrIcon } from "@/components/DscrBadge";
import type { CustomScenarioState } from "@/components/CustomScenarioSlider";

interface StressScenarioTableProps {
  stressScenarios: StressScenarioResult[];
  customState: CustomScenarioState;
  populationForecast?: PopulationForecastResult | null;
  input: InvestmentInput;
  result: InvestmentResult;
}

export function StressScenarioTable({
  stressScenarios,
  customState,
  populationForecast,
  input,
  result,
}: StressScenarioTableProps) {
  const { customScenario, customLoading, customLoanRateDelta, customVacancyRateDelta } =
    customState;
  const [expandedScenario, setExpandedScenario] = useState<number | null>(null);

  const actualV = input.actualVacancyRate > 0 ? input.actualVacancyRate : input.vacancyRate;

  return (
    <>
      {/* Mobile: accordion (lg:hidden) */}
      <div className="space-y-1 lg:hidden">
        {stressScenarios.map((s, idx) => {
          const isCompound = s.label === "複合ストレス";
          const isExpanded = expandedScenario === idx;
          const dscrIcon = getDscrIcon(s.dscr);
          const safeBadge = getDscrBadge(s.dscr);
          return (
            <div
              key={s.label}
              className={`rounded-lg border ${isCompound ? "border-orange-200 bg-orange-50" : "border-border bg-background"}`}
            >
              <button
                type="button"
                className="flex w-full items-center justify-between px-3 py-3 text-left min-h-[44px]"
                onClick={() => setExpandedScenario(isExpanded ? null : idx)}
              >
                <span className="text-sm font-medium">
                  {s.label}
                  {isCompound && <span className="ml-1 text-xs text-orange-600">★</span>}
                </span>
                <div className="flex items-center gap-2">
                  {safeBadge}
                  {isExpanded ? (
                    <ChevronUp className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-muted-foreground" />
                  )}
                </div>
              </button>
              {isExpanded && (
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 border-t px-3 pb-3 pt-2 text-sm">
                  <span className="text-muted-foreground">
                    <TermTooltip term="dscr">DSCR</TermTooltip>
                  </span>
                  <span
                    className={`flex items-center justify-end gap-1 font-medium ${getDscrColorClass(s.dscr)}`}
                  >
                    {dscrIcon}
                    {s.dscr.toFixed(2)}
                  </span>
                  <span className="text-muted-foreground">
                    <TermTooltip term="breakeven_cf">CF黒転年</TermTooltip>
                  </span>
                  <span className="text-right">
                    {s.breakEvenYear === -1 ? "なし" : `${s.breakEvenYear}年目`}
                  </span>
                  <span className="text-muted-foreground">金利△</span>
                  <span className="text-right">
                    {s.interestRateDelta !== 0
                      ? `+${(s.interestRateDelta * 100).toFixed(1)}%`
                      : "±0"}
                  </span>
                  <span className="text-muted-foreground">空室△</span>
                  <span className="text-right">
                    {s.vacancyRateDelta !== 0 ? `+${(s.vacancyRateDelta * 100).toFixed(0)}%` : "±0"}
                  </span>
                </div>
              )}
            </div>
          );
        })}
        {/* カスタムシナリオ行（モバイル） */}
        {customLoading && (customLoanRateDelta > 0 || customVacancyRateDelta > 0) && (
          <div className="rounded-lg border border-blue-200 bg-blue-50">
            <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              カスタム（計算中）
            </div>
          </div>
        )}
        {!customLoading && customScenario && (
          <div className="rounded-lg border border-blue-200 bg-blue-50">
            <button
              type="button"
              className="flex w-full items-center justify-between px-3 py-3 text-left min-h-[44px]"
              onClick={() =>
                setExpandedScenario(
                  expandedScenario === stressScenarios.length ? null : stressScenarios.length
                )
              }
            >
              <span className="text-sm font-medium text-blue-700">
                カスタム <span className="text-xs text-blue-500">★</span>
              </span>
              <div className="flex items-center gap-2">
                {getDscrBadge(customScenario.dscr)}
                {expandedScenario === stressScenarios.length ? (
                  <ChevronUp className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                )}
              </div>
            </button>
            {expandedScenario === stressScenarios.length && (
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 border-t px-3 pb-3 pt-2 text-sm">
                <span className="text-muted-foreground">
                  <TermTooltip term="dscr">DSCR</TermTooltip>
                </span>
                <span
                  className={`text-right font-medium ${getDscrColorClass(customScenario.dscr)}`}
                >
                  {customScenario.dscr.toFixed(2)}
                </span>
                <span className="text-muted-foreground">
                  <TermTooltip term="breakeven_cf">CF黒転年</TermTooltip>
                </span>
                <span className="text-right">
                  {customScenario.breakEvenYear === -1
                    ? "なし"
                    : `${customScenario.breakEvenYear}年目`}
                </span>
                <span className="text-muted-foreground">金利△</span>
                <span className="text-right">
                  {customLoanRateDelta > 0 ? `+${customLoanRateDelta.toFixed(1)}%` : "±0"}
                </span>
                <span className="text-muted-foreground">空室△</span>
                <span className="text-right">
                  {customVacancyRateDelta > 0 ? `+${customVacancyRateDelta.toFixed(0)}%` : "±0"}
                </span>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Desktop: horizontal table (hidden on mobile) */}
      <div className="hidden lg:block overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-muted-foreground">
              <th className="pb-2 text-left font-medium">シナリオ</th>
              <th className="pb-2 text-right font-medium">金利△</th>
              <th className="pb-2 text-right font-medium">空室△</th>
              <th className="pb-2 text-right font-medium">
                <TermTooltip term="dscr">DSCR</TermTooltip>{" "}
                <TermTooltip term="dscrThreshold">基準</TermTooltip>
              </th>
              <th className="pb-2 text-right font-medium">
                <TermTooltip term="breakeven_cf">CF黒転年</TermTooltip>
              </th>
              <th className="pb-2 text-right font-medium">判定</th>
            </tr>
          </thead>
          <tbody>
            {stressScenarios.map((s) => {
              const isCompound = s.label === "複合ストレス";
              const rowBg = isCompound ? "bg-orange-50" : "";
              const safeBadge = getDscrBadge(s.dscr);
              const dscrIcon = getDscrIcon(s.dscr);
              return (
                <tr key={s.label} className={`border-b last:border-0 ${rowBg}`}>
                  <td className="py-2 font-medium">
                    {s.label}
                    {isCompound && <span className="ml-1 text-xs text-orange-600">★</span>}
                  </td>
                  <td className="py-2 text-right">
                    {s.interestRateDelta !== 0
                      ? `+${(s.interestRateDelta * 100).toFixed(1)}%`
                      : "±0"}
                  </td>
                  <td className="py-2 text-right">
                    {s.vacancyRateDelta !== 0 ? `+${(s.vacancyRateDelta * 100).toFixed(0)}%` : "±0"}
                  </td>
                  <td
                    data-testid="dscr-value"
                    className={`py-2 text-right font-medium ${getDscrColorClass(s.dscr)}`}
                  >
                    <span className="inline-flex items-center justify-end gap-1">
                      {dscrIcon}
                      {s.dscr.toFixed(2)}
                    </span>
                  </td>
                  <td className="py-2 text-right text-muted-foreground">
                    {s.breakEvenYear === -1 ? "なし" : `${s.breakEvenYear}年目`}
                  </td>
                  <td className="py-2 text-right">{safeBadge}</td>
                </tr>
              );
            })}
            {/* カスタムシナリオ行（デスクトップ） */}
            {customLoading && (customLoanRateDelta > 0 || customVacancyRateDelta > 0) && (
              <tr className="border-b bg-blue-50">
                <td className="py-2 font-medium text-blue-700" colSpan={6}>
                  <span className="flex items-center gap-1">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    カスタム（計算中）
                  </span>
                </td>
              </tr>
            )}
            {!customLoading && customScenario && (
              <tr className="border-b bg-blue-50">
                <td className="py-2 font-medium text-blue-700">
                  カスタム <span className="ml-0.5 text-xs text-blue-500">★</span>
                </td>
                <td className="py-2 text-right">
                  {customLoanRateDelta > 0 ? `+${customLoanRateDelta.toFixed(1)}%` : "±0"}
                </td>
                <td className="py-2 text-right">
                  {customVacancyRateDelta > 0 ? `+${customVacancyRateDelta.toFixed(0)}%` : "±0"}
                </td>
                <td
                  className={`py-2 text-right font-medium ${getDscrColorClass(customScenario.dscr)}`}
                >
                  {customScenario.dscr.toFixed(2)}
                </td>
                <td className="py-2 text-right text-muted-foreground">
                  {customScenario.breakEvenYear === -1
                    ? "なし"
                    : `${customScenario.breakEvenYear}年目`}
                </td>
                <td className="py-2 text-right">{getDscrBadge(customScenario.dscr)}</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <p className="mt-2 text-xs text-muted-foreground">
        ※ <TermTooltip term="dscr">DSCR</TermTooltip> = <TermTooltip term="noi">NOI</TermTooltip>
        （賃料下落・空室調整済み）/
        年間ローン返済額（保有期間内の最悪年）。1.2以上が実務基準、1.0以上が銀行審査の最低ライン（
        <span className="text-green-600">緑≥1.2</span>・
        <span className="text-yellow-600">黄1.0〜1.2</span>・
        <span className="text-red-600">赤&lt;1.0</span>
        ）。CF黒転年は累積CF黒転年度（税引後）。年間CFがプラスに転じた年であり、自己資金の回収年ではありません。
      </p>

      {/* 人口減少シナリオ */}
      {populationForecast &&
        populationForecast.snapshots.length > 0 &&
        (() => {
          const popV = Math.min(actualV + populationForecast.vacancyRateDelta, 0.99);
          const popNetYield = result.grossYield * (1 - popV) * (1 - input.expenseRate);
          const popAnnualRent = result.grossYield * result.totalInvestment * (1 - popV);
          const popCF =
            popAnnualRent -
            (result.yearlyResults[0]?.annualLoanPayment ?? 0) -
            (result.yearlyResults[0]?.annualExpenses ?? 0);
          const isDeficit = popCF < 0;
          return (
            <div
              className={`mt-4 rounded-md border p-3 ${isDeficit ? "border-red-300 bg-red-50" : "border-yellow-200 bg-yellow-50"}`}
            >
              <p className="flex items-center gap-1 text-sm font-semibold mb-2">
                <Users className="h-4 w-4" />
                人口減少シナリオ（30年後推計）
                {isDeficit && (
                  <Badge variant="danger" className="ml-2 flex items-center gap-1">
                    <AlertTriangle className="h-3 w-3" />
                    赤字転落
                  </Badge>
                )}
              </p>
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                <div className="text-muted-foreground">想定空室率</div>
                <div className="text-right font-medium">{(popV * 100).toFixed(0)}%</div>
                <div className="text-muted-foreground">表面利回り</div>
                <div
                  className={`text-right font-medium ${popNetYield * 100 >= 8 ? "text-green-600" : "text-red-600"}`}
                >
                  {(result.grossYield * (1 - popV) * 100).toFixed(2)}%
                </div>
                <div className="text-muted-foreground">
                  <TermTooltip term="netYield">実質利回り</TermTooltip>
                </div>
                <div className="text-right text-muted-foreground">
                  {(popNetYield * 100).toFixed(2)}%
                </div>
                <div className="text-muted-foreground">年間CF（概算）</div>
                <div
                  className={`flex items-center justify-end gap-1 font-bold ${isDeficit ? "text-red-700" : "text-green-700"}`}
                >
                  {isDeficit && <AlertTriangle className="h-3.5 w-3.5 shrink-0" />}
                  {formatMan(popCF)}
                </div>
              </div>
              <p className="mt-2 text-xs text-muted-foreground">
                ※ このエリアの30年後人口推計: {(populationForecast.changeRate30yr * 100).toFixed(0)}
                %（現在比）／トレンド: {populationForecast.trend}
              </p>
            </div>
          );
        })()}
    </>
  );
}

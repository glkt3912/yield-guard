"use client";
import React, { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Slider } from "@/components/ui/slider";
import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceStats,
  PopulationForecastResult,
  StressScenarioResult,
  YieldScenarios,
} from "@/types/investment";
import { formatMan, formatPct, formatYen } from "@/lib/utils";
import { calcYieldBenchmark } from "@/lib/yieldBenchmark";
import {
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  CheckCircle,
  CheckCircle2,
  XCircle,
  Users,
  ChevronDown,
  ChevronUp,
  Loader2,
} from "lucide-react";
import { TermTooltip } from "@/components/ui/TermTooltip";
import { MobileSummaryCard } from "@/components/MobileSummaryCard";
import { analyze } from "@/lib/api";

const CUSTOM_SCENARIO_LABEL = "カスタム" as const;

export function getDscrColorClass(dscr: number): string {
  if (dscr >= 1.2) return "text-green-600";
  if (dscr >= 1.0) return "text-yellow-600";
  return "text-red-600";
}

export function getDscrBadge(dscr: number): React.ReactElement {
  if (dscr >= 1.2)
    return (
      <Badge variant="success" className="flex items-center gap-1">
        <CheckCircle2 className="h-3 w-3" />
        安全
      </Badge>
    );
  if (dscr >= 1.0)
    return (
      <Badge variant="warning" className="flex items-center gap-1">
        <AlertTriangle className="h-3 w-3" />
        注意
      </Badge>
    );
  return (
    <Badge variant="danger" className="flex items-center gap-1">
      <XCircle className="h-3 w-3" />
      危険
    </Badge>
  );
}

function getDscrIcon(dscr: number) {
  if (dscr >= 1.2) return <CheckCircle2 className="h-3 w-3" />;
  if (dscr >= 1.0) return <AlertTriangle className="h-3 w-3" />;
  return <XCircle className="h-3 w-3" />;
}

/**
 * Mobile 1×2 KPI strip showing metrics not already in MobileSummaryCard.
 * MobileSummaryCard covers yield/DSCR/dead-cross; this adds investment totals.
 */
function MobileKpiGrid({ result }: { result: InvestmentResult }) {
  const firstYearCF = result.yearlyResults[0]?.afterTaxCashFlow ?? 0;

  const kpis = [
    {
      label: "総投資額",
      value: formatMan(result.totalInvestment),
      sub: "諸経費込み",
      color: "text-primary",
      bg: "bg-muted/40 border-border",
    },
    {
      label: "1年目税引後CF",
      value: formatYen(firstYearCF),
      sub: "年間キャッシュフロー",
      color: firstYearCF >= 0 ? "text-green-600" : "text-red-600",
      bg: firstYearCF >= 0 ? "bg-green-50 border-green-200" : "bg-red-50 border-red-200",
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-3 lg:hidden">
      {kpis.map((kpi) => (
        <div key={kpi.label} className={`rounded-xl border p-3 ${kpi.bg}`}>
          <p className="text-xs text-muted-foreground">{kpi.label}</p>
          <p className={`mt-1 text-xl font-bold leading-tight ${kpi.color}`}>{kpi.value}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">{kpi.sub}</p>
        </div>
      ))}
    </div>
  );
}

interface Props {
  result: InvestmentResult;
  input: InvestmentInput;
  populationForecast?: PopulationForecastResult | null;
  landPriceStats?: LandPriceStats | null;
}

export function YieldAnalysis({ result, input, populationForecast, landPriceStats }: Props) {
  const yieldPct = result.grossYield * 100;
  const netYieldPct = result.netYield * 100;
  const isGood = result.isAboveYieldTarget;
  const targetPct = result.yieldTarget * 100;
  const maxYieldPct = targetPct * 2; // ゲージ上限: 目標の2倍（目標が中央）

  // 人口シナリオ用（populationForecast表示のために残す）
  const actualV = input.actualVacancyRate > 0 ? input.actualVacancyRate : input.vacancyRate;

  const stressScenarios: StressScenarioResult[] = result.stressScenarios ?? [];
  const [expandedScenario, setExpandedScenario] = useState<number | null>(null);

  // カスタムストレステスト入力
  const [customLoanRateDelta, setCustomLoanRateDelta] = useState(0); // 0〜3 (%)
  const [customVacancyRateDelta, setCustomVacancyRateDelta] = useState(0); // 0〜30 (%)
  const [customScenario, setCustomScenario] = useState<StressScenarioResult | null>(null);
  const [customLoading, setCustomLoading] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const fetchCustomScenario = useCallback(
    (loanDeltaPct: number, vacancyDeltaPct: number) => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      const loanDelta = loanDeltaPct / 100;
      const vacancyDelta = vacancyDeltaPct / 100;
      if (loanDelta === 0 && vacancyDelta === 0) {
        setCustomScenario(null);
        return;
      }
      setCustomLoading(true);
      debounceRef.current = setTimeout(async () => {
        try {
          const res = await analyze({
            ...input,
            loanRateDelta: loanDelta,
            vacancyRateDelta: vacancyDelta,
          });
          const custom = res.stressScenarios.find((s) => s.label === CUSTOM_SCENARIO_LABEL) ?? null;
          setCustomScenario(custom);
        } catch (err) {
          console.error("[CustomStressTest] fetch failed:", err);
          setCustomScenario(null);
        } finally {
          setCustomLoading(false);
        }
      }, 400);
    },
    [input]
  );

  useEffect(() => {
    fetchCustomScenario(customLoanRateDelta, customVacancyRateDelta);
  }, [customLoanRateDelta, customVacancyRateDelta, fetchCustomScenario]);

  const gaugePosition = Math.min(yieldPct / maxYieldPct, 1) * 100;
  const targetPosition = 50; // 目標は常にゲージ中央

  const yieldBenchmark = useMemo(() => {
    if (!landPriceStats || landPriceStats.medianTsubo <= 0) return null;
    if (input.monthlyRent <= 0) return null;
    const userYield =
      input.landPrice + input.buildingCost > 0
        ? (input.monthlyRent * 12) / (input.landPrice + input.buildingCost)
        : undefined;
    return calcYieldBenchmark({
      medianTsubo: landPriceStats.medianTsubo,
      minTsubo: landPriceStats.minTsubo,
      maxTsubo: landPriceStats.maxTsubo,
      landAreaSqm: input.landArea,
      monthlyRent: input.monthlyRent,
      buildingCost: input.buildingCost,
      userYield,
    });
  }, [landPriceStats, input.landArea, input.monthlyRent, input.buildingCost, input.landPrice]);

  return (
    <div className="space-y-4">
      {/* Mobile: professional summary card for agents (hidden on desktop) */}
      <MobileSummaryCard
        result={result}
        input={input}
        yieldPct={yieldPct}
        netYieldPct={netYieldPct}
      />

      {/* Mobile: investment totals strip (hidden on desktop) */}
      <MobileKpiGrid result={result} />

      {/* メイン利回り表示 */}
      <Card className={`border-2 ${isGood ? "border-green-400" : "border-red-400"}`}>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div className="min-w-0 flex-1">
              <p className="text-xs text-muted-foreground sm:text-sm">
                表面利回り（満室想定年収 / 総投資額）
              </p>
              <div className="flex items-end gap-2">
                <span
                  className={`text-4xl font-bold sm:text-5xl ${isGood ? "text-green-600" : "text-red-600"}`}
                >
                  {yieldPct.toFixed(2)}
                </span>
                <span className="mb-1 text-xl font-semibold text-muted-foreground sm:mb-2 sm:text-2xl">
                  %
                </span>
              </div>
              <p className="mt-1 text-xs text-muted-foreground sm:text-sm">
                <TermTooltip term="netYield">実質利回り</TermTooltip>（空室・経費控除後）：
                {netYieldPct.toFixed(2)}%
              </p>
            </div>
            <div className="ml-3 flex flex-col items-center gap-2 shrink-0">
              {isGood ? (
                <>
                  <CheckCircle className="h-10 w-10 text-green-500 sm:h-12 sm:w-12" />
                  <Badge variant="success">{targetPct}%超え ✓</Badge>
                </>
              ) : (
                <>
                  <AlertTriangle className="h-10 w-10 text-red-500 sm:h-12 sm:w-12" />
                  <Badge variant="danger">{targetPct}%未満 ✗</Badge>
                </>
              )}
            </div>
          </div>

          {/* 目標利回り境界線ゲージ（目標が中央） */}
          <div className="mt-4">
            <div className="flex justify-between text-xs text-muted-foreground mb-1">
              <span>0%</span>
              <span className="font-semibold text-orange-500">目標 {targetPct}%</span>
              <span>{maxYieldPct}%+</span>
            </div>
            <div className="relative h-3 rounded-full bg-muted overflow-hidden">
              <div className="absolute inset-y-0 left-0 rounded-full bg-gradient-to-r from-red-400 via-yellow-400 to-green-400 w-full" />
              {/* 現在値マーカー */}
              <div
                className="absolute top-0 h-full w-1 bg-foreground/80 rounded"
                style={{ left: `${gaugePosition}%` }}
              />
              {/* 8%ライン（スケールと連動） */}
              <div
                className="absolute top-0 h-full w-0.5 bg-orange-500"
                style={{ left: `${targetPosition}%` }}
              />
            </div>
          </div>

          {/* 市場想定利回りベンチマーク */}
          {yieldBenchmark && (
            <div className="mt-2 rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
              <span>
                このエリアの市場想定利回り: 約
                {(yieldBenchmark.estimatedYieldTypical * 100).toFixed(1)}%
              </span>
              <Badge
                className="ml-2"
                title={yieldBenchmark.judgmentLabel}
                variant={
                  yieldBenchmark.judgment === "realistic"
                    ? "outline"
                    : yieldBenchmark.judgment === "slightly-high"
                      ? "warning"
                      : "danger"
                }
              >
                {yieldBenchmark.judgment === "realistic"
                  ? "現実的"
                  : yieldBenchmark.judgment === "slightly-high"
                    ? "やや高め"
                    : "大幅に高め"}
              </Badge>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ストレステストシナリオ比較 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">ストレステストシナリオ（銀行融資審査用）</CardTitle>
        </CardHeader>
        <CardContent>
          {/* カスタムストレステスト入力スライダー */}
          <div className="mb-4 rounded-lg border border-border bg-muted/30 p-4 space-y-4">
            <p className="text-sm font-medium">カスタムシナリオ設定</p>
            <Slider
              label="金利オフセット"
              value={customLoanRateDelta}
              min={0}
              max={3}
              step={0.1}
              onChange={(v) => setCustomLoanRateDelta(v)}
              formatValue={(v) => (v === 0 ? "±0" : `+${v.toFixed(1)}%`)}
            />
            <Slider
              label="空室率オフセット"
              value={customVacancyRateDelta}
              min={0}
              max={30}
              step={1}
              onChange={(v) => setCustomVacancyRateDelta(v)}
              formatValue={(v) => (v === 0 ? "±0" : `+${v.toFixed(0)}%`)}
            />
            {(customLoanRateDelta > 0 || customVacancyRateDelta > 0) && customLoading && (
              <p className="flex items-center gap-1 text-xs text-muted-foreground">
                <Loader2 className="h-3 w-3 animate-spin" />
                計算中…
              </p>
            )}
          </div>

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
                      <span className="text-muted-foreground">DSCR</span>
                      <span
                        className={`flex items-center justify-end gap-1 font-medium ${getDscrColorClass(s.dscr)}`}
                      >
                        {dscrIcon}
                        {s.dscr.toFixed(2)}
                      </span>
                      <span className="text-muted-foreground">CF黒転年</span>
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
                        {s.vacancyRateDelta !== 0
                          ? `+${(s.vacancyRateDelta * 100).toFixed(0)}%`
                          : "±0"}
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
                    <span className="text-muted-foreground">DSCR</span>
                    <span
                      className={`text-right font-medium ${getDscrColorClass(customScenario.dscr)}`}
                    >
                      {customScenario.dscr.toFixed(2)}
                    </span>
                    <span className="text-muted-foreground">CF黒転年</span>
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
                  <th className="pb-2 text-right font-medium">CF黒転年</th>
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
                        {s.vacancyRateDelta !== 0
                          ? `+${(s.vacancyRateDelta * 100).toFixed(0)}%`
                          : "±0"}
                      </td>
                      <td className={`py-2 text-right font-medium ${getDscrColorClass(s.dscr)}`}>
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
            ※ <TermTooltip term="dscr">DSCR</TermTooltip> ={" "}
            <TermTooltip term="noi">NOI</TermTooltip>（賃料下落・空室調整済み）/
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
                    ※ このエリアの30年後人口推計:{" "}
                    {(populationForecast.changeRate30yr * 100).toFixed(0)}%（現在比）／トレンド:{" "}
                    {populationForecast.trend}
                  </p>
                </div>
              );
            })()}
        </CardContent>
      </Card>

      {/* 空室シナリオ比較 */}
      {result.yieldScenarios && (
        <VacancyScenarioCard
          yieldScenarios={result.yieldScenarios}
          vacancyRate={input.vacancyRate}
          totalInvestment={result.totalInvestment}
        />
      )}

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
              {
                label: "年間実効賃料収入（空室控除後）",
                value: formatYen(result.yearlyResults[0]?.annualRent ?? 0),
              },
              {
                label: "年間ローン返済",
                value: formatYen(result.yearlyResults[0]?.annualLoanPayment ?? 0),
              },
              {
                label: "1年目税引後CF",
                value: formatYen(result.yearlyResults[0]?.afterTaxCashFlow ?? 0),
              },
            ].map(({ label, value, highlight }) => (
              <div key={label} className="flex justify-between text-sm">
                <dt
                  className={`text-muted-foreground ${highlight ? "font-semibold text-foreground" : ""}`}
                >
                  {label}
                </dt>
                <dd className={`font-medium ${highlight ? "text-primary" : ""}`}>{value}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      {/* 目標利回り未達の場合: 逆算カード（ISSUE-16: either-or を明示） */}
      {!isGood && (
        <Card className="border-orange-200 bg-orange-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-orange-800">
              <TrendingDown className="h-5 w-5" />
              {targetPct}%達成のために必要な改善（いずれか一方）
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

      {/* 目標利回り以上の場合: 余裕度 */}
      {isGood && (
        <Card className="border-green-200 bg-green-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-green-800">
              <TrendingUp className="h-5 w-5" />
              {targetPct}%超え達成！余裕度
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-green-700">
              目標の{targetPct}%に対して{" "}
              <span className="font-bold">
                +{formatPct(result.grossYield - result.yieldTarget)}
              </span>{" "}
              の余裕があります。
            </p>
            <p className="mt-2 text-xs text-muted-foreground">
              ※満室想定賃料が
              {formatPct((result.grossYield - result.yieldTarget) / result.grossYield)}下落すると
              {targetPct}%を下回ります （空室変動の影響は別途考慮してください）
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
  totalInvestment: number;
}

function VacancyScenarioCard({
  yieldScenarios,
  vacancyRate,
  totalInvestment,
}: VacancyScenarioCardProps) {
  const scenarios = [
    {
      label: "楽観",
      vacancyPct: (Math.min(vacancyRate * 0.5, 0.99) * 100).toFixed(0),
      annualRent: yieldScenarios.optimistic.annualRent,
      effectiveYield: yieldScenarios.optimistic.annualRent / totalInvestment,
      colorClass: "text-green-600",
    },
    {
      label: "標準",
      vacancyPct: (Math.min(vacancyRate * 1.0, 0.99) * 100).toFixed(0),
      annualRent: yieldScenarios.standard.annualRent,
      effectiveYield: yieldScenarios.standard.annualRent / totalInvestment,
      colorClass: "text-blue-600",
    },
    {
      label: "悲観",
      vacancyPct: (Math.min(vacancyRate * 1.5, 0.99) * 100).toFixed(0),
      annualRent: yieldScenarios.pessimistic.annualRent,
      effectiveYield: yieldScenarios.pessimistic.annualRent / totalInvestment,
      colorClass: "text-red-600",
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">空室シナリオ別 年間賃料収入</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-muted-foreground">
                <th className="pb-2 text-left font-medium">シナリオ</th>
                <th className="pb-2 text-right font-medium">想定空室率</th>
                <th className="pb-2 text-right font-medium">年間実効賃料</th>
                <th className="pb-2 text-right font-medium">実効利回り</th>
              </tr>
            </thead>
            <tbody>
              {scenarios.map((s) => (
                <tr key={s.label} className="border-b last:border-0">
                  <td className={`py-2 font-medium ${s.colorClass}`}>{s.label}</td>
                  <td className="py-2 text-right text-muted-foreground">{s.vacancyPct}%</td>
                  <td className="py-2 text-right font-medium">{formatYen(s.annualRent)}</td>
                  <td className={`py-2 text-right font-medium ${s.colorClass}`}>
                    {(s.effectiveYield * 100).toFixed(2)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          ※ 楽観: 想定空室率×0.5、標準: 想定空室率×1.0、悲観:
          想定空室率×1.5。実効利回りは空室控除後年間賃料/総投資額。
        </p>
      </CardContent>
    </Card>
  );
}

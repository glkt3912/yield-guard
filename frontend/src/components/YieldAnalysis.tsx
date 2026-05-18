"use client";
import React, { useState, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceStats,
  PopulationForecastResult,
  YieldScenarios,
} from "@/types/investment";
import { formatMan, formatPct, formatYen } from "@/lib/utils";
import { TrendingUp, TrendingDown, AlertTriangle, ChevronDown, ChevronUp } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { MobileSummaryCard } from "@/components/MobileSummaryCard";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { YieldGauge } from "@/components/YieldGauge";
import { CustomScenarioSlider, type CustomScenarioState } from "@/components/CustomScenarioSlider";
import { StressScenarioTable } from "@/components/StressScenarioTable";

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
      color: firstYearCF >= 0 ? "text-success-fg" : "text-danger-fg",
      bg: firstYearCF >= 0 ? "bg-success-bg border-success-bg" : "bg-danger-bg border-danger-bg",
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
  isAnalyzing?: boolean;
}

export function YieldAnalysis({
  result,
  input,
  populationForecast,
  landPriceStats,
  isAnalyzing = false,
}: Props) {
  const yieldPct = result.grossYield * 100;
  const netYieldPct = result.netYield * 100;
  const isGood = result.isAboveYieldTarget;
  const targetPct = result.yieldTarget * 100;

  const [aiOpen, setAiOpen] = useState(true);
  const [customState, setCustomState] = useState<CustomScenarioState>({
    customScenario: null,
    customLoading: false,
    customLoanRateDelta: 0,
    customVacancyRateDelta: 0,
  });

  const handleCustomStateChange = useCallback((state: CustomScenarioState) => {
    setCustomState(state);
  }, []);

  return (
    <div className="space-y-4">
      {isAnalyzing ? (
        <Skeleton className="h-20 w-full rounded-lg" />
      ) : result.aiSummary ? (
        <Card className="border border-info-bg bg-info-bg/50">
          <CardHeader className="cursor-pointer pb-2 pt-4" onClick={() => setAiOpen(!aiOpen)}>
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-info-fg">AIレポート</CardTitle>
              {aiOpen ? (
                <ChevronUp className="h-4 w-4 text-info-icon" />
              ) : (
                <ChevronDown className="h-4 w-4 text-info-icon" />
              )}
            </div>
          </CardHeader>
          {aiOpen && (
            <CardContent className="pt-0 text-sm text-info-fg">{result.aiSummary}</CardContent>
          )}
        </Card>
      ) : null}
      {result.aiSummary && !isAnalyzing && (
        <Alert className="mt-2">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription className="text-xs text-muted-foreground">
            AIレポートは参考情報です。投資判断の根拠とせず、必ず専門家にご相談ください。
          </AlertDescription>
        </Alert>
      )}

      <MobileSummaryCard
        result={result}
        input={input}
        yieldPct={yieldPct}
        netYieldPct={netYieldPct}
      />

      <MobileKpiGrid result={result} />

      <YieldGauge
        yieldPct={yieldPct}
        netYieldPct={netYieldPct}
        isGood={isGood}
        targetPct={targetPct}
        input={input}
        landPriceStats={landPriceStats}
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">ストレステストシナリオ（銀行融資審査用）</CardTitle>
        </CardHeader>
        <CardContent>
          <CustomScenarioSlider input={input} onStateChange={handleCustomStateChange} />
          <StressScenarioTable
            stressScenarios={result.stressScenarios ?? []}
            customState={customState}
            populationForecast={populationForecast}
            input={input}
            result={result}
          />
        </CardContent>
      </Card>

      {result.yieldScenarios && (
        <VacancyScenarioCard
          yieldScenarios={result.yieldScenarios}
          vacancyRate={input.vacancyRate}
          totalInvestment={result.totalInvestment}
        />
      )}

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

      {!isGood && (
        <Card className="border-warning-bg bg-warning-bg">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-warning-fg">
              <TrendingDown className="h-5 w-5" />
              {targetPct}%達成のために必要な改善（いずれか一方）
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="rounded-md bg-card/70 p-3">
              <p className="text-xs text-muted-foreground">
                土地価格 <strong>または</strong> 建築費のどちらか一方を削減する必要がある額
              </p>
              <p className="text-xl font-bold text-warning-fg">
                ▼ {formatMan(result.requiredCostReduction)}
              </p>
            </div>
            <div className="rounded-md bg-card/70 p-3">
              <p className="text-xs text-muted-foreground">または、必要な月額賃料（満室想定）</p>
              <p className="text-xl font-bold text-warning-fg">
                ▲ {formatYen(result.requiredMonthlyRent)}/月
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {isGood && (
        <Card className="border-success-bg bg-success-bg">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base text-success-fg">
              <TrendingUp className="h-5 w-5" />
              {targetPct}%超え達成！余裕度
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-success-fg">
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
      colorClass: "text-success-icon",
    },
    {
      label: "標準",
      vacancyPct: (Math.min(vacancyRate * 1.0, 0.99) * 100).toFixed(0),
      annualRent: yieldScenarios.standard.annualRent,
      effectiveYield: yieldScenarios.standard.annualRent / totalInvestment,
      colorClass: "text-info-icon",
    },
    {
      label: "悲観",
      vacancyPct: (Math.min(vacancyRate * 1.5, 0.99) * 100).toFixed(0),
      annualRent: yieldScenarios.pessimistic.annualRent,
      effectiveYield: yieldScenarios.pessimistic.annualRent / totalInvestment,
      colorClass: "text-danger-icon",
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

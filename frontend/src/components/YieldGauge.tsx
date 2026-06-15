"use client";
import React, { useMemo } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CheckCircle, AlertTriangle } from "lucide-react";
import { TermTooltip } from "@/components/ui/TermTooltip";
import { calcYieldBenchmark } from "@/lib/yieldBenchmark";
import type { InvestmentInput, LandPriceStats } from "@/types/investment";

interface YieldGaugeProps {
  yieldPct: number;
  totalInvestmentYieldPct?: number;
  netYieldPct: number;
  isGood: boolean;
  targetPct: number;
  input: InvestmentInput;
  landPriceStats?: LandPriceStats | null;
}

export function YieldGauge({
  yieldPct,
  totalInvestmentYieldPct,
  netYieldPct,
  isGood,
  targetPct,
  input,
  landPriceStats,
}: YieldGaugeProps) {
  const maxYieldPct = targetPct * 2;
  const gaugePosition = Math.min(yieldPct / maxYieldPct, 1) * 100;
  const targetPosition = 50;

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
    <Card className={`border-2 ${isGood ? "border-green-400" : "border-red-400"}`}>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between">
          <div className="min-w-0 flex-1">
            <p className="text-xs text-muted-foreground sm:text-sm">
              <TermTooltip term="grossYield">表面利回り</TermTooltip>（満室想定年収 / 物件価格）
            </p>
            <div className="flex items-end gap-2">
              <span
                data-testid="gross-yield-value"
                className={`text-4xl font-bold sm:text-5xl ${isGood ? "text-green-600" : "text-red-600"}`}
              >
                {yieldPct.toFixed(2)}
              </span>
              <span className="mb-1 text-xl font-semibold text-muted-foreground sm:mb-2 sm:text-2xl">
                %
              </span>
            </div>
            {totalInvestmentYieldPct != null && (
              <p className="mt-1 text-xs text-muted-foreground sm:text-sm">
                <TermTooltip term="totalInvestmentYield">総投資利回り</TermTooltip>
                （諸費用込み）：{totalInvestmentYieldPct.toFixed(2)}%
              </p>
            )}
            <p className="mt-1 text-xs text-muted-foreground sm:text-sm">
              <TermTooltip term="netYield">実質利回り</TermTooltip>（空室・経費控除後）：
              {netYieldPct.toFixed(2)}%
            </p>
          </div>
          <div className="ml-3 flex flex-col items-center gap-2 shrink-0">
            {isGood ? (
              <>
                <CheckCircle className="h-10 w-10 text-green-500 sm:h-12 sm:w-12" />
                <Badge data-testid="yield-threshold-badge" variant="success">
                  {targetPct}%超え ✓
                </Badge>
              </>
            ) : (
              <>
                <AlertTriangle className="h-10 w-10 text-red-500 sm:h-12 sm:w-12" />
                <Badge data-testid="yield-threshold-badge" variant="danger">
                  {targetPct}%未満 ✗
                </Badge>
              </>
            )}
          </div>
        </div>

        <div className="mt-4">
          <div className="flex justify-between text-xs text-muted-foreground mb-1">
            <span>0%</span>
            <span className="font-semibold text-orange-500">目標 {targetPct}%</span>
            <span>{maxYieldPct}%+</span>
          </div>
          <div className="relative h-3 rounded-full bg-muted overflow-hidden">
            <div className="absolute inset-y-0 left-0 w-1/3 bg-red-400" />
            <div className="absolute inset-y-0 left-1/3 w-1/3 bg-yellow-400" />
            <div className="absolute inset-y-0 left-2/3 w-1/3 bg-green-500" />
            <div
              className="absolute top-0 h-full w-1 bg-foreground/80 rounded"
              style={{ left: `${gaugePosition}%` }}
            />
            <div
              className="absolute top-0 h-full w-0.5 bg-orange-500"
              style={{ left: `${targetPosition}%` }}
            />
          </div>
        </div>

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
  );
}

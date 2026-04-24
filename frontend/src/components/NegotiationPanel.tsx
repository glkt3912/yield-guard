"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceComparison,
  TheoreticalPriceResult,
} from "@/types/investment";
import { formatYen, formatPct } from "@/lib/utils";
import { TrendingDown, TrendingUp, Minus } from "lucide-react";

const TSUBO_PER_SQM = 3.30578;

function DiffBadge({ diff, pct }: { diff: number; pct: number }) {
  if (Math.abs(pct) < 0.1) {
    return (
      <span className="inline-flex items-center gap-1 text-sm text-muted-foreground">
        <Minus className="h-3.5 w-3.5" />
        ほぼ同額
      </span>
    );
  }
  const expensive = diff > 0;
  return (
    <span
      className={`inline-flex items-center gap-1 text-sm font-medium ${expensive ? "text-red-600" : "text-emerald-600"}`}
    >
      {expensive ? (
        <TrendingUp className="h-3.5 w-3.5" />
      ) : (
        <TrendingDown className="h-3.5 w-3.5" />
      )}
      {expensive ? "▲" : "▼"}
      {formatYen(Math.abs(diff))}（{expensive ? "+" : ""}
      {pct.toFixed(1)}%）
    </span>
  );
}

interface Props {
  result: InvestmentResult;
  input: InvestmentInput;
  comparison: LandPriceComparison | null;
  theoreticalPrice: TheoreticalPriceResult | null;
}

export default function NegotiationPanel({
  result,
  input,
  comparison,
  theoreticalPrice,
}: Props) {
  const askingPropertyPrice = input.landPrice + input.buildingCost;

  // Max purchase price = max total investment / (1 + miscExpenseRate)
  const maxTotalInvestment = result.totalInvestment - result.requiredCostReduction;
  const maxPropertyPrice =
    input.miscExpenseRate > -1
      ? maxTotalInvestment / (1 + input.miscExpenseRate)
      : maxTotalInvestment;

  const diffFromAsking = maxPropertyPrice - askingPropertyPrice;
  const diffPct =
    askingPropertyPrice > 0 ? (diffFromAsking / askingPropertyPrice) * 100 : 0;

  // Market-implied property price from land transaction data
  const marketLandValue =
    comparison && input.landArea > 0
      ? comparison.stats.medianTsubo * (input.landArea / TSUBO_PER_SQM)
      : null;
  const marketPropertyPrice =
    marketLandValue !== null ? marketLandValue + input.buildingCost : null;
  const marketDiff =
    marketPropertyPrice !== null ? askingPropertyPrice - marketPropertyPrice : null;
  const marketDiffPct =
    marketPropertyPrice !== null && marketPropertyPrice > 0
      ? (marketDiff! / marketPropertyPrice) * 100
      : null;

  // Theoretical price from estimate endpoint
  const theoreticalPropertyPrice = theoreticalPrice?.theoreticalPriceJPY ?? null;
  const theoreticalDiff =
    theoreticalPropertyPrice !== null
      ? askingPropertyPrice - theoreticalPropertyPrice
      : null;
  const theoreticalDiffPct =
    theoreticalPropertyPrice !== null && theoreticalPropertyPrice > 0
      ? (theoreticalDiff! / theoreticalPropertyPrice) * 100
      : null;

  // Negotiation range: between theoretical/market lower bound and max purchase price
  const rangeAnchors = [
    theoreticalPropertyPrice,
    marketPropertyPrice,
    maxPropertyPrice,
  ].filter((v): v is number => v !== null && v > 0);
  const rangeLow = rangeAnchors.length > 0 ? Math.min(...rangeAnchors) : null;
  const rangeHigh = rangeAnchors.length > 0 ? Math.max(...rangeAnchors) : null;

  const showNegotiationSection =
    (comparison !== null && marketPropertyPrice !== null) ||
    theoreticalPropertyPrice !== null;

  return (
    <Card className="rounded-xl shadow-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-base font-semibold">
          逆算・交渉シミュレーション
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Section 1: Reverse yield calculation (#296) */}
        <section>
          <h3 className="mb-3 text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            逆算モード
          </h3>
          <p className="mb-3 text-xs text-muted-foreground">
            目標利回り{" "}
            <span className="font-semibold text-foreground">
              {formatPct(input.yieldTarget)}
            </span>{" "}
            ／ 想定賃料{" "}
            <span className="font-semibold text-foreground">
              ¥{input.monthlyRent.toLocaleString("ja-JP")}
            </span>{" "}
            円/月
          </p>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div className="rounded-lg bg-muted/50 p-3">
              <p className="text-xs text-muted-foreground mb-1">最大買付可能価格</p>
              <p className="text-lg font-bold">{formatYen(maxPropertyPrice)}</p>
            </div>
            <div className="rounded-lg bg-muted/50 p-3">
              <p className="text-xs text-muted-foreground mb-1">売出価格との差額</p>
              {askingPropertyPrice > 0 ? (
                <DiffBadge diff={-diffFromAsking} pct={-diffPct} />
              ) : (
                <p className="text-sm text-muted-foreground">—</p>
              )}
              {askingPropertyPrice > 0 && diffFromAsking < 0 && (
                <p className="mt-1 text-xs text-muted-foreground">
                  値引き必要額: {formatYen(Math.abs(diffFromAsking))}
                </p>
              )}
            </div>
            <div className="rounded-lg bg-muted/50 p-3">
              <p className="text-xs text-muted-foreground mb-1">
                売出価格での表面利回り
              </p>
              <p
                className={`text-lg font-bold ${result.isAboveYieldTarget ? "text-emerald-600" : "text-red-600"}`}
              >
                {formatPct(result.grossYield)}
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                目標 {formatPct(input.yieldTarget)}
              </p>
            </div>
          </div>
        </section>

        {/* Section 2: Negotiation simulation (#297) */}
        {showNegotiationSection && (
          <section className="border-t pt-4">
            <h3 className="mb-3 text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              交渉シミュレーション
            </h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <tbody className="divide-y divide-muted/50">
                  <tr>
                    <td className="py-2 pr-4 text-muted-foreground whitespace-nowrap">
                      売出価格
                    </td>
                    <td className="py-2 font-semibold">
                      {formatYen(askingPropertyPrice)}
                    </td>
                    <td className="py-2 pl-4" />
                  </tr>
                  {marketPropertyPrice !== null && marketDiff !== null && marketDiffPct !== null && (
                    <tr>
                      <td className="py-2 pr-4 text-muted-foreground whitespace-nowrap">
                        市場取引実勢
                      </td>
                      <td className="py-2 font-semibold">
                        {formatYen(marketPropertyPrice)}
                      </td>
                      <td className="py-2 pl-4">
                        <DiffBadge diff={marketDiff} pct={marketDiffPct} />
                      </td>
                    </tr>
                  )}
                  {theoreticalPropertyPrice !== null && theoreticalDiff !== null && theoreticalDiffPct !== null && (
                    <tr>
                      <td className="py-2 pr-4 text-muted-foreground whitespace-nowrap">
                        理論価格
                      </td>
                      <td className="py-2 font-semibold">
                        {formatYen(theoreticalPropertyPrice)}
                      </td>
                      <td className="py-2 pl-4">
                        <DiffBadge diff={theoreticalDiff} pct={theoreticalDiffPct} />
                      </td>
                    </tr>
                  )}
                  {result.requiredCostReduction > 0 && (
                    <tr>
                      <td className="py-2 pr-4 text-muted-foreground whitespace-nowrap">
                        目標利回り達成に必要な値引き
                      </td>
                      <td className="py-2 font-semibold text-red-600">
                        {formatYen(result.requiredCostReduction)}
                      </td>
                      <td className="py-2 pl-4 text-red-600 text-sm">
                        ({formatPct(-result.requiredCostReduction / result.totalInvestment)})
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            {rangeLow !== null && rangeHigh !== null && rangeLow !== rangeHigh && (
              <div className="mt-3 rounded-lg bg-blue-50 dark:bg-blue-950/30 p-3">
                <p className="text-xs text-muted-foreground mb-1">
                  交渉推奨価格レンジ
                </p>
                <p className="text-sm font-semibold text-blue-700 dark:text-blue-300">
                  {formatYen(rangeLow)} 〜 {formatYen(rangeHigh)}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  この範囲での成約であれば目標利回りを満たしつつ市場実勢に沿った交渉が可能です
                </p>
              </div>
            )}
          </section>
        )}
      </CardContent>
    </Card>
  );
}

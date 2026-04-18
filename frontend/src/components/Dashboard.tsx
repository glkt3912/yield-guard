"use client";
import React, { useState } from "react";
import { InvestmentForm } from "@/components/InvestmentForm";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import { CashFlowChart } from "@/components/CashFlowChart";
import { DeadCrossChart } from "@/components/DeadCrossChart";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import CostBreakdown from "@/components/CostBreakdown";
import type { InvestmentInput, InvestmentResult, LandPriceComparison, TheoreticalPriceResult, StationRidershipResult, PopulationForecastResult } from "@/types/investment";
import { analyze, compareLandPrice, estimateLandPrice, fetchStationRidership, fetchPopulationForecast } from "@/lib/api";
import { ShieldAlert, XOctagon, AlertTriangle } from "lucide-react";

/** 直近2年分の期間（国交省API形式: YYYYQ） */
function getCurrentPeriods(): { year: number; quarter: number; toYear: number; toQuarter: number } {
  const now = new Date();
  const toYear = now.getFullYear();
  const toQuarter = Math.ceil((now.getMonth() + 1) / 3);
  return { year: toYear - 2, quarter: 1, toYear, toQuarter };
}

export function Dashboard() {
  const [result, setResult] = useState<InvestmentResult | null>(null);
  const [comparison, setComparison] = useState<LandPriceComparison | null>(null);
  const [theoreticalPrice, setTheoreticalPrice] = useState<TheoreticalPriceResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastInput, setLastInput] = useState<InvestmentInput | null>(null);
  const [stationRidership, setStationRidership] = useState<StationRidershipResult[] | null>(null);
  const [populationForecast, setPopulationForecast] = useState<PopulationForecastResult | null>(null);

  const handleAnalyze = async (input: InvestmentInput) => {
    setLoading(true);
    setError(null);
    try {
      const res = await analyze(input);
      setResult(res);
      setLastInput(input);
    } catch (e) {
      setError(e instanceof Error ? e.message : "シミュレーションに失敗しました");
    } finally {
      setLoading(false);
    }
  };

  const handleFetchLandPrices = async (area: string, city: string, lat?: number, lng?: number) => {
    setLoading(true);
    setError(null);
    setStationRidership(null);
    setPopulationForecast(null);
    const { year, quarter, toYear, toQuarter } = getCurrentPeriods();
    try {
      const baseParams = {
        area,
        city,
        year,
        quarter,
        toYear,
        toQuarter,
        price: lastInput?.landPrice ?? 5_000_000,
        areaSqm: lastInput?.landArea ?? 0,
      };
      const comp = await compareLandPrice(baseParams);
      setComparison(comp);

      if (lastInput && lastInput.landArea > 0) {
        try {
          const est = await estimateLandPrice({
            ...baseParams,
            buildingAge: lastInput.buildingAge,
            stationMinutes: lastInput.stationMinutes,
          });
          setTheoreticalPrice(est);
        } catch {
          setTheoreticalPrice(null);
        }
      }
      if (lat !== undefined && lng !== undefined) {
        const [ridership, population] = await Promise.allSettled([
          fetchStationRidership({ lat, lng }),
          fetchPopulationForecast({ lat, lng }),
        ]);
        setStationRidership(ridership.status === "fulfilled" ? ridership.value : null);
        setPopulationForecast(population.status === "fulfilled" ? population.value : null);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "相場データの取得に失敗しました");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-white px-6 py-4 shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center gap-3">
          <ShieldAlert className="h-7 w-7 text-primary" />
          <div>
            <h1 className="text-xl font-bold text-foreground">Yield-Guard</h1>
            <p className="text-xs text-muted-foreground">不動産投資リスク可視化ツール</p>
          </div>
          <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
            <span className="h-2 w-2 rounded-full bg-green-400" />
            国交省API使用
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6">
        {error && (
          <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            ⚠ {error}
          </div>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[400px_1fr]">
          <aside>
            <InvestmentForm
              onAnalyze={handleAnalyze}
              onFetchLandPrices={handleFetchLandPrices}
              loading={loading}
            />
          </aside>

          <section className="space-y-6">
            {!result && !comparison && (
              <div className="flex h-80 items-center justify-center rounded-xl border-2 border-dashed border-muted-foreground/20">
                <div className="text-center text-muted-foreground">
                  <ShieldAlert className="mx-auto mb-3 h-12 w-12 opacity-30" />
                  <p className="text-sm font-medium">左のフォームから条件を入力して</p>
                  <p className="text-sm">シミュレーションを実行してください</p>
                </div>
              </div>
            )}

            {comparison && <LandPriceAnalysis comparison={comparison} input={lastInput} theoreticalPrice={theoreticalPrice} stationRidership={stationRidership} populationForecast={populationForecast} />}

            {result && lastInput && (
              <>
                {result.criticalErrors.length > 0 && (
                  <div className="space-y-2">
                    {result.criticalErrors.map((err) => (
                      <div
                        key={err.code}
                        className={`flex items-start gap-3 rounded-md border-2 p-4 ${
                          err.status === "REJECT"
                            ? "border-red-500 bg-red-50 text-red-900"
                            : "border-yellow-400 bg-yellow-50 text-yellow-900"
                        }`}
                      >
                        {err.status === "REJECT"
                          ? <XOctagon className="h-5 w-5 shrink-0 mt-0.5 text-red-600" />
                          : <AlertTriangle className="h-5 w-5 shrink-0 mt-0.5 text-yellow-600" />
                        }
                        <div>
                          <p className="text-sm font-bold">
                            {err.status === "REJECT" ? "⛔ 一発退場" : "⚠ 警告"}: {err.code}
                          </p>
                          <p className="text-sm mt-0.5">{err.message}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                <YieldAnalysis result={result} input={lastInput} populationForecast={populationForecast} />
                {result.acquisitionCosts && (
                  <div className="rounded-xl border bg-white p-5 shadow-sm">
                    <CostBreakdown
                      input={lastInput}
                      acquisitionCosts={result.acquisitionCosts}
                      yearlyResults={result.yearlyResults}
                    />
                  </div>
                )}
                {/* 自己資金 = 総投資額 - ローン金額（ISSUE-22: 投資回収年の正確な計算に使用） */}
                <CashFlowChart result={result} equityInvested={result.totalInvestment - lastInput.loanAmount} />
                <DeadCrossChart result={result} />
              </>
            )}
          </section>
        </div>
      </main>
    </div>
  );
}

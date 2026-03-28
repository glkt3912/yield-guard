"use client";
import React, { useState } from "react";
import { InvestmentForm } from "@/components/InvestmentForm";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import { CashFlowChart } from "@/components/CashFlowChart";
import { DeadCrossChart } from "@/components/DeadCrossChart";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import type { InvestmentInput, InvestmentResult, LandPriceComparison } from "@/types/investment";
import { analyze, compareLandPrice } from "@/lib/api";
import { ShieldAlert } from "lucide-react";

/** 直近2年分の期間（国交省API形式: YYYYQ） */
function getCurrentPeriods(): { from: string; to: string } {
  const year = new Date().getFullYear();
  return { from: `${year - 2}1`, to: `${year}4` };
}

export function Dashboard() {
  const [result, setResult] = useState<InvestmentResult | null>(null);
  const [comparison, setComparison] = useState<LandPriceComparison | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastInput, setLastInput] = useState<InvestmentInput | null>(null);

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

  const handleFetchLandPrices = async (area: string, city: string) => {
    setLoading(true);
    setError(null);
    const { from, to } = getCurrentPeriods();
    try {
      const comp = await compareLandPrice({
        area,
        city,
        from,
        to,
        price: lastInput?.landPrice ?? 5_000_000,
        areaSqm: lastInput?.landArea ?? 0, // ユーザー入力の面積を使用（ISSUE-15）
      });
      setComparison(comp);
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

            {comparison && <LandPriceAnalysis comparison={comparison} />}

            {result && lastInput && (
              <>
                <YieldAnalysis result={result} />
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

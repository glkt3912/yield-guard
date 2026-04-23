"use client";
import React, { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { encodeUrlParams, decodeUrlParams } from "@/lib/urlParams";
import { InvestmentForm } from "@/components/InvestmentForm";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import { CashFlowChart } from "@/components/CashFlowChart";
import { DeadCrossChart } from "@/components/DeadCrossChart";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import CostBreakdown from "@/components/CostBreakdown";
import { LoanOptimizationPanel } from "@/components/LoanOptimizationPanel";
import RenovationPanel from "@/components/RenovationPanel";
import type { InvestmentInput, InvestmentResult, LandPriceComparison, TheoreticalPriceResult, StationRidershipResult, PopulationForecastResult, AppraisalComparisonResult, UrbanRisk, SimulationMode, LoanMethod } from "@/types/investment";
import { analyze, compareLandPrice, estimateLandPrice, fetchStationRidership, fetchPopulationForecast, fetchLandAppraisals, fetchUrbanRisks } from "@/lib/api";
import { ShieldAlert, Info, FileDown, Share2, Check } from "lucide-react";
import { CriticalErrorBanner } from "@/components/CriticalErrorBanner";
import { downloadReportPDF } from "@/lib/generatePdf";

/** 直近2年分の期間（国交省API形式: YYYYQ） */
function getCurrentPeriods(): { year: number; quarter: number; toYear: number; toQuarter: number } {
  const now = new Date();
  const toYear = now.getFullYear();
  const toQuarter = Math.ceil((now.getMonth() + 1) / 3);
  return { year: toYear - 2, quarter: 1, toYear, toQuarter };
}

interface DashboardProps {
  initialParams?: URLSearchParams | null;
}

export function Dashboard({ initialParams }: DashboardProps = {}) {
  const router = useRouter();

  // Decode URL params once on mount
  const decoded = initialParams ? decodeUrlParams(initialParams) : null;

  const [result, setResult] = useState<InvestmentResult | null>(null);
  const [comparison, setComparison] = useState<LandPriceComparison | null>(null);
  const [theoreticalPrice, setTheoreticalPrice] = useState<TheoreticalPriceResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastInput, setLastInput] = useState<InvestmentInput | null>(null);
  const [stationRidership, setStationRidership] = useState<StationRidershipResult[] | null>(null);
  const [populationForecast, setPopulationForecast] = useState<PopulationForecastResult | null>(null);
  const [landAppraisal, setLandAppraisal] = useState<AppraisalComparisonResult | null>(null);
  const [externalUrbanRisks, setExternalUrbanRisks] = useState<UrbanRisk[] | null>(null);
  const [simulationMode, setSimulationMode] = useState<SimulationMode>(
    decoded?.mode ?? "quick"
  );
  const [modeNotice, setModeNotice] = useState(false);
  const [pdfGenerating, setPdfGenerating] = useState(false);
  const [loanMethod, setLoanMethod] = useState<LoanMethod>("equal-payment");
  const [copied, setCopied] = useState(false);
  const noticeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const copiedTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (noticeTimer.current) clearTimeout(noticeTimer.current);
      if (copiedTimer.current) clearTimeout(copiedTimer.current);
    };
  }, []);

  const handleModeChange = (mode: SimulationMode) => {
    if (result) {
      if (noticeTimer.current) clearTimeout(noticeTimer.current);
      setModeNotice(true);
      noticeTimer.current = setTimeout(() => setModeNotice(false), 4000);
    }
    setSimulationMode(mode);
    setResult(null);
  };

  const handleAnalyze = async (input: InvestmentInput, quickTotalMan?: string) => {
    setLoading(true);
    setError(null);
    const inputWithMethod = { ...input, loanMethod };
    try {
      const res = await analyze(inputWithMethod);
      setResult(res);
      setLastInput(inputWithMethod);

      // Update URL with current simulation conditions
      const urlParams = encodeUrlParams(simulationMode, inputWithMethod, quickTotalMan);
      const qs = urlParams.toString();
      router.replace(qs ? `?${qs}` : "?", { scroll: false });
    } catch (e) {
      setError(e instanceof Error ? e.message : "シミュレーションに失敗しました");
    } finally {
      setLoading(false);
    }
  };

  const handleLoanMethodChange = async (method: LoanMethod) => {
    setLoanMethod(method);
    if (!lastInput) return;
    setLoading(true);
    setError(null);
    try {
      const res = await analyze({ ...lastInput, loanMethod: method });
      setResult(res);
      setLastInput((prev) => prev ? { ...prev, loanMethod: method } : prev);
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
    setLandAppraisal(null);
    setExternalUrbanRisks(null);
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
      const [appraisal] = await Promise.allSettled([
        fetchLandAppraisals({ area, year: toYear, city: city || undefined }),
      ]);
      setLandAppraisal(appraisal.status === "fulfilled" ? appraisal.value : null);

      if (lat !== undefined && lng !== undefined) {
        const [ridership, population, urbanRisks] = await Promise.allSettled([
          fetchStationRidership({ lat, lng }),
          fetchPopulationForecast({ lat, lng }),
          fetchUrbanRisks(lat, lng),
        ]);
        setStationRidership(ridership.status === "fulfilled" ? ridership.value : null);
        setPopulationForecast(population.status === "fulfilled" ? population.value : null);
        setExternalUrbanRisks(urbanRisks.status === "fulfilled" ? urbanRisks.value : null);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "相場データの取得に失敗しました。しばらく後に再試行してください");
    } finally {
      setLoading(false);
    }
  };

  const handleShare = () => {
    navigator.clipboard.writeText(window.location.href).then(() => {
      setCopied(true);
      if (copiedTimer.current) clearTimeout(copiedTimer.current);
      copiedTimer.current = setTimeout(() => setCopied(false), 1500);
    });
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
          <div className="ml-auto flex items-center gap-3">
            {result && lastInput && (
              <button
                onClick={handleShare}
                className="flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium shadow-sm hover:bg-muted transition-colors"
                title="この条件をクリップボードにコピー"
              >
                {copied ? (
                  <Check className="h-3.5 w-3.5 text-green-600" />
                ) : (
                  <Share2 className="h-3.5 w-3.5" />
                )}
                {copied ? "コピーしました" : "この条件を共有"}
              </button>
            )}
            {result && lastInput && (
              <button
                disabled={pdfGenerating}
                onClick={async () => {
                  setPdfGenerating(true);
                  try {
                    await downloadReportPDF(lastInput, result);
                  } catch {
                    setError("PDF の生成に失敗しました。しばらく後で再試行してください。");
                  } finally {
                    setPdfGenerating(false);
                  }
                }}
                className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-primary/90 disabled:opacity-60"
              >
                <FileDown className="h-3.5 w-3.5" />
                {pdfGenerating ? "生成中..." : "PDFレポート出力"}
              </button>
            )}
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="h-2 w-2 rounded-full bg-green-400" />
              国交省API使用
            </div>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6">
        {error && (
          <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            ⚠ {error}
          </div>
        )}

        {modeNotice && (
          <div className="mb-4 flex items-center gap-2 rounded-md border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800">
            <Info className="h-4 w-4 shrink-0" />
            モードを切り替えたため、結果をクリアしました。再度シミュレーションを実行してください。
          </div>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[400px_1fr]">
          <aside>
            <InvestmentForm
              onAnalyze={handleAnalyze}
              onFetchLandPrices={handleFetchLandPrices}
              loading={loading}
              simulationMode={simulationMode}
              onModeChange={handleModeChange}
              initialInput={decoded?.input}
              initialQuickTotalPriceMan={decoded?.quickTotalPriceMan}
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

            {comparison && <LandPriceAnalysis comparison={comparison} input={lastInput} theoreticalPrice={theoreticalPrice} stationRidership={stationRidership} populationForecast={populationForecast} landAppraisal={landAppraisal} externalUrbanRisks={externalUrbanRisks} />}

            {result && lastInput && (
              <>
                <CriticalErrorBanner errors={result.criticalErrors} />
                <YieldAnalysis result={result} input={lastInput} populationForecast={populationForecast} />
                <LoanOptimizationPanel
                  result={result}
                  loanMethod={loanMethod}
                  onLoanMethodChange={handleLoanMethodChange}
                  loanAmount={lastInput.loanAmount}
                />
                {simulationMode === "full" && (
                  <>
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
              </>
            )}
            <RenovationPanel />
          </section>
        </div>
      </main>
    </div>
  );
}

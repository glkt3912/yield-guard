"use client";
import React, { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { decodeUrlParams } from "@/lib/urlParams";
import { InvestmentForm } from "@/components/InvestmentForm";
import type { SimulationMode } from "@/types/investment";
import {
  ShieldAlert,
  Info,
  FileDown,
  Share2,
  Check,
  SlidersHorizontal,
  WifiOff,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { downloadReportPDF } from "@/lib/generatePdf";
import { FirstTimerGuide } from "@/components/FirstTimerGuide";
import { SAMPLE_PROPERTY, ONBOARDING_KEY } from "@/lib/sampleProperty";
import { FormSheet } from "@/components/FormSheet";
import { ResultsSection } from "@/components/ResultsSection";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import { useInvestmentSimulation } from "@/hooks/useInvestmentSimulation";

interface DashboardProps {
  initialParams?: URLSearchParams | null;
}

export function Dashboard({ initialParams }: DashboardProps = {}) {
  const router = useRouter();
  const decoded = initialParams ? decodeUrlParams(initialParams) : null;

  const isOnline = useNetworkStatus();

  const [simulationMode, setSimulationMode] = useState<SimulationMode>(decoded?.mode ?? "quick");
  const [modeNotice, setModeNotice] = useState(false);
  const [pdfGenerating, setPdfGenerating] = useState(false);
  const [pdfError, setPdfError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [mobileFormOpen, setMobileFormOpen] = useState(false);
  const [showGuide, setShowGuide] = useState(false);
  const [activeTab, setActiveTab] = useState<"simulation" | "area-discovery">("simulation");
  const [selectedMunicipalityMsg, setSelectedMunicipalityMsg] = useState<string | null>(null);

  const noticeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const copiedTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  const simulation = useInvestmentSimulation({
    simulationMode,
    onUrlUpdate: (qs) => router.replace(qs ? `?${qs}` : "?", { scroll: false }),
  });

  const {
    result,
    comparison,
    theoreticalPrice,
    stationRidership,
    populationForecast,
    landAppraisal,
    externalUrbanRisks,
    investmentScore,
    hazardRisks,
    loading,
    error,
    lastInput,
    propertyLat,
    propertyLng,
    monteCarloResult,
    monteCarloLoading,
    loanMethod,
    handleAnalyze,
    handleFetchLandPrices,
    handleMonteCarlo,
    handleLoanMethodChange,
    setPropertyCoords,
    clearResult,
  } = simulation;

  useEffect(() => {
    return () => {
      if (noticeTimer.current) clearTimeout(noticeTimer.current);
      if (copiedTimer.current) clearTimeout(copiedTimer.current);
    };
  }, []);

  useEffect(() => {
    if (mobileFormOpen) closeButtonRef.current?.focus();
  }, [mobileFormOpen]);

  useEffect(() => {
    try {
      if (!localStorage.getItem(ONBOARDING_KEY)) {
        setShowGuide(true);
      }
    } catch {
      // localStorage unavailable (e.g. test environment or private browsing)
    }
  }, []);

  const handleModeChange = (mode: SimulationMode) => {
    if (result) {
      if (noticeTimer.current) clearTimeout(noticeTimer.current);
      setModeNotice(true);
      noticeTimer.current = setTimeout(() => setModeNotice(false), 4000);
    }
    setSimulationMode(mode);
    clearResult();
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
      {showGuide && (
        <FirstTimerGuide
          sampleProperty={SAMPLE_PROPERTY}
          onUseSample={(input) => {
            localStorage.setItem(ONBOARDING_KEY, "1");
            setShowGuide(false);
            handleAnalyze(input);
          }}
          onDismiss={() => {
            localStorage.setItem(ONBOARDING_KEY, "1");
            setShowGuide(false);
          }}
        />
      )}
      <header className="border-b bg-white px-4 py-3 shadow-sm lg:px-6 lg:py-4">
        <div className="mx-auto flex max-w-7xl items-center gap-3">
          <ShieldAlert className="h-6 w-6 text-primary lg:h-7 lg:w-7" />
          <div>
            <h1 className="text-lg font-bold text-foreground lg:text-xl">Yield-Guard</h1>
            <p className="text-xs text-muted-foreground">不動産投資リスク可視化ツール</p>
          </div>
          <div className="ml-auto flex items-center gap-2 lg:gap-3">
            {result && lastInput && (
              <button
                onClick={handleShare}
                className="hidden items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium shadow-sm hover:bg-muted transition-colors lg:flex"
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
                  setPdfError(null);
                  try {
                    await downloadReportPDF(lastInput, result);
                  } catch {
                    setPdfError("PDF の生成に失敗しました。しばらく後で再試行してください。");
                  } finally {
                    setPdfGenerating(false);
                  }
                }}
                className="hidden items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-primary/90 disabled:opacity-60 lg:flex"
              >
                <FileDown className="h-3.5 w-3.5" />
                {pdfGenerating ? "生成中..." : "PDFレポート出力"}
              </button>
            )}
            {isOnline === true && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="h-2 w-2 rounded-full bg-green-400" />
                <span className="hidden sm:inline">国交省API使用</span>
              </div>
            )}
            {isOnline === false && (
              <div className="flex items-center gap-1.5 rounded-md border border-amber-300 bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-800">
                <WifiOff className="h-3.5 w-3.5" />
                オフライン
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-4 pb-24 lg:py-6 lg:pb-6">
        {error && (
          <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            ⚠ {error}
          </div>
        )}

        {pdfError && (
          <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            ⚠ {pdfError}
          </div>
        )}

        {modeNotice && (
          <div className="mb-4 flex items-center gap-2 rounded-md border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800">
            <Info className="h-4 w-4 shrink-0" />
            モードを切り替えたため、結果をクリアしました。再度シミュレーションを実行してください。
          </div>
        )}

        {isOnline === false && (
          <div className="mb-4 flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
            <WifiOff className="h-4 w-4 shrink-0" />
            <span>
              <strong>オフラインモード：</strong>
              シミュレーションはデバイス内で計算されます。「相場を取得」はネットワーク接続が回復するまで使用できません。
            </span>
          </div>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[400px_1fr]">
          {/* Form: hidden on mobile (shown in bottom sheet), visible on desktop */}
          <aside className="hidden lg:block">
            <InvestmentForm
              onAnalyze={(input, quickTotalMan) => {
                setMobileFormOpen(false);
                return handleAnalyze(input, quickTotalMan);
              }}
              onFetchLandPrices={handleFetchLandPrices}
              loading={loading}
              simulationMode={simulationMode}
              onModeChange={handleModeChange}
              initialInput={decoded?.input}
              initialQuickTotalPriceMan={decoded?.quickTotalPriceMan}
              isOnline={isOnline}
              externalLat={propertyLat}
              externalLng={propertyLng}
            />
          </aside>

          <ResultsSection
            activeTab={activeTab}
            setActiveTab={setActiveTab}
            selectedMunicipalityMsg={selectedMunicipalityMsg}
            setSelectedMunicipalityMsg={setSelectedMunicipalityMsg}
            simulationMode={simulationMode}
            result={result}
            comparison={comparison}
            theoreticalPrice={theoreticalPrice}
            stationRidership={stationRidership}
            populationForecast={populationForecast}
            landAppraisal={landAppraisal}
            externalUrbanRisks={externalUrbanRisks}
            investmentScore={investmentScore}
            hazardRisks={hazardRisks}
            lastInput={lastInput}
            propertyLat={propertyLat}
            propertyLng={propertyLng}
            monteCarloResult={monteCarloResult}
            monteCarloLoading={monteCarloLoading}
            onMonteCarlo={handleMonteCarlo}
            loanMethod={loanMethod}
            onLoanMethodChange={handleLoanMethodChange}
            onTileSelect={setPropertyCoords}
          />
        </div>
      </main>

      {/* Mobile: floating "条件を編集" button */}
      <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 lg:hidden">
        <button
          onClick={() => setMobileFormOpen(true)}
          className="flex min-h-[44px] items-center gap-2 rounded-full bg-primary px-6 py-3 text-sm font-semibold text-white shadow-lg hover:bg-primary/90 active:scale-95 transition-transform"
          aria-label="条件を編集"
        >
          <SlidersHorizontal className="h-4 w-4" />
          条件を編集
        </button>
      </div>

      {/* Mobile: bottom sheet overlay */}
      <FormSheet
        isOpen={mobileFormOpen}
        onClose={() => setMobileFormOpen(false)}
        closeButtonRef={closeButtonRef}
      >
        <InvestmentForm
          onAnalyze={(input, quickTotalMan) => {
            setMobileFormOpen(false);
            return handleAnalyze(input, quickTotalMan);
          }}
          onFetchLandPrices={handleFetchLandPrices}
          loading={loading}
          simulationMode={simulationMode}
          onModeChange={handleModeChange}
          initialInput={lastInput ?? decoded?.input}
          initialQuickTotalPriceMan={lastInput ? undefined : decoded?.quickTotalPriceMan}
          isOnline={isOnline}
          externalLat={propertyLat}
          externalLng={propertyLng}
          showModeToggle={true}
        />
      </FormSheet>
    </div>
  );
}

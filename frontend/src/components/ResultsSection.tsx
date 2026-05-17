"use client";
import React, { useCallback, useState } from "react";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import { StatusSummary } from "@/components/StatusSummary";
import { KpiStrip } from "@/components/KpiStrip";
import CashFlowChart from "@/components/CashFlowChart";
import DeadCrossChart from "@/components/DeadCrossChart";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import CostBreakdown from "@/components/CostBreakdown";
import { LoanOptimizationPanel } from "@/components/LoanOptimizationPanel";
import RenovationPanel from "@/components/RenovationPanel";
import MultiExitCompareTable from "@/components/MultiExitCompareTable";
import { InvestmentScoreCard } from "@/components/InvestmentScoreCard";
import { CriticalErrorBanner } from "@/components/CriticalErrorBanner";
import { HazardAlertBanner } from "@/components/HazardAlertBanner";
import NegotiationPanel from "@/components/NegotiationPanel";
import WatchlistPanel from "@/components/WatchlistPanel";
import { AreaDiscovery } from "@/components/AreaDiscovery";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import dynamic from "next/dynamic";
import { AlertTriangle, BarChart3, ChevronDown, ChevronUp, ShieldAlert } from "lucide-react";
import { PREFECTURE_CENTERS } from "@/lib/prefectureCenters";
import type {
  InvestmentInput,
  InvestmentResult,
  LandPriceComparison,
  TheoreticalPriceResult,
  StationRidershipResult,
  PopulationForecastResult,
  AppraisalComparisonResult,
  UrbanRisk,
  SimulationMode,
  LoanMethod,
  MonteCarloResult,
  InvestmentScoreResult,
} from "@/types/investment";

const InvestmentScoreHeatmap = dynamic(() => import("./InvestmentScoreHeatmap"), { ssr: false });
const MonteCarloChart = dynamic(
  () => import("@/components/MonteCarloChart").then((m) => ({ default: m.MonteCarloChart })),
  {
    loading: () => <Skeleton className="h-64 w-full rounded-xl" />,
  }
);
const LoanComparePanel = dynamic(
  () => import("@/components/LoanComparePanel").then((m) => ({ default: m.LoanComparePanel })),
  {
    loading: () => <Skeleton className="h-64 w-full rounded-xl" />,
  }
);
const DueDiligenceChecklist = dynamic(() => import("@/components/DueDiligenceChecklist"), {
  loading: () => <Skeleton className="h-96 w-full rounded-xl" />,
});

type ResultsTab = "finance" | "loan";
const DEFAULT_RESULTS_TAB: ResultsTab = "finance";

interface ResultsSectionProps {
  activeTab: "simulation" | "area-discovery";
  setActiveTab: (tab: "simulation" | "area-discovery") => void;
  selectedMunicipalityMsg: string | null;
  setSelectedMunicipalityMsg: (msg: string | null) => void;
  simulationMode: SimulationMode;
  result: InvestmentResult | null;
  comparison: LandPriceComparison | null;
  theoreticalPrice: TheoreticalPriceResult | null;
  stationRidership: StationRidershipResult[] | null;
  populationForecast: PopulationForecastResult | null;
  landAppraisal: AppraisalComparisonResult | null;
  externalUrbanRisks: UrbanRisk[] | null;
  investmentScore: InvestmentScoreResult | null;
  hazardRisks: UrbanRisk[] | null;
  lastInput: InvestmentInput | null;
  propertyLat: number | undefined;
  propertyLng: number | undefined;
  monteCarloResult: MonteCarloResult | null;
  monteCarloLoading: boolean;
  onMonteCarlo: () => Promise<void>;
  loanMethod: LoanMethod;
  onLoanMethodChange: (method: LoanMethod) => Promise<void>;
  onTileSelect: (lat: number, lng: number) => void;
  onApplyRecommend?: (vacancyRate: number, rentDeclineRate: number) => void;
  loading?: boolean;
}

export function ResultsSection({
  activeTab,
  setActiveTab,
  selectedMunicipalityMsg,
  setSelectedMunicipalityMsg,
  simulationMode,
  result,
  comparison,
  theoreticalPrice,
  stationRidership,
  populationForecast,
  landAppraisal,
  externalUrbanRisks,
  investmentScore,
  hazardRisks,
  lastInput,
  propertyLat,
  propertyLng,
  monteCarloResult,
  monteCarloLoading,
  onMonteCarlo,
  loanMethod,
  onLoanMethodChange,
  onTileSelect,
  onApplyRecommend,
  loading = false,
}: ResultsSectionProps) {
  const [municipalityCenter, setMunicipalityCenter] = useState<{
    lat: number;
    lng: number;
  } | null>(null);

  const [tabState, setTabState] = useState<{
    tab: ResultsTab;
    seenResult: InvestmentResult | null;
  }>({ tab: DEFAULT_RESULTS_TAB, seenResult: null });

  // result が変わったら概要タブへ自動リセット（useEffect不使用）
  const resultsTab = result !== tabState.seenResult ? DEFAULT_RESULTS_TAB : tabState.tab;

  const handleResultsTabChange = (tab: ResultsTab) => {
    setTabState({ tab, seenResult: result });
  };

  const handleAreaTileSelect = useCallback(
    (lat: number, lng: number) => {
      onTileSelect(lat, lng);
      setActiveTab("simulation");
    },
    [onTileSelect, setActiveTab]
  );

  return (
    <section className="space-y-6">
      {/* 最上位タブ: シミュレーション | エリアを探す */}
      <div className="flex gap-1 rounded-lg border bg-muted/30 p-1 w-fit">
        <button
          onClick={() => setActiveTab("simulation")}
          className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
            activeTab === "simulation"
              ? "bg-card shadow-sm text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          シミュレーション
        </button>
        <button
          onClick={() => {
            setActiveTab("area-discovery");
            setSelectedMunicipalityMsg(null);
            setMunicipalityCenter(null);
          }}
          className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
            activeTab === "area-discovery"
              ? "bg-card shadow-sm text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          エリアを探す
        </button>
      </div>

      {activeTab === "area-discovery" && (
        <>
          {selectedMunicipalityMsg && (
            <div className="rounded-md border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800">
              {selectedMunicipalityMsg}
            </div>
          )}
          <AreaDiscovery
            onMunicipalitySelect={(_code, name, prefecture) => {
              setSelectedMunicipalityMsg(
                `「${name}」を選択しました。下の地図上で物件位置をクリックしてください。`
              );
              setMunicipalityCenter(
                PREFECTURE_CENTERS[prefecture] ?? { lat: 35.6812, lng: 139.7671 }
              );
            }}
          />
          {municipalityCenter && (
            <InvestmentScoreHeatmap
              centerLat={municipalityCenter.lat}
              centerLng={municipalityCenter.lng}
              onTileSelect={handleAreaTileSelect}
            />
          )}
        </>
      )}

      {activeTab === "simulation" && loading && !result && (
        <div className="space-y-4">
          <Skeleton className="h-16 w-full rounded-xl" />
          <Skeleton className="h-32 w-full rounded-xl" />
          <Skeleton className="h-48 w-full rounded-xl" />
        </div>
      )}

      {activeTab === "simulation" && !loading && !result && !comparison && (
        <div className="flex h-64 items-center justify-center rounded-xl border-2 border-dashed border-muted-foreground/20 lg:h-80">
          <div className="text-center text-muted-foreground">
            <ShieldAlert className="mx-auto mb-3 h-10 w-10 opacity-30 lg:h-12 lg:w-12" />
            <p className="text-sm font-medium">条件を入力してシミュレーションを実行してください</p>
            <p className="mt-1 text-xs lg:hidden">下の「条件を編集」ボタンから入力できます</p>
            <p className="mt-1 hidden text-sm lg:block">左のフォームから条件を入力してください</p>
          </div>
        </div>
      )}

      {activeTab === "simulation" && (
        <>
          {/* Hero: 判定・KPI を最上位に固定表示 */}
          {result && lastInput && (
            <div className="space-y-3">
              <CriticalErrorBanner errors={result.criticalErrors} />
              <StatusSummary result={result} />
              <KpiStrip
                result={result}
                yieldTarget={lastInput.yieldTarget}
                holdingYears={lastInput.holdingYears}
              />
            </div>
          )}
          <HazardAlertBanner hazardRisks={hazardRisks} externalUrbanRisks={externalUrbanRisks} />

          {/* コンテキストパネル: データが揃い次第タブ外に常時表示 */}
          {investmentScore && (
            <InvestmentScoreCard
              score={investmentScore}
              populationChangeRate={populationForecast?.changeRate30yr}
              onApplyRecommend={onApplyRecommend}
            />
          )}
          {propertyLat !== undefined && (
            <InvestmentScoreHeatmap
              centerLat={propertyLat}
              centerLng={propertyLng}
              onTileSelect={onTileSelect}
            />
          )}
          {comparison && (
            <LandPriceAnalysis
              comparison={comparison}
              input={lastInput}
              theoreticalPrice={theoreticalPrice}
              stationRidership={stationRidership}
              populationForecast={populationForecast}
              landAppraisal={landAppraisal}
              externalUrbanRisks={externalUrbanRisks}
              hazardRisks={hazardRisks}
            />
          )}

          {/* 結果タブナビゲーション */}
          {result && (
            <div
              role="tablist"
              aria-label="結果タブ"
              className="flex gap-1 rounded-lg border bg-muted/30 p-1 w-full sm:w-fit overflow-x-auto"
            >
              {[
                { id: "finance" as ResultsTab, label: "財務分析" },
                { id: "loan" as ResultsTab, label: "ローン・交渉" },
              ].map(({ id, label }) => (
                <button
                  key={id}
                  role="tab"
                  aria-selected={resultsTab === id}
                  onClick={() => handleResultsTabChange(id)}
                  className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors whitespace-nowrap ${
                    resultsTab === id
                      ? "bg-card shadow-sm text-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          )}

          {/* Tab: 財務分析 */}
          {result && lastInput && resultsTab === "finance" && (
            <div className="space-y-6">
              <YieldAnalysis
                result={result}
                input={lastInput}
                populationForecast={populationForecast}
                landPriceStats={comparison?.stats ?? null}
                isAnalyzing={loading}
              />
              {result.multiExitComparison && result.multiExitComparison.length > 0 && (
                <MultiExitCompareTable rows={result.multiExitComparison} />
              )}
              {simulationMode === "full" && (
                <>
                  {result.acquisitionCosts && (
                    <div className="rounded-xl border bg-card p-5 shadow-sm">
                      <CostBreakdown
                        input={lastInput}
                        acquisitionCosts={result.acquisitionCosts}
                        yearlyResults={result.yearlyResults}
                      />
                    </div>
                  )}
                  {/* 自己資金 = 総投資額 - ローン金額（ISSUE-22: 投資回収年の正確な計算に使用） */}
                  <CashFlowChart
                    result={result}
                    equityInvested={result.totalInvestment - lastInput.loanAmount}
                  />
                  <DeadCrossChart result={result} />
                  <div className="flex justify-center">
                    <Button onClick={onMonteCarlo} loading={monteCarloLoading} size="md">
                      モンテカルロ実行（1,000試行）
                    </Button>
                  </div>
                  {monteCarloResult && <MonteCarloChart result={monteCarloResult} />}
                </>
              )}
              {simulationMode === "quick" && (
                <div className="rounded-lg border border-dashed border-muted-foreground/30 p-4 text-center text-sm text-muted-foreground">
                  <p>キャッシュフローグラフ・デッドクロス分析・モンテカルロは</p>
                  <p className="font-medium mt-1">詳細モードで利用できます</p>
                </div>
              )}
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription className="text-xs text-muted-foreground">
                  計算結果は参考値です。実際の投資判断は専門家にご相談ください。本シミュレーションは消費税・損益通算の詳細を考慮しない場合があります。
                </AlertDescription>
              </Alert>
            </div>
          )}

          {/* Tab 3: ローン・交渉 */}
          {result && lastInput && resultsTab === "loan" && (
            <LoanTabContent
              result={result}
              lastInput={lastInput}
              comparison={comparison}
              theoreticalPrice={theoreticalPrice}
              loanMethod={loanMethod}
              onLoanMethodChange={onLoanMethodChange}
            />
          )}

          {/* アクションパネル: タブ外に常時表示（reload後も見える） */}
          <RenovationPanel />
          <WatchlistPanel currentResult={result ?? undefined} />
          {lastInput && (
            <DueDiligenceChecklist
              propertyKey={`${lastInput.landPrice}-${lastInput.buildingCost}-${lastInput.monthlyRent}-${lastInput.loanAmount}`}
            />
          )}
        </>
      )}
    </section>
  );
}

interface LoanTabContentProps {
  result: InvestmentResult;
  lastInput: InvestmentInput;
  comparison: LandPriceComparison | null;
  theoreticalPrice: TheoreticalPriceResult | null;
  loanMethod: LoanMethod;
  onLoanMethodChange: (method: LoanMethod) => Promise<void>;
}

function LoanTabContent({
  result,
  lastInput,
  comparison,
  theoreticalPrice,
  loanMethod,
  onLoanMethodChange,
}: LoanTabContentProps) {
  const [isLoanCompareOpen, setIsLoanCompareOpen] = useState(false);

  return (
    <div className="space-y-6">
      <NegotiationPanel
        result={result}
        input={lastInput}
        comparison={comparison}
        theoreticalPrice={theoreticalPrice}
      />
      <LoanOptimizationPanel
        result={result}
        loanMethod={loanMethod}
        onLoanMethodChange={onLoanMethodChange}
        loanAmount={lastInput.loanAmount}
      />
      <div className="rounded-lg border bg-card shadow-sm">
        <button
          type="button"
          onClick={() => setIsLoanCompareOpen((v) => !v)}
          aria-expanded={isLoanCompareOpen}
          className="flex w-full items-center justify-between px-5 py-4 text-left"
        >
          <span className="flex items-center gap-2 text-base font-semibold">
            <BarChart3 className="h-4 w-4 text-primary" />
            複数融資条件の横並び比較
          </span>
          {isLoanCompareOpen ? (
            <ChevronUp className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          )}
        </button>
        {isLoanCompareOpen && (
          <div className="border-t">
            <LoanComparePanel baseInput={lastInput} />
          </div>
        )}
      </div>
    </div>
  );
}

"use client";
import React from "react";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import CashFlowChart from "@/components/CashFlowChart";
import DeadCrossChart from "@/components/DeadCrossChart";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import CostBreakdown from "@/components/CostBreakdown";
import { LoanOptimizationPanel } from "@/components/LoanOptimizationPanel";
import RenovationPanel from "@/components/RenovationPanel";
import { InvestmentScoreCard } from "@/components/InvestmentScoreCard";
import { CriticalErrorBanner } from "@/components/CriticalErrorBanner";
import { MonteCarloChart } from "@/components/MonteCarloChart";
import NegotiationPanel from "@/components/NegotiationPanel";
import WatchlistPanel from "@/components/WatchlistPanel";
import { AreaDiscovery } from "@/components/AreaDiscovery";
import { SimulationModeToggle } from "@/components/SimulationModeToggle";
import { Button } from "@/components/ui/button";
import dynamic from "next/dynamic";
import { ShieldAlert } from "lucide-react";
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

interface ResultsSectionProps {
  activeTab: "simulation" | "area-discovery";
  setActiveTab: (tab: "simulation" | "area-discovery") => void;
  selectedMunicipalityMsg: string | null;
  setSelectedMunicipalityMsg: (msg: string | null) => void;
  simulationMode: SimulationMode;
  onModeChange: (mode: SimulationMode) => void;
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
}

export function ResultsSection({
  activeTab,
  setActiveTab,
  selectedMunicipalityMsg,
  setSelectedMunicipalityMsg,
  simulationMode,
  onModeChange,
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
}: ResultsSectionProps) {
  return (
    <section className="space-y-6">
      {/* Tab toggle */}
      <div className="flex gap-1 rounded-lg border bg-muted/30 p-1 w-fit">
        <button
          onClick={() => setActiveTab("simulation")}
          className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
            activeTab === "simulation"
              ? "bg-white shadow-sm text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          シミュレーション
        </button>
        <button
          onClick={() => {
            setActiveTab("area-discovery");
            setSelectedMunicipalityMsg(null);
          }}
          className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
            activeTab === "area-discovery"
              ? "bg-white shadow-sm text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          エリアを探す
        </button>
      </div>

      {/* Mobile: simulation mode toggle */}
      {activeTab === "simulation" && (
        <SimulationModeToggle mode={simulationMode} onChange={onModeChange} className="lg:hidden" />
      )}

      {activeTab === "area-discovery" && (
        <>
          {selectedMunicipalityMsg && (
            <div className="rounded-md border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800">
              {selectedMunicipalityMsg}
            </div>
          )}
          <AreaDiscovery
            onMunicipalitySelect={(_code, name) => {
              setSelectedMunicipalityMsg(
                `「${name}」を選択しました。下の地図上で物件位置をクリックしてください。`
              );
            }}
          />
        </>
      )}

      {activeTab === "simulation" && !result && !comparison && (
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
          {investmentScore && <InvestmentScoreCard score={investmentScore} />}

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

          {result && lastInput && (
            <>
              <CriticalErrorBanner errors={result.criticalErrors} />
              <YieldAnalysis
                result={result}
                input={lastInput}
                populationForecast={populationForecast}
              />
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
            </>
          )}
          <RenovationPanel />
          <WatchlistPanel />
        </>
      )}
    </section>
  );
}

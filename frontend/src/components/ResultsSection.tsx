"use client";
import React, { useState } from "react";
import { YieldAnalysis } from "@/components/YieldAnalysis";
import CashFlowChart from "@/components/CashFlowChart";
import DeadCrossChart from "@/components/DeadCrossChart";
import { LandPriceAnalysis } from "@/components/LandPriceAnalysis";
import CostBreakdown from "@/components/CostBreakdown";
import { LoanOptimizationPanel } from "@/components/LoanOptimizationPanel";
import { LoanComparePanel } from "@/components/LoanComparePanel";
import RenovationPanel from "@/components/RenovationPanel";
import { InvestmentScoreCard } from "@/components/InvestmentScoreCard";
import { CriticalErrorBanner } from "@/components/CriticalErrorBanner";
import { MonteCarloChart } from "@/components/MonteCarloChart";
import NegotiationPanel from "@/components/NegotiationPanel";
import WatchlistPanel from "@/components/WatchlistPanel";
import { AreaDiscovery } from "@/components/AreaDiscovery";
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

const PREFECTURE_CENTERS: Record<string, { lat: number; lng: number }> = {
  "01": { lat: 43.06, lng: 141.35 },
  "02": { lat: 40.82, lng: 140.74 },
  "03": { lat: 39.7, lng: 141.15 },
  "04": { lat: 38.27, lng: 140.87 },
  "05": { lat: 39.72, lng: 140.1 },
  "06": { lat: 38.24, lng: 140.36 },
  "07": { lat: 37.75, lng: 140.47 },
  "08": { lat: 36.34, lng: 140.45 },
  "09": { lat: 36.57, lng: 139.88 },
  "10": { lat: 36.39, lng: 139.06 },
  "11": { lat: 35.86, lng: 139.65 },
  "12": { lat: 35.61, lng: 140.12 },
  "13": { lat: 35.69, lng: 139.69 },
  "14": { lat: 35.45, lng: 139.64 },
  "15": { lat: 37.9, lng: 139.02 },
  "16": { lat: 36.7, lng: 137.21 },
  "17": { lat: 36.59, lng: 136.63 },
  "18": { lat: 36.06, lng: 136.22 },
  "19": { lat: 35.66, lng: 138.57 },
  "20": { lat: 36.65, lng: 138.18 },
  "21": { lat: 35.39, lng: 136.72 },
  "22": { lat: 34.98, lng: 138.38 },
  "23": { lat: 35.18, lng: 136.91 },
  "24": { lat: 34.73, lng: 136.51 },
  "25": { lat: 35.0, lng: 135.87 },
  "26": { lat: 35.02, lng: 135.75 },
  "27": { lat: 34.69, lng: 135.5 },
  "28": { lat: 34.69, lng: 135.18 },
  "29": { lat: 34.69, lng: 135.83 },
  "30": { lat: 34.22, lng: 135.17 },
  "31": { lat: 35.5, lng: 134.24 },
  "32": { lat: 35.47, lng: 133.06 },
  "33": { lat: 34.66, lng: 133.93 },
  "34": { lat: 34.4, lng: 132.46 },
  "35": { lat: 34.19, lng: 131.47 },
  "36": { lat: 34.07, lng: 134.55 },
  "37": { lat: 34.34, lng: 134.04 },
  "38": { lat: 33.84, lng: 132.77 },
  "39": { lat: 33.56, lng: 133.53 },
  "40": { lat: 33.61, lng: 130.42 },
  "41": { lat: 33.25, lng: 130.3 },
  "42": { lat: 32.74, lng: 129.87 },
  "43": { lat: 32.79, lng: 130.74 },
  "44": { lat: 33.24, lng: 131.61 },
  "45": { lat: 31.91, lng: 131.42 },
  "46": { lat: 31.56, lng: 130.56 },
  "47": { lat: 26.21, lng: 127.68 },
};

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
}: ResultsSectionProps) {
  const [municipalityCenter, setMunicipalityCenter] = useState<{
    lat: number;
    lng: number;
  } | null>(null);

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
            setMunicipalityCenter(null);
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
              onTileSelect={(lat, lng) => {
                onTileSelect(lat, lng);
                setActiveTab("simulation");
              }}
            />
          )}
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
              <LoanComparePanel baseInput={lastInput} />
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
          <WatchlistPanel currentResult={result ?? undefined} />
        </>
      )}
    </section>
  );
}

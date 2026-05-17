"use client";
import { useState, useCallback, useRef, useEffect } from "react";
import type {
  InvestmentInput,
  InvestmentResult,
  LoanMethod,
  SimulationMode,
  MonteCarloResult,
  LandPriceComparison,
  TheoreticalPriceResult,
  StationRidershipResult,
  PopulationForecastResult,
  AppraisalComparisonResult,
  UrbanRisk,
  InvestmentScoreResult,
} from "@/types/investment";
import { useAnalyzeApi } from "./useAnalyzeApi";
import { useExternalData } from "./useExternalData";
import { useMonteCarloSim } from "./useMonteCarloSim";

export interface UseInvestmentSimulationParams {
  simulationMode: SimulationMode;
  onUrlUpdate: (qs: string) => void;
}

export interface UseInvestmentSimulationResult {
  result: InvestmentResult | null;
  comparison: LandPriceComparison | null;
  theoreticalPrice: TheoreticalPriceResult | null;
  stationRidership: StationRidershipResult[] | null;
  populationForecast: PopulationForecastResult | null;
  landAppraisal: AppraisalComparisonResult | null;
  externalUrbanRisks: UrbanRisk[] | null;
  investmentScore: InvestmentScoreResult | null;
  hazardRisks: UrbanRisk[] | null;
  loading: boolean;
  error: string | null;
  lastInput: InvestmentInput | null;
  propertyLat: number | undefined;
  propertyLng: number | undefined;
  monteCarloResult: MonteCarloResult | null;
  monteCarloLoading: boolean;
  loanMethod: LoanMethod;
  handleAnalyze: (input: InvestmentInput, quickTotalMan?: string) => Promise<void>;
  handleFetchLandPrices: (area: string, city: string, lat?: number, lng?: number) => Promise<void>;
  handleMonteCarlo: () => Promise<void>;
  handleLoanMethodChange: (method: LoanMethod) => Promise<void>;
  setPropertyCoords: (lat: number, lng: number) => void;
  clearResult: () => void;
}

export function useInvestmentSimulation(
  params: UseInvestmentSimulationParams
): UseInvestmentSimulationResult {
  // Shared state owned by orchestrator
  const [result, setResult] = useState<InvestmentResult | null>(null);
  const [lastInput, setLastInput] = useState<InvestmentInput | null>(null);

  // Stable refs for params and mutable values accessed inside async callbacks
  const paramsRef = useRef(params);
  useEffect(() => {
    paramsRef.current = params;
  });

  const lastInputRef = useRef<InvestmentInput | null>(lastInput);
  useEffect(() => {
    lastInputRef.current = lastInput;
  }, [lastInput]);

  const loanMethodRef = useRef<LoanMethod>("equal-payment");

  // Sub-hook: Monte Carlo (declared first so its setter can be passed to analyze)
  const monteCarlo = useMonteCarloSim(lastInputRef);

  const handleClearMonteCarloResult = useCallback(() => {
    monteCarlo.setMonteCarloResult(null);
  }, [monteCarlo]);

  const handleResult = useCallback((res: InvestmentResult | null) => {
    setResult(res);
  }, []);

  const handleLastInput = useCallback((input: InvestmentInput | null) => {
    setLastInput(input);
  }, []);

  // Sub-hook: analyze API
  const analyze = useAnalyzeApi(
    lastInputRef,
    loanMethodRef,
    paramsRef,
    handleResult,
    handleLastInput,
    handleClearMonteCarloResult
  );

  // Sub-hook: external data (land prices, geo data)
  const external = useExternalData(lastInputRef);

  // Combine loading and error from both analyze and external data
  const loading = analyze.loading || external.loading;
  const error = analyze.error ?? external.error ?? monteCarlo.monteCarloError;

  const clearResult = useCallback(() => {
    setResult(null);
  }, []);

  return {
    result,
    comparison: external.comparison,
    theoreticalPrice: external.theoreticalPrice,
    stationRidership: external.stationRidership,
    populationForecast: external.populationForecast,
    landAppraisal: external.landAppraisal,
    externalUrbanRisks: external.externalUrbanRisks,
    investmentScore: external.investmentScore,
    hazardRisks: external.hazardRisks,
    loading,
    error,
    lastInput,
    propertyLat: external.propertyLat,
    propertyLng: external.propertyLng,
    monteCarloResult: monteCarlo.monteCarloResult,
    monteCarloLoading: monteCarlo.monteCarloLoading,
    loanMethod: analyze.loanMethod,
    handleAnalyze: analyze.handleAnalyze,
    handleFetchLandPrices: external.handleFetchLandPrices,
    handleMonteCarlo: monteCarlo.handleMonteCarlo,
    handleLoanMethodChange: analyze.handleLoanMethodChange,
    setPropertyCoords: external.setPropertyCoords,
    clearResult,
  };
}

"use client";
import { useState, useCallback, useRef, useEffect } from "react";
import {
  analyze as analyzeOnline,
  compareLandPrice,
  estimateLandPrice,
  fetchStationRidership,
  fetchPopulationForecast,
  fetchLandAppraisals,
  fetchUrbanRisks,
  fetchHazardInfo,
  fetchInvestmentScore,
  simulate,
} from "@/lib/api";
import { analyze as analyzeOffline } from "@/lib/investment";
import { encodeUrlParams } from "@/lib/urlParams";
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

interface AnalysisResult {
  result: InvestmentResult | null;
  comparison: LandPriceComparison | null;
  theoreticalPrice: TheoreticalPriceResult | null;
  stationRidership: StationRidershipResult[] | null;
  populationForecast: PopulationForecastResult | null;
  landAppraisal: AppraisalComparisonResult | null;
  externalUrbanRisks: UrbanRisk[] | null;
  investmentScore: InvestmentScoreResult | null;
  hazardRisks: UrbanRisk[] | null;
}

const INITIAL_ANALYSIS_RESULT: AnalysisResult = {
  result: null,
  comparison: null,
  theoreticalPrice: null,
  stationRidership: null,
  populationForecast: null,
  landAppraisal: null,
  externalUrbanRisks: null,
  investmentScore: null,
  hazardRisks: null,
};

function getCurrentPeriods(): { year: number; quarter: number; toYear: number; toQuarter: number } {
  const now = new Date();
  const toYear = now.getFullYear();
  const toQuarter = Math.ceil((now.getMonth() + 1) / 3);
  return { year: toYear - 2, quarter: 1, toYear, toQuarter };
}

export interface UseInvestmentSimulationParams {
  isOnline: boolean | null;
  simulationMode: SimulationMode;
  onUrlUpdate: (qs: string) => void;
}

export interface UseInvestmentSimulationResult extends Omit<AnalysisResult, never> {
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
  const [analysisResult, setAnalysisResult] = useState<AnalysisResult>(INITIAL_ANALYSIS_RESULT);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastInput, setLastInput] = useState<InvestmentInput | null>(null);
  const [propertyLat, setPropertyLat] = useState<number | undefined>(undefined);
  const [propertyLng, setPropertyLng] = useState<number | undefined>(undefined);
  const [monteCarloResult, setMonteCarloResult] = useState<MonteCarloResult | null>(null);
  const [monteCarloLoading, setMonteCarloLoading] = useState(false);
  const [loanMethod, setLoanMethod] = useState<LoanMethod>("equal-payment");

  // Stable refs for params to avoid stale closures in callbacks
  const paramsRef = useRef(params);
  useEffect(() => {
    paramsRef.current = params;
  });

  // Refs for mutable values accessed inside async callbacks
  const lastInputRef = useRef(lastInput);
  useEffect(() => {
    lastInputRef.current = lastInput;
  }, [lastInput]);
  const loanMethodRef = useRef<LoanMethod>("equal-payment");

  const handleAnalyze = useCallback(async (input: InvestmentInput, quickTotalMan?: string) => {
    const { isOnline, simulationMode, onUrlUpdate } = paramsRef.current;
    const currentLoanMethod = loanMethodRef.current;
    setLoading(true);
    setError(null);
    setMonteCarloResult(null);
    const inputWithMethod = { ...input, loanMethod: currentLoanMethod };
    try {
      const res =
        isOnline === false ? analyzeOffline(inputWithMethod) : await analyzeOnline(inputWithMethod);
      setAnalysisResult((prev) => ({ ...prev, result: res }));
      setLastInput(inputWithMethod);
      const urlParams = encodeUrlParams(simulationMode, inputWithMethod, quickTotalMan);
      const qs = urlParams.toString();
      onUrlUpdate(qs);
    } catch (e) {
      setError(e instanceof Error ? e.message : "シミュレーションに失敗しました");
    } finally {
      setLoading(false);
    }
  }, []);

  const handleMonteCarlo = useCallback(async () => {
    const current = lastInputRef.current;
    if (!current) return;
    setMonteCarloLoading(true);
    try {
      const res = await simulate({ base: current, simulations: 1000 });
      setMonteCarloResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : "モンテカルロシミュレーションに失敗しました");
    } finally {
      setMonteCarloLoading(false);
    }
  }, []);

  const handleLoanMethodChange = useCallback(async (method: LoanMethod) => {
    setLoanMethod(method);
    loanMethodRef.current = method;
    const current = lastInputRef.current;
    if (!current) return;
    const { isOnline } = paramsRef.current;
    setLoading(true);
    setError(null);
    setMonteCarloResult(null);
    try {
      const updatedInput = { ...current, loanMethod: method };
      const res =
        isOnline === false ? analyzeOffline(updatedInput) : await analyzeOnline(updatedInput);
      setAnalysisResult((prev) => ({ ...prev, result: res }));
      setLastInput((prev) => (prev ? { ...prev, loanMethod: method } : prev));
    } catch (e) {
      setError(e instanceof Error ? e.message : "シミュレーションに失敗しました");
    } finally {
      setLoading(false);
    }
  }, []);

  const handleFetchLandPrices = useCallback(
    async (area: string, city: string, lat?: number, lng?: number) => {
      const { isOnline } = paramsRef.current;
      setLoading(true);
      setError(null);
      setAnalysisResult((prev) => ({
        ...prev,
        stationRidership: null,
        populationForecast: null,
        landAppraisal: null,
        externalUrbanRisks: null,
        investmentScore: null,
        hazardRisks: null,
      }));
      if (lat !== undefined && lng !== undefined) {
        setPropertyLat(lat);
        setPropertyLng(lng);
      }
      const { year, quarter, toYear, toQuarter } = getCurrentPeriods();
      const current = lastInputRef.current;
      try {
        const baseParams = {
          area,
          city,
          year,
          quarter,
          toYear,
          toQuarter,
          price: current?.landPrice ?? 5_000_000,
          areaSqm: current?.landArea ?? 0,
        };
        const comp = await compareLandPrice(baseParams);
        setAnalysisResult((prev) => ({ ...prev, comparison: comp }));

        if (current && current.landArea > 0) {
          try {
            const est = await estimateLandPrice({
              ...baseParams,
              buildingAge: current.buildingAge,
              stationMinutes: current.stationMinutes,
            });
            setAnalysisResult((prev) => ({ ...prev, theoreticalPrice: est }));
          } catch {
            setAnalysisResult((prev) => ({ ...prev, theoreticalPrice: null }));
          }
        }

        const [appraisal] = await Promise.allSettled([
          fetchLandAppraisals({ area, year: toYear, city: city || undefined }),
        ]);
        setAnalysisResult((prev) => ({
          ...prev,
          landAppraisal: appraisal.status === "fulfilled" ? appraisal.value : null,
        }));

        if (lat !== undefined && lng !== undefined) {
          const [ridership, population, urbanRisks, scoreResult, hazard] = await Promise.allSettled(
            [
              fetchStationRidership({ lat, lng }),
              fetchPopulationForecast({ lat, lng }),
              fetchUrbanRisks(lat, lng),
              fetchInvestmentScore({ lat, lng }),
              fetchHazardInfo(lat, lng),
            ]
          );
          setAnalysisResult((prev) => ({
            ...prev,
            stationRidership: ridership.status === "fulfilled" ? ridership.value : null,
            populationForecast: population.status === "fulfilled" ? population.value : null,
            externalUrbanRisks: urbanRisks.status === "fulfilled" ? urbanRisks.value : null,
            investmentScore: scoreResult.status === "fulfilled" ? scoreResult.value : null,
            hazardRisks: hazard.status === "fulfilled" ? hazard.value : null,
          }));
        }

        // オフラインでも相場比較は実行可能なため isOnline チェック不要
        void isOnline;
      } catch (e) {
        setError(
          e instanceof Error
            ? e.message
            : "相場データの取得に失敗しました。しばらく後に再試行してください"
        );
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const setPropertyCoords = useCallback((lat: number, lng: number) => {
    setPropertyLat(lat);
    setPropertyLng(lng);
  }, []);

  const clearResult = useCallback(() => {
    setAnalysisResult((prev) => ({ ...prev, result: null }));
  }, []);

  return {
    ...analysisResult,
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
  };
}

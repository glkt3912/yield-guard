"use client";
import { useState, useCallback } from "react";
import type { RefObject } from "react";
import {
  compareLandPrice,
  estimateLandPrice,
  fetchStationRidership,
  fetchPopulationForecast,
  fetchLandAppraisals,
  fetchUrbanRisks,
  fetchHazardInfo,
  fetchInvestmentScore,
} from "@/lib/api";
import type {
  InvestmentInput,
  LandPriceComparison,
  TheoreticalPriceResult,
  StationRidershipResult,
  PopulationForecastResult,
  AppraisalComparisonResult,
  UrbanRisk,
  InvestmentScoreResult,
} from "@/types/investment";

function getCurrentPeriods(): { year: number; quarter: number; toYear: number; toQuarter: number } {
  const now = new Date();
  const toYear = now.getFullYear();
  const toQuarter = Math.ceil((now.getMonth() + 1) / 3);
  return { year: toYear - 2, quarter: 1, toYear, toQuarter };
}

export interface UseExternalDataResult {
  loading: boolean;
  error: string | null;
  comparison: LandPriceComparison | null;
  theoreticalPrice: TheoreticalPriceResult | null;
  landAppraisal: AppraisalComparisonResult | null;
  stationRidership: StationRidershipResult[] | null;
  populationForecast: PopulationForecastResult | null;
  externalUrbanRisks: UrbanRisk[] | null;
  investmentScore: InvestmentScoreResult | null;
  hazardRisks: UrbanRisk[] | null;
  propertyLat: number | undefined;
  propertyLng: number | undefined;
  setPropertyCoords: (lat: number, lng: number) => void;
  handleFetchLandPrices: (area: string, city: string, lat?: number, lng?: number) => Promise<void>;
}

export function useExternalData(
  lastInputRef: RefObject<InvestmentInput | null>
): UseExternalDataResult {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [comparison, setComparison] = useState<LandPriceComparison | null>(null);
  const [theoreticalPrice, setTheoreticalPrice] = useState<TheoreticalPriceResult | null>(null);
  const [landAppraisal, setLandAppraisal] = useState<AppraisalComparisonResult | null>(null);
  const [stationRidership, setStationRidership] = useState<StationRidershipResult[] | null>(null);
  const [populationForecast, setPopulationForecast] = useState<PopulationForecastResult | null>(
    null
  );
  const [externalUrbanRisks, setExternalUrbanRisks] = useState<UrbanRisk[] | null>(null);
  const [investmentScore, setInvestmentScore] = useState<InvestmentScoreResult | null>(null);
  const [hazardRisks, setHazardRisks] = useState<UrbanRisk[] | null>(null);
  const [propertyLat, setPropertyLat] = useState<number | undefined>(undefined);
  const [propertyLng, setPropertyLng] = useState<number | undefined>(undefined);

  const setPropertyCoords = useCallback((lat: number, lng: number) => {
    setPropertyLat(lat);
    setPropertyLng(lng);
  }, []);

  const handleFetchLandPrices = useCallback(
    async (area: string, city: string, lat?: number, lng?: number) => {
      setLoading(true);
      setError(null);
      setStationRidership(null);
      setPopulationForecast(null);
      setLandAppraisal(null);
      setExternalUrbanRisks(null);
      setInvestmentScore(null);
      setHazardRisks(null);

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
        setComparison(comp);

        if (current && current.landArea > 0) {
          try {
            const est = await estimateLandPrice({
              ...baseParams,
              buildingAge: current.buildingAge,
              stationMinutes: current.stationMinutes,
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
          const [ridership, population, urbanRisks, scoreResult, hazard] = await Promise.allSettled(
            [
              fetchStationRidership({ lat, lng }),
              fetchPopulationForecast({ lat, lng }),
              fetchUrbanRisks(lat, lng),
              fetchInvestmentScore({ lat, lng }),
              fetchHazardInfo(lat, lng),
            ]
          );
          setStationRidership(ridership.status === "fulfilled" ? ridership.value : null);
          setPopulationForecast(population.status === "fulfilled" ? population.value : null);
          setExternalUrbanRisks(urbanRisks.status === "fulfilled" ? urbanRisks.value : null);
          setInvestmentScore(scoreResult.status === "fulfilled" ? scoreResult.value : null);
          setHazardRisks(hazard.status === "fulfilled" ? hazard.value : null);
        }
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
    [lastInputRef]
  );

  return {
    loading,
    error,
    comparison,
    theoreticalPrice,
    landAppraisal,
    stationRidership,
    populationForecast,
    externalUrbanRisks,
    investmentScore,
    hazardRisks,
    propertyLat,
    propertyLng,
    setPropertyCoords,
    handleFetchLandPrices,
  };
}

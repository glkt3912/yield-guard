"use client";
import { useState, useCallback } from "react";
import type { RefObject } from "react";
import { simulate } from "@/lib/api";
import type { InvestmentInput, MonteCarloResult } from "@/types/investment";

export interface UseMonteCarloSimResult {
  monteCarloResult: MonteCarloResult | null;
  monteCarloLoading: boolean;
  monteCarloError: string | null;
  handleMonteCarlo: () => Promise<void>;
  setMonteCarloResult: (result: MonteCarloResult | null) => void;
}

export function useMonteCarloSim(
  lastInputRef: RefObject<InvestmentInput | null>
): UseMonteCarloSimResult {
  const [monteCarloResult, setMonteCarloResult] = useState<MonteCarloResult | null>(null);
  const [monteCarloLoading, setMonteCarloLoading] = useState(false);
  const [monteCarloError, setMonteCarloError] = useState<string | null>(null);

  const handleMonteCarlo = useCallback(async () => {
    const current = lastInputRef.current;
    if (!current) return;
    setMonteCarloLoading(true);
    setMonteCarloError(null);
    try {
      const res = await simulate({ base: current, simulations: 1000 });
      setMonteCarloResult(res);
    } catch (e) {
      setMonteCarloError(
        e instanceof Error ? e.message : "モンテカルロシミュレーションに失敗しました"
      );
    } finally {
      setMonteCarloLoading(false);
    }
  }, [lastInputRef]);

  return {
    monteCarloResult,
    monteCarloLoading,
    monteCarloError,
    handleMonteCarlo,
    setMonteCarloResult,
  };
}

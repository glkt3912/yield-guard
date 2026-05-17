"use client";
import { useState, useCallback } from "react";
import type { RefObject, MutableRefObject } from "react";
import { analyze as analyzeOnline } from "@/lib/api";
import { encodeUrlParams } from "@/lib/urlParams";
import type {
  InvestmentInput,
  InvestmentResult,
  LoanMethod,
  SimulationMode,
} from "@/types/investment";

interface AnalyzeApiParams {
  simulationMode: SimulationMode;
  onUrlUpdate: (qs: string) => void;
}

function toErrorMessage(e: unknown, fallback: string): string {
  if (e instanceof TypeError && e.message.toLowerCase().includes("fetch")) {
    return "インターネット接続を確認してください";
  }
  return e instanceof Error ? e.message : fallback;
}

export interface UseAnalyzeApiResult {
  loading: boolean;
  error: string | null;
  loanMethod: LoanMethod;
  setLoanMethod: (method: LoanMethod) => void;
  handleAnalyze: (input: InvestmentInput, quickTotalMan?: string) => Promise<void>;
  handleLoanMethodChange: (method: LoanMethod) => Promise<void>;
  setResult: (result: InvestmentResult | null) => void;
  setLastInput: (input: InvestmentInput | null) => void;
  clearMonteCarloResult: () => void;
}

export function useAnalyzeApi(
  lastInputRef: RefObject<InvestmentInput | null>,
  loanMethodRef: MutableRefObject<LoanMethod>,
  paramsRef: RefObject<AnalyzeApiParams>,
  onResult: (result: InvestmentResult | null) => void,
  onLastInput: (input: InvestmentInput | null) => void,
  onClearMonteCarloResult: () => void
): UseAnalyzeApiResult {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loanMethod, setLoanMethodState] = useState<LoanMethod>("equal-payment");

  const setLoanMethod = useCallback((method: LoanMethod) => {
    setLoanMethodState(method);
    loanMethodRef.current = method;
  }, [loanMethodRef]);

  const handleAnalyze = useCallback(async (input: InvestmentInput, quickTotalMan?: string) => {
    const { simulationMode, onUrlUpdate } = paramsRef.current!;
    const currentLoanMethod = loanMethodRef.current;
    setLoading(true);
    setError(null);
    onClearMonteCarloResult();
    const inputWithMethod = { ...input, loanMethod: currentLoanMethod };
    try {
      const res = await analyzeOnline(inputWithMethod);
      onResult(res);
      onLastInput(inputWithMethod);
      const urlParams = encodeUrlParams(simulationMode, inputWithMethod, quickTotalMan);
      const qs = urlParams.toString();
      onUrlUpdate(qs);
    } catch (e) {
      setError(toErrorMessage(e, "シミュレーションに失敗しました"));
    } finally {
      setLoading(false);
    }
  }, [loanMethodRef, paramsRef, onResult, onLastInput, onClearMonteCarloResult]);

  const handleLoanMethodChange = useCallback(async (method: LoanMethod) => {
    setLoanMethodState(method);
    loanMethodRef.current = method;
    const current = lastInputRef.current;
    if (!current) return;
    setLoading(true);
    setError(null);
    onClearMonteCarloResult();
    try {
      const updatedInput = { ...current, loanMethod: method };
      const res = await analyzeOnline(updatedInput);
      onResult(res);
      onLastInput({ ...current, loanMethod: method });
    } catch (e) {
      setError(toErrorMessage(e, "シミュレーションに失敗しました"));
    } finally {
      setLoading(false);
    }
  }, [lastInputRef, loanMethodRef, onResult, onLastInput, onClearMonteCarloResult]);

  const setResult = useCallback((result: InvestmentResult | null) => {
    onResult(result);
  }, [onResult]);

  const setLastInput = useCallback((input: InvestmentInput | null) => {
    onLastInput(input);
  }, [onLastInput]);

  const clearMonteCarloResult = useCallback(() => {
    onClearMonteCarloResult();
  }, [onClearMonteCarloResult]);

  return {
    loading,
    error,
    loanMethod,
    setLoanMethod,
    handleAnalyze,
    handleLoanMethodChange,
    setResult,
    setLastInput,
    clearMonteCarloResult,
  };
}

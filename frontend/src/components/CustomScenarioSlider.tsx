"use client";
import React, { useState, useEffect, useRef, useCallback, useLayoutEffect } from "react";
import { Loader2 } from "lucide-react";
import { Slider } from "@/components/ui/slider";
import { analyze } from "@/lib/api";
import type { InvestmentInput, StressScenarioResult } from "@/types/investment";

const CUSTOM_SCENARIO_LABEL = "カスタム" as const;

export interface CustomScenarioState {
  customScenario: StressScenarioResult | null;
  customLoading: boolean;
  customLoanRateDelta: number;
  customVacancyRateDelta: number;
}

interface CustomScenarioSliderProps {
  input: InvestmentInput;
  onStateChange: (state: CustomScenarioState) => void;
}

export function CustomScenarioSlider({ input, onStateChange }: CustomScenarioSliderProps) {
  const [customLoanRateDelta, setCustomLoanRateDelta] = useState(0);
  const [customVacancyRateDelta, setCustomVacancyRateDelta] = useState(0);
  const [customScenario, setCustomScenario] = useState<StressScenarioResult | null>(null);
  const [customLoading, setCustomLoading] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onStateChangeRef = useRef(onStateChange);

  useLayoutEffect(() => {
    onStateChangeRef.current = onStateChange;
  });

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const fetchCustomScenario = useCallback(
    (loanDeltaPct: number, vacancyDeltaPct: number) => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      const loanDelta = loanDeltaPct / 100;
      const vacancyDelta = vacancyDeltaPct / 100;
      if (loanDelta === 0 && vacancyDelta === 0) {
        setCustomScenario(null);
        return;
      }
      setCustomLoading(true);
      debounceRef.current = setTimeout(async () => {
        try {
          const res = await analyze({
            ...input,
            loanRateDelta: loanDelta,
            vacancyRateDelta: vacancyDelta,
          });
          const custom = res.stressScenarios.find((s) => s.label === CUSTOM_SCENARIO_LABEL) ?? null;
          setCustomScenario(custom);
        } catch (err) {
          console.error("[CustomStressTest] fetch failed:", err);
          setCustomScenario(null);
        } finally {
          setCustomLoading(false);
        }
      }, 400);
    },
    [input]
  );

  useEffect(() => {
    fetchCustomScenario(customLoanRateDelta, customVacancyRateDelta);
  }, [customLoanRateDelta, customVacancyRateDelta, fetchCustomScenario]);

  useEffect(() => {
    onStateChangeRef.current({ customScenario, customLoading, customLoanRateDelta, customVacancyRateDelta });
  }, [customScenario, customLoading, customLoanRateDelta, customVacancyRateDelta]);

  return (
    <div className="mb-4 rounded-lg border border-border bg-muted/30 p-4 space-y-4">
      <p className="text-sm font-medium">カスタムシナリオ設定</p>
      <Slider
        label="金利オフセット"
        value={customLoanRateDelta}
        min={0}
        max={3}
        step={0.1}
        onChange={(v) => setCustomLoanRateDelta(v)}
        formatValue={(v) => (v === 0 ? "±0" : `+${v.toFixed(1)}%`)}
      />
      <Slider
        label="空室率オフセット"
        value={customVacancyRateDelta}
        min={0}
        max={30}
        step={1}
        onChange={(v) => setCustomVacancyRateDelta(v)}
        formatValue={(v) => (v === 0 ? "±0" : `+${v.toFixed(0)}%`)}
      />
      {(customLoanRateDelta > 0 || customVacancyRateDelta > 0) && customLoading && (
        <p className="flex items-center gap-1 text-xs text-muted-foreground">
          <Loader2 className="h-3 w-3 animate-spin" />
          計算中…
        </p>
      )}
    </div>
  );
}

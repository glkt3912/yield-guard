"use client";
import React, { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { SimulationModeToggle } from "@/components/SimulationModeToggle";
import type { InvestmentInput, SimulationMode, RateAdjustment } from "@/types/investment";
import { DEFAULT_INPUT, QUICK_MODE_DEFAULTS } from "@/types/investment";
import { fetchGeocode } from "@/lib/api";
import { type ZoningType } from "@/lib/zoning";
import { useMunicipalitiesLoader } from "@/hooks/useMunicipalitiesLoader";
import { useRentDeclineHint } from "@/hooks/useRentDeclineHint";
import { QuickModeForm, type QuickHistoryEntry } from "@/components/QuickModeForm";
import { FullModeForm } from "@/components/FullModeForm";
import { type GeocodeState } from "@/components/sections/LocationSection";

const QUICK_HISTORY_KEY = "yield-guard:quick-history";
const QUICK_HISTORY_MAX = 5;

function loadQuickHistory(): QuickHistoryEntry[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(QUICK_HISTORY_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed as QuickHistoryEntry[];
  } catch {
    return [];
  }
}

function saveQuickHistory(entry: QuickHistoryEntry): QuickHistoryEntry[] {
  try {
    const history = loadQuickHistory();
    const next = [entry, ...history].slice(0, QUICK_HISTORY_MAX);
    localStorage.setItem(QUICK_HISTORY_KEY, JSON.stringify(next));
    return next;
  } catch {
    return loadQuickHistory();
  }
}

type FormErrors = Partial<Record<keyof InvestmentInput | "quickTotalPrice", string>>;

function validateFull(input: InvestmentInput): FormErrors {
  const e: FormErrors = {};
  if (input.landPrice <= 0 || input.landPrice > 10_000_000_000)
    e.landPrice = "1〜100億円の範囲で入力してください";
  if (input.buildingCost <= 0 || input.buildingCost > 10_000_000_000)
    e.buildingCost = "1〜100億円の範囲で入力してください";
  if (input.monthlyRent <= 0) e.monthlyRent = "正の値を入力してください";
  if (input.vacancyRate < 0 || input.vacancyRate >= 1.0)
    e.vacancyRate = "0〜99%の範囲で入力してください";
  if (input.loanAmount < 0) e.loanAmount = "0以上の値を入力してください";
  if (input.annualLoanRate < 0 || input.annualLoanRate > 0.3)
    e.annualLoanRate = "0〜30%の範囲で入力してください";
  if (input.loanYears < 0 || input.loanYears > 50) e.loanYears = "0〜50年の範囲で入力してください";
  if (input.miscExpenseRate < 0 || input.miscExpenseRate > 0.5)
    e.miscExpenseRate = "0〜50%の範囲で入力してください";
  if (input.expenseRate < 0 || input.expenseRate > 0.9)
    e.expenseRate = "0〜90%の範囲で入力してください";
  if (input.incomeTaxRate < 0 || input.incomeTaxRate > 0.6)
    e.incomeTaxRate = "0〜60%の範囲で入力してください";
  if (input.exitYieldTarget <= 0 || input.exitYieldTarget > 0.5)
    e.exitYieldTarget = "0%超〜50%の範囲で入力してください";
  if (input.yieldTarget <= 0 || input.yieldTarget > 0.5)
    e.yieldTarget = "0%超〜50%の範囲で入力してください";
  if (input.holdingYears < 0 || input.holdingYears > 50)
    e.holdingYears = "0〜50年の範囲で入力してください";
  if ((input.discountRate ?? 0.05) < 0 || (input.discountRate ?? 0.05) > 0.3)
    e.discountRate = "0〜30%の範囲で入力してください";
  if ((input.priceDeclineRate ?? 0) < 0 || (input.priceDeclineRate ?? 0) > 0.1)
    e.priceDeclineRate = "0〜10%の範囲で入力してください";
  return e;
}

function validateQuick(quickTotalPriceMan: string, input: InvestmentInput): FormErrors {
  const e: FormErrors = {};
  const total = parseFloat(quickTotalPriceMan) || 0;
  if (total <= 0 || total * 10_000 > 10_000_000_000)
    e.quickTotalPrice = "1〜100億円の範囲で入力してください";
  if (input.monthlyRent <= 0) e.monthlyRent = "正の値を入力してください";
  if (input.loanAmount < 0) e.loanAmount = "0以上の値を入力してください";
  return e;
}

interface Props {
  onAnalyze: (input: InvestmentInput, quickTotalMan?: string) => Promise<void>;
  onFetchLandPrices: (area: string, city: string, lat?: number, lng?: number) => Promise<void>;
  loading: boolean;
  simulationMode: SimulationMode;
  onModeChange: (mode: SimulationMode) => void;
  initialInput?: Partial<InvestmentInput>;
  initialQuickTotalPriceMan?: string;
  isOnline?: boolean | null;
  externalLat?: number;
  externalLng?: number;
  showModeToggle?: boolean;
}

export function InvestmentForm({
  onAnalyze,
  onFetchLandPrices,
  loading,
  simulationMode,
  onModeChange,
  initialInput,
  initialQuickTotalPriceMan,
  isOnline = null,
  externalLat,
  externalLng,
  showModeToggle = true,
}: Props) {
  const [input, setInput] = useState<InvestmentInput>({ ...DEFAULT_INPUT, ...initialInput });
  const [area, setArea] = useState("10");
  const [propertyLat, setPropertyLat] = useState("");
  const [propertyLng, setPropertyLng] = useState("");
  const [addressInput, setAddressInput] = useState("");
  const [geocodeStatus, setGeocodeStatus] = useState<"idle" | "loading" | "success" | "error">(
    "idle"
  );
  const [geocodeError, setGeocodeError] = useState("");
  const [geocodeLocationType, setGeocodeLocationType] = useState("");
  const [showManualCoords, setShowManualCoords] = useState(false);
  const [touched, setTouched] = useState<Set<string>>(new Set());
  const [zoningType, setZoningType] = useState<ZoningType>("");
  const [quickTotalPriceMan, setQuickTotalPriceMan] = useState(initialQuickTotalPriceMan ?? "");
  const [showLandSection, setShowLandSection] = useState(false);
  const [isCashPurchase, setIsCashPurchase] = useState(false);
  const [savedLoanAmount, setSavedLoanAmount] = useState(DEFAULT_INPUT.loanAmount);
  const [savedLoanYears, setSavedLoanYears] = useState(DEFAULT_INPUT.loanYears);
  const [showCustomLoan, setShowCustomLoan] = useState(false);
  const [quickHistory, setQuickHistory] = useState<QuickHistoryEntry[]>([]);
  const [rateScheduleEnabled, setRateScheduleEnabled] = useState(false);

  const isQuick = simulationMode === "quick";

  // Municipalities hook
  const {
    municipalities,
    loading: muniLoading,
    error: muniError,
    city,
    setCity,
    muniFilter,
    setMuniFilter,
    filteredMunicipalities,
    loadMunicipalities,
  } = useMunicipalitiesLoader("10");

  // Rent decline hint hook
  const setNumRef = useRef<(key: keyof InvestmentInput, value: number) => void>(null!);
  const {
    rentHint,
    loading: rentHintLoading,
    error: rentHintError,
    fetch: fetchRentHint,
    clear: clearRentHint,
  } = useRentDeclineHint((rate) => setNumRef.current("rentDeclineRate", rate));

  const currentErrors = useMemo(
    () => (isQuick ? validateQuick(quickTotalPriceMan, input) : validateFull(input)),
    [isQuick, quickTotalPriceMan, input]
  );
  const scheduleHasErrors = useMemo(() => {
    if (!rateScheduleEnabled) return false;
    return input.rateAdjustmentSchedule.some((step, i) => {
      const maxYear = input.loanYears || 35;
      const prevYear = i > 0 ? input.rateAdjustmentSchedule[i - 1].afterYear : 1;
      return (
        step.afterYear < 2 ||
        step.afterYear > maxYear ||
        step.afterYear <= prevYear ||
        step.rate <= 0 ||
        step.rate > 0.3
      );
    });
  }, [rateScheduleEnabled, input.rateAdjustmentSchedule, input.loanYears]);

  const hasErrors = Object.keys(currentErrors).length > 0 || scheduleHasErrors;

  const canAddRateStep = useMemo(() => {
    const schedule = input.rateAdjustmentSchedule;
    if (schedule.length >= 3) return false;
    const maxYear = input.loanYears || 35;
    const lastYear = schedule.length > 0 ? schedule[schedule.length - 1].afterYear : 1;
    return lastYear < maxYear;
  }, [input.rateAdjustmentSchedule, input.loanYears]);

  const fieldError = (key: string) =>
    touched.has(key) ? currentErrors[key as keyof FormErrors] : undefined;

  const markTouched = (key: string) =>
    setTouched((prev) => (prev.has(key) ? prev : new Set([...prev, key])));

  const setNum = useCallback((key: keyof InvestmentInput, value: number) => {
    setInput((prev) => ({ ...prev, [key]: value }));
    markTouched(key);
  }, []);

  // Keep ref in sync for useRentDeclineHint callback
  setNumRef.current = setNum;

  const setStr = (key: keyof InvestmentInput, value: string) =>
    setInput((prev) => ({ ...prev, [key]: value }));

  useEffect(() => {
    if (externalLat !== undefined && externalLng !== undefined) {
      setPropertyLat(externalLat.toFixed(6));
      setPropertyLng(externalLng.toFixed(6));
    }
  }, [externalLat, externalLng]);

  // モード切替時に現金購入フラグをリセットし、ローン値を復元する
  useEffect(() => {
    if (isCashPurchase) {
      setIsCashPurchase(false);
      setInput((prev) => ({ ...prev, loanAmount: savedLoanAmount, loanYears: savedLoanYears }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isQuick]);

  // cityが変わるたびにレント参考値をクリア（手動変更・自動選択どちらも）
  useEffect(() => {
    clearRentHint();
  }, [city, clearRentHint]);

  useEffect(() => {
    setQuickHistory(loadQuickHistory());
  }, []);

  const handleAreaChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const code = e.target.value;
    setArea(code);
    loadMunicipalities(code);
  };

  const handleCityChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setCity(e.target.value);
  };

  const handleGeocode = useCallback(async () => {
    if (!addressInput.trim()) return;
    setGeocodeStatus("loading");
    setGeocodeError("");
    try {
      const result = await fetchGeocode(addressInput.trim());
      setPropertyLat(result.lat.toFixed(6));
      setPropertyLng(result.lng.toFixed(6));
      setGeocodeLocationType(result.locationType);
      setGeocodeStatus("success");
      setShowManualCoords(false);
    } catch (e) {
      setGeocodeError(e instanceof Error ? e.message : "座標取得に失敗しました");
      setGeocodeStatus("error");
      setShowManualCoords(true);
    }
  }, [addressInput]);

  const handleFetchRentHint = useCallback(async () => {
    await fetchRentHint(area, city);
  }, [fetchRentHint, area, city]);

  const handleCashPurchaseToggle = (checked: boolean) => {
    setIsCashPurchase(checked);
    if (checked) {
      setSavedLoanAmount(input.loanAmount);
      setSavedLoanYears(input.loanYears);
      setInput((prev) => ({ ...prev, loanAmount: 0, loanYears: 0 }));
      setShowCustomLoan(false);
    } else {
      setInput((prev) => ({ ...prev, loanAmount: savedLoanAmount, loanYears: savedLoanYears }));
    }
  };

  const addRateStep = useCallback(() => {
    setInput((prev) => {
      const schedule = prev.rateAdjustmentSchedule;
      const maxYear = prev.loanYears || 35;
      const lastYear = schedule.length > 0 ? schedule[schedule.length - 1].afterYear : 1;
      if (lastYear >= maxYear) return prev;
      const nextYear = Math.min(lastYear + 5, maxYear);
      const newStep: RateAdjustment = { afterYear: nextYear, rate: prev.annualLoanRate + 0.005 };
      return { ...prev, rateAdjustmentSchedule: [...prev.rateAdjustmentSchedule, newStep] };
    });
  }, []);

  const removeRateStep = useCallback((index: number) => {
    setInput((prev) => ({
      ...prev,
      rateAdjustmentSchedule: prev.rateAdjustmentSchedule.filter((_, i) => i !== index),
    }));
  }, []);

  const updateRateStep = useCallback(
    (index: number, field: keyof RateAdjustment, value: number) => {
      setInput((prev) => ({
        ...prev,
        rateAdjustmentSchedule: prev.rateAdjustmentSchedule.map((s, i) =>
          i === index ? { ...s, [field]: value } : s
        ),
      }));
    },
    []
  );

  const addCapexEvent = useCallback(() => {
    setInput((prev) => {
      const schedule = prev.capexSchedule ?? [];
      if (schedule.length >= 5) return prev;
      const lastYear = schedule.length > 0 ? schedule[schedule.length - 1].year : 0;
      const nextYear = Math.min(lastYear + 5, prev.holdingYears || 20);
      return { ...prev, capexSchedule: [...schedule, { year: nextYear, amount: 1_000_000 }] };
    });
  }, []);

  const removeCapexEvent = useCallback((index: number) => {
    setInput((prev) => ({
      ...prev,
      capexSchedule: (prev.capexSchedule ?? []).filter((_, i) => i !== index),
    }));
  }, []);

  const updateCapexEvent = useCallback((index: number, field: "year" | "amount", value: number) => {
    setInput((prev) => ({
      ...prev,
      capexSchedule: (prev.capexSchedule ?? []).map((ev, i) =>
        i === index ? { ...ev, [field]: value } : ev
      ),
    }));
  }, []);

  const handleRateScheduleToggle = useCallback((enabled: boolean) => {
    setRateScheduleEnabled(enabled);
    if (!enabled) {
      setInput((prev) => ({ ...prev, rateAdjustmentSchedule: [] }));
    }
  }, []);

  const sortedInput = (src: InvestmentInput): InvestmentInput => ({
    ...src,
    rateAdjustmentSchedule: [...src.rateAdjustmentSchedule].sort(
      (a, b) => a.afterYear - b.afterYear
    ),
  });

  const handleAnalyze = () => {
    if (hasErrors) return;
    if (isQuick) {
      const total = (parseFloat(quickTotalPriceMan) || 0) * 10_000;
      const autoLoanAmount = !isCashPurchase && !showCustomLoan ? total * 0.8 : input.loanAmount;
      const payload: InvestmentInput = sortedInput({
        ...input,
        ...QUICK_MODE_DEFAULTS,
        landPrice: total * 0.7,
        buildingCost: total * 0.3,
        loanAmount: isCashPurchase ? 0 : autoLoanAmount,
      });
      const entry: QuickHistoryEntry = {
        totalPriceMan: quickTotalPriceMan,
        monthlyRentYen: String(input.monthlyRent),
        ts: Date.now(),
      };
      setQuickHistory(saveQuickHistory(entry));
      onAnalyze(payload, quickTotalPriceMan);
    } else {
      onAnalyze(sortedInput(input));
    }
  };

  const handleRestoreLastInput = () => {
    if (quickHistory.length === 0) return;
    const last = quickHistory[0];
    setQuickTotalPriceMan(last.totalPriceMan);
    setNum("monthlyRent", parseFloat(last.monthlyRentYen) || 0);
  };

  // geocode オブジェクト（FullModeForm 向け）
  const geocode = useMemo<GeocodeState>(
    () => ({
      address: addressInput,
      status: geocodeStatus,
      error: geocodeError,
      locationType: geocodeLocationType,
      lat: propertyLat,
      lng: propertyLng,
    }),
    [addressInput, geocodeStatus, geocodeError, geocodeLocationType, propertyLat, propertyLng]
  );
  const onGeocodeChange = useCallback((patch: Partial<GeocodeState>) => {
    if (patch.address !== undefined) setAddressInput(patch.address);
    if (patch.status !== undefined) setGeocodeStatus(patch.status);
    if (patch.error !== undefined) setGeocodeError(patch.error);
    if (patch.locationType !== undefined) setGeocodeLocationType(patch.locationType);
    if (patch.lat !== undefined) setPropertyLat(patch.lat);
    if (patch.lng !== undefined) setPropertyLng(patch.lng);
  }, []);

  // rateSchedule / capex ハンドラオブジェクト（FullModeForm 向け）
  const rateSchedule = useMemo(
    () => ({
      enabled: rateScheduleEnabled,
      onToggle: handleRateScheduleToggle,
      onAdd: addRateStep,
      onRemove: removeRateStep,
      onUpdate: updateRateStep,
      canAdd: canAddRateStep,
    }),
    [
      rateScheduleEnabled,
      handleRateScheduleToggle,
      addRateStep,
      removeRateStep,
      updateRateStep,
      canAddRateStep,
    ]
  );
  const capex = useMemo(
    () => ({
      onAdd: addCapexEvent,
      onRemove: removeCapexEvent,
      onUpdate: updateCapexEvent,
    }),
    [addCapexEvent, removeCapexEvent, updateCapexEvent]
  );

  const sharedMuniProps = {
    area,
    handleAreaChange,
    city,
    handleCityChange,
    muniFilter,
    setMuniFilter,
    filteredMunicipalities,
    muniLoading,
    muniError,
    isOnline,
    propertyLat,
    propertyLng,
    loading,
    onFetchLandPrices,
  };

  return (
    <div className="space-y-4">
      {showModeToggle && <SimulationModeToggle mode={simulationMode} onChange={onModeChange} />}

      {isQuick ? (
        <QuickModeForm
          {...sharedMuniProps}
          showLandSection={showLandSection}
          setShowLandSection={setShowLandSection}
          quickHistory={quickHistory}
          handleRestoreLastInput={handleRestoreLastInput}
          quickTotalPriceMan={quickTotalPriceMan}
          setQuickTotalPriceMan={setQuickTotalPriceMan}
          markTouched={markTouched}
          fieldError={fieldError}
          input={input}
          setNum={setNum}
          isCashPurchase={isCashPurchase}
          handleCashPurchaseToggle={handleCashPurchaseToggle}
          showCustomLoan={showCustomLoan}
          setShowCustomLoan={setShowCustomLoan}
          hasErrors={hasErrors}
          handleAnalyze={handleAnalyze}
        />
      ) : (
        <FullModeForm
          area={area}
          handleAreaChange={handleAreaChange}
          city={city}
          handleCityChange={handleCityChange}
          muniFilter={muniFilter}
          setMuniFilter={setMuniFilter}
          filteredMunicipalities={filteredMunicipalities}
          muniLoading={muniLoading}
          muniError={muniError}
          isOnline={isOnline}
          geocode={geocode}
          onGeocodeChange={onGeocodeChange}
          showManualCoords={showManualCoords}
          handleGeocode={handleGeocode}
          loading={loading}
          onFetchLandPrices={onFetchLandPrices}
          input={input}
          setNum={setNum}
          setStr={setStr}
          fieldError={fieldError}
          rentHint={rentHint}
          rentHintLoading={rentHintLoading}
          rentHintError={rentHintError}
          handleFetchRentHint={handleFetchRentHint}
          isCashPurchase={isCashPurchase}
          handleCashPurchaseToggle={handleCashPurchaseToggle}
          zoningType={zoningType}
          setZoningType={setZoningType}
          rateSchedule={rateSchedule}
          capex={capex}
          hasErrors={hasErrors}
          handleAnalyze={handleAnalyze}
        />
      )}
    </div>
  );
}

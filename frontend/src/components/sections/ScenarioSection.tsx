"use client";
import React from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Plus, Trash2 } from "lucide-react";
import type { InvestmentInput, RateAdjustment } from "@/types/investment";
import { formatPct } from "@/lib/utils";

interface ScenarioSectionProps {
  input: InvestmentInput;
  setNum: (key: keyof InvestmentInput, value: number) => void;
  rateScheduleEnabled: boolean;
  handleRateScheduleToggle: (enabled: boolean) => void;
  addRateStep: () => void;
  removeRateStep: (i: number) => void;
  updateRateStep: (i: number, field: keyof RateAdjustment, value: number) => void;
  canAddRateStep: boolean;
  addCapexEvent: () => void;
  removeCapexEvent: (i: number) => void;
  updateCapexEvent: (i: number, field: "year" | "amount", value: number) => void;
}

export function ScenarioSection({
  input,
  setNum,
  rateScheduleEnabled,
  handleRateScheduleToggle,
  addRateStep,
  removeRateStep,
  updateRateStep,
  canAddRateStep,
  addCapexEvent,
  removeCapexEvent,
  updateCapexEvent,
}: ScenarioSectionProps) {
  return (
    <>
      {/* 変動金利シミュレーション */}
      <div className="border-t pt-4 space-y-3">
        <div className="flex items-center justify-between">
          <p className="text-sm font-semibold text-foreground">変動金利シミュレーション</p>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={rateScheduleEnabled}
              onChange={(e) => handleRateScheduleToggle(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300"
            />
            <span className="text-xs text-muted-foreground">有効にする</span>
          </label>
        </div>
        {rateScheduleEnabled && (
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground">
              指定年以降の適用金利（絶対値）を設定します。最大3ステップ。
            </p>
            {input.rateAdjustmentSchedule.map((step, i) => {
              const maxYear = input.loanYears || 35;
              const prevYear = i > 0 ? input.rateAdjustmentSchedule[i - 1].afterYear : 1;
              const yearError =
                step.afterYear < 2
                  ? "2年目以降を指定してください"
                  : step.afterYear > maxYear
                    ? `${maxYear}年以内を指定してください`
                    : step.afterYear <= prevYear
                      ? `前のステップより後の年を指定してください`
                      : undefined;
              const rateError =
                step.rate <= 0 || step.rate > 0.3 ? "0%超〜30%の範囲で入力してください" : undefined;
              return (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    label={i === 0 ? "開始年" : ""}
                    type="number"
                    inputMode="numeric"
                    suffix="年目〜"
                    min="2"
                    max={String(maxYear)}
                    step="1"
                    value={String(step.afterYear)}
                    onChange={(e) => updateRateStep(i, "afterYear", parseInt(e.target.value) || 2)}
                    error={yearError}
                  />
                  <Input
                    label={i === 0 ? "適用金利" : ""}
                    type="number"
                    inputMode="decimal"
                    suffix="%"
                    min="0.1"
                    max="30"
                    step="0.1"
                    value={(step.rate * 100).toFixed(2)}
                    onChange={(e) =>
                      updateRateStep(i, "rate", (parseFloat(e.target.value) || 0) / 100)
                    }
                    error={rateError}
                  />
                  <button
                    type="button"
                    onClick={() => removeRateStep(i)}
                    className={`text-muted-foreground hover:text-destructive flex-shrink-0 ${i === 0 ? "mt-5" : ""}`}
                    aria-label="削除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              );
            })}
            {canAddRateStep && (
              <button
                type="button"
                onClick={addRateStep}
                className="flex items-center gap-1 text-xs text-primary hover:underline"
              >
                <Plus className="h-3 w-3" />
                金利ステップを追加
              </button>
            )}
          </div>
        )}
      </div>

      {/* ストレステスト */}
      <div className="border-t pt-4 space-y-4">
        <p className="text-sm font-semibold text-foreground">ストレステスト</p>
        <Slider
          label="空室率の上昇"
          value={input.vacancyRateDelta}
          min={0}
          max={0.3}
          step={0.01}
          onChange={(v) => setNum("vacancyRateDelta", v)}
          formatValue={(v) => `+${formatPct(v)}`}
        />
        <Slider
          label="金利の上昇"
          value={input.loanRateDelta}
          min={0}
          max={0.03}
          step={0.001}
          onChange={(v) => setNum("loanRateDelta", v)}
          formatValue={(v) => `+${formatPct(v)}`}
        />
        {rateScheduleEnabled && input.loanRateDelta > 0 && (
          <p className="text-xs text-muted-foreground">
            金利上昇分はスケジュール後の金利にも上乗せされます
          </p>
        )}
      </div>

      {/* 大規模修繕費スケジュール */}
      <div className="border-t pt-4">
        <div className="flex items-center justify-between mb-3">
          <p className="text-sm font-semibold text-foreground">大規模修繕費スケジュール</p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="text-xs h-7 px-2"
            disabled={(input.capexSchedule ?? []).length >= 5}
            onClick={addCapexEvent}
          >
            ＋追加
          </Button>
        </div>
        {(input.capexSchedule ?? []).length === 0 ? (
          <p className="text-xs text-muted-foreground">修繕費なし（追加ボタンで最大5件入力可）</p>
        ) : (
          <div className="space-y-2">
            {(input.capexSchedule ?? []).map((ev, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input
                  label="何年目"
                  type="number"
                  suffix="年目"
                  step="1"
                  min="1"
                  max={input.holdingYears || 35}
                  value={String(ev.year)}
                  onChange={(e) => updateCapexEvent(i, "year", parseInt(e.target.value) || 1)}
                  className="w-24"
                />
                <Input
                  label="金額"
                  type="number"
                  suffix="万円"
                  step="10"
                  min="0"
                  value={String(Math.round(ev.amount / 10_000))}
                  onChange={(e) =>
                    updateCapexEvent(i, "amount", (parseFloat(e.target.value) || 0) * 10_000)
                  }
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="text-xs h-7 px-2 mt-5 shrink-0"
                  onClick={() => removeCapexEvent(i)}
                >
                  削除
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}

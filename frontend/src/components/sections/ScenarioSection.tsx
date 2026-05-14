"use client";
import React from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Plus, Trash2 } from "lucide-react";
import type { InvestmentInput, RateAdjustment } from "@/types/investment";
import { formatPct } from "@/lib/utils";

const ALL_EXIT_YEARS = [5, 10, 15, 20] as const;

export interface RateScheduleHandlers {
  enabled: boolean;
  onToggle: (v: boolean) => void;
  onAdd: () => void;
  onRemove: (i: number) => void;
  onUpdate: (i: number, field: keyof RateAdjustment, value: number) => void;
  canAdd: boolean;
}

export interface CapexHandlers {
  onAdd: () => void;
  onRemove: (i: number) => void;
  onUpdate: (i: number, field: "year" | "amount", value: number) => void;
}

interface ScenarioSectionProps {
  input: InvestmentInput;
  setNum: (key: keyof InvestmentInput, value: number) => void;
  setField: <K extends keyof InvestmentInput>(key: K, value: InvestmentInput[K]) => void;
  rateSchedule: RateScheduleHandlers;
  capex: CapexHandlers;
}

export function ScenarioSection({
  input,
  setNum,
  setField,
  rateSchedule,
  capex,
}: ScenarioSectionProps) {
  const exitYears: number[] = input.exitYears ?? [...ALL_EXIT_YEARS];

  function toggleExitYear(yr: number) {
    const next = exitYears.includes(yr)
      ? exitYears.filter((y) => y !== yr)
      : [...exitYears, yr].sort((a, b) => a - b);
    setField("exitYears", next.length > 0 ? next : [...ALL_EXIT_YEARS]);
  }
  return (
    <>
      {/* 変動金利シミュレーション */}
      <div className="border-t pt-4 space-y-3">
        <div className="flex items-center justify-between">
          <p className="text-sm font-semibold text-foreground">変動金利シミュレーション</p>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={rateSchedule.enabled}
              onChange={(e) => rateSchedule.onToggle(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300"
            />
            <span className="text-xs text-muted-foreground">有効にする</span>
          </label>
        </div>
        {rateSchedule.enabled && (
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
                    onChange={(e) =>
                      rateSchedule.onUpdate(i, "afterYear", parseInt(e.target.value) || 2)
                    }
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
                      rateSchedule.onUpdate(i, "rate", (parseFloat(e.target.value) || 0) / 100)
                    }
                    error={rateError}
                  />
                  <button
                    type="button"
                    onClick={() => rateSchedule.onRemove(i)}
                    className={`text-muted-foreground hover:text-destructive flex-shrink-0 ${i === 0 ? "mt-5" : ""}`}
                    aria-label="削除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              );
            })}
            {rateSchedule.canAdd && (
              <button
                type="button"
                onClick={rateSchedule.onAdd}
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
        {rateSchedule.enabled && input.loanRateDelta > 0 && (
          <p className="text-xs text-muted-foreground">
            金利上昇分はスケジュール後の金利にも上乗せされます
          </p>
        )}
      </div>

      {/* 複数保有年数 出口比較 */}
      <div className="border-t pt-4 space-y-2">
        <p className="text-sm font-semibold text-foreground">出口比較（保有年数）</p>
        <p className="text-xs text-muted-foreground">比較する保有年数を選択してください</p>
        <div className="flex gap-4 flex-wrap">
          {ALL_EXIT_YEARS.map((yr) => (
            <label key={yr} className="flex items-center gap-1.5 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={exitYears.includes(yr)}
                onChange={() => toggleExitYear(yr)}
                className="h-4 w-4 rounded border-gray-300"
              />
              <span className="text-sm">{yr}年</span>
            </label>
          ))}
        </div>
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
            onClick={capex.onAdd}
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
                  onChange={(e) => capex.onUpdate(i, "year", parseInt(e.target.value) || 1)}
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
                    capex.onUpdate(i, "amount", (parseFloat(e.target.value) || 0) * 10_000)
                  }
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="text-xs h-7 px-2 mt-5 shrink-0"
                  onClick={() => capex.onRemove(i)}
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

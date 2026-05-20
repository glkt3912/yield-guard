"use client";
import { Zap, SlidersHorizontal } from "lucide-react";
import type { SimulationMode } from "@/types/investment";

interface Props {
  mode: SimulationMode;
  onChange: (mode: SimulationMode) => void;
  className?: string;
}

export function SimulationModeToggle({ mode, onChange, className }: Props) {
  return (
    <div
      className={`flex rounded-lg border bg-muted p-1 gap-1${className ? ` ${className}` : ""}`}
      role="group"
      aria-label="シミュレーションモード"
    >
      <button
        type="button"
        role="radio"
        aria-checked={mode === "quick"}
        onClick={() => onChange("quick")}
        className={`flex flex-1 flex-col items-center justify-center gap-0.5 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
          mode === "quick"
            ? "bg-primary text-white shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
      >
        <span className="flex items-center gap-1.5">
          <Zap className="h-3.5 w-3.5" />
          かんたん判定
        </span>
        <span className="text-xs font-normal opacity-75">2項目で即判定</span>
      </button>
      <button
        type="button"
        role="radio"
        aria-checked={mode === "full"}
        onClick={() => onChange("full")}
        className={`flex flex-1 flex-col items-center justify-center gap-0.5 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
          mode === "full"
            ? "bg-primary text-white shadow-sm"
            : "text-muted-foreground hover:text-foreground"
        }`}
      >
        <span className="flex items-center gap-1.5">
          <SlidersHorizontal className="h-3.5 w-3.5" />
          くわしく分析
        </span>
        <span className="text-xs font-normal opacity-75">17項目で精密分析</span>
      </button>
    </div>
  );
}

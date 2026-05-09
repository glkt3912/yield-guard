"use client";
import React from "react";
import { useRentValidation, type RentDeviationLevel } from "@/hooks/useRentValidation";

interface RentValidationHintProps {
  monthlyRent: number;
  area: string;
  city: string;
}

function badge(
  level: RentDeviationLevel,
  deviationPct: number
): { className: string; message: string } | null {
  if (level === null) return null;
  const sign = deviationPct >= 0 ? "+" : "";
  const pctStr = `${sign}${deviationPct.toFixed(1)}%`;

  switch (level) {
    case "normal":
      return {
        className: "text-muted-foreground",
        message: `相場比 ${pctStr}（適正範囲）`,
      };
    case "high-warn":
      return {
        className: "text-yellow-700",
        message: `相場比 ${pctStr} — 相場より高め。空室リスクを考慮してください`,
      };
    case "high-danger":
      return {
        className: "text-destructive",
        message: `相場比 ${pctStr} — 相場から大きく乖離しています`,
      };
    case "low-note":
      return {
        className: "text-blue-700",
        message: `相場比 ${pctStr} — 相場より低め。利回り改善の余地があります`,
      };
    default:
      return null;
  }
}

export function RentValidationHint({ monthlyRent, area, city }: RentValidationHintProps) {
  const { deviationPct, level, loading } = useRentValidation(monthlyRent, area, city);

  if (loading) {
    return <p className="text-xs text-muted-foreground mt-1">賃料相場を確認中…</p>;
  }

  if (deviationPct === null || level === null) return null;

  const info = badge(level, deviationPct);
  if (!info) return null;

  return <p className={`text-xs mt-1 ${info.className}`}>{info.message}</p>;
}

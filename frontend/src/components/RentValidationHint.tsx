"use client";
import React from "react";
import { useRentValidation, type RentDeviationLevel } from "@/hooks/useRentValidation";

interface RentValidationHintProps {
  monthlyRent: number;
  area: string;
  city: string;
  landArea?: number;
}

function badge(
  level: RentDeviationLevel,
  deviationPct: number,
  city: string
): { className: string; message: string } | null {
  if (level === null) return null;
  const sign = deviationPct >= 0 ? "+" : "";
  const pctStr = `${sign}${deviationPct.toFixed(1)}%`;
  const scopeNote = city ? "" : "（都道府県全体）";

  switch (level) {
    case "normal":
      return {
        className: "text-muted-foreground",
        message: `相場比 ${pctStr}${scopeNote}（適正範囲）`,
      };
    case "high-warn":
      return {
        className: "text-yellow-700",
        message: `相場比 ${pctStr}${scopeNote} — 相場より高め。空室リスクを考慮してください`,
      };
    case "high-danger":
      return {
        className: "text-destructive",
        message: `相場比 ${pctStr}${scopeNote} — 相場から大きく乖離しています`,
      };
    case "low-note":
      return {
        className: "text-blue-700",
        message: `相場比 ${pctStr}${scopeNote} — 相場より低め。利回り改善の余地があります`,
      };
    default:
      return null;
  }
}

export function RentValidationHint({ monthlyRent, area, city, landArea }: RentValidationHintProps) {
  const { stats, deviationPct, level, loading, lowSample, lowConfidence } = useRentValidation(
    monthlyRent,
    area,
    city,
    landArea
  );

  if (loading) {
    return <p className="text-xs text-muted-foreground mt-1">賃料相場を確認中…</p>;
  }

  if (deviationPct === null || level === null) return null;

  const info = badge(level, deviationPct, city);
  if (!info) return null;

  return (
    <div className="mt-1">
      <p className={`text-xs ${info.className}`}>{info.message}</p>
      {lowConfidence && stats && (
        <p className="text-xs text-destructive mt-0.5">
          ※ データが少なく参考程度です（{stats.count}件）
        </p>
      )}
      {!lowConfidence && lowSample && stats && (
        <p className="text-xs text-muted-foreground mt-0.5">
          ※ サンプル数が少ないため参考値です（{stats.count}件）
        </p>
      )}
    </div>
  );
}

"use client";
import { useState, useEffect, useRef } from "react";
import { fetchRentStats, type RentStatsResult } from "@/lib/api";

export type RentDeviationLevel = "normal" | "high-warn" | "high-danger" | "low-note" | null;

export interface RentValidationState {
  stats: RentStatsResult | null;
  deviationPct: number | null;
  level: RentDeviationLevel;
  loading: boolean;
}

const DEBOUNCE_MS = 600;

/**
 * monthlyRent と area/municipality が揃ったとき、エリア賃料相場を取得して
 * 入力賃料の乖離度を返す。
 * - ±10% 以内     → "normal"
 * - +10%〜+30%   → "high-warn"
 * - +30% 超       → "high-danger"
 * - -20% 以下     → "low-note"
 */
export function useRentValidation(
  monthlyRent: number,
  area: string,
  city: string
): RentValidationState {
  const [stats, setStats] = useState<RentStatsResult | null>(null);
  const [loading, setLoading] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    // area が未設定、または賃料が 0 以下の場合はリセット
    if (!area || monthlyRent <= 0) {
      setStats(null);
      setLoading(false);
      return;
    }

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    timerRef.current = setTimeout(async () => {
      setLoading(true);
      try {
        const result = await fetchRentStats({
          area,
          municipality: city || undefined,
        });
        setStats(result);
      } catch {
        // データなし・ネットワークエラーはサイレント
        setStats(null);
      } finally {
        setLoading(false);
      }
    }, DEBOUNCE_MS);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [monthlyRent, area, city]);

  if (!stats || !stats.median || monthlyRent <= 0) {
    return { stats, deviationPct: null, level: null, loading };
  }

  const deviationPct = ((monthlyRent - stats.median) / stats.median) * 100;

  let level: RentDeviationLevel = "normal";
  if (deviationPct >= 30) {
    level = "high-danger";
  } else if (deviationPct >= 10) {
    level = "high-warn";
  } else if (deviationPct <= -20) {
    level = "low-note";
  }

  return { stats, deviationPct, level, loading };
}

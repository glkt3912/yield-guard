"use client";
import { useState, useCallback, useRef, useEffect } from "react";
import { fetchRentDeclineHint } from "@/lib/api";
import type { RentDeclineHint } from "@/types/investment";

export interface UseRentDeclineHintResult {
  rentHint: RentDeclineHint | null;
  loading: boolean;
  error: string | null;
  fetch: (area: string, city: string) => Promise<void>;
  clear: () => void;
}

export function useRentDeclineHint(
  onHintApplied: (rate: number) => void
): UseRentDeclineHintResult {
  const [rentHint, setRentHint] = useState<RentDeclineHint | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Stable ref to avoid stale closure in fetch callback
  const onHintAppliedRef = useRef(onHintApplied);
  useEffect(() => {
    onHintAppliedRef.current = onHintApplied;
  });

  const fetch = useCallback(async (area: string, city: string) => {
    setLoading(true);
    setError(null);
    setRentHint(null);
    try {
      const hint = await fetchRentDeclineHint({ area, municipality: city || undefined });
      setRentHint(hint);
      if (!hint.fallbackUsed && hint.hintRate > 0) {
        onHintAppliedRef.current(hint.hintRate);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "参考値の取得に失敗しました");
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setRentHint(null);
    setError(null);
  }, []);

  return { rentHint, loading, error, fetch, clear };
}

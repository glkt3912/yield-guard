"use client";
import { useState, useEffect, useCallback, useMemo } from "react";
import { fetchMunicipalities, type Municipality } from "@/lib/api";

export interface UseMunicipalitiesLoaderResult {
  municipalities: Municipality[];
  loading: boolean;
  error: string | null;
  city: string;
  setCity: (id: string) => void;
  muniFilter: string;
  setMuniFilter: (v: string) => void;
  filteredMunicipalities: Municipality[];
  loadMunicipalities: (areaCode: string) => Promise<void>;
}

export function useMunicipalitiesLoader(
  initialAreaCode = "10"
): UseMunicipalitiesLoaderResult {
  const [municipalities, setMunicipalities] = useState<Municipality[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [city, setCity] = useState("");
  const [muniFilter, setMuniFilter] = useState("");

  const filteredMunicipalities = useMemo(
    () =>
      muniFilter.trim()
        ? municipalities.filter((m) => m.name.includes(muniFilter.trim()))
        : municipalities,
    [municipalities, muniFilter]
  );

  const loadMunicipalities = useCallback(async (areaCode: string) => {
    setLoading(true);
    setError(null);
    setCity("");
    setMuniFilter("");
    try {
      const data = await fetchMunicipalities(areaCode);
      setMunicipalities(data);
      if (data.length > 0) setCity(data[0].id);
    } catch (e) {
      setMunicipalities([]);
      setError(e instanceof Error ? e.message : "市区町村の取得に失敗しました");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadMunicipalities(initialAreaCode);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadMunicipalities]);

  // 絞り込み結果が1件になったとき自動選択
  useEffect(() => {
    if (filteredMunicipalities.length === 1) {
      setCity(filteredMunicipalities[0].id);
    }
  }, [filteredMunicipalities]);

  return {
    municipalities,
    loading,
    error,
    city,
    setCity,
    muniFilter,
    setMuniFilter,
    filteredMunicipalities,
    loadMunicipalities,
  };
}

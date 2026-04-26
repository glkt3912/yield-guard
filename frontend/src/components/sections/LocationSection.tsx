"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Search } from "lucide-react";
import { PREFECTURES, getPeriodLabel } from "@/lib/investmentFormConstants";
import type { Municipality } from "@/lib/api";

const LOCATION_TYPE_LABEL: Record<string, string> = {
  ROOFTOP: "番地レベルで取得",
  RANGE_INTERPOLATED: "住所レベルで取得",
  GEOMETRIC_CENTER: "地点レベルで取得",
  APPROXIMATE: "近似位置で取得（精度低）",
};

interface LocationSectionProps {
  area: string;
  handleAreaChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
  city: string;
  handleCityChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
  muniFilter: string;
  setMuniFilter: (v: string) => void;
  filteredMunicipalities: Municipality[];
  muniLoading: boolean;
  muniError: string | null;
  isOnline: boolean | null;
  propertyLat: string;
  setPropertyLat: (v: string) => void;
  propertyLng: string;
  setPropertyLng: (v: string) => void;
  addressInput: string;
  setAddressInput: (v: string) => void;
  geocodeStatus: "idle" | "loading" | "success" | "error";
  setGeocodeStatus: (v: "idle" | "loading" | "success" | "error") => void;
  geocodeError: string;
  geocodeLocationType: string;
  setGeocodeLocationType: (v: string) => void;
  showManualCoords: boolean;
  handleGeocode: () => Promise<void>;
  loading: boolean;
  onFetchLandPrices: (area: string, city: string, lat?: number, lng?: number) => Promise<void>;
}

export function LocationSection({
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
  setPropertyLat,
  propertyLng,
  setPropertyLng,
  addressInput,
  setAddressInput,
  geocodeStatus,
  setGeocodeStatus,
  geocodeError,
  geocodeLocationType,
  setGeocodeLocationType,
  showManualCoords,
  handleGeocode,
  loading,
  onFetchLandPrices,
}: LocationSectionProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Search className="h-5 w-5 text-primary" />
          土地相場データ取得
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Select label="都道府県" value={area} onChange={handleAreaChange} options={PREFECTURES} />
          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-foreground">市区町村</label>
            <input
              type="text"
              placeholder="例: 前橋市"
              value={muniFilter}
              onChange={(e) => setMuniFilter(e.target.value)}
              className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
              aria-label="市区町村を検索"
            />
            <select
              value={city}
              onChange={handleCityChange}
              className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value="">（全市区町村）</option>
              {muniLoading ? (
                <option disabled>読み込み中...</option>
              ) : filteredMunicipalities.length === 0 && muniFilter.trim() ? (
                <option disabled>該当なし</option>
              ) : (
                filteredMunicipalities.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))
              )}
            </select>
            {muniError && <p className="text-xs text-destructive">{muniError}</p>}
            {muniFilter.trim() && filteredMunicipalities.length > 0 && (
              <p className="text-xs text-muted-foreground">{filteredMunicipalities.length}件該当</p>
            )}
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-foreground">物件住所（任意）</label>
            <p className="text-xs text-muted-foreground">
              丁目・番地まで入力してください（建物名・部屋番号は不要）
            </p>
            <div className="flex gap-2">
              <input
                type="text"
                placeholder="例: 東京都渋谷区道玄坂1-2"
                value={addressInput}
                onChange={(e) => {
                  setAddressInput(e.target.value);
                  setGeocodeStatus("idle");
                  setGeocodeLocationType("");
                  setPropertyLat("");
                  setPropertyLng("");
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleGeocode();
                }}
                className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                aria-label="物件住所"
              />
              <Button
                variant="outline"
                loading={geocodeStatus === "loading"}
                disabled={!addressInput.trim() || geocodeStatus === "loading"}
                onClick={handleGeocode}
                className="shrink-0"
              >
                座標を取得
              </Button>
            </div>
            {geocodeStatus === "success" && (
              <p className="text-xs text-green-600">
                ✓ {LOCATION_TYPE_LABEL[geocodeLocationType] ?? "座標を取得しました"}
              </p>
            )}
            {geocodeStatus === "error" && (
              <p className="text-xs text-destructive">{geocodeError}</p>
            )}
          </div>
          {showManualCoords && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="flex flex-col gap-1">
                <label className="text-sm font-medium text-foreground">緯度（手動入力）</label>
                <input
                  type="number"
                  inputMode="decimal"
                  placeholder="例: 35.6762"
                  step="0.0001"
                  value={propertyLat}
                  onChange={(e) => setPropertyLat(e.target.value)}
                  className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                  aria-label="物件の緯度"
                />
              </div>
              <div className="flex flex-col gap-1">
                <label className="text-sm font-medium text-foreground">経度（手動入力）</label>
                <input
                  type="number"
                  inputMode="decimal"
                  placeholder="例: 139.6503"
                  step="0.0001"
                  value={propertyLng}
                  onChange={(e) => setPropertyLng(e.target.value)}
                  className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                  aria-label="物件の経度"
                />
              </div>
            </div>
          )}
        </div>
        <p className="text-xs text-muted-foreground">
          {getPeriodLabel()}
          分の宅地取引実績（国交省公式API）を取得します。緯度・経度を入力すると周辺駅の需要スコアも取得します
        </p>
        {isOnline === false && (
          <p className="text-xs text-amber-700">オフライン中は相場取得を利用できません</p>
        )}
        <Button
          variant="outline"
          className="w-full"
          loading={loading}
          disabled={isOnline === false}
          onClick={() => {
            const lat = parseFloat(propertyLat);
            const lng = parseFloat(propertyLng);
            const hasCoords = !isNaN(lat) && !isNaN(lng);
            onFetchLandPrices(area, city, hasCoords ? lat : undefined, hasCoords ? lng : undefined);
          }}
        >
          <Search className="h-4 w-4" />
          相場データを取得
        </Button>
      </CardContent>
    </Card>
  );
}

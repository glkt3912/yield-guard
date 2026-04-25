"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import type { InvestmentInput, BuildingType, RateAdjustment, CapexEvent } from "@/types/investment";
import { toMan, fromMan, toPct, fromPct, formatPct } from "@/lib/utils";
import type { Municipality } from "@/lib/api";
import type { RentDeclineHint } from "@/types/investment";
import {
  PREFECTURES,
  BUILDING_TYPES,
  RENT_DECLINE_DEFAULTS,
  getPeriodLabel,
} from "@/lib/investmentFormConstants";
import { ZONING_TYPES, ZONING_META, type ZoningType } from "@/lib/zoning";
import { Search, Calculator, Info, AlertTriangle, ShieldCheck, Plus, Trash2 } from "lucide-react";

const LOCATION_TYPE_LABEL: Record<string, string> = {
  ROOFTOP: "番地レベルで取得",
  RANGE_INTERPOLATED: "住所レベルで取得",
  GEOMETRIC_CENTER: "地点レベルで取得",
  APPROXIMATE: "近似位置で取得（精度低）",
};

function calcFromTax(taxMan: number): number {
  return taxMan / 0.1;
}
function calcFromAssessment(totalMan: number, landA: number, buildingA: number): number {
  return landA + buildingA > 0 ? totalMan * (buildingA / (landA + buildingA)) : 0;
}

interface FullModeFormProps {
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
  input: InvestmentInput;
  setNum: (key: keyof InvestmentInput, value: number) => void;
  setStr: (key: keyof InvestmentInput, value: string) => void;
  fieldError: (key: string) => string | undefined;
  showBuildingHelper: boolean;
  setShowBuildingHelper: React.Dispatch<React.SetStateAction<boolean>>;
  taxAmount: string;
  setTaxAmount: (v: string) => void;
  landAssess: string;
  setLandAssess: (v: string) => void;
  buildingAssess: string;
  setBuildingAssess: (v: string) => void;
  totalPrice: string;
  setTotalPrice: (v: string) => void;
  rentHint: RentDeclineHint | null;
  rentHintLoading: boolean;
  rentHintError: string | null;
  handleFetchRentHint: () => Promise<void>;
  isCashPurchase: boolean;
  handleCashPurchaseToggle: (checked: boolean) => void;
  zoningType: ZoningType;
  setZoningType: (v: ZoningType) => void;
  rateScheduleEnabled: boolean;
  handleRateScheduleToggle: (enabled: boolean) => void;
  addRateStep: () => void;
  removeRateStep: (i: number) => void;
  updateRateStep: (i: number, field: keyof RateAdjustment, value: number) => void;
  canAddRateStep: boolean;
  addCapexEvent: () => void;
  removeCapexEvent: (i: number) => void;
  updateCapexEvent: (i: number, field: "year" | "amount", value: number) => void;
  hasErrors: boolean;
  handleAnalyze: () => void;
}

export function FullModeForm({
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
  input,
  setNum,
  setStr,
  fieldError,
  showBuildingHelper,
  setShowBuildingHelper,
  taxAmount,
  setTaxAmount,
  landAssess,
  setLandAssess,
  buildingAssess,
  setBuildingAssess,
  totalPrice,
  setTotalPrice,
  rentHint,
  rentHintLoading,
  rentHintError,
  handleFetchRentHint,
  isCashPurchase,
  handleCashPurchaseToggle,
  zoningType,
  setZoningType,
  rateScheduleEnabled,
  handleRateScheduleToggle,
  addRateStep,
  removeRateStep,
  updateRateStep,
  canAddRateStep,
  addCapexEvent,
  removeCapexEvent,
  updateCapexEvent,
  hasErrors,
  handleAnalyze,
}: FullModeFormProps) {
  return (
    <>
      {/* 相場データ取得（常時表示） */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Search className="h-5 w-5 text-primary" />
            土地相場データ取得
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Select
              label="都道府県"
              value={area}
              onChange={handleAreaChange}
              options={PREFECTURES}
            />
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-foreground">市区町村</label>
              <input
                type="text"
                placeholder="例: 前橋市"
                value={muniFilter}
                onChange={(e) => setMuniFilter(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                aria-label="市区町村を検索"
              />
              <select
                value={city}
                onChange={handleCityChange}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
                <p className="text-xs text-muted-foreground">
                  {filteredMunicipalities.length}件該当
                </p>
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
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
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
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
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
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
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
              onFetchLandPrices(
                area,
                city,
                hasCoords ? lat : undefined,
                hasCoords ? lng : undefined
              );
            }}
          >
            <Search className="h-4 w-4" />
            相場データを取得
          </Button>
        </CardContent>
      </Card>

      {/* 物件・投資条件（フラット） */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calculator className="h-5 w-5 text-primary" />
            物件・投資条件
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* 物件情報 */}
          <div>
            <p className="text-sm font-semibold text-foreground mb-3">物件情報</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <Input
                label="土地取得価格"
                type="number"
                inputMode="numeric"
                suffix="万円"
                value={toMan(input.landPrice)}
                onChange={(e) => setNum("landPrice", fromMan(e.target.value))}
                error={fieldError("landPrice")}
              />
              <Input
                label="土地面積"
                type="number"
                inputMode="decimal"
                suffix="m²"
                value={String(input.landArea)}
                onChange={(e) => setNum("landArea", parseFloat(e.target.value) || 0)}
              />
              <div className="sm:col-span-2">
                <Input
                  label="建物価格"
                  type="number"
                  inputMode="numeric"
                  suffix="万円"
                  value={toMan(input.buildingCost)}
                  onChange={(e) => setNum("buildingCost", fromMan(e.target.value))}
                  error={fieldError("buildingCost")}
                />
                <p className="text-xs text-muted-foreground mt-1">
                  新築は建設費、中古は売買契約書記載の建物価格（消費税の課税対象部分）を入力
                </p>

                <button
                  type="button"
                  className="mt-2 text-xs text-primary underline-offset-2 hover:underline"
                  onClick={() => setShowBuildingHelper((p) => !p)}
                >
                  {showBuildingHelper
                    ? "▲ 按分ヘルパーを閉じる"
                    : "▼ 建物価格がわからない場合（按分ヘルパー）"}
                </button>

                {showBuildingHelper && (
                  <div className="mt-2 rounded-md bg-muted/50 p-3 space-y-3 text-sm">
                    <div>
                      <p className="font-medium text-xs mb-1">① 消費税額から計算（最も確実）</p>
                      <div className="flex gap-2 items-end">
                        <div className="flex-1">
                          <Input
                            label="消費税額"
                            type="number"
                            inputMode="numeric"
                            suffix="万円"
                            value={taxAmount}
                            onChange={(e) => setTaxAmount(e.target.value)}
                          />
                        </div>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          className="mb-0 shrink-0"
                          onClick={() => {
                            const tax = parseFloat(taxAmount) || 0;
                            const result = calcFromTax(tax);
                            if (result > 0) setNum("buildingCost", result * 10_000);
                          }}
                        >
                          適用
                        </Button>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        建物価格 = 消費税額 ÷ 0.1
                      </p>
                    </div>
                    <div>
                      <p className="font-medium text-xs mb-1">② 固定資産税評価額の比率で按分</p>
                      <div className="grid grid-cols-3 gap-2">
                        <Input
                          label="土地評価額"
                          type="number"
                          inputMode="numeric"
                          suffix="万円"
                          value={landAssess}
                          onChange={(e) => setLandAssess(e.target.value)}
                        />
                        <Input
                          label="建物評価額"
                          type="number"
                          inputMode="numeric"
                          suffix="万円"
                          value={buildingAssess}
                          onChange={(e) => setBuildingAssess(e.target.value)}
                        />
                        <Input
                          label="購入総額"
                          type="number"
                          inputMode="numeric"
                          suffix="万円"
                          value={totalPrice}
                          onChange={(e) => setTotalPrice(e.target.value)}
                        />
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="mt-2"
                        onClick={() => {
                          const total = parseFloat(totalPrice) || 0;
                          const land = parseFloat(landAssess) || 0;
                          const building = parseFloat(buildingAssess) || 0;
                          const result = calcFromAssessment(total, land, building);
                          if (result > 0) setNum("buildingCost", result * 10_000);
                        }}
                      >
                        按分して適用
                      </Button>
                      <p className="text-xs text-muted-foreground mt-1">
                        建物価格 = 総額 × 建物評価 ÷（土地+建物評価）
                      </p>
                    </div>
                  </div>
                )}
              </div>
              <Input
                label="築年数"
                type="number"
                inputMode="numeric"
                suffix="年（0=新築）"
                value={String(input.buildingAge)}
                onChange={(e) => setNum("buildingAge", parseInt(e.target.value) || 0)}
              />
              <Input
                label="最寄り駅徒歩"
                type="number"
                inputMode="numeric"
                suffix="分（0=未入力）"
                value={String(input.stationMinutes)}
                onChange={(e) => setNum("stationMinutes", parseInt(e.target.value) || 0)}
              />
              <Select
                label="建物構造"
                value={input.buildingType}
                onChange={(e) => {
                  const bt = e.target.value as BuildingType;
                  setStr("buildingType", bt);
                  setNum("rentDeclineRate", RENT_DECLINE_DEFAULTS[bt]);
                }}
                options={BUILDING_TYPES}
              />
              <div>
                <Input
                  label="年間賃料下落率"
                  type="number"
                  suffix="%"
                  step="0.1"
                  value={toPct(input.rentDeclineRate, 1)}
                  onChange={(e) => setNum("rentDeclineRate", fromPct(e.target.value))}
                />
                <p className="text-xs text-muted-foreground mt-1">
                  建物構造から自動設定（例: 木造1%/年）
                </p>
                <div className="mt-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={!area || rentHintLoading}
                    onClick={handleFetchRentHint}
                    className="text-xs h-7 px-2"
                  >
                    {rentHintLoading ? "取得中…" : "地域の参考値を取得"}
                  </Button>
                  {rentHintError && (
                    <p className="text-xs text-destructive mt-1">{rentHintError}</p>
                  )}
                  {rentHint && !rentHintError && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {rentHint.fallbackUsed
                        ? "データ不足のため地域参考値なし。構造別平均値を推奨します"
                        : `地価公示より参考値: ${toPct(rentHint.hintRate, 1)}%/年（${rentHint.dataPointCount}件）`}
                    </p>
                  )}
                </div>
              </div>

              {/* 賃料上昇シナリオ（新築・リノベ向け） */}
              <div className="col-span-full">
                <details className="group">
                  <summary className="cursor-pointer text-xs text-muted-foreground hover:text-foreground select-none">
                    ▶ 賃料上昇期の設定（新築・リノベ向け）
                  </summary>
                  <div className="mt-2 grid grid-cols-2 gap-3 pl-2 border-l-2 border-muted">
                    <Input
                      label="年間上昇率"
                      type="number"
                      suffix="%"
                      step="0.1"
                      min="0"
                      max="10"
                      value={toPct(input.rentGrowthRate ?? 0, 1)}
                      onChange={(e) => setNum("rentGrowthRate", fromPct(e.target.value))}
                    />
                    <Input
                      label="上昇期間"
                      type="number"
                      suffix="年"
                      step="1"
                      min="0"
                      max="20"
                      value={String(input.rentGrowthYears ?? 0)}
                      onChange={(e) => setNum("rentGrowthYears", parseInt(e.target.value) || 0)}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground mt-1 pl-2">
                    上昇期終了後は年間賃料下落率が適用されます
                  </p>
                </details>
              </div>
            </div>
          </div>

          {/* 収益条件 */}
          <div className="border-t pt-4">
            <p className="text-sm font-semibold text-foreground mb-3">収益条件</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <Input
                label="想定月額賃料"
                type="number"
                inputMode="numeric"
                suffix="円"
                value={String(input.monthlyRent)}
                onChange={(e) => setNum("monthlyRent", parseFloat(e.target.value) || 0)}
                error={fieldError("monthlyRent")}
              />
              <Input
                label="現況空室率"
                type="number"
                inputMode="numeric"
                suffix="%"
                step="1"
                value={toPct(input.actualVacancyRate, 0)}
                onChange={(e) => setNum("actualVacancyRate", fromPct(e.target.value))}
              />
              <Input
                label="想定空室率（長期）"
                type="number"
                inputMode="numeric"
                suffix="%"
                step="1"
                value={toPct(input.vacancyRate, 0)}
                onChange={(e) => setNum("vacancyRate", fromPct(e.target.value))}
                error={fieldError("vacancyRate")}
              />
              <Input
                label="運営経費率※"
                type="number"
                inputMode="numeric"
                suffix="%"
                step="1"
                value={toPct(input.expenseRate, 0)}
                onChange={(e) => setNum("expenseRate", fromPct(e.target.value))}
                error={fieldError("expenseRate")}
              />
              <Input
                label="諸経費率"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="0.5"
                value={toPct(input.miscExpenseRate, 1)}
                onChange={(e) => setNum("miscExpenseRate", fromPct(e.target.value))}
                error={fieldError("miscExpenseRate")}
              />
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              ※運営経費率はローン利息を含みません（管理費・修繕費・固定資産税・保険等）
            </p>
            <div className="mt-3 space-y-2">
              <Select
                label="用途地域（任意）"
                value={zoningType}
                onChange={(e) => setZoningType(e.target.value as ZoningType)}
                options={[
                  { value: "", label: "（未選択）" },
                  ...ZONING_TYPES.map((z) => ({ value: z, label: z })),
                ]}
              />
              {zoningType &&
                (() => {
                  const meta = ZONING_META[zoningType];
                  if (!meta) return null;
                  if (meta.riskLevel === 0)
                    return (
                      <div className="flex items-start gap-2 rounded-md border border-green-200 bg-green-50 p-2 text-green-800 text-xs">
                        <ShieldCheck className="h-4 w-4 shrink-0 mt-0.5" />
                        <span>良好な住環境です</span>
                      </div>
                    );
                  const styles: Record<number, string> = {
                    1: "border-blue-200 bg-blue-50 text-blue-800",
                    2: "border-yellow-300 bg-yellow-50 text-yellow-800",
                    3: "border-red-300 bg-red-50 text-red-800",
                  };
                  const labels: Record<number, string> = {
                    1: "低リスク",
                    2: "中リスク",
                    3: "高リスク",
                  };
                  return (
                    <div
                      className={`flex items-start gap-2 rounded-md border p-2 text-xs ${styles[meta.riskLevel]}`}
                    >
                      <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                      <div>
                        <span className="font-semibold">{labels[meta.riskLevel]}: </span>
                        <span>{meta.riskMessage}</span>
                      </div>
                    </div>
                  );
                })()}
            </div>
          </div>

          {/* ローン条件 */}
          <div className="border-t pt-4">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm font-semibold text-foreground">ローン条件</p>
              <label className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={isCashPurchase}
                  onChange={(e) => handleCashPurchaseToggle(e.target.checked)}
                  className="h-4 w-4 rounded border-input accent-primary"
                  aria-label="現金購入（ローンなし）"
                />
                <span className="text-sm font-medium">現金購入（ローンなし）</span>
              </label>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <Input
                label="ローン金額"
                type="number"
                inputMode="numeric"
                suffix="万円"
                value={toMan(input.loanAmount)}
                onChange={(e) => {
                  if (!isCashPurchase) setNum("loanAmount", fromMan(e.target.value));
                }}
                error={fieldError("loanAmount")}
                disabled={isCashPurchase}
              />
              <Input
                label="年利"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="0.01"
                value={toPct(input.annualLoanRate)}
                onChange={(e) => {
                  if (!isCashPurchase) setNum("annualLoanRate", fromPct(e.target.value));
                }}
                error={fieldError("annualLoanRate")}
                disabled={isCashPurchase}
              />
              <Input
                label="返済期間"
                type="number"
                inputMode="numeric"
                suffix="年"
                value={String(input.loanYears)}
                onChange={(e) => {
                  if (!isCashPurchase) setNum("loanYears", parseInt(e.target.value) || 0);
                }}
                error={fieldError("loanYears")}
                disabled={isCashPurchase}
              />
            </div>
          </div>

          {/* 出口戦略 */}
          <div className="border-t pt-4">
            <p className="text-sm font-semibold text-foreground mb-3">出口戦略</p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <Input
                label="売却予定年数"
                type="number"
                inputMode="numeric"
                suffix="年後"
                value={String(input.holdingYears)}
                onChange={(e) => setNum("holdingYears", parseInt(e.target.value) || 10)}
                error={fieldError("holdingYears")}
              />
              <Input
                label="売却時目標利回り（実質）"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="0.5"
                value={toPct(input.exitYieldTarget, 1)}
                onChange={(e) => setNum("exitYieldTarget", fromPct(e.target.value))}
                error={fieldError("exitYieldTarget")}
              />
              <Input
                label="目標表面利回り"
                type="number"
                inputMode="decimal"
                suffix="%"
                step="0.5"
                value={toPct(input.yieldTarget, 1)}
                onChange={(e) => setNum("yieldTarget", fromPct(e.target.value))}
                error={fieldError("yieldTarget")}
              />
              <Input
                label="所得税率（実効）"
                type="number"
                inputMode="numeric"
                suffix="%"
                step="1"
                value={toPct(input.incomeTaxRate, 0)}
                onChange={(e) => setNum("incomeTaxRate", fromPct(e.target.value))}
                error={fieldError("incomeTaxRate")}
              />
            </div>

            {/* NPV / IRR 設定 */}
            <div className="mt-4">
              <p className="text-xs font-semibold text-muted-foreground mb-2">NPV / IRR 設定</p>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <Input
                  label="割引率"
                  type="number"
                  inputMode="decimal"
                  suffix="%"
                  step="0.1"
                  value={toPct(input.discountRate ?? 0.05, 1)}
                  onChange={(e) => setNum("discountRate", fromPct(e.target.value))}
                  error={fieldError("discountRate")}
                />
                <Input
                  label="物件価格下落率"
                  type="number"
                  inputMode="decimal"
                  suffix="%"
                  step="0.1"
                  value={toPct(input.priceDeclineRate ?? 0, 1)}
                  onChange={(e) => setNum("priceDeclineRate", fromPct(e.target.value))}
                  error={fieldError("priceDeclineRate")}
                />
                <Select
                  label="減価償却方式"
                  value={input.depreciationMethod ?? "straight-line"}
                  onChange={(e) => setStr("depreciationMethod", e.target.value)}
                  options={[
                    { value: "straight-line", label: "定額法" },
                    { value: "declining-balance", label: "定率法" },
                  ]}
                />
              </div>
            </div>
          </div>

          {/* 変動金利スケジュール */}
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
                    step.rate <= 0 || step.rate > 0.3
                      ? "0%超〜30%の範囲で入力してください"
                      : undefined;
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
                          updateRateStep(i, "afterYear", parseInt(e.target.value) || 2)
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
                      onChange={(e) => updateCapexEvent(i, "amount", (parseFloat(e.target.value) || 0) * 10_000)}
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

          <Button
            className="w-full"
            size="lg"
            loading={loading}
            disabled={hasErrors || loading}
            onClick={handleAnalyze}
          >
            <Calculator className="h-5 w-5" />
            シミュレーション実行
          </Button>

          <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3 text-xs text-yellow-800 space-y-1">
            <p className="flex items-center gap-1 font-semibold">
              <Info className="h-3 w-3" />
              免責事項
            </p>
            <ul className="list-disc list-inside space-y-0.5">
              <li>計算結果は参考値であり、税務上の助言ではありません</li>
              <li>消費税・損益通算・各種特例（3000万控除等）は考慮していません</li>
              <li>所得税率は給与所得との合算後の実効税率を入力してください</li>
              <li>中古物件の耐用年数は「築年数」から簡便法で算出しています</li>
              <li>実際の投資判断は税理士・不動産の専門家にご相談ください</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </>
  );
}

"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Calculator, Info } from "lucide-react";
import type { InvestmentInput, RateAdjustment, RentDeclineHint } from "@/types/investment";
import type { Municipality } from "@/lib/api";
import type { ZoningType } from "@/lib/zoning";
import { LocationSection } from "@/components/sections/LocationSection";
import { PropertyInfoSection } from "@/components/sections/PropertyInfoSection";
import { LoanSection } from "@/components/sections/LoanSection";
import { ScenarioSection } from "@/components/sections/ScenarioSection";

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
      <LocationSection
        area={area}
        handleAreaChange={handleAreaChange}
        city={city}
        handleCityChange={handleCityChange}
        muniFilter={muniFilter}
        setMuniFilter={setMuniFilter}
        filteredMunicipalities={filteredMunicipalities}
        muniLoading={muniLoading}
        muniError={muniError}
        isOnline={isOnline}
        propertyLat={propertyLat}
        setPropertyLat={setPropertyLat}
        propertyLng={propertyLng}
        setPropertyLng={setPropertyLng}
        addressInput={addressInput}
        setAddressInput={setAddressInput}
        geocodeStatus={geocodeStatus}
        setGeocodeStatus={setGeocodeStatus}
        geocodeError={geocodeError}
        geocodeLocationType={geocodeLocationType}
        setGeocodeLocationType={setGeocodeLocationType}
        showManualCoords={showManualCoords}
        handleGeocode={handleGeocode}
        loading={loading}
        onFetchLandPrices={onFetchLandPrices}
      />

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calculator className="h-5 w-5 text-primary" />
            物件・投資条件
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <PropertyInfoSection
            input={input}
            setNum={setNum}
            setStr={setStr}
            fieldError={fieldError}
            showBuildingHelper={showBuildingHelper}
            setShowBuildingHelper={setShowBuildingHelper}
            taxAmount={taxAmount}
            setTaxAmount={setTaxAmount}
            landAssess={landAssess}
            setLandAssess={setLandAssess}
            buildingAssess={buildingAssess}
            setBuildingAssess={setBuildingAssess}
            totalPrice={totalPrice}
            setTotalPrice={setTotalPrice}
            rentHint={rentHint}
            rentHintLoading={rentHintLoading}
            rentHintError={rentHintError}
            handleFetchRentHint={handleFetchRentHint}
            zoningType={zoningType}
            setZoningType={setZoningType}
          />

          <LoanSection
            input={input}
            setNum={setNum}
            setStr={setStr}
            fieldError={fieldError}
            isCashPurchase={isCashPurchase}
            handleCashPurchaseToggle={handleCashPurchaseToggle}
          />

          <ScenarioSection
            input={input}
            setNum={setNum}
            rateScheduleEnabled={rateScheduleEnabled}
            handleRateScheduleToggle={handleRateScheduleToggle}
            addRateStep={addRateStep}
            removeRateStep={removeRateStep}
            updateRateStep={updateRateStep}
            canAddRateStep={canAddRateStep}
            addCapexEvent={addCapexEvent}
            removeCapexEvent={removeCapexEvent}
            updateCapexEvent={updateCapexEvent}
          />

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

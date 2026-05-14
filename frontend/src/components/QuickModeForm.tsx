"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import type { InvestmentInput } from "@/types/investment";
import { toMan, fromMan, toPct, fromPct } from "@/lib/utils";
import type { Municipality } from "@/lib/api";
import { PREFECTURES, getPeriodLabel } from "@/lib/investmentFormConstants";
import { Search, Calculator, Info, ChevronDown, ChevronUp } from "lucide-react";
import { RentValidationHint } from "@/components/RentValidationHint";

export interface QuickHistoryEntry {
  totalPriceMan: string;
  monthlyRentYen: string;
  ts: number;
}

interface QuickModeFormProps {
  area: string;
  handleAreaChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
  city: string;
  handleCityChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
  muniFilter: string;
  setMuniFilter: (v: string) => void;
  filteredMunicipalities: Municipality[];
  muniLoading: boolean;
  muniError: string | null;
  showLandSection: boolean;
  setShowLandSection: React.Dispatch<React.SetStateAction<boolean>>;
  isOnline: boolean | null;
  propertyLat: string;
  propertyLng: string;
  loading: boolean;
  onFetchLandPrices: (area: string, city: string, lat?: number, lng?: number) => Promise<void>;
  quickHistory: QuickHistoryEntry[];
  handleRestoreLastInput: () => void;
  quickTotalPriceMan: string;
  setQuickTotalPriceMan: (v: string) => void;
  markTouched: (key: string) => void;
  fieldError: (key: string) => string | undefined;
  input: InvestmentInput;
  setNum: (key: keyof InvestmentInput, value: number) => void;
  isCashPurchase: boolean;
  handleCashPurchaseToggle: (checked: boolean) => void;
  showCustomLoan: boolean;
  setShowCustomLoan: (v: boolean) => void;
  hasErrors: boolean;
  handleAnalyze: () => void;
}

export function QuickModeForm({
  area,
  handleAreaChange,
  city,
  handleCityChange,
  muniFilter,
  setMuniFilter,
  filteredMunicipalities,
  muniLoading,
  muniError,
  showLandSection,
  setShowLandSection,
  isOnline,
  propertyLat,
  propertyLng,
  loading,
  onFetchLandPrices,
  quickHistory,
  handleRestoreLastInput,
  quickTotalPriceMan,
  setQuickTotalPriceMan,
  markTouched,
  fieldError,
  input,
  setNum,
  isCashPurchase,
  handleCashPurchaseToggle,
  showCustomLoan,
  setShowCustomLoan,
  hasErrors,
  handleAnalyze,
}: QuickModeFormProps) {
  return (
    <>
      {/* 土地相場データ取得（折りたたみ） */}
      <div className="rounded-lg border">
        <button
          type="button"
          className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium text-muted-foreground hover:text-foreground"
          onClick={() => setShowLandSection((p) => !p)}
        >
          <span className="flex items-center gap-2">
            <Search className="h-4 w-4" />
            土地相場データを取得（任意）
          </span>
          {showLandSection ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          )}
        </button>
        {showLandSection && (
          <div className="border-t px-4 pb-4 pt-3 space-y-3">
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
                  className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                />
                <select
                  value={city}
                  onChange={handleCityChange}
                  className="flex h-11 sm:h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <option value="">（全市区町村）</option>
                  {muniLoading ? (
                    <option disabled>読み込み中...</option>
                  ) : (
                    filteredMunicipalities.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.name}
                      </option>
                    ))
                  )}
                </select>
                {muniError && <p className="text-xs text-destructive">{muniError}</p>}
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              {getPeriodLabel()}分の宅地取引実績（国交省公式API）を取得します
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
          </div>
        )}
      </div>

      {/* クイックモード入力フォーム */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calculator className="h-5 w-5 text-primary" />
            物件・投資条件
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {quickHistory.length > 0 && (
            <button
              type="button"
              className="flex w-full items-center justify-center gap-2 rounded-md border border-primary/30 bg-primary/5 px-3 py-2 min-h-[44px] text-sm font-medium text-primary hover:bg-primary/10 transition-colors"
              onClick={handleRestoreLastInput}
            >
              前回の入力から開始
            </button>
          )}
          <div className="space-y-3">
            <div>
              <Input
                label="物件価格（土地＋建物の総額）"
                type="number"
                inputMode="numeric"
                suffix="万円"
                value={quickTotalPriceMan}
                onChange={(e) => {
                  setQuickTotalPriceMan(e.target.value);
                  markTouched("quickTotalPrice");
                }}
                error={fieldError("quickTotalPrice")}
              />
              <p className="text-xs text-muted-foreground mt-1">
                内部で土地70%・建物30%に按分して計算します
              </p>
            </div>
            <div>
              <Input
                label="想定月額賃料"
                type="number"
                inputMode="numeric"
                suffix="円"
                value={String(input.monthlyRent)}
                onChange={(e) => setNum("monthlyRent", parseFloat(e.target.value) || 0)}
                error={fieldError("monthlyRent")}
              />
              <RentValidationHint monthlyRent={input.monthlyRent} area={area} city={city} landArea={input.landArea} />
            </div>
            <div className="space-y-2">
              <label className="flex items-center gap-2 cursor-pointer select-none min-h-[44px]">
                <input
                  type="checkbox"
                  checked={isCashPurchase}
                  onChange={(e) => handleCashPurchaseToggle(e.target.checked)}
                  className="h-4 w-4 rounded border-input accent-primary"
                  aria-label="現金購入（ローンなし）"
                />
                <span className="text-sm font-medium">現金購入（ローンなし）</span>
              </label>
              {!isCashPurchase && (
                <>
                  {!showCustomLoan && (
                    <p className="text-xs text-muted-foreground">
                      ローン 80% 自動適用中 —{" "}
                      <button
                        type="button"
                        className="underline underline-offset-2 hover:text-foreground"
                        onClick={() => setShowCustomLoan(true)}
                      >
                        カスタム設定
                      </button>
                    </p>
                  )}
                  {showCustomLoan && (
                    <div className="space-y-1">
                      <Input
                        label="ローン金額"
                        type="number"
                        inputMode="numeric"
                        suffix="万円"
                        value={toMan(input.loanAmount)}
                        onChange={(e) => setNum("loanAmount", fromMan(e.target.value))}
                        error={fieldError("loanAmount")}
                      />
                      <button
                        type="button"
                        className="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
                        onClick={() => setShowCustomLoan(false)}
                      >
                        80% 自動計算に戻す
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
            <Input
              label="目標利回り"
              type="number"
              inputMode="decimal"
              suffix="%"
              step="0.5"
              value={toPct(input.yieldTarget, 1)}
              onChange={(e) => setNum("yieldTarget", fromPct(e.target.value))}
              error={fieldError("yieldTarget")}
            />
          </div>

          <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-800 space-y-0.5">
            <p className="font-semibold">自動設定されるデフォルト値</p>
            <p>
              空室率 5%・運営経費率 20%・建物構造: 木造・
              {isCashPurchase
                ? "現金購入（ローンなし）"
                : !showCustomLoan
                  ? "ローン 80%・年利 1.5%・返済期間 35年"
                  : "年利 1.5%・返済期間 35年"}
              ・10年後売却・所得税率 33%
            </p>
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
              <li>実際の投資判断は税理士・不動産の専門家にご相談ください</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </>
  );
}

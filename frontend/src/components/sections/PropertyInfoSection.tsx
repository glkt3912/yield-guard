"use client";
import React, { useState } from "react";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { AlertTriangle, ShieldCheck } from "lucide-react";
import type { InvestmentInput, BuildingType, RentDeclineHint } from "@/types/investment";
import { toMan, fromMan, toPct, fromPct } from "@/lib/utils";
import { BUILDING_TYPES, RENT_DECLINE_DEFAULTS } from "@/lib/investmentFormConstants";
import { ZONING_TYPES, ZONING_META, type ZoningType } from "@/lib/zoning";

function calcFromTax(taxMan: number): number {
  return taxMan / 0.1;
}
function calcFromAssessment(totalMan: number, landA: number, buildingA: number): number {
  return landA + buildingA > 0 ? totalMan * (buildingA / (landA + buildingA)) : 0;
}

// 建物価格按分ヘルパー（state を自己完結）
function BuildingPriceHelper({ onApply }: { onApply: (value: number) => void }) {
  const [taxAmount, setTaxAmount] = useState("");
  const [landAssess, setLandAssess] = useState("");
  const [buildingAssess, setBuildingAssess] = useState("");
  const [totalPrice, setTotalPrice] = useState("");

  return (
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
              const result = calcFromTax(parseFloat(taxAmount) || 0);
              if (result > 0) onApply(result * 10_000);
            }}
          >
            適用
          </Button>
        </div>
        <p className="text-xs text-muted-foreground mt-1">建物価格 = 消費税額 ÷ 0.1</p>
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
            const result = calcFromAssessment(
              parseFloat(totalPrice) || 0,
              parseFloat(landAssess) || 0,
              parseFloat(buildingAssess) || 0
            );
            if (result > 0) onApply(result * 10_000);
          }}
        >
          按分して適用
        </Button>
        <p className="text-xs text-muted-foreground mt-1">
          建物価格 = 総額 × 建物評価 ÷（土地+建物評価）
        </p>
      </div>
    </div>
  );
}

// 賃料下落率ヒント表示
function RentDeclineHintDisplay({
  hint,
  loading,
  error,
  onFetch,
  disabled,
}: {
  hint: RentDeclineHint | null;
  loading: boolean;
  error: string | null;
  onFetch: () => Promise<void>;
  disabled: boolean;
}) {
  return (
    <div className="mt-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || loading}
        onClick={onFetch}
        className="text-xs h-7 px-2"
      >
        {loading ? "取得中…" : "地域の参考値を取得"}
      </Button>
      {error && <p className="text-xs text-destructive mt-1">{error}</p>}
      {hint && !error && (
        <p className="text-xs text-muted-foreground mt-1">
          {hint.fallbackUsed
            ? "データ不足のため地域参考値なし。構造別平均値を推奨します"
            : `地価公示より参考値: ${toPct(hint.hintRate, 1)}%/年（${hint.dataPointCount}件）`}
        </p>
      )}
    </div>
  );
}

// 用途地域リスクバッジ
function ZoningRiskBadge({
  zoningType,
  onZoningChange,
}: {
  zoningType: ZoningType;
  onZoningChange: (v: ZoningType) => void;
}) {
  const meta = zoningType ? ZONING_META[zoningType] : null;
  const styles: Record<number, string> = {
    0: "border-green-200 bg-green-50 text-green-800",
    1: "border-blue-200 bg-blue-50 text-blue-800",
    2: "border-yellow-300 bg-yellow-50 text-yellow-800",
    3: "border-red-300 bg-red-50 text-red-800",
  };
  const labels: Record<number, string> = { 0: "", 1: "低リスク", 2: "中リスク", 3: "高リスク" };

  return (
    <div className="space-y-2">
      <Select
        label="用途地域（任意）"
        value={zoningType}
        onChange={(e) => onZoningChange(e.target.value as ZoningType)}
        options={[
          { value: "", label: "（未選択）" },
          ...ZONING_TYPES.map((z) => ({ value: z, label: z })),
        ]}
      />
      {meta && (
        <div
          className={`flex items-start gap-2 rounded-md border p-2 text-xs ${styles[meta.riskLevel]}`}
        >
          {meta.riskLevel === 0 ? (
            <ShieldCheck className="h-4 w-4 shrink-0 mt-0.5" />
          ) : (
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
          )}
          <div>
            {labels[meta.riskLevel] && (
              <span className="font-semibold">{labels[meta.riskLevel]}: </span>
            )}
            <span>{meta.riskLevel === 0 ? "良好な住環境です" : meta.riskMessage}</span>
          </div>
        </div>
      )}
    </div>
  );
}

interface PropertyInfoSectionProps {
  input: InvestmentInput;
  setNum: (key: keyof InvestmentInput, value: number) => void;
  setStr: (key: keyof InvestmentInput, value: string) => void;
  fieldError: (key: string) => string | undefined;
  rentHint: RentDeclineHint | null;
  rentHintLoading: boolean;
  rentHintError: string | null;
  handleFetchRentHint: () => Promise<void>;
  zoningType: ZoningType;
  setZoningType: (v: ZoningType) => void;
}

export function PropertyInfoSection({
  input,
  setNum,
  setStr,
  fieldError,
  rentHint,
  rentHintLoading,
  rentHintError,
  handleFetchRentHint,
  zoningType,
  setZoningType,
}: PropertyInfoSectionProps) {
  const [showBuildingHelper, setShowBuildingHelper] = useState(false);

  return (
    <>
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
            <div className={showBuildingHelper ? "block" : "hidden"}>
              <BuildingPriceHelper onApply={(value) => setNum("buildingCost", value)} />
            </div>
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
            <RentDeclineHintDisplay
              hint={rentHint}
              loading={rentHintLoading}
              error={rentHintError}
              onFetch={handleFetchRentHint}
              disabled={!input.buildingType}
            />
          </div>
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
          <Input
            label="融資諸費用率"
            type="number"
            inputMode="decimal"
            suffix="%"
            step="0.1"
            value={toPct(input.loanFeeRate ?? 0, 1)}
            onChange={(e) => setNum("loanFeeRate", fromPct(e.target.value))}
          />
        </div>
        <p className="text-xs text-muted-foreground mt-2">
          ※運営経費率はローン利息を含みません（管理費・修繕費・固定資産税・保険等）
        </p>
        <div className="mt-3">
          <ZoningRiskBadge zoningType={zoningType} onZoningChange={setZoningType} />
        </div>
      </div>
    </>
  );
}

"use client";
import React from "react";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import type { InvestmentInput } from "@/types/investment";
import { toMan, fromMan, toPct, fromPct } from "@/lib/utils";

interface LoanSectionProps {
  input: InvestmentInput;
  setNum: (key: keyof InvestmentInput, value: number) => void;
  setStr: (key: keyof InvestmentInput, value: string) => void;
  fieldError: (key: string) => string | undefined;
  isCashPurchase: boolean;
  handleCashPurchaseToggle: (checked: boolean) => void;
}

export function LoanSection({
  input,
  setNum,
  setStr,
  fieldError,
  isCashPurchase,
  handleCashPurchaseToggle,
}: LoanSectionProps) {
  return (
    <>
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
            <div className="flex flex-col gap-1">
              <Select
                label="減価償却方式"
                value={input.depreciationMethod ?? "straight-line"}
                onChange={(e) => setStr("depreciationMethod", e.target.value)}
                options={[
                  { value: "straight-line", label: "定額法" },
                  { value: "declining-balance", label: "定率法" },
                ]}
              />
              <p className="text-xs text-muted-foreground">
                ※ 定率法は建物には適用不可（1998年4月以降取得）
              </p>
              {(input.depreciationMethod ?? "straight-line") === "declining-balance" && (
                <p className="text-yellow-700 bg-yellow-50 border border-yellow-200 rounded p-2 text-xs">
                  ⚠️
                  定率法は建物には適用できません。バックエンドでエラーになります。定額法を選択してください。
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

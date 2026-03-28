"use client";
import React, { useState, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import type { InvestmentInput, BuildingType } from "@/types/investment";
import { DEFAULT_INPUT } from "@/types/investment";
import { formatPct } from "@/lib/utils";
import { fetchPrefectures } from "@/lib/api";
import { Search, Calculator } from "lucide-react";

interface Props {
  onAnalyze: (input: InvestmentInput) => Promise<void>;
  onFetchLandPrices: (area: string, city: string) => Promise<void>;
  loading: boolean;
}

const BUILDING_TYPES: { value: BuildingType; label: string }[] = [
  { value: "木造", label: "木造（耐用22年）" },
  { value: "軽量鉄骨", label: "軽量鉄骨（耐用27年）" },
  { value: "重量鉄骨", label: "重量鉄骨（耐用34年）" },
  { value: "RC造", label: "RC造（耐用47年）" },
];

export function InvestmentForm({ onAnalyze, onFetchLandPrices, loading }: Props) {
  const [input, setInput] = useState<InvestmentInput>(DEFAULT_INPUT);
  const [area, setArea] = useState("10");
  const [city, setCity] = useState("10201");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [prefectures, setPrefectures] = useState<{ value: string; label: string }[]>([]);

  useEffect(() => {
    fetchPrefectures()
      .then((list) => {
        const sorted = list
          .map((p) => ({ value: p.code, label: p.name }))
          .sort((a, b) => Number(a.value) - Number(b.value));
        setPrefectures(sorted);
      })
      .catch(() => {
        // バックエンド未起動時のフォールバック
        setPrefectures([{ value: "10", label: "群馬県（フォールバック）" }]);
      });
  }, []);

  const set = (key: keyof InvestmentInput, value: number | string) =>
    setInput((prev) => ({ ...prev, [key]: value }));

  const manToYen = (v: string) => {
    const n = parseFloat(v);
    return isNaN(n) ? 0 : n * 10_000;
  };

  const yenToMan = (v: number) => (v / 10_000).toFixed(0);

  return (
    <div className="space-y-4">
      {/* 相場データ取得 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Search className="h-5 w-5 text-primary" />
            土地相場データ取得
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Select
              label="都道府県"
              value={area}
              onChange={(e) => setArea(e.target.value)}
              options={prefectures}
            />
            <Input
              label="市区町村コード"
              value={city}
              onChange={(e) => setCity(e.target.value)}
              placeholder="例: 10201"
            />
          </div>
          <p className="text-xs text-muted-foreground">
            直近2年分（2023〜2024年）の宅地取引実績を取得します
          </p>
          <Button
            variant="outline"
            className="w-full"
            loading={loading}
            onClick={() => onFetchLandPrices(area, city)}
          >
            <Search className="h-4 w-4" />
            相場データを取得
          </Button>
        </CardContent>
      </Card>

      {/* 物件情報 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calculator className="h-5 w-5 text-primary" />
            物件・投資条件
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <Input
              label="土地取得価格"
              type="number"
              suffix="万円"
              defaultValue={yenToMan(input.landPrice)}
              onBlur={(e) => set("landPrice", manToYen(e.target.value))}
            />
            <Input
              label="建築費"
              type="number"
              suffix="万円"
              defaultValue={yenToMan(input.buildingCost)}
              onBlur={(e) => set("buildingCost", manToYen(e.target.value))}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Input
              label="想定月額賃料"
              type="number"
              suffix="円"
              defaultValue={input.monthlyRent}
              onBlur={(e) => set("monthlyRent", parseFloat(e.target.value) || 0)}
            />
            <Select
              label="建物構造"
              value={input.buildingType}
              onChange={(e) => set("buildingType", e.target.value as BuildingType)}
              options={BUILDING_TYPES}
            />
          </div>

          {/* ローン条件 */}
          <div className="border-t pt-3">
            <p className="mb-3 text-sm font-medium text-muted-foreground">ローン条件</p>
            <div className="grid grid-cols-3 gap-3">
              <Input
                label="ローン金額"
                type="number"
                suffix="万円"
                defaultValue={yenToMan(input.loanAmount)}
                onBlur={(e) => set("loanAmount", manToYen(e.target.value))}
              />
              <Input
                label="年利"
                type="number"
                suffix="%"
                step="0.01"
                defaultValue={(input.annualLoanRate * 100).toFixed(2)}
                onBlur={(e) => set("annualLoanRate", (parseFloat(e.target.value) || 0) / 100)}
              />
              <Input
                label="返済期間"
                type="number"
                suffix="年"
                defaultValue={input.loanYears}
                onBlur={(e) => set("loanYears", parseInt(e.target.value) || 35)}
              />
            </div>
          </div>

          {/* 出口戦略 */}
          <div className="border-t pt-3">
            <p className="mb-3 text-sm font-medium text-muted-foreground">出口戦略</p>
            <div className="grid grid-cols-2 gap-3">
              <Input
                label="売却予定年数"
                type="number"
                suffix="年後"
                defaultValue={input.holdingYears}
                onBlur={(e) => set("holdingYears", parseInt(e.target.value) || 10)}
              />
              <Input
                label="売却時目標利回り"
                type="number"
                suffix="%"
                step="0.5"
                defaultValue={(input.exitYieldTarget * 100).toFixed(1)}
                onBlur={(e) => set("exitYieldTarget", (parseFloat(e.target.value) || 6) / 100)}
              />
            </div>
          </div>

          {/* 詳細設定 */}
          <button
            className="text-xs text-primary underline-offset-2 hover:underline"
            onClick={() => setShowAdvanced((p) => !p)}
          >
            {showAdvanced ? "▲ 詳細設定を閉じる" : "▼ 詳細設定（諸経費率・所得税率など）"}
          </button>

          {showAdvanced && (
            <div className="space-y-3 rounded-md bg-muted/50 p-3">
              <div className="grid grid-cols-2 gap-3">
                <Input
                  label="諸経費率"
                  type="number"
                  suffix="%"
                  step="0.5"
                  defaultValue={(input.miscExpenseRate * 100).toFixed(1)}
                  onBlur={(e) => set("miscExpenseRate", (parseFloat(e.target.value) || 7) / 100)}
                />
                <Input
                  label="空室率"
                  type="number"
                  suffix="%"
                  step="1"
                  defaultValue={(input.vacancyRate * 100).toFixed(0)}
                  onBlur={(e) => set("vacancyRate", (parseFloat(e.target.value) || 5) / 100)}
                />
                <Input
                  label="運営経費率"
                  type="number"
                  suffix="%"
                  step="1"
                  defaultValue={(input.expenseRate * 100).toFixed(0)}
                  onBlur={(e) => set("expenseRate", (parseFloat(e.target.value) || 20) / 100)}
                />
                <Input
                  label="所得税率"
                  type="number"
                  suffix="%"
                  step="1"
                  defaultValue={(input.incomeTaxRate * 100).toFixed(0)}
                  onBlur={(e) => set("incomeTaxRate", (parseFloat(e.target.value) || 33) / 100)}
                />
              </div>
            </div>
          )}

          {/* ストレステスト */}
          <div className="border-t pt-3 space-y-4">
            <p className="text-sm font-medium text-muted-foreground">ストレステスト</p>
            <Slider
              label="空室率の上昇"
              value={input.vacancyRateDelta}
              min={0}
              max={0.3}
              step={0.01}
              onChange={(v) => set("vacancyRateDelta", v)}
              formatValue={(v) => `+${formatPct(v)}`}
            />
            <Slider
              label="金利の上昇"
              value={input.loanRateDelta}
              min={0}
              max={0.03}
              step={0.001}
              onChange={(v) => set("loanRateDelta", v)}
              formatValue={(v) => `+${formatPct(v)}`}
            />
          </div>

          <Button
            className="w-full"
            size="lg"
            loading={loading}
            onClick={() => onAnalyze(input)}
          >
            <Calculator className="h-5 w-5" />
            シミュレーション実行
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

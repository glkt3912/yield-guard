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
import { Search, Calculator, Info } from "lucide-react";

// 全都道府県のスタティックフォールバック（バックエンド未起動時用）
const STATIC_PREFECTURES = [
  { value: "01", label: "北海道" }, { value: "02", label: "青森県" },
  { value: "03", label: "岩手県" }, { value: "04", label: "宮城県" },
  { value: "05", label: "秋田県" }, { value: "06", label: "山形県" },
  { value: "07", label: "福島県" }, { value: "08", label: "茨城県" },
  { value: "09", label: "栃木県" }, { value: "10", label: "群馬県" },
  { value: "11", label: "埼玉県" }, { value: "12", label: "千葉県" },
  { value: "13", label: "東京都" }, { value: "14", label: "神奈川県" },
  { value: "23", label: "愛知県" }, { value: "26", label: "京都府" },
  { value: "27", label: "大阪府" }, { value: "28", label: "兵庫県" },
  { value: "40", label: "福岡県" }, { value: "47", label: "沖縄県" },
];

const BUILDING_TYPES: { value: BuildingType; label: string }[] = [
  { value: "木造", label: "木造（耐用22年）" },
  { value: "軽量鉄骨(3mm以下)", label: "軽量鉄骨・薄板（耐用19年）" },
  { value: "軽量鉄骨(4mm以下)", label: "軽量鉄骨（耐用27年）" },
  { value: "重量鉄骨", label: "重量鉄骨（耐用34年）" },
  { value: "RC造", label: "RC造（耐用47年）" },
  { value: "SRC造", label: "SRC造（耐用47年）" },
];

interface Props {
  onAnalyze: (input: InvestmentInput) => Promise<void>;
  onFetchLandPrices: (area: string, city: string) => Promise<void>;
  loading: boolean;
}

/** 現在の直近2年分の期間ラベルを生成 */
function getPeriodLabel(): string {
  const year = new Date().getFullYear();
  return `${year - 2}〜${year - 1}年`;
}

export function InvestmentForm({ onAnalyze, onFetchLandPrices, loading }: Props) {
  const [input, setInput] = useState<InvestmentInput>(DEFAULT_INPUT);
  const [area, setArea] = useState("10");
  const [city, setCity] = useState("10201");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [prefectures, setPrefectures] = useState<{ value: string; label: string }[]>(STATIC_PREFECTURES);

  useEffect(() => {
    fetchPrefectures()
      .then((list) =>
        setPrefectures(list.map((p) => ({ value: p.code, label: p.name })))
      )
      .catch(() => {
        // バックエンド未起動時: 全都道府県スタティックリストを維持
      });
  }, []);

  // 制御コンポーネント用ヘルパー
  const setNum = (key: keyof InvestmentInput, value: number) =>
    setInput((prev) => ({ ...prev, [key]: value }));
  const setStr = (key: keyof InvestmentInput, value: string) =>
    setInput((prev) => ({ ...prev, [key]: value }));

  const toMan = (yen: number) => String(Math.round(yen / 10_000));
  const fromMan = (s: string) => (parseFloat(s) || 0) * 10_000;
  const toPct = (rate: number, digits = 2) => (rate * 100).toFixed(digits);
  const fromPct = (s: string) => (parseFloat(s) || 0) / 100;

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
              placeholder="例: 10201（前橋市）"
            />
          </div>
          <p className="text-xs text-muted-foreground">
            {getPeriodLabel()}分の宅地取引実績（国交省公式API）を取得します
          </p>
          <Button variant="outline" className="w-full" loading={loading}
            onClick={() => onFetchLandPrices(area, city)}>
            <Search className="h-4 w-4" />相場データを取得
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
            <Input label="土地取得価格" type="number" suffix="万円"
              value={toMan(input.landPrice)}
              onChange={(e) => setNum("landPrice", fromMan(e.target.value))} />
            <Input label="土地面積" type="number" suffix="m²"
              value={String(input.landArea)}
              onChange={(e) => setNum("landArea", parseFloat(e.target.value) || 0)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Input label="建築費" type="number" suffix="万円"
              value={toMan(input.buildingCost)}
              onChange={(e) => setNum("buildingCost", fromMan(e.target.value))} />
            <Input label="築年数" type="number" suffix="年（0=新築）"
              value={String(input.buildingAge)}
              onChange={(e) => setNum("buildingAge", parseInt(e.target.value) || 0)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Input label="想定月額賃料" type="number" suffix="円"
              value={String(input.monthlyRent)}
              onChange={(e) => setNum("monthlyRent", parseFloat(e.target.value) || 0)} />
            <Select label="建物構造" value={input.buildingType}
              onChange={(e) => setStr("buildingType", e.target.value as BuildingType)}
              options={BUILDING_TYPES} />
          </div>

          {/* ローン条件 */}
          <div className="border-t pt-3">
            <p className="mb-3 text-sm font-medium text-muted-foreground">ローン条件</p>
            <div className="grid grid-cols-3 gap-3">
              <Input label="ローン金額" type="number" suffix="万円"
                value={toMan(input.loanAmount)}
                onChange={(e) => setNum("loanAmount", fromMan(e.target.value))} />
              <Input label="年利" type="number" suffix="%" step="0.01"
                value={toPct(input.annualLoanRate)}
                onChange={(e) => setNum("annualLoanRate", fromPct(e.target.value))} />
              <Input label="返済期間" type="number" suffix="年"
                value={String(input.loanYears)}
                onChange={(e) => setNum("loanYears", parseInt(e.target.value) || 35)} />
            </div>
          </div>

          {/* 出口戦略 */}
          <div className="border-t pt-3">
            <p className="mb-3 text-sm font-medium text-muted-foreground">出口戦略</p>
            <div className="grid grid-cols-2 gap-3">
              <Input label="売却予定年数" type="number" suffix="年後"
                value={String(input.holdingYears)}
                onChange={(e) => setNum("holdingYears", parseInt(e.target.value) || 10)} />
              <Input label="売却時目標利回り（実質）" type="number" suffix="%" step="0.5"
                value={toPct(input.exitYieldTarget, 1)}
                onChange={(e) => setNum("exitYieldTarget", fromPct(e.target.value))} />
            </div>
          </div>

          {/* 詳細設定 */}
          <button className="text-xs text-primary underline-offset-2 hover:underline"
            onClick={() => setShowAdvanced((p) => !p)}>
            {showAdvanced ? "▲ 詳細設定を閉じる" : "▼ 詳細設定（諸経費率・空室率など）"}
          </button>

          {showAdvanced && (
            <div className="space-y-3 rounded-md bg-muted/50 p-3">
              <div className="grid grid-cols-2 gap-3">
                <Input label="諸経費率" type="number" suffix="%" step="0.5"
                  value={toPct(input.miscExpenseRate, 1)}
                  onChange={(e) => setNum("miscExpenseRate", fromPct(e.target.value))} />
                <Input label="空室率" type="number" suffix="%" step="1"
                  value={toPct(input.vacancyRate, 0)}
                  onChange={(e) => setNum("vacancyRate", fromPct(e.target.value))} />
                <Input label="運営経費率※" type="number" suffix="%" step="1"
                  value={toPct(input.expenseRate, 0)}
                  onChange={(e) => setNum("expenseRate", fromPct(e.target.value))} />
                <Input label="所得税率（実効）" type="number" suffix="%" step="1"
                  value={toPct(input.incomeTaxRate, 0)}
                  onChange={(e) => setNum("incomeTaxRate", fromPct(e.target.value))} />
              </div>
              <p className="text-xs text-muted-foreground">
                ※運営経費率はローン利息を含みません（管理費・修繕費・固定資産税・保険等）
              </p>
            </div>
          )}

          {/* ストレステスト */}
          <div className="border-t pt-3 space-y-4">
            <p className="text-sm font-medium text-muted-foreground">ストレステスト</p>
            <Slider label="空室率の上昇" value={input.vacancyRateDelta}
              min={0} max={0.3} step={0.01}
              onChange={(v) => setNum("vacancyRateDelta", v)}
              formatValue={(v) => `+${formatPct(v)}`} />
            <Slider label="金利の上昇" value={input.loanRateDelta}
              min={0} max={0.03} step={0.001}
              onChange={(v) => setNum("loanRateDelta", v)}
              formatValue={(v) => `+${formatPct(v)}`} />
          </div>

          <Button className="w-full" size="lg" loading={loading}
            onClick={() => onAnalyze(input)}>
            <Calculator className="h-5 w-5" />
            シミュレーション実行
          </Button>

          {/* 免責事項 */}
          <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3 text-xs text-yellow-800 space-y-1">
            <p className="flex items-center gap-1 font-semibold">
              <Info className="h-3 w-3" />免責事項
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
    </div>
  );
}

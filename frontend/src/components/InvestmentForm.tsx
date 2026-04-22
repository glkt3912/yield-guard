"use client";
import React, { useState, useEffect, useCallback, useMemo } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import type { InvestmentInput, BuildingType, SimulationMode } from "@/types/investment";
import { DEFAULT_INPUT, QUICK_MODE_DEFAULTS } from "@/types/investment";
import { formatPct } from "@/lib/utils";
import { fetchMunicipalities, type Municipality } from "@/lib/api";
import { Search, Calculator, Info, AlertTriangle, ShieldCheck, Zap, SlidersHorizontal } from "lucide-react";
import { ZONING_TYPES, ZONING_META, type ZoningType } from "@/lib/zoning";

// 全47都道府県（法的に変容しない静的データ）
const PREFECTURES = [
  { value: "01", label: "北海道" }, { value: "02", label: "青森県" },
  { value: "03", label: "岩手県" }, { value: "04", label: "宮城県" },
  { value: "05", label: "秋田県" }, { value: "06", label: "山形県" },
  { value: "07", label: "福島県" }, { value: "08", label: "茨城県" },
  { value: "09", label: "栃木県" }, { value: "10", label: "群馬県" },
  { value: "11", label: "埼玉県" }, { value: "12", label: "千葉県" },
  { value: "13", label: "東京都" }, { value: "14", label: "神奈川県" },
  { value: "15", label: "新潟県" }, { value: "16", label: "富山県" },
  { value: "17", label: "石川県" }, { value: "18", label: "福井県" },
  { value: "19", label: "山梨県" }, { value: "20", label: "長野県" },
  { value: "21", label: "岐阜県" }, { value: "22", label: "静岡県" },
  { value: "23", label: "愛知県" }, { value: "24", label: "三重県" },
  { value: "25", label: "滋賀県" }, { value: "26", label: "京都府" },
  { value: "27", label: "大阪府" }, { value: "28", label: "兵庫県" },
  { value: "29", label: "奈良県" }, { value: "30", label: "和歌山県" },
  { value: "31", label: "鳥取県" }, { value: "32", label: "島根県" },
  { value: "33", label: "岡山県" }, { value: "34", label: "広島県" },
  { value: "35", label: "山口県" }, { value: "36", label: "徳島県" },
  { value: "37", label: "香川県" }, { value: "38", label: "愛媛県" },
  { value: "39", label: "高知県" }, { value: "40", label: "福岡県" },
  { value: "41", label: "佐賀県" }, { value: "42", label: "長崎県" },
  { value: "43", label: "熊本県" }, { value: "44", label: "大分県" },
  { value: "45", label: "宮崎県" }, { value: "46", label: "鹿児島県" },
  { value: "47", label: "沖縄県" },
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
  onFetchLandPrices: (area: string, city: string, lat?: number, lng?: number) => Promise<void>;
  loading: boolean;
  simulationMode: SimulationMode;
  onModeChange: (mode: SimulationMode) => void;
}

/** 現在の直近2年分の期間ラベルを生成 */
function getPeriodLabel(): string {
  const year = new Date().getFullYear();
  return `${year - 2}〜${year - 1}年`;
}

type FormErrors = Partial<Record<keyof InvestmentInput, string>>;

function validateFull(input: InvestmentInput): FormErrors {
  const e: FormErrors = {};
  if (input.landPrice <= 0 || input.landPrice > 10_000_000_000)
    e.landPrice = "1〜100億円の範囲で入力してください";
  if (input.buildingCost <= 0 || input.buildingCost > 10_000_000_000)
    e.buildingCost = "1〜100億円の範囲で入力してください";
  if (input.monthlyRent <= 0)
    e.monthlyRent = "正の値を入力してください";
  if (input.vacancyRate < 0 || input.vacancyRate >= 1.0)
    e.vacancyRate = "0〜99%の範囲で入力してください";
  if (input.loanAmount < 0)
    e.loanAmount = "0以上の値を入力してください";
  if (input.annualLoanRate < 0 || input.annualLoanRate > 0.3)
    e.annualLoanRate = "0〜30%の範囲で入力してください";
  if (input.loanYears < 0 || input.loanYears > 50)
    e.loanYears = "0〜50年の範囲で入力してください";
  if (input.miscExpenseRate < 0 || input.miscExpenseRate > 0.5)
    e.miscExpenseRate = "0〜50%の範囲で入力してください";
  if (input.expenseRate < 0 || input.expenseRate > 0.9)
    e.expenseRate = "0〜90%の範囲で入力してください";
  if (input.incomeTaxRate < 0 || input.incomeTaxRate > 0.6)
    e.incomeTaxRate = "0〜60%の範囲で入力してください";
  if (input.exitYieldTarget <= 0 || input.exitYieldTarget > 0.5)
    e.exitYieldTarget = "0%超〜50%の範囲で入力してください";
  if (input.holdingYears < 0 || input.holdingYears > 50)
    e.holdingYears = "0〜50年の範囲で入力してください";
  return e;
}

function validateQuick(input: InvestmentInput): FormErrors {
  const e: FormErrors = {};
  if (input.landPrice <= 0 || input.landPrice > 10_000_000_000)
    e.landPrice = "1〜100億円の範囲で入力してください";
  if (input.buildingCost <= 0 || input.buildingCost > 10_000_000_000)
    e.buildingCost = "1〜100億円の範囲で入力してください";
  if (input.monthlyRent <= 0)
    e.monthlyRent = "正の値を入力してください";
  if (input.loanAmount < 0)
    e.loanAmount = "0以上の値を入力してください";
  if (input.loanYears < 0 || input.loanYears > 50)
    e.loanYears = "0〜50年の範囲で入力してください";
  if (input.holdingYears < 0 || input.holdingYears > 50)
    e.holdingYears = "0〜50年の範囲で入力してください";
  return e;
}

export function InvestmentForm({ onAnalyze, onFetchLandPrices, loading, simulationMode, onModeChange }: Props) {
  const [input, setInput] = useState<InvestmentInput>(DEFAULT_INPUT);
  const [area, setArea] = useState("10");
  const [city, setCity] = useState("");
  const [propertyLat, setPropertyLat] = useState("");
  const [propertyLng, setPropertyLng] = useState("");
  const [municipalities, setMunicipalities] = useState<Municipality[]>([]);
  const [muniLoading, setMuniLoading] = useState(false);
  const [muniFilter, setMuniFilter] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [errors, setErrors] = useState<FormErrors>({});
  const [zoningType, setZoningType] = useState<ZoningType>("");

  // 按分ヘルパーの状態
  const [showBuildingHelper, setShowBuildingHelper] = useState(false);
  const [taxAmount, setTaxAmount] = useState("");
  const [landAssess, setLandAssess] = useState("");
  const [buildingAssess, setBuildingAssess] = useState("");
  const [totalPrice, setTotalPrice] = useState("");

  // 按分計算ユーティリティ
  const calcFromTax = (taxMan: number) => taxMan / 0.1;
  const calcFromAssessment = (totalMan: number, landA: number, buildingA: number) =>
    landA + buildingA > 0 ? totalMan * (buildingA / (landA + buildingA)) : 0;

  const isQuick = simulationMode === "quick";

  const filteredMunicipalities = useMemo(
    () => muniFilter.trim()
      ? municipalities.filter((m) => m.name.includes(muniFilter.trim()))
      : municipalities,
    [municipalities, muniFilter]
  );

  const loadMunicipalities = useCallback(async (areaCode: string) => {
    setMuniLoading(true);
    setCity("");
    setMuniFilter("");
    try {
      const data = await fetchMunicipalities(areaCode);
      setMunicipalities(data);
      if (data.length > 0) setCity(data[0].id);
    } catch {
      setMunicipalities([]);
    } finally {
      setMuniLoading(false);
    }
  }, []);

  // 初回マウント時に初期都道府県の市区町村を取得
  useEffect(() => {
    loadMunicipalities("10");
  }, [loadMunicipalities]);

  // フィルタ結果が1件になったら自動選択
  useEffect(() => {
    if (filteredMunicipalities.length === 1) {
      setCity(filteredMunicipalities[0].id);
    }
  }, [filteredMunicipalities]);

  const handleAreaChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const code = e.target.value;
    setArea(code);
    loadMunicipalities(code);
  };

  // 制御コンポーネント用ヘルパー
  const setNum = (key: keyof InvestmentInput, value: number) => {
    setInput((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => {
      if (!prev[key]) return prev;
      const next = { ...prev };
      delete next[key];
      return next;
    });
  };
  const setStr = (key: keyof InvestmentInput, value: string) =>
    setInput((prev) => ({ ...prev, [key]: value }));

  const toMan = (yen: number) => String(Math.round(yen / 10_000));
  const fromMan = (s: string) => (parseFloat(s) || 0) * 10_000;
  const toPct = (rate: number, digits = 2) => (rate * 100).toFixed(digits);
  const fromPct = (s: string) => (parseFloat(s) || 0) / 100;

  const handleAnalyze = () => {
    const errs = isQuick ? validateQuick(input) : validateFull(input);
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;
    const payload: InvestmentInput = isQuick
      ? { ...input, ...QUICK_MODE_DEFAULTS } as InvestmentInput
      : input;
    onAnalyze(payload);
  };

  return (
    <div className="space-y-4">
      {/* モード切替 */}
      <div className="flex rounded-lg border bg-muted p-1 gap-1" role="group" aria-label="シミュレーションモード">
        <button
          type="button"
          role="radio"
          aria-checked={simulationMode === "quick"}
          onClick={() => onModeChange("quick")}
          className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
            simulationMode === "quick"
              ? "bg-primary text-white shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <Zap className="h-3.5 w-3.5" />
          クイック
        </button>
        <button
          type="button"
          role="radio"
          aria-checked={simulationMode === "full"}
          onClick={() => onModeChange("full")}
          className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
            simulationMode === "full"
              ? "bg-primary text-white shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <SlidersHorizontal className="h-3.5 w-3.5" />
          詳細
        </button>
      </div>

      {isQuick && (
        <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-800">
          <span className="font-semibold">クイックモード:</span>{" "}
          6項目のみ入力してください。空室率5%・運営経費率20%・建物構造:木造 をデフォルトとして計算します。
        </div>
      )}

      {/* 相場データ取得 */}
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
                onChange={(e) => setCity(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="">（全市区町村）</option>
                {muniLoading
                  ? <option disabled>読み込み中...</option>
                  : filteredMunicipalities.length === 0 && muniFilter.trim()
                  ? <option disabled>該当なし</option>
                  : filteredMunicipalities.map((m) => (
                      <option key={m.id} value={m.id}>{m.name}</option>
                    ))
                }
              </select>
              {muniFilter.trim() && filteredMunicipalities.length > 0 && (
                <p className="text-xs text-muted-foreground">{filteredMunicipalities.length}件該当</p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-foreground">緯度（任意）</label>
              <input
                type="number"
                placeholder="例: 35.6762"
                step="0.0001"
                value={propertyLat}
                onChange={(e) => setPropertyLat(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                aria-label="物件の緯度"
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-foreground">経度（任意）</label>
              <input
                type="number"
                placeholder="例: 139.6503"
                step="0.0001"
                value={propertyLng}
                onChange={(e) => setPropertyLng(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring placeholder:text-muted-foreground"
                aria-label="物件の経度"
              />
            </div>
          </div>
          <p className="text-xs text-muted-foreground">
            {getPeriodLabel()}分の宅地取引実績（国交省公式API）を取得します。緯度・経度を入力すると周辺駅の需要スコアも取得します
          </p>
          <Button variant="outline" className="w-full" loading={loading}
            onClick={() => {
              const lat = parseFloat(propertyLat);
              const lng = parseFloat(propertyLng);
              const hasCoords = !isNaN(lat) && !isNaN(lng);
              onFetchLandPrices(area, city, hasCoords ? lat : undefined, hasCoords ? lng : undefined);
            }}>
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
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Input label="土地取得価格" type="number" suffix="万円"
              value={toMan(input.landPrice)}
              onChange={(e) => setNum("landPrice", fromMan(e.target.value))}
              error={errors.landPrice} />
            {!isQuick && (
              <Input label="土地面積" type="number" suffix="m²"
                value={String(input.landArea)}
                onChange={(e) => setNum("landArea", parseFloat(e.target.value) || 0)} />
            )}
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <Input label="建物価格" type="number" suffix="万円"
                value={toMan(input.buildingCost)}
                onChange={(e) => setNum("buildingCost", fromMan(e.target.value))}
                error={errors.buildingCost} />
              <p className="text-xs text-muted-foreground mt-1">新築は建設費、中古は売買契約書記載の建物価格（消費税の課税対象部分）を入力</p>

              {/* 按分ヘルパー */}
              <button
                type="button"
                className="mt-2 text-xs text-primary underline-offset-2 hover:underline"
                onClick={() => setShowBuildingHelper((p) => !p)}
              >
                {showBuildingHelper ? "▲ 按分ヘルパーを閉じる" : "▼ 建物価格がわからない場合（按分ヘルパー）"}
              </button>

              {showBuildingHelper && (
                <div className="mt-2 rounded-md bg-muted/50 p-3 space-y-3 text-sm">
                  {/* 方法1: 消費税から計算 */}
                  <div>
                    <p className="font-medium text-xs mb-1">① 消費税額から計算（最も確実）</p>
                    <div className="flex gap-2 items-end">
                      <div className="flex-1">
                        <Input
                          label="消費税額"
                          type="number"
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
                    <p className="text-xs text-muted-foreground mt-1">建物価格 = 消費税額 ÷ 0.1</p>
                  </div>

                  {/* 方法2: 固定資産税評価額から計算 */}
                  <div>
                    <p className="font-medium text-xs mb-1">② 固定資産税評価額の比率で按分</p>
                    <div className="grid grid-cols-3 gap-2">
                      <Input
                        label="土地評価額"
                        type="number"
                        suffix="万円"
                        value={landAssess}
                        onChange={(e) => setLandAssess(e.target.value)}
                      />
                      <Input
                        label="建物評価額"
                        type="number"
                        suffix="万円"
                        value={buildingAssess}
                        onChange={(e) => setBuildingAssess(e.target.value)}
                      />
                      <Input
                        label="購入総額"
                        type="number"
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
                    <p className="text-xs text-muted-foreground mt-1">建物価格 = 総額 × 建物評価 ÷（土地+建物評価）</p>
                  </div>
                </div>
              )}
            </div>
            {!isQuick && (
              <Input label="築年数" type="number" suffix="年（0=新築）"
                value={String(input.buildingAge)}
                onChange={(e) => setNum("buildingAge", parseInt(e.target.value) || 0)} />
            )}
            {!isQuick && (
              <Input label="最寄り駅徒歩" type="number" suffix="分（0=未入力）"
                value={String(input.stationMinutes)}
                onChange={(e) => setNum("stationMinutes", parseInt(e.target.value) || 0)} />
            )}
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Input label="想定月額賃料" type="number" suffix="円"
              value={String(input.monthlyRent)}
              onChange={(e) => setNum("monthlyRent", parseFloat(e.target.value) || 0)}
              error={errors.monthlyRent} />
            {!isQuick && (
              <Select label="建物構造" value={input.buildingType}
                onChange={(e) => setStr("buildingType", e.target.value as BuildingType)}
                options={BUILDING_TYPES} />
            )}
          </div>

          {/* ローン条件 */}
          <div className="border-t pt-3">
            <p className="mb-3 text-sm font-medium text-muted-foreground">ローン条件</p>
            <div className={`grid grid-cols-1 ${isQuick ? "sm:grid-cols-2" : "sm:grid-cols-3"} gap-3`}>
              <Input label="ローン金額" type="number" suffix="万円"
                value={toMan(input.loanAmount)}
                onChange={(e) => setNum("loanAmount", fromMan(e.target.value))}
                error={errors.loanAmount} />
              {!isQuick && (
                <Input label="年利" type="number" suffix="%" step="0.01"
                  value={toPct(input.annualLoanRate)}
                  onChange={(e) => setNum("annualLoanRate", fromPct(e.target.value))}
                  error={errors.annualLoanRate} />
              )}
              <Input label="返済期間" type="number" suffix="年"
                value={String(input.loanYears)}
                onChange={(e) => setNum("loanYears", parseInt(e.target.value) || 35)}
                error={errors.loanYears} />
            </div>
          </div>

          {/* 出口戦略 */}
          <div className="border-t pt-3">
            <p className="mb-3 text-sm font-medium text-muted-foreground">出口戦略</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <Input label="売却予定年数" type="number" suffix="年後"
                value={String(input.holdingYears)}
                onChange={(e) => setNum("holdingYears", parseInt(e.target.value) || 10)}
                error={errors.holdingYears} />
              {!isQuick && (
                <Input label="売却時目標利回り（実質）" type="number" suffix="%" step="0.5"
                  value={toPct(input.exitYieldTarget, 1)}
                  onChange={(e) => setNum("exitYieldTarget", fromPct(e.target.value))}
                  error={errors.exitYieldTarget} />
              )}
            </div>
          </div>

          {/* 用途地域 */}
          {!isQuick && (
            <div className="border-t pt-3 space-y-2">
              <Select
                label="用途地域（任意）"
                value={zoningType}
                onChange={(e) => setZoningType(e.target.value as ZoningType)}
                options={[
                  { value: "", label: "（未選択）" },
                  ...ZONING_TYPES.map((z) => ({ value: z, label: z })),
                ]}
              />
              {zoningType && (() => {
                const meta = ZONING_META[zoningType];
                if (!meta) return null;
                if (meta.riskLevel === 0) return (
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
                const labels: Record<number, string> = { 1: "低リスク", 2: "中リスク", 3: "高リスク" };
                return (
                  <div className={`flex items-start gap-2 rounded-md border p-2 text-xs ${styles[meta.riskLevel]}`}>
                    <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                    <div>
                      <span className="font-semibold">{labels[meta.riskLevel]}: </span>
                      <span>{meta.riskMessage}</span>
                    </div>
                  </div>
                );
              })()}
            </div>
          )}

          {/* 詳細設定 */}
          {!isQuick && (
            <>
              <button className="text-xs text-primary underline-offset-2 hover:underline"
                onClick={() => setShowAdvanced((p) => !p)}>
                {showAdvanced ? "▲ 詳細設定を閉じる" : "▼ 詳細設定（諸経費率・空室率など）"}
              </button>

              {showAdvanced && (
                <div className="space-y-3 rounded-md bg-muted/50 p-3">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <Input label="諸経費率" type="number" suffix="%" step="0.5"
                      value={toPct(input.miscExpenseRate, 1)}
                      onChange={(e) => setNum("miscExpenseRate", fromPct(e.target.value))}
                      error={errors.miscExpenseRate} />
                    <Input label="現況空室率" type="number" suffix="%" step="1"
                      value={toPct(input.actualVacancyRate, 0)}
                      onChange={(e) => setNum("actualVacancyRate", fromPct(e.target.value))} />
                    <Input label="想定空室率（長期）" type="number" suffix="%" step="1"
                      value={toPct(input.vacancyRate, 0)}
                      onChange={(e) => setNum("vacancyRate", fromPct(e.target.value))}
                      error={errors.vacancyRate} />
                    <Input label="運営経費率※" type="number" suffix="%" step="1"
                      value={toPct(input.expenseRate, 0)}
                      onChange={(e) => setNum("expenseRate", fromPct(e.target.value))}
                      error={errors.expenseRate} />
                    <Input label="所得税率（実効）" type="number" suffix="%" step="1"
                      value={toPct(input.incomeTaxRate, 0)}
                      onChange={(e) => setNum("incomeTaxRate", fromPct(e.target.value))}
                      error={errors.incomeTaxRate} />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    ※運営経費率はローン利息を含みません（管理費・修繕費・固定資産税・保険等）
                  </p>
                </div>
              )}
            </>
          )}

          {/* ストレステスト */}
          {!isQuick && (
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
          )}

          <Button className="w-full" size="lg" loading={loading}
            onClick={handleAnalyze}>
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

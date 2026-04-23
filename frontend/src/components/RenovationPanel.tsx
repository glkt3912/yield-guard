"use client";

import { useState, useMemo } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ReferenceLine,
  ResponsiveContainer,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { analyzeRenovation } from "@/lib/api";
import type { RenovationInput, RenovationItem, RenovationResult } from "@/types/investment";
import { DEFAULT_RENOVATION_INPUT } from "@/types/investment";
import { formatMan, formatPct } from "@/lib/utils";

const cellInput = "rounded border border-input bg-background px-2 py-0.5 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring";

function emptyItem(): RenovationItem {
  return {
    name: "",
    cost: 0,
    expectedMonthlyRentIncrease: 0,
    isSelfWork: false,
    selfLaborHours: 0,
  };
}

export default function RenovationPanel() {
  const [globals, setGlobals] = useState<Omit<RenovationInput, "items">>({
    propertyPrice: DEFAULT_RENOVATION_INPUT.propertyPrice,
    annualBaseRent: DEFAULT_RENOVATION_INPUT.annualBaseRent,
    annualExpenses: DEFAULT_RENOVATION_INPUT.annualExpenses,
    effectiveTaxRate: DEFAULT_RENOVATION_INPUT.effectiveTaxRate,
    selfLaborRatePerHour: DEFAULT_RENOVATION_INPUT.selfLaborRatePerHour,
  });
  const [items, setItems] = useState<RenovationItem[]>([emptyItem()]);
  const [result, setResult] = useState<RenovationResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function updateGlobal<K extends keyof typeof globals>(key: K, value: (typeof globals)[K]) {
    setGlobals((prev) => ({ ...prev, [key]: value }));
  }

  function updateItem(index: number, field: keyof RenovationItem, value: string | number | boolean) {
    setItems((prev) =>
      prev.map((item, i) => {
        if (i !== index) return item;
        const updated = { ...item, [field]: value };
        if (field === "isSelfWork" && !value) {
          updated.selfLaborHours = 0;
        }
        return updated;
      })
    );
  }

  function addItem() {
    setItems((prev) => [...prev, emptyItem()]);
  }

  function removeItem(index: number) {
    setItems((prev) => prev.filter((_, i) => i !== index));
  }

  async function handleSubmit() {
    if (items.length === 0) {
      setError("工事項目を1件以上追加してください");
      return;
    }
    if (items.some((item) => item.cost <= 0)) {
      setError("工事費は正の値を入力してください");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await analyzeRenovation({ ...globals, items });
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : "エラーが発生しました");
    } finally {
      setLoading(false);
    }
  }

  const recoveryChartData = useMemo(() => {
    if (!result || !result.isRecoverable) return [];
    const maxYears = Math.min(Math.ceil(result.recoveryYears) + 2, 50);
    return Array.from({ length: maxYears + 1 }, (_, i) => ({
      year: `${i}年`,
      累積賃料増加額: Math.round((result.annualRentIncrease * i) / 10_000),
      リフォーム費用: Math.round(result.totalRenovationCost / 10_000),
    }));
  }, [result]);

  const recoveryYearLabel =
    result?.isRecoverable ? `${Math.ceil(result.recoveryYears)}年` : null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">修繕費回収期間シミュレーション</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Global inputs */}
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <Input
            label="物件取得価格"
            type="number"
            suffix="万円"
            value={globals.propertyPrice / 10_000}
            onChange={(e) => updateGlobal("propertyPrice", Number(e.target.value) * 10_000)}
          />
          <Input
            label="リフォーム前年間家賃"
            type="number"
            suffix="万円"
            value={globals.annualBaseRent / 10_000}
            onChange={(e) => updateGlobal("annualBaseRent", Number(e.target.value) * 10_000)}
          />
          <Input
            label="年間経費"
            type="number"
            suffix="万円"
            value={globals.annualExpenses / 10_000}
            onChange={(e) => updateGlobal("annualExpenses", Number(e.target.value) * 10_000)}
          />
          <Input
            label="実効税率"
            type="number"
            suffix="%"
            step="1"
            min="0"
            max="100"
            value={Math.round(globals.effectiveTaxRate * 100)}
            onChange={(e) => updateGlobal("effectiveTaxRate", Number(e.target.value) / 100)}
          />
          <Input
            label="セルフリフォーム時給"
            type="number"
            suffix="円"
            value={globals.selfLaborRatePerHour}
            onChange={(e) => updateGlobal("selfLaborRatePerHour", Number(e.target.value))}
          />
        </div>

        {/* Items table */}
        <div>
          <p className="text-sm font-semibold mb-2">工事項目</p>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-xs text-muted-foreground">
                  <th className="py-1 pr-2 text-left">部位名</th>
                  <th className="py-1 pr-2 text-right">工事費（万円）</th>
                  <th className="py-1 pr-2 text-right">賃料アップ（月額円）</th>
                  <th className="py-1 pr-2 text-center">セルフ</th>
                  <th className="py-1 pr-2 text-right">工数（h）</th>
                  <th className="py-1" />
                </tr>
              </thead>
              <tbody>
                {items.map((item, i) => (
                  <tr key={i} className="border-b last:border-0">
                    <td className="py-1 pr-2">
                      <input
                        type="text"
                        placeholder="例：内装"
                        className={`w-full ${cellInput}`}
                        value={item.name}
                        onChange={(e) => updateItem(i, "name", e.target.value)}
                      />
                    </td>
                    <td className="py-1 pr-2">
                      <input
                        type="number"
                        min="0"
                        className={`w-20 text-right ${cellInput}`}
                        value={item.cost / 10_000}
                        onChange={(e) => updateItem(i, "cost", Number(e.target.value) * 10_000)}
                      />
                    </td>
                    <td className="py-1 pr-2">
                      <input
                        type="number"
                        min="0"
                        className={`w-24 text-right ${cellInput}`}
                        value={item.expectedMonthlyRentIncrease}
                        onChange={(e) => updateItem(i, "expectedMonthlyRentIncrease", Number(e.target.value))}
                      />
                    </td>
                    <td className="py-1 pr-2 text-center">
                      <input
                        type="checkbox"
                        checked={item.isSelfWork}
                        onChange={(e) => updateItem(i, "isSelfWork", e.target.checked)}
                      />
                    </td>
                    <td className="py-1 pr-2">
                      {item.isSelfWork && (
                        <input
                          type="number"
                          min="0"
                          className={`w-16 text-right ${cellInput}`}
                          value={item.selfLaborHours}
                          onChange={(e) => updateItem(i, "selfLaborHours", Number(e.target.value))}
                        />
                      )}
                    </td>
                    <td className="py-1">
                      {items.length > 1 && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          aria-label={`工事項目${i + 1}を削除`}
                          onClick={() => removeItem(i)}
                          className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
                        >
                          ×
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            aria-label="工事項目を追加"
            onClick={addItem}
            className="mt-2 text-xs text-primary"
          >
            ＋ 行を追加
          </Button>
        </div>

        {error && <p className="text-sm text-destructive">{error}</p>}

        <Button onClick={handleSubmit} disabled={loading} className="w-full sm:w-auto">
          {loading ? "計算中..." : "リフォーム分析を実行"}
        </Button>

        {/* Results */}
        {result && (
          <div className="space-y-4">
            {/* Summary cards */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div className="rounded-md border p-3 text-center">
                <p className="text-xs text-muted-foreground">修繕費回収期間</p>
                <p className="text-lg font-bold">
                  {result.isRecoverable
                    ? `${result.recoveryYears.toFixed(1)}年`
                    : "回収不可"}
                </p>
              </div>
              <div className="rounded-md border p-3 text-center">
                <p className="text-xs text-muted-foreground">節税効果</p>
                <p className="text-lg font-bold text-green-600">{formatMan(result.taxSavings)}</p>
              </div>
              <div className="rounded-md border p-3 text-center">
                <p className="text-xs text-muted-foreground">仮想人件費</p>
                <p className="text-lg font-bold">{formatMan(result.virtualLaborCost)}</p>
              </div>
              <div className="rounded-md border p-3 text-center">
                <p className="text-xs text-muted-foreground">実質利回り</p>
                <p className={`text-lg font-bold ${result.actualYield >= 0.08 ? "text-green-600" : "text-yellow-600"}`}>
                  {formatPct(result.actualYield)}
                </p>
              </div>
            </div>

            {/* Classification table */}
            <div>
              <p className="text-sm font-semibold mb-2">工事分類結果</p>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-xs text-muted-foreground">
                    <th className="py-1 pr-3 text-left">部位名</th>
                    <th className="py-1 pr-3 text-right">工事費</th>
                    <th className="py-1 pr-3 text-right">月額賃料アップ</th>
                    <th className="py-1 text-center">分類</th>
                  </tr>
                </thead>
                <tbody>
                  {result.classifiedItems.map((item, i) => (
                    <tr key={i} className="border-b last:border-0">
                      <td className="py-1 pr-3">{item.name || `工事${i + 1}`}</td>
                      <td className="py-1 pr-3 text-right">{formatMan(item.cost)}</td>
                      <td className="py-1 pr-3 text-right">
                        {item.expectedMonthlyRentIncrease.toLocaleString()}円
                      </td>
                      <td className="py-1 text-center">
                        <span
                          className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${
                            item.isCapitalExpenditure
                              ? "bg-amber-100 text-amber-800"
                              : "bg-blue-100 text-blue-800"
                          }`}
                        >
                          {item.isCapitalExpenditure ? "資本的支出" : "修繕費"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="mt-2 flex gap-4 text-xs text-muted-foreground">
                <span>資本的支出合計: {formatMan(result.capitalExpenditures)}</span>
                <span>修繕費合計: {formatMan(result.repairExpenses)}</span>
              </div>
            </div>

            {/* Recovery timeline chart */}
            {result.isRecoverable && (
              <div>
                <p className="text-sm font-semibold mb-2">回収タイムライン</p>
                <ResponsiveContainer width="100%" height={220}>
                  <LineChart data={recoveryChartData} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="year" tick={{ fontSize: 11 }} />
                    <YAxis
                      tick={{ fontSize: 11 }}
                      tickFormatter={(v) => `${v}万`}
                    />
                    <Tooltip formatter={(value: number) => [`${value}万円`]} />
                    <Legend />
                    <Line
                      type="monotone"
                      dataKey="累積賃料増加額"
                      stroke="#2563eb"
                      dot={false}
                    />
                    <Line
                      type="monotone"
                      dataKey="リフォーム費用"
                      stroke="#dc2626"
                      strokeDasharray="5 5"
                      dot={false}
                    />
                    {recoveryYearLabel && (
                      <ReferenceLine
                        x={recoveryYearLabel}
                        stroke="#16a34a"
                        strokeDasharray="4 4"
                        label={{ value: "回収", fill: "#16a34a", fontSize: 11 }}
                      />
                    )}
                  </LineChart>
                </ResponsiveContainer>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

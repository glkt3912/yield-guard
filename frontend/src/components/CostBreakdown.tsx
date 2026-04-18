"use client";

import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from "recharts";
import { AcquisitionCostBreakdown, InvestmentInput, YearlyResult } from "@/types/investment";

interface Props {
  input: InvestmentInput;
  acquisitionCosts: AcquisitionCostBreakdown;
  yearlyResults: YearlyResult[];
}

const fmt = (n: number) =>
  n >= 10_000_000
    ? `${(n / 10_000_000).toFixed(1)}千万円`
    : `${Math.round(n / 10_000).toLocaleString()}万円`;

const COLORS = [
  "#3b82f6", "#10b981", "#f59e0b", "#ef4444",
  "#8b5cf6", "#06b6d4", "#f97316",
];

export default function CostBreakdown({ input, acquisitionCosts, yearlyResults }: Props) {
  // 初期投資の内訳（miscExpenses は別軸の概算値のため表示しない）
  const initialCostItems = [
    { name: "土地", value: input.landPrice },
    { name: "建物", value: input.buildingCost },
    { name: "仲介手数料", value: acquisitionCosts.brokerageFee },
    { name: "印紙税", value: acquisitionCosts.stampDuty },
    { name: "登録免許税", value: acquisitionCosts.registrationTax },
    { name: "不動産取得税", value: acquisitionCosts.realEstateAcquisitionTax },
    ...(acquisitionCosts.propertyTaxProration > 0
      ? [{ name: "固定資産税日割り", value: acquisitionCosts.propertyTaxProration }]
      : []),
  ].filter((item) => item.value > 0);

  // 選択年（1年目）の年間費用内訳
  const year1 = yearlyResults[0];
  const annualCostItems = year1
    ? [
        { name: "ローン返済", value: year1.annualLoanPayment },
        { name: "運営経費", value: year1.annualExpenses },
        { name: "所得税", value: year1.incomeTax },
      ].filter((item) => item.value > 0)
    : [];

  const totalInitial = initialCostItems.reduce((s, i) => s + i.value, 0);

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">コスト内訳</h2>

      {/* 初期投資内訳 */}
      <div>
        <h3 className="text-sm font-medium text-gray-600 mb-3">
          初期投資の内訳（合計: {fmt(totalInitial)}）
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-center">
          <ResponsiveContainer width="100%" height={220}>
            <PieChart>
              <Pie
                data={initialCostItems}
                cx="50%"
                cy="50%"
                innerRadius={55}
                outerRadius={90}
                dataKey="value"
                nameKey="name"
              >
                {initialCostItems.map((item, idx) => (
                  <Cell key={item.name} fill={COLORS[idx % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip formatter={(v: number) => fmt(v)} />
            </PieChart>
          </ResponsiveContainer>

          <div className="space-y-1.5">
            {initialCostItems.map((item, idx) => (
              <div key={item.name} className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <span
                    className="inline-block w-3 h-3 rounded-sm"
                    style={{ background: COLORS[idx % COLORS.length] }}
                  />
                  <span className="text-gray-700">{item.name}</span>
                </div>
                <span className="font-mono text-gray-900">
                  {fmt(item.value)}
                  <span className="text-xs text-gray-400 ml-1">
                    ({((item.value / totalInitial) * 100).toFixed(1)}%)
                  </span>
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* 諸経費明細テーブル */}
      <div>
        <h3 className="text-sm font-medium text-gray-600 mb-2">取得時諸経費の明細</h3>
        <div className="rounded-lg border border-gray-200 overflow-hidden text-sm">
          <table className="w-full">
            <tbody className="divide-y divide-gray-100">
              {[
                ["仲介手数料（税込）", acquisitionCosts.brokerageFee],
                ["印紙税", acquisitionCosts.stampDuty],
                ["登録免許税", acquisitionCosts.registrationTax],
                ["不動産取得税（概算）", acquisitionCosts.realEstateAcquisitionTax],
                ...(acquisitionCosts.propertyTaxProration > 0
                  ? [["固定資産税日割り精算", acquisitionCosts.propertyTaxProration] as [string, number]]
                  : []),
              ].map(([label, value]) => (
                <tr key={label as string} className="hover:bg-gray-50">
                  <td className="px-4 py-2 text-gray-600">{label as string}</td>
                  <td className="px-4 py-2 text-right font-mono text-gray-900">
                    {fmt(value as number)}
                  </td>
                </tr>
              ))}
              <tr className="bg-gray-50 font-semibold">
                <td className="px-4 py-2 text-gray-800">合計</td>
                <td className="px-4 py-2 text-right font-mono text-gray-900">
                  {fmt(acquisitionCosts.total)}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* 1年目の年間費用内訳 */}
      {annualCostItems.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-600 mb-3">年間費用の内訳（1年目）</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-center">
            <ResponsiveContainer width="100%" height={180}>
              <PieChart>
                <Pie
                  data={annualCostItems}
                  cx="50%"
                  cy="50%"
                  innerRadius={45}
                  outerRadius={75}
                  dataKey="value"
                  nameKey="name"
                >
                  {annualCostItems.map((item, idx) => (
                    <Cell key={item.name} fill={COLORS[idx % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(v: number) => fmt(v)} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>

            <div className="space-y-1.5">
              {(() => {
                const total = annualCostItems.reduce((s, i) => s + i.value, 0);
                return annualCostItems.map((item, idx) => (
                  <div key={item.name} className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <span
                        className="inline-block w-3 h-3 rounded-sm"
                        style={{ background: COLORS[idx % COLORS.length] }}
                      />
                      <span className="text-gray-700">{item.name}</span>
                    </div>
                    <span className="font-mono text-gray-900">
                      {fmt(item.value)}
                      <span className="text-xs text-gray-400 ml-1">
                        ({((item.value / total) * 100).toFixed(1)}%)
                      </span>
                    </span>
                  </div>
                ));
              })()}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

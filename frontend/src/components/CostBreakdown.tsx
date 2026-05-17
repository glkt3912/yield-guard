"use client";

import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from "recharts";
import { AcquisitionCostBreakdown, InvestmentInput, YearlyResult } from "@/types/investment";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useChartHeight } from "@/lib/useChartHeight";
import { chartColors } from "@/lib/chartColors";

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
  chartColors.primary,
  chartColors.success,
  chartColors.warning,
  chartColors.danger,
  chartColors.secondary,
  chartColors.muted,
  chartColors.warning,
];

export default function CostBreakdown({ input, acquisitionCosts, yearlyResults }: Props) {
  const initialPieHeight = useChartHeight(180, 200, 220);
  const annualPieHeight = useChartHeight(160, 170, 180);
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

  const downPayment = input.landPrice + input.buildingCost - input.loanAmount;
  const emergencyReserve = input.monthlyRent * 3;
  const minimumRequired = Math.max(0, downPayment) + acquisitionCosts.total;
  const propertyValue = input.landPrice + input.buildingCost;
  const downPaymentRatio = propertyValue > 0 ? downPayment / propertyValue : 0;

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">コスト内訳</h2>

      {/* 必要自己資金サマリー */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">必要自己資金</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between text-sm font-semibold">
            <span className="text-gray-700">最低必要額</span>
            <span className="font-mono text-lg text-gray-900">{fmt(minimumRequired)}</span>
          </div>
          <div className="border-t pt-3 space-y-2">
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">内訳</p>
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-600">頭金（借入控除後）</span>
              <span className="font-mono text-gray-900">{fmt(downPayment)}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-600">諸費用合計</span>
              <span className="font-mono text-gray-900">{fmt(acquisitionCosts.total)}</span>
            </div>
          </div>
          <div className="border-t pt-3">
            <div className="flex items-center justify-between text-sm text-muted-foreground">
              <span>推奨: 緊急予備費</span>
              <span className="font-mono">+{fmt(emergencyReserve)} （月額家賃×3ヶ月）</span>
            </div>
          </div>
          {downPayment <= 0 && (
            <div className="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">
              頭金がありません。ローン審査が困難になる可能性があります
            </div>
          )}
          {downPayment > 0 && downPaymentRatio < 0.2 && (
            <div className="rounded-md bg-orange-50 border border-orange-200 px-3 py-2 text-sm text-orange-700">
              頭金比率が低め（20%未満）です
            </div>
          )}
        </CardContent>
      </Card>

      {/* 初期投資内訳 */}
      <div>
        <h3 className="text-sm font-medium text-gray-600 mb-3">
          初期投資の内訳（合計: {fmt(totalInitial)}）
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-center">
          <ResponsiveContainer width="100%" height={initialPieHeight}>
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
              <Tooltip formatter={(v) => fmt(Number(v))} />
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
                  ? [
                      ["固定資産税日割り精算", acquisitionCosts.propertyTaxProration] as [
                        string,
                        number,
                      ],
                    ]
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
            <ResponsiveContainer width="100%" height={annualPieHeight}>
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
                <Tooltip formatter={(v) => fmt(Number(v))} />
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

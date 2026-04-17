"use client";
import React from "react";
import {
  ScatterChart,
  Scatter,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
} from "recharts";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { InvestmentInput, LandPriceComparison, UrbanRiskLevel } from "@/types/investment";
import { formatMan, formatTsubo } from "@/lib/utils";
import { MapPin, AlertTriangle, SearchX, Home, Building2, ShieldAlert, ShieldCheck } from "lucide-react";

const SQM_PER_TSUBO = 3.30578;

type LandValueJudgment = "土地値割れ" | "土地値近辺" | "土地値超";

function calcLandValueJudgment(totalPrice: number, estimatedLandValue: number): LandValueJudgment {
  if (totalPrice < estimatedLandValue) return "土地値割れ";
  if (totalPrice <= estimatedLandValue * 1.5) return "土地値近辺";
  return "土地値超";
}

interface Props {
  comparison: LandPriceComparison;
  input?: InvestmentInput | null;
}

const ASSESSMENT_BADGE: Record<string, "success" | "warning" | "danger"> = {
  割安: "success",
  相場: "warning",
  割高: "danger",
};

const RISK_STYLE: Record<UrbanRiskLevel, { border: string; bg: string; text: string; icon: React.ReactNode }> = {
  ERROR: {
    border: "border-red-300",
    bg: "bg-red-50",
    text: "text-red-900",
    icon: <ShieldAlert className="h-4 w-4 shrink-0 text-red-600 mt-0.5" />,
  },
  WARNING: {
    border: "border-yellow-300",
    bg: "bg-yellow-50",
    text: "text-yellow-900",
    icon: <AlertTriangle className="h-4 w-4 shrink-0 text-yellow-600 mt-0.5" />,
  },
  INFO: {
    border: "border-blue-300",
    bg: "bg-blue-50",
    text: "text-blue-900",
    icon: <ShieldCheck className="h-4 w-4 shrink-0 text-blue-600 mt-0.5" />,
  },
};

export function LandPriceAnalysis({ comparison, input }: Props) {
  const { stats, assessment, inputPricePerTsubo, diffFromMedian } = comparison;

  const landValueSection = (() => {
    if (!input || stats.count === 0 || stats.medianTsubo === 0 || comparison.inputArea === 0) return null;
    const estimatedLandValue = stats.medianTsubo * (comparison.inputArea / SQM_PER_TSUBO);
    const totalPrice = input.landPrice + input.buildingCost;
    const judgment = calcLandValueJudgment(totalPrice, estimatedLandValue);
    const diff = estimatedLandValue - totalPrice;
    return { estimatedLandValue, totalPrice, judgment, diff };
  })();

  // 件数 0 件: 統計・グラフ・判定をすべて非表示
  if (stats.count === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5 text-primary" />
            土地価格相場分析
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-start gap-3 rounded-md border border-red-200 bg-red-50 p-4 text-red-800">
            <SearchX className="h-5 w-5 mt-0.5 shrink-0" />
            <div>
              <p className="text-sm font-semibold">取引データが見つかりませんでした</p>
              <p className="text-xs mt-1">エリア・取得期間の条件を変更して再取得してください。</p>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  const scatterData = stats.transactions
    .filter((t) => t.pricePerTsubo > 0 && t.area > 0)
    .map((t) => ({
      area: Math.round(t.area),
      tsubo: Math.round(t.pricePerTsubo / 10_000), // 万円/坪
    }));

  const badgeLabel = stats.lowDataWarning ? `${assessment}（参考値）` : assessment;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5 text-primary" />
            土地価格相場分析
          </CardTitle>
          <Badge variant={ASSESSMENT_BADGE[assessment] ?? "outline"}>{badgeLabel}</Badge>
        </div>
        <p className="text-xs text-muted-foreground">
          取引件数: {stats.count}件 　平均坪単価: {formatTsubo(stats.averageTsubo)} 　中央値: {formatTsubo(stats.medianTsubo)}
        </p>
      </CardHeader>
      <CardContent className="space-y-4">
        {stats.lowDataWarning && (
          <div className="flex items-start gap-3 rounded-md border-2 border-yellow-400 bg-yellow-50 p-4 text-yellow-900">
            <AlertTriangle className="h-5 w-5 mt-0.5 shrink-0 text-yellow-600" />
            <div>
              <p className="text-sm font-semibold">統計データが不足しています</p>
              <p className="text-xs mt-1">
                {stats.warningMessage ?? "取引データが少ないため、統計の信頼性が低い可能性があります。"}
              </p>
              <p className="text-xs mt-0.5">以下の結果は参考値としてご確認ください。</p>
            </div>
          </div>
        )}

        {/* 都市計画リスク警告 */}
        {stats.urbanRisks && stats.urbanRisks.length > 0 && (
          <div className="space-y-2">
            {stats.urbanRisks.map((risk) => {
              const style = RISK_STYLE[risk.level];
              return (
                <div
                  key={risk.code}
                  className={`flex items-start gap-2 rounded-md border ${style.border} ${style.bg} px-3 py-2`}
                >
                  {style.icon}
                  <div>
                    <p className={`text-xs font-semibold ${style.text}`}>{risk.title}</p>
                    <p className={`text-xs mt-0.5 ${style.text}`}>{risk.description}</p>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* 統計サマリー */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {[
            { label: "最安値", value: formatTsubo(stats.minTsubo) },
            { label: "中央値", value: formatTsubo(stats.medianTsubo) },
            { label: "平均値", value: formatTsubo(stats.averageTsubo) },
            { label: "最高値", value: formatTsubo(stats.maxTsubo) },
          ].map(({ label, value }) => (
            <div key={label} className="rounded-md border bg-muted/30 p-2 text-center">
              <p className="text-xs text-muted-foreground">{label}</p>
              <p className="text-sm font-semibold">{value}</p>
            </div>
          ))}
        </div>

        {/* 検討中価格 vs 相場 */}
        {inputPricePerTsubo > 0 && (
          <div
            className={`rounded-md border p-3 ${
              assessment === "割高"
                ? "border-red-200 bg-red-50"
                : assessment === "割安"
                ? "border-green-200 bg-green-50"
                : "border-yellow-200 bg-yellow-50"
            }`}
          >
            <p className="text-sm font-semibold">
              検討中の坪単価: {formatTsubo(inputPricePerTsubo)}
            </p>
            <p className="text-xs mt-1">
              中央値との差:{" "}
              <span className={diffFromMedian > 0 ? "text-red-600 font-bold" : "text-green-600 font-bold"}>
                {diffFromMedian > 0 ? "+" : ""}{formatTsubo(Math.abs(diffFromMedian))}
                （{diffFromMedian > 0 ? "割高" : "割安"}）
              </span>
            </p>
            {stats.lowDataWarning && (
              <p className="text-xs mt-1 text-yellow-700">※ データ件数不足のため参考値</p>
            )}
          </div>
        )}

        {/* 用途地域情報 */}
        {stats.zoning && (
          <div className="rounded-md border border-blue-200 bg-blue-50 p-3">
            <p className="flex items-center gap-1 text-xs font-semibold text-blue-800 mb-2">
              <Building2 className="h-3.5 w-3.5" />
              このエリアの代表的な用途地域（取引データより）
            </p>
            <div className="grid grid-cols-3 gap-2">
              {stats.zoning.cityPlanning && (
                <div className="text-center">
                  <p className="text-xs text-blue-600">用途地域</p>
                  <p className="text-xs font-medium text-blue-900">{stats.zoning.cityPlanning}</p>
                </div>
              )}
              {stats.zoning.buildingCoverage && (
                <div className="text-center">
                  <p className="text-xs text-blue-600">建ぺい率</p>
                  <p className="text-xs font-medium text-blue-900">{stats.zoning.buildingCoverage}</p>
                </div>
              )}
              {stats.zoning.floorAreaRatio && (
                <div className="text-center">
                  <p className="text-xs text-blue-600">容積率</p>
                  <p className="text-xs font-medium text-blue-900">{stats.zoning.floorAreaRatio}</p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* 散布図: 面積 vs 坪単価 */}
        {scatterData.length > 0 && (
          <>
            <p className="text-xs font-medium text-muted-foreground">取引データ分布（面積 vs 坪単価）</p>
            <ResponsiveContainer width="100%" height={220}>
              <ScatterChart margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis
                  dataKey="area"
                  name="面積"
                  unit="m²"
                  tick={{ fontSize: 10 }}
                  label={{ value: "面積(m²)", position: "insideBottom", offset: -4, fontSize: 10 }}
                />
                <YAxis
                  dataKey="tsubo"
                  name="坪単価"
                  unit="万円"
                  tick={{ fontSize: 10 }}
                  width={50}
                />
                <Tooltip
                  formatter={(v: number, name: string) => [
                    name === "tsubo" ? `${v}万円/坪` : `${v}m²`,
                    name === "tsubo" ? "坪単価" : "面積",
                  ]}
                />
                <Scatter data={scatterData} fill="#60a5fa" opacity={0.6} />
                {inputPricePerTsubo > 0 && (
                  <ReferenceLine
                    y={Math.round(inputPricePerTsubo / 10_000)}
                    stroke="#ef4444"
                    strokeWidth={2}
                    strokeDasharray="6 3"
                    label={{
                      value: "検討中",
                      position: "right",
                      fontSize: 10,
                      fill: "#ef4444",
                    }}
                  />
                )}
                <ReferenceLine
                  y={Math.round(stats.medianTsubo / 10_000)}
                  stroke="#f59e0b"
                  strokeWidth={1.5}
                  strokeDasharray="4 4"
                  label={{
                    value: "中央値",
                    position: "right",
                    fontSize: 10,
                    fill: "#f59e0b",
                  }}
                />
              </ScatterChart>
            </ResponsiveContainer>
          </>
        )}
        {/* 土地値割れ判定 */}
        {landValueSection && (() => {
          const { estimatedLandValue, totalPrice, judgment, diff } = landValueSection;
          const colorMap: Record<LandValueJudgment, string> = {
            "土地値割れ": "border-green-300 bg-green-50",
            "土地値近辺": "border-yellow-200 bg-yellow-50",
            "土地値超": "border-red-200 bg-red-50",
          };
          const badgeMap: Record<LandValueJudgment, "success" | "warning" | "danger"> = {
            "土地値割れ": "success",
            "土地値近辺": "warning",
            "土地値超": "danger",
          };
          const descMap: Record<LandValueJudgment, string> = {
            "土地値割れ": "解体しても土地売却で回収可能。安全余白あり",
            "土地値近辺": "建物価値を含むと相場水準。築古注意",
            "土地値超": "建物込み価格。建物価値が失われると含み損リスク",
          };
          return (
            <div className={`rounded-md border p-3 ${colorMap[judgment]}`}>
              <div className="flex items-center justify-between mb-2">
                <p className="flex items-center gap-1 text-sm font-semibold">
                  <Home className="h-4 w-4" />
                  土地値割れ判定
                </p>
                <Badge variant={badgeMap[judgment]}>{judgment}</Badge>
              </div>
              <dl className="space-y-1 text-sm">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">土地概算価値（中央値×面積）</dt>
                  <dd className="font-medium">{formatMan(estimatedLandValue)}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">総取得価格（土地+建物）</dt>
                  <dd className="font-medium">{formatMan(totalPrice)}</dd>
                </div>
                <div className="flex justify-between border-t pt-1">
                  <dt className="text-muted-foreground">差額（概算価値 − 総取得価格）</dt>
                  <dd className={`font-bold ${diff >= 0 ? "text-green-700" : "text-red-700"}`}>
                    {diff >= 0 ? "+" : ""}{formatMan(diff)}
                  </dd>
                </div>
              </dl>
              <p className="mt-2 text-xs text-muted-foreground">{descMap[judgment]}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                ※相場データは宅地取引の中央値を使用。用途地域により実際の価値は異なります。
              </p>
            </div>
          );
        })()}
      </CardContent>
    </Card>
  );
}

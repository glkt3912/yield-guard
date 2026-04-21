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
import type { InvestmentInput, LandPriceComparison, UrbanRisk, UrbanRiskLevel, TheoreticalPriceResult, StationRidershipResult, PopulationForecastResult, AppraisalComparisonResult } from "@/types/investment";
import { formatMan, formatTsubo } from "@/lib/utils";
import { MapPin, AlertTriangle, SearchX, Home, Building2, ShieldAlert, ShieldCheck, TrendingUp, Users } from "lucide-react";

const SQM_PER_TSUBO = 3.30578;

type LandValueJudgment = "土地値割れ" | "土地値近辺" | "土地値超";

function calcLandValueJudgment(totalPrice: number, estimatedLandValue: number): LandValueJudgment {
  if (totalPrice < estimatedLandValue) return "土地値割れ";
  if (totalPrice <= estimatedLandValue * 1.5) return "土地値近辺";
  return "土地値超";
}

const POPULATION_TREND_STYLE: Record<string, { border: string; bg: string; text: string }> = {
  "増加":         { border: "border-green-300",  bg: "bg-green-50",  text: "text-green-900" },
  "現状維持":     { border: "border-yellow-300", bg: "bg-yellow-50", text: "text-yellow-900" },
  "緩やかな減少": { border: "border-orange-300", bg: "bg-orange-50", text: "text-orange-900" },
  "急激な減少":   { border: "border-red-300",    bg: "bg-red-50",    text: "text-red-900" },
};

interface Props {
  comparison: LandPriceComparison;
  input?: InvestmentInput | null;
  theoreticalPrice?: TheoreticalPriceResult | null;
  stationRidership?: StationRidershipResult[] | null;
  populationForecast?: PopulationForecastResult | null;
  landAppraisal?: AppraisalComparisonResult | null;
  externalUrbanRisks?: UrbanRisk[] | null;
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

const RIDERSHIP_SCORE_LABEL: Record<string, { label: string; color: string }> = {
  A: { label: "A（超大型駅）", color: "text-purple-700" },
  B: { label: "B（大型駅）", color: "text-blue-700" },
  C: { label: "C（中型駅）", color: "text-green-700" },
  D: { label: "D（小型駅）", color: "text-yellow-700" },
  E: { label: "E（極小駅）", color: "text-red-700" },
};

export function LandPriceAnalysis({ comparison, input, theoreticalPrice, stationRidership, populationForecast, landAppraisal, externalUrbanRisks }: Props) {
  const { stats, assessment, inputPricePerTsubo, diffFromMedian } = comparison;

  const allUrbanRisks: UrbanRisk[] = [
    ...(stats.urbanRisks ?? []),
    ...(externalUrbanRisks ?? []).filter(r => !(stats.urbanRisks ?? []).some(e => e.code === r.code)),
  ];

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
        {allUrbanRisks.length > 0 && (
          <div className="space-y-2">
            {allUrbanRisks.map((risk) => {
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

        {/* 人口動態インジケーター */}
        {populationForecast && populationForecast.snapshots.length > 0 && (() => {
          const style = POPULATION_TREND_STYLE[populationForecast.trend] ?? POPULATION_TREND_STYLE["現状維持"];
          const displayYears = [2020, 2030, 2040, 2050];
          const snapshotMap = Object.fromEntries(populationForecast.snapshots.map(s => [s.year, s.pop]));
          const base2020 = snapshotMap[2020] ?? 0;
          return (
            <div className={`rounded-md border ${style.border} ${style.bg} p-3`}>
              <p className={`flex items-center gap-1 text-xs font-semibold mb-2 ${style.text}`}>
                <Users className="h-3.5 w-3.5" />
                人口動態インジケーター（XKT013 将来推計人口）
                <span className="ml-auto font-bold">{populationForecast.trend}</span>
              </p>
              <div className="grid grid-cols-4 gap-2">
                {displayYears.map(year => {
                  const pop = snapshotMap[year] ?? 0;
                  const ratePct = base2020 > 0 ? ((pop - base2020) / base2020 * 100) : 0;
                  return (
                    <div key={year} className="text-center">
                      <p className={`text-xs ${style.text} opacity-70`}>{year}年</p>
                      <p className={`text-xs font-semibold ${style.text}`}>{pop > 0 ? pop.toFixed(0) : "−"}人</p>
                      {year !== 2020 && base2020 > 0 && (
                        <p className={`text-xs ${ratePct >= 0 ? "text-green-700" : "text-red-700"}`}>
                          {ratePct >= 0 ? "+" : ""}{ratePct.toFixed(0)}%
                        </p>
                      )}
                    </div>
                  );
                })}
              </div>
              <p className={`mt-2 text-xs ${style.text} opacity-70`}>
                30年後変化率: {(populationForecast.changeRate30yr * 100).toFixed(0)}% ／ 推定空室率増加: +{(populationForecast.vacancyRateDelta * 100).toFixed(0)}%pt
              </p>
            </div>
          );
        })()}

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
                  tick={{ fontSize: 10 }}
                  label={{ value: "面積(m²)", position: "insideBottom", offset: -4, fontSize: 10 }}
                />
                <YAxis
                  dataKey="tsubo"
                  name="坪単価"
                  tickFormatter={(v) => `${v}万`}
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
        {/* 駅別乗降客数・需要スコア */}
        {stationRidership && stationRidership.length > 0 && (
          <div className="rounded-md border border-purple-200 bg-purple-50 p-3">
            <p className="text-xs font-semibold text-purple-800 mb-2">最寄り駅の乗降客数・賃貸需要スコア</p>
            <div className="space-y-1">
              {stationRidership.slice(0, 5).map((s) => {
                const scoreInfo = RIDERSHIP_SCORE_LABEL[s.demandScore] ?? { label: s.demandScore, color: "text-gray-700" };
                return (
                  <div key={`${s.stationName}-${s.lineName}`} className="flex items-center justify-between text-xs">
                    <span className="text-purple-900">{s.stationName}（{s.lineName}）</span>
                    <div className="flex items-center gap-2">
                      <span className="text-purple-700">{s.passengers.toLocaleString()}人/日</span>
                      <span className={`font-bold ${scoreInfo.color}`}>{scoreInfo.label}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* 地価公示2軸比較（XCT001 vs XIT001） */}
        {landAppraisal && landAppraisal.appraisalCount > 0 && (() => {
          const transactionMedianPerSqm = stats.medianTsubo > 0 ? stats.medianTsubo / SQM_PER_TSUBO : 0;
          const ratio = transactionMedianPerSqm > 0 && landAppraisal.appraisalMedianPerSqm > 0
            ? transactionMedianPerSqm / landAppraisal.appraisalMedianPerSqm
            : null;
          const trendStyle: Record<string, { border: string; bg: string; text: string }> = {
            "上昇": { border: "border-green-300",  bg: "bg-green-50",  text: "text-green-700" },
            "安定": { border: "border-yellow-300", bg: "bg-yellow-50", text: "text-yellow-700" },
            "下落": { border: "border-red-300",    bg: "bg-red-50",    text: "text-red-700" },
          };
          const style = trendStyle[landAppraisal.trendLabel] ?? trendStyle["安定"];
          return (
            <div className={`rounded-md border p-3 ${style.border} ${style.bg}`}>
              <p className={`flex items-center gap-1 text-sm font-semibold mb-2 ${style.text}`}>
                <Building2 className="h-4 w-4" />
                地価公示 vs 取引価格 2軸比較（XCT001）
              </p>
              <dl className="space-y-1 text-sm">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">公示価格中央値（住宅地）</dt>
                  <dd className="font-medium">{Math.round(landAppraisal.appraisalMedianPerSqm).toLocaleString()}円/m²</dd>
                </div>
                {transactionMedianPerSqm > 0 && (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">取引価格中央値</dt>
                    <dd className="font-medium">{Math.round(transactionMedianPerSqm).toLocaleString()}円/m²</dd>
                  </div>
                )}
                {ratio !== null && (
                  <div className="flex justify-between border-t pt-1">
                    <dt className="text-muted-foreground">取引/公示 比率</dt>
                    <dd className={`font-bold ${ratio > 1.1 ? "text-red-700" : ratio < 0.9 ? "text-green-700" : ""}`}>
                      {ratio.toFixed(2)}倍{ratio > 1.1 ? "（取引が公示を上回る）" : ratio < 0.9 ? "（取引が公示を下回る）" : "（公示と概ね一致）"}
                    </dd>
                  </div>
                )}
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">前年比トレンド</dt>
                  <dd className={`font-bold ${style.text}`}>
                    {landAppraisal.trendLabel}（{landAppraisal.appraisalTrend >= 0 ? "+" : ""}{(landAppraisal.appraisalTrend * 100).toFixed(1)}%）
                  </dd>
                </div>
                <div className="flex justify-between text-xs text-muted-foreground">
                  <dt>標準地点数</dt>
                  <dd>{landAppraisal.appraisalCount}地点</dd>
                </div>
              </dl>
              <p className="mt-2 text-xs text-muted-foreground">出典：国土交通省 不動産情報ライブラリ（XCT001）</p>
            </div>
          );
        })()}

        {/* 理論価格推定 */}
        {theoreticalPrice && (() => {
          const { theoreticalPriceJPY, deviationPct, ageCorrection, stationCorrection, ridershipCorrection, medianBuildingAge, medianStationMinutes, isLowDataWarning, hasStationData, hasRidershipData, ridershipScore } = theoreticalPrice;
          const isOverpriced = deviationPct > 20;
          const isUnderpriced = deviationPct < -20;
          const color = isOverpriced
            ? "border-red-200 bg-red-50"
            : isUnderpriced
            ? "border-green-200 bg-green-50"
            : "border-blue-200 bg-blue-50";
          return (
            <div className={`rounded-md border p-3 ${color}`}>
              <p className="flex items-center gap-1 text-sm font-semibold mb-2">
                <TrendingUp className="h-4 w-4" />
                理論価格推定（築年数・駅距離・需要スコア補正）
              </p>
              {isLowDataWarning && (
                <p className="text-xs text-yellow-700 mb-2">※ 築年数データが少ないため参考値</p>
              )}
              <dl className="space-y-1 text-sm">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">理論価格</dt>
                  <dd className="font-medium">{formatMan(theoreticalPriceJPY)}</dd>
                </div>
                <div className="flex justify-between border-t pt-1">
                  <dt className="text-muted-foreground">乖離率（販売価格 vs 理論価格）</dt>
                  <dd className={`font-bold ${isOverpriced ? "text-red-700" : isUnderpriced ? "text-green-700" : ""}`}>
                    {deviationPct > 0 ? "+" : ""}{deviationPct.toFixed(1)}%
                    {isOverpriced ? "（割高）" : isUnderpriced ? "（割安）" : "（相場）"}
                  </dd>
                </div>
                <div className="flex justify-between text-xs text-muted-foreground">
                  <dt>築年数補正（中央値{medianBuildingAge}年）</dt>
                  <dd>{ageCorrection >= 0 ? "+" : ""}{(ageCorrection * 100).toFixed(1)}%</dd>
                </div>
                {hasStationData && (
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <dt>駅距離補正（中央値{medianStationMinutes}分）</dt>
                    <dd>{stationCorrection >= 0 ? "+" : ""}{(stationCorrection * 100).toFixed(1)}%</dd>
                  </div>
                )}
                {hasRidershipData && ridershipScore && (
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <dt>需要スコア補正（スコア: {RIDERSHIP_SCORE_LABEL[ridershipScore]?.label ?? ridershipScore}）</dt>
                    <dd>{ridershipCorrection >= 0 ? "+" : ""}{(ridershipCorrection * 100).toFixed(1)}%</dd>
                  </div>
                )}
              </dl>
            </div>
          );
        })()}

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

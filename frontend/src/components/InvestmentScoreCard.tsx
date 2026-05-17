"use client";

import { Radar, RadarChart, PolarGrid, PolarAngleAxis, ResponsiveContainer } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { InvestmentScoreResult, ScoreItem } from "@/types/investment";
import { getRidershipRecommend, getPopulationVacancyDelta } from "@/lib/vacancyRateRecommender";

interface Props {
  score: InvestmentScoreResult;
  populationChangeRate?: number;
  onApplyRecommend?: (vacancyRate: number, rentDeclineRate: number) => void;
}

const GRADE_STYLE: Record<
  string,
  { badge: "success" | "default" | "warning" | "danger"; color: string }
> = {
  優良: { badge: "success", color: "text-green-600" },
  良好: { badge: "default", color: "text-blue-600" },
  普通: { badge: "warning", color: "text-yellow-600" },
  注意: { badge: "warning", color: "text-orange-600" },
  要注意: { badge: "danger", color: "text-red-600" },
};

function ScoreRow({ item }: { item: ScoreItem }) {
  const isPositive = item.score > 0;
  const isNegative = item.score < 0;
  return (
    <div className="flex items-start justify-between gap-2 py-1 text-xs border-b border last:border-0">
      <span className="text-muted-foreground min-w-[6rem]">{item.label}</span>
      <span
        className={`font-semibold tabular-nums min-w-[3rem] text-right ${
          isPositive ? "text-green-600" : isNegative ? "text-red-600" : "text-muted-foreground"
        }`}
      >
        {isPositive ? `+${item.score}` : item.score}
      </span>
      <span className="text-muted-foreground flex-1 text-right leading-tight">{item.description}</span>
    </div>
  );
}

export function InvestmentScoreCard({ score, populationChangeRate, onApplyRecommend }: Props) {
  const { totalScore, grade, breakdown } = score;
  const gradeStyle = GRADE_STYLE[grade] ?? GRADE_STYLE["普通"];

  const items: ScoreItem[] = [
    breakdown.population,
    breakdown.ridership,
    breakdown.urbanArea,
    breakdown.locationOptimization,
    breakdown.hazardRisk,
    breakdown.liquefactionRisk,
    breakdown.embankment,
    breakdown.disasterHistory,
  ];

  // Compute recommendation from ridership score (0–20 range from backend)
  const recommend = getRidershipRecommend(breakdown.ridership.score);
  const popDelta =
    populationChangeRate !== undefined ? getPopulationVacancyDelta(populationChangeRate) : 0;
  const recommendedVacancy = recommend.vacancyRate + popDelta;
  const recommendedRentDecline = recommend.rentDeclineRate;

  const handleApply = () => {
    onApplyRecommend?.(recommendedVacancy, recommendedRentDecline);
  };

  const tooltipText = [
    `乗降客数スコア ${recommend.grade} → 空室率 ${(recommend.vacancyRate * 100).toFixed(0)}% 推奨`,
    ...(popDelta > 0
      ? [`人口30年予測 −30%以上 → 空室率 +${(popDelta * 100).toFixed(0)}% 加算`]
      : []),
    `賃料下落率 ${(recommendedRentDecline * 100).toFixed(1)}%/年 推奨`,
  ].join(" / ");

  return (
    <Card className="border border-indigo-200">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-sm font-semibold text-indigo-900">
          <span>投資適地スコア（XKT013/015/001/003/025〜029/020/XST001）</span>
          <div className="flex items-center gap-2">
            <span className={`text-3xl font-bold tabular-nums ${gradeStyle.color}`}>
              {totalScore}
            </span>
            <span className="text-muted-foreground text-sm font-normal">/ 100</span>
            <Badge variant={gradeStyle.badge}>{grade}</Badge>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        {breakdown.radarData.length > 0 && (
          <div className="h-48 w-full mb-3">
            <ResponsiveContainer width="100%" height="100%">
              <RadarChart
                data={breakdown.radarData}
                margin={{ top: 8, right: 24, left: 24, bottom: 8 }}
              >
                <PolarGrid stroke="#e0e7ff" />
                <PolarAngleAxis dataKey="category" tick={{ fontSize: 10, fill: "#4338ca" }} />
                <Radar
                  name="スコア"
                  dataKey="score"
                  stroke="#6366f1"
                  fill="#6366f1"
                  fillOpacity={0.25}
                  strokeWidth={2}
                />
              </RadarChart>
            </ResponsiveContainer>
          </div>
        )}
        <div className="space-y-0">
          {items.map((item) => (
            <ScoreRow key={item.label} item={item} />
          ))}
        </div>

        {onApplyRecommend && (
          <div className="mt-3 rounded-md border border-indigo-100 bg-indigo-50 px-3 py-2">
            <p className="text-[11px] text-indigo-700 mb-2 leading-snug" title={tooltipText}>
              <span className="font-medium">推奨値：</span>
              {tooltipText}
            </p>
            <Button
              size="sm"
              variant="outline"
              className="h-8 text-xs border-indigo-300 text-indigo-700 hover:bg-indigo-100"
              onClick={handleApply}
              title={tooltipText}
            >
              推奨空室率 {(recommendedVacancy * 100).toFixed(0)}% ・賃料下落率{" "}
              {(recommendedRentDecline * 100).toFixed(1)}% を適用
            </Button>
          </div>
        )}

        <p className="mt-2 text-[10px] text-muted-foreground leading-tight">
          基準スコア50点 +
          各指標の加減点（0〜100にクランプ）。API取得失敗時は該当指標を0点として算出。
        </p>
      </CardContent>
    </Card>
  );
}

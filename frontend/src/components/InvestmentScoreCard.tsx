"use client";

import { Radar, RadarChart, PolarGrid, PolarAngleAxis, ResponsiveContainer } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { InvestmentScoreResult, ScoreItem } from "@/types/investment";

interface Props {
  score: InvestmentScoreResult;
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
    <div className="flex items-start justify-between gap-2 py-1 text-xs border-b border-gray-100 last:border-0">
      <span className="text-gray-600 min-w-[6rem]">{item.label}</span>
      <span
        className={`font-semibold tabular-nums min-w-[3rem] text-right ${
          isPositive ? "text-green-600" : isNegative ? "text-red-600" : "text-gray-500"
        }`}
      >
        {isPositive ? `+${item.score}` : item.score}
      </span>
      <span className="text-gray-500 flex-1 text-right leading-tight">{item.description}</span>
    </div>
  );
}

export function InvestmentScoreCard({ score }: Props) {
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

  return (
    <Card className="border border-indigo-200">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-sm font-semibold text-indigo-900">
          <span>投資適地スコア（XKT013/015/001/003/025〜029/020/XST001）</span>
          <div className="flex items-center gap-2">
            <span className={`text-3xl font-bold tabular-nums ${gradeStyle.color}`}>
              {totalScore}
            </span>
            <span className="text-gray-400 text-sm font-normal">/ 100</span>
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
        <p className="mt-2 text-[10px] text-gray-400 leading-tight">
          基準スコア50点 +
          各指標の加減点（0〜100にクランプ）。API取得失敗時は該当指標を0点として算出。
        </p>
      </CardContent>
    </Card>
  );
}

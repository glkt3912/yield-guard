import type { AreaDiscoveryResponse } from "@/lib/api";

export const areaDiscoveryFixture = {
  prefecture: "群馬県",
  items: [
    {
      municipalityCode: "10201",
      municipalityName: "前橋市",
      medianTsubo: 45000,
      transactionCount: 87,
      yieldDifficulty: "achievable" as const,
      yieldDifficultyLabel: "達成しやすい",
      landPriceTrend: "安定",
      dataSufficient: true,
    },
    {
      municipalityCode: "10202",
      municipalityName: "高崎市",
      medianTsubo: 52000,
      transactionCount: 112,
      yieldDifficulty: "slightly-difficult" as const,
      yieldDifficultyLabel: "やや難しい",
      landPriceTrend: "上昇",
      dataSufficient: true,
    },
  ],
} satisfies AreaDiscoveryResponse;

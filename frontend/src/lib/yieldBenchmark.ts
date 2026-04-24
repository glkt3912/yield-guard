const SQM_PER_TSUBO = 3.30578;
const STANDARD_BUILDING_COST_PER_TSUBO = 500_000;

export interface YieldBenchmark {
  estimatedYieldMin: number;
  estimatedYieldTypical: number;
  estimatedYieldMax: number;
  judgment: "realistic" | "slightly-high" | "high";
  judgmentLabel: string;
}

export function calcYieldBenchmark(params: {
  medianTsubo: number;
  minTsubo: number;
  maxTsubo: number;
  landAreaSqm: number;
  monthlyRent: number;
  buildingCost: number;
}): YieldBenchmark {
  const { medianTsubo, minTsubo, maxTsubo, landAreaSqm, monthlyRent, buildingCost } = params;
  const annualRent = monthlyRent * 12;
  const areaTsubo = landAreaSqm / SQM_PER_TSUBO;
  const buildingEst = buildingCost > 0 ? buildingCost : areaTsubo * STANDARD_BUILDING_COST_PER_TSUBO;

  const totalTypical = medianTsubo * areaTsubo + buildingEst;
  const totalHigh = maxTsubo * areaTsubo + buildingEst;
  const totalLow = minTsubo * areaTsubo + buildingEst;

  const estimatedYieldTypical = totalTypical > 0 ? annualRent / totalTypical : 0;
  const estimatedYieldMin = totalHigh > 0 ? annualRent / totalHigh : 0;
  const estimatedYieldMax = totalLow > 0 ? annualRent / totalLow : 0;

  let judgment: YieldBenchmark["judgment"];
  let judgmentLabel: string;
  if (estimatedYieldTypical <= 0) {
    judgment = "realistic";
    judgmentLabel = "データ不足のため判定できません";
  } else {
    const ratio = estimatedYieldTypical > 0 ? annualRent / totalTypical / estimatedYieldTypical : 1;
    if (ratio <= 1.0) {
      judgment = "realistic";
      judgmentLabel = "エリア相場と概ね合致しています";
    } else if (ratio <= 1.2) {
      judgment = "slightly-high";
      judgmentLabel = "やや高め — 賃料設定の根拠を確認してください";
    } else {
      judgment = "high";
      judgmentLabel = "エリア相場を大幅に上回る設定です";
    }
  }

  return { estimatedYieldMin, estimatedYieldTypical, estimatedYieldMax, judgment, judgmentLabel };
}

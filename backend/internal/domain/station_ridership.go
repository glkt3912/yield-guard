package domain

// RidershipDemandScore は駅乗降客数に基づく賃貸需要スコア（A〜E の5段階）
type RidershipDemandScore string

const (
	RidershipScoreA RidershipDemandScore = "A" // >=100,000人/日
	RidershipScoreB RidershipDemandScore = "B" // >=30,000人/日
	RidershipScoreC RidershipDemandScore = "C" // >=10,000人/日
	RidershipScoreD RidershipDemandScore = "D" // >=2,000人/日
	RidershipScoreE RidershipDemandScore = "E" // <2,000人/日

	// 需要スコア別の理論価格補正係数
	ridershipCorrA = 0.15
	ridershipCorrB = 0.08
	ridershipCorrC = 0.00
	ridershipCorrD = -0.08
	ridershipCorrE = -0.15
)

// IsValid は RidershipDemandScore が有効な値（A〜E）かどうかを返す
func (s RidershipDemandScore) IsValid() bool {
	switch s {
	case RidershipScoreA, RidershipScoreB, RidershipScoreC, RidershipScoreD, RidershipScoreE:
		return true
	}
	return false
}

// CalcRidershipDemandScore は日乗降客数から需要スコアを算出する
func CalcRidershipDemandScore(passengersPerDay int) RidershipDemandScore {
	switch {
	case passengersPerDay >= 100_000:
		return RidershipScoreA
	case passengersPerDay >= 30_000:
		return RidershipScoreB
	case passengersPerDay >= 10_000:
		return RidershipScoreC
	case passengersPerDay >= 2_000:
		return RidershipScoreD
	default:
		return RidershipScoreE
	}
}

// RidershipCorrectionFactor は需要スコアに対応する補正係数を返す
func RidershipCorrectionFactor(score RidershipDemandScore) float64 {
	switch score {
	case RidershipScoreA:
		return ridershipCorrA
	case RidershipScoreB:
		return ridershipCorrB
	case RidershipScoreC:
		return ridershipCorrC
	case RidershipScoreD:
		return ridershipCorrD
	default:
		return ridershipCorrE
	}
}

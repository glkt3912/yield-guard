package domain

import (
	"fmt"
	"strings"
)

// BuildUrbanRisksFromAPIs は MLIT 専用 API（XKT003/020/030/XST001）の結果からリスクを構築する。
// detectUrbanRisks（XIT001テキスト検出）とは独立して使用し、結果を呼び出し元でマージする。
func BuildUrbanRisksFromAPIs(
	locationItems []LocationOptimizationItem,
	embankmentItems []EmbankmentItem,
	roadItems []UrbanRoadItem,
	disasters []DisasterHistoryItem,
) []UrbanRisk {
	var risks []UrbanRisk

	// XKT003: 立地適正化計画フィーチャが存在し、居住誘導区域が含まれない場合
	if len(locationItems) > 0 {
		hasResidential := false
		for _, item := range locationItems {
			if strings.Contains(item.KubunNameJa, "居住誘導区域") {
				hasResidential = true
				break
			}
		}
		if !hasResidential {
			risks = append(risks, UrbanRisk{
				Code:        "OUTSIDE_RESIDENTIAL_GUIDANCE",
				Level:       UrbanRiskLevelWarning,
				Title:       "居住誘導区域外",
				Description: "立地適正化計画の居住誘導区域外です。将来的に行政サービスの縮小・インフラ維持コスト増加の可能性があります（コンパクトシティ計画）。",
			})
		}
	}

	// XKT020: 大規模盛土造成地フィーチャが存在する場合
	if len(embankmentItems) > 0 {
		embDesc := "大規模盛土造成地に該当します。地震時の沈下・崩壊リスクがあります。"
		if c := embankmentItems[0].Classification; c != "" {
			embDesc = fmt.Sprintf("大規模盛土造成地（%s）に該当します。地震時の沈下・崩壊リスクがあります。", c)
		}
		risks = append(risks, UrbanRisk{
			Code:        "LARGE_EMBANKMENT",
			Level:       UrbanRiskLevelWarning,
			Title:       "大規模盛土造成地",
			Description: embDesc,
		})
	}

	// XKT030: 都市計画道路（kubun_id=3011）フィーチャが存在する場合
	for _, item := range roadItems {
		if item.KubunID == 3011 {
			risks = append(risks, UrbanRisk{
				Code:        "URBAN_PLANNING_ROAD",
				Level:       UrbanRiskLevelWarning,
				Title:       "都市計画道路の予定地",
				Description: "都市計画道路の予定地に一部かかっています。将来的に建物の一部または全部が収用対象となる可能性があります。",
			})
			break
		}
	}

	// XST001: 災害履歴フィーチャが存在する場合
	if len(disasters) > 0 {
		names := make([]string, 0, len(disasters))
		seen := make(map[string]bool)
		for _, d := range disasters {
			if d.Name != "" && !seen[d.Name] {
				names = append(names, d.Name)
				seen[d.Name] = true
			}
		}
		desc := "このエリアで過去に災害が記録されています。"
		if len(names) > 0 {
			desc = fmt.Sprintf("このエリアで過去に災害が記録されています（%s）。", strings.Join(names, "・"))
		}
		risks = append(risks, UrbanRisk{
			Code:        "DISASTER_HISTORY",
			Level:       UrbanRiskLevelWarning,
			Title:       "災害履歴あり",
			Description: desc,
		})
	}

	return risks
}

// BuildHazardRisks は XKT026–029 ハザード API の結果を UrbanRisk スライスに変換する。
// 各ハザード種別で最も深刻な 1 件のみを返す（重複排除）。
func BuildHazardRisks(
	floods []FloodHazardItem,
	storms []StormHazardItem,
	tsunamis []TsunamiHazardItem,
	landslides []LandslideHazardItem,
) []UrbanRisk {
	var risks []UrbanRisk

	// XKT026: 洪水浸水想定区域 — DepthRank >= 3 を ERROR、それ以外を WARNING
	if len(floods) > 0 {
		worst := floods[0]
		for _, f := range floods[1:] {
			if f.DepthRank > worst.DepthRank {
				worst = f
			}
		}
		level := UrbanRiskLevelWarning
		if worst.DepthRank >= 3 {
			level = UrbanRiskLevelError
		}
		desc := fmt.Sprintf("洪水浸水想定区域（深さランク %d）に該当します。大雨・河川氾濫時に浸水リスクがあります。", worst.DepthRank)
		if worst.RiverName != "" {
			desc = fmt.Sprintf("洪水浸水想定区域（%s、深さランク %d）に該当します。大雨・河川氾濫時に浸水リスクがあります。", worst.RiverName, worst.DepthRank)
		}
		risks = append(risks, UrbanRisk{
			Code:        "FLOOD_HAZARD",
			Level:       level,
			Title:       "洪水浸水想定区域",
			Description: desc,
		})
	}

	// XKT027: 高潮浸水想定区域 — 存在すれば WARNING（深さ文字列は最初の非空エントリを採用）
	if len(storms) > 0 {
		depthJa := ""
		for _, s := range storms {
			if s.DepthJa != "" {
				depthJa = s.DepthJa
				break
			}
		}
		desc := "高潮浸水想定区域に該当します。台風・高波時に浸水リスクがあります。"
		if depthJa != "" {
			desc = fmt.Sprintf("高潮浸水想定区域（%s）に該当します。台風・高波時に浸水リスクがあります。", depthJa)
		}
		risks = append(risks, UrbanRisk{
			Code:        "STORM_HAZARD",
			Level:       UrbanRiskLevelWarning,
			Title:       "高潮浸水想定区域",
			Description: desc,
		})
	}

	// XKT028: 津波浸水想定区域 — 存在すれば ERROR
	if len(tsunamis) > 0 {
		desc := "津波浸水想定区域に該当します。沿岸部の津波・高潮による浸水リスクがあります。"
		if tsunamis[0].DepthJa != "" {
			desc = fmt.Sprintf("津波浸水想定区域（%s）に該当します。沿岸部の津波・高潮による浸水リスクがあります。", tsunamis[0].DepthJa)
		}
		risks = append(risks, UrbanRisk{
			Code:        "TSUNAMI_HAZARD",
			Level:       UrbanRiskLevelError,
			Title:       "津波浸水想定区域",
			Description: desc,
		})
	}

	// XKT029: 土砂災害警戒区域 — ZoneCode=1（特別警戒）を ERROR、ZoneCode=2（警戒）を WARNING
	if len(landslides) > 0 {
		// 最も深刻な ZoneCode（1 < 2、つまり 1 が最重要）を選択
		worst := landslides[0]
		for _, l := range landslides[1:] {
			if l.ZoneCode < worst.ZoneCode {
				worst = l
			}
		}
		level := UrbanRiskLevelWarning
		zoneName := "警戒区域"
		if worst.ZoneCode == 1 {
			level = UrbanRiskLevelError
			zoneName = "特別警戒区域"
		}
		phenomenonNames := map[int]string{1: "急傾斜地崩壊", 2: "土石流", 3: "地すべり"}
		pName, ok := phenomenonNames[worst.PhenomenonType]
		if !ok {
			pName = "土砂災害"
		}
		desc := fmt.Sprintf("土砂災害%s（%s）に該当します。大雨時の土砂崩れ・土石流リスクがあります。", zoneName, pName)
		risks = append(risks, UrbanRisk{
			Code:        "LANDSLIDE_HAZARD",
			Level:       level,
			Title:       "土砂災害警戒区域",
			Description: desc,
		})
	}

	return risks
}

// calcZoningSummary は取引データから最頻の用途地域情報を抽出する
func calcZoningSummary(transactions []LandTransaction) *ZoningSummary {
	if len(transactions) == 0 {
		return nil
	}
	cp := modalString(transactions, func(t LandTransaction) string { return t.CityPlanning })
	bc := modalString(transactions, func(t LandTransaction) string { return t.BuildingCoverage })
	far := modalString(transactions, func(t LandTransaction) string { return t.FloorAreaRatio })
	if cp == "" && bc == "" && far == "" {
		return nil
	}
	return &ZoningSummary{
		CityPlanning:     cp,
		BuildingCoverage: bc,
		FloorAreaRatio:   far,
	}
}

// detectUrbanRisks は取引データと用途地域サマリーから都市計画リスクを検出する
func detectUrbanRisks(transactions []LandTransaction, zoning *ZoningSummary) []UrbanRisk {
	var risks []UrbanRisk
	if zoning == nil {
		return risks
	}

	// 市街化調整区域（最頻値が調整区域）
	if strings.Contains(zoning.CityPlanning, "市街化調整区域") {
		risks = append(risks, UrbanRisk{
			Code:        "URBANIZATION_CONTROL_ZONE",
			Level:       UrbanRiskLevelError,
			Title:       "市街化調整区域",
			Description: "原則として新たな建築・用途変更が制限されます。既存建物の建替えが困難になる可能性があります。",
		})
	}

	// 都市計画区域外 / 非線引き区域
	if strings.Contains(zoning.CityPlanning, "非線引") || zoning.CityPlanning == "都市計画区域外" {
		risks = append(risks, UrbanRisk{
			Code:        "UNZONED_AREA",
			Level:       UrbanRiskLevelWarning,
			Title:       "非線引き・都市計画区域外",
			Description: "インフラ整備が遅れやすく、将来の資産価値が不安定になる可能性があります。",
		})
	}

	// 調整区域が最頻値でなくとも30%以上混在する場合の注意
	if !strings.Contains(zoning.CityPlanning, "市街化調整区域") {
		controlCount, totalWithCP := 0, 0
		for _, t := range transactions {
			if t.CityPlanning != "" {
				totalWithCP++
				if strings.Contains(t.CityPlanning, "市街化調整区域") {
					controlCount++
				}
			}
		}
		if totalWithCP > 0 && float64(controlCount)/float64(totalWithCP) >= 0.3 {
			ratio := float64(controlCount) / float64(totalWithCP) * 100
			risks = append(risks, UrbanRisk{
				Code:        "MIXED_ZONE_CAUTION",
				Level:       UrbanRiskLevelWarning,
				Title:       "市街化調整区域が混在",
				Description: fmt.Sprintf("エリア内取引の約%.0f%%が市街化調整区域です。対象物件の区域区分を必ず確認してください。", ratio),
			})
		}
	}

	return risks
}

// modalString は transactions から getter で取得した文字列の最頻値を返す（空文字は除外）
func modalString(transactions []LandTransaction, getter func(LandTransaction) string) string {
	counts := make(map[string]int, len(transactions))
	for _, t := range transactions {
		v := getter(t)
		if v != "" {
			counts[v]++
		}
	}
	best, bestCount := "", 0
	for v, c := range counts {
		if c > bestCount {
			best, bestCount = v, c
		}
	}
	return best
}

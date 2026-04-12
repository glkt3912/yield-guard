package domain

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed testdata/depreciation_cases.json
var depreciationFixture []byte

type depreciationCase struct {
	Name         string       `json:"name"`
	BuildingType BuildingType `json:"buildingType"`
	BuildingAge  int          `json:"buildingAge"`
	Expected     int          `json:"expected"`
	Note         string       `json:"note"`
}

func TestCalcResidualUsefulLife_GoldenMaster(t *testing.T) {
	t.Parallel()

	var cases []depreciationCase
	if err := json.Unmarshal(depreciationFixture, &cases); err != nil {
		t.Fatalf("Golden Master JSONのパースに失敗しました: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			got := CalcResidualUsefulLife(tc.BuildingType, tc.BuildingAge)
			if got != tc.Expected {
				t.Errorf(
					"CalcResidualUsefulLife(%q, %d) = %d, want %d\n  備考: %s",
					tc.BuildingType, tc.BuildingAge, got, tc.Expected, tc.Note,
				)
			}
		})
	}
}

package domain

import "testing"

// TestBuildingTypeUsefulLife は全建物種別の法定耐用年数を検証する
func TestBuildingTypeUsefulLife(t *testing.T) {
	tests := []struct {
		name     string
		bt       BuildingType
		wantLife int
	}{
		{"木造", BuildingTypeWood, 22},
		{"軽量鉄骨(3mm以下)", BuildingTypeLightSteelThin, 19},
		{"軽量鉄骨(4mm以下)", BuildingTypeLightSteel, 27},
		{"重量鉄骨", BuildingTypeHeavySteel, 34},
		{"RC造", BuildingTypeRC, 47},
		{"SRC造", BuildingTypeSRC, 47},
		{"不明(デフォルト)", BuildingType("unknown"), 22},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.bt.UsefulLife()
			if got != tt.wantLife {
				t.Errorf("UsefulLife() = %d, want %d", got, tt.wantLife)
			}
		})
	}
}

// TestCalcResidualUsefulLife は簡便法耐用年数の全分岐を検証する
func TestCalcResidualUsefulLife(t *testing.T) {
	tests := []struct {
		name        string
		bt          BuildingType
		buildingAge int
		want        int
	}{
		{
			// 新築: そのまま法定耐用年数
			name: "新築(age=0)", bt: BuildingTypeWood, buildingAge: 0, want: 22,
		},
		{
			// 負数も新築扱い
			name: "負数age", bt: BuildingTypeWood, buildingAge: -1, want: 22,
		},
		{
			// 法定内: (22-10) + int(10×0.2) = 12+2 = 14
			name: "法定内・木造10年", bt: BuildingTypeWood, buildingAge: 10, want: 14,
		},
		{
			// 法定内: (47-20) + int(20×0.2) = 27+4 = 31
			name: "法定内・RC20年", bt: BuildingTypeRC, buildingAge: 20, want: 31,
		},
		{
			// 法定年数ちょうど(超過扱い): int(22×0.2) = int(4.4) = 4
			name: "法定ちょうど・木造22年", bt: BuildingTypeWood, buildingAge: 22, want: 4,
		},
		{
			// 法定超過: int(22×0.2) = 4
			name: "法定超過・木造30年", bt: BuildingTypeWood, buildingAge: 30, want: 4,
		},
		{
			// 法定超過・LightSteelThin: int(19×0.2) = int(3.8) = 3
			name: "法定超過・軽量鉄骨(3mm)25年", bt: BuildingTypeLightSteelThin, buildingAge: 25, want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcResidualUsefulLife(tt.bt, tt.buildingAge)
			if got != tt.want {
				t.Errorf("CalcResidualUsefulLife(%q, %d) = %d, want %d", tt.bt, tt.buildingAge, got, tt.want)
			}
		})
	}
}

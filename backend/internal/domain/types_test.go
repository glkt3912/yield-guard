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

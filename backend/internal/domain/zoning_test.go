package domain

import "testing"

func TestParseZoningType(t *testing.T) {
	tests := []struct {
		input string
		want  ZoningType
	}{
		{"第一種低層住居専用地域", ZoningFirstLowRise},
		{"商業地域", ZoningCommercial},
		{"工業専用地域", ZoningExclusiveIndustrial},
		{"田園住居地域", ZoningGardenCity},
		{"不明な地域", ZoningUnknown},
		{"", ZoningUnknown},
	}
	for _, tt := range tests {
		if got := ParseZoningType(tt.input); got != tt.want {
			t.Errorf("ParseZoningType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestZoningMeta_RiskLevel(t *testing.T) {
	if ZoningFirstLowRise.Meta().RiskLevel != ZoningRiskNone {
		t.Error("first low rise should have no risk")
	}
	if ZoningExclusiveIndustrial.Meta().RiskLevel != ZoningRiskHigh {
		t.Error("exclusive industrial should be high risk")
	}
	if ZoningCommercial.Meta().RiskLevel != ZoningRiskCaution {
		t.Error("commercial should be caution risk")
	}
}

func TestParseCityPlanningArea(t *testing.T) {
	tests := []struct {
		input    string
		wantHigh bool
	}{
		{"市街化区域", false},
		{"市街化調整区域", true},
		{"非線引き区域", false},
		{"都市計画区域外", false},
		{"不明", false},
	}
	for _, tt := range tests {
		got := ParseCityPlanningArea(tt.input)
		if got.IsHighRisk() != tt.wantHigh {
			t.Errorf("ParseCityPlanningArea(%q).IsHighRisk() = %v, want %v", tt.input, got.IsHighRisk(), tt.wantHigh)
		}
	}
}

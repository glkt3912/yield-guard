package domain_test

import (
	"testing"

	"github.com/yield-guard/backend/internal/domain"
)

func TestCalcPopulationForecast(t *testing.T) {
	base := []domain.PopulationForecastItem{
		{Year: 2020, Pop: 1000},
		{Year: 2025, Pop: 980},
		{Year: 2030, Pop: 900},
		{Year: 2035, Pop: 820},
		{Year: 2040, Pop: 750},
		{Year: 2045, Pop: 700},
		{Year: 2050, Pop: 680},
	}

	tests := []struct {
		name              string
		items             []domain.PopulationForecastItem
		wantTrend         domain.PopulationTrend
		wantChangeNegative bool
		wantVacancyDelta  float64
	}{
		{
			name: "steep decline -32%",
			items: func() []domain.PopulationForecastItem {
				items := make([]domain.PopulationForecastItem, len(base))
				copy(items, base)
				return items
			}(),
			wantTrend:          domain.PopulationTrendSteepDecline,
			wantChangeNegative: true,
		},
		{
			name: "growth",
			items: []domain.PopulationForecastItem{
				{Year: 2020, Pop: 1000}, {Year: 2025, Pop: 1050},
				{Year: 2030, Pop: 1100}, {Year: 2035, Pop: 1150},
				{Year: 2040, Pop: 1200}, {Year: 2045, Pop: 1250},
				{Year: 2050, Pop: 1300},
			},
			wantTrend:          domain.PopulationTrendGrowth,
			wantChangeNegative: false,
			wantVacancyDelta:   0,
		},
		{
			name: "stable -3%",
			items: []domain.PopulationForecastItem{
				{Year: 2020, Pop: 1000}, {Year: 2025, Pop: 998},
				{Year: 2030, Pop: 995}, {Year: 2035, Pop: 990},
				{Year: 2040, Pop: 985}, {Year: 2045, Pop: 975},
				{Year: 2050, Pop: 970},
			},
			wantTrend:          domain.PopulationTrendStable,
			wantChangeNegative: true,
		},
		{
			name: "slow decline -12%",
			items: []domain.PopulationForecastItem{
				{Year: 2020, Pop: 1000}, {Year: 2025, Pop: 990},
				{Year: 2030, Pop: 970}, {Year: 2035, Pop: 950},
				{Year: 2040, Pop: 920}, {Year: 2045, Pop: 900},
				{Year: 2050, Pop: 880},
			},
			wantTrend:          domain.PopulationTrendSlowDecline,
			wantChangeNegative: true,
		},
		{
			name:             "empty items",
			items:            []domain.PopulationForecastItem{},
			wantTrend:        domain.PopulationTrendStable,
			wantVacancyDelta: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.CalcPopulationForecast(tt.items)
			if got.Trend != tt.wantTrend {
				t.Errorf("Trend = %q, want %q", got.Trend, tt.wantTrend)
			}
			if tt.wantChangeNegative && got.ChangeRate30yr >= 0 {
				t.Errorf("ChangeRate30yr = %f, want negative", got.ChangeRate30yr)
			}
			if !tt.wantChangeNegative && got.VacancyRateDelta != 0 && tt.wantVacancyDelta == 0 {
				t.Errorf("VacancyRateDelta = %f, want 0 for growth", got.VacancyRateDelta)
			}
			if got.VacancyRateDelta < 0 {
				t.Errorf("VacancyRateDelta must not be negative, got %f", got.VacancyRateDelta)
			}
		})
	}
}

func TestCalcPopulationForecast_VacancyDeltaFormula(t *testing.T) {
	items := []domain.PopulationForecastItem{
		{Year: 2020, Pop: 1000},
		{Year: 2025, Pop: 900},
		{Year: 2030, Pop: 800},
		{Year: 2035, Pop: 750},
		{Year: 2040, Pop: 720},
		{Year: 2045, Pop: 700},
		{Year: 2050, Pop: 680},
	}
	got := domain.CalcPopulationForecast(items)
	// changeRate = (680-1000)/1000 = -0.32, vacancyDelta = 0.32*0.5 = 0.16
	wantDelta := 0.16
	if got.VacancyRateDelta < wantDelta-0.001 || got.VacancyRateDelta > wantDelta+0.001 {
		t.Errorf("VacancyRateDelta = %f, want ~%f", got.VacancyRateDelta, wantDelta)
	}
}

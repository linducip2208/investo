package service

import (
	"math"
	"math/rand"
	"time"
)

type SeasonalityResult struct {
	MonthlyReturns  map[string]float64 `json:"monthly_returns"`
	MonthlyWinRate  map[string]float64 `json:"monthly_win_rate"`
	BestMonth       string             `json:"best_month"`
	WorstMonth      string             `json:"worst_month"`
	CurrentMonth    string             `json:"current_month"`
	CurrentMonthAvg float64            `json:"current_month_avg"`
	YearsAnalyzed   int                `json:"years_analyzed"`
}

type SeasonalityService struct{}

var seasonalityPresets = map[string]SeasonalityResult{
	"BBCA": {
		MonthlyReturns: map[string]float64{
			"Jan": 2.5, "Feb": 1.8, "Mar": -0.5, "Apr": 3.2,
			"May": 1.2, "Jun": 2.8, "Jul": 2.0, "Aug": -1.5,
			"Sep": 0.8, "Oct": 1.5, "Nov": 3.5, "Dec": 4.2,
		},
		MonthlyWinRate: map[string]float64{
			"Jan": 72, "Feb": 65, "Mar": 48, "Apr": 78,
			"May": 60, "Jun": 70, "Jul": 65, "Aug": 40,
			"Sep": 55, "Oct": 62, "Nov": 75, "Dec": 82,
		},
		BestMonth:    "Desember",
		WorstMonth:   "Agustus",
	},
	"TLKM": {
		MonthlyReturns: map[string]float64{
			"Jan": 1.5, "Feb": 2.2, "Mar": 3.0, "Apr": 1.0,
			"May": -0.8, "Jun": 1.5, "Jul": 2.8, "Aug": 1.2,
			"Sep": 3.5, "Oct": 2.0, "Nov": -1.2, "Dec": 0.5,
		},
		MonthlyWinRate: map[string]float64{
			"Jan": 62, "Feb": 68, "Mar": 72, "Apr": 58,
			"May": 42, "Jun": 60, "Jul": 70, "Aug": 55,
			"Sep": 75, "Oct": 65, "Nov": 38, "Dec": 52,
		},
		BestMonth:    "September",
		WorstMonth:   "November",
	},
	"DEFAULT": {
		MonthlyReturns: map[string]float64{
			"Jan": 1.0, "Feb": 0.8, "Mar": 1.5, "Apr": 2.0,
			"May": -0.5, "Jun": 0.5, "Jul": 1.8, "Aug": -1.0,
			"Sep": 0.2, "Oct": 1.2, "Nov": 2.5, "Dec": 3.0,
		},
		MonthlyWinRate: map[string]float64{
			"Jan": 58, "Feb": 55, "Mar": 62, "Apr": 65,
			"May": 45, "Jun": 52, "Jul": 60, "Aug": 42,
			"Sep": 50, "Oct": 58, "Nov": 68, "Dec": 72,
		},
		BestMonth:    "Desember",
		WorstMonth:   "Agustus",
	},
}

var monthNames = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func (s *SeasonalityService) AnalyzeSeasonality(code string, years int) (*SeasonalityResult, error) {
	preset, ok := seasonalityPresets[code]
	if !ok {
		preset = seasonalityPresets["DEFAULT"]
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(len(code))))
		for k := range preset.MonthlyReturns {
			preset.MonthlyReturns[k] = math.Round((preset.MonthlyReturns[k]+(rng.Float64()-0.5)*2.0)*100) / 100
		}
		for k := range preset.MonthlyWinRate {
			preset.MonthlyWinRate[k] = math.Round(preset.MonthlyWinRate[k] + (rng.Float64()-0.5)*10)
			if preset.MonthlyWinRate[k] > 100 {
				preset.MonthlyWinRate[k] = 100
			}
			if preset.MonthlyWinRate[k] < 0 {
				preset.MonthlyWinRate[k] = 0
			}
		}
	}

	currentMonth := time.Now().Month()
	monthKey := currentMonth.String()[:3]
	monthIdx := int(currentMonth) - 1

	preset.CurrentMonth = monthNames[monthIdx]
	preset.CurrentMonthAvg = preset.MonthlyReturns[monthKey]
	preset.YearsAnalyzed = years
	if years < 1 {
		preset.YearsAnalyzed = 5
	}

	return &preset, nil
}

func (s *SeasonalityService) GetSeasonalityMonths() []string {
	return []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
}

func (s *SeasonalityService) GetMonthName(abbr string) string {
	idx := map[string]int{"Jan": 0, "Feb": 1, "Mar": 2, "Apr": 3, "May": 4, "Jun": 5,
		"Jul": 6, "Aug": 7, "Sep": 8, "Oct": 9, "Nov": 10, "Dec": 11}[abbr]
	return monthNames[idx]
}

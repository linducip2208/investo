package service

import (
	"math"
	"sort"
	"time"

	"investo/internal/repository"
)

type SeasonalityResult struct {
	MonthlyReturns  map[string]float64 `json:"monthly_returns"`
	MonthlyWinRate  map[string]float64 `json:"monthly_win_rate"`
	BestMonth       string             `json:"best_month"`
	WorstMonth      string             `json:"worst_month"`
	CurrentMonth    string             `json:"current_month"`
	CurrentMonthAvg float64            `json:"current_month_avg"`
	YearsAnalyzed   int                `json:"years_analyzed"`
	IsIllustrative  bool               `json:"is_illustrative,omitempty"`
	Source          string             `json:"source,omitempty"`
	AsOf            string             `json:"as_of,omitempty"`
	DataPoints      int                `json:"data_points,omitempty"`
}

type SeasonalityService struct {
	StockRepo *repository.StockRepository
	PriceRepo *repository.StockPriceRepository
}

func NewSeasonalityService(stockRepo *repository.StockRepository, priceRepo *repository.StockPriceRepository) *SeasonalityService {
	return &SeasonalityService{StockRepo: stockRepo, PriceRepo: priceRepo}
}

// SetPriceSource wires optional price-history access after construction.
// Without it AnalyzeSeasonality falls back to clearly-labeled estimates.
func (s *SeasonalityService) SetPriceSource(stockRepo *repository.StockRepository, priceRepo *repository.StockPriceRepository) {
	s.StockRepo = stockRepo
	s.PriceRepo = priceRepo
}

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
	if years < 1 {
		years = 5
	}
	if real, ok := s.analyzeFromHistory(code, years); ok {
		return real, nil
	}
	return s.estimateFallback(code, years), nil
}

// analyzeFromHistory computes real monthly seasonality from price history.
// Returns ok=false when no price source is wired or history is insufficient
// (<12 monthly observations).
func (s *SeasonalityService) analyzeFromHistory(code string, years int) (*SeasonalityResult, bool) {
	if s == nil || s.StockRepo == nil || s.PriceRepo == nil {
		return nil, false
	}
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil || stock == nil {
		return nil, false
	}
	now := time.Now()
	start := now.AddDate(-years, 0, 0)
	prices, err := s.PriceRepo.FindByStockDate(stock.ID, start, now)
	if err != nil || len(prices) < 2 {
		return nil, false
	}
	sort.Slice(prices, func(i, j int) bool { return prices[i].Date.Before(prices[j].Date) })

	// Resample to month-end closes, then compute month-over-month returns.
	type monthClose struct {
		key   string // 2006-01
		mon   time.Month
		year  int
		close float64
		asOf  time.Time
	}
	var months []monthClose
	for _, p := range prices {
		if p.Close <= 0 {
			continue
		}
		key := p.Date.Format("2006-01")
		if len(months) > 0 && months[len(months)-1].key == key {
			months[len(months)-1].close = p.Close
			months[len(months)-1].asOf = p.Date
			continue
		}
		months = append(months, monthClose{key: key, mon: p.Date.Month(), year: p.Date.Year(), close: p.Close, asOf: p.Date})
	}
	if len(months) < 13 {
		return nil, false
	}

	byMonth := make(map[time.Month][]float64, 12)
	yearsSeen := make(map[int]bool, years)
	var lastAsOf time.Time
	count := 0
	for i := 1; i < len(months); i++ {
		prev, cur := months[i-1], months[i]
		if prev.close <= 0 {
			continue
		}
		ret := (cur.close - prev.close) / prev.close * 100
		byMonth[cur.mon] = append(byMonth[cur.mon], ret)
		yearsSeen[cur.year] = true
		lastAsOf = cur.asOf
		count++
	}
	if count < 12 {
		return nil, false
	}

	abbr := s.GetSeasonalityMonths()
	returns := make(map[string]float64, 12)
	winRate := make(map[string]float64, 12)
	bestAvg, worstAvg := math.Inf(-1), math.Inf(1)
	bestMonth, worstMonth := "", ""
	for m := time.January; m <= time.December; m++ {
		obs := byMonth[m]
		key := abbr[int(m)-1]
		if len(obs) == 0 {
			returns[key] = 0
			winRate[key] = 0
			continue
		}
		sum, wins := 0.0, 0
		for _, r := range obs {
			sum += r
			if r > 0 {
				wins++
			}
		}
		avg := sum / float64(len(obs))
		returns[key] = math.Round(avg*100) / 100
		winRate[key] = math.Round(float64(wins) / float64(len(obs)) * 100)
		if avg > bestAvg {
			bestAvg, bestMonth = avg, monthNames[int(m)-1]
		}
		if avg < worstAvg {
			worstAvg, worstMonth = avg, monthNames[int(m)-1]
		}
	}

	currentMonth := now.Month()
	monthKey := abbr[int(currentMonth)-1]
	yearsAnalyzed := len(yearsSeen)
	if yearsAnalyzed < 1 {
		yearsAnalyzed = 1
	}
	return &SeasonalityResult{
		MonthlyReturns:  returns,
		MonthlyWinRate:  winRate,
		BestMonth:       bestMonth,
		WorstMonth:      worstMonth,
		CurrentMonth:    monthNames[int(currentMonth)-1],
		CurrentMonthAvg: returns[monthKey],
		YearsAnalyzed:   yearsAnalyzed,
		Source:          "harga historis stock_prices (internal)",
		AsOf:            lastAsOf.Format("2006-01-02"),
		DataPoints:      count,
	}, true
}

// estimateFallback returns the static presets as clearly-labeled estimates.
// The maps are deep-copied so callers can never mutate the shared presets.
func (s *SeasonalityService) estimateFallback(code string, years int) *SeasonalityResult {
	preset, ok := seasonalityPresets[code]
	if !ok {
		preset = seasonalityPresets["DEFAULT"]
	}
	out := SeasonalityResult{
		MonthlyReturns:  make(map[string]float64, len(preset.MonthlyReturns)),
		MonthlyWinRate:  make(map[string]float64, len(preset.MonthlyWinRate)),
		BestMonth:       preset.BestMonth,
		WorstMonth:      preset.WorstMonth,
		YearsAnalyzed:   years,
		IsIllustrative:  true,
		Source:          "estimasi ilustratif — riwayat harga belum mencukupi",
		AsOf:            time.Now().Format("2006-01-02"),
	}
	for k, v := range preset.MonthlyReturns {
		out.MonthlyReturns[k] = v
	}
	for k, v := range preset.MonthlyWinRate {
		out.MonthlyWinRate[k] = v
	}

	currentMonth := time.Now().Month()
	monthKey := currentMonth.String()[:3]
	monthIdx := int(currentMonth) - 1

	out.CurrentMonth = monthNames[monthIdx]
	out.CurrentMonthAvg = out.MonthlyReturns[monthKey]

	return &out
}

func (s *SeasonalityService) GetSeasonalityMonths() []string {
	return []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
}

func (s *SeasonalityService) GetMonthName(abbr string) string {
	idx := map[string]int{"Jan": 0, "Feb": 1, "Mar": 2, "Apr": 3, "May": 4, "Jun": 5,
		"Jul": 6, "Aug": 7, "Sep": 8, "Oct": 9, "Nov": 10, "Dec": 11}[abbr]
	return monthNames[idx]
}

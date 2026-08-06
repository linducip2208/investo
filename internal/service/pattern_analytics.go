package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/repository"
	"investo/internal/service/pattern"
)

type PatternReliability struct {
	Pattern             string  `json:"pattern"`
	TotalOccurrences    int     `json:"total_occurrences"`
	SuccessfulPredictions int   `json:"successful_predictions"`
	SuccessRate         float64 `json:"success_rate"`
	AvgReturn5d         float64 `json:"avg_return_5d"`
	AvgReturn20d        float64 `json:"avg_return_20d"`
}

type PatternAnalyticsService struct {
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
	PatternService *pattern.PatternService
}

func (s *PatternAnalyticsService) AnalyzeReliability(code string) ([]PatternReliability, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("PatternAnalyticsService.AnalyzeReliability: stock not found: %w", err)
	}

	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	prices, err := s.StockPriceRepo.FindByStockDate(stock.ID, start, end)
	if err != nil || len(prices) == 0 {
		latest, err := s.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			return nil, fmt.Errorf("PatternAnalyticsService.AnalyzeReliability: no price data")
		}
		ReversePrices(latest)
		prices = latest
	}

	type agg struct {
		total     int
		success5  int
		success20 int
		sumRet5   float64
		sumRet20  float64
	}

	patternsAgg := make(map[string]*agg)
	windowSize := 60

	for i := windowSize; i < len(prices); i++ {
		window := prices[i-windowSize : i]
		detected := s.PatternService.DetectAll(window)
		if detected == nil || len(detected) == 0 {
			continue
		}

		patternIdx := i - 1
		if patternIdx < 0 {
			continue
		}
		entryPrice := prices[patternIdx].Close

		for _, pat := range detected {
			a, ok := patternsAgg[pat.Name]
			if !ok {
				a = &agg{}
				patternsAgg[pat.Name] = a
			}
			a.total++

			if patternIdx+5 < len(prices) {
				price5 := prices[patternIdx+5].Close
				ret5 := ((price5 - entryPrice) / entryPrice) * 100
				a.sumRet5 += ret5
				if price5 > entryPrice {
					a.success5++
				}
			}

			if patternIdx+20 < len(prices) {
				price20 := prices[patternIdx+20].Close
				ret20 := ((price20 - entryPrice) / entryPrice) * 100
				a.sumRet20 += ret20
				if price20 > entryPrice {
					a.success20++
				}
			}
		}
	}

	var results []PatternReliability
	for name, a := range patternsAgg {
		if a.total == 0 {
			continue
		}
		r := PatternReliability{
			Pattern:              name,
			TotalOccurrences:     a.total,
			SuccessfulPredictions: a.success5,
		}
		if a.total > 0 {
			r.SuccessRate = math.Round(float64(a.success5)/float64(a.total)*10000) / 100
		}
		if a.success5 > 0 {
			r.AvgReturn5d = math.Round(a.sumRet5/float64(a.success5)*100) / 100
		}
		if a.success20 > 0 {
			r.AvgReturn20d = math.Round(a.sumRet20/float64(a.success20)*100) / 100
		}
		results = append(results, r)
	}

	return results, nil
}

func (s *PatternAnalyticsService) MarketPatternStats() ([]PatternReliability, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("PatternAnalyticsService.MarketPatternStats: %w", err)
	}

	if len(stocks) > 10 {
		stocks = stocks[:10]
	}

	marketAgg := make(map[string]*struct {
		total     int
		success5  int
		success20 int
		sumRet5   float64
		sumRet20  float64
	})

	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	for _, stock := range stocks {
		prices, err := s.StockPriceRepo.FindByStockDate(stock.ID, start, end)
		if err != nil || len(prices) < 60 {
			latest, err := s.StockPriceRepo.FindLatest(stock.ID, 365)
			if err != nil || len(latest) < 60 {
				continue
			}
			ReversePrices(latest)
			prices = latest
		}

		windowSize := 60
		for i := windowSize; i < len(prices); i++ {
			window := prices[i-windowSize : i]
			detected := s.PatternService.DetectAll(window)
			if detected == nil || len(detected) == 0 {
				continue
			}

			patternIdx := i - 1
			if patternIdx < 0 {
				continue
			}
			entryPrice := prices[patternIdx].Close

			for _, pat := range detected {
				a, ok := marketAgg[pat.Name]
				if !ok {
					a = &struct {
						total     int
						success5  int
						success20 int
						sumRet5   float64
						sumRet20  float64
					}{}
					marketAgg[pat.Name] = a
				}
				a.total++

				if patternIdx+5 < len(prices) {
					price5 := prices[patternIdx+5].Close
					ret5 := ((price5 - entryPrice) / entryPrice) * 100
					a.sumRet5 += ret5
					if price5 > entryPrice {
						a.success5++
					}
				}

				if patternIdx+20 < len(prices) {
					price20 := prices[patternIdx+20].Close
					ret20 := ((price20 - entryPrice) / entryPrice) * 100
					a.sumRet20 += ret20
					if price20 > entryPrice {
						a.success20++
					}
				}
			}
		}
	}

	var results []PatternReliability
	for name, a := range marketAgg {
		if a.total == 0 {
			continue
		}
		r := PatternReliability{
			Pattern:              name,
			TotalOccurrences:     a.total,
			SuccessfulPredictions: a.success5,
		}
		if a.total > 0 {
			r.SuccessRate = math.Round(float64(a.success5)/float64(a.total)*10000) / 100
		}
		if a.success5 > 0 {
			r.AvgReturn5d = math.Round(a.sumRet5/float64(a.success5)*100) / 100
		}
		if a.success20 > 0 {
			r.AvgReturn20d = math.Round(a.sumRet20/float64(a.success20)*100) / 100
		}
		results = append(results, r)
	}

	return results, nil
}

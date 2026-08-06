package service

import (
	"fmt"
	"sort"
	"time"

	"investo/internal/repository"
	"investo/internal/service/pattern"
)

type PatternScannerService struct {
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
	PatternService *pattern.PatternService
}

type TimeframeScan struct {
	Code     string                  `json:"code"`
	Name     string                  `json:"name"`
	Timeframe string                 `json:"timeframe"`
	Patterns []pattern.PatternResult `json:"patterns"`
}

type timeframeSpec struct {
	label string
	days  int
}

var timeframes = []timeframeSpec{
	{"1D", 1},
	{"1W", 7},
	{"1M", 30},
}

func (s *PatternScannerService) ScanAllTimeframes(code string) ([]TimeframeScan, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("PatternScannerService.ScanAllTimeframes: stock not found: %w", err)
	}

	end := time.Now()
	var results []TimeframeScan

	for _, tf := range timeframes {
		start := end.AddDate(0, 0, -tf.days*5)
		prices, err := s.StockPriceRepo.FindByStockDate(stock.ID, start, end)
		if err != nil || len(prices) == 0 {
			latest, err := s.StockPriceRepo.FindLatest(stock.ID, tf.days*5)
			if err != nil || len(latest) == 0 {
				results = append(results, TimeframeScan{
					Code:     stock.Code,
					Name:     stock.Name,
					Timeframe: tf.label,
					Patterns: []pattern.PatternResult{},
				})
				continue
			}
			ReversePrices(latest)
			prices = latest
		}

		patterns := s.PatternService.DetectAll(prices)
		if patterns == nil {
			patterns = []pattern.PatternResult{}
		}

		results = append(results, TimeframeScan{
			Code:      stock.Code,
			Name:      stock.Name,
			Timeframe: tf.label,
			Patterns:  patterns,
		})
	}

	return results, nil
}

func (s *PatternScannerService) ScanMarket(timeframe string) ([]TimeframeScan, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("PatternScannerService.ScanMarket: %w", err)
	}

	var tf timeframeSpec
	switch timeframe {
	case "1D":
		tf = timeframes[0]
	case "1W":
		tf = timeframes[1]
	case "1M":
		tf = timeframes[2]
	default:
		tf = timeframes[0]
	}

	end := time.Now()
	start := end.AddDate(0, 0, -tf.days*5)

	var results []TimeframeScan

	for _, stock := range stocks {
		prices, err := s.StockPriceRepo.FindByStockDate(stock.ID, start, end)
		if err != nil || len(prices) == 0 {
			latest, err := s.StockPriceRepo.FindLatest(stock.ID, tf.days*5)
			if err != nil || len(latest) == 0 {
				continue
			}
			ReversePrices(latest)
			prices = latest
		}

		patterns := s.PatternService.DetectAll(prices)
		if patterns == nil || len(patterns) == 0 {
			continue
		}

		results = append(results, TimeframeScan{
			Code:      stock.Code,
			Name:      stock.Name,
			Timeframe: tf.label,
			Patterns:  patterns,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		maxRelI := 0
		maxRelJ := 0
		for _, p := range results[i].Patterns {
			if p.Reliability > maxRelI {
				maxRelI = p.Reliability
			}
		}
		for _, p := range results[j].Patterns {
			if p.Reliability > maxRelJ {
				maxRelJ = p.Reliability
			}
		}
		return maxRelI > maxRelJ
	})

	return results, nil
}

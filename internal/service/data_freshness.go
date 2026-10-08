package service

import (
	"fmt"
	"time"

	"investo/internal/repository"
)

type DataFreshness struct {
	StockCode       string    `json:"stock_code"`
	StockName       string    `json:"stock_name"`
	LastPriceUpdate time.Time `json:"last_price_update"`
	Age             string    `json:"age"`
	Status          string    `json:"status"`
}

type DataFreshnessService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (s *DataFreshnessService) CheckAll() ([]DataFreshness, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("DataFreshnessService.CheckAll: %w", err)
	}

	now := time.Now()
	var results []DataFreshness

	for _, stock := range stocks {
		lastUpdate, err := s.GetLastUpdate(stock.Code)
		if err != nil {
			results = append(results, DataFreshness{
				StockCode:       stock.Code,
				StockName:       stock.Name,
				LastPriceUpdate: time.Time{},
				Age:             "no data",
				Status:          "outdated",
			})
			continue
		}

		age := now.Sub(lastUpdate)
		ageStr := formatDuration(age)

		var status string
		switch {
		case age < 24*time.Hour:
			status = "fresh"
		case age < 72*time.Hour:
			status = "stale"
		default:
			status = "outdated"
		}

		results = append(results, DataFreshness{
			StockCode:       stock.Code,
			StockName:       stock.Name,
			LastPriceUpdate: lastUpdate,
			Age:             ageStr,
			Status:          status,
		})
	}

	return results, nil
}

func (s *DataFreshnessService) GetLastUpdate(stockCode string) (time.Time, error) {
	stock, err := s.StockRepo.FindByCode(stockCode)
	if err != nil {
		return time.Time{}, fmt.Errorf("DataFreshnessService.GetLastUpdate stock: %w", err)
	}

	prices, err := s.StockPriceRepo.FindLatest(stock.ID, 1)
	if err != nil {
		return time.Time{}, fmt.Errorf("DataFreshnessService.GetLastUpdate: %w", err)
	}

	if len(prices) == 0 {
		return time.Time{}, fmt.Errorf("no price data for %s", stockCode)
	}

	return prices[0].Date, nil
}

func formatDuration(d time.Duration) string {
	hours := int64(d.Hours())
	if hours < 1 {
		return "kurang dari 1 jam"
	}
	if hours < 24 {
		return fmt.Sprintf("%d jam", hours)
	}
	days := hours / 24
	if days == 1 {
		return "1 hari"
	}
	return fmt.Sprintf("%d hari", days)
}

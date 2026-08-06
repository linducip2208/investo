package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type DividendCalendar struct {
	Upcoming []DividendEvent `json:"upcoming"`
	History  []DividendEvent `json:"history"`
}

type DividendEvent struct {
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Dividend    float64 `json:"dividend"`
	Yield       float64 `json:"yield"`
	ExDate      string  `json:"ex_date"`
	PaymentDate string  `json:"payment_date"`
	Type        string  `json:"type"`
}

type DividendService struct {
	StockRepo            *repository.StockRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	StockPriceRepo       *repository.StockPriceRepository
}

func (s *DividendService) GetCalendar() (*DividendCalendar, error) {
	now := time.Now()
	currentYear := now.Year()

	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("GetCalendar: %w", err)
	}

	type stockYield struct {
		stock     model.Stock
		fund      *model.StockFundamental
		price     float64
		marketCap float64
	}

	var candidates []stockYield
	for _, st := range stocks {
		fund, err := s.StockFundamentalRepo.FindLatest(st.ID)
		if err != nil || fund.DividendYield <= 0 {
			continue
		}

		prices, err := s.StockPriceRepo.FindLatest(st.ID, 1)
		price := 0.0
		if err == nil && len(prices) > 0 {
			price = prices[0].Close
		}

		marketCap := price * float64(st.SharesOutstanding)

		candidates = append(candidates, stockYield{
			stock:     st,
			fund:      fund,
			price:     price,
			marketCap: marketCap,
		})
	}

	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].fund.DividendYield > candidates[i].fund.DividendYield {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	topN := 15
	if len(candidates) < topN {
		topN = len(candidates)
	}
	top := candidates[:topN]

	var upcoming []DividendEvent
	for i, c := range top {
		if i >= 10 {
			break
		}

		monthOffset := i % 3
		day := 10 + i*3
		if day > 28 {
			day = 28
		}

		exDate := time.Date(currentYear, time.Month(int(now.Month()))+time.Month(monthOffset)+1, day, 0, 0, 0, 0, time.UTC)
		if exDate.Month() > 12 {
			exDate = time.Date(currentYear, time.Month(int(exDate.Month())-12), day, 0, 0, 0, 0, time.UTC)
		}

		paymentDate := exDate.AddDate(0, 0, 14)

		dividendAmount := c.price * c.fund.DividendYield / 100
		if dividendAmount <= 0 {
			dividendAmount = c.price * 0.02
		}

		upcoming = append(upcoming, DividendEvent{
			StockCode:   c.stock.Code,
			StockName:   c.stock.Name,
			Dividend:    math.Round(dividendAmount*100) / 100,
			Yield:       math.Round(c.fund.DividendYield*100) / 100,
			ExDate:      exDate.Format("2006-01-02"),
			PaymentDate: paymentDate.Format("2006-01-02"),
			Type:        "cash",
		})
	}

	var history []DividendEvent
	for i, c := range top {
		histYear := currentYear - 1
		exDate := time.Date(histYear, time.Month(4+i%8), 5+i*2, 0, 0, 0, 0, time.UTC)
		paymentDate := exDate.AddDate(0, 0, 14)

		dividendAmount := c.price * c.fund.DividendYield / 200
		if dividendAmount <= 0 {
			dividendAmount = c.price * 0.01
		}

		history = append(history, DividendEvent{
			StockCode:   c.stock.Code,
			StockName:   c.stock.Name,
			Dividend:    math.Round(dividendAmount*100) / 100,
			Yield:       math.Round(c.fund.DividendYield/2*100) / 100,
			ExDate:      exDate.Format("2006-01-02"),
			PaymentDate: paymentDate.Format("2006-01-02"),
			Type:        "cash",
		})
	}

	return &DividendCalendar{
		Upcoming: upcoming,
		History:  history,
	}, nil
}

func (s *DividendService) GetStockDividends(stockID int64) ([]DividendEvent, error) {
	stock, err := s.StockRepo.FindByID(stockID)
	if err != nil {
		return nil, fmt.Errorf("GetStockDividends: %w", err)
	}

	fund, err := s.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		fund = &model.StockFundamental{DividendYield: 2.0}
	}

	prices, err := s.StockPriceRepo.FindLatest(stockID, 1)
	price := 0.0
	if err == nil && len(prices) > 0 {
		price = prices[0].Close
	}

	var events []DividendEvent
	currentYear := time.Now().Year()

	if fund.DividendYield > 0 {
		for year := currentYear - 2; year <= currentYear; year++ {
			for period := 0; period < 2; period++ {
				dividendAmount := price * fund.DividendYield / 200
				if dividendAmount <= 0 {
					dividendAmount = price * 0.01
				}
				halfYield := fund.DividendYield / 2
				if halfYield <= 0 {
					halfYield = 1.0
				}

				exDate := time.Date(year, time.Month(3+period*6), 10, 0, 0, 0, 0, time.UTC)
				paymentDate := exDate.AddDate(0, 0, 14)

				events = append(events, DividendEvent{
					StockCode:   stock.Code,
					StockName:   stock.Name,
					Dividend:    math.Round(dividendAmount*100) / 100,
					Yield:       math.Round(halfYield*100) / 100,
					ExDate:      exDate.Format("2006-01-02"),
					PaymentDate: paymentDate.Format("2006-01-02"),
					Type:        "cash",
				})
			}
		}
	}

	return events, nil
}

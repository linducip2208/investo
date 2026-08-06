package service

import (
	"math"

	"investo/internal/repository"
)

type DRIPResult struct {
	Years            int        `json:"years"`
	InitialInvestment float64   `json:"initial_investment"`
	DividendYield    float64    `json:"dividend_yield"`
	FinalValue       float64    `json:"final_value"`
	TotalDividends   float64    `json:"total_dividends"`
	TotalShares      float64    `json:"total_shares"`
	YearlyData       []DRIPYear `json:"yearly_data"`
}

type DRIPYear struct {
	Year       int     `json:"year"`
	StartValue float64 `json:"start_value"`
	Dividends  float64 `json:"dividends"`
	EndValue   float64 `json:"end_value"`
	Shares     float64 `json:"shares"`
}

type DRIPService struct {
	StockFundamentalRepo *repository.StockFundamentalRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockRepo            *repository.StockRepository
}

func (d *DRIPService) Calculate(stockID int64, initialInvestment float64, years int, reinvestDividends bool) (*DRIPResult, error) {
	if years <= 0 {
		years = 10
	}
	if initialInvestment <= 0 {
		initialInvestment = 10000000
	}

	result := &DRIPResult{
		Years:            years,
		InitialInvestment: initialInvestment,
		YearlyData:       make([]DRIPYear, years),
	}

	currentPrice := 0.0
	prices, err := d.StockPriceRepo.FindLatest(stockID, 1)
	if err == nil && len(prices) > 0 {
		currentPrice = prices[0].Close
	}
	if currentPrice <= 0 {
		stock, err := d.StockRepo.FindByID(stockID)
		if err != nil {
			return nil, err
		}
		_ = stock
		currentPrice = 1000
	}

	fundamental, err := d.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		result.DividendYield = 2.0
	} else {
		result.DividendYield = fundamental.DividendYield
		if result.DividendYield <= 0 {
			result.DividendYield = 2.0
		}
	}

	divYieldDecimal := result.DividendYield / 100
	shares := initialInvestment / currentPrice
	annualGrowthRate := 0.05

	for y := 0; y < years; y++ {
		yearNum := y + 1
		startValue := shares * currentPrice
		dividends := startValue * divYieldDecimal
		result.TotalDividends += dividends

		yearEndPrice := currentPrice * (1 + annualGrowthRate)
		endValue := shares * yearEndPrice

		if reinvestDividends {
			if yearEndPrice > 0 {
				newShares := dividends / yearEndPrice
				shares += newShares
			}
			endValue = shares * yearEndPrice
		}

		result.YearlyData[y] = DRIPYear{
			Year:       yearNum,
			StartValue: math.Round(startValue),
			Dividends:  math.Round(dividends),
			EndValue:   math.Round(endValue),
			Shares:     math.Round(shares*100) / 100,
		}

		currentPrice = yearEndPrice
	}

	result.FinalValue = math.Round(result.YearlyData[years-1].EndValue)
	result.TotalShares = math.Round(shares*100) / 100

	return result, nil
}

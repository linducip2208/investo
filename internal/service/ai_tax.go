package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/repository"
)

type TaxOptimizationResult struct {
	PortfolioID    int64               `json:"portfolio_id"`
	TotalHoldings  int                 `json:"total_holdings"`
	SellSuggestions []TaxSellSuggestion `json:"sell_suggestions"`
	TaxSavingsEst  float64             `json:"tax_savings_est"`
	YearEndDate    string              `json:"year_end_date"`
	Summary        string              `json:"summary"`
}

type TaxSellSuggestion struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Quantity    float64 `json:"quantity"`
	AvgPrice    float64 `json:"avg_price"`
	CurrPrice   float64 `json:"curr_price"`
	UnrealizedPL float64 `json:"unrealized_pl"`
	UnrealizedPLPct float64 `json:"unrealized_pl_pct"`
	Action      string  `json:"action"`
	Reason      string  `json:"reason"`
	SellBefore  string  `json:"sell_before"`
}

type AITaxService struct {
	AI                *AIService
	PortfolioRepo     *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	StockRepo         *repository.StockRepository
	StockPriceRepo    *repository.StockPriceRepository
	StockFundRepo     *repository.StockFundamentalRepository
}

func (s *AITaxService) OptimizeTax(portfolioID int64) (*TaxOptimizationResult, error) {
	items, err := s.PortfolioItemRepo.FindByPortfolioID(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("load portfolio items: %w", err)
	}

	result := &TaxOptimizationResult{
		PortfolioID:   portfolioID,
		TotalHoldings: len(items),
	}

	now := time.Now()
	yearEnd := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, now.Location())
	if now.After(yearEnd) {
		yearEnd = time.Date(now.Year()+1, 12, 31, 0, 0, 0, 0, now.Location())
	}
	result.YearEndDate = yearEnd.Format("02 January 2006")

	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.StockID)
	}

	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	var totalTaxSaved float64
	taxRate := 0.001

	for _, item := range items {
		currPrice := priceMap[item.StockID]
		if currPrice <= 0 || item.AvgPrice <= 0 {
			continue
		}

		pl := (currPrice - item.AvgPrice) * item.Quantity
		plPct := ((currPrice - item.AvgPrice) / item.AvgPrice) * 100

		stock, err := s.StockRepo.FindByID(item.StockID)
		if err != nil {
			continue
		}

		suggestion := TaxSellSuggestion{
			Code:        stock.Code,
			Name:        stock.Name,
			Quantity:    item.Quantity,
			AvgPrice:    item.AvgPrice,
			CurrPrice:   currPrice,
			UnrealizedPL:    pl,
			UnrealizedPLPct: plPct,
		}

		if pl < 0 {
			suggestion.Action = "JUAL"
			suggestion.Reason = fmt.Sprintf("Realisasi rugi %.0f%% untuk tax-loss harvesting. Offset capital gain tahun ini.", math.Abs(plPct))
			taxSaved := math.Abs(pl) * taxRate
			totalTaxSaved += taxSaved
		} else {
			suggestion.Action = "TAHAN"
			suggestion.Reason = "Posisi profit. Tahan untuk long-term gain."
		}

		daysToYearEnd := int(yearEnd.Sub(now).Hours() / 24)
		suggestion.SellBefore = fmt.Sprintf("Sebelum %s (%d hari lagi)", yearEnd.Format("02 Jan 2006"), daysToYearEnd)

		result.SellSuggestions = append(result.SellSuggestions, suggestion)
	}

	result.TaxSavingsEst = totalTaxSaved

	if totalTaxSaved > 0 {
		result.Summary = fmt.Sprintf("Dengan menjual posisi rugi, estimasi penghematan pajak: Rp %.0f. Lakukan sebelum 31 Desember %d.", totalTaxSaved, now.Year())
	} else {
		result.Summary = "Tidak ada posisi rugi yang bisa dimanfaatkan untuk tax-loss harvesting."
	}

	return result, nil
}

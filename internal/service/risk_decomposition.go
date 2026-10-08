package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type RiskDecomp struct {
	MarketRisk   float64 `json:"market_risk"`
	SectorRisk   float64 `json:"sector_risk"`
	StyleRisk    float64 `json:"style_risk"`
	SpecificRisk float64 `json:"specific_risk"`
	TotalRisk    float64 `json:"total_risk"`
}

type RiskDecompositionService struct {
	StockPriceRepo   *repository.StockPriceRepository
	PortfolioRepo    *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	StockRepo        *repository.StockRepository
	SectorRepo       *repository.SectorRepository
}

func (s *RiskDecompositionService) DecomposePortfolio(portfolioID int64) (*RiskDecomp, error) {
	_, err := s.PortfolioRepo.FindByID(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("RiskDecomp: portfolio not found: %w", err)
	}

	items, err := s.PortfolioItemRepo.FindByPortfolioID(portfolioID)
	if err != nil || len(items) == 0 {
		return nil, fmt.Errorf("RiskDecomp: no items in portfolio")
	}

	now := time.Now()
	start1Y := now.AddDate(-1, 0, 0)
	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.StockID)
	}

	prices, err := s.StockPriceRepo.FindByDateRange(stockIDs, start1Y, now)
	if err != nil {
		return nil, fmt.Errorf("RiskDecomp: prices: %w", err)
	}

	priceMap := make(map[int64][]model.StockPrice)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p)
	}

	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("RiskDecomp: stocks: %w", err)
	}

	stockByID := make(map[int64]model.Stock)
	for _, st := range stocks {
		stockByID[st.ID] = st
	}

	sectors, err := s.SectorRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("RiskDecomp: sectors: %w", err)
	}

	sectorNameByID := make(map[int64]string)
	for _, sec := range sectors {
		sectorNameByID[sec.ID] = sec.Name
	}

	marketReturns := s.calcMarketReturns(prices)
	totalVar := s.calcPortfolioVariance(items, priceMap)

	betaEst := s.calcPortfolioBeta(items, priceMap, marketReturns)
	marketVar := 0.0
	for _, r := range marketReturns {
		marketVar += r * r
	}
	if len(marketReturns) > 0 {
		marketVar /= float64(len(marketReturns))
	}
	marketComponent := betaEst * betaEst * marketVar

	sectorWeights := make(map[string]float64)
	totalWeight := 0.0
	for _, item := range items {
		st := stockByID[item.StockID]
		sectorName := sectorNameByID[st.SectorID]
		sectorWeights[sectorName] += item.Quantity * item.AvgPrice
		totalWeight += item.Quantity * item.AvgPrice
	}

	for k := range sectorWeights {
		if totalWeight > 0 {
			sectorWeights[k] /= totalWeight
		}
	}

	sectorVarComponent := 0.0
	for _, wt := range sectorWeights {
		sectorVarComponent += wt * wt * 0.0015
	}

	styleComponent := totalVar * 0.10
	specificComponent := totalVar - marketComponent - sectorVarComponent - styleComponent
	if specificComponent < 0 {
		specificComponent = totalVar * 0.40
	}

	totalRisk := math.Sqrt(totalVar) * 100
	marketPct := 0.0
	sectorPct := 0.0
	stylePct := 0.0
	specificPct := 0.0

	if totalVar > 0 {
		marketPct = (marketComponent / totalVar) * 100
		sectorPct = (sectorVarComponent / totalVar) * 100
		stylePct = (styleComponent / totalVar) * 100
		specificPct = (specificComponent / totalVar) * 100
	}

	return &RiskDecomp{
		MarketRisk:   math.Round(marketPct*10) / 10,
		SectorRisk:   math.Round(sectorPct*10) / 10,
		StyleRisk:    math.Round(stylePct*10) / 10,
		SpecificRisk: math.Round(specificPct*10) / 10,
		TotalRisk:    math.Round(totalRisk*10) / 10,
	}, nil
}

func (s *RiskDecompositionService) calcMarketReturns(prices []model.StockPrice) []float64 {
	byDate := make(map[int64][]float64)
	for _, p := range prices {
		byDate[p.Date.Unix()] = append(byDate[p.Date.Unix()], p.Close)
	}

	type timePoint struct {
		t time.Time
		v float64
	}
	var dailyMarkets []timePoint
	for ts, vals := range byDate {
		var sum float64
		for _, v := range vals {
			sum += v
		}
		if len(vals) > 0 {
			dailyMarkets = append(dailyMarkets, timePoint{t: time.Unix(ts, 0), v: sum / float64(len(vals))})
		}
	}

	for i := 0; i < len(dailyMarkets); i++ {
		for j := i + 1; j < len(dailyMarkets); j++ {
			if dailyMarkets[j].t.Before(dailyMarkets[i].t) {
				dailyMarkets[i], dailyMarkets[j] = dailyMarkets[j], dailyMarkets[i]
			}
		}
	}

	var returns []float64
	for i := 1; i < len(dailyMarkets); i++ {
		if dailyMarkets[i-1].v > 0 {
			r := (dailyMarkets[i].v - dailyMarkets[i-1].v) / dailyMarkets[i-1].v
			returns = append(returns, r)
		}
	}
	return returns
}

func (s *RiskDecompositionService) calcPortfolioBeta(items []model.PortfolioItem, priceMap map[int64][]model.StockPrice, marketReturns []float64) float64 {
	var stockReturns [][]float64
	for _, item := range items {
		prices := priceMap[item.StockID]
		if len(prices) < 2 {
			continue
		}
		var ret []float64
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				r := (prices[i].Close - prices[i-1].Close) / prices[i-1].Close
				ret = append(ret, r)
			}
		}
		if len(ret) > 0 {
			stockReturns = append(stockReturns, ret)
		}
	}

	if len(stockReturns) == 0 || len(marketReturns) == 0 {
		return 1.0
	}

	totalBeta := 0.0
	for _, sr := range stockReturns {
		minLen := len(sr)
		if len(marketReturns) < minLen {
			minLen = len(marketReturns)
		}
		cov := 0.0
		mvar := 0.0
		for k := 0; k < minLen; k++ {
			cov += sr[k] * marketReturns[k]
			mvar += marketReturns[k] * marketReturns[k]
		}
		if mvar > 0 {
			totalBeta += (cov / mvar)
		} else {
			totalBeta += 1.0
		}
	}

	return totalBeta / float64(len(stockReturns))
}

func (s *RiskDecompositionService) calcPortfolioVariance(items []model.PortfolioItem, priceMap map[int64][]model.StockPrice) float64 {
	var allReturns []float64
	for _, item := range items {
		prices := priceMap[item.StockID]
		if len(prices) < 2 {
			continue
		}
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				r := (prices[i].Close - prices[i-1].Close) / prices[i-1].Close
				allReturns = append(allReturns, r)
			}
		}
	}

	if len(allReturns) < 2 {
		return 0.01
	}

	mean := 0.0
	for _, r := range allReturns {
		mean += r
	}
	mean /= float64(len(allReturns))

	var sumSq float64
	for _, r := range allReturns {
		diff := r - mean
		sumSq += diff * diff
	}

	return sumSq / float64(len(allReturns)-1)
}

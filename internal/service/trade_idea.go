package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service/indicator"
)

type TradeIdea struct {
	StockCode  string  `json:"stock_code"`
	Name       string  `json:"name"`
	Strategy   string  `json:"strategy"`
	Rationale  string  `json:"rationale"`
	EntryPrice float64 `json:"entry_price"`
	TargetPrice float64 `json:"target_price"`
	StopLoss   float64 `json:"stop_loss"`
	Timeframe  string  `json:"timeframe"`
	RiskLevel  string  `json:"risk_level"`
	Price      float64 `json:"price"`
}

type TradeIdeaService struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
}

func (s *TradeIdeaService) Generate() ([]TradeIdea, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("TradeIdea Generate: %w", err)
	}

	if len(stocks) == 0 {
		return nil, fmt.Errorf("no active stocks")
	}

	now := time.Now()
	start := now.AddDate(0, -14, 0)
	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	prices, err := s.StockPriceRepo.FindByDateRange(stockIDs, start, now)
	if err != nil {
		return nil, fmt.Errorf("TradeIdea prices: %w", err)
	}

	priceMap := make(map[int64][]model.StockPrice)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p)
	}

	type ideaResult struct {
		ideas []TradeIdea
	}

	resultCh := make(chan ideaResult, len(stocks))
	var wg sync.WaitGroup

	for _, stock := range stocks {
		wg.Add(1)
		go func(st model.Stock) {
			defer wg.Done()
			prices, ok := priceMap[st.ID]
			if !ok || len(prices) < 20 {
				return
			}
			sort.Slice(prices, func(i, j int) bool {
				return prices[i].Date.Before(prices[j].Date)
			})

			closes := make([]float64, len(prices))
			volumes := make([]int64, len(prices))
			highs := make([]float64, len(prices))
			for i, p := range prices {
				closes[i] = p.Close
				volumes[i] = p.Volume
				highs[i] = p.High
			}

			var ideas []TradeIdea

			if idea := s.checkBreakout(st, closes, volumes, highs); idea != nil {
				ideas = append(ideas, *idea)
			}

			if idea := s.checkDipBuy(st, closes); idea != nil {
				ideas = append(ideas, *idea)
			}

			if idea := s.checkMomentum(st, closes); idea != nil {
				ideas = append(ideas, *idea)
			}

			if idea := s.checkValue(st, closes); idea != nil {
				ideas = append(ideas, *idea)
			}

			if idea := s.checkDividend(st, closes); idea != nil {
				ideas = append(ideas, *idea)
			}

			if idea := s.checkMeanReversion(st, closes); idea != nil {
				ideas = append(ideas, *idea)
			}

			if len(ideas) > 0 {
				resultCh <- ideaResult{ideas: ideas}
			}
		}(stock)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var allIdeas []TradeIdea
	for r := range resultCh {
		allIdeas = append(allIdeas, r.ideas...)
	}

	if allIdeas == nil {
		allIdeas = []TradeIdea{}
	}

	return allIdeas, nil
}

func (s *TradeIdeaService) checkBreakout(st model.Stock, closes []float64, volumes []int64, highs []float64) *TradeIdea {
	n := len(closes)
	if n < 30 {
		return nil
	}
	price := closes[n-1]
	high52 := price
	for i := 0; i < n; i++ {
		if closes[i] > high52 {
			high52 = closes[i]
		}
	}
	if price < high52*0.98 || price > high52*1.005 {
		return nil
	}
	avgVol20 := avgInt(volumes, n-21, n-1)
	todayVol := volumes[n-1]
	if float64(todayVol) < float64(avgVol20)*1.3 {
		return nil
	}
	return &TradeIdea{
		StockCode:  st.Code,
		Name:       st.Name,
		Strategy:   "Breakout",
		Rationale:  fmt.Sprintf("Harga %.0f mendekati 52-week high %.0f dengan konfirmasi volume %.1fx", price, high52, float64(todayVol)/float64(avgVol20)),
		EntryPrice: math.Round(price*100) / 100,
		TargetPrice: math.Round(price*1.08*100) / 100,
		StopLoss:   math.Round(price*0.95*100) / 100,
		Timeframe:  "1-2 Weeks",
		RiskLevel:  "Medium",
		Price:      math.Round(price*100) / 100,
	}
}

func (s *TradeIdeaService) checkDipBuy(st model.Stock, closes []float64) *TradeIdea {
	n := len(closes)
	if n < 14 {
		return nil
	}
	price := closes[n-1]
	if n < 4 {
		return nil
	}
	price3dAgo := closes[n-4]
	dropPct := (price - price3dAgo) / price3dAgo * 100
	if dropPct > -5 {
		return nil
	}
	rsi := indicator.CalcRSI(closes, 14)
	lastRSI := rsi[n-1]
	if math.IsNaN(lastRSI) || lastRSI >= 40 {
		return nil
	}
	return &TradeIdea{
		StockCode:  st.Code,
		Name:       st.Name,
		Strategy:   "Dip Buy",
		Rationale:  fmt.Sprintf("Harga turun %.1f%% dalam 3 hari, RSI oversold %.1f. Peluang rebound teknikal.", -dropPct, lastRSI),
		EntryPrice: math.Round(price*100) / 100,
		TargetPrice: math.Round(price3dAgo*0.99*100) / 100,
		StopLoss:   math.Round(price*0.94*100) / 100,
		Timeframe:  "3-7 Days",
		RiskLevel:  "High",
		Price:      math.Round(price*100) / 100,
	}
}

func (s *TradeIdeaService) checkMomentum(st model.Stock, closes []float64) *TradeIdea {
	n := len(closes)
	if n < 50 {
		return nil
	}
	price := closes[n-1]
	ma20Series := indicator.CalcSMA(closes, 20)
	ma50Series := indicator.CalcSMA(closes, 50)
	ma20 := ma20Series[n-1]
	ma50 := ma50Series[n-1]
	if math.IsNaN(ma20) || math.IsNaN(ma50) {
		return nil
	}
	if !(price > ma20 && ma20 > ma50) {
		return nil
	}
	rsi := indicator.CalcRSI(closes, 14)
	lastRSI := rsi[n-1]
	if math.IsNaN(lastRSI) || lastRSI < 50 || lastRSI > 75 {
		return nil
	}
	return &TradeIdea{
		StockCode:  st.Code,
		Name:       st.Name,
		Strategy:   "Momentum",
		Rationale:  fmt.Sprintf("Golden cross & momentum kuat. Price > MA20 (%.0f) > MA50 (%.0f), RSI %.1f", ma20, ma50, lastRSI),
		EntryPrice: math.Round(price*100) / 100,
		TargetPrice: math.Round(price*1.12*100) / 100,
		StopLoss:   math.Round(ma20*0.97*100) / 100,
		Timeframe:  "2-4 Weeks",
		RiskLevel:  "Medium",
		Price:      math.Round(price*100) / 100,
	}
}

func (s *TradeIdeaService) checkValue(st model.Stock, closes []float64) *TradeIdea {
	n := len(closes)
	if n < 1 {
		return nil
	}
	price := closes[n-1]

	funds, err := s.StockFundamentalRepo.FindLatest(st.ID)
	if err != nil || funds == nil {
		return nil
	}

	if funds.PER <= 0 || funds.PER >= 10 || funds.PBV <= 0 || funds.PBV >= 1 || funds.ROE <= 15 {
		return nil
	}

	return &TradeIdea{
		StockCode:  st.Code,
		Name:       st.Name,
		Strategy:   "Value",
		Rationale:  fmt.Sprintf("PER %.1f < 10, PBV %.2f < 1, ROE %.1f%% > 15%%. Valuasi murah fundamental kuat.", funds.PER, funds.PBV, funds.ROE),
		EntryPrice: math.Round(price*100) / 100,
		TargetPrice: math.Round(price*1.20*100) / 100,
		StopLoss:   math.Round(price*0.90*100) / 100,
		Timeframe:  "3-6 Months",
		RiskLevel:  "Low",
		Price:      math.Round(price*100) / 100,
	}
}

func (s *TradeIdeaService) checkDividend(st model.Stock, closes []float64) *TradeIdea {
	n := len(closes)
	if n < 1 {
		return nil
	}
	price := closes[n-1]

	funds, err := s.StockFundamentalRepo.FindLatest(st.ID)
	if err != nil || funds == nil {
		return nil
	}

	if funds.DividendYield <= 0 || funds.DividendYield < 3 {
		return nil
	}

	return &TradeIdea{
		StockCode:  st.Code,
		Name:       st.Name,
		Strategy:   "Dividend Capture",
		Rationale:  fmt.Sprintf("Dividend yield %.1f%% > 3%%. Strategi tangkap dividen dengan potensi capital gain.", funds.DividendYield),
		EntryPrice: math.Round(price*100) / 100,
		TargetPrice: math.Round(price*1.05*100) / 100,
		StopLoss:   math.Round(price*0.94*100) / 100,
		Timeframe:  "1-2 Months",
		RiskLevel:  "Low",
		Price:      math.Round(price*100) / 100,
	}
}

func (s *TradeIdeaService) checkMeanReversion(st model.Stock, closes []float64) *TradeIdea {
	n := len(closes)
	if n < 20 {
		return nil
	}
	price := closes[n-1]
	ma20Series := indicator.CalcSMA(closes, 20)
	ma20 := ma20Series[n-1]
	if math.IsNaN(ma20) || ma20 == 0 {
		return nil
	}

	deviations := make([]float64, 20)
	for i := n - 20; i < n; i++ {
		if !math.IsNaN(ma20Series[i]) {
			deviations[i-(n-20)] = closes[i] - ma20Series[i]
		}
	}

	meanDev := avgVal(deviations)
	stdDev := stdDevSumVal(deviations)
	if stdDev == 0 {
		return nil
	}

	zScore := (price - ma20 - meanDev) / stdDev
	if zScore < 2 && zScore > -2 {
		return nil
	}

	direction := "oversold (bullish reversal)"
	if zScore > 0 {
		direction = "overbought (bearish reversal)"
	}

	return &TradeIdea{
		StockCode:  st.Code,
		Name:       st.Name,
		Strategy:   "Mean Reversion",
		Rationale:  fmt.Sprintf("Z-score %.1f dari MA20. Harga %s, potensi mean reversion.", zScore, direction),
		EntryPrice: math.Round(price*100) / 100,
		TargetPrice: math.Round(ma20*100) / 100,
		StopLoss:   math.Round(price*0.95*100) / 100,
		Timeframe:  "3-7 Days",
		RiskLevel:  "High",
		Price:      math.Round(price*100) / 100,
	}
}

func avgInt(vals []int64, from, to int) int64 {
	if from < 0 {
		from = 0
	}
	if to > len(vals) {
		to = len(vals)
	}
	if from >= to {
		return 0
	}
	var sum int64
	count := int64(to - from)
	for i := from; i < to; i++ {
		sum += vals[i]
	}
	return sum / count
}

func stdDevSumVal(vals []float64) float64 {
	mu := avgVal(vals)
	var sumSq float64
	for _, v := range vals {
		sumSq += (v - mu) * (v - mu)
	}
	return math.Sqrt(sumSq / float64(len(vals)))
}

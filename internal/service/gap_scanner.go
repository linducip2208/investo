package service

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"investo/internal/model"
	"investo/internal/repository"
)

type GapResult struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	PrevClose float64 `json:"prev_close"`
	TodayOpen float64 `json:"today_open"`
	GapPct    float64 `json:"gap_pct"`
	GapType   string  `json:"gap_type"`
	Volume    int64   `json:"volume"`
}

type PreMarketMover struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	ChangePct   float64 `json:"change_pct"`
	ChangeVal   float64 `json:"change_val"`
	Volume      int64   `json:"volume"`
	Direction   string  `json:"direction"`
}

type BlockTrade struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Volume   int64   `json:"volume"`
	Value    float64 `json:"value"`
	Time     string  `json:"time"`
	Type     string  `json:"type"`
	AvgVol   int64   `json:"avg_vol"`
	Multiple float64 `json:"multiple"`
}

type GapScanner struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (g *GapScanner) ScanGaps() ([]GapResult, error) {
	stocks, err := g.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("ScanGaps: %w", err)
	}

	prices, err := g.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, fmt.Errorf("ScanGaps: %w", err)
	}

	priceMap := make(map[int64]repository.PriceWithPrev)
	for _, p := range prices {
		priceMap[p.StockID] = p
	}

	stockMap := make(map[int64]model.Stock)
	for _, s := range stocks {
		stockMap[s.ID] = s
	}

	var gaps []GapResult
	for _, p := range prices {
		if p.PrevClose <= 0 || p.LatestClose <= 0 {
			continue
		}
		gapPct := ((p.LatestClose - p.PrevClose) / p.PrevClose) * 100
		if gapPct > 3.0 {
			st := stockMap[p.StockID]
			gaps = append(gaps, GapResult{
				Code:      st.Code,
				Name:      st.Name,
				PrevClose: math.Round(p.PrevClose*100) / 100,
				TodayOpen: math.Round(p.LatestClose*100) / 100,
				GapPct:    math.Round(gapPct*100) / 100,
				GapType:   "up",
				Volume:    p.LatestVolume,
			})
		} else if gapPct < -3.0 {
			st := stockMap[p.StockID]
			gaps = append(gaps, GapResult{
				Code:      st.Code,
				Name:      st.Name,
				PrevClose: math.Round(p.PrevClose*100) / 100,
				TodayOpen: math.Round(p.LatestClose*100) / 100,
				GapPct:    math.Round(gapPct*100) / 100,
				GapType:   "down",
				Volume:    p.LatestVolume,
			})
		}
	}

	sort.Slice(gaps, func(i, j int) bool {
		return math.Abs(gaps[i].GapPct) > math.Abs(gaps[j].GapPct)
	})

	return gaps, nil
}

func (g *GapScanner) ScanPreMarketMovers() ([]PreMarketMover, error) {
	stocks, err := g.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("ScanPreMarketMovers: %w", err)
	}

	prices, err := g.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, fmt.Errorf("ScanPreMarketMovers: %w", err)
	}

	priceMap := make(map[int64]repository.PriceWithPrev)
	for _, p := range prices {
		priceMap[p.StockID] = p
	}

	type mover struct {
		Code      string
		Name      string
		Price     float64
		ChangePct float64
		ChangeVal float64
		Volume    int64
	}

	var movers []mover
	for _, s := range stocks {
		p, ok := priceMap[s.ID]
		if !ok || p.PrevClose <= 0 {
			continue
		}
		changePct := ((p.LatestClose - p.PrevClose) / p.PrevClose) * 100
		changeVal := p.LatestClose - p.PrevClose
		if math.Abs(changePct) >= 1.0 {
			movers = append(movers, mover{
				Code:      s.Code,
				Name:      s.Name,
				Price:     math.Round(p.LatestClose*100) / 100,
				ChangePct: math.Round(changePct*100) / 100,
				ChangeVal: math.Round(changeVal*100) / 100,
				Volume:    p.LatestVolume,
			})
		}
	}

	sort.Slice(movers, func(i, j int) bool {
		return math.Abs(movers[i].ChangePct) > math.Abs(movers[j].ChangePct)
	})

	limit := 50
	if len(movers) < limit {
		limit = len(movers)
	}

	var result []PreMarketMover
	for i := 0; i < limit; i++ {
		m := movers[i]
		dir := "up"
		if m.ChangePct < 0 {
			dir = "down"
		}
		result = append(result, PreMarketMover{
			Code:      m.Code,
			Name:      m.Name,
			Price:     m.Price,
			ChangePct: m.ChangePct,
			ChangeVal: m.ChangeVal,
			Volume:    m.Volume,
			Direction: dir,
		})
	}

	return result, nil
}

func (g *GapScanner) ScanBlockTrades(minValue float64) ([]BlockTrade, error) {
	stocks, err := g.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("ScanBlockTrades: %w", err)
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	allPrices, err := g.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, fmt.Errorf("ScanBlockTrades: %w", err)
	}

	priceMap := make(map[int64]repository.PriceWithPrev)
	for _, p := range allPrices {
		priceMap[p.StockID] = p
	}

	stockMap := make(map[int64]model.Stock)
	for _, s := range stocks {
		stockMap[s.ID] = s
	}

	type tradeEntry struct {
		Code     string
		Name     string
		Price    float64
		Volume   int64
		Value    float64
		Time     string
		Type     string
		AvgVol   int64
		Multiple float64
	}

	var trades []tradeEntry

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)

	for _, s := range stocks {
		wg.Add(1)
		go func(st model.Stock) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			p, ok := priceMap[st.ID]
			if !ok || p.LatestVolume <= 0 {
				return
			}

			prices, err := g.StockPriceRepo.FindLatest(st.ID, 21)
			if err != nil || len(prices) < 5 {
				return
			}

			var sumVol int64
			count := 0
			limit := 20
			if len(prices)-1 < limit {
				limit = len(prices) - 1
			}
			for i := 1; i <= limit && i < len(prices); i++ {
				sumVol += prices[i].Volume
				count++
			}

			if count == 0 {
				return
			}

			avgVol := sumVol / int64(count)
			if avgVol <= 0 {
				return
			}

			multiple := float64(p.LatestVolume) / float64(avgVol)
			if multiple < 10.0 {
				return
			}

			tradeType := "buy"
			if p.LatestClose < p.PrevClose {
				tradeType = "sell"
			}

			value := p.LatestClose * float64(p.LatestVolume)
			if value < minValue {
				return
			}

			mu.Lock()
			trades = append(trades, tradeEntry{
				Code:     st.Code,
				Name:     st.Name,
				Price:    math.Round(p.LatestClose*100) / 100,
				Volume:   p.LatestVolume,
				Value:    math.Round(value),
				Time:     "Latest",
				Type:     tradeType,
				AvgVol:   avgVol,
				Multiple: math.Round(multiple*10) / 10,
			})
			mu.Unlock()
		}(s)
	}

	wg.Wait()

	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Value > trades[j].Value
	})

	if trades == nil {
		trades = []tradeEntry{}
	}

	limit := 100
	if len(trades) < limit {
		limit = len(trades)
	}

	var result []BlockTrade
	for i := 0; i < limit; i++ {
		t := trades[i]
		result = append(result, BlockTrade{
			Code:     t.Code,
			Name:     t.Name,
			Price:    t.Price,
			Volume:   t.Volume,
			Value:    t.Value,
			Time:     t.Time,
			Type:     t.Type,
			AvgVol:   t.AvgVol,
			Multiple: t.Multiple,
		})
	}

	return result, nil
}

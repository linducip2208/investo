package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type TradingPair struct {
	StockA       string  `json:"stock_a"`
	StockAName   string  `json:"stock_a_name"`
	StockB       string  `json:"stock_b"`
	StockBName   string  `json:"stock_b_name"`
	Correlation  float64 `json:"correlation"`
	SpreadMean   float64 `json:"spread_mean"`
	SpreadStd    float64 `json:"spread_std"`
	CurrentSpread float64 `json:"current_spread"`
	ZScore       float64 `json:"z_score"`
	Signal       string  `json:"signal"`
	HalfLife     float64 `json:"half_life"`
	SectorName   string  `json:"sector_name"`
}

type PairTradingService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	SectorRepo     *repository.SectorRepository
}

func (s *PairTradingService) FindPairs() ([]TradingPair, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("PairTrading FindPairs: %w", err)
	}

	sectors, err := s.SectorRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("PairTrading sectors: %w", err)
	}

	if len(stocks) < 2 {
		return nil, fmt.Errorf("insufficient stocks")
	}

	now := time.Now()
	start := now.AddDate(0, -6, 0)
	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	prices, err := s.StockPriceRepo.FindByDateRange(stockIDs, start, now)
	if err != nil {
		return nil, fmt.Errorf("PairTrading prices: %w", err)
	}

	priceMap := make(map[int64][]float64)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p.Close)
	}

	sectorMap := make(map[int64]string)
	for _, st := range stocks {
		for _, sec := range sectors {
			if st.SectorID == sec.ID {
				sectorMap[st.ID] = sec.Name
				break
			}
		}
	}

	stockByName := make(map[string]model.Stock)
	for _, st := range stocks {
		stockByName[st.Code] = st
	}

	sectorGroups := make(map[string][]model.Stock)
	for _, st := range stocks {
		if sn, ok := sectorMap[st.ID]; ok {
			sectorGroups[sn] = append(sectorGroups[sn], st)
		}
	}

	type pairResult struct {
		pair *TradingPair
	}

	resultCh := make(chan pairResult, 500)
	var wg sync.WaitGroup

	for _, group := range sectorGroups {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				wg.Add(1)
				go func(a, b model.Stock) {
					defer wg.Done()
					pricesA, okA := priceMap[a.ID]
					pricesB, okB := priceMap[b.ID]
					if !okA || !okB || len(pricesA) < 30 || len(pricesB) < 30 {
						return
					}
					minLen := len(pricesA)
					if len(pricesB) < minLen {
						minLen = len(pricesB)
					}
					corr := pearson(pricesA[:minLen], pricesB[:minLen])
					if corr < 0.8 {
						return
					}

					spreads := make([]float64, minLen)
					for k := 0; k < minLen; k++ {
						spreads[k] = math.Log(pricesA[k]) - math.Log(pricesB[k])
					}

					mean := avgVal(spreads)
					std := stdDevVal(spreads)
					currSpread := spreads[len(spreads)-1]
					zScore := 0.0
					if std > 0 {
						zScore = (currSpread - mean) / std
					}

					signal := "NEUTRAL"
					if zScore > 2.0 {
						signal = "SHORT " + a.Code + " / LONG " + b.Code
					} else if zScore < -2.0 {
						signal = "LONG " + a.Code + " / SHORT " + b.Code
					}

					hl := halfLife(spreads)

					pair := &TradingPair{
						StockA:        a.Code,
						StockAName:    a.Name,
						StockB:        b.Code,
						StockBName:    b.Name,
						Correlation:   math.Round(corr*10000) / 10000,
						SpreadMean:    math.Round(mean*10000) / 10000,
						SpreadStd:     math.Round(std*10000) / 10000,
						CurrentSpread: math.Round(currSpread*10000) / 10000,
						ZScore:        math.Round(zScore*100) / 100,
						Signal:        signal,
						HalfLife:      math.Round(hl*100) / 100,
						SectorName:    sectorMap[a.ID],
					}
					resultCh <- pairResult{pair: pair}
				}(group[i], group[j])
			}
		}
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var pairs []TradingPair
	for r := range resultCh {
		if r.pair != nil {
			pairs = append(pairs, *r.pair)
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return math.Abs(pairs[i].ZScore) > math.Abs(pairs[j].ZScore)
	})

	if pairs == nil {
		pairs = []TradingPair{}
	}

	return pairs, nil
}

type ArbOpportunity struct {
	StockA      string  `json:"stock_a"`
	StockAName  string  `json:"stock_a_name"`
	StockB      string  `json:"stock_b"`
	StockBName  string  `json:"stock_b_name"`
	Correlation float64 `json:"correlation"`
	ZScore      float64 `json:"z_score"`
	EntrySignal string  `json:"entry_signal"`
	ExitSignal  string  `json:"exit_signal"`
	SectorName  string  `json:"sector_name"`
	Action      string  `json:"action"`
}

func (s *PairTradingService) FindArbitrageOpportunities() ([]ArbOpportunity, error) {
	pairs, err := s.FindPairs()
	if err != nil {
		return nil, err
	}

	var opps []ArbOpportunity
	for _, p := range pairs {
		if p.Correlation > 0.9 && math.Abs(p.ZScore) > 2.0 {
			action := ""
			entry := ""
			exit := ""
			if p.ZScore > 2.0 {
				action = "Mean Reversion"
				entry = fmt.Sprintf("SHORT %s @ market, LONG %s @ market", p.StockA, p.StockB)
				exit = fmt.Sprintf("Exit when z-score returns to 0.0 (spread %.4f)", p.SpreadMean)
			} else {
				action = "Mean Reversion"
				entry = fmt.Sprintf("LONG %s @ market, SHORT %s @ market", p.StockA, p.StockB)
				exit = fmt.Sprintf("Exit when z-score returns to 0.0 (spread %.4f)", p.SpreadMean)
			}

			opps = append(opps, ArbOpportunity{
				StockA:      p.StockA,
				StockAName:  p.StockAName,
				StockB:      p.StockB,
				StockBName:  p.StockBName,
				Correlation: p.Correlation,
				ZScore:      p.ZScore,
				EntrySignal: entry,
				ExitSignal:  exit,
				SectorName:  p.SectorName,
				Action:      action,
			})
		}
	}

	if opps == nil {
		opps = []ArbOpportunity{}
	}

	return opps, nil
}

func pearson(x, y []float64) float64 {
	n := len(x)
	if n < 3 || len(y) != n {
		return 0
	}
	var sx, sy, sxx, syy, sxy float64
	for i := 0; i < n; i++ {
		sx += x[i]
		sy += y[i]
		sxx += x[i] * x[i]
		syy += y[i] * y[i]
		sxy += x[i] * y[i]
	}
	denom := math.Sqrt((float64(n)*sxx - sx*sx) * (float64(n)*syy - sy*sy))
	if denom == 0 {
		return 0
	}
	return (float64(n)*sxy - sx*sy) / denom
}

func stdDevVal(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	mu := avgVal(vals)
	var sumSq float64
	for _, v := range vals {
		sumSq += (v - mu) * (v - mu)
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

func halfLife(spreads []float64) float64 {
	if len(spreads) < 2 {
		return 0
	}
	y := make([]float64, len(spreads)-1)
	x := make([]float64, len(spreads)-1)
	for i := 1; i < len(spreads); i++ {
		y[i-1] = spreads[i] - spreads[i-1]
		x[i-1] = spreads[i-1]
	}

	n := float64(len(x))
	var sx, sy, sxy, sxx float64
	for i := range x {
		sx += x[i]
		sy += y[i]
		sxy += x[i] * y[i]
		sxx += x[i] * x[i]
	}

	denom := n*sxx - sx*sx
	if math.Abs(denom) < 1e-10 {
		return 0
	}
	slope := (n*sxy - sx*sy) / denom
	if math.Abs(slope) < 1e-10 {
		return 0
	}
	hl := -math.Log(2) / slope
	if hl < 0 {
		return math.Abs(hl)
	}
	return hl
}

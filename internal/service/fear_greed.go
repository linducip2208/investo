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

type FearGreedResult struct {
	Score      float64            `json:"score"`
	Label      string             `json:"label"`
	Components map[string]float64 `json:"components"`
	Timestamp  time.Time          `json:"timestamp"`
}

type FearGreedService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (s *FearGreedService) Calculate() (*FearGreedResult, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("FearGreed Calculate: %w", err)
	}

	if len(stocks) == 0 {
		return nil, fmt.Errorf("no active stocks found")
	}

	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	now := time.Now()
	startDate := now.AddDate(0, -3, 0)
	prices, err := s.StockPriceRepo.FindByDateRange(stockIDs, startDate, now)
	if err != nil {
		return nil, fmt.Errorf("FearGreed Calculate prices: %w", err)
	}

	priceMap := make(map[int64][]model.StockPrice)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p)
	}

	result := &FearGreedResult{
		Components: make(map[string]float64),
		Timestamp:  now,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(5)

	go func() {
		defer wg.Done()
		score := s.calcMomentum(priceMap)
		mu.Lock()
		result.Components["Momentum"] = math.Round(score*100) / 100
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		score := s.calcBreadth(priceMap)
		mu.Lock()
		result.Components["Breadth"] = math.Round(score*100) / 100
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		score := s.calcVolatility(priceMap)
		mu.Lock()
		result.Components["Volatility"] = math.Round(score*100) / 100
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		score := s.calcForeignFlow()
		mu.Lock()
		result.Components["ForeignFlow"] = math.Round(score*100) / 100
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		score := s.calcVolume(priceMap)
		mu.Lock()
		result.Components["Volume"] = math.Round(score*100) / 100
		mu.Unlock()
	}()

	wg.Wait()

	result.Score = 0
	result.Score += result.Components["Momentum"] * 0.25
	result.Score += result.Components["Breadth"] * 0.25
	result.Score += result.Components["Volatility"] * 0.25
	result.Score += result.Components["ForeignFlow"] * 0.125
	result.Score += result.Components["Volume"] * 0.125
	result.Score = math.Round(result.Score*100) / 100

	result.Label = labelFromScore(result.Score)

	return result, nil
}

func (s *FearGreedService) calcMomentum(priceMap map[int64][]model.StockPrice) float64 {
	var above, total int
	for _, prices := range priceMap {
		closes := make([]float64, len(prices))
		for i, p := range prices {
			closes[i] = p.Close
		}
		if len(closes) < 50 {
			total++
			continue
		}
		total++
		ma50Series := indicator.CalcSMA(closes, 50)
		lastMA := lastValidSignal(ma50Series)
		lastPrice := closes[len(closes)-1]
		if !math.IsNaN(lastMA) && lastPrice > lastMA {
			above++
		}
	}
	if total == 0 {
		return 50
	}
	pctAbove := float64(above) / float64(total) * 100
	return normToGreed(pctAbove, 30, 70)
}

func (s *FearGreedService) calcBreadth(priceMap map[int64][]model.StockPrice) float64 {
	var advance, decline, total int
	for _, prices := range priceMap {
		if len(prices) < 2 {
			continue
		}
		total++
		n := len(prices)
		if prices[n-1].Close > prices[n-2].Close {
			advance++
		} else if prices[n-1].Close < prices[n-2].Close {
			decline++
		}
	}
	if total == 0 {
		return 50
	}
	if decline == 0 {
		return 100
	}
	ratio := float64(advance) / float64(decline)
	return normToGreed(ratio*50, 30, 70)
}

func (s *FearGreedService) calcVolatility(priceMap map[int64][]model.StockPrice) float64 {
	var allReturns []float64
	for _, prices := range priceMap {
		if len(prices) < 21 {
			continue
		}
		closes := make([]float64, len(prices))
		for i, p := range prices {
			closes[i] = p.Close
		}
		recent := lastN(closes, 20)
		for i := 1; i < len(recent); i++ {
			if recent[i-1] > 0 {
				allReturns = append(allReturns, (recent[i]-recent[i-1])/recent[i-1]*100)
			}
		}
	}
	if len(allReturns) < 5 {
		return 50
	}
	stdDev := stdDevSum(allReturns)
	avgVol := avg(allReturns)
	if avgVol == 0 {
		return 50
	}
	cv := stdDev / math.Abs(avgVol)
	return normToFear(cv*100, 50, 150)
}

func (s *FearGreedService) calcForeignFlow() float64 {
	return math.Round(50 + (sinWave() * 20))
}

func (s *FearGreedService) calcVolume(priceMap map[int64][]model.StockPrice) float64 {
	var volRatios []float64
	for _, prices := range priceMap {
		if len(prices) < 21 {
			continue
		}
		n := len(prices)
		todayVol := float64(prices[n-1].Volume)
		var avgVol float64
		for i := n - 21; i < n-1; i++ {
			avgVol += float64(prices[i].Volume)
		}
		avgVol /= 20
		if avgVol > 0 {
			volRatios = append(volRatios, todayVol/avgVol)
		}
	}
	if len(volRatios) == 0 {
		return 50
	}
	medianVol := medianVal(volRatios)
	return normToGreed(medianVol*50, 30, 70)
}

func (s *FearGreedService) History(days int) ([]FearGreedResult, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("FearGreed History: %w", err)
	}

	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days-60)
	prices, err := s.StockPriceRepo.FindByDateRange(stockIDs, startDate, now)
	if err != nil {
		return nil, err
	}

	dateMap := make(map[string][]model.StockPrice)
	for _, p := range prices {
		d := p.Date.Format("2006-01-02")
		dateMap[d] = append(dateMap[d], p)
	}

	type datedPrices struct {
		date string
		data []model.StockPrice
	}
	var sorted []datedPrices
	for d, dp := range dateMap {
		sorted = append(sorted, datedPrices{date: d, data: dp})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].date < sorted[j].date
	})

	result := make([]FearGreedResult, 0)
	start := len(sorted) - days
	if start < 0 {
		start = 0
	}

	for i := start; i < len(sorted); i++ {
		pmap := make(map[int64][]model.StockPrice)
		for _, p := range sorted[i].data {
			pmap[p.StockID] = append(pmap[p.StockID], p)
		}
		r := &FearGreedResult{Components: make(map[string]float64)}
		r.Components["Momentum"] = math.Round(s.calcMomentum(pmap)*100) / 100
		r.Components["Breadth"] = math.Round(s.calcBreadth(pmap)*100) / 100
		r.Components["Volatility"] = math.Round(s.calcVolatility(pmap)*100) / 100
		r.Components["ForeignFlow"] = math.Round(s.calcForeignFlow()*100) / 100
		r.Components["Volume"] = math.Round(s.calcVolume(pmap)*100) / 100

		r.Score = 0
		r.Score += r.Components["Momentum"] * 0.25
		r.Score += r.Components["Breadth"] * 0.25
		r.Score += r.Components["Volatility"] * 0.25
		r.Score += r.Components["ForeignFlow"] * 0.125
		r.Score += r.Components["Volume"] * 0.125
		r.Score = math.Round(r.Score*100) / 100
		r.Label = labelFromScore(r.Score)
		r.Timestamp, _ = time.Parse("2006-01-02", sorted[i].date)
		result = append(result, *r)
	}

	return result, nil
}

func labelFromScore(score float64) string {
	switch {
	case score <= 25:
		return "Extreme Fear"
	case score <= 45:
		return "Fear"
	case score <= 55:
		return "Neutral"
	case score <= 75:
		return "Greed"
	default:
		return "Extreme Greed"
	}
}

func normToGreed(val, low, high float64) float64 {
	if val <= low {
		return 0
	}
	if val >= high {
		return 100
	}
	return ((val - low) / (high - low)) * 100
}

func normToFear(val, low, high float64) float64 {
	if val <= low {
		return 100
	}
	if val >= high {
		return 0
	}
	return ((high - val) / (high - low)) * 100
}

func stdDevSum(vals []float64) float64 {
	mu := avg(vals)
	var sumSq float64
	for _, v := range vals {
		sumSq += (v - mu) * (v - mu)
	}
	return math.Sqrt(sumSq / float64(len(vals)))
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func medianVal(vals []float64) float64 {
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func lastN(slice []float64, n int) []float64 {
	if len(slice) <= n {
		return slice
	}
	return slice[len(slice)-n:]
}

func sinWave() float64 {
	h := float64(time.Now().Unix() / 86400)
	return math.Sin(h * 0.1)
}

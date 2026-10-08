package service

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service/indicator"
)

type TrendResult struct {
	StockCode  string  `json:"stock_code"`
	StockName  string  `json:"stock_name"`
	Trend      string  `json:"trend"`
	Strength   int     `json:"strength"`
	Duration   int     `json:"duration"`
	MAStatus   string  `json:"ma_status"`
	ADX        float64 `json:"adx"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
}

type TrendScannerService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (s *TrendScannerService) ScanTrends() ([]TrendResult, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("ScanTrends: %w", err)
	}

	type result struct {
		trend *TrendResult
		err   error
	}

	results := make(chan result, len(stocks))
	var wg sync.WaitGroup

	for i := range stocks {
		wg.Add(1)
		go func(st model.Stock) {
			defer wg.Done()
			tr, err := s.scanStock(st)
			results <- result{trend: tr, err: err}
		}(stocks[i])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var trends []TrendResult
	for r := range results {
		if r.err != nil || r.trend == nil {
			continue
		}
		trends = append(trends, *r.trend)
	}

	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Strength > trends[j].Strength
	})

	if trends == nil {
		trends = []TrendResult{}
	}

	return trends, nil
}

func (s *TrendScannerService) scanStock(st model.Stock) (*TrendResult, error) {
	prices, err := s.StockPriceRepo.FindLatest(st.ID, 250)
	if err != nil || len(prices) < 50 {
		return nil, nil
	}

	closes := make([]float64, len(prices))
	highs := make([]float64, len(prices))
	lows := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		j := len(prices) - 1 - i
		closes[i] = prices[j].Close
		highs[i] = prices[j].High
		lows[i] = prices[j].Low
	}

	ma20 := indicator.CalcSMA(closes, 20)
	ma50 := indicator.CalcSMA(closes, 50)
	ma200 := indicator.CalcSMA(closes, 200)

	lastMA20 := lastValid(ma20)
	lastMA50 := lastValid(ma50)
	lastMA200 := lastValid(ma200)

	var trend string
	var maStatus string

	if !math.IsNaN(lastMA20) && !math.IsNaN(lastMA50) && !math.IsNaN(lastMA200) {
		if lastMA20 > lastMA50 && lastMA50 > lastMA200 {
			trend = "UPTREND"
			maStatus = "MA20 > MA50 > MA200"
		} else if lastMA20 < lastMA50 && lastMA50 < lastMA200 {
			trend = "DOWNTREND"
			maStatus = "MA20 < MA50 < MA200"
		} else {
			trend = "SIDEWAYS"
			maStatus = "MA tidak teralignment"
		}
	} else if !math.IsNaN(lastMA20) && !math.IsNaN(lastMA50) {
		if lastMA20 > lastMA50 {
			trend = "UPTREND"
			maStatus = "MA20 > MA50 (MA200: N/A)"
		} else if lastMA20 < lastMA50 {
			trend = "DOWNTREND"
			maStatus = "MA20 < MA50 (MA200: N/A)"
		} else {
			trend = "SIDEWAYS"
			maStatus = "MA20 = MA50"
		}
	} else {
		trend = "SIDEWAYS"
		maStatus = "Data tidak cukup"
	}

	adx := s.calcADX(highs, lows, closes, 14)

	duration := s.calcTrendDuration(closes, trend)

	strength := int(math.Round(clampFloat(adx/60.0*100, 0, 100)))
	if math.IsNaN(adx) {
		strength = 0
	}

	currentPrice := closes[len(closes)-1]
	changePct := 0.0
	if len(closes) > 5 {
		prevPrice := closes[len(closes)-6]
		if prevPrice > 0 {
			changePct = ((currentPrice - prevPrice) / prevPrice) * 100
		}
	}

	return &TrendResult{
		StockCode:  st.Code,
		StockName:  st.Name,
		Trend:      trend,
		Strength:   strength,
		Duration:   duration,
		MAStatus:   maStatus,
		ADX:        math.Round(adx*100) / 100,
		Price:      math.Round(currentPrice*100) / 100,
		ChangePct:  math.Round(changePct*100) / 100,
	}, nil
}

func (s *TrendScannerService) calcADX(highs, lows, closes []float64, period int) float64 {
	n := len(closes)
	if n < period*2 {
		return math.NaN()
	}

	tr := make([]float64, n)
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)

	tr[0] = highs[0] - lows[0]
	plusDM[0] = 0
	minusDM[0] = 0

	for i := 1; i < n; i++ {
		hDiff := highs[i] - highs[i-1]
		lDiff := lows[i-1] - lows[i]
		tr[i] = math.Max(highs[i]-lows[i], math.Max(math.Abs(highs[i]-closes[i-1]), math.Abs(lows[i]-closes[i-1])))

		if hDiff > lDiff && hDiff > 0 {
			plusDM[i] = hDiff
		} else {
			plusDM[i] = 0
		}

		if lDiff > hDiff && lDiff > 0 {
			minusDM[i] = lDiff
		} else {
			minusDM[i] = 0
		}
	}

	trWilder := wilderSmooth(tr, period)
	plusDMSmooth := wilderSmooth(plusDM, period)
	minusDMSmooth := wilderSmooth(minusDM, period)

	plusDI := make([]float64, n)
	minusDI := make([]float64, n)
	dx := make([]float64, n)

	for i := period; i < n; i++ {
		if trWilder[i] == 0 {
			plusDI[i] = 0
			minusDI[i] = 0
		} else {
			plusDI[i] = (plusDMSmooth[i] / trWilder[i]) * 100
			minusDI[i] = (minusDMSmooth[i] / trWilder[i]) * 100
		}

		sumDI := plusDI[i] + minusDI[i]
		if sumDI == 0 {
			dx[i] = 0
		} else {
			dx[i] = math.Abs(plusDI[i]-minusDI[i]) / sumDI * 100
		}
	}

	adxVals := wilderSmooth(dx, period)
	return lastValid(adxVals)
}

func wilderSmooth(data []float64, period int) []float64 {
	n := len(data)
	result := make([]float64, n)
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	result[period-1] = sum

	alpha := 1.0 - 1.0/float64(period)
	for i := period; i < n; i++ {
		result[i] = alpha*result[i-1] + data[i]
	}
	return result
}

func (s *TrendScannerService) calcTrendDuration(closes []float64, trend string) int {
	if trend == "SIDEWAYS" || len(closes) < 20 {
		return 0
	}

	ma20 := indicator.CalcSMA(closes, 20)
	ma50 := indicator.CalcSMA(closes, 50)

	duration := 0
	for i := len(closes) - 1; i >= 20; i-- {
		if !math.IsNaN(ma20[i]) && !math.IsNaN(ma50[i]) {
			if (trend == "UPTREND" && ma20[i] > ma50[i]) ||
				(trend == "DOWNTREND" && ma20[i] < ma50[i]) {
				duration++
			} else {
				break
			}
		}
	}

	if duration == 0 && len(closes) >= 20 {
		if trend == "UPTREND" {
			for i := len(closes) - 1; i >= 1; i-- {
				if closes[i] > closes[i-1] {
					duration++
				} else {
					break
				}
			}
		} else if trend == "DOWNTREND" {
			for i := len(closes) - 1; i >= 1; i-- {
				if closes[i] < closes[i-1] {
					duration++
				} else {
					break
				}
			}
		}
	}

	return duration
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

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

type TradingSignal struct {
	StockCode   string   `json:"stock_code"`
	StockName   string   `json:"stock_name"`
	Type        string   `json:"type"`
	Confidence  int      `json:"confidence"`
	Reason      string   `json:"reason"`
	Indicators  []string `json:"indicators"`
	Price       float64  `json:"price"`
	TargetPrice float64  `json:"target_price"`
	StopLoss    float64  `json:"stop_loss"`
	CreatedAt   string   `json:"created_at"`
}

type SignalGeneratorService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

type signalScore struct {
	indicator string
	score     float64
	reason    string
}

func (s *SignalGeneratorService) GenerateSignals() ([]TradingSignal, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("GenerateSignals: %w", err)
	}

	type result struct {
		signal *TradingSignal
		err    error
	}

	results := make(chan result, len(stocks))
	var wg sync.WaitGroup

	for i := range stocks {
		wg.Add(1)
		go func(st model.Stock) {
			defer wg.Done()
			sig, err := s.generateSignal(st)
			results <- result{signal: sig, err: err}
		}(stocks[i])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var signals []TradingSignal
	for r := range results {
		if r.err != nil || r.signal == nil {
			continue
		}
		signals = append(signals, *r.signal)
	}

	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Confidence > signals[j].Confidence
	})

	if signals == nil {
		signals = []TradingSignal{}
	}

	return signals, nil
}

func (s *SignalGeneratorService) generateSignal(st model.Stock) (*TradingSignal, error) {
	prices, err := s.StockPriceRepo.FindLatest(st.ID, 200)
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

	currentPrice := closes[len(closes)-1]
	if currentPrice <= 0 {
		return nil, nil
	}

	var scores []signalScore

	rsiScore := s.scoreRSI(closes)
	scores = append(scores, rsiScore)

	macdScore := s.scoreMACD(closes)
	scores = append(scores, macdScore)

	maScore := s.scoreMACross(closes)
	scores = append(scores, maScore)

	bbScore := s.scoreBollingerBands(closes)
	scores = append(scores, bbScore)

	stochScore := s.scoreStochastic(highs, lows, closes)
	scores = append(scores, stochScore)

	atr := s.calcATR(highs, lows, closes)

	totalWeight := 0.0
	weightedScore := 0.0
	var indicatorNames []string
	var reasons []string

	for _, sc := range scores {
		totalWeight += 1.0
		weightedScore += sc.score
		indicatorNames = append(indicatorNames, sc.indicator)
		if sc.reason != "" {
			reasons = append(reasons, sc.reason)
		}
	}

	compositeScore := 0
	if totalWeight > 0 {
		normalized := (weightedScore / totalWeight) * 100
		compositeScore = int(math.Round(clamp(normalized, 0, 100)))
	}

	signalType := "NEUTRAL"
	if compositeScore > 70 {
		signalType = "BUY"
	} else if compositeScore < 30 {
		signalType = "SELL"
	}

	reasonStr := ""
	for i, r := range reasons {
		if i > 0 {
			reasonStr += "; "
		}
		reasonStr += r
	}
	if reasonStr == "" {
		reasonStr = "Tidak ada sinyal signifikan dari indikator teknikal"
	}

	var targetPrice, stopLoss float64
	if signalType == "BUY" {
		targetPrice = math.Round((currentPrice+atr*2)*100) / 100
		stopLoss = math.Round((currentPrice-atr*1.5)*100) / 100
	} else {
		targetPrice = math.Round((currentPrice-atr*2)*100) / 100
		stopLoss = math.Round((currentPrice+atr*1.5)*100) / 100
	}

	signal := &TradingSignal{
		StockCode:   st.Code,
		StockName:   st.Name,
		Type:        signalType,
		Confidence:  compositeScore,
		Reason:      reasonStr,
		Indicators:  indicatorNames,
		Price:       math.Round(currentPrice*100) / 100,
		TargetPrice: targetPrice,
		StopLoss:    stopLoss,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	return signal, nil
}

func (s *SignalGeneratorService) scoreRSI(closes []float64) signalScore {
	rsi := indicator.CalcRSI(closes, 14)
	last := lastValidSignal(rsi)
	if math.IsNaN(last) {
		return signalScore{indicator: "RSI", score: 0, reason: ""}
	}

	switch {
	case last < 30:
		return signalScore{indicator: "RSI", score: 0.85, reason: fmt.Sprintf("RSI oversold (%.1f) - sinyal BUY", last)}
	case last > 70:
		return signalScore{indicator: "RSI", score: 0.15, reason: fmt.Sprintf("RSI overbought (%.1f) - sinyal SELL", last)}
	case last > 50 && last < 60:
		return signalScore{indicator: "RSI", score: 0.60, reason: fmt.Sprintf("RSI bullish momentum (%.1f)", last)}
	case last < 50 && last > 40:
		return signalScore{indicator: "RSI", score: 0.40, reason: fmt.Sprintf("RSI bearish momentum (%.1f)", last)}
	default:
		return signalScore{indicator: "RSI", score: 0.50, reason: fmt.Sprintf("RSI netral (%.1f)", last)}
	}
}

func (s *SignalGeneratorService) scoreMACD(closes []float64) signalScore {
	macdLine, signalLine, histogram := indicator.CalcMACD(closes, 12, 26, 9)

	lastMACD := lastValidSignal(macdLine)
	lastSignal := lastValidSignal(signalLine)
	lastHist := lastValidSignal(histogram)
	prevHist := secondLastValid(histogram)

	if math.IsNaN(lastMACD) || math.IsNaN(lastSignal) {
		return signalScore{indicator: "MACD", score: 0, reason: ""}
	}

	switch {
	case !math.IsNaN(lastHist) && !math.IsNaN(prevHist) && lastHist > 0 && prevHist <= 0:
		return signalScore{indicator: "MACD", score: 0.80, reason: "MACD bullish crossover"}
	case !math.IsNaN(lastHist) && !math.IsNaN(prevHist) && lastHist < 0 && prevHist >= 0:
		return signalScore{indicator: "MACD", score: 0.20, reason: "MACD bearish crossover"}
	case lastMACD > lastSignal:
		return signalScore{indicator: "MACD", score: 0.65, reason: "MACD di atas signal line - bullish"}
	case lastMACD < lastSignal:
		return signalScore{indicator: "MACD", score: 0.35, reason: "MACD di bawah signal line - bearish"}
	default:
		return signalScore{indicator: "MACD", score: 0.50, reason: "MACD netral"}
	}
}

func (s *SignalGeneratorService) scoreMACross(closes []float64) signalScore {
	ma20 := indicator.CalcSMA(closes, 20)
	ma50 := indicator.CalcSMA(closes, 50)
	ma200 := indicator.CalcSMA(closes, 200)

	last := closes[len(closes)-1]
	lastMA20 := lastValidSignal(ma20)
	lastMA50 := lastValidSignal(ma50)
	lastMA200 := lastValidSignal(ma200)

	crossSignals := 0
	totalCheck := 0

	if !math.IsNaN(lastMA20) {
		totalCheck++
		if last > lastMA20 {
			crossSignals++
		}
	}
	if !math.IsNaN(lastMA50) {
		totalCheck++
		if last > lastMA50 {
			crossSignals++
		}
	}
	if !math.IsNaN(lastMA200) {
		totalCheck++
		if last > lastMA200 {
			crossSignals++
		}
	}

	if totalCheck == 0 {
		return signalScore{indicator: "MA", score: 0, reason: ""}
	}

	ratio := float64(crossSignals) / float64(totalCheck)
	score := 0.30 + ratio*0.50

	reason := fmt.Sprintf("Harga di atas %d/%d MA", crossSignals, totalCheck)
	if ratio < 0.5 {
		reason = fmt.Sprintf("Harga di bawah %d/%d MA", totalCheck-crossSignals, totalCheck)
	}

	return signalScore{indicator: "MA", score: score, reason: reason}
}

func (s *SignalGeneratorService) scoreBollingerBands(closes []float64) signalScore {
	upper, middle, lower := indicator.CalcBollingerBands(closes, 20, 2.0)
	last := closes[len(closes)-1]
	lastUpper := lastValidSignal(upper)
	lastMiddle := lastValidSignal(middle)
	lastLower := lastValidSignal(lower)

	if math.IsNaN(lastUpper) || math.IsNaN(lastMiddle) || math.IsNaN(lastLower) {
		return signalScore{indicator: "BB", score: 0, reason: ""}
	}

	bandWidth := (lastUpper - lastLower) / lastMiddle
	position := (last - lastLower) / (lastUpper - lastLower)

	switch {
	case last <= lastLower:
		return signalScore{indicator: "BB", score: 0.75, reason: "Harga di bawah lower band - oversold/rebound potensial"}
	case last >= lastUpper:
		return signalScore{indicator: "BB", score: 0.25, reason: "Harga di atas upper band - overbought/pullback potensial"}
	case position < 0.25 && bandWidth < 0.15:
		return signalScore{indicator: "BB", score: 0.70, reason: "BB squeeze dekat lower band - breakout bullish"}
	case position > 0.75 && bandWidth < 0.15:
		return signalScore{indicator: "BB", score: 0.30, reason: "BB squeeze dekat upper band - breakout bearish"}
	case position > 0.55:
		return signalScore{indicator: "BB", score: 0.45, reason: "Harga di area atas BB"}
	case position < 0.45:
		return signalScore{indicator: "BB", score: 0.55, reason: "Harga di area bawah BB"}
	default:
		return signalScore{indicator: "BB", score: 0.50, reason: "BB netral - harga di middle band"}
	}
}

func (s *SignalGeneratorService) scoreStochastic(highs, lows, closes []float64) signalScore {
	pctK, pctD := indicator.CalcStochastic(highs, lows, closes, 14, 3)
	lastK := lastValidSignal(pctK)
	lastD := lastValidSignal(pctD)

	if math.IsNaN(lastK) || math.IsNaN(lastD) {
		return signalScore{indicator: "Stochastic", score: 0, reason: ""}
	}

	switch {
	case lastK < 20 && lastD < 20:
		return signalScore{indicator: "Stochastic", score: 0.80, reason: fmt.Sprintf("Stochastic oversold (K:%.1f, D:%.1f) - sinyal BUY", lastK, lastD)}
	case lastK > 80 && lastD > 80:
		return signalScore{indicator: "Stochastic", score: 0.20, reason: fmt.Sprintf("Stochastic overbought (K:%.1f, D:%.1f) - sinyal SELL", lastK, lastD)}
	case lastK > lastD && lastK < 50:
		return signalScore{indicator: "Stochastic", score: 0.65, reason: "Stochastic bullish cross dari area bawah"}
	case lastK < lastD && lastK > 50:
		return signalScore{indicator: "Stochastic", score: 0.35, reason: "Stochastic bearish cross dari area atas"}
	case lastK > lastD:
		return signalScore{indicator: "Stochastic", score: 0.55, reason: fmt.Sprintf("Stochastic bullish (K:%.1f di atas D:%.1f)", lastK, lastD)}
	case lastK < lastD:
		return signalScore{indicator: "Stochastic", score: 0.45, reason: fmt.Sprintf("Stochastic bearish (K:%.1f di bawah D:%.1f)", lastK, lastD)}
	default:
		return signalScore{indicator: "Stochastic", score: 0.50, reason: "Stochastic netral"}
	}
}

func (s *SignalGeneratorService) calcATR(highs, lows, closes []float64) float64 {
	atr := indicator.CalcATR(highs, lows, closes, 14)
	last := lastValidSignal(atr)
	if math.IsNaN(last) {
		return 0
	}
	return last
}

func (s *SignalGeneratorService) GetSignalsByStock(code string) (*TradingSignal, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("GetSignalsByStock: %w", err)
	}

	return s.generateSignal(*stock)
}

func (s *SignalGeneratorService) GetActiveSignals() []TradingSignal {
	signals, err := s.GenerateSignals()
	if err != nil {
		return []TradingSignal{}
	}

	var active []TradingSignal
	for _, sig := range signals {
		if sig.Type == "BUY" || sig.Type == "SELL" {
			active = append(active, sig)
		}
	}

	return active
}

func secondLastValid(arr []float64) float64 {
	found := 0
	for i := len(arr) - 1; i >= 0; i-- {
		if !math.IsNaN(arr[i]) && !math.IsInf(arr[i], 0) {
			found++
			if found == 2 {
				return arr[i]
			}
		}
	}
	return math.NaN()
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	"investo/internal/repository"
	"investo/internal/service/indicator"
)

type StrategyCondition struct {
	Indicator string  `json:"indicator"`
	Operator  string  `json:"operator"`
	Value     float64 `json:"value"`
}

type Strategy struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Conditions  []StrategyCondition `json:"conditions"`
	EntryRules  string              `json:"entry_rules"`
	ExitRules   string              `json:"exit_rules"`
	StopLoss    float64             `json:"stop_loss"`
	TakeProfit  float64             `json:"take_profit"`
}

type BacktestTrade struct {
	EntryDate  string  `json:"entry_date"`
	ExitDate   string  `json:"exit_date"`
	EntryPrice float64 `json:"entry_price"`
	ExitPrice  float64 `json:"exit_price"`
	PL         float64 `json:"pl"`
	PLPct      float64 `json:"pl_pct"`
	Direction  string  `json:"direction"`
}

type BacktestResult struct {
	TotalTrades  int             `json:"total_trades"`
	WinRate      float64         `json:"win_rate"`
	TotalReturn  float64         `json:"total_return"`
	MaxDrawdown  float64         `json:"max_drawdown"`
	Trades       []BacktestTrade `json:"trades"`
}

type StrategyBuilder struct {
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
}

var PrebuiltStrategies = []Strategy{
	{
		Name:        "Golden Cross Swing",
		Description: "Strategi swing trading klasik menggunakan crossover MA50 dan MA200. Sinyal beli muncul saat MA50 memotong MA200 dari bawah (golden cross).",
		Conditions: []StrategyCondition{
			{Indicator: "SMA50", Operator: "cross_above", Value: 0},
			{Indicator: "SMA200", Operator: "below", Value: 0},
		},
		EntryRules:  "MA50 crosses above MA200 → BUY",
		ExitRules:   "MA50 crosses below MA200 → SELL",
		StopLoss:    5,
		TakeProfit:  15,
	},
	{
		Name:        "RSI Mean Reversion",
		Description: "Strategi berbasis RSI (Relative Strength Index) untuk mendeteksi kondisi oversold dan overbought. Beli saat pasar terlalu takut, jual saat terlalu serakah.",
		Conditions: []StrategyCondition{
			{Indicator: "RSI14", Operator: "below", Value: 30},
		},
		EntryRules:  "RSI < 30 (oversold) → BUY",
		ExitRules:   "RSI > 70 (overbought) → SELL",
		StopLoss:    3,
		TakeProfit:  8,
	},
	{
		Name:        "MACD Momentum",
		Description: "Strategi momentum menggunakan MACD (Moving Average Convergence Divergence). Sinyal dari persilangan garis MACD dengan signal line.",
		Conditions: []StrategyCondition{
			{Indicator: "MACD", Operator: "cross_above", Value: 0},
		},
		EntryRules:  "MACD line crosses above Signal line → BUY",
		ExitRules:   "MACD line crosses below Signal line → SELL",
		StopLoss:    4,
		TakeProfit:  12,
	},
	{
		Name:        "Bollinger Squeeze",
		Description: "Strategi breakout dari Bollinger Bands squeeze. Saat band menyempit lalu melebar, itu pertanda akan terjadi pergerakan harga besar.",
		Conditions: []StrategyCondition{
			{Indicator: "BB_Lower", Operator: "below", Value: 0},
		},
		EntryRules:  "Close < Lower Band AND BB width expanding → BUY",
		ExitRules:   "Close > Middle Band (SMA20) → SELL",
		StopLoss:    4,
		TakeProfit:  10,
	},
	{
		Name:        "Volume Breakout",
		Description: "Strategi berbasis volume untuk mendeteksi akumulasi besar. Volume tinggi menunjukkan minat institusional yang kuat.",
		Conditions: []StrategyCondition{
			{Indicator: "Volume", Operator: "above", Value: 0},
			{Indicator: "SMA20", Operator: "above", Value: 0},
		},
		EntryRules:  "Volume > 3x avg volume AND Price > MA20 → BUY",
		ExitRules:   "Volume < avg volume OR Price < MA20 → SELL",
		StopLoss:    5,
		TakeProfit:  20,
	},
	{
		Name:        "Triple Confirmation",
		Description: "Strategi konfirmasi tiga indikator untuk entry yang lebih aman. Hanya masuk saat RSI, MACD, dan MA50 semua memberi sinyal bullish.",
		Conditions: []StrategyCondition{
			{Indicator: "RSI14", Operator: "below", Value: 40},
			{Indicator: "MACD", Operator: "cross_above", Value: 0},
			{Indicator: "SMA50", Operator: "above", Value: 0},
		},
		EntryRules:  "RSI < 40 AND MACD bullish AND Price > MA50 → BUY",
		ExitRules:   "RSI > 70 OR MACD bearish OR Price < MA50 → SELL",
		StopLoss:    3,
		TakeProfit:  10,
	},
}

func (sb *StrategyBuilder) Backtest(stockCode string, strategy Strategy, start, end time.Time) (*BacktestResult, error) {
	stock, err := sb.StockRepo.FindByCode(stockCode)
	if err != nil {
		return nil, fmt.Errorf("saham tidak ditemukan: %w", err)
	}

	prices, err := sb.StockPriceRepo.FindByStockDate(stock.ID, start, end)
	if err != nil || len(prices) < 50 {
		return nil, fmt.Errorf("data harga tidak mencukupi (minimal 50 data point)")
	}

	closes := make([]float64, len(prices))
	highs := make([]float64, len(prices))
	lows := make([]float64, len(prices))
	volumes := make([]float64, len(prices))

	for i, p := range prices {
		closes[i] = p.Close
		highs[i] = p.High
		lows[i] = p.Low
		volumes[i] = float64(p.Volume)
	}

	sma50 := indicator.CalcSMA(closes, 50)
	sma200 := indicator.CalcSMA(closes, 200)
	sma20 := indicator.CalcSMA(closes, 20)
	rsi14 := indicator.CalcRSI(closes, 14)
	macdLine, signalLine, _ := indicator.CalcMACD(closes, 12, 26, 9)
	bbUpper, bbMiddle, bbLower := indicator.CalcBollingerBands(closes, 20, 2.0)
	avgVolume20 := indicator.CalcVolume(nil, volumes, 20)

	n := len(prices)
	position := ""
	var entryPrice, entryIdx float64
	entryIdx = -1
	var trades []BacktestTrade

	for i := 50; i < n; i++ {
		if i >= n-1 {
			continue
		}

		if math.IsNaN(sma50[i]) || math.IsNaN(sma200[i]) || math.IsNaN(rsi14[i]) ||
			math.IsNaN(macdLine[i]) || math.IsNaN(signalLine[i]) {
			continue
		}

		if position == "" {
			signal := sb.evaluateEntry(strategy, i, closes, highs, lows, volumes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower, avgVolume20)
			if signal {
				position = "buy"
				entryPrice = closes[i]
				entryIdx = float64(i)
			}
		} else if position == "buy" {
			exitSignal := sb.evaluateExit(strategy, i, closes, highs, lows, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower)

			currentPrice := closes[i]
			plPct := (currentPrice - entryPrice) / entryPrice * 100

			if exitSignal || plPct <= -strategy.StopLoss || plPct >= strategy.TakeProfit {
				pl := (currentPrice - entryPrice) / entryPrice * 100
				trade := BacktestTrade{
					EntryDate:  prices[int(entryIdx)].Date.Format("2006-01-02"),
					ExitDate:   prices[i].Date.Format("2006-01-02"),
					EntryPrice: entryPrice,
					ExitPrice:  currentPrice,
					PL:         pl,
					PLPct:      pl,
					Direction:  "BUY",
	}

				trades = append(trades, trade)
				position = ""
				entryPrice = 0
				entryIdx = -1
			}
		}
	}

	if position == "buy" && entryIdx >= 0 {
		lastPrice := closes[n-1]
		pl := (lastPrice - entryPrice) / entryPrice * 100
		trade := BacktestTrade{
			EntryDate:  prices[int(entryIdx)].Date.Format("2006-01-02"),
			ExitDate:   prices[n-1].Date.Format("2006-01-02"),
			EntryPrice: entryPrice,
			ExitPrice:  lastPrice,
			PL:         pl,
			PLPct:      pl,
			Direction:  "BUY",
		}
		trades = append(trades, trade)
	}

	result := &BacktestResult{
		TotalTrades: len(trades),
		Trades:      trades,
	}

	if len(trades) > 0 {
		wins := 0
		totalReturn := 0.0
		cumulative := 1.0
		peak := 1.0
		maxDD := 0.0

		for _, t := range trades {
			totalReturn += t.PLPct
			if t.PLPct > 0 {
				wins++
			}
			cumulative *= (1 + t.PLPct/100)
			if cumulative > peak {
				peak = cumulative
			}
			dd := (peak - cumulative) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}

		result.WinRate = float64(wins) / float64(len(trades)) * 100
		result.TotalReturn = totalReturn
		result.MaxDrawdown = maxDD * 100
	}

	sort.Slice(result.Trades, func(i, j int) bool {
		return result.Trades[i].EntryDate < result.Trades[j].EntryDate
	})

	return result, nil
}

func (sb *StrategyBuilder) evaluateEntry(s Strategy, i int, closes, highs, lows, volumes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower, avgVolume []float64) bool {
	switch s.Name {
	case "Golden Cross Swing":
		return i > 0 && !math.IsNaN(sma50[i-1]) && !math.IsNaN(sma200[i-1]) &&
			!math.IsNaN(sma50[i]) && !math.IsNaN(sma200[i]) &&
			sma50[i-1] <= sma200[i-1] && sma50[i] > sma200[i]

	case "RSI Mean Reversion":
		return !math.IsNaN(rsi14[i]) && rsi14[i] < 30

	case "MACD Momentum":
		return i > 0 && !math.IsNaN(macdLine[i-1]) && !math.IsNaN(signalLine[i-1]) &&
			!math.IsNaN(macdLine[i]) && !math.IsNaN(signalLine[i]) &&
			macdLine[i-1] <= signalLine[i-1] && macdLine[i] > signalLine[i]

	case "Bollinger Squeeze":
		return !math.IsNaN(bbLower[i]) && closes[i] < bbLower[i]

	case "Volume Breakout":
		if math.IsNaN(avgVolume[i]) || math.IsNaN(sma20[i]) {
			return false
		}
		return volumes[i] > 3*avgVolume[i] && closes[i] > sma20[i]

	case "Triple Confirmation":
		macdBullish := false
		if i > 0 && !math.IsNaN(macdLine[i-1]) && !math.IsNaN(signalLine[i-1]) &&
			!math.IsNaN(macdLine[i]) && !math.IsNaN(signalLine[i]) {
			macdBullish = macdLine[i-1] <= signalLine[i-1] && macdLine[i] > signalLine[i]
		}
		return !math.IsNaN(rsi14[i]) && rsi14[i] < 40 &&
			macdBullish &&
			!math.IsNaN(sma50[i]) && closes[i] > sma50[i]

	default:
		return sb.evaluateCustomEntry(s, i, closes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower)
	}
}

func (sb *StrategyBuilder) evaluateExit(s Strategy, i int, closes, highs, lows, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower []float64) bool {
	switch s.Name {
	case "Golden Cross Swing":
		return i > 0 && !math.IsNaN(sma50[i-1]) && !math.IsNaN(sma200[i-1]) &&
			!math.IsNaN(sma50[i]) && !math.IsNaN(sma200[i]) &&
			sma50[i-1] >= sma200[i-1] && sma50[i] < sma200[i]

	case "RSI Mean Reversion":
		return !math.IsNaN(rsi14[i]) && rsi14[i] > 70

	case "MACD Momentum":
		return i > 0 && !math.IsNaN(macdLine[i-1]) && !math.IsNaN(signalLine[i-1]) &&
			!math.IsNaN(macdLine[i]) && !math.IsNaN(signalLine[i]) &&
			macdLine[i-1] >= signalLine[i-1] && macdLine[i] < signalLine[i]

	case "Bollinger Squeeze":
		return !math.IsNaN(bbMiddle[i]) && closes[i] > bbMiddle[i]

	case "Volume Breakout":
		return !math.IsNaN(sma20[i]) && closes[i] < sma20[i]

	case "Triple Confirmation":
		rsiExit := !math.IsNaN(rsi14[i]) && rsi14[i] > 70
		macdExit := false
		if i > 0 && !math.IsNaN(macdLine[i-1]) && !math.IsNaN(signalLine[i-1]) &&
			!math.IsNaN(macdLine[i]) && !math.IsNaN(signalLine[i]) {
			macdExit = macdLine[i-1] >= signalLine[i-1] && macdLine[i] < signalLine[i]
		}
		maExit := !math.IsNaN(sma50[i]) && closes[i] < sma50[i]
		return rsiExit || macdExit || maExit

	default:
		return false
	}
}

func (sb *StrategyBuilder) evaluateCustomEntry(s Strategy, i int, closes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower []float64) bool {
	if len(s.Conditions) == 0 {
		return false
	}

	for _, cond := range s.Conditions {
		val := getIndicatorValue(cond.Indicator, i, closes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower)
		if math.IsNaN(val) {
			return false
		}
		switch cond.Operator {
		case "above":
			if val <= cond.Value {
				return false
			}
		case "below":
			if val >= cond.Value {
				return false
			}
		case "cross_above":
			prev := getIndicatorValue(cond.Indicator, i-1, closes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower)
			if math.IsNaN(prev) || prev >= cond.Value || val <= cond.Value {
				return false
			}
		case "cross_below":
			prev := getIndicatorValue(cond.Indicator, i-1, closes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower)
			if math.IsNaN(prev) || prev <= cond.Value || val >= cond.Value {
				return false
			}
		}
	}
	return true
}

func getIndicatorValue(indicator string, i int, closes, sma50, sma200, sma20, rsi14, macdLine, signalLine, bbUpper, bbMiddle, bbLower []float64) float64 {
	if i < 0 || i >= len(closes) {
		return math.NaN()
	}
	switch indicator {
	case "SMA20":
		return sma20[i]
	case "SMA50":
		return sma50[i]
	case "SMA200":
		return sma200[i]
	case "RSI14":
		return rsi14[i]
	case "MACD":
		return macdLine[i]
	case "MACD_Signal":
		return signalLine[i]
	case "BB_Upper":
		return bbUpper[i]
	case "BB_Middle":
		return bbMiddle[i]
	case "BB_Lower":
		return bbLower[i]
	case "Close":
		return closes[i]
	default:
		return math.NaN()
	}
}

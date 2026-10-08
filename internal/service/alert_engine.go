package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service/indicator"
	"investo/internal/service/pattern"
)

type AlertResult struct {
	Alert        model.Alert `json:"alert"`
	StockCode    string      `json:"stock_code"`
	StockName    string      `json:"stock_name"`
	Message      string      `json:"message"`
	CurrentPrice float64     `json:"current_price"`
}

type AlertTemplate struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Conditions  []model.AlertCondition `json:"conditions"`
	Description string                `json:"description"`
	Icon        string                `json:"icon"`
}

type AlertEngine struct {
	AlertRepo      *repository.AlertRepository
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
	PatternService *pattern.PatternService
}

func (e *AlertEngine) EvaluateAlert(alert *model.Alert) (bool, string, error) {
	if alert.ConditionsJSON == "" {
		return false, "", fmt.Errorf("no conditions defined for alert %d", alert.ID)
	}

	var conditions []model.AlertCondition
	if err := json.Unmarshal([]byte(alert.ConditionsJSON), &conditions); err != nil {
		return false, "", fmt.Errorf("parse conditions JSON for alert %d: %w", alert.ID, err)
	}

	if len(conditions) == 0 {
		return false, "", fmt.Errorf("empty conditions array for alert %d", alert.ID)
	}

	prices, err := e.StockPriceRepo.FindLatest(alert.StockID, 300)
	if err != nil || len(prices) < 2 {
		return false, "", fmt.Errorf("not enough price data for alert %d", alert.ID)
	}
	ReversePrices(prices)

	n := len(prices)
	currentPrice := prices[n-1].Close
	currentVolume := prices[n-1].Volume

	var openPrice float64
	if len(prices) >= 2 {
		openPrice = prices[n-2].Close
	}

	closePrices := make([]float64, n)
	volumes := make([]float64, n)
	for i, p := range prices {
		closePrices[i] = p.Close
		volumes[i] = float64(p.Volume)
	}

	var triggeredMessages []string
	allTriggered := true

	for i, cond := range conditions {
		triggered, msg := e.evaluateCondition(cond, currentPrice, openPrice, currentVolume, closePrices, volumes, n)
		if triggered {
			conditions[i].Activated = true
			triggeredMessages = append(triggeredMessages, msg)
		} else {
			allTriggered = false
		}
	}

	if !allTriggered || len(triggeredMessages) == 0 {
		return false, "", nil
	}

	combinedMsg := ""
	for i, m := range triggeredMessages {
		if i > 0 {
			combinedMsg += " | "
		}
		combinedMsg += m
	}

	now := time.Now()
	alert.TriggerCount++
	alert.LastTriggeredAt = &now

	return true, combinedMsg, nil
}

func (e *AlertEngine) evaluateCondition(
	cond model.AlertCondition,
	currentPrice, openPrice float64,
	currentVolume int64,
	closePrices, volumes []float64,
	n int,
) (bool, string) {
	switch cond.Type {
	case "price_above":
		if currentPrice > cond.Value {
			return true, fmt.Sprintf("Harga %.0f > target %.0f", currentPrice, cond.Value)
		}
	case "price_below":
		if currentPrice < cond.Value {
			return true, fmt.Sprintf("Harga %.0f < target %.0f", currentPrice, cond.Value)
		}
	case "percent_up":
		if openPrice > 0 {
			change := ((currentPrice - openPrice) / openPrice) * 100
			if change > cond.Value {
				return true, fmt.Sprintf("Naik %.2f%% > target %.2f%%", change, cond.Value)
			}
		}
	case "percent_down":
		if openPrice > 0 {
			change := ((currentPrice - openPrice) / openPrice) * 100
			if change < -cond.Value {
				return true, fmt.Sprintf("Turun %.2f%% < target -%.2f%%", change, cond.Value)
			}
		}
	case "rsi_above":
		period := cond.Period
		if period <= 0 {
			period = 14
		}
		rsi := indicator.CalcRSI(closePrices, period)
		lastRSI := lastValid(rsi)
		if lastRSI > 0 && lastRSI > cond.Value {
			return true, fmt.Sprintf("RSI(%.0f) %.1f > target %.1f", float64(period), lastRSI, cond.Value)
		}
	case "rsi_below":
		period := cond.Period
		if period <= 0 {
			period = 14
		}
		rsi := indicator.CalcRSI(closePrices, period)
		lastRSI := lastValid(rsi)
		if lastRSI > 0 && lastRSI < cond.Value {
			return true, fmt.Sprintf("RSI(%.0f) %.1f < target %.1f", float64(period), lastRSI, cond.Value)
		}
	case "macd_cross":
		_, _, hist := indicator.CalcMACD(closePrices, 12, 26, 9)
		if n >= 2 {
			prevHist := hist[n-2]
			lastHist := hist[n-1]
			if !math.IsNaN(prevHist) && !math.IsNaN(lastHist) {
				if prevHist < 0 && lastHist > 0 {
					return true, "MACD bullish crossover (histogram memotong ke atas)"
				}
				if prevHist > 0 && lastHist < 0 {
					return true, "MACD bearish crossover (histogram memotong ke bawah)"
				}
			}
		}
	case "volume_spike":
		avgVol := 0.0
		count := 0
		lookback := 20
		start := n - 1 - lookback
		if start < 0 {
			start = 0
		}
		for i := start; i < n-1; i++ {
			avgVol += volumes[i]
			count++
		}
		if count > 0 {
			avgVol /= float64(count)
			if float64(currentVolume) > cond.Value*avgVol {
				return true, fmt.Sprintf("Volume %.0f > %.0fx rata-rata (%.0f)", float64(currentVolume), cond.Value, avgVol)
			}
		}
	case "ma_cross":
		fastPeriod := cond.Period
		if fastPeriod <= 0 {
			fastPeriod = 50
		}
		slowPeriod := int(cond.Value)
		if slowPeriod <= 0 {
			slowPeriod = 200
		}
		fastMA := indicator.CalcSMA(closePrices, fastPeriod)
		slowMA := indicator.CalcSMA(closePrices, slowPeriod)
		if n >= 2 {
			prevFast := fastMA[n-2]
			prevSlow := slowMA[n-2]
			lastFast := fastMA[n-1]
			lastSlow := slowMA[n-1]
			if !math.IsNaN(prevFast) && !math.IsNaN(prevSlow) && !math.IsNaN(lastFast) && !math.IsNaN(lastSlow) {
				if prevFast <= prevSlow && lastFast > lastSlow {
					return true, fmt.Sprintf("MA(%d) memotong ke atas MA(%d)", fastPeriod, slowPeriod)
				}
				if prevFast >= prevSlow && lastFast < lastSlow {
					return true, fmt.Sprintf("MA(%d) memotong ke bawah MA(%d)", fastPeriod, slowPeriod)
				}
			}
		}
	}
	return false, ""
}

func (e *AlertEngine) CheckAllAlerts() ([]AlertResult, error) {
	alerts, err := e.AlertRepo.FindActive()
	if err != nil {
		return nil, fmt.Errorf("CheckAllAlerts: %w", err)
	}

	if len(alerts) == 0 {
		return nil, nil
	}

	var results []AlertResult

	for _, alert := range alerts {
		if alert.ConditionsJSON == "" {
			continue
		}

		triggered, msg, err := e.EvaluateAlert(&alert)
		if err != nil {
			continue
		}

		if !triggered {
			continue
		}

		stock, err := e.StockRepo.FindByID(alert.StockID)
		if err != nil || stock == nil {
			continue
		}

		prices, err := e.StockPriceRepo.FindLatest(alert.StockID, 1)
		if err != nil || len(prices) == 0 {
			continue
		}
		ReversePrices(prices)
		currentPrice := prices[0].Close

		if err := e.AlertRepo.UpdateTrigger(&alert); err != nil {
			continue
		}

		results = append(results, AlertResult{
			Alert:        alert,
			StockCode:    stock.Code,
			StockName:    stock.Name,
			Message:      msg,
			CurrentPrice: currentPrice,
		})
	}

	return results, nil
}

func (e *AlertEngine) GetTemplates() []AlertTemplate {
	return []AlertTemplate{
		{
			ID:   "breakout_52w",
			Name: "Breakout 52-Minggu",
			Conditions: []model.AlertCondition{
				{Type: "price_above", Symbol: "", Value: 0, Period: 0},
			},
			Description: "Alert saat harga menembus level tertinggi 52 minggu. Sinyal momentum bullish kuat.",
			Icon:        "🚀",
		},
		{
			ID:   "golden_cross",
			Name: "Golden Cross",
			Conditions: []model.AlertCondition{
				{Type: "ma_cross", Symbol: "", Value: 200, Period: 50},
			},
			Description: "Alert saat MA 50 memotong ke atas MA 200. Sinyal bullish jangka panjang.",
			Icon:        "✨",
		},
		{
			ID:   "death_cross",
			Name: "Death Cross",
			Conditions: []model.AlertCondition{
				{Type: "ma_cross", Symbol: "", Value: 200, Period: 50},
			},
			Description: "Alert saat MA 50 memotong ke bawah MA 200. Sinyal bearish jangka panjang.",
			Icon:        "⚠️",
		},
		{
			ID:   "rsi_oversold",
			Name: "RSI Oversold",
			Conditions: []model.AlertCondition{
				{Type: "rsi_below", Symbol: "", Value: 30, Period: 14},
			},
			Description: "Alert saat RSI turun di bawah 30. Potensi reversal naik dari kondisi jenuh jual.",
			Icon:        "📉",
		},
		{
			ID:   "rsi_overbought",
			Name: "RSI Overbought",
			Conditions: []model.AlertCondition{
				{Type: "rsi_above", Symbol: "", Value: 70, Period: 14},
			},
			Description: "Alert saat RSI naik di atas 70. Potensi reversal turun dari kondisi jenuh beli.",
			Icon:        "📈",
		},
		{
			ID:   "volume_breakout",
			Name: "Volume Breakout",
			Conditions: []model.AlertCondition{
				{Type: "volume_spike", Symbol: "", Value: 3, Period: 0},
			},
			Description: "Alert saat volume perdagangan melonjak 3x lipat dari rata-rata. Indikasi minat pasar besar.",
			Icon:        "📊",
		},
		{
			ID:   "support_bounce",
			Name: "Support Bounce",
			Conditions: []model.AlertCondition{
				{Type: "price_above", Symbol: "", Value: 0, Period: 0},
			},
			Description: "Alert saat harga memantul dari level support. Masukkan harga support sebagai nilai target.",
			Icon:        "🔄",
		},
		{
			ID:   "earnings_surprise",
			Name: "Earnings Surprise",
			Conditions: []model.AlertCondition{
				{Type: "percent_up", Symbol: "", Value: 5, Period: 0},
			},
			Description: "Alert saat saham naik/turun 5% dalam sehari. Indikasi katalis besar (laporan keuangan, berita).",
			Icon:        "💎",
		},
	}
}

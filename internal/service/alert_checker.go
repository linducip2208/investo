package service

import (
	"fmt"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service/indicator"
	"investo/internal/service/pattern"
	"math"
	"time"
)

type AlertHit struct {
	Alert        model.Alert   `json:"alert"`
	CurrentPrice float64       `json:"current_price"`
	StockCode    string        `json:"stock_code"`
	HitType      string        `json:"hit_type"`
	Description  string        `json:"description"`
}

type AlertChecker struct {
	AlertRepo       *repository.AlertRepository
	StockPriceRepo  *repository.StockPriceRepository
	StockRepo       *repository.StockRepository
	WatchlistRepo   *repository.WatchlistRepository
	WatchlistItemRepo *repository.WatchlistItemRepository
	PatternService  *pattern.PatternService
	AnomalyService  *AnomalyService
}

func (c *AlertChecker) CheckAlerts() ([]AlertHit, error) {
	alerts, err := c.AlertRepo.FindActive()
	if err != nil {
		return nil, fmt.Errorf("AlertChecker.CheckAlerts: %w", err)
	}

	if len(alerts) == 0 {
		return nil, nil
	}

	stockIDs := make([]int64, 0, len(alerts))
	stockIDSet := make(map[int64]bool)
	for _, alert := range alerts {
		if !stockIDSet[alert.StockID] {
			stockIDs = append(stockIDs, alert.StockID)
			stockIDSet[alert.StockID] = true
		}
	}

	latestPrices, err := c.StockPriceRepo.GetLatestPrices(stockIDs)
	if err != nil {
		return nil, fmt.Errorf("AlertChecker.CheckAlerts prices: %w", err)
	}

	var hits []AlertHit

	for _, alert := range alerts {
		currentPrice, ok := latestPrices[alert.StockID]
		if !ok || currentPrice == 0 {
			continue
		}

		triggered := false
		switch alert.Condition {
		case "above":
			if currentPrice >= alert.TargetPrice {
				triggered = true
			}
		case "below":
			if currentPrice <= alert.TargetPrice {
				triggered = true
			}
		case "percent_up":
			threshold := alert.TargetPrice
			change := ((currentPrice - alert.TargetPrice) / alert.TargetPrice) * 100
			if change >= threshold {
				triggered = true
			}
		case "percent_down":
			threshold := alert.TargetPrice
			change := ((alert.TargetPrice - currentPrice) / alert.TargetPrice) * 100
			if change >= threshold {
				triggered = true
			}
		case "cross_above":
			if currentPrice >= alert.TargetPrice {
				triggered = true
			}
		case "cross_below":
			if currentPrice <= alert.TargetPrice {
				triggered = true
			}
		default:
			if math.Abs(currentPrice-alert.TargetPrice) < 0.01 {
				triggered = true
			}
		}

		if !triggered {
			continue
		}

		stockCode := ""
		stock, err := c.StockRepo.FindByID(alert.StockID)
		if err == nil && stock != nil {
			stockCode = stock.Code
		}

		hits = append(hits, AlertHit{
			Alert:        alert,
			CurrentPrice: currentPrice,
			StockCode:    stockCode,
		})
	}

	return hits, nil
}

func (c *AlertChecker) CheckPatternAlerts() ([]AlertHit, error) {
	alerts, err := c.AlertRepo.FindActive()
	if err != nil {
		return nil, fmt.Errorf("CheckPatternAlerts: %w", err)
	}

	if len(alerts) == 0 {
		return nil, nil
	}

	var hits []AlertHit
	processedStocks := make(map[int64]bool)

	for _, alert := range alerts {
		if processedStocks[alert.StockID] {
			continue
		}
		processedStocks[alert.StockID] = true

		prices, err := c.StockPriceRepo.FindByStockDate(alert.StockID, time.Now().AddDate(0, -3, 0), time.Now())
		if err != nil || len(prices) < 3 {
			latest, err := c.StockPriceRepo.FindLatest(alert.StockID, 90)
			if err != nil || len(latest) < 3 {
				continue
			}
			ReversePrices(latest)
			prices = latest
		}

		patterns := c.PatternService.DetectAll(prices)
		if len(patterns) == 0 {
			continue
		}

		stock, err := c.StockRepo.FindByID(alert.StockID)
		if err != nil || stock == nil {
			continue
		}

		currentPrice := 0.0
		if len(prices) > 0 {
			currentPrice = prices[len(prices)-1].Close
		}

		for _, p := range patterns {
			direction := p.Type
			desc := fmt.Sprintf("Terdeteksi pola %s (%s) pada %s (%s)", p.Name, p.Description, stock.Code, stock.Name)
			hits = append(hits, AlertHit{
				Alert:        alert,
				CurrentPrice: currentPrice,
				StockCode:    stock.Code,
				HitType:      "pattern_" + direction,
				Description:  desc,
			})
		}
	}

	return hits, nil
}

func (c *AlertChecker) CheckIndicatorAlerts() ([]AlertHit, error) {
	alerts, err := c.AlertRepo.FindActive()
	if err != nil {
		return nil, fmt.Errorf("CheckIndicatorAlerts: %w", err)
	}

	if len(alerts) == 0 {
		return nil, nil
	}

	var hits []AlertHit
	processedStocks := make(map[int64]bool)

	for _, alert := range alerts {
		if processedStocks[alert.StockID] {
			continue
		}
		processedStocks[alert.StockID] = true

		prices, err := c.StockPriceRepo.FindByStockDate(alert.StockID, time.Now().AddDate(0, -6, 0), time.Now())
		if err != nil || len(prices) < 60 {
			latest, err := c.StockPriceRepo.FindLatest(alert.StockID, 180)
			if err != nil || len(latest) < 60 {
				continue
			}
			ReversePrices(latest)
			prices = latest
		}

		stock, err := c.StockRepo.FindByID(alert.StockID)
		if err != nil || stock == nil {
			continue
		}

		n := len(prices)
		closePrices := make([]float64, n)
		for i, p := range prices {
			closePrices[i] = p.Close
		}

		currentPrice := closePrices[n-1]

		rsi := indicator.CalcRSI(closePrices, 14)
		lastRSI := lastValid(rsi)
		if lastRSI <= 30 {
			desc := fmt.Sprintf("RSI Oversold (%.1f) pada %s (%s) — potensi reversal naik", lastRSI, stock.Code, stock.Name)
			hits = append(hits, AlertHit{
				Alert:        alert,
				CurrentPrice: currentPrice,
				StockCode:    stock.Code,
				HitType:      "indicator_rsi_oversold",
				Description:  desc,
			})
		} else if lastRSI >= 70 {
			desc := fmt.Sprintf("RSI Overbought (%.1f) pada %s (%s) — potensi reversal turun", lastRSI, stock.Code, stock.Name)
			hits = append(hits, AlertHit{
				Alert:        alert,
				CurrentPrice: currentPrice,
				StockCode:    stock.Code,
				HitType:      "indicator_rsi_overbought",
				Description:  desc,
			})
		}

		macdLine, signalLine, hist := indicator.CalcMACD(closePrices, 12, 26, 9)
		lastMACD := lastValid(macdLine)
		lastSignal := lastValid(signalLine)
		lastHist := lastValid(hist)

		if lastMACD > 0 && lastSignal > 0 && len(hist) >= 2 {
			prevHist := hist[len(hist)-2]
			if prevHist < 0 && lastHist > 0 {
				desc := fmt.Sprintf("MACD Bullish Crossover pada %s (%s) — sinyal beli", stock.Code, stock.Name)
				hits = append(hits, AlertHit{
					Alert:        alert,
					CurrentPrice: currentPrice,
					StockCode:    stock.Code,
					HitType:      "indicator_macd_bullish",
					Description:  desc,
				})
			} else if prevHist > 0 && lastHist < 0 {
				desc := fmt.Sprintf("MACD Bearish Crossover pada %s (%s) — sinyal jual", stock.Code, stock.Name)
				hits = append(hits, AlertHit{
					Alert:        alert,
					CurrentPrice: currentPrice,
					StockCode:    stock.Code,
					HitType:      "indicator_macd_bearish",
					Description:  desc,
				})
			}
		}

		sma50 := indicator.CalcSMA(closePrices, 50)
		sma200 := indicator.CalcSMA(closePrices, 200)
		lastSMA50 := lastValid(sma50)
		lastSMA200 := lastValid(sma200)

		if lastSMA50 > 0 && lastSMA200 > 0 && n >= 2 {
			prevSMA50 := sma50[n-2]
			prevSMA200 := sma200[n-2]

			if prevSMA50 <= prevSMA200 && lastSMA50 > lastSMA200 {
				desc := fmt.Sprintf("Golden Cross terdeteksi pada %s (%s)! SMA 50 memotong SMA 200 ke atas — sinyal bullish jangka panjang", stock.Code, stock.Name)
				hits = append(hits, AlertHit{
					Alert:        alert,
					CurrentPrice: currentPrice,
					StockCode:    stock.Code,
					HitType:      "indicator_golden_cross",
					Description:  desc,
				})
			} else if prevSMA50 >= prevSMA200 && lastSMA50 < lastSMA200 {
				desc := fmt.Sprintf("Death Cross terdeteksi pada %s (%s)! SMA 50 memotong SMA 200 ke bawah — sinyal bearish jangka panjang", stock.Code, stock.Name)
				hits = append(hits, AlertHit{
					Alert:        alert,
					CurrentPrice: currentPrice,
					StockCode:    stock.Code,
					HitType:      "indicator_death_cross",
					Description:  desc,
				})
			}
		}
	}

	return hits, nil
}

func (c *AlertChecker) CheckAnomalyAlerts() ([]AlertHit, error) {
	alerts, err := c.AlertRepo.FindActive()
	if err != nil {
		return nil, fmt.Errorf("CheckAnomalyAlerts: %w", err)
	}

	if len(alerts) == 0 || c.AnomalyService == nil {
		return nil, nil
	}

	anomalies, err := c.AnomalyService.DetectPriceAnomalies()
	if err != nil {
		return nil, fmt.Errorf("CheckAnomalyAlerts anomalies: %w", err)
	}

	if len(anomalies) == 0 {
		return nil, nil
	}

	stockCodeMap := make(map[string]model.Alert)
	for _, alert := range alerts {
		stock, err := c.StockRepo.FindByID(alert.StockID)
		if err != nil || stock == nil {
			continue
		}
		stockCodeMap[stock.Code] = alert
	}

	var hits []AlertHit
	for _, anom := range anomalies {
		if alert, ok := stockCodeMap[anom.StockCode]; ok {
			hits = append(hits, AlertHit{
				Alert:        alert,
				CurrentPrice: anom.Value,
				StockCode:    anom.StockCode,
				HitType:      "anomaly_" + anom.Type,
				Description:  anom.Description,
			})
		}
	}

	gapAnomalies, err := c.AnomalyService.DetectGapAnomalies()
	if err == nil {
		for _, ga := range gapAnomalies {
			if alert, ok := stockCodeMap[ga.StockCode]; ok {
				hits = append(hits, AlertHit{
					Alert:        alert,
					CurrentPrice: ga.Value,
					StockCode:    ga.StockCode,
					HitType:      "anomaly_gap",
					Description:  ga.Description,
				})
			}
		}
	}

	return hits, nil
}

func lastValid(arr []float64) float64 {
	for i := len(arr) - 1; i >= 0; i-- {
		if !math.IsNaN(arr[i]) && arr[i] != 0 {
			return arr[i]
		}
	}
	return 0
}

func lastValidSignal(arr []float64) float64 {
	for i := len(arr) - 1; i >= 0; i-- {
		if !math.IsNaN(arr[i]) && !math.IsInf(arr[i], 0) {
			return arr[i]
		}
	}
	return math.NaN()
}

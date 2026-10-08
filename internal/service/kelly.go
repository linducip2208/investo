package service

import (
	"fmt"
	"math"
)

func CalcKelly(winRate, avgWin, avgLoss float64) float64 {
	if winRate <= 0 || winRate >= 1 || avgWin <= 0 || avgLoss <= 0 {
		return 0
	}

	lossProb := 1.0 - winRate
	kelly := ((winRate / avgLoss) - (lossProb / avgWin)) * avgLoss

	if kelly < 0 {
		return 0
	}

	return kelly
}

func CalcOptimalPosition(portfolioValue, stockPrice, winRate, avgWin, avgLoss float64) (float64, error) {
	if portfolioValue <= 0 {
		return 0, fmt.Errorf("portfolio value must be positive")
	}
	if stockPrice <= 0 {
		return 0, fmt.Errorf("stock price must be positive")
	}

	kelly := CalcKelly(winRate, avgWin, avgLoss)
	if kelly <= 0 {
		return 0, fmt.Errorf("kelly criterion suggests no position")
	}

	halfKelly := kelly * 0.5
	positionValue := portfolioValue * halfKelly
	lots := positionValue / stockPrice

	lots = math.Floor(lots)
	if lots <= 0 {
		lots = 1
	}

	return lots, nil
}

func CalcRiskOfRuin(winRate, avgWin, avgLoss, capital, riskPerTrade float64) float64 {
	if winRate <= 0 || winRate >= 1 || avgWin <= 0 || avgLoss <= 0 || capital <= 0 || riskPerTrade <= 0 {
		if capital <= 0 {
			return 1.0
		}
		return 0
	}

	edge := (winRate * avgWin) - ((1.0 - winRate) * avgLoss)
	if edge <= 0 {
		return 1.0
	}

	n := capital / riskPerTrade
	if n <= 0 {
		return 1.0
	}

	p := winRate
	q := 1.0 - p

	ratio := q / p
	prob := math.Pow(ratio, n)

	if prob > 1.0 {
		prob = 1.0
	}

	return prob
}

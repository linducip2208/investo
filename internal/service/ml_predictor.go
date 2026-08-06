package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/repository"
)

type Prediction struct {
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	PredictedPrice float64   `json:"predicted_price"`
	LowerBound     float64   `json:"lower_bound"`
	UpperBound     float64   `json:"upper_bound"`
	Confidence     float64   `json:"confidence"`
	Trend          string    `json:"trend"`
	RSquared       float64   `json:"r_squared"`
	Slope          float64   `json:"slope"`
	Intercept      float64   `json:"intercept"`
	Support        float64   `json:"support"`
	Resistance     float64   `json:"resistance"`
	Forecast       []DayForecast `json:"forecast"`
}

type DayForecast struct {
	Date     string  `json:"date"`
	Low      float64 `json:"low"`
	High     float64 `json:"high"`
	Mid      float64 `json:"mid"`
}

type MLPredictor struct {
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
}

func (ml *MLPredictor) PredictPrice(code string, days int) (*Prediction, error) {
	stock, err := ml.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("PredictPrice: stock not found: %s", code)
	}

	prices, err := ml.StockPriceRepo.FindLatest(stock.ID, 90)
	if err != nil {
		return nil, fmt.Errorf("PredictPrice: %w", err)
	}

	if len(prices) < 5 {
		return nil, fmt.Errorf("PredictPrice: insufficient data (%d points)", len(prices))
	}

	closes := make([]float64, len(prices))
	for i := range prices {
		closes[len(prices)-1-i] = prices[i].Close
	}

	slope, intercept, rSquared := simpleLinearRegression(closes)

	lastIdx := float64(len(closes) - 1)
	trendLine := make([]float64, len(closes))
	for i := range closes {
		trendLine[i] = slope*float64(i) + intercept
	}

	residuals := make([]float64, len(closes))
	for i := range closes {
		residuals[i] = closes[i] - trendLine[i]
	}

	residualStd := stdDev(residuals)

	predictedPrice := slope*(lastIdx+float64(days)) + intercept
	se := residualStd * math.Sqrt(1+1/float64(len(closes))+math.Pow(float64(days), 2)/sumSqDev(float64(len(closes))))
	tValue := 1.96
	margin := tValue * se

	lowerBound := predictedPrice - margin
	upperBound := predictedPrice + margin
	if lowerBound < 0 {
		lowerBound = 0
	}

	confidence := 0.0
	if rSquared >= 0.8 {
		confidence = 85 + (rSquared-0.8)*50
	} else if rSquared >= 0.6 {
		confidence = 60 + (rSquared-0.6)*125
	} else if rSquared >= 0.4 {
		confidence = 30 + (rSquared-0.4)*150
	} else {
		confidence = rSquared * 75
	}
	if confidence < 10 {
		confidence = 10
	}
	if confidence > 95 {
		confidence = 95
	}
	confidence = math.Round(confidence*10) / 10

	trend := "sideways"
	if slope > 0.01 {
		trend = "up"
	} else if slope < -0.01 {
		trend = "down"
	}

	support := closes[0]
	resistance := closes[0]
	for _, c := range closes {
		if c < support {
			support = c
		}
		if c > resistance {
			resistance = c
		}
	}

	var forecast []DayForecast
	for d := 1; d <= days; d++ {
		dayPred := slope*(lastIdx+float64(d)) + intercept
		daySe := residualStd * math.Sqrt(1+1/float64(len(closes))+math.Pow(float64(d), 2)/sumSqDev(float64(len(closes))))
		dayMargin := tValue * daySe
		dayLow := dayPred - dayMargin
		dayHigh := dayPred + dayMargin
		if dayLow < 0 {
			dayLow = 0
		}

		forecastDate := time.Now().AddDate(0, 0, d).Format("2006-01-02")

		forecast = append(forecast, DayForecast{
			Date: forecastDate,
			Low:  math.Round(dayLow*100) / 100,
			High: math.Round(dayHigh*100) / 100,
			Mid:  math.Round(dayPred*100) / 100,
		})
	}

	return &Prediction{
		Code:           code,
		Name:           stock.Name,
		PredictedPrice: math.Round(predictedPrice*100) / 100,
		LowerBound:     math.Round(lowerBound*100) / 100,
		UpperBound:     math.Round(upperBound*100) / 100,
		Confidence:     confidence,
		Trend:          trend,
		RSquared:       math.Round(rSquared*10000) / 10000,
		Slope:          math.Round(slope*10000) / 10000,
		Intercept:      math.Round(intercept*100) / 100,
		Support:        math.Round(support*100) / 100,
		Resistance:     math.Round(resistance*100) / 100,
		Forecast:       forecast,
	}, nil
}

func simpleLinearRegression(y []float64) (slope, intercept, rSquared float64) {
	n := len(y)
	if n < 2 {
		return 0, y[0], 0
	}

	var sumX, sumY, sumXY, sumXX, sumYY float64
	for i := 0; i < n; i++ {
		x := float64(i)
		yi := y[i]
		sumX += x
		sumY += yi
		sumXY += x * yi
		sumXX += x * x
		sumYY += yi * yi
	}

	denom := float64(n)*sumXX - sumX*sumX
	if denom == 0 {
		return 0, sumY / float64(n), 0
	}

	slope = (float64(n)*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / float64(n)

	meanY := sumY / float64(n)
	var ssTot, ssRes float64
	for i := 0; i < n; i++ {
		pred := slope*float64(i) + intercept
		ssTot += (y[i] - meanY) * (y[i] - meanY)
		ssRes += (y[i] - pred) * (y[i] - pred)
	}

	if ssTot == 0 {
		rSquared = 1.0
	} else {
		rSquared = 1 - ssRes/ssTot
		if rSquared < 0 {
			rSquared = 0
		}
	}

	return
}

func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var sumSq float64
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(values)-1)
	return math.Sqrt(variance)
}

func sumSqDev(n float64) float64 {
	mean := n / 2
	var sum float64
	for i := 0.0; i < n; i++ {
		diff := i - mean
		sum += diff * diff
	}
	return sum
}

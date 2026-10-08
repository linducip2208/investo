package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type CustomIndex struct {
	ID          int64              `json:"id"`
	UserID      int64              `json:"user_id"`
	Name        string             `json:"name"`
	Stocks      []string           `json:"stocks"`
	Weights     map[string]float64 `json:"weights"`
	BaseValue   float64            `json:"base_value"`
	CreatedAt   time.Time          `json:"created_at"`
}

type IndexValueResult struct {
	Index        CustomIndex               `json:"index"`
	CurrentValue float64                   `json:"current_value"`
	ChangePct    float64                   `json:"change_pct"`
	Components   []IndexComponentValue     `json:"components"`
}

type IndexComponentValue struct {
	Code   string  `json:"code"`
	Weight float64 `json:"weight"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

type CustomIndexService struct {
	CustomIndexRepo *repository.CustomIndexRepository
	StockRepo       *repository.StockRepository
	StockPriceRepo  *repository.StockPriceRepository
}

func (s *CustomIndexService) CalculateValue(indexID int64) (*IndexValueResult, error) {
	idx, err := s.CustomIndexRepo.FindByID(indexID)
	if err != nil {
		return nil, fmt.Errorf("CustomIndex CalculateValue: %w", err)
	}

	var stocks []string
	var weights map[string]float64
	if err := json.Unmarshal([]byte(idx.StocksJSON), &stocks); err != nil {
		return nil, fmt.Errorf("unmarshal stocks: %w", err)
	}
	if err := json.Unmarshal([]byte(idx.WeightsJSON), &weights); err != nil {
		weights = make(map[string]float64)
		even := 100.0 / float64(len(stocks))
		for _, s := range stocks {
			weights[s] = even
		}
	}

	ci := CustomIndex{
		ID:        idx.ID,
		UserID:    idx.UserID,
		Name:      idx.Name,
		Stocks:    stocks,
		Weights:   weights,
		BaseValue: idx.BaseValue,
		CreatedAt: idx.CreatedAt,
	}

	result := &IndexValueResult{
		Index: ci,
	}

	var totalWeight float64
	for _, w := range weights {
		totalWeight += w
	}

	for _, code := range stocks {
		stock, err := s.StockRepo.FindByCode(code)
		if err != nil {
			comp := IndexComponentValue{Code: code, Weight: weights[code], Price: 0, Change: 0}
			result.Components = append(result.Components, comp)
			continue
		}

		latestPrices, _ := s.StockPriceRepo.FindLatest(stock.ID, 2)
		price := 0.0
		change := 0.0
		if len(latestPrices) >= 2 {
			prices := make([]model.StockPrice, len(latestPrices))
			for i := len(latestPrices) - 1; i >= 0; i-- {
				prices[len(latestPrices)-1-i] = latestPrices[i]
			}
			price = prices[0].Close
			if prices[1].Close > 0 {
				change = ((prices[0].Close - prices[1].Close) / prices[1].Close) * 100
			}
		} else if len(latestPrices) == 1 {
			price = latestPrices[0].Close
		}

		normalizedWeight := weights[code] / totalWeight
		wgtdPrice := price * normalizedWeight / 100.0

		result.CurrentValue += wgtdPrice

		comp := IndexComponentValue{
			Code:   code,
			Weight: weights[code],
			Price:  math.Round(price*100) / 100,
			Change: math.Round(change*100) / 100,
		}
		result.Components = append(result.Components, comp)
	}

	if totalWeight > 0 && result.CurrentValue > 0 {
		result.CurrentValue = math.Round(result.CurrentValue*100) / 100
	}

	if idx.BaseValue > 0 {
		result.ChangePct = math.Round(((result.CurrentValue-idx.BaseValue)/idx.BaseValue)*10000) / 100
	}

	return result, nil
}

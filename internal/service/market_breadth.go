package service

import (
	"math"

	"investo/internal/repository"
)

type MarketBreadthService struct {
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
}

type MarketBreadth struct {
	Advance         int     `json:"advance"`
	Decline         int     `json:"decline"`
	Unchanged       int     `json:"unchanged"`
	AdvancePercent  float64 `json:"advance_percent"`
	DeclinePercent  float64 `json:"decline_percent"`
	TotalVolume     int64   `json:"total_volume"`
	TotalValue      float64 `json:"total_value"`
	ForeignBuy      float64 `json:"foreign_buy"`
	ForeignSell     float64 `json:"foreign_sell"`
	ForeignNet      float64 `json:"foreign_net"`
}

func (s *MarketBreadthService) Calculate() (*MarketBreadth, error) {
	pricesWithPrev, err := s.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, err
	}

	mb := &MarketBreadth{}

	for _, p := range pricesWithPrev {
		if p.LatestClose > p.PrevClose {
			mb.Advance++
		} else if p.LatestClose < p.PrevClose {
			mb.Decline++
		} else {
			mb.Unchanged++
		}

		mb.TotalVolume += p.LatestVolume
		mb.TotalValue += p.LatestClose * float64(p.LatestVolume)
	}

	total := float64(mb.Advance + mb.Decline + mb.Unchanged)
	if total > 0 {
		mb.AdvancePercent = math.Round((float64(mb.Advance)/total*100)*100) / 100
		mb.DeclinePercent = math.Round((float64(mb.Decline)/total*100)*100) / 100
	}

	return mb, nil
}

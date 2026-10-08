package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type LiquidityZone struct {
	Price    float64 `json:"price"`
	Volume   int64   `json:"volume"`
	Type     string  `json:"type"`
	Strength float64 `json:"strength"`
}

type LiquidityService struct {
	StockPriceRepo *repository.StockPriceRepository
}

func (s *LiquidityService) FindLiquidityZones(stockID int64) ([]LiquidityZone, error) {
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

	prices, err := s.StockPriceRepo.FindByStockDate(stockID, start, end)
	if err != nil {
		return nil, fmt.Errorf("liquidity: %w", err)
	}
	if len(prices) == 0 {
		return nil, nil
	}

	return computeZones(prices), nil
}

func (s *LiquidityService) FindLiquidityZonesRange(stockID int64, startDate, endDate string) ([]LiquidityZone, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		start = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		end = time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	}

	prices, err := s.StockPriceRepo.FindByStockDate(stockID, start, end)
	if err != nil {
		return nil, fmt.Errorf("liquidity: %w", err)
	}
	if len(prices) == 0 {
		return nil, nil
	}

	return computeZones(prices), nil
}

func computeZones(prices []model.StockPrice) []LiquidityZone {
	type priceLevel struct {
		price     int64
		volume    int64
		count     int
		totalHigh float64
	}

	levels := make(map[int64]*priceLevel)
	var totalVolume int64
	var latestClose float64

	for i, p := range prices {
		level := int64(math.Round(p.Close))
		if i == len(prices)-1 {
			latestClose = p.Close
		}
		if _, ok := levels[level]; !ok {
			levels[level] = &priceLevel{price: level}
		}
		levels[level].volume += p.Volume
		levels[level].count++
		levels[level].totalHigh += p.High
		totalVolume += p.Volume
	}

	if len(levels) == 0 || totalVolume == 0 {
		return nil
	}

	avgVolumePerLevel := float64(totalVolume) / float64(len(levels))
	threshold := avgVolumePerLevel * 2.0

	var zones []LiquidityZone
	for _, lv := range levels {
		if float64(lv.volume) >= threshold {
			zoneType := "resistance"
			if float64(lv.price) < latestClose {
				zoneType = "support"
			}
			distanceFromPrice := math.Abs(float64(lv.price)-latestClose) / latestClose
			if distanceFromPrice < 0.001 {
				distanceFromPrice = 0.001
			}
			strength := (float64(lv.volume)/float64(totalVolume)*100.0 + (1.0/distanceFromPrice)*5.0 + 2.0) / 3.0

			zones = append(zones, LiquidityZone{
				Price:    float64(lv.price),
				Volume:   lv.volume,
				Type:     zoneType,
				Strength: math.Round(strength*100) / 100,
			})
		}
	}

	sort.Slice(zones, func(i, j int) bool {
		if zones[i].Type != zones[j].Type {
			if zones[i].Type == "support" {
				return true
			}
			return false
		}
		return zones[i].Strength > zones[j].Strength
	})

	if len(zones) > 15 {
		zones = zones[:15]
	}

	return zones
}

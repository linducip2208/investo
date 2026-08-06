package service

import (
	"math"
	"sort"
)

type ProfileLevel struct {
	Price    float64 `json:"price"`
	Volume   int64   `json:"volume"`
	Type     string  `json:"type"`
	CumPct   float64 `json:"cumulative_pct"`
}

type VolumeProfile struct {
	Levels     []ProfileLevel `json:"levels"`
	POC        float64        `json:"poc"`
	VAH        float64        `json:"vah"`
	VAL        float64        `json:"val"`
	TotalVolume int64         `json:"total_volume"`
}

type MarketProfileService struct{}

func NewMarketProfileService() *MarketProfileService {
	return &MarketProfileService{}
}

func (s *MarketProfileService) CalcVolumeProfile(prices []float64, volumes []float64, bins int) (*VolumeProfile, error) {
	if len(prices) == 0 || len(volumes) == 0 {
		return &VolumeProfile{}, nil
	}
	if len(prices) != len(volumes) {
		minLen := len(prices)
		if len(volumes) < minLen {
			minLen = len(volumes)
		}
		prices = prices[:minLen]
		volumes = volumes[:minLen]
	}

	minPrice := prices[0]
	maxPrice := prices[0]
	var totalVol int64
	for i, p := range prices {
		if p < minPrice {
			minPrice = p
		}
		if p > maxPrice {
			maxPrice = p
		}
		totalVol += int64(volumes[i])
	}

	if minPrice == maxPrice {
		minPrice -= 0.01
		maxPrice += 0.01
	}

	if bins <= 0 {
		bins = 20
	}

	binSize := (maxPrice - minPrice) / float64(bins)
	if binSize <= 0 {
		binSize = 0.01
	}

	binsMap := make(map[int]int64)
	binPrices := make(map[int]float64)
	for i := 0; i < bins; i++ {
		binCenter := minPrice + (float64(i)+0.5)*binSize
		binPrices[i] = math.Round(binCenter*100) / 100
		binsMap[i] = 0
	}

	for i, p := range prices {
		binIdx := int((p - minPrice) / binSize)
		if binIdx < 0 {
			binIdx = 0
		}
		if binIdx >= bins {
			binIdx = bins - 1
		}
		binsMap[binIdx] += int64(volumes[i])
	}

	var levels []ProfileLevel
	var maxVol int64
	for i := 0; i < bins; i++ {
		vol := binsMap[i]
		if vol > maxVol {
			maxVol = vol
		}
		levels = append(levels, ProfileLevel{
			Price:    binPrices[i],
			Volume:   vol,
			Type:     "normal",
		})
	}

	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Volume > levels[j].Volume
	})

	var poc float64
	if len(levels) > 0 && maxVol > 0 {
		poc = levels[0].Price
		for i := range levels {
			if levels[i].Volume == maxVol {
				levels[i].Type = "POC"
				poc = levels[i].Price
				break
			}
		}
	}

	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Price < levels[j].Price
	})

	var cumVol int64
	for i := range levels {
		cumVol += levels[i].Volume
		if totalVol > 0 {
			levels[i].CumPct = math.Round(float64(cumVol)/float64(totalVol)*10000) / 100
		}
	}

	if totalVol > 0 {
		targetVol := totalVol * 7 / 10
		var tempCum int64
		var vah, val float64
		valSet := false

		for i := range levels {
			tempCum += levels[i].Volume
			if !valSet && float64(tempCum) >= float64(totalVol)*0.15 {
				val = levels[i].Price
				valSet = true
			}
			if float64(tempCum) >= float64(totalVol)*0.85 {
				vah = levels[i].Price
				break
			}
		}

		_ = targetVol

		for i := range levels {
			if val > 0 && vah > 0 {
				if levels[i].Price >= val && levels[i].Price <= vah {
					if levels[i].Type == "normal" {
						levels[i].Type = "VA"
					}
				}
			}
			if val > 0 && math.Abs(levels[i].Price-val) < 0.0001 {
				levels[i].Type = "VAL"
			}
			if vah > 0 && math.Abs(levels[i].Price-vah) < 0.0001 {
				levels[i].Type = "VAH"
			}
		}

		return &VolumeProfile{
			Levels:      levels,
			POC:         poc,
			VAH:         vah,
			VAL:         val,
			TotalVolume: totalVol,
		}, nil
	}

	return &VolumeProfile{
		Levels:      levels,
		POC:         poc,
		TotalVolume: totalVol,
	}, nil
}

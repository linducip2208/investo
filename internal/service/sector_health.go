package service

import (
	"math"
	"sort"

	"investo/internal/repository"
)

type SectorHealth struct {
	SectorName       string  `json:"sector_name"`
	SectorSlug       string  `json:"sector_slug"`
	StockCount       int     `json:"stock_count"`
	HealthScore      float64 `json:"health_score"`
	PERAvg           float64 `json:"per_avg"`
	PBVAvg           float64 `json:"pbv_avg"`
	ROEAvg           float64 `json:"roe_avg"`
	MomentumScore    float64 `json:"momentum_score"`
	ForeignFlowScore float64 `json:"foreign_flow_score"`
	CompositeScore   float64 `json:"composite_score"`
}

type SectorHealthService struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
}

func (s *SectorHealthService) CalculateAll() ([]SectorHealth, error) {
	sectors, err := s.SectorRepo.FindAll()
	if err != nil {
		return nil, err
	}

	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	sectorStocks := make(map[int64][]int64)
	sectorName := make(map[int64]string)
	sectorSlug := make(map[int64]string)
	for _, sec := range sectors {
		sectorName[sec.ID] = sec.Name
		sectorSlug[sec.ID] = sec.Slug
	}
	for _, st := range stocks {
		sectorStocks[st.SectorID] = append(sectorStocks[st.SectorID], st.ID)
	}

	var results []SectorHealth
	for secID, stockIDs := range sectorStocks {
		if len(stockIDs) == 0 {
			continue
		}

		var pers, pbvs, roes, momentumChanges []float64
		count := 0

		for _, sid := range stockIDs {
			fund, err := s.StockFundamentalRepo.FindLatest(sid)
			if err != nil || fund == nil {
				continue
			}
			if fund.PER > 0 && fund.PER < 100 {
				pers = append(pers, fund.PER)
			}
			if fund.PBV > 0 && fund.PBV < 20 {
				pbvs = append(pbvs, fund.PBV)
			}
			if fund.ROE > -50 && fund.ROE < 100 {
				roes = append(roes, fund.ROE)
			}

			change, _ := s.StockPriceRepo.GetPriceChange(sid, 21)
			if math.Abs(change) < 100 {
				momentumChanges = append(momentumChanges, change)
			}
			count++
		}

		if count == 0 {
			continue
		}

		perAvg := sectorAvg(pers)
		pbvAvg := sectorAvg(pbvs)
		roeAvg := sectorAvg(roes)
		momentumAvg := sectorAvg(momentumChanges)

		perScore := normalizeInverse(perAvg, 5, 40)
		pbvScore := normalizeInverse(pbvAvg, 0.5, 8)
		roeScore := normalizeDirect(roeAvg, 0, 30)
		momentumScore := normalizeDirect(momentumAvg, -20, 20)
		foreignScore := 50.0

		composite := perScore*0.30 + pbvScore*0.15 + roeScore*0.30 + momentumScore*0.10 + foreignScore*0.15

		healthScore := composite

		results = append(results, SectorHealth{
			SectorName:       sectorName[secID],
			SectorSlug:       sectorSlug[secID],
			StockCount:       count,
			HealthScore:      math.Round(healthScore*10) / 10,
			PERAvg:           math.Round(perAvg*100) / 100,
			PBVAvg:           math.Round(pbvAvg*100) / 100,
			ROEAvg:           math.Round(roeAvg*100) / 100,
			MomentumScore:    math.Round(momentumScore*10) / 10,
			ForeignFlowScore: math.Round(foreignScore*10) / 10,
			CompositeScore:   math.Round(composite*10) / 10,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].HealthScore > results[j].HealthScore
	})

	return results, nil
}

func sectorAvg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func normalizeInverse(value, min, max float64) float64 {
	if max <= min {
		return 50
	}
	if value <= min {
		return 100
	}
	if value >= max {
		return 0
	}
	return 100 * (1 - (value-min)/(max-min))
}

func normalizeDirect(value, min, max float64) float64 {
	if max <= min {
		return 50
	}
	if value <= min {
		return 0
	}
	if value >= max {
		return 100
	}
	return 100 * (value - min) / (max - min)
}

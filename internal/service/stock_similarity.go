package service

import (
	"math"
	"sort"

	"investo/internal/repository"
)

type SimilarStock struct {
	StockCode        string   `json:"stock_code"`
	Name             string   `json:"name"`
	SimilarityScore  float64  `json:"similarity_score"`
	CommonFactors    []string `json:"common_factors"`
	SectorName       string   `json:"sector_name"`
	MarketCap        float64  `json:"market_cap"`
	Per              float64  `json:"per"`
}

type StockSimilarityService struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
}

func (s *StockSimilarityService) FindSimilar(code string) ([]SimilarStock, error) {
	target, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, err
	}

	targetFund, _ := s.StockFundamentalRepo.FindLatest(target.ID)

	allStocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	var stockIDs []int64
	for _, st := range allStocks {
		stockIDs = append(stockIDs, st.ID)
	}
	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	targetPrice := priceMap[target.ID]
	targetMarketCap := float64(target.SharesOutstanding) * targetPrice
	targetPER := 0.0
	if targetFund != nil {
		targetPER = targetFund.PER
	}

	type scored struct {
		stock    SimilarStock
		score    float64
	}

	var candidates []scored

	for _, st := range allStocks {
		if st.ID == target.ID {
			continue
		}

		price := priceMap[st.ID]
		if price <= 0 {
			continue
		}

		marketCap := float64(st.SharesOutstanding) * price
		fund, _ := s.StockFundamentalRepo.FindLatest(st.ID)

		per := 0.0
		if fund != nil {
			per = fund.PER
		}

		var factors []string
		sectorScore := 0.0
		if st.SectorID == target.SectorID {
			sectorScore = 50.0
			factors = append(factors, "Sektor yang sama")
		}

		capScore := 0.0
		if targetMarketCap > 0 && marketCap > 0 {
			ratio := math.Min(marketCap, targetMarketCap) / math.Max(marketCap, targetMarketCap)
			capScore = 20.0 * ratio
			if ratio > 0.7 {
				factors = append(factors, "Kapitalisasi pasar mirip")
			}
		}

		perScore := 0.0
		if targetPER > 0 && per > 0 {
			perDiff := math.Abs(targetPER - per)
			maxPER := math.Max(targetPER, per)
			if maxPER > 0 {
				perSim := 1 - (perDiff / maxPER)
				if perSim > 0 {
					perScore = 15.0 * perSim
				}
				if perSim > 0.7 {
					factors = append(factors, "PER dalam range yang mirip")
				}
			}
		}

		corrScore := s.calculatePriceCorrelation(target.ID, st.ID)
		if corrScore > 0.7 {
			factors = append(factors, "Korelasi harga tinggi")
		}

		totalScore := sectorScore + capScore + perScore + corrScore*15.0

		if len(factors) == 0 {
			factors = append(factors, "Saham terkait")
		}

		sectorName := ""
		sec, _ := s.SectorRepo.FindByID(st.SectorID)
		if sec != nil {
			sectorName = sec.Name
		}

		candidates = append(candidates, scored{
			stock: SimilarStock{
				StockCode:       st.Code,
				Name:            st.Name,
				SimilarityScore: math.Round(totalScore*10) / 10,
				CommonFactors:   factors,
				SectorName:      sectorName,
				MarketCap:       math.Round(marketCap),
				Per:             math.Round(per*100) / 100,
			},
			score: totalScore,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	limit := 5
	if len(candidates) < limit {
		limit = len(candidates)
	}

	var result []SimilarStock
	for i := 0; i < limit; i++ {
		result = append(result, candidates[i].stock)
	}
	if result == nil {
		result = []SimilarStock{}
	}

	return result, nil
}

func (s *StockSimilarityService) calculatePriceCorrelation(stockA, stockB int64) float64 {
	pricesA, errA := s.StockPriceRepo.FindLatest(stockA, 60)
	pricesB, errB := s.StockPriceRepo.FindLatest(stockB, 60)
	if errA != nil || errB != nil || len(pricesA) < 5 || len(pricesB) < 5 {
		return 0
	}

	minLen := len(pricesA)
	if len(pricesB) < minLen {
		minLen = len(pricesB)
	}

	var changesA, changesB []float64
	for i := 1; i < minLen; i++ {
		chA := (pricesA[i].Close - pricesA[i-1].Close) / pricesA[i-1].Close
		chB := (pricesB[i].Close - pricesB[i-1].Close) / pricesB[i-1].Close
		changesA = append(changesA, chA)
		changesB = append(changesB, chB)
	}

	if len(changesA) < 3 {
		return 0
	}

	var sA, sB, sAA, sBB, sAB float64
	n := float64(len(changesA))
	for i := range changesA {
		sA += changesA[i]
		sB += changesB[i]
		sAA += changesA[i] * changesA[i]
		sBB += changesB[i] * changesB[i]
		sAB += changesA[i] * changesB[i]
	}

	num := n*sAB - sA*sB
	den := math.Sqrt((n*sAA - sA*sA) * (n*sBB - sB*sB))
	if den == 0 {
		return 0
	}

	corr := num / den
	if corr > 1 {
		corr = 1
	}
	if corr < -1 {
		corr = -1
	}
	return corr
}

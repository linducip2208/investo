package service

import (
	"fmt"
	"math"
	"strings"

	"investo/internal/model"
	"investo/internal/repository"
)

type ScreenerService struct {
	StockRepo            *repository.StockRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	StockPriceRepo       *repository.StockPriceRepository
	SectorRepo           *repository.SectorRepository
}

type ScreenerCriteria struct {
	MinPER       float64 `json:"min_per"`
	MaxPER       float64 `json:"max_per"`
	MinPBV       float64 `json:"min_pbv"`
	MaxPBV       float64 `json:"max_pbv"`
	MinROE       float64 `json:"min_roe"`
	MaxROE       float64 `json:"max_roe"`
	MinDER       float64 `json:"min_der"`
	MaxDER       float64 `json:"max_der"`
	MinDivYield  float64 `json:"min_div_yield"`
	MinEPSGrowth float64 `json:"min_eps_growth"`
	MarketCapMin float64 `json:"market_cap_min"`
	MarketCapMax float64 `json:"market_cap_max"`
	SectorID     int64   `json:"sector_id"`
	SortBy       string  `json:"sort_by"`
	SortOrder    string  `json:"sort_order"`
}

type ScreenerResult struct {
	ID            int64   `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	SectorName    string  `json:"sector_name"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"change_percent"`
	PER           float64 `json:"per"`
	PBV           float64 `json:"pbv"`
	ROE           float64 `json:"roe"`
	DER           float64 `json:"der"`
	NPM           float64 `json:"npm"`
	DivYield      float64 `json:"div_yield"`
	MarketCap     float64 `json:"market_cap"`
	Score         int     `json:"score"`
}

func (s *ScreenerService) Screen(criteria ScreenerCriteria) ([]ScreenerResult, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("Screen: %w", err)
	}

	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	prices, err := s.StockPriceRepo.GetLatestPrices(stockIDs)
	if err != nil {
		prices = make(map[int64]float64)
	}

	sectors, err := s.SectorRepo.FindAll()
	sectorMap := make(map[int64]string)
	if err == nil {
		for _, sec := range sectors {
			sectorMap[sec.ID] = sec.Name
		}
	}

	var results = make([]ScreenerResult, 0)

	for _, st := range stocks {
		if criteria.SectorID > 0 && st.SectorID != criteria.SectorID {
			continue
		}

		fund, err := s.StockFundamentalRepo.FindLatest(st.ID)
		if err != nil {
			continue
		}

		price := prices[st.ID]

		if criteria.MinPER > 0 && fund.PER < criteria.MinPER {
			continue
		}
		if criteria.MaxPER > 0 && fund.PER > criteria.MaxPER {
			continue
		}
		if criteria.MinPBV > 0 && fund.PBV < criteria.MinPBV {
			continue
		}
		if criteria.MaxPBV > 0 && fund.PBV > criteria.MaxPBV {
			continue
		}
		if criteria.MinROE > 0 && fund.ROE < criteria.MinROE {
			continue
		}
		if criteria.MaxROE > 0 && fund.ROE > criteria.MaxROE {
			continue
		}
		if criteria.MinDER > 0 && fund.DER > criteria.MinDER {
			continue
		}
		if criteria.MaxDER > 0 && fund.DER < criteria.MaxDER {
			continue
		}
		if criteria.MinDivYield > 0 && fund.DividendYield < criteria.MinDivYield {
			continue
		}

		marketCap := price * float64(st.SharesOutstanding)

		if criteria.MarketCapMin > 0 && marketCap < criteria.MarketCapMin {
			continue
		}
		if criteria.MarketCapMax > 0 && marketCap > criteria.MarketCapMax {
			continue
		}

		changePercent := 0.0

		score := s.calcScore(fund, price, st.SharesOutstanding, criteria)

		results = append(results, ScreenerResult{
			ID:            st.ID,
			Code:          st.Code,
			Name:          st.Name,
			SectorName:    sectorMap[st.SectorID],
			Price:         math.Round(price*100) / 100,
			ChangePercent: math.Round(changePercent*100) / 100,
			PER:           math.Round(fund.PER*100) / 100,
			PBV:           math.Round(fund.PBV*100) / 100,
			ROE:           math.Round(fund.ROE*100) / 100,
			DER:           math.Round(fund.DER*100) / 100,
			NPM:           math.Round(fund.NetProfitMargin*100) / 100,
			DivYield:      math.Round(fund.DividendYield*100) / 100,
			MarketCap:     marketCap,
			Score:         score,
		})
	}

	s.sortResults(results, criteria.SortBy, criteria.SortOrder)

	return results, nil
}

func (s *ScreenerService) calcScore(fund *model.StockFundamental, _ float64, _ int64, criteria ScreenerCriteria) int {
	score := 50

	if criteria.MinPER > 0 && criteria.MaxPER > 0 {
		midPER := (criteria.MinPER + criteria.MaxPER) / 2
		diff := math.Abs(fund.PER-midPER) / midPER
		if diff < 0.2 {
			score += 10
		} else if diff < 0.5 {
			score += 5
		}
	}

	if fund.ROE >= 15 {
		score += 15
	} else if fund.ROE >= 10 {
		score += 10
	} else if fund.ROE >= 5 {
		score += 5
	} else if fund.ROE < 0 {
		score -= 10
	}

	if fund.PER > 0 && fund.PER < 15 {
		score += 10
	} else if fund.PER > 0 && fund.PER < 20 {
		score += 5
	} else if fund.PER > 30 {
		score -= 5
	}

	if fund.PBV > 0 && fund.PBV < 2 {
		score += 10
	} else if fund.PBV > 0 && fund.PBV < 3 {
		score += 5
	}

	if fund.DER <= 1.0 {
		score += 10
	} else if fund.DER <= 2.0 {
		score += 5
	} else if fund.DER > 3.0 {
		score -= 5
	}

	if fund.NetProfitMargin >= 10 {
		score += 10
	} else if fund.NetProfitMargin >= 5 {
		score += 5
	}

	if fund.DividendYield >= 4 {
		score += 10
	} else if fund.DividendYield >= 2 {
		score += 5
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func (s *ScreenerService) sortResults(results []ScreenerResult, sortBy, sortOrder string) {
	asc := strings.ToLower(sortOrder) != "desc"

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			var less bool
			switch strings.ToLower(sortBy) {
			case "per":
				less = results[i].PER < results[j].PER
			case "pbv":
				less = results[i].PBV < results[j].PBV
			case "roe":
				less = results[i].ROE < results[j].ROE
			case "change":
				less = results[i].ChangePercent < results[j].ChangePercent
			case "marketcap":
				less = results[i].MarketCap < results[j].MarketCap
			case "score":
				less = results[i].Score < results[j].Score
			default:
				less = results[i].MarketCap < results[j].MarketCap
			}

			swap := false
			if asc && less {
				swap = true
			}
			if !asc && !less {
				swap = true
			}

			if swap {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func (s *ScreenerService) NaturalLanguageScreen(query string) ([]ScreenerResult, error) {
	parser := &NLParser{}
	criteria, err := parser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("NaturalLanguageScreen parse: %w", err)
	}
	return s.Screen(criteria)
}

func (s *ScreenerService) QuickScreen(preset string) ([]ScreenerResult, error) {
	var criteria ScreenerCriteria
	criteria.SortBy = "score"
	criteria.SortOrder = "desc"

	switch strings.ToLower(preset) {
	case "value":
		criteria.MaxPER = 15
		criteria.MaxPBV = 2
		criteria.MinROE = 10
		criteria.SortBy = "per"
		criteria.SortOrder = "asc"
	case "growth":
		criteria.MinROE = 15
		criteria.SortBy = "roe"
		criteria.SortOrder = "desc"
	case "dividend":
		criteria.MinDivYield = 3
		criteria.MaxDER = 1.5
		criteria.MinROE = 5
		criteria.SortBy = "div_yield"
		criteria.SortOrder = "desc"
	case "momentum":
		criteria.MinROE = 5
		criteria.SortBy = "change"
		criteria.SortOrder = "desc"
	case "defensive":
		criteria.MaxDER = 1.0
		criteria.MinROE = 10
		criteria.MinDivYield = 2
		criteria.MarketCapMin = 10_000_000_000_000
		criteria.SortBy = "marketcap"
		criteria.SortOrder = "desc"
	case "turnaround":
		criteria.MaxPER = 12
		criteria.MinROE = 3
		criteria.SortBy = "roe"
		criteria.SortOrder = "desc"
	default:
		criteria.SortBy = "marketcap"
		criteria.SortOrder = "desc"
	}

	return s.Screen(criteria)
}



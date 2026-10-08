package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type RSResult struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	RSRating   float64 `json:"rs_rating"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
	Sector     string  `json:"sector"`
	Return1Mo  float64 `json:"return_1mo"`
	Return3Mo  float64 `json:"return_3mo"`
}

type HLResult struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	NewHigh          float64 `json:"new_high"`
	NewLow           float64 `json:"new_low"`
	Date             string `json:"date"`
	Price            float64 `json:"price"`
	Volume           int64   `json:"volume"`
	ConsecutiveDays  int    `json:"consecutive_days"`
	Type             string `json:"type"`
}

type RSRanking struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	SectorRepo     *repository.SectorRepository
}

func (rs *RSRanking) CalcRSRanking() ([]RSResult, error) {
	stocks, err := rs.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("CalcRSRanking: %w", err)
	}

	sectors, err := rs.SectorRepo.FindAll()
	sectorMap := make(map[int64]string)
	if err == nil {
		for _, s := range sectors {
			sectorMap[s.ID] = s.Name
		}
	}

	type stockReturn struct {
		id       int64
		code     string
		name     string
		sector   string
		price    float64
		return1m float64
		return3m float64
	}

	end := time.Now()
	start12m := end.AddDate(-1, 0, 0)
	start3m := end.AddDate(0, -3, 0)
	start1m := end.AddDate(0, -1, 0)

	var allIDs []int64
	for _, s := range stocks {
		allIDs = append(allIDs, s.ID)
	}

	allPrices, err := rs.StockPriceRepo.FindByDateRange(allIDs, start12m, end)
	if err != nil {
		return nil, fmt.Errorf("CalcRSRanking price fetch: %w", err)
	}

	priceByStock := make(map[int64][]model.StockPrice)
	for _, p := range allPrices {
		priceByStock[p.StockID] = append(priceByStock[p.StockID], p)
	}

	var returns []stockReturn
	for _, s := range stocks {
		prices, ok := priceByStock[s.ID]
		if !ok || len(prices) < 2 {
			continue
		}

		latestPrice := prices[len(prices)-1].Close
		if latestPrice <= 0 {
			continue
		}

		ret1m := calcReturnFrom(prices, start1m)
		ret3m := calcReturnFrom(prices, start3m)

		returns = append(returns, stockReturn{
			id:       s.ID,
			code:     s.Code,
			name:     s.Name,
			sector:   sectorMap[s.SectorID],
			price:    math.Round(latestPrice*100) / 100,
			return1m: math.Round(ret1m*100) / 100,
			return3m: math.Round(ret3m*100) / 100,
		})
	}

	minRet := math.MaxFloat64
	maxRet := -math.MaxFloat64
	for _, r := range returns {
		score := r.return1m*0.4 + r.return3m*0.6
		if score < minRet {
			minRet = score
		}
		if score > maxRet {
			maxRet = score
		}
	}

	var results []RSResult
	scoreRange := maxRet - minRet
	for _, r := range returns {
		score := r.return1m*0.4 + r.return3m*0.6
		var rsRating float64
		if scoreRange > 0 {
			rsRating = ((score - minRet) / scoreRange) * 100
		} else {
			rsRating = 50
		}
		rsRating = math.Round(rsRating*10) / 10
		if rsRating < 0 {
			rsRating = 0
		}
		if rsRating > 100 {
			rsRating = 100
		}

		results = append(results, RSResult{
			Code:      r.code,
			Name:      r.name,
			RSRating:  rsRating,
			Price:     r.price,
			ChangePct: r.return1m,
			Sector:    r.sector,
			Return1Mo: r.return1m,
			Return3Mo: r.return3m,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].RSRating > results[j].RSRating
	})

	return results, nil
}

func calcReturnFrom(prices []model.StockPrice, fromDate time.Time) float64 {
	if len(prices) == 0 {
		return 0
	}

	latestClose := prices[len(prices)-1].Close
	if latestClose <= 0 {
		return 0
	}

	startPrice := latestClose
	for i := 0; i < len(prices); i++ {
		if !prices[i].Date.Before(fromDate) {
			if i > 0 {
				startPrice = prices[i-1].Close
			} else {
				startPrice = prices[i].Close
			}
			break
		}
	}

	if startPrice <= 0 {
		return 0
	}

	return ((latestClose - startPrice) / startPrice) * 100
}

func (rs *RSRanking) ScanNewHighs() ([]HLResult, error) {
	return rs.scanNewHL("high")
}

func (rs *RSRanking) ScanNewLows() ([]HLResult, error) {
	return rs.scanNewHL("low")
}

func (rs *RSRanking) scanNewHL(hlType string) ([]HLResult, error) {
	stocks, err := rs.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("scanNewHL: %w", err)
	}

	var results []HLResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)

	for _, s := range stocks {
		wg.Add(1)
		go func(st model.Stock) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prices, err := rs.StockPriceRepo.FindLatest(st.ID, 260)
			if err != nil || len(prices) < 20 {
				return
			}

			latestPrice := prices[0].Close
			if latestPrice <= 0 {
				return
			}

			high52 := latestPrice
			low52 := latestPrice
			for _, p := range prices {
				if p.Close > high52 {
					high52 = p.Close
				}
				if p.Close < low52 {
					low52 = p.Close
				}
			}

			isNewHigh := latestPrice >= high52*0.995
			isNewLow := latestPrice <= low52*1.005

			var r HLResult
			if hlType == "high" && isNewHigh {
				r = HLResult{
					Code: st.Code, Name: st.Name,
					NewHigh: math.Round(high52*100) / 100,
					Date:    prices[0].Date.Format("2006-01-02"),
					Price:   math.Round(latestPrice*100) / 100,
					Volume:  prices[0].Volume,
					Type:    "high",
				}
			} else if hlType == "low" && isNewLow {
				r = HLResult{
					Code: st.Code, Name: st.Name,
					NewLow: math.Round(low52*100) / 100,
					Date:   prices[0].Date.Format("2006-01-02"),
					Price:  math.Round(latestPrice*100) / 100,
					Volume: prices[0].Volume,
					Type:   "low",
				}
			} else {
				return
			}

			consecutiveDays := 1
			for i := 1; i < len(prices)-1; i++ {
				if hlType == "high" && prices[i].Close >= r.NewHigh*0.995 {
					consecutiveDays++
				} else if hlType == "low" && prices[i].Close <= r.NewLow*1.005 {
					consecutiveDays++
				} else {
					break
				}
			}
			r.ConsecutiveDays = consecutiveDays

			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(s)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		if hlType == "high" {
			return results[i].Price > results[j].Price
		}
		return results[i].Price < results[j].Price
	})

	if results == nil {
		results = []HLResult{}
	}

	return results, nil
}

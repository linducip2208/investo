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

type SectorRotation struct {
	SectorName     string  `json:"sector_name"`
	WeeklyReturn   float64 `json:"weekly_return"`
	MonthlyReturn  float64 `json:"monthly_return"`
	QuarterlyReturn float64 `json:"quarterly_return"`
	MomentumScore  float64 `json:"momentum_score"`
	Trend          string  `json:"trend"`
	StockCount     int     `json:"stock_count"`
}

type SectorRotationService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	SectorRepo     *repository.SectorRepository
}

func (s *SectorRotationService) Analyze() ([]SectorRotation, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("SectorRotation Analyze: %w", err)
	}

	sectors, err := s.SectorRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("SectorRotation sectors: %w", err)
	}

	if len(stocks) == 0 || len(sectors) == 0 {
		return nil, fmt.Errorf("insufficient data")
	}

	now := time.Now()
	startQuarter := now.AddDate(0, -4, 0)
	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	prices, err := s.StockPriceRepo.FindByDateRange(stockIDs, startQuarter, now)
	if err != nil {
		return nil, fmt.Errorf("SectorRotation prices: %w", err)
	}

	priceMap := make(map[int64][]model.StockPrice)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p)
	}

	sectorStocks := make(map[string][]model.Stock)
	for _, st := range stocks {
		for _, sec := range sectors {
			if st.SectorID == sec.ID {
				sectorStocks[sec.Name] = append(sectorStocks[sec.Name], st)
				break
			}
		}
	}

	type sectorResult struct {
		name  string
		rot   *SectorRotation
		err   error
	}

	resultCh := make(chan sectorResult, len(sectorStocks))
	var wg sync.WaitGroup

	for sectorName, sectorStockList := range sectorStocks {
		wg.Add(1)
		go func(name string, stockList []model.Stock) {
			defer wg.Done()
			rot, err := s.analyzeSector(name, stockList, priceMap, now)
			resultCh <- sectorResult{name: name, rot: rot, err: err}
		}(sectorName, sectorStockList)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var rotations []SectorRotation
	for r := range resultCh {
		if r.err == nil && r.rot != nil {
			rotations = append(rotations, *r.rot)
		}
	}

	sort.Slice(rotations, func(i, j int) bool {
		return rotations[i].MomentumScore > rotations[j].MomentumScore
	})

	if len(rotations) > 0 {
		maxScore := rotations[0].MomentumScore
		minScore := rotations[len(rotations)-1].MomentumScore
		for i := range rotations {
			if rotations[i].MomentumScore == maxScore && maxScore > 0 {
				rotations[i].Trend = "Leading"
			} else if rotations[i].MomentumScore == minScore && minScore < maxScore {
				rotations[i].Trend = "Lagging"
			} else {
				rotations[i].Trend = "Neutral"
			}
		}
	}

	return rotations, nil
}

func (s *SectorRotationService) analyzeSector(name string, stocks []model.Stock, priceMap map[int64][]model.StockPrice, now time.Time) (*SectorRotation, error) {
	if len(stocks) == 0 {
		return nil, nil
	}

	var weeklyReturns, monthlyReturns, quarterlyReturns []float64

	for _, st := range stocks {
		prices, ok := priceMap[st.ID]
		if !ok || len(prices) < 2 {
			continue
		}

		sort.Slice(prices, func(i, j int) bool {
			return prices[i].Date.Before(prices[j].Date)
		})

		latest := prices[len(prices)-1].Close

		weekAgo := findPriceAt(prices, now.AddDate(0, 0, -7))
		monthAgo := findPriceAt(prices, now.AddDate(0, -1, 0))
		quarterAgo := findPriceAt(prices, now.AddDate(0, -3, 0))

		if weekAgo > 0 && latest > 0 {
			weeklyReturns = append(weeklyReturns, (latest-weekAgo)/weekAgo*100)
		}
		if monthAgo > 0 && latest > 0 {
			monthlyReturns = append(monthlyReturns, (latest-monthAgo)/monthAgo*100)
		}
		if quarterAgo > 0 && latest > 0 {
			quarterlyReturns = append(quarterlyReturns, (latest-quarterAgo)/quarterAgo*100)
		}
	}

	if len(weeklyReturns) == 0 {
		return nil, nil
	}

	avgW := avgVal(weeklyReturns)
	avgM := avgVal(monthlyReturns)
	avgQ := avgVal(quarterlyReturns)

	momentumScore := 0.0
	if avgW > 0 {
		momentumScore += 30
	}
	if avgM > 0 {
		momentumScore += 35
	}
	if avgQ > 0 {
		momentumScore += 35
	}
	magnitude := (math.Abs(avgW)*0.5 + math.Abs(avgM)*0.3 + math.Abs(avgQ)*0.2)
	momentumScore = math.Min(100, momentumScore+math.Min(30, magnitude*3))

	return &SectorRotation{
		SectorName:      name,
		WeeklyReturn:    math.Round(avgW*100) / 100,
		MonthlyReturn:   math.Round(avgM*100) / 100,
		QuarterlyReturn: math.Round(avgQ*100) / 100,
		MomentumScore:   math.Round(momentumScore*100) / 100,
		Trend:           "Neutral",
		StockCount:      len(stocks),
	}, nil
}

func findPriceAt(prices []model.StockPrice, target time.Time) float64 {
	minDiff := time.Hour * 24 * 365
	var closest float64
	for _, p := range prices {
		diff := p.Date.Sub(target)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closest = p.Close
		}
	}
	return closest
}

type LifecycleStage struct {
	Sector          string  `json:"sector"`
	Stage           string  `json:"stage"`
	RevenueGrowth   float64 `json:"revenue_growth"`
	InnovationIndex float64 `json:"innovation_index"`
	CompetitionLevel float64 `json:"competition_level"`
	Description     string  `json:"description"`
}

func (s *SectorRotationService) AnalyzeLifecycle() ([]LifecycleStage, error) {
	rotations, err := s.Analyze()
	if err != nil {
		return nil, err
	}

	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	sectors, err := s.SectorRepo.FindAll()
	if err != nil {
		return nil, err
	}

	sectorStocks := make(map[string][]model.Stock)
	for _, st := range stocks {
		for _, sec := range sectors {
			if st.SectorID == sec.ID {
				sectorStocks[sec.Name] = append(sectorStocks[sec.Name], st)
				break
			}
		}
	}

	var stages []LifecycleStage
	for _, rot := range rotations {
		stList := sectorStocks[rot.SectorName]
		stockCount := len(stList)

		revGrowth := rot.QuarterlyReturn
		if rot.MomentumScore > 70 {
			revGrowth += 10
		}

		innovationIdx := 0.0
		if stockCount > 5 {
			innovationIdx = math.Min(100, float64(stockCount)*3.0)
		} else {
			innovationIdx = float64(stockCount) * 8.0
		}

		competitionLvl := math.Min(100, float64(stockCount)*8.0)
		if stockCount > 10 {
			competitionLvl = 85
		} else if stockCount > 5 {
			competitionLvl = 60
		}

		stage := "Mature"
		desc := "Stable growth with consolidated market structure"
		if rot.MomentumScore > 65 && stockCount > 5 {
			stage = "Growth"
			desc = "High revenue growth with many new entrants expanding the market"
		} else if rot.MomentumScore < 25 && rot.QuarterlyReturn < 0 {
			stage = "Decline"
			desc = "Negative growth trajectory with shrinking market size"
		}

		stages = append(stages, LifecycleStage{
			Sector:           rot.SectorName,
			Stage:            stage,
			RevenueGrowth:    math.Round(revGrowth*100) / 100,
			InnovationIndex:  math.Round(innovationIdx*100) / 100,
			CompetitionLevel: math.Round(competitionLvl*100) / 100,
			Description:      desc,
		})
	}

	if stages == nil {
		stages = []LifecycleStage{}
	}

	return stages, nil
}

func avgVal(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

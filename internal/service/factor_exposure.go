package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type FactorExposure struct {
	StockCode  string  `json:"stock_code"`
	StockName  string  `json:"stock_name"`
	Value      float64 `json:"value"`
	Size       float64 `json:"size"`
	Momentum   float64 `json:"momentum"`
	Quality    float64 `json:"quality"`
	LowVol     float64 `json:"low_vol"`
	Growth     float64 `json:"growth"`
	Composite  float64 `json:"composite"`
}

type FactorTiming struct {
	Factor       string  `json:"factor"`
	CurrentScore float64 `json:"current_score"`
	Signal       string  `json:"signal"`
	Rationale    string  `json:"rationale"`
}

type FactorExposureService struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
}

func (s *FactorExposureService) CalculateExposure(code string) (*FactorExposure, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("FactorExposure: stock not found: %w", err)
	}

	fund, err := s.StockFundamentalRepo.FindLatest(stock.ID)
	if err != nil {
		fund = &model.StockFundamental{}
	}

	now := time.Now()
	start6M := now.AddDate(0, -6, 0)
	prices, err := s.StockPriceRepo.FindByStockDate(stock.ID, start6M, now)
	if err != nil || len(prices) < 30 {
		prices = nil
	}

	exp := &FactorExposure{
		StockCode: stock.Code,
		StockName: stock.Name,
	}

	exp.Value = s.calcValue(fund)
	exp.Size = s.calcSize(stock)
	exp.Momentum = s.calcMomentum(prices)
	exp.Quality = s.calcQuality(fund)
	exp.LowVol = s.calcLowVol(prices)
	exp.Growth = s.calcGrowth(fund)

	exp.Composite = math.Round((exp.Value+exp.Size+exp.Momentum+exp.Quality+exp.LowVol+exp.Growth)/6*100) / 100

	return exp, nil
}

func (s *FactorExposureService) calcValue(f *model.StockFundamental) float64 {
	score := 0.0
	if f.PER > 0 && f.PER < 15 {
		score += 30
	} else if f.PER > 0 && f.PER < 20 {
		score += 15
	}
	if f.PBV > 0 && f.PBV < 1.5 {
		score += 40
	} else if f.PBV > 0 && f.PBV < 2.5 {
		score += 20
	}
	if f.DividendYield > 3 {
		score += 30
	} else if f.DividendYield > 1 {
		score += 15
	}
	return math.Min(100, math.Round(score*10)/10)
}

func (s *FactorExposureService) calcSize(stock *model.Stock) float64 {
	shares := float64(stock.SharesOutstanding)
	if shares <= 0 {
		return 50
	}
	switch {
	case shares > 100_000_000_000:
		return 0
	case shares > 50_000_000_000:
		return 20
	case shares > 10_000_000_000:
		return 50
	case shares > 1_000_000_000:
		return 80
	default:
		return 100
	}
}

func (s *FactorExposureService) calcMomentum(prices []model.StockPrice) float64 {
	if len(prices) < 2 {
		return 50
	}
	latest := prices[len(prices)-1].Close
	halfIdx := len(prices) / 2
	if halfIdx >= len(prices) {
		halfIdx = len(prices) - 1
	}
	midPoint := prices[halfIdx].Close
	if midPoint <= 0 {
		return 50
	}
	ret := ((latest - midPoint) / midPoint) * 100
	score := math.Min(100, math.Max(0, 50+ret*3))
	return math.Round(score*10) / 10
}

func (s *FactorExposureService) calcQuality(f *model.StockFundamental) float64 {
	score := 0.0
	if f.ROE > 20 {
		score += 40
	} else if f.ROE > 10 {
		score += 25
	}
	if f.DER < 1.0 {
		score += 30
	} else if f.DER < 2.0 {
		score += 15
	}
	if f.NetProfitMargin > 10 {
		score += 30
	} else if f.NetProfitMargin > 5 {
		score += 15
	}
	return math.Min(100, math.Round(score*10)/10)
}

func (s *FactorExposureService) calcLowVol(prices []model.StockPrice) float64 {
	if len(prices) < 30 {
		return 50
	}
	score := 0.0
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1].Close > 0 {
			returns[i-1] = (prices[i].Close - prices[i-1].Close) / prices[i-1].Close * 100
		}
	}
	dailyVol := stdDevVal(returns)
	if dailyVol < 1.5 {
		score += 50
	} else if dailyVol < 3.0 {
		score += 30
	} else {
		score += 10
	}
	beta := s.estimateBeta(returns)
	if beta < 0.8 {
		score += 50
	} else if beta < 1.2 {
		score += 30
	} else {
		score += 10
	}
	return math.Min(100, math.Round(score*10)/10)
}

func (s *FactorExposureService) estimateBeta(returns []float64) float64 {
	if len(returns) < 2 {
		return 1.0
	}
	var sumR, sumR2 float64
	for _, r := range returns {
		sumR += r
		sumR2 += r * r
	}
	n := float64(len(returns))
	variance := (sumR2 - sumR*sumR/n) / (n - 1)
	if variance <= 0 {
		return 1.0
	}
	annualizedVol := math.Sqrt(variance*252) / 100
	if annualizedVol < 0.15 {
		return 0.5
	} else if annualizedVol < 0.25 {
		return 0.8
	} else if annualizedVol < 0.35 {
		return 1.2
	}
	return 1.8
}

func (s *FactorExposureService) calcGrowth(f *model.StockFundamental) float64 {
	score := 0.0
	if f.EPS > 0 {
		epsGrowth := 0.0
		if f.Revenue > 0 {
			epsGrowth = 10.0
		}
		if epsGrowth > 15 {
			score += 50
		} else if epsGrowth > 5 {
			score += 30
		} else {
			score += 10
		}
	}
	revGrowth := 0.0
	if f.Revenue > 0 {
		revGrowth = 10.0
	}
	if revGrowth > 10 {
		score += 50
	} else if revGrowth > 5 {
		score += 30
	} else {
		score += 10
	}
	return math.Min(100, math.Round(score*10)/10)
}

func (s *FactorExposureService) GetAllExposures() ([]FactorExposure, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("FactorExposure GetAllExposures: %w", err)
	}

	now := time.Now()
	start6M := now.AddDate(0, -6, 0)
	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	prices, _ := s.StockPriceRepo.FindByDateRange(stockIDs, start6M, now)
	priceMap := make(map[int64][]model.StockPrice)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p)
	}

	stockMap := make(map[string]model.Stock)
	for _, st := range stocks {
		stockMap[st.Code] = st
	}

	allFunds, _ := s.getAllFundamentals(stockIDs)
	fundMap := make(map[int64]*model.StockFundamental)
	for _, f := range allFunds {
		fundMap[f.StockID] = f
	}

	var exposures []FactorExposure
	for _, st := range stocks {
		fund := fundMap[st.ID]
		if fund == nil {
			fund = &model.StockFundamental{}
		}
		prices := priceMap[st.ID]

		exp := FactorExposure{
			StockCode: st.Code,
			StockName: st.Name,
		}
		exp.Value = s.calcValue(fund)
		exp.Size = s.calcSize(&st)
		exp.Momentum = s.calcMomentum(prices)
		exp.Quality = s.calcQuality(fund)
		exp.LowVol = s.calcLowVol(prices)
		exp.Growth = s.calcGrowth(fund)
		exp.Composite = math.Round((exp.Value+exp.Size+exp.Momentum+exp.Quality+exp.LowVol+exp.Growth)/6*100) / 100

		exposures = append(exposures, exp)
	}

	sort.Slice(exposures, func(i, j int) bool {
		return exposures[i].Composite > exposures[j].Composite
	})

	return exposures, nil
}

func (s *FactorExposureService) getAllFundamentals(stockIDs []int64) ([]*model.StockFundamental, error) {
	var funds []*model.StockFundamental
	for _, id := range stockIDs {
		f, err := s.StockFundamentalRepo.FindLatest(id)
		if err == nil && f != nil {
			funds = append(funds, f)
		}
	}
	return funds, nil
}

func (s *FactorExposureService) GetFactorTiming() ([]FactorTiming, error) {
	exposures, err := s.GetAllExposures()
	if err != nil {
		return nil, fmt.Errorf("FactorTiming: %w", err)
	}

	factors := []string{"Value", "Size", "Momentum", "Quality", "LowVol", "Growth"}

	topN := int(math.Max(1, float64(len(exposures))/5.0))

	type factorScore struct {
		factor string
		score  float64
		avg    float64
	}

	var factorScores []factorScore

	for _, factor := range factors {
		sort.Slice(exposures, func(i, j int) bool {
			switch factor {
			case "Value":
				return exposures[i].Value > exposures[j].Value
			case "Size":
				return exposures[i].Size > exposures[j].Size
			case "Momentum":
				return exposures[i].Momentum > exposures[j].Momentum
			case "Quality":
				return exposures[i].Quality > exposures[j].Quality
			case "LowVol":
				return exposures[i].LowVol > exposures[j].LowVol
			case "Growth":
				return exposures[i].Growth > exposures[j].Growth
			}
			return false
		})

		var sum float64
		limit := int(math.Min(float64(topN), float64(len(exposures))))
		for i := 0; i < limit; i++ {
			switch factor {
			case "Value":
				sum += exposures[i].Value
			case "Size":
				sum += exposures[i].Size
			case "Momentum":
				sum += exposures[i].Momentum
			case "Quality":
				sum += exposures[i].Quality
			case "LowVol":
				sum += exposures[i].LowVol
			case "Growth":
				sum += exposures[i].Growth
			}
		}

		avg := 0.0
		if limit > 0 {
			avg = math.Round(sum/float64(limit)*10) / 10
		}

		factorScores = append(factorScores, factorScore{
			factor: factor,
			score:  avg,
			avg:    avg,
		})
	}

	sort.Slice(factorScores, func(i, j int) bool {
		return factorScores[i].score > factorScores[j].score
	})

	var timings []FactorTiming

	allAvg := 0.0
	for _, fs := range factorScores {
		allAvg += fs.avg
	}
	allAvg /= float64(len(factorScores))

	for _, fs := range factorScores {
		signal := "Neutral"
		rationale := ""
		if fs.score > allAvg*1.2 {
			signal = "Overweight"
			rationale = fmt.Sprintf("%s factor is outperforming: top-quintile stocks average %.1f vs overall average %.1f", fs.factor, fs.score, allAvg)
		} else if fs.score < allAvg*0.8 {
			signal = "Underweight"
			rationale = fmt.Sprintf("%s factor is underperforming: top-quintile stocks average %.1f vs overall average %.1f", fs.factor, fs.score, allAvg)
		} else {
			signal = "Neutral"
			rationale = fmt.Sprintf("%s factor is performing near market average", fs.factor)
		}

		timings = append(timings, FactorTiming{
			Factor:       fs.factor,
			CurrentScore: fs.score,
			Signal:       signal,
			Rationale:    rationale,
		})
	}

	return timings, nil
}

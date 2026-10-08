package service

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type PortfolioPerformance struct {
	TotalValue        float64               `json:"total_value"`
	TotalCost         float64               `json:"total_cost"`
	TotalGain         float64               `json:"total_gain"`
	TotalGainPercent  float64               `json:"total_gain_percent"`
	DailyChange       float64               `json:"daily_change"`
	DailyChangePercent float64              `json:"daily_change_percent"`
	Holdings          []HoldingPerformance  `json:"holdings"`
}

type HoldingPerformance struct {
	Stock            model.Stock `json:"stock"`
	Quantity         float64     `json:"quantity"`
	AvgPrice         float64     `json:"avg_price"`
	CurrentPrice     float64     `json:"current_price"`
	MarketValue      float64     `json:"market_value"`
	CostBasis        float64     `json:"cost_basis"`
	Gain             float64     `json:"gain"`
	GainPercent      float64     `json:"gain_percent"`
	Weight           float64     `json:"weight"`
	DayChange        float64     `json:"day_change"`
	DayChangePercent float64     `json:"day_change_percent"`
}

type FrontierPoint struct {
	Risk   float64 `json:"risk"`
	Return float64 `json:"return"`
}

type Attribution struct {
	TopGainers      []HoldingPerformance `json:"top_gainers"`
	TopLosers       []HoldingPerformance `json:"top_losers"`
	SectorAllocation map[string]float64  `json:"sector_allocation"`
	BestDay         string               `json:"best_day"`
	WorstDay        string               `json:"worst_day"`
	WinRate         float64              `json:"win_rate"`
}

type PortfolioAnalytics struct {
	StockPriceRepo        *repository.StockPriceRepository
	PortfolioItemRepo     *repository.PortfolioItemRepository
	PortfolioRepo         *repository.PortfolioRepository
	SectorRepo            *repository.SectorRepository
	StockRepo             *repository.StockRepository
	StockFundamentalRepo  *repository.StockFundamentalRepository
}

func (a *PortfolioAnalytics) CalcPerformance(portfolioID int64) (*PortfolioPerformance, error) {
	items, err := a.PortfolioItemRepo.GetWithStock(portfolioID)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return &PortfolioPerformance{}, nil
	}

	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.Stock.ID)
	}

	latestPrices, err := a.StockPriceRepo.GetLatestPrices(stockIDs)
	if err != nil {
		return nil, err
	}

	prevPrices := make(map[int64]float64)
	for _, stockID := range stockIDs {
		prices, err := a.StockPriceRepo.FindLatest(stockID, 2)
		if err == nil && len(prices) >= 2 {
			prevPrices[stockID] = prices[1].Close
		} else {
			prevPrices[stockID] = latestPrices[stockID]
		}
	}

	perf := &PortfolioPerformance{}
	var holdings []HoldingPerformance

	for _, item := range items {
		currentPrice := latestPrices[item.Stock.ID]
		costBasis := item.Item.Quantity * item.Item.AvgPrice
		marketValue := item.Item.Quantity * currentPrice
		gain := marketValue - costBasis
		gainPercent := 0.0
		if costBasis > 0 {
			gainPercent = (gain / costBasis) * 100
		}

		prevPrice := prevPrices[item.Stock.ID]
		dayChange := 0.0
		dayChangePercent := 0.0
		if prevPrice > 0 {
			dayChange = (currentPrice - prevPrice) * item.Item.Quantity
			dayChangePercent = ((currentPrice - prevPrice) / prevPrice) * 100
		}

		perf.TotalValue += marketValue
		perf.TotalCost += costBasis
		perf.DailyChange += dayChange

		holdings = append(holdings, HoldingPerformance{
			Stock:            item.Stock,
			Quantity:         item.Item.Quantity,
			AvgPrice:         item.Item.AvgPrice,
			CurrentPrice:     currentPrice,
			MarketValue:      marketValue,
			CostBasis:        costBasis,
			Gain:             gain,
			GainPercent:      gainPercent,
			Weight:           0,
			DayChange:        dayChange,
			DayChangePercent: dayChangePercent,
		})
	}

	perf.TotalGain = perf.TotalValue - perf.TotalCost
	if perf.TotalCost > 0 {
		perf.TotalGainPercent = (perf.TotalGain / perf.TotalCost) * 100
	}
	if perf.TotalValue > 0 {
		perf.DailyChangePercent = (perf.DailyChange / (perf.TotalValue - perf.DailyChange)) * 100
	}

	for i := range holdings {
		if perf.TotalValue > 0 {
			holdings[i].Weight = (holdings[i].MarketValue / perf.TotalValue) * 100
		}
	}

	perf.Holdings = holdings
	return perf, nil
}

func (a *PortfolioAnalytics) CalcEfficientFrontier(portfolioID int64) ([]FrontierPoint, error) {
	items, err := a.PortfolioItemRepo.GetWithStock(portfolioID)
	if err != nil {
		return nil, err
	}

	if len(items) < 2 {
		return nil, nil
	}

	type stockReturns struct {
		StockID  int64
		Returns  []float64
		AvgRet   float64
		StdDev   float64
	}

	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.Stock.ID)
	}

	prices, err := a.StockPriceRepo.FindByDateRange(stockIDs, start, end)
	if err != nil {
		return nil, err
	}

	priceMap := make(map[int64][]model.StockPrice)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p)
	}

	var returnsList []stockReturns
	for _, item := range items {
		sp, ok := priceMap[item.Stock.ID]
		if !ok || len(sp) < 2 {
			continue
		}
		var retValues []float64
		for i := 1; i < len(sp); i++ {
			if sp[i-1].Close > 0 {
				r := (sp[i].Close - sp[i-1].Close) / sp[i-1].Close
				retValues = append(retValues, r)
			}
		}
		if len(retValues) < 2 {
			continue
		}
		avg := 0.0
		for _, r := range retValues {
			avg += r
		}
		avg /= float64(len(retValues))
		variance := 0.0
		for _, r := range retValues {
			variance += (r - avg) * (r - avg)
		}
		stddev := math.Sqrt(variance / float64(len(retValues)))
		returnsList = append(returnsList, stockReturns{
			StockID: item.Stock.ID,
			Returns: retValues,
			AvgRet:  avg,
			StdDev:  stddev,
		})
	}

	if len(returnsList) < 2 {
		return nil, nil
	}

	n := len(returnsList)
	var points []FrontierPoint

	for iteration := 0; iteration < 500; iteration++ {
		weights := make([]float64, n)
		sum := 0.0
		for i := 0; i < n; i++ {
			w := rand.Float64()
			weights[i] = w
			sum += w
		}
		for i := 0; i < n; i++ {
			weights[i] /= sum
		}

		portfolioReturn := 0.0
		for i := 0; i < n; i++ {
			portfolioReturn += weights[i] * returnsList[i].AvgRet
		}

		portfolioRisk := 0.0
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				cov := covariance(returnsList[i].Returns, returnsList[j].Returns)
				portfolioRisk += weights[i] * weights[j] * cov
			}
		}
		portfolioRisk = math.Sqrt(portfolioRisk)

		points = append(points, FrontierPoint{
			Risk:   portfolioRisk * 100,
			Return: portfolioReturn * 100 * 252,
		})
	}

	return points, nil
}

func covariance(x, y []float64) float64 {
	minLen := len(x)
	if len(y) < minLen {
		minLen = len(y)
	}
	if minLen < 2 {
		return 0
	}
	meanX := 0.0
	meanY := 0.0
	for i := 0; i < minLen; i++ {
		meanX += x[i]
		meanY += y[i]
	}
	meanX /= float64(minLen)
	meanY /= float64(minLen)
	cov := 0.0
	for i := 0; i < minLen; i++ {
		cov += (x[i] - meanX) * (y[i] - meanY)
	}
	return cov / float64(minLen)
}

func (a *PortfolioAnalytics) CalcAttribution(portfolioID int64) (*Attribution, error) {
	perf, err := a.CalcPerformance(portfolioID)
	if err != nil {
		return nil, err
	}

	att := &Attribution{
		SectorAllocation: make(map[string]float64),
	}

	sorted := make([]HoldingPerformance, len(perf.Holdings))
	copy(sorted, perf.Holdings)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].GainPercent > sorted[j].GainPercent
	})

	for i := 0; i < 3 && i < len(sorted); i++ {
		att.TopGainers = append(att.TopGainers, sorted[i])
	}
	for i := len(sorted) - 1; i >= 0 && i >= len(sorted)-3; i-- {
		if sorted[i].GainPercent < 0 {
			att.TopLosers = append(att.TopLosers, sorted[i])
		}
	}

	sectors, err := a.SectorRepo.FindAll()
	if err == nil {
		sectorMap := make(map[int64]string)
		for _, s := range sectors {
			sectorMap[s.ID] = s.Name
		}
		for _, h := range perf.Holdings {
			sectorName := sectorMap[h.Stock.SectorID]
			if sectorName == "" {
				sectorName = "Lainnya"
			}
			att.SectorAllocation[sectorName] += h.Weight
		}
	}

	var stockIDs []int64
	for _, h := range perf.Holdings {
		stockIDs = append(stockIDs, h.Stock.ID)
	}

	if len(stockIDs) > 0 {
		end := time.Now()
		start := end.AddDate(0, -3, 0)
		prices, err := a.StockPriceRepo.FindByDateRange(stockIDs, start, end)
		if err == nil && len(prices) > 0 {
			dateMap := make(map[string]float64)
			for _, p := range prices {
				d := p.Date.Format("2006-01-02")
				prevClose := 0.0
				if _, ok := dateMap[d]; ok {
					dateMap[d] = prevClose
					continue
				}
				dateMap[d] = dateMap[d] + p.Close
			}

			var dates []string
			for d := range dateMap {
				dates = append(dates, d)
			}
			sort.Strings(dates)

			if len(dates) >= 2 {
				bestDay := dates[0]
				worstDay := dates[0]
				bestChange := -999999.0
				worstChange := 999999.0
				upDays := 0

				for i := 1; i < len(dates); i++ {
					if i > 0 {
						change := 0.0
						if dateMap[dates[i-1]] > 0 {
							change = (dateMap[dates[i]] - dateMap[dates[i-1]]) / dateMap[dates[i-1]]
						}
						if change > 0 {
							upDays++
						}
						if change > bestChange {
							bestChange = change
							bestDay = dates[i]
						}
						if change < worstChange {
							worstChange = change
							worstDay = dates[i]
						}
					}
				}

				att.BestDay = bestDay
				att.WorstDay = worstDay
				att.WinRate = float64(upDays) / float64(len(dates)-1) * 100
			}
		}
	}

	return att, nil
}

type MonteCarloResult struct {
	Simulations     int               `json:"simulations"`
	MedianReturn    float64           `json:"median_return"`
	Percentile5     float64           `json:"percentile_5"`
	Percentile95    float64           `json:"percentile_95"`
	ProbPositive    float64           `json:"prob_positive"`
	ProjectedValues []float64         `json:"projected_values"`
	Distribution    []DistributionBin `json:"distribution"`
}

type DistributionBin struct {
	Return float64 `json:"return"`
	Count  int     `json:"count"`
}

func (a *PortfolioAnalytics) RunMonteCarlo(portfolioID int64, simulations int, years int) (*MonteCarloResult, error) {
	if simulations <= 0 {
		simulations = 1000
	}
	if years <= 0 {
		years = 1
	}

	perf, err := a.CalcPerformance(portfolioID)
	if err != nil {
		return nil, err
	}
	if len(perf.Holdings) == 0 {
		return &MonteCarloResult{Simulations: simulations}, nil
	}

	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	var stockIDs []int64
	for _, h := range perf.Holdings {
		stockIDs = append(stockIDs, h.Stock.ID)
	}

	prices, err := a.StockPriceRepo.FindByDateRange(stockIDs, start, end)
	if err != nil {
		return nil, err
	}

	priceMap := make(map[int64][]float64)
	for _, p := range prices {
		priceMap[p.StockID] = append(priceMap[p.StockID], p.Close)
	}

	type holdingReturns struct {
		Weight   float64
		Returns  []float64
	}

	var allReturns []holdingReturns
	for _, h := range perf.Holdings {
		closes, ok := priceMap[h.Stock.ID]
		if !ok || len(closes) < 2 {
			continue
		}
		var rets []float64
		for i := 1; i < len(closes); i++ {
			ret := (closes[i] - closes[i-1]) / closes[i-1]
			rets = append(rets, ret)
		}
		if len(rets) == 0 {
			continue
		}
		weight := 0.0
		if perf.TotalValue > 0 {
			weight = h.MarketValue / perf.TotalValue
		}
		allReturns = append(allReturns, holdingReturns{
			Weight:  weight,
			Returns: rets,
		})
	}

	if len(allReturns) == 0 {
		return &MonteCarloResult{Simulations: simulations}, nil
	}

	tradingDays := 252 * years
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var finalValues []float64
	positiveCount := 0

	for sim := 0; sim < simulations; sim++ {
		totalValue := perf.TotalValue
		for day := 0; day < tradingDays; day++ {
			dayReturn := 0.0
			for _, hr := range allReturns {
				idx := rng.Intn(len(hr.Returns))
				dayReturn += hr.Weight * hr.Returns[idx]
			}
			totalValue *= (1 + dayReturn)
		}
		finalValue := (totalValue - perf.TotalValue) / perf.TotalValue * 100
		finalValues = append(finalValues, finalValue)
		if finalValue > 0 {
			positiveCount++
		}
	}

	sort.Float64s(finalValues)

	result := &MonteCarloResult{
		Simulations:     simulations,
		ProjectedValues: finalValues,
	}

	n := len(finalValues)
	if n > 0 {
		result.ProbPositive = float64(positiveCount) / float64(n) * 100
		if n%2 == 0 {
			result.MedianReturn = (finalValues[n/2-1] + finalValues[n/2]) / 2
		} else {
			result.MedianReturn = finalValues[n/2]
		}
		p5Idx := int(float64(n) * 0.05)
		if p5Idx >= n {
			p5Idx = n - 1
		}
		result.Percentile5 = finalValues[p5Idx]
		p95Idx := int(float64(n) * 0.95)
		if p95Idx >= n {
			p95Idx = n - 1
		}
		result.Percentile95 = finalValues[p95Idx]
	}

	numBins := 50
	if n > 0 {
		minVal := finalValues[0]
		maxVal := finalValues[n-1]
		binWidth := (maxVal - minVal) / float64(numBins)
		if binWidth <= 0 {
			binWidth = 1
		}
		bins := make([]int, numBins)
		for _, v := range finalValues {
			binIdx := int((v - minVal) / binWidth)
			if binIdx >= numBins {
				binIdx = numBins - 1
			}
			if binIdx < 0 {
				binIdx = 0
			}
			bins[binIdx]++
		}
		for i := 0; i < numBins; i++ {
			result.Distribution = append(result.Distribution, DistributionBin{
				Return: minVal + binWidth*float64(i) + binWidth/2,
				Count:  bins[i],
			})
		}
	}

	return result, nil
}

type StressTestResult struct {
	Scenario          string  `json:"scenario"`
	PortfolioChange   float64 `json:"portfolio_change"`
	IHSGChange        float64 `json:"ihsg_change"`
	ImpactDescription string  `json:"impact_description"`
}

var stressScenarios = map[string]struct {
	IHSGChange     float64
	SectorImpacts  map[string]float64
	Description    string
}{
	"2008_crash": {
		IHSGChange: -50,
		SectorImpacts: map[string]float64{
			"Keuangan":          -65,
			"Properti":          -70,
			"Infrastruktur":     -45,
			"Pertambangan":      -55,
			"Konsumsi":          -35,
			"Pertanian":         -30,
		},
		Description: "Krisis finansial global 2008. IHSG anjlok -50%, sektor keuangan dan properti paling terdampak.",
	},
	"covid_crash": {
		IHSGChange: -37,
		SectorImpacts: map[string]float64{
			"Pariwisata":        -70,
			"Transportasi":      -65,
			"Properti":          -50,
			"Keuangan":          -40,
			"Konsumsi":          -25,
			"Kesehatan":         15,
			"Teknologi":         10,
		},
		Description: "Pandemi COVID-19 2020. IHSG -37%, IDR melemah 15%. Sektor pariwisata & transportasi terpukul, kesehatan & teknologi naik.",
	},
	"1998_crisis": {
		IHSGChange: -60,
		SectorImpacts: map[string]float64{
			"Keuangan":          -90,
			"Properti":          -85,
			"Infrastruktur":     -60,
			"Konsumsi":          -50,
			"Pertambangan":      -45,
			"Pertanian":         -30,
		},
		Description: "Krisis moneter 1998. IHSG -60%, IDR -80%. Sektor perbankan kolaps, properti jatuh, terjadi rush besar-besaran.",
	},
	"interest_hike": {
		IHSGChange: -15,
		SectorImpacts: map[string]float64{
			"Keuangan":          -15,
			"Properti":          -25,
			"Infrastruktur":     -18,
			"Konsumsi":          -10,
			"Teknologi":         -12,
		},
		Description: "BI rate naik 2%. Sektor properti dan perbankan tertekan karena kenaikan biaya pinjaman.",
	},
	"commodity_boom": {
		IHSGChange: 20,
		SectorImpacts: map[string]float64{
			"Pertambangan":      50,
			"Energi":            40,
			"Pertanian":         25,
			"Infrastruktur":     15,
			"Keuangan":          10,
			"Transportasi":      -5,
			"Konsumsi":          -8,
		},
		Description: "Commodity super cycle. Batu bara +50%, metal +30%. Sektor energi dan pertambangan melonjak, konsumsi tertekan inflasi.",
	},
}

func (a *PortfolioAnalytics) RunStressTest(portfolioID int64, scenario string) (*StressTestResult, error) {
	sc, ok := stressScenarios[scenario]
	if !ok {
		return nil, fmt.Errorf("skenario tidak ditemukan: %s", scenario)
	}

	perf, err := a.CalcPerformance(portfolioID)
	if err != nil {
		return nil, err
	}

	result := &StressTestResult{
		Scenario:          scenario,
		IHSGChange:        sc.IHSGChange,
		ImpactDescription: sc.Description,
	}

	if len(perf.Holdings) == 0 {
		result.PortfolioChange = sc.IHSGChange
		result.ImpactDescription = "Portfolio kosong, dampak mengikuti IHSG."
		return result, nil
	}

	sectors, err := a.SectorRepo.FindAll()
	if err != nil {
		return nil, err
	}
	sectorMap := make(map[int64]string)
	for _, s := range sectors {
		sectorMap[s.ID] = s.Name
	}

	weightedImpact := 0.0
	totalWeight := 0.0

	for _, h := range perf.Holdings {
		sectorName := sectorMap[h.Stock.SectorID]
		impact := sc.IHSGChange
		if s, ok := sc.SectorImpacts[sectorName]; ok {
			impact = s
		}
		weightedImpact += impact * h.Weight
		totalWeight += h.Weight
	}

	if totalWeight > 0 {
		result.PortfolioChange = weightedImpact / totalWeight
	} else {
		result.PortfolioChange = sc.IHSGChange
	}

	if result.PortfolioChange < -50 {
		result.ImpactDescription = fmt.Sprintf("Dampak sangat berat: %.1f%%. Portfolio sangat terkonsentrasi di sektor yang paling terdampak. Rekomendasi: diversifikasi ulang.", result.PortfolioChange)
	} else if result.PortfolioChange < -20 {
		result.ImpactDescription = fmt.Sprintf("Dampak signifikan: %.1f%%. Portfolio rentan terhadap skenario ini. Pertimbangkan hedging atau realokasi aset.", result.PortfolioChange)
	} else if result.PortfolioChange < 0 {
		result.ImpactDescription = fmt.Sprintf("Dampak moderat: %.1f%%. Portfolio cukup tangguh, namun tetap terdampak secara umum.", result.PortfolioChange)
	} else {
		result.ImpactDescription = fmt.Sprintf("Dampak positif: +%.1f%%. Alokasi sektor portfolio diuntungkan dalam skenario ini.", result.PortfolioChange)
	}

	return result, nil
}

type TaxLossOpportunity struct {
	StockCode      string  `json:"stock_code"`
	StockName      string  `json:"stock_name"`
	CurrentPrice   float64 `json:"current_price"`
	AvgPrice       float64 `json:"avg_price"`
	Loss           float64 `json:"loss"`
	LossPercent    float64 `json:"loss_percent"`
	HoldingDays    int     `json:"holding_days"`
	Recommendation string  `json:"recommendation"`
}

func (a *PortfolioAnalytics) FindTaxLossOpportunities(portfolioID int64) ([]TaxLossOpportunity, error) {
	perf, err := a.CalcPerformance(portfolioID)
	if err != nil {
		return nil, err
	}

	var items []PortfolioItemWithStock
	rawItems, err := a.PortfolioItemRepo.GetWithStock(portfolioID)
	if err != nil {
		return nil, err
	}
	for _, ri := range rawItems {
		items = append(items, PortfolioItemWithStock{
			Item:  ri.Item,
			Stock: ri.Stock,
		})
	}
	_ = items

	var opportunities []TaxLossOpportunity
	now := time.Now()

	for _, h := range perf.Holdings {
		if h.CurrentPrice >= h.AvgPrice {
			continue
		}

		rawItems, _ := a.PortfolioItemRepo.GetWithStock(portfolioID)
		var holdingDays int
		for _, ri := range rawItems {
			if ri.Stock.ID == h.Stock.ID {
				holdingDays = int(now.Sub(ri.Item.CreatedAt).Hours() / 24)
				break
			}
		}

		if holdingDays <= 30 {
			continue
		}

		lossAmount := (h.AvgPrice - h.CurrentPrice) * h.Quantity
		lossPercent := (h.AvgPrice - h.CurrentPrice) / h.AvgPrice * 100

		recommendation := "Pertimbangkan hold"
		if lossPercent > 20 {
			recommendation = "Jual untuk realisasi rugi pajak"
		} else if lossPercent > 10 {
			recommendation = "Evaluasi: rugi signifikan, bisa direalisasi"
		}

		opportunities = append(opportunities, TaxLossOpportunity{
			StockCode:      h.Stock.Code,
			StockName:      h.Stock.Name,
			CurrentPrice:   h.CurrentPrice,
			AvgPrice:       h.AvgPrice,
			Loss:           lossAmount,
			LossPercent:    lossPercent,
			HoldingDays:    holdingDays,
			Recommendation: recommendation,
		})
	}

	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].Loss > opportunities[j].Loss
	})

	return opportunities, nil
}

type ProjectionResult struct {
	Years         int       `json:"years"`
	InitialEPS    float64   `json:"initial_eps"`
	GrowthRate    float64   `json:"growth_rate"`
	ProjectedEPS  []float64 `json:"projected_eps"`
	ProjectedPrice []float64 `json:"projected_price"`
	FairValue     float64   `json:"fair_value"`
}

func (a *PortfolioAnalytics) ProjectStock(stockID int64, growthRate, discountRate float64, years int) (*ProjectionResult, error) {
	if years <= 0 {
		years = 5
	}
	if growthRate < 0 {
		growthRate = 0
	}
	if discountRate < 0 {
		discountRate = 7
	}

	fundamental, err := a.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		return nil, fmt.Errorf("data fundamental tidak ditemukan: %w", err)
	}

	initialEPS := fundamental.EPS
	currentPER := fundamental.PER
	if currentPER <= 0 {
		stock, err := a.StockRepo.FindByID(stockID)
		if err != nil {
			return nil, err
		}
		prices, _ := a.StockPriceRepo.FindLatest(stockID, 1)
		if len(prices) > 0 && initialEPS > 0 {
			currentPER = prices[0].Close / initialEPS
		}
		_ = stock
		if currentPER <= 0 {
			currentPER = 15
		}
	}

	result := &ProjectionResult{
		Years:         years,
		InitialEPS:    initialEPS,
		GrowthRate:    growthRate,
		ProjectedEPS:  make([]float64, years),
		ProjectedPrice: make([]float64, years),
	}

	growthFactor := 1 + growthRate/100
	discountFactor := 1 + discountRate/100

	for y := 0; y < years; y++ {
		projectedEPS := initialEPS * math.Pow(growthFactor, float64(y+1))
		result.ProjectedEPS[y] = projectedEPS
		result.ProjectedPrice[y] = projectedEPS * currentPER
		result.FairValue += result.ProjectedPrice[y] / math.Pow(discountFactor, float64(y+1))
	}

	return result, nil
}

type PortfolioItemWithStock struct {
	Item  model.PortfolioItem
	Stock model.Stock
}

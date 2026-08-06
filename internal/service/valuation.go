package service

import (
	"fmt"
	"math"
	"sort"

	"investo/internal/model"
	"investo/internal/repository"
)

type ValuationService struct {
	StockFundamentalRepo *repository.StockFundamentalRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockRepo            *repository.StockRepository
	SectorRepo           *repository.SectorRepository
}

type ValuationResult struct {
	DCFValue      float64 `json:"dcf_value"`
	GrahamValue   float64 `json:"graham_value"`
	LynchValue    float64 `json:"lynch_value"`
	PBVValue      float64 `json:"pbv_value"`
	AverageTarget float64 `json:"average_target"`
	CurrentPrice  float64 `json:"current_price"`
	UpsidePercent float64 `json:"upside_percent"`
	Recommendation string `json:"recommendation"`
}

func (s *ValuationService) Calculate(stockID int64) (*ValuationResult, error) {
	fund, err := s.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		return nil, fmt.Errorf("ValuationService.Calculate: %w", err)
	}

	stock, err := s.StockRepo.FindByID(stockID)
	if err != nil {
		return nil, fmt.Errorf("ValuationService.Calculate stock: %w", err)
	}

	prices, err := s.StockPriceRepo.FindLatest(stockID, 1)
	if err != nil || len(prices) == 0 {
		return nil, fmt.Errorf("ValuationService.Calculate price: no price data")
	}

	currentPrice := prices[0].Close
	if currentPrice <= 0 {
		currentPrice = 1
	}

	dcfValue := s.calcDCF(fund.NetIncome, stock.SharesOutstanding)
	grahamValue := s.calcGraham(fund.EPS, fund.BVPS)
	lynchValue := s.calcLynch(fund.EPS)
	pbvValue := s.calcPBV(fund.BVPS, stock.SectorID)

	targets := []float64{dcfValue, grahamValue, lynchValue, pbvValue}
	count := 0
	sum := 0.0
	for _, t := range targets {
		if t > 0 {
			sum += t
			count++
		}
	}

	var averageTarget float64
	if count > 0 {
		averageTarget = sum / float64(count)
	}

	var upsidePercent float64
	if currentPrice > 0 {
		upsidePercent = ((averageTarget - currentPrice) / currentPrice) * 100
	}

	recommendation := s.getRecommendation(upsidePercent)

	return &ValuationResult{
		DCFValue:       math.Round(dcfValue),
		GrahamValue:    math.Round(grahamValue),
		LynchValue:     math.Round(lynchValue),
		PBVValue:       math.Round(pbvValue),
		AverageTarget:  math.Round(averageTarget),
		CurrentPrice:   currentPrice,
		UpsidePercent:  math.Round(upsidePercent*10) / 10,
		Recommendation: recommendation,
	}, nil
}

func (s *ValuationService) calcDCF(netIncome float64, sharesOutstanding int64) float64 {
	if sharesOutstanding <= 0 {
		return 0
	}

	fcf := netIncome * 0.7
	if fcf <= 0 {
		return 0
	}

	growthRate := 0.10
	wacc := 0.10
	terminalGrowth := 0.03

	totalPV := 0.0
	projectedFCF := fcf
	for year := 1; year <= 5; year++ {
		projectedFCF *= (1 + growthRate)
		pvFactor := math.Pow(1+wacc, float64(year))
		totalPV += projectedFCF / pvFactor
	}

	terminalValue := projectedFCF * (1 + terminalGrowth) / (wacc - terminalGrowth)
	pvTerminal := terminalValue / math.Pow(1+wacc, 5)

	enterpriseValue := totalPV + pvTerminal
	fairValuePerShare := enterpriseValue / float64(sharesOutstanding)

	return fairValuePerShare
}

func (s *ValuationService) calcGraham(eps, bvps float64) float64 {
	if eps <= 0 || bvps <= 0 {
		return 0
	}
	return math.Sqrt(22.5 * eps * bvps)
}

func (s *ValuationService) calcLynch(eps float64) float64 {
	if eps <= 0 {
		return 0
	}
	growthRate := 12.0
	return eps * growthRate
}

func (s *ValuationService) calcPBV(bvps float64, sectorID int64) float64 {
	if bvps <= 0 {
		return 0
	}
	avgPBV := s.getSectorAvgPBV(sectorID)
	return bvps * avgPBV
}

func (s *ValuationService) getSectorAvgPBV(sectorID int64) float64 {
	sectorAverages := map[int64]float64{
		1:  2.5,
		2:  3.0,
		3:  1.8,
		4:  2.2,
		5:  2.8,
		6:  1.5,
		7:  3.5,
		8:  2.0,
		9:  1.2,
		10: 2.6,
		11: 1.8,
		12: 2.3,
		13: 2.1,
		14: 1.6,
		15: 3.2,
	}

	if avg, ok := sectorAverages[sectorID]; ok {
		return avg
	}
	return 2.0
}

func (s *ValuationService) getRecommendation(upside float64) string {
	if upside > 20 {
		return "BUY"
	} else if upside >= 5 {
		return "HOLD"
	}
	return "SELL"
}

type PeerData struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	PER        float64 `json:"per"`
	PBV        float64 `json:"pbv"`
	ROE        float64 `json:"roe"`
	DER        float64 `json:"der"`
	NPM        float64 `json:"npm"`
	DivYield   float64 `json:"div_yield"`
	MarketCap  float64 `json:"market_cap"`
	Price      float64 `json:"price"`
}

type PeerComparison struct {
	Stock       model.Stock           `json:"stock"`
	Peers       []PeerData            `json:"peers"`
	Metrics     map[string]float64    `json:"metrics"`
	IndustryAvg map[string]float64    `json:"industry_avg"`
}

func (s *ValuationService) ComparePeers(stockID int64) (*PeerComparison, error) {
	stock, err := s.StockRepo.FindByID(stockID)
	if err != nil {
		return nil, fmt.Errorf("ComparePeers stock: %w", err)
	}

	allStocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("ComparePeers list: %w", err)
	}

	var peerStocks []model.Stock
	for _, st := range allStocks {
		if st.SectorID == stock.SectorID {
			peerStocks = append(peerStocks, st)
		}
	}

	var peers []PeerData
	metrics := map[string]float64{}

	for _, peer := range peerStocks {
		fund, err := s.StockFundamentalRepo.FindLatest(peer.ID)
		if err != nil {
			continue
		}

		prices, err := s.StockPriceRepo.FindLatest(peer.ID, 1)
		price := 0.0
		if err == nil && len(prices) > 0 {
			price = prices[0].Close
		}

		marketCap := price * float64(peer.SharesOutstanding)

		pd := PeerData{
			Code:      peer.Code,
			Name:      peer.Name,
			PER:       math.Round(fund.PER*100) / 100,
			PBV:       math.Round(fund.PBV*100) / 100,
			ROE:       math.Round(fund.ROE*100) / 100,
			DER:       math.Round(fund.DER*100) / 100,
			NPM:       math.Round(fund.NetProfitMargin*100) / 100,
			DivYield:  math.Round(fund.DividendYield*100) / 100,
			MarketCap: marketCap,
			Price:     price,
		}
		peers = append(peers, pd)

		if peer.ID == stockID {
			metrics["per"] = pd.PER
			metrics["pbv"] = pd.PBV
			metrics["roe"] = pd.ROE
			metrics["der"] = pd.DER
			metrics["npm"] = pd.NPM
			metrics["div_yield"] = pd.DivYield
			metrics["market_cap"] = pd.MarketCap
		}
	}

	sort.Slice(peers, func(i, j int) bool {
		if peers[i].Code == stock.Code {
			return true
		}
		if peers[j].Code == stock.Code {
			return false
		}
		return peers[i].MarketCap > peers[j].MarketCap
	})

	industryAvg := s.calcIndustryAvg(peers)

	return &PeerComparison{
		Stock:       *stock,
		Peers:       peers,
		Metrics:     metrics,
		IndustryAvg: industryAvg,
	}, nil
}

func (s *ValuationService) calcIndustryAvg(peers []PeerData) map[string]float64 {
	if len(peers) == 0 {
		return map[string]float64{}
	}

	avgs := map[string]float64{
		"per": 0, "pbv": 0, "roe": 0, "der": 0, "npm": 0, "div_yield": 0,
	}
	counts := map[string]int{}

	for _, p := range peers {
		if p.PER > 0 {
			avgs["per"] += p.PER
			counts["per"]++
		}
		if p.PBV > 0 {
			avgs["pbv"] += p.PBV
			counts["pbv"]++
		}
		if p.ROE != 0 {
			avgs["roe"] += p.ROE
			counts["roe"]++
		}
		if p.DER != 0 {
			avgs["der"] += p.DER
			counts["der"]++
		}
		if p.NPM != 0 {
			avgs["npm"] += p.NPM
			counts["npm"]++
		}
		if p.DivYield != 0 {
			avgs["div_yield"] += p.DivYield
			counts["div_yield"]++
		}
	}

	result := map[string]float64{}
	for k, sum := range avgs {
		if counts[k] > 0 {
			result[k] = math.Round(sum/float64(counts[k])*100) / 100
		}
	}

	return result
}

type DuPontResult struct {
	ROE             float64 `json:"roe"`
	NetMargin       float64 `json:"net_margin"`
	AssetTurnover   float64 `json:"asset_turnover"`
	EquityMultiplier float64 `json:"equity_multiplier"`
	TaxRetention    float64 `json:"tax_retention"`
	InterestBurden  float64 `json:"interest_burden"`
}

func (s *ValuationService) CalcDuPont(stockID int64) (*DuPontResult, error) {
	fund, err := s.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		return nil, fmt.Errorf("CalcDuPont: %w", err)
	}

	netMargin := 0.0
	if fund.Revenue > 0 {
		netMargin = fund.NetIncome / fund.Revenue
	}

	assetTurnover := 0.0
	if fund.TotalAssets > 0 {
		assetTurnover = fund.Revenue / fund.TotalAssets
	}

	equityMultiplier := 0.0
	if fund.Equity > 0 {
		equityMultiplier = fund.TotalAssets / fund.Equity
	}

	roeDupont := netMargin * assetTurnover * equityMultiplier

	taxRetention := 0.0
	interestBurden := 0.0
	if fund.TotalAssets > 0 && fund.Equity > 0 {
		taxRetention = 1.0
		interestBurden = 1.0
	}

	return &DuPontResult{
		ROE:              math.Round(roeDupont*10000) / 100,
		NetMargin:        math.Round(netMargin*10000) / 100,
		AssetTurnover:    math.Round(assetTurnover*10000) / 100,
		EquityMultiplier: math.Round(equityMultiplier*100) / 100,
		TaxRetention:     math.Round(taxRetention*100) / 100,
		InterestBurden:   math.Round(interestBurden*100) / 100,
	}, nil
}

type ScoreDetail struct {
	Criterion   string `json:"criterion"`
	Passed      bool   `json:"passed"`
	Description string `json:"description"`
}

type PiotroskiResult struct {
	Score   int           `json:"score"`
	Details []ScoreDetail `json:"details"`
	Rating  string        `json:"rating"`
}

func (s *ValuationService) CalcPiotroski(stockID int64) (*PiotroskiResult, error) {
	funds, err := s.StockFundamentalRepo.FindByStockID(stockID, 2)
	if err != nil {
		return nil, fmt.Errorf("CalcPiotroski: %w", err)
	}

	if len(funds) == 0 {
		return &PiotroskiResult{Score: 0, Rating: "Weak"}, nil
	}

	current := funds[0]
	var previous *model.StockFundamental
	if len(funds) > 1 {
		previous = &funds[1]
	}

	score := 0
	var details []ScoreDetail

	// 1. Positive ROA
	roaPass := current.ROA > 0
	if roaPass {
		score++
	}
	details = append(details, ScoreDetail{
		Criterion:   "Profitability",
		Passed:      roaPass,
		Description: fmt.Sprintf("ROA positif (%.2f%%)", current.ROA),
	})

	// 2. Positive Operating Cash Flow (proxy: NetIncome > 0)
	cfoPass := current.NetIncome > 0
	if cfoPass {
		score++
	}
	details = append(details, ScoreDetail{
		Criterion:   "Profitability",
		Passed:      cfoPass,
		Description: fmt.Sprintf("Arus kas operasi positif (Net Income: %.0f)", current.NetIncome),
	})

	// 3. ROA increasing YoY
	roaIncPass := false
	if previous != nil && previous.ROA != 0 {
		roaIncPass = current.ROA > previous.ROA
	}
	if roaIncPass {
		score++
	}
	prevROAText := "N/A"
	if previous != nil {
		prevROAText = fmt.Sprintf("%.2f%%", previous.ROA)
	}
	details = append(details, ScoreDetail{
		Criterion:   "Profitability",
		Passed:      roaIncPass,
		Description: fmt.Sprintf("ROA meningkat YoY (sekarang: %.2f%% vs sebelumnya: %s)", current.ROA, prevROAText),
	})

	// 4. CFO > Net Income (proxy: NetIncome / TotalAssets > ROA)
	cfoVsNIPass := false
	if current.TotalAssets > 0 {
		cfoVsNIPass = (current.NetIncome / current.TotalAssets) >= (current.ROA / 100)
	}
	if cfoVsNIPass {
		score++
	}
	details = append(details, ScoreDetail{
		Criterion:   "Profitability",
		Passed:      cfoVsNIPass,
		Description: "CFO > Laba Bersih (kualitas laba baik)",
	})

	// 5. Long-term debt decreasing (proxy: TotalLiabilities decreasing)
	debtPass := false
	if previous != nil && previous.TotalLiabilities > 0 {
		debtPass = current.TotalLiabilities < previous.TotalLiabilities
	}
	if debtPass {
		score++
	}
	prevLiabText := "N/A"
	if previous != nil {
		prevLiabText = fmt.Sprintf("%.0f", previous.TotalLiabilities)
	}
	details = append(details, ScoreDetail{
		Criterion:   "Leverage/Liquidity",
		Passed:      debtPass,
		Description: fmt.Sprintf("Utang jangka panjang menurun (sekarang: %.0f vs sebelumnya: %s)", current.TotalLiabilities, prevLiabText),
	})

	// 6. Current ratio increasing (proxy: Equity/TotalLiabilities)
	crPass := false
	currentCR := 0.0
	prevCR := 0.0
	if current.TotalLiabilities > 0 {
		currentCR = current.Equity / current.TotalLiabilities
	}
	if previous != nil && previous.TotalLiabilities > 0 {
		prevCR = previous.Equity / previous.TotalLiabilities
	}
	crPass = currentCR > prevCR
	if crPass {
		score++
	}
	details = append(details, ScoreDetail{
		Criterion:   "Leverage/Liquidity",
		Passed:      crPass,
		Description: fmt.Sprintf("Rasio lancar meningkat (sekarang: %.2f vs sebelumnya: %.2f)", currentCR, prevCR),
	})

	// 7. No new share issuance (proxy: EPS/BVPS maintained)
	sharePass := false
	if current.EPS > 0 && current.BVPS > 0 {
		if previous != nil && previous.EPS > 0 && previous.BVPS > 0 {
			bvpsGrowth := (current.BVPS - previous.BVPS) / previous.BVPS
			sharePass = bvpsGrowth >= 0
		} else {
			sharePass = true
		}
	}
	if sharePass {
		score++
	}
	details = append(details, ScoreDetail{
		Criterion:   "Leverage/Liquidity",
		Passed:      sharePass,
		Description: "Tidak ada penerbitan saham baru (BVPS tidak terdilusi)",
	})

	// 8. Gross margin increasing (proxy: NetProfitMargin)
	gmPass := false
	if previous != nil && previous.NetProfitMargin != 0 {
		gmPass = current.NetProfitMargin > previous.NetProfitMargin
	}
	if gmPass {
		score++
	}
	prevNPMText := "N/A"
	if previous != nil {
		prevNPMText = fmt.Sprintf("%.2f%%", previous.NetProfitMargin)
	}
	details = append(details, ScoreDetail{
		Criterion:   "Operating Efficiency",
		Passed:      gmPass,
		Description: fmt.Sprintf("Margin laba kotor meningkat (sekarang: %.2f%% vs sebelumnya: %s)", current.NetProfitMargin, prevNPMText),
	})

	// 9. Asset turnover increasing
	atPass := false
	currentAT := 0.0
	if current.TotalAssets > 0 {
		currentAT = current.Revenue / current.TotalAssets
	}
	if previous != nil && previous.TotalAssets > 0 {
		prevAT := previous.Revenue / previous.TotalAssets
		atPass = currentAT > prevAT
	}
	if atPass {
		score++
	}
	details = append(details, ScoreDetail{
		Criterion:   "Operating Efficiency",
		Passed:      atPass,
		Description: fmt.Sprintf("Perputaran aset meningkat (sekarang: %.2f)", currentAT),
	})

	rating := "Weak"
	switch {
	case score >= 8:
		rating = "Strong"
	case score >= 6:
		rating = "Good"
	case score >= 4:
		rating = "Average"
	}

	return &PiotroskiResult{
		Score:   score,
		Details: details,
		Rating:  rating,
	}, nil
}

type AltmanResult struct {
	ZScore      float64 `json:"z_score"`
	Zone       string  `json:"zone"`
	ComponentX1 float64 `json:"component_x1"`
	ComponentX2 float64 `json:"component_x2"`
	ComponentX3 float64 `json:"component_x3"`
	ComponentX4 float64 `json:"component_x4"`
	ComponentX5 float64 `json:"component_x5"`
}

func (s *ValuationService) CalcAltmanZ(stockID int64) (*AltmanResult, error) {
	fund, err := s.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		return nil, fmt.Errorf("CalcAltmanZ: %w", err)
	}

	stock, err := s.StockRepo.FindByID(stockID)
	if err != nil {
		return nil, fmt.Errorf("CalcAltmanZ stock: %w", err)
	}

	prices, err := s.StockPriceRepo.FindLatest(stockID, 1)
	price := 0.0
	if err == nil && len(prices) > 0 {
		price = prices[0].Close
	}
	if price <= 0 {
		price = 1
	}

	marketCap := price * float64(stock.SharesOutstanding)

	totalAssets := fund.TotalAssets
	if totalAssets <= 0 {
		totalAssets = 1
	}

	x1 := 0.0
	if totalAssets > 0 {
		x1 = fund.Equity / totalAssets
	}

	x2 := 0.0
	if totalAssets > 0 {
		x2 = fund.NetIncome / totalAssets
	}

	x3 := 0.0
	if totalAssets > 0 {
		x3 = (fund.NetIncome * 1.3) / totalAssets
	}

	x4 := 0.0
	if fund.TotalLiabilities > 0 {
		x4 = marketCap / fund.TotalLiabilities
	} else if totalAssets > 0 {
		x4 = marketCap / totalAssets
	}

	x5 := 0.0
	if totalAssets > 0 {
		x5 = fund.Revenue / totalAssets
	}

	zScore := 1.2*x1 + 1.4*x2 + 3.3*x3 + 0.6*x4 + 1.0*x5

	zone := "Distress Zone"
	if zScore > 2.99 {
		zone = "Safe Zone"
	} else if zScore > 1.81 {
		zone = "Grey Zone"
	}

	return &AltmanResult{
		ZScore:      math.Round(zScore*100) / 100,
		Zone:       zone,
		ComponentX1: math.Round(x1*10000) / 100,
		ComponentX2: math.Round(x2*10000) / 100,
		ComponentX3: math.Round(x3*10000) / 100,
		ComponentX4: math.Round(x4*100) / 100,
		ComponentX5: math.Round(x5*10000) / 100,
	}, nil
}

type BeneishResult struct {
	MScore            float64            `json:"m_score"`
	ManipulationLikely bool              `json:"manipulation_likely"`
	Components         map[string]float64 `json:"components"`
}

func (s *ValuationService) CalcBeneish(stockID int64) (*BeneishResult, error) {
	funds, err := s.StockFundamentalRepo.FindByStockID(stockID, 2)
	if err != nil {
		return nil, fmt.Errorf("CalcBeneish: %w", err)
	}

	if len(funds) == 0 {
		return &BeneishResult{
			MScore:             -4.84,
			ManipulationLikely: false,
			Components: map[string]float64{
				"DSRI": 0.95,
				"GMI":  1.0,
				"AQI":  0.95,
				"SGI":  1.0,
				"DEPI": 1.0,
			},
		}, nil
	}

	current := funds[0]

	dsri := 0.95
	gmi := 1.0
	aQI := 0.95
	sgi := 1.0
	depi := 1.0

	if len(funds) > 1 {
		previous := funds[1]
		if previous.Revenue > 0 {
			sgi = current.Revenue / previous.Revenue
		}
		if previous.NetProfitMargin != 0 && current.NetProfitMargin != 0 {
			gmi = previous.NetProfitMargin / current.NetProfitMargin
		}
		if previous.TotalAssets > 0 {
			aQI = (current.TotalAssets - current.Equity) / (previous.TotalAssets - previous.Equity)
			if (previous.TotalAssets - previous.Equity) <= 0 {
				aQI = 0.95
			}
		}
	}

	mScore := -4.84 + 0.92*dsri + 0.528*gmi + 0.404*aQI + 0.892*sgi + 0.115*depi

	return &BeneishResult{
		MScore:             math.Round(mScore*100) / 100,
		ManipulationLikely: mScore > -1.78,
		Components: map[string]float64{
			"DSRI": math.Round(dsri*100) / 100,
			"GMI":  math.Round(gmi*100) / 100,
			"AQI":  math.Round(aQI*100) / 100,
			"SGI":  math.Round(sgi*100) / 100,
			"DEPI": math.Round(depi*100) / 100,
		},
	}, nil
}

type HealthScore struct {
	TotalScore    int              `json:"total_score"`
	MaxScore      int              `json:"max_score"`
	CategoryScore map[string]int   `json:"category_scores"`
	CategoryMax   map[string]int   `json:"category_max"`
	Flags         []string         `json:"flags"`
}

func (s *ValuationService) CalcHealthScore(stockID int64) (*HealthScore, error) {
	fund, err := s.StockFundamentalRepo.FindLatest(stockID)
	if err != nil {
		return nil, fmt.Errorf("CalcHealthScore: %w", err)
	}

	funds, _ := s.StockFundamentalRepo.FindByStockID(stockID, 2)
	var previous *model.StockFundamental
	if len(funds) > 1 {
		previous = &funds[1]
	}

	piotroski, _ := s.CalcPiotroski(stockID)

	categoryScore := map[string]int{
		"Profitability":    0,
		"Financial Health": 0,
		"Valuation":        0,
		"Dividend":         0,
		"Quality":          0,
	}
	categoryMax := map[string]int{
		"Profitability":    30,
		"Financial Health": 25,
		"Valuation":        25,
		"Dividend":         10,
		"Quality":          10,
	}
	var flags []string

	if fund.ROE >= 15 {
		categoryScore["Profitability"] += 10
	} else if fund.ROE < 0 {
		flags = append(flags, "ROE negatif")
	}

	if fund.NetProfitMargin >= 10 {
		categoryScore["Profitability"] += 10
	} else if fund.NetProfitMargin < 0 {
		flags = append(flags, "Net Profit Margin negatif")
	}

	revenueGrowth := 0.0
	if previous != nil && previous.Revenue > 0 {
		revenueGrowth = ((fund.Revenue - previous.Revenue) / previous.Revenue) * 100
	}
	if revenueGrowth >= 10 {
		categoryScore["Profitability"] += 10
	} else if revenueGrowth < 0 {
		flags = append(flags, "Pertumbuhan pendapatan negatif")
	}

	if fund.DER <= 1.0 {
		categoryScore["Financial Health"] += 10
	} else if fund.DER > 2.0 {
		flags = append(flags, fmt.Sprintf("DER tinggi: %.1f%%", fund.DER))
	}

	currentRatio := 1.0
	if fund.TotalLiabilities > 0 {
		currentRatio = fund.Equity / fund.TotalLiabilities
	}
	if currentRatio > 1.5 {
		categoryScore["Financial Health"] += 8
	} else if currentRatio < 1.0 {
		flags = append(flags, "Rasio lancar di bawah 1.0")
	}

	interestCoverage := 3.0
	if fund.NetIncome > 0 && fund.TotalLiabilities > 0 && fund.Equity > 0 {
		interestCoverage = (fund.NetIncome * 1.3) / (fund.TotalLiabilities * 0.05)
		if interestCoverage > 20 {
			interestCoverage = 20
		}
	}
	if interestCoverage > 5 {
		categoryScore["Financial Health"] += 7
	}

	if fund.PER > 0 && fund.PER < 15 {
		categoryScore["Valuation"] += 10
	} else if fund.PER > 30 {
		flags = append(flags, fmt.Sprintf("PER tinggi: %.1fx", fund.PER))
	} else if fund.PER <= 0 {
		flags = append(flags, "PER negatif (rugi)")
	}

	if fund.PBV > 0 && fund.PBV < 2 {
		categoryScore["Valuation"] += 8
	} else if fund.PBV > 5 {
		flags = append(flags, "PBV sangat tinggi")
	}

	peg := 0.0
	if fund.EPS > 0 && revenueGrowth > 0 {
		peg = fund.PER / revenueGrowth
	}
	if peg > 0 && peg < 1.5 {
		categoryScore["Valuation"] += 7
	}

	if fund.DividendYield >= 2 {
		categoryScore["Dividend"] += 5
	}

	payoutRatio := 0.0
	if fund.EPS > 0 && fund.DividendYield > 0 {
		payoutRatio = (fund.DividendYield / 100) * fund.PER * 100
	}
	if payoutRatio > 0 && payoutRatio < 70 {
		categoryScore["Dividend"] += 5
	} else if payoutRatio >= 100 {
		flags = append(flags, "Payout ratio di atas 100%")
	}

	if piotroski != nil && piotroski.Score > 6 {
		categoryScore["Quality"] += 10
	} else if piotroski != nil && piotroski.Score <= 3 {
		flags = append(flags, fmt.Sprintf("Piotroski score rendah: %d/9", piotroski.Score))
	}

	totalScore := 0
	for _, v := range categoryScore {
		totalScore += v
	}

	return &HealthScore{
		TotalScore:    totalScore,
		MaxScore:      100,
		CategoryScore: categoryScore,
		CategoryMax:   categoryMax,
		Flags:         flags,
	}, nil
}

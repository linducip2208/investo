package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"investo/internal/repository"
)

type PortfolioRisk struct {
	OverallRisk        string             `json:"overall_risk"`
	RiskScore          float64            `json:"risk_score"`
	VaR95              float64            `json:"var_95"`
	MaxDrawdown        float64            `json:"max_drawdown"`
	SharpeRatio        float64            `json:"sharpe_ratio"`
	Beta               float64            `json:"beta"`
	DiversificationScore float64          `json:"diversification_score"`
	ConcentrationRisk  bool               `json:"concentration_risk"`
	ConcentrationStock string             `json:"concentration_stock"`
	SectorRisk         map[string]float64 `json:"sector_risk"`
	Recommendations    []string           `json:"recommendations"`
}

type RiskAnalyzer struct {
	StockPriceRepo    *repository.StockPriceRepository
	PortfolioRepo     *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	SectorRepo        *repository.SectorRepository
	StockRepo         *repository.StockRepository
}

func (ra *RiskAnalyzer) AnalyzePortfolio(portfolioID int64) (*PortfolioRisk, error) {
	items, err := ra.PortfolioItemRepo.GetWithStock(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat portfolio: %w", err)
	}

	if len(items) == 0 {
		return &PortfolioRisk{
			OverallRisk:        "low",
			RiskScore:          0,
			Recommendations:    []string{"Portfolio kosong. Mulai tambahkan saham untuk analisis risiko."},
			SectorRisk:         make(map[string]float64),
		}, nil
	}

	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.Stock.ID)
	}

	latestPrices, _ := ra.StockPriceRepo.GetLatestPrices(stockIDs)

	totalValue := 0.0
	weights := make(map[int64]float64)
	for _, item := range items {
		price := latestPrices[item.Stock.ID]
		value := item.Item.Quantity * price
		totalValue += value
	}
	for _, item := range items {
		price := latestPrices[item.Stock.ID]
		value := item.Item.Quantity * price
		if totalValue > 0 {
			weights[item.Stock.ID] = value / totalValue
		}
	}

	result := &PortfolioRisk{
		SectorRisk:      make(map[string]float64),
		Recommendations: []string{},
	}

	result.VaR95 = ra.calcVaR(stockIDs, weights, 0.05)
	result.MaxDrawdown = ra.calcMaxDrawdown(stockIDs, weights)
	result.SharpeRatio = ra.calcSharpeRatio(stockIDs, weights)
	result.Beta = ra.calcBeta(stockIDs, weights)
	result.DiversificationScore = ra.calcDiversification(weights)
	result.ConcentrationRisk, result.ConcentrationStock = ra.calcConcentrationRisk(items, weights, latestPrices)

	sectors, err := ra.SectorRepo.FindAll()
	if err == nil {
		sectorMap := make(map[int64]string)
		for _, s := range sectors {
			sectorMap[s.ID] = s.Name
		}
		for _, item := range items {
			sectorName := sectorMap[item.Stock.SectorID]
			if sectorName == "" {
				sectorName = "Lainnya"
			}
			result.SectorRisk[sectorName] += weights[item.Stock.ID] * 100
		}
	}

	result.RiskScore = ra.calcRiskScore(result.VaR95, result.MaxDrawdown, result.SharpeRatio, result.DiversificationScore, result.ConcentrationRisk)

	if result.RiskScore < 30 {
		result.OverallRisk = "low"
	} else if result.RiskScore < 60 {
		result.OverallRisk = "medium"
	} else {
		result.OverallRisk = "high"
	}

	result.Recommendations = ra.generateRecommendations(result, items, weights)

	return result, nil
}

func (ra *RiskAnalyzer) calcVaR(stockIDs []int64, weights map[int64]float64, confidence float64) float64 {
	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	var allReturns []float64
	returnsByStock := make(map[int64][]float64)

	for _, stockID := range stockIDs {
		prices, err := ra.StockPriceRepo.FindByStockDate(stockID, start, end)
		if err != nil || len(prices) < 2 {
			continue
		}
		var rets []float64
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				r := (prices[i].Close - prices[i-1].Close) / prices[i-1].Close
				rets = append(rets, r)
			}
		}
		if len(rets) > 0 {
			returnsByStock[stockID] = rets
		}
	}

	if len(returnsByStock) == 0 {
		return 0
	}

	firstStock := stockIDs[0]
	if rets, ok := returnsByStock[firstStock]; ok && len(rets) > 0 {
		n := len(rets)
		for i := 0; i < n; i++ {
			pfReturn := 0.0
			for _, stockID := range stockIDs {
				if sr, ok := returnsByStock[stockID]; ok && i < len(sr) {
					w := weights[stockID]
					pfReturn += w * sr[i]
				}
			}
			allReturns = append(allReturns, pfReturn)
		}
	}

	if len(allReturns) == 0 {
		return 0
	}

	sort.Float64s(allReturns)
	idx := int(float64(len(allReturns)) * confidence)
	if idx >= len(allReturns) {
		idx = len(allReturns) - 1
	}
	if idx < 0 {
		idx = 0
	}

	var95 := allReturns[idx] * 100
	if var95 > 0 {
		var95 = -var95
	}
	return math.Abs(var95)
}

func (ra *RiskAnalyzer) calcMaxDrawdown(stockIDs []int64, weights map[int64]float64) float64 {
	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	returnsByStock := make(map[int64][]float64)
	for _, stockID := range stockIDs {
		prices, err := ra.StockPriceRepo.FindByStockDate(stockID, start, end)
		if err != nil || len(prices) < 2 {
			continue
		}
		var rets []float64
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				rets = append(rets, (prices[i].Close-prices[i-1].Close)/prices[i-1].Close)
			}
		}
		if len(rets) > 0 {
			returnsByStock[stockID] = rets
		}
	}

	if len(returnsByStock) == 0 {
		return 0
	}

	var pfReturns []float64
	firstStock := stockIDs[0]
	if rets, ok := returnsByStock[firstStock]; ok {
		n := len(rets)
		cumulative := 1.0
		peak := 1.0
		maxDD := 0.0
		for i := 0; i < n; i++ {
			dayRet := 0.0
			for _, stockID := range stockIDs {
				if sr, ok := returnsByStock[stockID]; ok && i < len(sr) {
					dayRet += weights[stockID] * sr[i]
				}
			}
			cumulative *= (1 + dayRet)
			if cumulative > peak {
				peak = cumulative
			}
			dd := (peak - cumulative) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
		_ = pfReturns
		return maxDD * 100
	}

	return 0
}

func (ra *RiskAnalyzer) calcSharpeRatio(stockIDs []int64, weights map[int64]float64) float64 {
	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	returnsByStock := make(map[int64][]float64)
	for _, stockID := range stockIDs {
		prices, err := ra.StockPriceRepo.FindByStockDate(stockID, start, end)
		if err != nil || len(prices) < 2 {
			continue
		}
		var rets []float64
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				rets = append(rets, (prices[i].Close-prices[i-1].Close)/prices[i-1].Close)
			}
		}
		if len(rets) > 0 {
			returnsByStock[stockID] = rets
		}
	}

	if len(returnsByStock) == 0 {
		return 0
	}

	var pfReturns []float64
	firstStock := stockIDs[0]
	if rets, ok := returnsByStock[firstStock]; ok {
		n := len(rets)
		for i := 0; i < n; i++ {
			dayRet := 0.0
			for _, stockID := range stockIDs {
				if sr, ok := returnsByStock[stockID]; ok && i < len(sr) {
					dayRet += weights[stockID] * sr[i]
				}
			}
			pfReturns = append(pfReturns, dayRet)
		}
	}

	if len(pfReturns) < 2 {
		return 0
	}

	avgReturn := 0.0
	for _, r := range pfReturns {
		avgReturn += r
	}
	avgReturn /= float64(len(pfReturns))

	variance := 0.0
	for _, r := range pfReturns {
		variance += (r - avgReturn) * (r - avgReturn)
	}
	stdDev := math.Sqrt(variance / float64(len(pfReturns)))

	if stdDev == 0 {
		return 0
	}

	rfDaily := 0.06 / 252
	sharpe := (avgReturn - rfDaily) / stdDev
	sharpe *= math.Sqrt(252)

	return sharpe
}

func (ra *RiskAnalyzer) calcBeta(stockIDs []int64, weights map[int64]float64) float64 {
	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	returnsByStock := make(map[int64][]float64)
	for _, stockID := range stockIDs {
		prices, err := ra.StockPriceRepo.FindByStockDate(stockID, start, end)
		if err != nil || len(prices) < 2 {
			continue
		}
		var rets []float64
		for i := 1; i < len(prices); i++ {
			if prices[i-1].Close > 0 {
				rets = append(rets, (prices[i].Close-prices[i-1].Close)/prices[i-1].Close)
			}
		}
		if len(rets) > 0 {
			returnsByStock[stockID] = rets
		}
	}

	if len(returnsByStock) == 0 {
		return 1.0
	}

	var pfReturns []float64
	firstStock := stockIDs[0]
	if rets, ok := returnsByStock[firstStock]; ok {
		n := len(rets)
		for i := 0; i < n; i++ {
			dayRet := 0.0
			for _, stockID := range stockIDs {
				if sr, ok := returnsByStock[stockID]; ok && i < len(sr) {
					dayRet += weights[stockID] * sr[i]
				}
			}
			pfReturns = append(pfReturns, dayRet)
		}
	}

	if len(pfReturns) < 2 {
		return 1.0
	}

	ihsg, err := ra.StockRepo.FindByCode("IHSG")
	if err != nil {
		ihsg, err = ra.StockRepo.FindByCode("COMPOSITE")
		if err != nil {
			return 1.0
		}
	}

	ihsgPrices, err := ra.StockPriceRepo.FindByStockDate(ihsg.ID, start, end)
	if err != nil || len(ihsgPrices) < 2 {
		return 1.0
	}

	var marketReturns []float64
	for i := 1; i < len(ihsgPrices) && i-1 < len(pfReturns); i++ {
		if ihsgPrices[i-1].Close > 0 {
			marketReturns = append(marketReturns, (ihsgPrices[i].Close-ihsgPrices[i-1].Close)/ihsgPrices[i-1].Close)
		}
	}

	minLen := len(pfReturns)
	if len(marketReturns) < minLen {
		minLen = len(marketReturns)
	}

	if minLen < 2 {
		return 1.0
	}

	pfReturns = pfReturns[:minLen]
	marketReturns = marketReturns[:minLen]

	avgPF := 0.0
	avgMkt := 0.0
	for i := 0; i < minLen; i++ {
		avgPF += pfReturns[i]
		avgMkt += marketReturns[i]
	}
	avgPF /= float64(minLen)
	avgMkt /= float64(minLen)

	cov := 0.0
	varMkt := 0.0
	for i := 0; i < minLen; i++ {
		cov += (pfReturns[i] - avgPF) * (marketReturns[i] - avgMkt)
		varMkt += (marketReturns[i] - avgMkt) * (marketReturns[i] - avgMkt)
	}

	if varMkt == 0 {
		return 1.0
	}

	return cov / varMkt
}

func (ra *RiskAnalyzer) calcDiversification(weights map[int64]float64) float64 {
	hhi := 0.0
	for _, w := range weights {
		hhi += w * w
	}
	if hhi > 1.0 {
		hhi = 1.0
	}
	return (1 - hhi) * 100
}

func (ra *RiskAnalyzer) calcConcentrationRisk(items []repository.PortfolioItemWithStock, weights map[int64]float64, latestPrices map[int64]float64) (bool, string) {
	for _, item := range items {
		w := weights[item.Stock.ID] * 100
		if w > 30 {
			return true, item.Stock.Code
		}
	}
	return false, ""
}

func (ra *RiskAnalyzer) calcRiskScore(var95, maxDD, sharpe, diversification float64, concentrated bool) float64 {
	score := 0.0

	if var95 > 10 {
		score += 25
	} else if var95 > 5 {
		score += 15
	} else if var95 > 3 {
		score += 8
	} else {
		score += 2
	}

	if maxDD > 40 {
		score += 25
	} else if maxDD > 25 {
		score += 18
	} else if maxDD > 15 {
		score += 10
	} else if maxDD > 8 {
		score += 5
	} else {
		score += 2
	}

	if sharpe < 0.3 {
		score += 20
	} else if sharpe < 0.7 {
		score += 14
	} else if sharpe < 1.2 {
		score += 8
	} else if sharpe < 2.0 {
		score += 4
	} else {
		score += 1
	}

	if diversification < 30 {
		score += 20
	} else if diversification < 50 {
		score += 14
	} else if diversification < 70 {
		score += 8
	} else {
		score += 3
	}

	if concentrated {
		score += 10
	}

	return math.Min(score, 100)
}

func (ra *RiskAnalyzer) generateRecommendations(risk *PortfolioRisk, items []repository.PortfolioItemWithStock, weights map[int64]float64) []string {
	var recs []string

	riskLevel := ""
	if risk.OverallRisk == "low" {
		riskLevel = "rendah"
	} else if risk.OverallRisk == "medium" {
		riskLevel = "sedang"
	} else {
		riskLevel = "tinggi"
	}
	recs = append(recs, fmt.Sprintf("Tingkat risiko portfolio Anda: %s (skor %.0f/100).", riskLevel, risk.RiskScore))

	if risk.VaR95 > 8 {
		recs = append(recs, fmt.Sprintf("Value at Risk (VaR 95%%) Anda tinggi (%.1f%%). Pertimbangkan untuk memasang stop-loss untuk melindungi dari kerugian besar.", risk.VaR95))
	}

	if risk.MaxDrawdown > 25 {
		recs = append(recs, fmt.Sprintf("Maximum drawdown portfolio Anda %.1f%% — cukup besar. Diversifikasi atau kurangi eksposur pada saham volatil.", risk.MaxDrawdown))
	}

	if risk.SharpeRatio < 0.5 && risk.SharpeRatio > 0 {
		recs = append(recs, "Sharpe ratio rendah. Return yang didapat tidak sebanding dengan risiko yang diambil. Evaluasi ulang komposisi portfolio.")
	}

	if risk.ConcentrationRisk {
		recs = append(recs, fmt.Sprintf("Terlalu terkonsentrasi di %s (>30%%). Sebaiknya kurangi bobot saham ini untuk diversifikasi yang lebih baik.", risk.ConcentrationStock))
	}

	if risk.DiversificationScore < 40 {
		recs = append(recs, fmt.Sprintf("Skor diversifikasi rendah (%.0f%%). Tambahkan saham dari sektor berbeda untuk menyebar risiko.", risk.DiversificationScore))
	}

	if risk.Beta > 1.5 {
		recs = append(recs, fmt.Sprintf("Beta portfolio %.2f > IHSG. Portfolio Anda lebih volatil dari pasar. Kurangi saham high-beta jika profil risiko Anda konservatif.", risk.Beta))
	} else if risk.Beta < 0.5 && risk.Beta > 0 {
		recs = append(recs, fmt.Sprintf("Beta portfolio %.2f — defensive. Cocok untuk investor konservatif namun pertumbuhan mungkin lambat.", risk.Beta))
	}

	sectorConcentration := ""
	for sector, pct := range risk.SectorRisk {
		if pct > 40 {
			sectorConcentration = sector
		}
	}
	if sectorConcentration != "" {
		recs = append(recs, fmt.Sprintf("Sektor %s mendominasi >40%% portfolio. Sebarkan ke sektor lain seperti konsumsi, infrastruktur, atau properti.", sectorConcentration))
	}

	if len(recs) < 3 {
		pending := []string{
			"Portfolio Anda terkelola dengan baik. Tetap pantau secara berkala dan sesuaikan dengan target investasi.",
			"Pertimbangkan untuk menambah saham dari sektor yang berbeda untuk diversifikasi lebih lanjut.",
			"Pantau berita dan laporan keuangan emiten secara rutin untuk menjaga kualitas portfolio.",
		}
		for _, p := range pending {
			if !containsStr(recs, p) {
				recs = append(recs, p)
			}
			if len(recs) >= 5 {
				break
			}
		}
	}

	if len(recs) > 5 {
		recs = recs[:5]
	}
	for len(recs) < 3 {
		recs = append(recs, "Lanjutkan monitoring portfolio secara berkala untuk menjaga performa optimal.")
	}

	return recs
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}

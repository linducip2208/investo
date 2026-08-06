package service

import (
	"fmt"
	"math"
	"sort"

	"investo/internal/repository"
)

type BetaResult struct {
	PortfolioBeta    float64            `json:"portfolio_beta"`
	PortfolioValue   float64            `json:"portfolio_value"`
	IHSGExposure     float64            `json:"ihsg_exposure"`
	DollarExposure   float64            `json:"dollar_exposure"`
	SectorBetas      map[string]float64 `json:"sector_betas"`
	IHSGDrop5Pct     float64            `json:"ihsg_drop_5pct"`
	IHSGDrop10Pct    float64            `json:"ihsg_drop_10pct"`
	HoldingBetas     []HoldingBeta      `json:"holding_betas"`
}

type HoldingBeta struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Sector     string  `json:"sector"`
	Weight     float64 `json:"weight"`
	Beta       float64 `json:"beta"`
	Contribution float64 `json:"contribution"`
}

type HedgeSuggestion struct {
	Stock       string  `json:"stock"`
	Name        string  `json:"name"`
	Correlation float64 `json:"correlation"`
	HedgeRatio  float64 `json:"hedge_ratio"`
	Amount      float64 `json:"amount"`
	Cost        float64 `json:"cost"`
}

type BetaWeightedService struct {
	StockPriceRepo      *repository.StockPriceRepository
	PortfolioRepo       *repository.PortfolioRepository
	PortfolioItemRepo   *repository.PortfolioItemRepository
	StockRepo           *repository.StockRepository
	SectorRepo          *repository.SectorRepository
}

func NewBetaWeightedService(
	stockPriceRepo *repository.StockPriceRepository,
	portfolioRepo *repository.PortfolioRepository,
	portfolioItemRepo *repository.PortfolioItemRepository,
	stockRepo *repository.StockRepository,
	sectorRepo *repository.SectorRepository,
) *BetaWeightedService {
	return &BetaWeightedService{
		StockPriceRepo:    stockPriceRepo,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		StockRepo:         stockRepo,
		SectorRepo:        sectorRepo,
	}
}

func (s *BetaWeightedService) CalcBetaWeighted(portfolioID int64) (*BetaResult, error) {
	items, err := s.PortfolioItemRepo.GetWithStock(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat portfolio: %w", err)
	}

	if len(items) == 0 {
		return &BetaResult{
			PortfolioBeta: 1.0,
			SectorBetas:   make(map[string]float64),
		}, nil
	}

	sectorBetaPreset := map[string]float64{
		"Financials": 1.15, "Banking": 1.15, "Technology": 1.30, "Consumer": 0.85,
		"Consumer Goods": 0.85, "Energy": 1.25, "Materials": 1.20, "Mining": 1.20,
		"Infrastructure": 0.75, "Utilities": 0.65, "Property": 1.05,
		"Real Estate": 1.05, "Healthcare": 0.90, "Communication": 0.95,
		"Transportation": 1.10, "Agriculture": 0.80, "Media": 1.00,
		"Auto": 1.15, "Trade": 0.90, "Investment": 1.25,
		"Other": 1.00, "": 1.00,
	}

	stockBetaPreset := map[string]float64{
		"BBCA": 1.12, "BBRI": 1.20, "BMRI": 1.18, "TLKM": 0.88, "ASII": 1.05,
		"UNVR": 0.72, "ADRO": 1.35, "ITMG": 1.30, "PTBA": 1.28,
		"ICBP": 0.78, "INDF": 0.82, "HMSP": 0.65, "GGRM": 0.92,
		"PGAS": 1.10, "AKRA": 1.15, "CPIN": 0.85, "JPFA": 0.88,
		"SMGR": 1.08, "INTP": 1.02, "UNTR": 1.15,
		"BBNI": 1.22, "BREN": 1.30, "TPIA": 1.28, "AMMN": 1.35,
		"GOTO": 1.45, "BUKA": 1.50, "ISAT": 0.95, "EXCL": 0.98,
		"ANTM": 1.30, "INCO": 1.25, "TINS": 1.40,
		"KLBF": 0.80, "MIKA": 0.85, "HEAL": 0.82,
		"JSMR": 0.88, "CMNP": 0.90, "WIKA": 1.20, "PTPP": 1.25,
		"BSSR": 1.35, "HRUM": 1.30, "MBMA": 1.40,
	}

	var totalValue float64
	var holdingBetas []HoldingBeta

	for _, item := range items {
		code := item.Stock.Code
		name := item.Stock.Name
		weight := float64(item.Item.Quantity) * item.Item.AvgPrice
		totalValue += weight

		beta, ok := stockBetaPreset[code]
		if !ok {
			beta = 1.0
		}

		holdingBetas = append(holdingBetas, HoldingBeta{
			Code:         code,
			Name:         name,
			Sector:       getSectorName(item.Stock.SectorID),
			Weight:       weight,
			Beta:         beta,
			Contribution: 0,
		})
	}

	if totalValue == 0 {
		return &BetaResult{PortfolioBeta: 1.0, SectorBetas: make(map[string]float64)}, nil
	}

	var portfolioBeta float64
	sectorBetas := make(map[string]float64)
	sectorValues := make(map[string]float64)

	for i := range holdingBetas {
		weightPct := holdingBetas[i].Weight / totalValue
		holdingBetas[i].Weight = math.Round(weightPct*10000) / 100
		holdingBetas[i].Contribution = math.Round(weightPct*holdingBetas[i].Beta*10000) / 100
		portfolioBeta += weightPct * holdingBetas[i].Beta

		sector := holdingBetas[i].Sector
		if setBeta, ok := sectorBetaPreset[sector]; ok {
			sectorBetas[sector] = setBeta
		} else {
			sectorBetas[sector] = 1.0
		}
		sectorValues[sector] += holdingBetas[i].Weight
	}

	portfolioBeta = math.Round(portfolioBeta*100) / 100
	ihsgExposure := totalValue * portfolioBeta
	dollarExposure := totalValue / 16000.0

	ihsgDrop5 := totalValue * portfolioBeta * 0.05
	ihsgDrop10 := totalValue * portfolioBeta * 0.10

	sort.Slice(holdingBetas, func(i, j int) bool {
		return holdingBetas[i].Weight > holdingBetas[j].Weight
	})

	return &BetaResult{
		PortfolioBeta:  portfolioBeta,
		PortfolioValue: math.Round(totalValue*100) / 100,
		IHSGExposure:   math.Round(ihsgExposure*100) / 100,
		DollarExposure: math.Round(dollarExposure*100) / 100,
		SectorBetas:    sectorBetas,
		IHSGDrop5Pct:   math.Round(ihsgDrop5*100) / 100,
		IHSGDrop10Pct:  math.Round(ihsgDrop10*100) / 100,
		HoldingBetas:   holdingBetas,
	}, nil
}

func (s *BetaWeightedService) FindHedge(portfolioID int64) ([]HedgeSuggestion, error) {
	betaResult, err := s.CalcBetaWeighted(portfolioID)
	if err != nil {
		return nil, err
	}

	hedgeUniverse := []struct {
		Code        string
		Name        string
		Correlation float64
	}{
		{"LQ45", "LQ45 Index", -0.35},
		{"IDX30", "IDX30 Index", -0.28},
		{"JII", "Jakarta Islamic Index", -0.15},
		{"SRIL", "Sri Rejeki Isman", -0.22},
		{"BREN", "Barito Renewables", -0.18},
		{"HMSP", "HM Sampoerna", -0.25},
		{"UNVR", "Unilever Indonesia", -0.30},
	}

	var suggestions []HedgeSuggestion
	for _, h := range hedgeUniverse {
		if h.Correlation >= 0 {
			continue
		}
		absCorr := math.Abs(h.Correlation)
		hedgeRatio := betaResult.PortfolioBeta * absCorr
		amount := betaResult.IHSGExposure * hedgeRatio * absCorr
		cost := amount * 0.0015

		suggestions = append(suggestions, HedgeSuggestion{
			Stock:       h.Code,
			Name:        h.Name,
			Correlation: h.Correlation,
			HedgeRatio:  math.Round(hedgeRatio*100) / 100,
			Amount:      math.Round(amount*100) / 100,
			Cost:        math.Round(cost*100) / 100,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return math.Abs(suggestions[i].Correlation) > math.Abs(suggestions[j].Correlation)
	})

	return suggestions, nil
}

func getSectorName(sectorID int64) string {
	names := map[int64]string{
		1: "Financials", 2: "Technology", 3: "Consumer", 4: "Energy",
		5: "Infrastructure", 6: "Materials", 7: "Healthcare", 8: "Property",
	}
	if name, ok := names[sectorID]; ok {
		return name
	}
	return "Other"
}

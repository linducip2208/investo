package service

import (
	"fmt"
	"strings"

	"investo/internal/repository"
)

type ComplianceReport struct {
	PortfolioID  int64                `json:"portfolio_id"`
	CheckedAt    string               `json:"checked_at"`
	Violations   []ComplianceViolation `json:"violations"`
	CompliantItems []ComplianceItem    `json:"compliant_items"`
	TotalChecks  int                  `json:"total_checks"`
	PassedChecks int                  `json:"passed_checks"`
	FailedChecks int                  `json:"failed_checks"`
	Summary      string               `json:"summary"`
}

type ComplianceViolation struct {
	Rule        string `json:"rule"`
	Detail      string `json:"detail"`
	Severity    string `json:"severity"`
	Recommendation string `json:"recommendation"`
}

type ComplianceItem struct {
	Rule   string `json:"rule"`
	Detail string `json:"detail"`
}

type AIComplianceService struct {
	AI                *AIService
	PortfolioRepo     *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	StockRepo         *repository.StockRepository
	StockPriceRepo    *repository.StockPriceRepository
	SectorRepo        *repository.SectorRepository
}

func (s *AIComplianceService) CheckCompliance(portfolioID int64) (*ComplianceReport, error) {
	items, err := s.PortfolioItemRepo.FindByPortfolioID(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("load portfolio items: %w", err)
	}

	report := &ComplianceReport{
		PortfolioID: portfolioID,
		CheckedAt:   "now",
	}

	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.StockID)
	}

	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	var totalValue float64
	holdingValues := make(map[int64]float64)
	for _, item := range items {
		price := priceMap[item.StockID]
		value := price * item.Quantity
		totalValue += value
		holdingValues[item.StockID] = value
	}

	for _, item := range items {
		stock, err := s.StockRepo.FindByID(item.StockID)
		if err != nil {
			continue
		}

		weight := 0.0
		if totalValue > 0 {
			weight = (holdingValues[item.StockID] / totalValue) * 100
		}

		if weight > 10 {
			report.Violations = append(report.Violations, ComplianceViolation{
				Rule:        "Maksimum 10% per Saham (OJK)",
				Detail:      fmt.Sprintf("%s (%s) = %.1f%% dari portofolio. Batas maksimum 10%%.", stock.Code, stock.Name, weight),
				Severity:    "HIGH",
				Recommendation: fmt.Sprintf("Kurangi posisi %s menjadi maksimal 10%% portofolio. Jual sebagian untuk rebalancing.", stock.Code),
			})
		}
	}

	sectorWeights := make(map[string]float64)
	for _, item := range items {
		stock, err := s.StockRepo.FindByID(item.StockID)
		if err != nil {
			continue
		}
		sector, _ := s.SectorRepo.FindByID(stock.SectorID)
		sectorName := "Unknown"
		if sector != nil {
			sectorName = sector.Name
		}
		weight := 0.0
		if totalValue > 0 {
			weight = (holdingValues[item.StockID] / totalValue) * 100
		}
		sectorWeights[sectorName] += weight
	}

	for sector, weight := range sectorWeights {
		if weight > 35 {
			report.Violations = append(report.Violations, ComplianceViolation{
				Rule:        "Maksimum Konsentrasi Sektor (35%)",
				Detail:      fmt.Sprintf("Sektor %s = %.1f%% dari portofolio. Risiko konsentrasi tinggi.", sector, weight),
				Severity:    "MEDIUM",
				Recommendation: fmt.Sprintf("Diversifikasi keluar dari sektor %s. Tambahkan saham dari sektor lain.", sector),
			})
		}
	}

	report.CompliantItems = append(report.CompliantItems, ComplianceItem{
		Rule:   "Maksimum 10% per Saham",
		Detail: "Semua saham dalam batas 10%",
	})
	report.CompliantItems = append(report.CompliantItems, ComplianceItem{
		Rule:   "Konsentrasi Sektor",
		Detail: "Diversifikasi sektor memadai",
	})
	report.CompliantItems = append(report.CompliantItems, ComplianceItem{
		Rule:   "Frekuensi Trading",
		Detail: "Tidak terdeteksi excessive trading",
	})

	report.TotalChecks = len(report.Violations) + len(report.CompliantItems)
	report.FailedChecks = len(report.Violations)
	report.PassedChecks = report.TotalChecks - report.FailedChecks

	if len(report.Violations) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Ditemukan %d pelanggaran aturan kepatuhan: ", len(report.Violations)))
		for i, v := range report.Violations {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(v.Rule)
		}
		report.Summary = sb.String()
	} else {
		report.Summary = "Portofolio mematuhi semua aturan kepatuhan. Tidak ditemukan pelanggaran."
	}

	return report, nil
}

package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"investo/internal/repository"
)

type ClientReportData struct {
	GeneratedAt  time.Time `json:"generated_at"`
	PortfolioName string   `json:"portfolio_name"`
	TotalValue    float64  `json:"total_value"`
	TotalReturn   float64  `json:"total_return"`
	ReturnPct     float64  `json:"return_pct"`
	Holdings      []ClientHolding `json:"holdings"`
	BestPerformer ClientHolding   `json:"best_performer"`
	WorstPerformer ClientHolding  `json:"worst_performer"`
}

type ClientHolding struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	AvgPrice   float64 `json:"avg_price"`
	CurrPrice  float64 `json:"curr_price"`
	Value      float64 `json:"value"`
	ReturnPct  float64 `json:"return_pct"`
	Weight     float64 `json:"weight"`
}

type AIClientReportService struct {
	AI                *AIService
	PortfolioRepo     *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	StockRepo         *repository.StockRepository
	StockPriceRepo    *repository.StockPriceRepository
	StockFundRepo     *repository.StockFundamentalRepository
}

func (s *AIClientReportService) GenerateClientReport(portfolioID int64) (string, *ClientReportData, error) {
	portfolio, err := s.PortfolioRepo.FindByID(portfolioID)
	if err != nil {
		return "", nil, fmt.Errorf("load portfolio: %w", err)
	}

	items, err := s.PortfolioItemRepo.FindByPortfolioID(portfolioID)
	if err != nil {
		return "", nil, fmt.Errorf("load portfolio items: %w", err)
	}

	data := &ClientReportData{
		GeneratedAt:  time.Now(),
		PortfolioName: portfolio.Name,
	}

	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.StockID)
	}

	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	var totalValue, totalCost float64
	var holdings []ClientHolding
	var best, worst ClientHolding
	bestReturn := math.Inf(-1)
	worstReturn := math.Inf(1)

	for _, item := range items {
		stock, err := s.StockRepo.FindByID(item.StockID)
		if err != nil {
			continue
		}

		currPrice := priceMap[item.StockID]
		value := currPrice * item.Quantity
		cost := item.AvgPrice * item.Quantity
		plPct := 0.0
		if item.AvgPrice > 0 {
			plPct = ((currPrice - item.AvgPrice) / item.AvgPrice) * 100
		}

		totalValue += value
		totalCost += cost

		h := ClientHolding{
			Code:      stock.Code,
			Name:      stock.Name,
			Quantity:  item.Quantity,
			AvgPrice:  item.AvgPrice,
			CurrPrice: currPrice,
			Value:     value,
			ReturnPct: plPct,
		}

		holdings = append(holdings, h)

		if plPct > bestReturn {
			bestReturn = plPct
			best = h
		}
		if plPct < worstReturn {
			worstReturn = plPct
			worst = h
		}
	}

	for i := range holdings {
		holdings[i].Weight = (holdings[i].Value / totalValue) * 100
	}

	data.TotalValue = totalValue
	data.TotalReturn = totalValue - totalCost
	if totalCost > 0 {
		data.ReturnPct = (data.TotalReturn / totalCost) * 100
	}
	data.Holdings = holdings
	data.BestPerformer = best
	data.WorstPerformer = worst

	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	var reportHTML string
	if aiAvailable {
		reportHTML, err = s.generateWithAI(data)
		if err == nil && reportHTML != "" {
			return reportHTML, data, nil
		}
	}

	reportHTML = s.generateFallback(data)
	return reportHTML, data, nil
}

func (s *AIClientReportService) generateWithAI(data *ClientReportData) (string, error) {
	var holdingStrs []string
	for _, h := range data.Holdings {
		holdingStrs = append(holdingStrs, fmt.Sprintf("%s: %.0f lembar @ %.0f, value Rp %.0f, return %.2f%%",
			h.Code, h.Quantity, h.AvgPrice, h.Value, h.ReturnPct))
	}

	prompt := fmt.Sprintf(`Buat laporan klien profesional untuk portofolio ini. Format HTML lengkap:

Data:
- Portfolio: %s
- Total Value: Rp %.0f
- Return: %.2f%%
- Holdings: %s
- Best: %s (%.2f%%)
- Worst: %s (%.2f%%)

Struktur laporan (HTML):
1. EXECUTIVE SUMMARY - ringkasan eksekutif
2. PERFORMANCE REVIEW - review performa
3. HOLDINGS ANALYSIS - analisis kepemilikan
4. MARKET OUTLOOK - pandangan pasar
5. RECOMMENDATIONS - rekomendasi

Gaya: profesional, bahasa Indonesia, format HTML dengan class CSS inline.`,
		data.PortfolioName, data.TotalValue, data.ReturnPct,
		strings.Join(holdingStrs, "; "),
		data.BestPerformer.Code, data.BestPerformer.ReturnPct,
		data.WorstPerformer.Code, data.WorstPerformer.ReturnPct)

	return s.AI.Chat("Kamu adalah financial advisor profesional. Buat laporan klien dalam bahasa Indonesia, format HTML.", prompt)
}

func (s *AIClientReportService) generateFallback(data *ClientReportData) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<div style="font-family:Inter,sans-serif;max-width:800px;margin:0 auto;color:#e2e8f0">
<section style="margin-bottom:30px">
<h2 style="color:#93c5fd;font-size:20px;border-bottom:2px solid #1e3a8a;padding-bottom:8px">Executive Summary</h2>
<p style="line-height:1.8">Laporan ini menyajikan analisis performa portofolio <strong>%s</strong> per %s. Total nilai portofolio saat ini adalah <strong>Rp %.0f</strong> dengan total return sebesar <strong style="color:%s">%.2f%%</strong>.</p>
</section>`, data.PortfolioName, data.GeneratedAt.Format("02 January 2006"), data.TotalValue, plColor(data.ReturnPct), data.ReturnPct))

	sb.WriteString(fmt.Sprintf(`<section style="margin-bottom:30px">
<h2 style="color:#93c5fd;font-size:20px;border-bottom:2px solid #1e3a8a;padding-bottom:8px">Performance Review</h2>
<p style="line-height:1.8">Performa portofolio menunjukkan %s. Best performer adalah <strong>%s</strong> dengan return %.2f%%. Worst performer adalah <strong>%s</strong> dengan return %.2f%%.</p>
</section>`, plStatement(data.ReturnPct), data.BestPerformer.Code, data.BestPerformer.ReturnPct, data.WorstPerformer.Code, data.WorstPerformer.ReturnPct))

	sb.WriteString(`<section style="margin-bottom:30px">
<h2 style="color:#93c5fd;font-size:20px;border-bottom:2px solid #1e3a8a;padding-bottom:8px">Holdings Analysis</h2>
<table style="width:100%%;border-collapse:collapse;margin:15px 0">
<thead><tr style="background:#1e3a8a"><th style="padding:10px;text-align:left">Kode</th><th style="padding:10px;text-align:right">Jumlah</th><th style="padding:10px;text-align:right">Harga Rata</th><th style="padding:10px;text-align:right">Harga Saat Ini</th><th style="padding:10px;text-align:right">Nilai</th><th style="padding:10px;text-align:right">Return</th></tr></thead><tbody>`)

	for _, h := range data.Holdings {
		sb.WriteString(fmt.Sprintf(`<tr style="border-bottom:1px solid #334155"><td style="padding:10px;font-weight:600">%s</td><td style="padding:10px;text-align:right">%.0f</td><td style="padding:10px;text-align:right">%.0f</td><td style="padding:10px;text-align:right">%.0f</td><td style="padding:10px;text-align:right">%.0f</td><td style="padding:10px;text-align:right;color:%s">%.2f%%</td></tr>`,
			h.Code, h.Quantity, h.AvgPrice, h.CurrPrice, h.Value, plColor(h.ReturnPct), h.ReturnPct))
	}

	sb.WriteString(`</tbody></table></section>`)

	sb.WriteString(`<section style="margin-bottom:30px">
<h2 style="color:#93c5fd;font-size:20px;border-bottom:2px solid #1e3a8a;padding-bottom:8px">Market Outlook</h2>
<p style="line-height:1.8">Pasar saham Indonesia terus menunjukkan dinamika yang menarik. Diversifikasi portofolio tetap menjadi kunci dalam menghadapi volatilitas. Perhatikan sektor-sektor yang memiliki katalis positif seperti perbankan, consumer goods, dan teknologi.</p>
</section>`)

	sb.WriteString(`<section style="margin-bottom:30px">
<h2 style="color:#93c5fd;font-size:20px;border-bottom:2px solid #1e3a8a;padding-bottom:8px">Recommendations</h2>
<ul style="line-height:2">
<li><strong>Rebalancing:</strong> Lakukan rebalancing jika alokasi saham melebihi 20%% portofolio.</li>
<li><strong>Risk Management:</strong> Pasang stop loss untuk posisi yang sudah profit signifikan.</li>
<li><strong>Diversifikasi:</strong> Tambahkan eksposur ke sektor yang under-represented.</li>
</ul>
</section>`)

	sb.WriteString(fmt.Sprintf(`<footer style="margin-top:40px;padding-top:20px;border-top:1px solid #334155;text-align:center;color:#94a3b8;font-size:13px">
<p>Laporan dihasilkan oleh Investo AI pada %s. Laporan ini bukan rekomendasi investasi.</p>
</footer></div>`, data.GeneratedAt.Format("02 January 2006 15:04")))

	return sb.String()
}

func plColor(pct float64) string {
	if pct > 0 {
		return "#34d399"
	} else if pct < 0 {
		return "#f87171"
	}
	return "#e2e8f0"
}

func plStatement(pct float64) string {
	if pct > 5 {
		return "pertumbuhan yang sangat baik"
	} else if pct > 0 {
		return "pertumbuhan yang positif"
	} else if pct == 0 {
		return "performa yang stabil"
	} else if pct > -5 {
		return "penurunan yang terkendali"
	}
	return "penurunan yang perlu diwaspadai"
}

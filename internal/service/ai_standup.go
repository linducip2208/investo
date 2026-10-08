package service

import (
	"fmt"
	"strings"
	"time"

	"investo/internal/repository"
)

type AIStandupService struct {
	AI             *AIService
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	ForexRepo      *repository.ForexRepository
	PortfolioRepo  *repository.PortfolioRepository
	BreadthSvc     *MarketBreadthService
}

type StandupData struct {
	Date          string  `json:"date"`
	IHSGChange    float64 `json:"ihsg_change"`
	AdvanceCount  int     `json:"advance_count"`
	DeclineCount  int     `json:"decline_count"`
	USDRate       float64 `json:"usd_rate"`
	TopGainers    string  `json:"top_gainers"`
	TopLosers     string  `json:"top_losers"`
	PortfolioPL   float64 `json:"portfolio_pl"`
	PortfolioPLPct float64 `json:"portfolio_pl_pct"`
}

func (s *AIStandupService) GenerateStandup(userID int64) (string, *StandupData, error) {
	now := time.Now()
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta != nil {
		now = now.In(jakarta)
	}

	data := &StandupData{Date: now.Format("02 January 2006")}

	breadth, err := s.BreadthSvc.Calculate()
	if err == nil {
		data.AdvanceCount = breadth.Advance
		data.DeclineCount = breadth.Decline
		total := breadth.Advance + breadth.Decline + breadth.Unchanged
		if total > 0 {
			pct := float64(breadth.Advance) / float64(total) * 100
			data.IHSGChange = pct
		}
	}

	stocks, _ := s.StockRepo.ListActive()
	if len(stocks) > 0 {
		var stockIDs []int64
		for _, st := range stocks {
			stockIDs = append(stockIDs, st.ID)
		}
		priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

		type mover struct {
			Code   string
			Change float64
		}
		var gainers, losers []mover
		for _, st := range stocks {
			_ = priceMap[st.ID]
			change, _ := s.StockPriceRepo.GetPriceChange(st.ID, 1)
			m := mover{Code: st.Code, Change: change}
			if change > 0 {
				gainers = append(gainers, m)
			} else if change < 0 {
				losers = append(losers, m)
			}
		}

		if len(gainers) > 3 {
			gainers = gainers[:3]
		}
		if len(losers) > 3 {
			losers = losers[:3]
		}

		var gainerStrs []string
		for _, g := range gainers {
			gainerStrs = append(gainerStrs, fmt.Sprintf("%s (+%.2f%%)", g.Code, g.Change))
		}
		var loserStrs []string
		for _, l := range losers {
			loserStrs = append(loserStrs, fmt.Sprintf("%s (%.2f%%)", l.Code, l.Change))
		}

		data.TopGainers = strings.Join(gainerStrs, ", ")
		data.TopLosers = strings.Join(loserStrs, ", ")
	}

	usdPair, _ := s.ForexRepo.FindPairByCurrencies("USD", "IDR")
	if usdPair != nil {
		rates, _ := s.ForexRepo.FindLatestRates()
		for _, r := range rates {
			if r.PairID == usdPair.ID {
				data.USDRate = r.Close
				break
			}
		}
	}

	if userID > 0 {
		portfolios, _ := s.PortfolioRepo.FindByUserID(userID)
		if len(portfolios) > 0 {
			var totalValue, totalCost float64
			for _, p := range portfolios {
				_ = p
			}
			if totalCost > 0 {
				data.PortfolioPL = totalValue - totalCost
				data.PortfolioPLPct = (data.PortfolioPL / totalCost) * 100
			}
		}
	}

	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	var briefing string
	if aiAvailable {
		briefing, err = s.generateWithAI(data)
		if err == nil && briefing != "" {
			return briefing, data, nil
		}
	}

	briefing = s.generateFallback(data)
	return briefing, data, nil
}

func (s *AIStandupService) generateWithAI(data *StandupData) (string, error) {
	prompt := fmt.Sprintf(`Buat morning briefing pasar saham Indonesia untuk tim trader. Format "trader meeting":

Selamat pagi team. Berikut update pasar hari ini:

Data:
- Tanggal: %s
- IHSG breadth: %d naik, %d turun
- USD/IDR: %.0f
- Top gainers: %s
- Top losers: %s
- Portfolio P/L: %.2f%%

Format briefing:
1. MARKET OVERVIEW: analisa singkat kondisi pasar
2. TOP MOVERS: komentar AI tentang saham yang bergerak
3. PORTFOLIO UPDATE: observasi performa portofolio
4. RISK ALERTS: peringatan risiko jika ada
5. Tanya: "Ada pertanyaan sebelum kita mulai trading?"

Gaya: profesional, ringkas, bahasa Indonesia. Tidak perlu disclaimer panjang.`,
		data.Date, data.AdvanceCount, data.DeclineCount,
		data.USDRate, data.TopGainers, data.TopLosers, data.PortfolioPLPct)

	return s.AI.Chat("Kamu adalah analis pasar profesional. Buat morning briefing dalam bahasa Indonesia gaya trader meeting. Ringkas dan informatif.", prompt)
}

func (s *AIStandupService) generateFallback(data *StandupData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<p><strong>Selamat pagi team.</strong> Berikut update pasar hari ini, %s:</p>", data.Date))

	sb.WriteString("<h3>Market Overview</h3>")
	breadthStr := "seimbang"
	if data.AdvanceCount > data.DeclineCount {
		breadthStr = "positif"
	} else if data.DeclineCount > data.AdvanceCount {
		breadthStr = "negatif"
	}
	sb.WriteString(fmt.Sprintf("<p>Breadth pasar hari ini <strong>%s</strong> dengan %d saham naik dan %d saham turun. ", breadthStr, data.AdvanceCount, data.DeclineCount))
	if data.USDRate > 0 {
		sb.WriteString(fmt.Sprintf("USD/IDR diperdagangkan di level <strong>Rp %.0f</strong>.</p>", data.USDRate))
	}

	if data.TopGainers != "" {
		sb.WriteString("<h3>Top Movers</h3>")
		sb.WriteString(fmt.Sprintf("<p><strong>Gainers:</strong> %s</p>", data.TopGainers))
	}
	if data.TopLosers != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>Losers:</strong> %s</p>", data.TopLosers))
	}

	if data.PortfolioPLPct != 0 {
		sb.WriteString("<h3>Portfolio Update</h3>")
		plStr := "naik"
		color := "text-green-400"
		if data.PortfolioPLPct < 0 {
			plStr = "turun"
			color = "text-red-400"
		}
		sb.WriteString(fmt.Sprintf("<p>Portfolio Anda <strong class=\"%s\">%s %.2f%%</strong>.</p>", color, plStr, data.PortfolioPLPct))
	}

	sb.WriteString("<h3>Risk Alerts</h3>")
	sb.WriteString("<p>Tidak ada peringatan risiko signifikan untuk hari ini. Tetap pantau posisi Anda.</p>")

	sb.WriteString("<hr>")
	sb.WriteString("<p><strong>Ada pertanyaan sebelum kita mulai trading?</strong></p>")

	return sb.String()
}

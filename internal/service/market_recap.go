package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"investo/internal/repository"
)

type RecapService struct {
	StockRepo        *repository.StockRepository
	StockPriceRepo   *repository.StockPriceRepository
	SectorRepo       *repository.SectorRepository
	ForexRepo        *repository.ForexRepository
	SentimentService *SentimentService
	BreadthService   *MarketBreadthService
}

type DailyRecap struct {
	Date                 string            `json:"date"`
	IHSGSummary          string            `json:"ihsg_summary"`
	TopGainers           []string          `json:"top_gainers"`
	TopLosers            []string          `json:"top_losers"`
	MostActive           []string          `json:"most_active"`
	SectorSummary        map[string]string `json:"sector_summary"`
	ForexSummary         string            `json:"forex_summary"`
	MarketBreadthSummary string            `json:"market_breadth_summary"`
	Anomalies            []string          `json:"anomalies"`
	Highlights           []string          `json:"highlights"`
	MarketSentiment      string            `json:"market_sentiment"`
	TopGainersPct        []float64         `json:"top_gainers_pct"`
	TopLosersPct         []float64         `json:"top_losers_pct"`
	MostActiveVol        []int64           `json:"most_active_vol"`
	BreadthAdvance       int               `json:"breadth_advance"`
	BreadthDecline       int               `json:"breadth_decline"`
	BreadthUnchanged     int               `json:"breadth_unchanged"`
	BreadthAdvancePct    float64           `json:"breadth_advance_pct"`
	BreadthDeclinePct    float64           `json:"breadth_decline_pct"`
	SectorNames          []string          `json:"sector_names"`
	SectorChanges        []float64         `json:"sector_changes"`
}

type stockWithChange struct {
	Code     string
	Name     string
	Price    float64
	Change   float64
	ChangePct float64
	Volume   int64
	SectorID int64
}

func (s *RecapService) Generate() (*DailyRecap, error) {
	recap := &DailyRecap{
		Date:          time.Now().Format("2006-01-02"),
		SectorSummary: make(map[string]string),
	}

	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("RecapService.Generate stocks: %w", err)
	}

	pricesWithPrev, err := s.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, fmt.Errorf("RecapService.Generate prices: %w", err)
	}

	priceMap := make(map[int64]repository.PriceWithPrev)
	for _, p := range pricesWithPrev {
		priceMap[p.StockID] = p
	}

	stockMap := make(map[int64]struct {
		Code     string
		Name     string
		SectorID int64
	})
	for _, st := range stocks {
		stockMap[st.ID] = struct {
			Code     string
			Name     string
			SectorID int64
		}{st.Code, st.Name, st.SectorID}
	}

	var marketStocks []stockWithChange
	for _, pp := range pricesWithPrev {
		stInfo, ok := stockMap[pp.StockID]
		if !ok {
			continue
		}

		change := pp.LatestClose - pp.PrevClose
		changePct := 0.0
		if pp.PrevClose > 0 {
			changePct = (change / pp.PrevClose) * 100
		}

		marketStocks = append(marketStocks, stockWithChange{
			Code:      stInfo.Code,
			Name:      stInfo.Name,
			Price:     pp.LatestClose,
			Change:    change,
			ChangePct: changePct,
			Volume:    pp.LatestVolume,
			SectorID:  stInfo.SectorID,
		})
	}

	if len(marketStocks) > 0 {
		sort.Slice(marketStocks, func(i, j int) bool {
			return marketStocks[i].ChangePct > marketStocks[j].ChangePct
		})

		gainerCount := 5
		if len(marketStocks) < 5 {
			gainerCount = len(marketStocks)
		}
		for i := 0; i < gainerCount; i++ {
			s := marketStocks[i]
			recap.TopGainers = append(recap.TopGainers, fmt.Sprintf("%s (%s) +%.2f%%", s.Code, s.Name, s.ChangePct))
			recap.TopGainersPct = append(recap.TopGainersPct, math.Round(s.ChangePct*100)/100)
		}

		sort.Slice(marketStocks, func(i, j int) bool {
			return marketStocks[i].ChangePct < marketStocks[j].ChangePct
		})

		loserCount := 5
		if len(marketStocks) < 5 {
			loserCount = len(marketStocks)
		}
		for i := 0; i < loserCount; i++ {
			s := marketStocks[i]
			recap.TopLosers = append(recap.TopLosers, fmt.Sprintf("%s (%s) %.2f%%", s.Code, s.Name, s.ChangePct))
			recap.TopLosersPct = append(recap.TopLosersPct, math.Round(s.ChangePct*100)/100)
		}

		sort.Slice(marketStocks, func(i, j int) bool {
			return marketStocks[i].Volume > marketStocks[j].Volume
		})

		activeCount := 5
		if len(marketStocks) < 5 {
			activeCount = len(marketStocks)
		}
		for i := 0; i < activeCount; i++ {
			s := marketStocks[i]
			volumeStr := formatVolumeShort(s.Volume)
			recap.MostActive = append(recap.MostActive, fmt.Sprintf("%s (%s) %s lembar", s.Code, s.Name, volumeStr))
			recap.MostActiveVol = append(recap.MostActiveVol, s.Volume)
		}
	}

	sectors, _ := s.SectorRepo.FindAll()
	sectorMap := make(map[int64]string)
	for _, sec := range sectors {
		sectorMap[sec.ID] = sec.Name
	}

	sectorChanges := make(map[int64][]float64)
	for _, ms := range marketStocks {
		sectorChanges[ms.SectorID] = append(sectorChanges[ms.SectorID], ms.ChangePct)
	}

	for secID, changes := range sectorChanges {
		avg := 0.0
		for _, c := range changes {
			avg += c
		}
		avg = avg / float64(len(changes))
		avg = math.Round(avg*100) / 100

		secName := sectorMap[secID]
		if avg >= 0 {
			recap.SectorSummary[secName] = fmt.Sprintf("naik %.2f%%", avg)
		} else {
			recap.SectorSummary[secName] = fmt.Sprintf("turun %.2f%%", math.Abs(avg))
		}
		recap.SectorNames = append(recap.SectorNames, secName)
		recap.SectorChanges = append(recap.SectorChanges, avg)
	}

	mb, err := s.BreadthService.Calculate()
	if err == nil && mb != nil {
		recap.BreadthAdvance = mb.Advance
		recap.BreadthDecline = mb.Decline
		recap.BreadthUnchanged = mb.Unchanged
		recap.BreadthAdvancePct = mb.AdvancePercent
		recap.BreadthDeclinePct = mb.DeclinePercent

		if mb.Advance > mb.Decline {
			recap.MarketBreadthSummary = fmt.Sprintf("%d saham naik, %d saham turun, %d tidak berubah — pasar positif", mb.Advance, mb.Decline, mb.Unchanged)
		} else if mb.Decline > mb.Advance {
			recap.MarketBreadthSummary = fmt.Sprintf("%d saham turun, %d saham naik, %d tidak berubah — pasar negatif", mb.Decline, mb.Advance, mb.Unchanged)
		} else {
			recap.MarketBreadthSummary = fmt.Sprintf("%d saham naik, %d saham turun, %d tidak berubah — pasar seimbang", mb.Advance, mb.Decline, mb.Unchanged)
		}

		advPct := mb.AdvancePercent
		if advPct > 60 {
			recap.MarketSentiment = "bullish"
		} else if mb.DeclinePercent > 60 {
			recap.MarketSentiment = "bearish"
		} else {
			recap.MarketSentiment = "mixed"
		}
	}

	forexRates, err := s.ForexRepo.FindLatestRates()
	if err == nil && len(forexRates) > 0 {
		var usdRate float64
		for _, fr := range forexRates {
			pair, _ := s.ForexRepo.FindPairByID(fr.PairID)
			if pair != nil && pair.BaseCurrency == "USD" && pair.QuoteCurrency == "IDR" {
				usdRate = fr.Close
				break
			}
		}
		if usdRate > 0 {
			recap.ForexSummary = fmt.Sprintf("USD/IDR: Rp %.0f. ", usdRate)
		}
		recap.ForexSummary += fmt.Sprintf("%d pasangan mata uang tersedia", len(forexRates))
	} else {
		recap.ForexSummary = "Data forex tidak tersedia"
	}

	avgChange := 0.0
	for _, ms := range marketStocks {
		avgChange += ms.ChangePct
	}
	if len(marketStocks) > 0 {
		avgChange = avgChange / float64(len(marketStocks))
	}

	if avgChange > 0 {
		recap.IHSGSummary = fmt.Sprintf("Pasar saham Indonesia ditutup positif dengan rata-rata perubahan %.2f%%. Sebanyak %d saham menguat dan %d saham melemah.", avgChange, recap.BreadthAdvance, recap.BreadthDecline)
	} else if avgChange < 0 {
		recap.IHSGSummary = fmt.Sprintf("Pasar saham Indonesia ditutup negatif dengan rata-rata perubahan %.2f%%. Sebanyak %d saham melemah dan %d saham menguat.", avgChange, recap.BreadthDecline, recap.BreadthAdvance)
	} else {
		recap.IHSGSummary = fmt.Sprintf("Pasar saham Indonesia ditutup flat. %d saham naik, %d saham turun, %d tidak berubah.", recap.BreadthAdvance, recap.BreadthDecline, recap.BreadthUnchanged)
	}

	recap.Highlights = s.generateHighlights(marketStocks, sectorMap, recap)

	return recap, nil
}

func (s *RecapService) generateHighlights(stocks []stockWithChange, sectorMap map[int64]string, recap *DailyRecap) []string {
	var highlights []string

	sectorAvg := make(map[int64]struct {
		Total float64
		Count int
		Name  string
	})
	for _, ms := range stocks {
		d := sectorAvg[ms.SectorID]
		d.Total += ms.ChangePct
		d.Count++
		d.Name = sectorMap[ms.SectorID]
		sectorAvg[ms.SectorID] = d
	}

	var bestSector string
	bestAvg := -999.0
	var worstSector string
	worstAvg := 999.0

	for _, d := range sectorAvg {
		if d.Count == 0 {
			continue
		}
		avg := d.Total / float64(d.Count)
		if avg > bestAvg {
			bestAvg = avg
			bestSector = d.Name
		}
		if avg < worstAvg {
			worstAvg = avg
			worstSector = d.Name
		}
	}

	if bestSector != "" && bestAvg > 0 {
		highlights = append(highlights, fmt.Sprintf("Sektor %s memimpin penguatan hari ini dengan rata-rata kenaikan %.1f%%", bestSector, bestAvg))
	}

	if worstSector != "" && worstAvg < 0 {
		highlights = append(highlights, fmt.Sprintf("Sektor %s mengalami tekanan dengan rata-rata penurunan %.1f%%", worstSector, math.Abs(worstAvg)))
	}

	if len(recap.TopGainers) > 0 {
		parts := strings.SplitN(recap.TopGainers[0], " ", 2)
		code := parts[0]
		highlights = append(highlights, fmt.Sprintf("%s menjadi top gainer hari ini dengan kenaikan %.2f%%", code, recap.TopGainersPct[0]))
	}

	if len(recap.MostActive) > 0 {
		parts := strings.SplitN(recap.MostActive[0], " ", 2)
		code := parts[0]
		volStr := formatVolumeShort(recap.MostActiveVol[0])
		highlights = append(highlights, fmt.Sprintf("%s mencatat volume perdagangan tertinggi dengan %s lembar saham", code, volStr))
	}

	if recap.BreadthAdvance > recap.BreadthDecline*2 {
		highlights = append(highlights, fmt.Sprintf("Market breadth sangat bullish: %d saham naik vs %d saham turun (rasio %.1f:1)", recap.BreadthAdvance, recap.BreadthDecline, float64(recap.BreadthAdvance)/float64(max(recap.BreadthDecline, 1))))
	} else if recap.BreadthDecline > recap.BreadthAdvance*2 {
		highlights = append(highlights, fmt.Sprintf("Market breadth bearish: %d saham turun vs %d saham naik (rasio %.1f:1)", recap.BreadthDecline, recap.BreadthAdvance, float64(recap.BreadthDecline)/float64(max(recap.BreadthAdvance, 1))))
	}

	return highlights
}

func formatVolumeShort(vol int64) string {
	if vol >= 1_000_000_000 {
		return fmt.Sprintf("%.1f M", float64(vol)/1_000_000_000)
	}
	if vol >= 1_000_000 {
		return fmt.Sprintf("%.1f Jt", float64(vol)/1_000_000)
	}
	if vol >= 1_000 {
		return fmt.Sprintf("%.0f rb", float64(vol)/1_000)
	}
	return fmt.Sprintf("%d", vol)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

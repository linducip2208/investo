package service

import (
	"fmt"
	"strings"
)

type COTReport struct {
	Pair            string  `json:"pair"`
	LongPositions   int64   `json:"long_positions"`
	ShortPositions  int64   `json:"short_positions"`
	NetPositions    int64   `json:"net_positions"`
	OpenInterest    int64   `json:"open_interest"`
	ChangeWeek      int64   `json:"change_week"`
	ChangeMonth     int64   `json:"change_month"`
	LongPct         float64 `json:"long_pct"`
	ShortPct        float64 `json:"short_pct"`
	COTIndex        float64 `json:"cot_index"`
	Sentiment       string  `json:"sentiment"`
	LastUpdated     string  `json:"last_updated"`
	IsIllustrative  bool    `json:"is_illustrative"`
	HistoricalPositions []COTWeek `json:"historical_positions,omitempty"`
}

type COTWeek struct {
	WeekStart      string `json:"week_start"`
	LongPositions  int64  `json:"long_positions"`
	ShortPositions int64  `json:"short_positions"`
	NetPositions   int64  `json:"net_positions"`
}

type COTDataService struct{}

var sampleCOTData = map[string]COTReport{
	"EUR/USD": {
		Pair: "EUR/USD", LongPositions: 185000, ShortPositions: 142000,
		NetPositions: 43000, OpenInterest: 327000,
		ChangeWeek: 8500, ChangeMonth: -2300,
		LastUpdated: "2026-08-01",
	},
	"GBP/USD": {
		Pair: "GBP/USD", LongPositions: 95000, ShortPositions: 112000,
		NetPositions: -17000, OpenInterest: 207000,
		ChangeWeek: -4200, ChangeMonth: 8100,
		LastUpdated: "2026-08-01",
	},
	"USD/JPY": {
		Pair: "USD/JPY", LongPositions: 72000, ShortPositions: 68000,
		NetPositions: 4000, OpenInterest: 140000,
		ChangeWeek: 1200, ChangeMonth: -5600,
		LastUpdated: "2026-08-01",
	},
	"USD/CHF": {
		Pair: "USD/CHF", LongPositions: 24000, ShortPositions: 45000,
		NetPositions: -21000, OpenInterest: 69000,
		ChangeWeek: -3800, ChangeMonth: 1500,
		LastUpdated: "2026-08-01",
	},
	"AUD/USD": {
		Pair: "AUD/USD", LongPositions: 58000, ShortPositions: 51000,
		NetPositions: 7000, OpenInterest: 109000,
		ChangeWeek: 3200, ChangeMonth: -1100,
		LastUpdated: "2026-08-01",
	},
	"NZD/USD": {
		Pair: "NZD/USD", LongPositions: 31000, ShortPositions: 28000,
		NetPositions: 3000, OpenInterest: 59000,
		ChangeWeek: 800, ChangeMonth: 2100,
		LastUpdated: "2026-08-01",
	},
	"USD/CAD": {
		Pair: "USD/CAD", LongPositions: 44000, ShortPositions: 62000,
		NetPositions: -18000, OpenInterest: 106000,
		ChangeWeek: -2100, ChangeMonth: -4400,
		LastUpdated: "2026-08-01",
	},
	"EUR/JPY": {
		Pair: "EUR/JPY", LongPositions: 66000, ShortPositions: 41000,
		NetPositions: 25000, OpenInterest: 107000,
		ChangeWeek: 5100, ChangeMonth: 3700,
		LastUpdated: "2026-08-01",
	},
	"GBP/JPY": {
		Pair: "GBP/JPY", LongPositions: 38000, ShortPositions: 44000,
		NetPositions: -6000, OpenInterest: 82000,
		ChangeWeek: -1500, ChangeMonth: 2800,
		LastUpdated: "2026-08-01",
	},
	"EUR/GBP": {
		Pair: "EUR/GBP", LongPositions: 52000, ShortPositions: 47000,
		NetPositions: 5000, OpenInterest: 99000,
		ChangeWeek: 1900, ChangeMonth: -800,
		LastUpdated: "2026-08-01",
	},
}

var historicalCOT = map[string][]COTWeek{
	"EUR/USD": {
		{WeekStart: "2026-07-04", LongPositions: 172000, ShortPositions: 155000, NetPositions: 17000},
		{WeekStart: "2026-07-11", LongPositions: 178000, ShortPositions: 148000, NetPositions: 30000},
		{WeekStart: "2026-07-18", LongPositions: 181000, ShortPositions: 144000, NetPositions: 37000},
		{WeekStart: "2026-07-25", LongPositions: 185000, ShortPositions: 142000, NetPositions: 43000},
	},
	"GBP/USD": {
		{WeekStart: "2026-07-04", LongPositions: 102000, ShortPositions: 106000, NetPositions: -4000},
		{WeekStart: "2026-07-11", LongPositions: 98000, ShortPositions: 110000, NetPositions: -12000},
		{WeekStart: "2026-07-18", LongPositions: 96000, ShortPositions: 113000, NetPositions: -17000},
		{WeekStart: "2026-07-25", LongPositions: 95000, ShortPositions: 112000, NetPositions: -17000},
	},
	"AUD/USD": {
		{WeekStart: "2026-07-04", LongPositions: 52000, ShortPositions: 55000, NetPositions: -3000},
		{WeekStart: "2026-07-11", LongPositions: 55000, ShortPositions: 52000, NetPositions: 3000},
		{WeekStart: "2026-07-18", LongPositions: 56000, ShortPositions: 51000, NetPositions: 5000},
		{WeekStart: "2026-07-25", LongPositions: 58000, ShortPositions: 51000, NetPositions: 7000},
	},
}

func NewCOTDataService() *COTDataService {
	return &COTDataService{}
}

func (s *COTDataService) GetCOTData(pair string) (*COTReport, error) {
	pair = strings.ToUpper(pair)
	if !strings.Contains(pair, "/") {
		if len(pair) == 6 {
			pair = pair[:3] + "/" + pair[3:]
		}
	}

	report, ok := sampleCOTData[pair]
	if !ok {
		return nil, fmt.Errorf("data COT untuk %s tidak tersedia. COT real membutuhkan API berbayar (CFTC/ICE)", pair)
	}

	total := report.LongPositions + report.ShortPositions
	if total > 0 {
		report.LongPct = float64(report.LongPositions) / float64(total) * 100
		report.ShortPct = float64(report.ShortPositions) / float64(total) * 100
	}

	allNets := collectNetPositions()

	minNet := int64(99999999)
	maxNet := int64(-99999999)
	for _, net := range allNets {
		if net < minNet {
			minNet = net
		}
		if net > maxNet {
			maxNet = net
		}
	}

	if maxNet > minNet {
		report.COTIndex = float64(report.NetPositions-minNet) / float64(maxNet-minNet) * 100
	}

	if report.COTIndex >= 80 {
		report.Sentiment = "Sangat Bullish (overbought pada kontrak)"
	} else if report.COTIndex >= 60 {
		report.Sentiment = "Bullish"
	} else if report.COTIndex >= 40 {
		report.Sentiment = "Netral"
	} else if report.COTIndex >= 20 {
		report.Sentiment = "Bearish"
	} else {
		report.Sentiment = "Sangat Bearish (oversold pada kontrak)"
	}

	report.IsIllustrative = true

	if hist, ok := historicalCOT[pair]; ok {
		report.HistoricalPositions = hist
	}

	return &report, nil
}

func (s *COTDataService) GetAllCOTData() []COTReport {
	var reports []COTReport
	for _, r := range sampleCOTData {
		total := r.LongPositions + r.ShortPositions
		if total > 0 {
			r.LongPct = float64(r.LongPositions) / float64(total) * 100
			r.ShortPct = float64(r.ShortPositions) / float64(total) * 100
		}
		r.IsIllustrative = true
		reports = append(reports, r)
	}
	return reports
}

func collectNetPositions() []int64 {
	var nets []int64
	for _, r := range sampleCOTData {
		nets = append(nets, r.NetPositions)
	}
	return nets
}

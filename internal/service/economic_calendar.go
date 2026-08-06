package service

import (
	"sort"
	"strings"
	"time"
)

type EconomicEvent struct {
	Date       string  `json:"date"`
	Time       string  `json:"time"`
	Currency   string  `json:"currency"`
	Event      string  `json:"event"`
	Importance string  `json:"importance"`
	Previous   string  `json:"previous"`
	Forecast   string  `json:"forecast"`
	Actual     string  `json:"actual"`
}

type EconomicCalendarService struct{}

func (s *EconomicCalendarService) GetUpcomingEvents(days int) []EconomicEvent {
	cutoff := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	var result []EconomicEvent
	for _, e := range prebuiltEvents {
		if e.Date >= today && e.Date <= cutoff {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		importanceOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		return importanceOrder[result[i].Importance] < importanceOrder[result[j].Importance]
	})
	return result
}

func (s *EconomicCalendarService) GetTodayEvents() []EconomicEvent {
	today := time.Now().Format("2006-01-02")
	var result []EconomicEvent
	for _, e := range prebuiltEvents {
		if e.Date == today {
			result = append(result, e)
		}
	}
	return result
}

func (s *EconomicCalendarService) FilterEvents(events []EconomicEvent, importance, currency, dateFrom, dateTo string) []EconomicEvent {
	var result []EconomicEvent
	for _, e := range events {
		if importance != "" && importance != "all" && e.Importance != importance {
			continue
		}
		if currency != "" && currency != "all" && !strings.EqualFold(e.Currency, currency) {
			continue
		}
		if dateFrom != "" && e.Date < dateFrom {
			continue
		}
		if dateTo != "" && e.Date > dateTo {
			continue
		}
		result = append(result, e)
	}
	return result
}

var prebuiltEvents = []EconomicEvent{
	{
		Date: "2026-01-08", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Januari",
		Importance: "high", Previous: "5.75%", Forecast: "5.75%", Actual: "",
	},
	{
		Date: "2026-01-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Desember",
		Importance: "medium", Previous: "$2.41B", Forecast: "$2.30B", Actual: "",
	},
	{
		Date: "2026-01-15", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Desember",
		Importance: "high", Previous: "2.7%", Forecast: "2.8%", Actual: "",
	},
	{
		Date: "2026-01-20", Time: "11:00", Currency: "IDR", Event: "Indonesia GDP Q4 2025",
		Importance: "high", Previous: "5.02%", Forecast: "5.03%", Actual: "",
	},
	{
		Date: "2026-01-28", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.75%", Forecast: "2.75%", Actual: "",
	},
	{
		Date: "2026-01-29", Time: "19:30", Currency: "USD", Event: "US GDP Q4 2025 (Advance)",
		Importance: "high", Previous: "3.1%", Forecast: "2.8%", Actual: "",
	},
	{
		Date: "2026-02-02", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Januari",
		Importance: "high", Previous: "1.57%", Forecast: "1.60%", Actual: "",
	},
	{
		Date: "2026-02-07", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Januari",
		Importance: "high", Previous: "256K", Forecast: "180K", Actual: "",
	},
	{
		Date: "2026-02-12", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Januari",
		Importance: "high", Previous: "2.8%", Forecast: "2.7%", Actual: "",
	},
	{
		Date: "2026-02-14", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Februari",
		Importance: "high", Previous: "5.75%", Forecast: "5.75%", Actual: "",
	},
	{
		Date: "2026-02-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Januari",
		Importance: "medium", Previous: "$2.30B", Forecast: "$2.20B", Actual: "",
	},
	{
		Date: "2026-03-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Februari",
		Importance: "high", Previous: "1.60%", Forecast: "1.55%", Actual: "",
	},
	{
		Date: "2026-03-05", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.75%", Forecast: "2.50%", Actual: "",
	},
	{
		Date: "2026-03-07", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Februari",
		Importance: "high", Previous: "180K", Forecast: "190K", Actual: "",
	},
	{
		Date: "2026-03-12", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Februari",
		Importance: "high", Previous: "2.7%", Forecast: "2.6%", Actual: "",
	},
	{
		Date: "2026-03-17", Time: "19:00", Currency: "USD", Event: "FOMC Federal Funds Rate",
		Importance: "high", Previous: "4.25%", Forecast: "4.25%", Actual: "",
	},
	{
		Date: "2026-03-19", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Maret",
		Importance: "high", Previous: "5.75%", Forecast: "5.75%", Actual: "",
	},
	{
		Date: "2026-03-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Februari",
		Importance: "medium", Previous: "$2.20B", Forecast: "$2.30B", Actual: "",
	},
	{
		Date: "2026-04-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Maret",
		Importance: "high", Previous: "1.55%", Forecast: "1.60%", Actual: "",
	},
	{
		Date: "2026-04-04", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Maret",
		Importance: "high", Previous: "190K", Forecast: "175K", Actual: "",
	},
	{
		Date: "2026-04-10", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Maret",
		Importance: "high", Previous: "2.6%", Forecast: "2.5%", Actual: "",
	},
	{
		Date: "2026-04-16", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) April",
		Importance: "high", Previous: "5.75%", Forecast: "5.50%", Actual: "",
	},
	{
		Date: "2026-04-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Maret",
		Importance: "medium", Previous: "$2.30B", Forecast: "$2.15B", Actual: "",
	},
	{
		Date: "2026-04-23", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.50%", Forecast: "2.50%", Actual: "",
	},
	{
		Date: "2026-04-28", Time: "19:30", Currency: "USD", Event: "US GDP Q1 2026 (Advance)",
		Importance: "high", Previous: "2.8%", Forecast: "2.5%", Actual: "",
	},
	{
		Date: "2026-05-02", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) April",
		Importance: "high", Previous: "1.60%", Forecast: "1.55%", Actual: "",
	},
	{
		Date: "2026-05-02", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls April",
		Importance: "high", Previous: "175K", Forecast: "185K", Actual: "",
	},
	{
		Date: "2026-05-05", Time: "19:00", Currency: "USD", Event: "FOMC Federal Funds Rate",
		Importance: "high", Previous: "4.25%", Forecast: "4.00%", Actual: "",
	},
	{
		Date: "2026-05-14", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Mei",
		Importance: "high", Previous: "5.50%", Forecast: "5.50%", Actual: "",
	},
	{
		Date: "2026-05-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance April",
		Importance: "medium", Previous: "$2.15B", Forecast: "$2.40B", Actual: "",
	},
	{
		Date: "2026-05-14", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) April",
		Importance: "high", Previous: "2.5%", Forecast: "2.6%", Actual: "",
	},
	{
		Date: "2026-06-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Mei",
		Importance: "high", Previous: "1.55%", Forecast: "1.60%", Actual: "",
	},
	{
		Date: "2026-06-06", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Mei",
		Importance: "high", Previous: "185K", Forecast: "200K", Actual: "",
	},
	{
		Date: "2026-06-11", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.50%", Forecast: "2.25%", Actual: "",
	},
	{
		Date: "2026-06-12", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Mei",
		Importance: "high", Previous: "2.6%", Forecast: "2.5%", Actual: "",
	},
	{
		Date: "2026-06-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Mei",
		Importance: "medium", Previous: "$2.40B", Forecast: "$2.10B", Actual: "",
	},
	{
		Date: "2026-06-18", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Juni",
		Importance: "high", Previous: "5.50%", Forecast: "5.25%", Actual: "",
	},
	{
		Date: "2026-06-24", Time: "19:00", Currency: "USD", Event: "FOMC Federal Funds Rate",
		Importance: "high", Previous: "4.00%", Forecast: "4.00%", Actual: "",
	},
	{
		Date: "2026-07-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Juni",
		Importance: "high", Previous: "1.60%", Forecast: "1.55%", Actual: "",
	},
	{
		Date: "2026-07-03", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Juni",
		Importance: "high", Previous: "200K", Forecast: "190K", Actual: "",
	},
	{
		Date: "2026-07-15", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Juni",
		Importance: "high", Previous: "2.5%", Forecast: "2.4%", Actual: "",
	},
	{
		Date: "2026-07-16", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Juli",
		Importance: "high", Previous: "5.25%", Forecast: "5.25%", Actual: "",
	},
	{
		Date: "2026-07-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Juni",
		Importance: "medium", Previous: "$2.10B", Forecast: "$2.30B", Actual: "",
	},
	{
		Date: "2026-07-23", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.25%", Forecast: "2.25%", Actual: "",
	},
	{
		Date: "2026-07-30", Time: "19:30", Currency: "USD", Event: "US GDP Q2 2026 (Advance)",
		Importance: "high", Previous: "2.5%", Forecast: "2.7%", Actual: "",
	},
	{
		Date: "2026-08-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Juli",
		Importance: "high", Previous: "1.55%", Forecast: "1.50%", Actual: "",
	},
	{
		Date: "2026-08-07", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Juli",
		Importance: "high", Previous: "190K", Forecast: "175K", Actual: "",
	},
	{
		Date: "2026-08-13", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Juli",
		Importance: "high", Previous: "2.4%", Forecast: "2.3%", Actual: "",
	},
	{
		Date: "2026-08-20", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Agustus",
		Importance: "high", Previous: "5.25%", Forecast: "5.00%", Actual: "",
	},
	{
		Date: "2026-08-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Juli",
		Importance: "medium", Previous: "$2.30B", Forecast: "$2.45B", Actual: "",
	},
	{
		Date: "2026-09-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Agustus",
		Importance: "high", Previous: "1.50%", Forecast: "1.55%", Actual: "",
	},
	{
		Date: "2026-09-04", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Agustus",
		Importance: "high", Previous: "175K", Forecast: "170K", Actual: "",
	},
	{
		Date: "2026-09-10", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.25%", Forecast: "2.00%", Actual: "",
	},
	{
		Date: "2026-09-12", Time: "19:00", Currency: "USD", Event: "FOMC Federal Funds Rate",
		Importance: "high", Previous: "4.00%", Forecast: "3.75%", Actual: "",
	},
	{
		Date: "2026-09-17", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) September",
		Importance: "high", Previous: "5.00%", Forecast: "5.00%", Actual: "",
	},
	{
		Date: "2026-09-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Agustus",
		Importance: "medium", Previous: "$2.45B", Forecast: "$2.35B", Actual: "",
	},
	{
		Date: "2026-10-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) September",
		Importance: "high", Previous: "1.55%", Forecast: "1.60%", Actual: "",
	},
	{
		Date: "2026-10-03", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls September",
		Importance: "high", Previous: "170K", Forecast: "180K", Actual: "",
	},
	{
		Date: "2026-10-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance September",
		Importance: "medium", Previous: "$2.35B", Forecast: "$2.25B", Actual: "",
	},
	{
		Date: "2026-10-22", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Oktober",
		Importance: "high", Previous: "5.00%", Forecast: "5.00%", Actual: "",
	},
	{
		Date: "2026-10-29", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.00%", Forecast: "2.00%", Actual: "",
	},
	{
		Date: "2026-10-29", Time: "19:30", Currency: "USD", Event: "US GDP Q3 2026 (Advance)",
		Importance: "high", Previous: "2.7%", Forecast: "2.5%", Actual: "",
	},
	{
		Date: "2026-11-02", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) Oktober",
		Importance: "high", Previous: "1.60%", Forecast: "1.55%", Actual: "",
	},
	{
		Date: "2026-11-07", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls Oktober",
		Importance: "high", Previous: "180K", Forecast: "165K", Actual: "",
	},
	{
		Date: "2026-11-12", Time: "19:00", Currency: "USD", Event: "FOMC Federal Funds Rate",
		Importance: "high", Previous: "3.75%", Forecast: "3.75%", Actual: "",
	},
	{
		Date: "2026-11-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance Oktober",
		Importance: "medium", Previous: "$2.25B", Forecast: "$2.15B", Actual: "",
	},
	{
		Date: "2026-11-19", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) November",
		Importance: "high", Previous: "5.00%", Forecast: "4.75%", Actual: "",
	},
	{
		Date: "2026-11-14", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) Oktober",
		Importance: "high", Previous: "2.3%", Forecast: "2.3%", Actual: "",
	},
	{
		Date: "2026-12-01", Time: "07:30", Currency: "IDR", Event: "Indonesia Inflation (CPI) November",
		Importance: "high", Previous: "1.55%", Forecast: "1.50%", Actual: "",
	},
	{
		Date: "2026-12-05", Time: "19:30", Currency: "USD", Event: "US Nonfarm Payrolls November",
		Importance: "high", Previous: "165K", Forecast: "175K", Actual: "",
	},
	{
		Date: "2026-12-11", Time: "19:30", Currency: "USD", Event: "US CPI (YoY) November",
		Importance: "high", Previous: "2.3%", Forecast: "2.2%", Actual: "",
	},
	{
		Date: "2026-12-17", Time: "14:00", Currency: "IDR", Event: "BI Rapat Dewan Gubernur (RDG) Desember",
		Importance: "high", Previous: "4.75%", Forecast: "4.75%", Actual: "",
	},
	{
		Date: "2026-12-15", Time: "11:00", Currency: "IDR", Event: "Indonesia Trade Balance November",
		Importance: "medium", Previous: "$2.15B", Forecast: "$2.25B", Actual: "",
	},
	{
		Date: "2026-12-17", Time: "14:00", Currency: "EUR", Event: "ECB Interest Rate Decision",
		Importance: "high", Previous: "2.00%", Forecast: "1.75%", Actual: "",
	},
	{
		Date: "2026-12-16", Time: "19:00", Currency: "USD", Event: "FOMC Federal Funds Rate (Final 2026)",
		Importance: "high", Previous: "3.75%", Forecast: "3.50%", Actual: "",
	},
	{
		Date: "2026-07-15", Time: "19:30", Currency: "USD", Event: "US Retail Sales (MoM) Juni",
		Importance: "medium", Previous: "0.3%", Forecast: "0.2%", Actual: "",
	},
	{
		Date: "2026-08-15", Time: "19:30", Currency: "USD", Event: "US Retail Sales (MoM) Juli",
		Importance: "medium", Previous: "0.2%", Forecast: "0.3%", Actual: "",
	},
	{
		Date: "2026-04-16", Time: "08:30", Currency: "CNY", Event: "China GDP Q1 2026",
		Importance: "high", Previous: "5.0%", Forecast: "4.8%", Actual: "",
	},
	{
		Date: "2026-07-16", Time: "08:30", Currency: "CNY", Event: "China GDP Q2 2026",
		Importance: "high", Previous: "4.8%", Forecast: "4.7%", Actual: "",
	},
	{
		Date: "2026-10-16", Time: "08:30", Currency: "CNY", Event: "China GDP Q3 2026",
		Importance: "high", Previous: "4.7%", Forecast: "4.6%", Actual: "",
	},
}

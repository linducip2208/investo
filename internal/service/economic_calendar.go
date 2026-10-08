package service

import (
	"sort"
	"strings"
	"time"
)

type EconomicEvent struct {
	Date       string `json:"date"`
	Time       string `json:"time"`
	Currency   string `json:"currency"`
	Event      string `json:"event"`
	Importance string `json:"importance"`
	Previous   string `json:"previous"`
	Forecast   string `json:"forecast"`
	Actual     string `json:"actual"`
	Schedule   string `json:"schedule,omitempty"`
	Note       string `json:"note,omitempty"`
}

type EconomicCalendarService struct{}

const tentativeNote = "jadwal dapat berubah"

// rollingEvents generates recurring macro events relative to now so the
// calendar never goes stale. All dates are estimates of the usual release
// schedule — never presented as confirmed. Pure function, covered by tests.
func rollingEvents(from, to time.Time) []EconomicEvent {
	var events []EconomicEvent
	add := func(date time.Time, clock, currency, name, importance string) {
		if date.Before(from) || date.After(to) {
			return
		}
		events = append(events, EconomicEvent{
			Date: date.Format("2006-01-02"), Time: clock, Currency: currency,
			Event: name, Importance: importance,
			Previous: "-", Forecast: "-", Actual: "",
			Schedule: "tentative", Note: tentativeNote,
		})
	}

	// Monthly: BPS inflation (1st), trade balance (15th), BI RDG (~18th).
	month := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	for ; !month.After(to); month = month.AddDate(0, 1, 0) {
		y, m := month.Year(), month.Month()
		add(time.Date(y, m, 1, 0, 0, 0, 0, time.UTC), "07:30", "IDR", "Indonesia Inflation (CPI) "+indonesianMonth(m), "high")
		add(time.Date(y, m, 15, 0, 0, 0, 0, time.UTC), "11:00", "IDR", "Indonesia Trade Balance "+indonesianMonth(m), "medium")
		add(time.Date(y, m, 18, 0, 0, 0, 0, time.UTC), "14:00", "IDR", "BI Rapat Dewan Gubernur (RDG) "+indonesianMonth(m), "high")
		// US NFP: first Friday. US CPI: ~12th.
		add(firstWeekdayOfMonth(y, m, time.Friday), "19:30", "USD", "US Nonfarm Payrolls "+indonesianMonth(m), "high")
		add(time.Date(y, m, 12, 0, 0, 0, 0, time.UTC), "19:30", "USD", "US CPI (YoY) "+indonesianMonth(m), "high")
	}

	// Quarterly: ID GDP (Feb/May/Aug/Nov), earnings season, China GDP.
	quarterMonths := []time.Month{time.February, time.May, time.August, time.November}
	for _, m := range quarterMonths {
		for _, y := range []int{from.Year(), from.Year() + 1} {
			q := (int(m)-1)/3 + 1
			add(time.Date(y, m, 5, 0, 0, 0, 0, time.UTC), "11:00", "IDR", "Indonesia GDP Q"+quarterStr(q), "high")
			add(time.Date(y, m, 10, 0, 0, 0, 0, time.UTC), "09:00", "IDR", "Musim Laporan Keuangan Q"+quarterStr(q)+" (Earnings Season)", "medium")
			add(time.Date(y, m, 16, 0, 0, 0, 0, time.UTC), "08:30", "CNY", "China GDP Q"+quarterStr(q), "high")
			add(time.Date(y, m, 29, 0, 0, 0, 0, time.UTC), "19:30", "USD", "US GDP Q"+quarterStr(q)+" (Advance)", "high")
		}
	}

	// FOMC ~8x yearly: anchor then step ~6 weeks.
	for d := time.Date(2025, 1, 29, 0, 0, 0, 0, time.UTC); !d.After(to); d = d.AddDate(0, 0, 45) {
		add(d, "19:00", "USD", "FOMC Federal Funds Rate", "high")
	}
	// ECB ~8x yearly.
	for d := time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC); !d.After(to); d = d.AddDate(0, 0, 42) {
		add(d, "14:00", "EUR", "ECB Interest Rate Decision", "high")
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date < events[j].Date
		}
		importanceOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		return importanceOrder[events[i].Importance] < importanceOrder[events[j].Importance]
	})
	return events
}

func quarterStr(q int) string {
	return string(rune('0' + q))
}

func indonesianMonth(m time.Month) string {
	names := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	if m < 1 || m > 12 {
		return ""
	}
	return names[int(m)]
}

func firstWeekdayOfMonth(year int, month time.Month, weekday time.Weekday) time.Time {
	d := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	for d.Weekday() != weekday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

func (s *EconomicCalendarService) GetUpcomingEvents(days int) []EconomicEvent {
	if days < 1 {
		days = 30
	}
	now := time.Now()
	cutoff := now.AddDate(0, 0, days)
	result := rollingEvents(startOfDay(now), endOfDay(cutoff))
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
	now := time.Now()
	return rollingEvents(startOfDay(now), endOfDay(now))
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
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

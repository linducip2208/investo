package service

import (
	"fmt"
	"strings"
	"time"
)

type CalendarEvent struct {
	UID     string `json:"uid"`
	Title   string `json:"title"`
	Date    string `json:"date"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Notes   string `json:"notes"`
}

type AICalendarService struct {
	AI *AIService
}

func (s *AICalendarService) GenerateCalendarSync() ([]CalendarEvent, error) {
	now := time.Now()
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta != nil {
		now = now.In(jakarta)
	}

	events := []CalendarEvent{
		{
			UID:   "dividend-bbca-q4-" + now.Format("2006"),
			Title: "Dividen BBCA (estimasi Q4)",
			Date:  time.Date(now.Year(), 12, 15, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
			Type:  "dividend",
			Code:  "BBCA",
			Notes: "Dividen interim Q4 BBCA. Estimasi berdasarkan data historis.",
		},
		{
			UID:   "dividend-bbri-q3-" + now.Format("2006"),
			Title: "Dividen BBRI (estimasi Q3)",
			Date:  time.Date(now.Year(), 9, 20, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
			Type:  "dividend",
			Code:  "BBRI",
			Notes: "Dividen interim Q3 BBRI. Estimasi berdasarkan data historis.",
		},
		{
			UID:   "dividend-tlkm-" + now.Format("2006"),
			Title: "Dividen TLKM (year-end)",
			Date:  time.Date(now.Year(), 5, 25, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
			Type:  "dividend",
			Code:  "TLKM",
			Notes: "Dividen final TLKM. Estimasi berdasarkan data historis.",
		},
		{
			UID:   "ipo-watch-" + now.Format("2006"),
			Title: "IPO Pipeline - Watchlist",
			Date:  now.AddDate(0, 0, 14).Format("2006-01-02"),
			Type:  "ipo",
			Code:  "",
			Notes: "Pantau jadwal IPO mendatang di BEI.",
		},
		{
			UID:   "economic-gdp-" + now.Format("2006"),
			Title: "Rilis GDP Indonesia Q" + fmt.Sprintf("%d", (int(now.Month())-1)/3+1),
			Date:  now.Format("2006-01-02"),
			Type:  "economic",
			Code:  "",
			Notes: "Data pertumbuhan ekonomi Indonesia. Perkiraan BPS.",
		},
		{
			UID:   "market-close-idul-fitri-" + now.Format("2006"),
			Title: "Libur Bursa - Idul Fitri (estimasi)",
			Date:  time.Date(now.Year(), 4, 5, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
			Type:  "holiday",
			Code:  "",
			Notes: "Bursa Efek Indonesia tutup. Estimasi tanggal libur nasional.",
		},
	}

	return events, nil
}

func (s *AICalendarService) GenerateICS(events []CalendarEvent) string {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Investo//Calendar Sync//EN\r\n")
	sb.WriteString("X-WR-CALNAME:Investo - Market Events\r\n")
	sb.WriteString("X-WR-CALDESC:Kalender pasar saham & dividen dari Investo\r\n")
	sb.WriteString("REFRESH-INTERVAL;VALUE=DURATION:PT12H\r\n")

	for _, e := range events {
		dateFormatted := strings.ReplaceAll(e.Date, "-", "")
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s\r\n", e.UID))
		sb.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", dateFormatted))
		sb.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", dateFormatted))

		summary := e.Title
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", summary))
		if e.Notes != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", e.Notes))
		}

		categories := e.Type
		sb.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", categories))
		sb.WriteString("TRANSP:TRANSPARENT\r\n")
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")
	return sb.String()
}

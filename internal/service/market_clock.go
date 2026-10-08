package service

import (
	"time"
)

type MarketStatus struct {
	Status      string `json:"status"`
	NextOpen    string `json:"next_open"`
	NextClose   string `json:"next_close"`
	CurrentTime string `json:"current_time"`
}

type MarketClock struct {
	loc *time.Location
}

func NewMarketClock() *MarketClock {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return &MarketClock{loc: loc}
}

func (mc *MarketClock) GetMarketStatus() MarketStatus {
	now := time.Now().In(mc.loc)
	weekday := now.Weekday()

	if weekday == time.Saturday || weekday == time.Sunday {
		return MarketStatus{
			Status:      "weekend",
			NextOpen:    mc.nextMondayOpen(now).Format("2006-01-02 09:00:00"),
			NextClose:   mc.nextMondayClose(now).Format("2006-01-02 16:00:00"),
			CurrentTime: now.Format("2006-01-02 15:04:05"),
		}
	}

	preOpen := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, mc.loc)
	marketOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, mc.loc)
	lunchStart := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, mc.loc)
	lunchEnd := time.Date(now.Year(), now.Month(), now.Day(), 13, 30, 0, 0, mc.loc)
	marketClose := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, mc.loc)

	ms := MarketStatus{
		CurrentTime: now.Format("2006-01-02 15:04:05"),
	}

	if now.Before(preOpen) {
		ms.Status = "pre_open"
		ms.NextOpen = preOpen.Format("2006-01-02 15:04:05")
		ms.NextClose = marketClose.Format("2006-01-02 15:04:05")
	} else if now.Before(marketOpen) {
		ms.Status = "pre_open"
		ms.NextOpen = marketOpen.Format("2006-01-02 15:04:05")
		ms.NextClose = marketClose.Format("2006-01-02 15:04:05")
	} else if now.Before(lunchStart) {
		ms.Status = "trading"
		ms.NextOpen = now.Format("2006-01-02 15:04:05")
		ms.NextClose = lunchStart.Format("2006-01-02 15:04:05")
	} else if now.Before(lunchEnd) {
		ms.Status = "lunch_break"
		ms.NextOpen = lunchEnd.Format("2006-01-02 15:04:05")
		ms.NextClose = marketClose.Format("2006-01-02 15:04:05")
	} else if now.Before(marketClose) {
		ms.Status = "trading"
		ms.NextOpen = now.Format("2006-01-02 15:04:05")
		ms.NextClose = marketClose.Format("2006-01-02 15:04:05")
	} else {
		ms.Status = "post_close"
		nextDay := mc.nextTradingDay(marketClose)
		ms.NextOpen = nextDay.Format("2006-01-02 15:04:05")
		nextDayClose := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 16, 0, 0, 0, mc.loc)
		ms.NextClose = nextDayClose.Format("2006-01-02 15:04:05")
	}

	return ms
}

func (mc *MarketClock) nextTradingDay(from time.Time) time.Time {
	next := time.Date(from.Year(), from.Month(), from.Day(), 9, 0, 0, 0, mc.loc).AddDate(0, 0, 1)
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (mc *MarketClock) nextMondayOpen(from time.Time) time.Time {
	for from.Weekday() != time.Monday {
		from = from.AddDate(0, 0, 1)
	}
	return time.Date(from.Year(), from.Month(), from.Day(), 9, 0, 0, 0, mc.loc)
}

func (mc *MarketClock) nextMondayClose(from time.Time) time.Time {
	for from.Weekday() != time.Monday {
		from = from.AddDate(0, 0, 1)
	}
	return time.Date(from.Year(), from.Month(), from.Day(), 16, 0, 0, 0, mc.loc)
}

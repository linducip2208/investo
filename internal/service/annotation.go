package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type ChartAnnotation struct {
	Type  string  `json:"type"`
	Label string  `json:"label"`
	Price float64 `json:"price"`
	Date  string  `json:"date"`
	Color string  `json:"color"`
}

type AnnotationService struct {
	StockActionRepo *repository.StockActionRepository
}

func (s *AnnotationService) GenerateAutoAnnotations(code string, prices []model.StockPrice) []ChartAnnotation {
	if len(prices) == 0 {
		return nil
	}

	var annotations []ChartAnnotation
	n := len(prices)

	var week52High, week52Low float64
	var allTimeHigh, allTimeLow float64
	week52StartIdx := 0
	if n > 252 {
		week52StartIdx = n - 252
	}

	allTimeHigh = prices[0].High
	allTimeLow = prices[0].Low
	allTimeHighDate := prices[0].Date
	allTimeLowDate := prices[0].Date

	week52High = prices[week52StartIdx].High
	week52Low = prices[week52StartIdx].Low
	week52HighDate := prices[week52StartIdx].Date
	week52LowDate := prices[week52StartIdx].Date

	var highVolDays []struct {
		date   time.Time
		volume int64
		price  float64
	}

	for i, p := range prices {
		if p.High > allTimeHigh {
			allTimeHigh = p.High
			allTimeHighDate = p.Date
		}
		if p.Low < allTimeLow {
			allTimeLow = p.Low
			allTimeLowDate = p.Date
		}

		if i >= week52StartIdx {
			if p.High > week52High {
				week52High = p.High
				week52HighDate = p.Date
			}
			if p.Low < week52Low {
				week52Low = p.Low
				week52LowDate = p.Date
			}
		}

		highVolDays = append(highVolDays, struct {
			date   time.Time
			volume int64
			price  float64
		}{p.Date, p.Volume, p.Close})

		if len(highVolDays) > 3 {
			sort.Slice(highVolDays, func(i, j int) bool {
				return highVolDays[i].volume > highVolDays[j].volume
			})
			highVolDays = highVolDays[:3]
		}
	}

	sort.Slice(highVolDays, func(i, j int) bool {
		return highVolDays[i].volume > highVolDays[j].volume
	})

	if week52High > 0 {
		annotations = append(annotations, ChartAnnotation{
			Type:  "52w_high",
			Label: "52-Wk High",
			Price: math.Round(week52High*100) / 100,
			Date:  week52HighDate.Format("2006-01-02"),
			Color: "#ef4444",
		})
	}
	if week52Low > 0 {
		annotations = append(annotations, ChartAnnotation{
			Type:  "52w_low",
			Label: "52-Wk Low",
			Price: math.Round(week52Low*100) / 100,
			Date:  week52LowDate.Format("2006-01-02"),
			Color: "#10b981",
		})
	}
	if allTimeHigh > 0 {
		annotations = append(annotations, ChartAnnotation{
			Type:  "ath",
			Label: "All-Time High",
			Price: math.Round(allTimeHigh*100) / 100,
			Date:  allTimeHighDate.Format("2006-01-02"),
			Color: "#dc2626",
		})
	}
	if allTimeLow > 0 {
		annotations = append(annotations, ChartAnnotation{
			Type:  "atl",
			Label: "All-Time Low",
			Price: math.Round(allTimeLow*100) / 100,
			Date:  allTimeLowDate.Format("2006-01-02"),
			Color: "#059669",
		})
	}

	window := 20
	for i := window; i < n-window; i++ {
		if prices[i].Close > prices[i-1].Close {
			isBreakout := true
			highestBefore := prices[i-window].High
			for j := i - window; j < i; j++ {
				if prices[j].High > highestBefore {
					highestBefore = prices[j].High
				}
			}
			if prices[i].Close > highestBefore && prices[i-1].Close <= highestBefore {
				_ = isBreakout
				annotations = append(annotations, ChartAnnotation{
					Type:  "breakout",
					Label: "Breakout",
					Price: math.Round(prices[i].Close*100) / 100,
					Date:  prices[i].Date.Format("2006-01-02"),
					Color: "#3b82f6",
				})
			}
		}

		if prices[i].Close < prices[i-1].Close {
			lowestBefore := prices[i-window].Low
			for j := i - window; j < i; j++ {
				if prices[j].Low < lowestBefore {
					lowestBefore = prices[j].Low
				}
			}
			if prices[i].Close < lowestBefore && prices[i-1].Close >= lowestBefore {
				annotations = append(annotations, ChartAnnotation{
					Type:  "breakdown",
					Label: "Breakdown",
					Price: math.Round(prices[i].Close*100) / 100,
					Date:  prices[i].Date.Format("2006-01-02"),
					Color: "#f97316",
				})
			}
		}
	}

	for _, hvd := range highVolDays {
		if hvd.volume > 0 {
			annotations = append(annotations, ChartAnnotation{
				Type:  "high_volume",
				Label: fmt.Sprintf("High Vol (%d)", hvd.volume),
				Price: math.Round(hvd.price*100) / 100,
				Date:  hvd.date.Format("2006-01-02"),
				Color: "#8b5cf6",
			})
		}
	}

	return annotations
}

func (s *AnnotationService) GetDividendAnnotations(stockID int64) []ChartAnnotation {
	if s.StockActionRepo == nil {
		return nil
	}

	actions, err := s.StockActionRepo.FindByStockID(stockID)
	if err != nil || len(actions) == 0 {
		return nil
	}

	var annotations []ChartAnnotation
	for _, a := range actions {
		if a.ActionType == "dividend" {
			annotations = append(annotations, ChartAnnotation{
				Type:  "dividend",
				Label: fmt.Sprintf("Div: %.0f", a.Price),
				Price: a.Price,
				Date:  a.ExDate.Format("2006-01-02"),
				Color: "#06b6d4",
			})
		}
		if a.ActionType == "split" {
			annotations = append(annotations, ChartAnnotation{
				Type:  "split",
				Label: fmt.Sprintf("Split: %.1f:1", a.Ratio),
				Price: a.Price,
				Date:  a.ExDate.Format("2006-01-02"),
				Color: "#a855f7",
			})
		}
	}

	return annotations
}

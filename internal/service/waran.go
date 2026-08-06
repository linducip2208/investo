package service

import (
	"math"
	"time"
)

type WaranData struct {
	StockCode         string  `json:"stock_code"`
	WaranCode         string  `json:"waran_code"`
	ExercisePrice     float64 `json:"exercise_price"`
	CurrentStockPrice float64 `json:"current_stock_price"`
	WaranPrice        float64 `json:"waran_price"`
	ExpiryDate        string  `json:"expiry_date"`
	Premium           float64 `json:"premium"`
	Gearing           float64 `json:"gearing"`
	BreakEven         float64 `json:"break_even"`
	Status            string  `json:"status"`
	DaysToExpiry      int     `json:"days_to_expiry"`
}

type WaranService struct{}

func (w *WaranService) GetAllWaran() []WaranData {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	warans := []struct {
		stockCode     string
		waranCode     string
		exercisePrice float64
		stockPrice    float64
		waranPrice    float64
		expiryDate    string
	}{
		{"BUKA", "BUKA-W", 320, 144, 8, "2027-06-30"},
		{"ARTO", "ARTO-W", 1250, 2320, 1080, "2026-12-15"},
		{"MEDC", "MEDC-W", 1500, 1265, 10, "2027-03-20"},
		{"ISAT", "ISAT-W", 1000, 2690, 1700, "2027-01-10"},
		{"SMMA", "SMMA-W", 100, 136, 38, "2026-09-30"},
		{"AISA", "AISA-W", 1200, 1650, 460, "2027-05-15"},
		{"GIAA", "GIAA-W", 60, 72, 18, "2027-11-05"},
		{"FREN", "FREN-W", 60, 42, 3, "2026-08-25"},
		{"POWR", "POWR-W", 300, 254, 8, "2027-02-14"},
		{"SMDR", "SMDR-W", 280, 338, 72, "2027-07-01"},
		{"BLUE", "BLUE-W", 4000, 4320, 520, "2026-10-08"},
		{"WOOD", "WOOD-W", 280, 156, 3, "2027-04-22"},
		{"INDY", "INDY-W", 1200, 1480, 310, "2026-11-30"},
		{"BIPI", "BIPI-W", 80, 56, 2, "2027-08-18"},
		{"SUPR", "SUPR-W", 1000, 2520, 1540, "2027-09-12"},
	}

	var result []WaranData
	for _, wr := range warans {
		expiry, _ := time.Parse("2006-01-02", wr.expiryDate)
		daysToExpiry := int(expiry.Sub(now).Hours() / 24)
		if daysToExpiry < 0 {
			daysToExpiry = 0
		}

		breakEven := wr.waranPrice + wr.exercisePrice
		premium := 0.0
		if wr.stockPrice > 0 {
			premium = ((breakEven - wr.stockPrice) / wr.stockPrice) * 100
		}

		gearing := 0.0
		if wr.waranPrice > 0 {
			gearing = wr.stockPrice / wr.waranPrice
		}

		status := "Out of the Money"
		if wr.stockPrice > wr.exercisePrice {
			status = "In the Money"
		}

		result = append(result, WaranData{
			StockCode:         wr.stockCode,
			WaranCode:         wr.waranCode,
			ExercisePrice:     wr.exercisePrice,
			CurrentStockPrice: wr.stockPrice,
			WaranPrice:        wr.waranPrice,
			ExpiryDate:        wr.expiryDate,
			Premium:           math.Round(premium*100) / 100,
			Gearing:           math.Round(gearing*100) / 100,
			BreakEven:         breakEven,
			Status:            status,
			DaysToExpiry:      daysToExpiry,
		})
	}

	return result
}

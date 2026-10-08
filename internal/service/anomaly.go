package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/repository"
)

type AnomalyService struct {
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
}

type AnomalyResult struct {
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
}

func (s *AnomalyService) DetectPriceAnomalies() ([]AnomalyResult, error) {
	pricesWithPrev, err := s.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, fmt.Errorf("DetectPriceAnomalies: %w", err)
	}

	if len(pricesWithPrev) == 0 {
		return nil, nil
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -20)

	type stockData struct {
		stockID      int64
		latestClose  float64
		latestVolume int64
		prevClose    float64
	}
	var allStocks []stockData
	stockIDSet := make(map[int64]bool)

	for _, p := range pricesWithPrev {
		allStocks = append(allStocks, stockData{
			stockID:      p.StockID,
			latestClose:  p.LatestClose,
			latestVolume: p.LatestVolume,
			prevClose:    p.PrevClose,
		})
		stockIDSet[p.StockID] = true
	}

	stockIDs := make([]int64, 0, len(stockIDSet))
	for id := range stockIDSet {
		stockIDs = append(stockIDs, id)
	}

	historicalPrices, err := s.StockPriceRepo.FindByDateRange(stockIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("DetectPriceAnomalies historical: %w", err)
	}

	priceHistory := make(map[int64][]float64)
	volumeHistory := make(map[int64][]int64)
	for _, p := range historicalPrices {
		priceHistory[p.StockID] = append(priceHistory[p.StockID], p.Close)
		volumeHistory[p.StockID] = append(volumeHistory[p.StockID], p.Volume)
	}

	var results []AnomalyResult

	for _, sd := range allStocks {
		if len(priceHistory[sd.stockID]) < 10 {
			continue
		}

		stock, err := s.StockRepo.FindByID(sd.stockID)
		stockCode := ""
		stockName := ""
		if err == nil && stock != nil {
			stockCode = stock.Code
			stockName = stock.Name
		}

		prices := priceHistory[sd.stockID]
		changes := make([]float64, 0, len(prices)-1)
		for i := 1; i < len(prices); i++ {
			if prices[i-1] > 0 {
				change := (prices[i] - prices[i-1]) / prices[i-1] * 100
				changes = append(changes, change)
			}
		}

		if len(changes) < 5 {
			continue
		}

		mean := meanFloat(changes)
		stdDev := stdDevFloat(changes, mean)

		if sd.prevClose > 0 && stdDev > 0 {
			todayChange := (sd.latestClose - sd.prevClose) / sd.prevClose * 100
			sigma := math.Abs(todayChange-mean) / stdDev

			if sigma >= 2 {
				severity := "low"
				description := fmt.Sprintf("Harga %s mengalami perubahan %.2f%% (%.1f\u03c3 dari mean 20-hari).", "", todayChange, sigma)
				if sigma >= 4 {
					severity = "high"
					description = fmt.Sprintf("PERINGATAN: Harga %s melonjak ekstrim %.2f%% (%.1f\u03c3)! Kemungkinan ada berita besar atau manipulasi pasar.", "", todayChange, sigma)
				} else if sigma >= 3 {
					severity = "medium"
					description = fmt.Sprintf("Harga %s bergerak signifikan %.2f%% (%.1f\u03c3). Monitor dengan ketat.", "", todayChange, sigma)
				}

			results = append(results, AnomalyResult{
				StockCode:   stockCode,
				StockName:   stockName,
				Type:        "price",
				Severity:    severity,
				Description: description,
				Value:       todayChange,
				Threshold:   mean + (2 * stdDev),
			})
			}
		}

		volumes := volumeHistory[sd.stockID]
		if len(volumes) >= 5 {
			avgVol := meanInt(volumes[:len(volumes)-1])
			if avgVol > 0 && float64(sd.latestVolume) > float64(avgVol)*3 {
				ratio := float64(sd.latestVolume) / float64(avgVol)
				severity := "low"
				desc := fmt.Sprintf("Volume perdagangan %.0fx dari rata-rata 20-hari.", ratio)
				if ratio >= 5 {
					severity = "high"
					desc = fmt.Sprintf("PERINGATAN: Volume melonjak ekstrim %.0fx! Indikasi akumulasi atau distribusi besar.", ratio)
				} else if ratio >= 4 {
					severity = "medium"
					desc = fmt.Sprintf("Volume naik signifikan %.0fx dari rata-rata. Perhatikan pergerakan selanjutnya.", ratio)
				}

			results = append(results, AnomalyResult{
				StockCode:   stockCode,
				StockName:   stockName,
				Type:        "volume",
				Severity:    severity,
				Description: desc,
				Value:       float64(sd.latestVolume),
				Threshold:   float64(avgVol) * 3,
			})
			}
		}
	}

	return results, nil
}

func (s *AnomalyService) DetectGapAnomalies() ([]AnomalyResult, error) {
	pricesWithPrev, err := s.StockPriceRepo.GetAllLatestPricesWithPrev()
	if err != nil {
		return nil, fmt.Errorf("DetectGapAnomalies: %w", err)
	}

	var results []AnomalyResult

	for _, p := range pricesWithPrev {
		if p.PrevClose <= 0 {
			continue
		}

		stock, err := s.StockRepo.FindByID(p.StockID)
		stockCode := ""
		stockName := ""
		if err == nil && stock != nil {
			stockCode = stock.Code
			stockName = stock.Name
		}

		latestPrices, err := s.StockPriceRepo.FindLatest(p.StockID, 1)
		if err != nil || len(latestPrices) == 0 {
			continue
		}

		todayOpen := latestPrices[0].Open
		if todayOpen <= 0 {
			continue
		}

		gap := (todayOpen - p.PrevClose) / p.PrevClose * 100

		if math.Abs(gap) > 5 {
			direction := "naik"
			severity := "low"
			if gap < 0 {
				direction = "turun"
			}

			absGap := math.Abs(gap)
			if absGap > 10 {
				severity = "high"
			} else if absGap > 7 {
				severity = "medium"
			}

		results = append(results, AnomalyResult{
			StockCode:   stockCode,
			StockName:   stockName,
			Type:        "gap",
			Severity:    severity,
			Description: fmt.Sprintf("Gap %s %.2f%% dari close kemarin (%.2f) ke open hari ini (%.2f).", direction, gap, p.PrevClose, todayOpen),
			Value:       gap,
			Threshold:   5.0,
		})
		}
	}

	return results, nil
}

func meanFloat(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func stdDevFloat(data []float64, mean float64) float64 {
	if len(data) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range data {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(data)-1))
}

func meanInt(data []int64) int64 {
	if len(data) == 0 {
		return 0
	}
	sum := int64(0)
	for _, v := range data {
		sum += v
	}
	return sum / int64(len(data))
}

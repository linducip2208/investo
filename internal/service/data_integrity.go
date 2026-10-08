package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type IntegrityIssue struct {
	StockCode string  `json:"stock_code"`
	StockName string  `json:"stock_name"`
	Issue     string  `json:"issue"`
	Severity  string  `json:"severity"`
	Value     float64 `json:"value"`
}

type DataIntegrityService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (s *DataIntegrityService) RunCheck() ([]IntegrityIssue, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("DataIntegrityService.RunCheck: %w", err)
	}

	now := time.Now()
	var issues []IntegrityIssue

	for _, stock := range stocks {
		prices, err := s.StockPriceRepo.FindLatest(stock.ID, 200)
		if err != nil || len(prices) == 0 {
			issues = append(issues, IntegrityIssue{
				StockCode: stock.Code,
				StockName: stock.Name,
				Issue:     "Tidak ada data harga",
				Severity:  "critical",
				Value:     0,
			})
			continue
		}

		issues = append(issues, s.checkPriceGaps(stock, prices)...)

		issues = append(issues, s.checkPriceSpikes(stock, prices)...)

		issues = append(issues, s.checkZeroNegative(stock, prices)...)

		issues = append(issues, s.checkStaleData(stock, prices, now)...)

		issues = append(issues, s.checkSuspendedVolume(stock, prices)...)
	}

	return issues, nil
}

func (s *DataIntegrityService) checkPriceGaps(stock model.Stock, prices []model.StockPrice) []IntegrityIssue {
	var issues []IntegrityIssue
	for i := 0; i < len(prices)-1; i++ {
		gap := prices[i].Date.Sub(prices[i+1].Date)
		tradingDays := int(gap.Hours() / 24)
		if tradingDays > 5 {
			issues = append(issues, IntegrityIssue{
				StockCode: stock.Code,
				StockName: stock.Name,
				Issue:     fmt.Sprintf("Gap data harga: %d hari tanpa data antara %s dan %s", tradingDays, prices[i+1].Date.Format("2006-01-02"), prices[i].Date.Format("2006-01-02")),
				Severity:  "warning",
				Value:     float64(tradingDays),
			})
		}
	}
	return issues
}

func (s *DataIntegrityService) checkPriceSpikes(stock model.Stock, prices []model.StockPrice) []IntegrityIssue {
	var issues []IntegrityIssue
	for i := 0; i < len(prices)-1; i++ {
		if prices[i+1].Close == 0 {
			continue
		}
		change := math.Abs((prices[i].Close-prices[i+1].Close)/prices[i+1].Close) * 100
		if change > 30 {
			issues = append(issues, IntegrityIssue{
				StockCode: stock.Code,
				StockName: stock.Name,
				Issue:     fmt.Sprintf("Lonjakan harga %.1f%% pada %s: %.2f -> %.2f", change, prices[i].Date.Format("2006-01-02"), prices[i+1].Close, prices[i].Close),
				Severity:  "warning",
				Value:     change,
			})
		}
	}
	return issues
}

func (s *DataIntegrityService) checkZeroNegative(stock model.Stock, prices []model.StockPrice) []IntegrityIssue {
	var issues []IntegrityIssue
	for _, p := range prices {
		if p.Close <= 0 || p.Open <= 0 || p.High <= 0 || p.Low <= 0 {
			issues = append(issues, IntegrityIssue{
				StockCode: stock.Code,
				StockName: stock.Name,
				Issue:     fmt.Sprintf("Harga nol/negatif pada %s: O=%.2f H=%.2f L=%.2f C=%.2f", p.Date.Format("2006-01-02"), p.Open, p.High, p.Low, p.Close),
				Severity:  "critical",
				Value:     p.Close,
			})
		}
	}
	return issues
}

func (s *DataIntegrityService) checkStaleData(stock model.Stock, prices []model.StockPrice, now time.Time) []IntegrityIssue {
	var issues []IntegrityIssue
	if len(prices) > 0 {
		latestDate := prices[0].Date
		staleDays := int(now.Sub(latestDate).Hours() / 24)
		if staleDays > 7 {
			issues = append(issues, IntegrityIssue{
				StockCode: stock.Code,
				StockName: stock.Name,
				Issue:     fmt.Sprintf("Data kadaluarsa: update terakhir %s (%d hari lalu)", latestDate.Format("2006-01-02"), staleDays),
				Severity:  "warning",
				Value:     float64(staleDays),
			})
		}
	}
	return issues
}

func (s *DataIntegrityService) checkSuspendedVolume(stock model.Stock, prices []model.StockPrice) []IntegrityIssue {
	var issues []IntegrityIssue
	consecutiveZero := 0
	for _, p := range prices {
		if p.Volume == 0 {
			consecutiveZero++
		} else {
			consecutiveZero = 0
		}
		if consecutiveZero > 10 {
			issues = append(issues, IntegrityIssue{
				StockCode: stock.Code,
				StockName: stock.Name,
				Issue:     fmt.Sprintf("Volume 0 berturut-turut >10 hari (terakhir %s): kemungkinan delisting/suspensi", p.Date.Format("2006-01-02")),
				Severity:  "warning",
				Value:     float64(consecutiveZero),
			})
			break
		}
	}
	return issues
}

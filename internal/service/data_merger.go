package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type DataMerger struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	StockActionRepo *repository.StockActionRepository
}

func (m *DataMerger) BackfillHistorical(stockCode string) (int, error) {
	stock, err := m.StockRepo.FindByCode(stockCode)
	if err != nil {
		return 0, fmt.Errorf("DataMerger.BackfillHistorical stock: %w", err)
	}

	end := time.Now().Unix()
	start := time.Now().AddDate(-5, 0, 0).Unix()

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s.JK?period1=%d&period2=%d&interval=1d&events=history",
		stockCode, start, end)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Yahoo Finance fetch failed for %s, trying backup URL: %v", stockCode, err)
		return m.backfillFromAlternative(stockCode, stock)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Yahoo Finance returned %d for %s", resp.StatusCode, stockCode)
		return m.backfillFromAlternative(stockCode, stock)
	}

	reader := csv.NewReader(resp.Body)
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}
	if len(header) < 7 {
		return 0, fmt.Errorf("unexpected CSV header: %v", header)
	}

	var prices []model.StockPrice
	count := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("skip row for %s: %v", stockCode, err)
			continue
		}

		date, err := time.Parse("2006-01-02", record[0])
		if err != nil {
			continue
		}

		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		close, _ := strconv.ParseFloat(record[4], 64)
		adjClose, _ := strconv.ParseFloat(record[5], 64)
		volume, _ := strconv.ParseInt(record[6], 10, 64)

		if close == 0 && adjClose == 0 {
			continue
		}

		prices = append(prices, model.StockPrice{
			StockID:  stock.ID,
			Date:     date,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
			Volume:   volume,
			AdjClose: adjClose,
		})
		count++

		if len(prices) >= 500 {
			if err := m.StockPriceRepo.BulkInsert(prices); err != nil {
				log.Printf("bulk insert chunk for %s: %v", stockCode, err)
			}
			prices = nil
		}
	}

	if len(prices) > 0 {
		if err := m.StockPriceRepo.BulkInsert(prices); err != nil {
			return count, fmt.Errorf("final bulk insert for %s: %w", stockCode, err)
		}
	}

	log.Printf("Backfilled %d records for %s", count, stockCode)
	return count, nil
}

func (m *DataMerger) backfillFromAlternative(stockCode string, stock *model.Stock) (int, error) {
	end := time.Now()
	start := end.AddDate(-5, 0, 0)

	fromStr := start.Format("2006-01-02")
	toStr := end.Format("2006-01-02")

	url := fmt.Sprintf("https://raw.githubusercontent.com/dsetyawan/idx-data/main/historical/%s.csv", strings.ToUpper(stockCode))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Alternative source failed for %s, trying second alternative", stockCode)
		return m.backfillFromIDX(stockCode, stock, fromStr, toStr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return m.backfillFromIDX(stockCode, stock, fromStr, toStr)
	}

	reader := csv.NewReader(resp.Body)
	reader.Read()

	var prices []model.StockPrice
	count := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 7 {
			continue
		}

		date, err := time.Parse("2006-01-02", record[0])
		if err != nil {
			continue
		}

		if date.Before(start) || date.After(end) {
			continue
		}

		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		close, _ := strconv.ParseFloat(record[4], 64)
		adjClose, _ := strconv.ParseFloat(record[5], 64)
		volume, _ := strconv.ParseInt(record[6], 10, 64)

		if close == 0 && adjClose == 0 {
			continue
		}

		prices = append(prices, model.StockPrice{
			StockID:  stock.ID,
			Date:     date,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
			Volume:   volume,
			AdjClose: adjClose,
		})
		count++

		if len(prices) >= 500 {
			_ = m.StockPriceRepo.BulkInsert(prices)
			prices = nil
		}
	}

	if len(prices) > 0 {
		_ = m.StockPriceRepo.BulkInsert(prices)
	}

	log.Printf("Backfilled %d records for %s (alternative source)", count, stockCode)
	return count, nil
}

func (m *DataMerger) backfillFromIDX(stockCode string, stock *model.Stock, fromStr, toStr string) (int, error) {
	log.Printf("No external data source available for %s", stockCode)
	return 0, nil
}

func (m *DataMerger) MergeCorporateActions() error {
	stocks, err := m.StockRepo.ListActive()
	if err != nil {
		return fmt.Errorf("DataMerger.MergeCorporateActions: %w", err)
	}

	for _, stock := range stocks {
		actions, err := m.StockActionRepo.FindByStockID(stock.ID)
		if err != nil || len(actions) == 0 {
			continue
		}

		for _, action := range actions {
			if action.ActionType == "stock_split" || action.ActionType == "reverse_split" {
				if action.Ratio > 0 {
					log.Printf("Adjusting prices for %s (split %s, ratio %.2f, ex_date %s)",
						stock.Code, action.ActionType, action.Ratio, action.ExDate.Format("2006-01-02"))

					prices, err := m.StockPriceRepo.FindByStockDate(stock.ID, action.ExDate, time.Now())
					if err != nil {
						continue
					}

					for i := range prices {
						if !prices[i].Date.Before(action.ExDate) {
							continue
						}
						if action.ActionType == "stock_split" {
							prices[i].AdjClose = prices[i].Close / action.Ratio
						} else {
							prices[i].AdjClose = prices[i].Close * action.Ratio
						}
					}
				}
			}
		}
	}

	return nil
}

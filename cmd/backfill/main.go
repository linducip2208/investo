package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"investo/internal/config"
	"investo/internal/database"
	"investo/internal/repository"
	"investo/internal/service/scraper"
)

func main() {
	log.SetFlags(log.Ltime)

	days := flag.Int("days", 90, "Number of days to backfill (default 90 = 3 months)")
	force := flag.Bool("force", false, "Re-fetch even if data exists")
	flag.Parse()

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer db.Close()

	stockRepo := &repository.StockRepository{DB: db}
	priceRepo := &repository.StockPriceRepository{DB: db}

	stocks, err := stockRepo.ListActive()
	if err != nil {
		log.Fatalf("List stocks: %v", err)
	}

	totalStocks := len(stocks)
	log.Printf("Backfilling %d days for %d stocks...", *days, totalStocks)

	yahoo := &scraper.YahooScraper{}

	end := time.Now()
	start := end.AddDate(0, 0, -(*days))

	total := 0
	skipped := 0
	failed := 0

	for i, s := range stocks {
		pct := float64(i+1) / float64(totalStocks) * 100

		if !*force {
			existing, _ := priceRepo.CountByStockID(s.ID)
			if existing > 0 {
				skipped++
				fmt.Printf("\r  [%.1f%%] %d/%d skip=%d err=%d", pct, i+1, totalStocks, skipped, failed)
				continue
			}
		}

		prices, err := yahoo.FetchHistorical(s.Code+".JK", start, end)
		if err != nil {
			failed++
			fmt.Printf("\r  [%.1f%%] %d/%d skip=%d err=%d | %s: %v", pct, i+1, totalStocks, skipped, failed, s.Code, err)
			continue
		}
		if len(prices) == 0 {
			failed++
			fmt.Printf("\r  [%.1f%%] %d/%d skip=%d err=%d", pct, i+1, totalStocks, skipped, failed)
			continue
		}

		for j := range prices {
			prices[j].StockID = s.ID
		}

		if err := priceRepo.BulkInsert(prices); err != nil {
			failed++
			fmt.Printf("\r  [%.1f%%] %d/%d skip=%d err=%d | %s: insert error %v", pct, i+1, totalStocks, skipped, failed, s.Code, err)
			continue
		}

		total += len(prices)
		fmt.Printf("\r  [%.1f%%] %d/%d | %s: %d points", pct, i+1, totalStocks, s.Code, len(prices))
		time.Sleep(150 * time.Millisecond)
	}

	fmt.Println()
	log.Printf("Done! %d total price points for %d stocks (skipped: %d, failed: %d)", total, totalStocks, skipped, failed)
}

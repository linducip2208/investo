package main

import (
	"log"
	"sync"
	"time"

	"investo/internal/config"
	"investo/internal/database"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service/scraper"
)

func main() {
	log.SetFlags(log.Ltime)
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer db.Close()

	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta == nil {
		jakarta = time.FixedZone("WIB", 7*3600)
	}

	priceRepo := &repository.StockPriceRepository{DB: db}
	stockRepo := &repository.StockRepository{DB: db}
	yahoo := &scraper.YahooScraper{}

	end := time.Now().In(jakarta)
	start := end.AddDate(0, 0, -5)

	stocks, err := stockRepo.ListActive()
	if err != nil {
		log.Fatalf("List stocks: %v", err)
	}

	log.Printf("Cron fetch: %d stocks, %s ~ %s", len(stocks), start.Format("2006-01-02"), end.Format("2006-01-02"))

	var mu sync.Mutex
	var fetched, errors int
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, s := range stocks {
		wg.Add(1)
		go func(stock model.Stock) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prices, err := yahoo.FetchHistorical(stock.Code+".JK", start, end)
			if err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
				return
			}
			if len(prices) > 0 {
				for j := range prices {
					prices[j].StockID = stock.ID
				}
				if err := priceRepo.BulkInsert(prices); err != nil {
					mu.Lock()
					errors++
					mu.Unlock()
					return
				}
				mu.Lock()
				fetched++
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()

	log.Printf("Done: %d updated, %d errors", fetched, errors)
}
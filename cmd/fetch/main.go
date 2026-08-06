package main

import (
	"log"
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
		log.Fatalf("DB connect: %v", err)
	}
	defer db.Close()

	database.RunMigrations(db, "internal/database/migrations")

	stockRepo := &repository.StockRepository{DB: db}
	priceRepo := &repository.StockPriceRepository{DB: db}
	forexRepo := &repository.ForexRepository{DB: db}

	idx := &scraper.IDXScraper{}
	yahoo := &scraper.YahooScraper{}

	stocks, _, err := idx.FetchStockList()
	if err != nil {
		log.Fatalf("Fetch stock list: %v", err)
	}

	log.Printf("Seeding %d stocks...", len(stocks))
	for _, s := range stocks {
		existing, _ := stockRepo.FindByCode(s.Code)
		if existing != nil {
			log.Printf("  %s already exists, skip", s.Code)
			continue
		}
		s.ListingDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		s.SharesOutstanding = 10_000_000_000
		if _, err := stockRepo.Create(&s); err != nil {
			log.Printf("  %s FAIL: %v", s.Code, err)
			continue
		}
		log.Printf("  %s OK", s.Code)
	}

	log.Println("Fetching prices from Yahoo Finance (delayed)...")
	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	allStocks, _ := stockRepo.ListActive()
	for i, s := range allStocks {
		log.Printf("  [%d/%d] %s...", i+1, len(allStocks), s.Code)
		prices, err := yahoo.FetchHistorical(s.Code+".JK", start, end)
		if err != nil {
			log.Printf("    skip: %v", err)
			continue
		}
		if len(prices) > 0 {
			for j := range prices {
				prices[j].StockID = s.ID
			}
			if err := priceRepo.BulkInsert(prices); err != nil {
				log.Printf("    bulk insert: %v", err)
			} else {
				log.Printf("    %d prices saved", len(prices))
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	log.Println("Fetching forex rates...")
	pairs, _ := forexRepo.FindAllPairs()
	for _, pair := range pairs {
		log.Printf("  %s/%s...", pair.BaseCurrency, pair.QuoteCurrency)
		open, high, low, close, err := yahoo.FetchForexRate(pair.BaseCurrency, pair.QuoteCurrency)
		if err != nil {
			log.Printf("    skip: %v", err)
			continue
		}
		rate := model.ForexRate{
			PairID: pair.ID,
			Date:   time.Now(),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
		}
		if err := forexRepo.BulkInsertRates([]model.ForexRate{rate}); err != nil {
			log.Printf("    insert: %v", err)
		} else {
			log.Printf("    rate=%.4f saved", close)
		}
		time.Sleep(200 * time.Millisecond)
	}

	log.Println("Data fetch complete!")
}

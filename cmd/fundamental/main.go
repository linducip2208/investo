package main

import (
	"log"
	"time"

	"investo/internal/config"
	"investo/internal/database"
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

	stockRepo := &repository.StockRepository{DB: db}
	fundRepo := &repository.StockFundamentalRepository{DB: db}

	stocks, _ := stockRepo.ListActive()
	idx := &scraper.IDXScraper{}

	log.Printf("Generating fundamentals for %d stocks...", len(stocks))
	for i, s := range stocks {
		existing, _ := fundRepo.FindLatest(s.ID)
		if existing != nil && existing.ID > 0 {
			continue
		}
		fund, err := idx.FetchFundamentals(s.Code)
		if err != nil {
			log.Printf("  [%d/%d] %s: skip (%v)", i+1, len(stocks), s.Code, err)
			continue
		}
		fund.StockID = s.ID
		if err := fundRepo.Upsert(fund); err != nil {
			log.Printf("  [%d/%d] %s: upsert error %v", i+1, len(stocks), s.Code, err)
		} else {
			log.Printf("  [%d/%d] %s: PER=%.1f PBV=%.1f ROE=%.1f%%", i+1, len(stocks), s.Code, fund.PER, fund.PBV, fund.ROE)
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Done!")
}

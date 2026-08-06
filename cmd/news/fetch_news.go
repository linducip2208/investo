package main

import (
	"log"
	"strings"
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
		log.Fatalf("DB connect: %v", err)
	}
	defer db.Close()

	database.RunMigrations(db, "internal/database/migrations")

	newsScraper := &scraper.NewsScraper{}
	newsRepo := &repository.NewsRepository{DB: db}
	stockRepo := &repository.StockRepository{DB: db}

	log.Println("Fetching latest news from Yahoo Finance...")
	newsList, err := newsScraper.FetchMarketNews(20)
	if err != nil {
		log.Fatalf("Fetch news failed: %v", err)
	}

	log.Printf("Got %d articles", len(newsList))

	stocks, err := stockRepo.ListActive()
	if err != nil {
		log.Fatalf("List stocks: %v", err)
	}

	for i := range newsList {
		newsList[i].PublishedAt = newsList[i].PublishedAt.UTC()
		newsList[i].CreatedAt = time.Now().UTC()
	}

	if err := newsRepo.BulkInsert(newsList); err != nil {
		log.Printf("Bulk insert news: %v", err)
	}

	for _, article := range newsList {
		saved, err := newsRepo.FindBySlug(article.Slug)
		if err != nil || saved == nil {
			continue
		}

		upperTitle := strings.ToUpper(article.Title)
		upperContent := strings.ToUpper(article.Content)

		for _, s := range stocks {
			code := strings.ToUpper(s.Code)
			if strings.Contains(upperTitle, code) || strings.Contains(upperContent, code) {
				if err := newsRepo.LinkStock(saved.ID, s.ID); err != nil {
					log.Printf("  Link %s -> %s: %v", article.Title[:min(50, len(article.Title))], s.Code, err)
				} else {
					log.Printf("  Linked: %s -> %s", article.Title[:min(50, len(article.Title))], s.Code)
				}
			}
		}
	}

	log.Println("News fetch complete!")
}

package model

import "time"

type News struct {
	ID             int64     `json:"id" db:"id"`
	Title          string    `json:"title" db:"title"`
	Slug           string    `json:"slug" db:"slug"`
	Content        string    `json:"content" db:"content"`
	Source         string    `json:"source" db:"source"`
	SourceURL      string    `json:"source_url" db:"source_url"`
	ImageURL       string    `json:"image_url" db:"image_url"`
	PublishedAt    time.Time `json:"published_at" db:"published_at"`
	SentimentScore float64   `json:"sentiment_score" db:"sentiment_score"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type NewsStock struct {
	NewsID  int64 `json:"news_id" db:"news_id"`
	StockID int64 `json:"stock_id" db:"stock_id"`
}

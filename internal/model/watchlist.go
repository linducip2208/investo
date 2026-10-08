package model

import "time"

type Watchlist struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	IsDefault bool      `json:"is_default" db:"is_default"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type WatchlistItem struct {
	ID          int64     `json:"id" db:"id"`
	WatchlistID int64     `json:"watchlist_id" db:"watchlist_id"`
	StockID     int64     `json:"stock_id" db:"stock_id"`
	AddedAt     time.Time `json:"added_at" db:"added_at"`
}

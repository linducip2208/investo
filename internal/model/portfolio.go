package model

import "time"

type Portfolio struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type PortfolioItem struct {
	ID          int64     `json:"id" db:"id"`
	PortfolioID int64     `json:"portfolio_id" db:"portfolio_id"`
	StockID     int64     `json:"stock_id" db:"stock_id"`
	Type        string    `json:"type" db:"type"`
	Quantity    float64   `json:"quantity" db:"quantity"`
	AvgPrice    float64   `json:"avg_price" db:"avg_price"`
	Notes       string    `json:"notes" db:"notes"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

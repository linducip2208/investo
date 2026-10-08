package model

import "time"

type PaperPortfolio struct {
	ID             int64     `json:"id" db:"id"`
	UserID         int64     `json:"user_id" db:"user_id"`
	Name           string    `json:"name" db:"name"`
	InitialBalance float64   `json:"initial_balance" db:"initial_balance"`
	CashBalance    float64   `json:"cash_balance" db:"cash_balance"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type PaperTrade struct {
	ID          int64     `json:"id" db:"id"`
	PortfolioID int64     `json:"portfolio_id" db:"portfolio_id"`
	StockCode   string    `json:"stock_code" db:"stock_code"`
	Type        string    `json:"type" db:"type"`
	Quantity    float64   `json:"quantity" db:"quantity"`
	Price       float64   `json:"price" db:"price"`
	Total       float64   `json:"total" db:"total"`
	Fee         float64   `json:"fee" db:"fee"`
	ExecutedAt  time.Time `json:"executed_at" db:"executed_at"`
}

type PaperHolding struct {
	ID          int64   `json:"id" db:"id"`
	PortfolioID int64   `json:"portfolio_id" db:"portfolio_id"`
	StockCode   string  `json:"stock_code" db:"stock_code"`
	Quantity    float64 `json:"quantity" db:"quantity"`
	AvgPrice    float64 `json:"avg_price" db:"avg_price"`
	StockName   string  `json:"stock_name,omitempty" db:"-"`
	CurrentPrice float64 `json:"current_price,omitempty" db:"-"`
	MarketValue float64  `json:"market_value,omitempty" db:"-"`
	PL          float64  `json:"pl,omitempty" db:"-"`
	PLPercent   float64  `json:"pl_percent,omitempty" db:"-"`
}

type PaperLeader struct {
	UserID       int64   `json:"user_id" db:"user_id"`
	UserName     string  `json:"user_name" db:"user_name"`
	PortfolioID  int64   `json:"portfolio_id" db:"portfolio_id"`
	InitialBalance float64 `json:"initial_balance" db:"initial_balance"`
	CurrentValue float64 `json:"current_value" db:"-"`
	ReturnPct    float64 `json:"return_pct" db:"-"`
}

type Invoice struct {
	Number        string
	Date          string
	CustomerName  string
	CustomerEmail string
	Items         []InvoiceItem
	Total         float64
	Status        string
	Plan          string
}

type InvoiceItem struct {
	Description string
	Qty         int
	Amount      float64
}

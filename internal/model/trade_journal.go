package model

import "time"

type TradeJournal struct {
	ID            int64     `json:"id" db:"id"`
	UserID        int64     `json:"user_id" db:"user_id"`
	StockCode     string    `json:"stock_code" db:"stock_code"`
	EntryDate     time.Time `json:"entry_date" db:"entry_date"`
	ExitDate      *time.Time `json:"exit_date" db:"exit_date"`
	EntryPrice    float64   `json:"entry_price" db:"entry_price"`
	ExitPrice     *float64  `json:"exit_price" db:"exit_price"`
	Quantity      float64   `json:"quantity" db:"quantity"`
	Direction     string    `json:"direction" db:"direction"`
	StrategyUsed  string    `json:"strategy_used" db:"strategy_used"`
	Notes         string    `json:"notes" db:"notes"`
	Emotions      string    `json:"emotions" db:"emotions"`
	Outcome       string    `json:"outcome" db:"outcome"`
	ProfitLoss    *float64  `json:"profit_loss" db:"profit_loss"`
	ProfitLossPct *float64  `json:"profit_loss_pct" db:"profit_loss_pct"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type TradeJournalStats struct {
	TotalTrades int     `json:"total_trades"`
	WinRate     float64 `json:"win_rate"`
	TotalPL     float64 `json:"total_pl"`
	AvgPL       float64 `json:"avg_pl"`
	BestTrade   float64 `json:"best_trade"`
	WorstTrade  float64 `json:"worst_trade"`
}

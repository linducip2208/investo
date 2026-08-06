package model

import "time"

type ForexPair struct {
	ID            int64  `json:"id" db:"id"`
	BaseCurrency  string `json:"base_currency" db:"base_currency"`
	QuoteCurrency string `json:"quote_currency" db:"quote_currency"`
	Name          string `json:"name" db:"name"`
	Group         string `json:"group" db:"group"`
}

type ForexRate struct {
	ID     int64     `json:"id" db:"id"`
	PairID int64     `json:"pair_id" db:"pair_id"`
	Date   time.Time `json:"date" db:"date"`
	Open   float64   `json:"open" db:"open"`
	High   float64   `json:"high" db:"high"`
	Low    float64   `json:"low" db:"low"`
	Close  float64   `json:"close" db:"close"`
}

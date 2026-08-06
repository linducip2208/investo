package model

import "time"

type Screener struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	FiltersJSON string    `json:"filters_json" db:"filters_json"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

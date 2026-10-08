package model

import "time"

type PricePrediction struct {
	ID             int64     `json:"id" db:"id"`
	UserID         int64     `json:"user_id" db:"user_id"`
	StockCode      string    `json:"stock_code" db:"stock_code"`
	PredictedPrice float64   `json:"predicted_price" db:"predicted_price"`
	PredictedDate  string    `json:"predicted_date" db:"predicted_date"`
	ActualPrice    *float64  `json:"actual_price" db:"actual_price"`
	Accuracy       *float64  `json:"accuracy" db:"accuracy"`
	Points         int       `json:"points" db:"points"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UserName       string    `json:"user_name" db:"user_name"`
	StockName      string    `json:"stock_name" db:"stock_name"`
}

type PredictionLeader struct {
	UserID        int64   `json:"user_id" db:"user_id"`
	UserName      string  `json:"user_name" db:"user_name"`
	TotalPoints   int     `json:"total_points" db:"total_points"`
	Predictions   int     `json:"predictions" db:"predictions"`
	AvgAccuracy   float64 `json:"avg_accuracy" db:"avg_accuracy"`
}

type AchievementRow struct {
	ID             int64      `json:"id" db:"id"`
	UserID         int64      `json:"user_id" db:"user_id"`
	AchievementKey string     `json:"achievement_key" db:"achievement_key"`
	Progress       int        `json:"progress" db:"progress"`
	UnlockedAt     *time.Time `json:"unlocked_at" db:"unlocked_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

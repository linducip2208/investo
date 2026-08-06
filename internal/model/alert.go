package model

import "time"

type Alert struct {
	ID               int64      `json:"id" db:"id"`
	UserID           int64      `json:"user_id" db:"user_id"`
	StockID          int64      `json:"stock_id" db:"stock_id"`
	Condition        string     `json:"condition" db:"condition"`
	TargetPrice      float64    `json:"target_price" db:"target_price"`
	ConditionsJSON   string     `json:"conditions_json" db:"conditions_json"`
	NotificationType string     `json:"notification_type" db:"notification_type"`
	TriggerCount     int        `json:"trigger_count" db:"trigger_count"`
	LastTriggeredAt  *time.Time `json:"last_triggered_at" db:"last_triggered_at"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

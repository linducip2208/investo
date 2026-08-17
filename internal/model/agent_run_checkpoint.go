package model

import "time"

type AgentRunCheckpoint struct {
	ID          int64     `json:"id" db:"id"`
	Ticker      string    `json:"ticker" db:"ticker"`
	RunDate     time.Time `json:"run_date" db:"run_date"`
	Phase       string    `json:"phase" db:"phase"`
	ReportsJSON string    `json:"reports_json" db:"reports_json"`
	DebateLog   string    `json:"debate_log" db:"debate_log"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

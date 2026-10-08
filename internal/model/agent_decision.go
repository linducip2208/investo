package model

import "time"

type AgentDecision struct {
	ID             int64     `json:"id" db:"id"`
	Ticker         string    `json:"ticker" db:"ticker"`
	DecisionDate   time.Time `json:"decision_date" db:"decision_date"`
	FinalSignal    string    `json:"final_signal" db:"final_signal"`
	EntryPrice     float64   `json:"entry_price" db:"entry_price"`
	TargetPrice    float64   `json:"target_price" db:"target_price"`
	StopLoss       float64   `json:"stop_loss" db:"stop_loss"`
	PositionPct    float64   `json:"position_pct" db:"position_pct"`
	Confidence     float64   `json:"confidence" db:"confidence"`
	RiskScore      float64   `json:"risk_score" db:"risk_score"`
	ReportsJSON    string    `json:"reports_json" db:"reports_json"`
	DebateLog      string    `json:"debate_log" db:"debate_log"`
	Rationale      string    `json:"rationale" db:"rationale"`
	Outcome        string    `json:"outcome" db:"outcome"`
	RealizedReturn float64   `json:"realized_return" db:"realized_return"`
	Reflection     string    `json:"reflection" db:"reflection"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

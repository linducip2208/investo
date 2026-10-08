package model

import "time"

// AgentMandate is a saved "fund" strategy config (ala AI Hedge Fund mandate):
// it captures persona, debate rounds, risk profile, position caps, and a watchlist,
// without being tied to a single run.
type AgentMandate struct {
	ID           int64     `json:"id" db:"id"`
	UserID       int64     `json:"user_id" db:"user_id"`
	Name         string    `json:"name" db:"name"`
	PersonaStyle string    `json:"persona_style" db:"persona_style"`
	DebateRounds int       `json:"debate_rounds" db:"debate_rounds"`
	RiskProfile  string    `json:"risk_profile" db:"risk_profile"`
	MaxPosition  float64   `json:"max_position" db:"max_position"`
	TickersJSON  string    `json:"tickers_json" db:"tickers_json"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

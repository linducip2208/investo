package model

import "time"

type Bot struct {
	ID        int64             `json:"id" db:"id"`
	UserID    int64             `json:"user_id" db:"user_id"`
	Name      string            `json:"name" db:"name"`
	Type      string            `json:"type" db:"type"`
	Config    string            `json:"config" db:"config"`
	Active    bool              `json:"active" db:"active"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
}

type BotConfig struct {
	BotToken string `json:"bot_token,omitempty"`
	ChatID   string `json:"chat_id,omitempty"`
	URL      string `json:"url,omitempty"`
	Secret   string `json:"secret,omitempty"`
	SMTPHost string `json:"smtp_host,omitempty"`
	SMTPPort string `json:"smtp_port,omitempty"`
	SMTPUser string `json:"smtp_user,omitempty"`
	SMTPPass string `json:"smtp_pass,omitempty"`
}

type Subscription struct {
	ID        int64      `json:"id" db:"id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	Plan      string     `json:"plan" db:"plan"`
	Status    string     `json:"status" db:"status"`
	StartedAt time.Time  `json:"started_at" db:"started_at"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
}

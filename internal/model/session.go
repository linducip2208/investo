package model

import "time"

type Session struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	UserAgent string    `db:"user_agent"`
	IP        string    `db:"ip"`
	CreatedAt time.Time `db:"created_at"`
	LastSeenAt time.Time `db:"last_seen_at"`
	RevokedAt *time.Time `db:"revoked_at"`
}

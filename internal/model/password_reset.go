package model

import "time"

type PasswordReset struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time `db:"created_at"`
}

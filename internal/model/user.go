package model

import "time"

type User struct {
	ID              int64      `json:"id" db:"id"`
	Name            string     `json:"name" db:"name"`
	Email           string     `json:"email" db:"email"`
	Password        string     `json:"-" db:"password"`
	Role            string     `json:"role" db:"role"`
	Plan            string     `json:"plan" db:"plan"`
	EmailVerifiedAt *time.Time `json:"-" db:"email_verified_at"`
	TwoFASecret     *string    `json:"-" db:"twofa_secret"`
	TwoFAEnabled    bool       `json:"-" db:"twofa_enabled"`
	GoogleID        *string    `json:"-" db:"google_id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

package model

import "time"

type ApprovalRequest struct {
	ID              int64      `json:"id" db:"id"`
	UserID          int64      `json:"user_id" db:"user_id"`
	Type            string     `json:"type" db:"type"`
	Amount          float64    `json:"amount" db:"amount"`
	Description     string     `json:"description" db:"description"`
	Status          string     `json:"status" db:"status"`
	RequestedAt     time.Time  `json:"requested_at" db:"requested_at"`
	ReviewedBy      *int64     `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at" db:"reviewed_at"`
	RejectionReason string     `json:"rejection_reason" db:"rejection_reason"`
}

package model

import "time"

type Payment struct {
	ID          int64      `db:"id"`
	UserID      int64      `db:"user_id"`
	OrderID     string     `db:"order_id"`
	Plan        string     `db:"plan"`
	Amount      int64      `db:"amount"`
	Status      string     `db:"status"`
	RawCallback string     `db:"raw_callback"`
	PaidAt      *time.Time `db:"paid_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

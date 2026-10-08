package repository

import (
	"database/sql"
	"fmt"

	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type PaymentRepository struct {
	DB *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

func (r *PaymentRepository) UpsertByOrderID(p *model.Payment) error {
	query := `INSERT INTO payments (user_id, order_id, plan, amount, status, raw_callback, paid_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), raw_callback = VALUES(raw_callback), paid_at = VALUES(paid_at), plan = VALUES(plan), amount = VALUES(amount)`
	if _, err := r.DB.Exec(query, p.UserID, p.OrderID, p.Plan, p.Amount, p.Status, p.RawCallback, p.PaidAt); err != nil {
		return fmt.Errorf("PaymentRepository.UpsertByOrderID: %w", err)
	}
	return nil
}

func (r *PaymentRepository) FindByOrderID(orderID string) (*model.Payment, error) {
	var p model.Payment
	query := `SELECT * FROM payments WHERE order_id = ? LIMIT 1`
	if err := r.DB.Get(&p, query, orderID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("PaymentRepository.FindByOrderID: not found")
		}
		return nil, fmt.Errorf("PaymentRepository.FindByOrderID: %w", err)
	}
	return &p, nil
}

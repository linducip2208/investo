package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"investo/internal/model"
)

type ApprovalRepository struct {
	DB *sqlx.DB
}

func (r *ApprovalRepository) Create(a *model.ApprovalRequest) (int64, error) {
	query := `INSERT INTO approval_requests (user_id, type, amount, description, status)
		VALUES (?, ?, ?, ?, ?)`
	res, err := r.DB.Exec(query, a.UserID, a.Type, a.Amount, a.Description, a.Status)
	if err != nil {
		return 0, fmt.Errorf("ApprovalRepository.Create: %w", err)
	}
	return res.LastInsertId()
}

func (r *ApprovalRepository) FindPending() ([]model.ApprovalRequest, error) {
	var reqs []model.ApprovalRequest
	query := `SELECT * FROM approval_requests WHERE status = 'pending' ORDER BY requested_at DESC`
	if err := r.DB.Select(&reqs, query); err != nil {
		return nil, fmt.Errorf("ApprovalRepository.FindPending: %w", err)
	}
	return reqs, nil
}

func (r *ApprovalRepository) FindByID(id int64) (*model.ApprovalRequest, error) {
	var a model.ApprovalRequest
	if err := r.DB.Get(&a, `SELECT * FROM approval_requests WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("ApprovalRepository.FindByID: %w", err)
	}
	return &a, nil
}

func (r *ApprovalRepository) UpdateStatus(id int64, status string, reviewedBy int64, reason string) error {
	now := time.Now()
	query := `UPDATE approval_requests SET status = ?, reviewed_by = ?, reviewed_at = ?, rejection_reason = ? WHERE id = ?`
	_, err := r.DB.Exec(query, status, reviewedBy, now, reason, id)
	if err != nil {
		return fmt.Errorf("ApprovalRepository.UpdateStatus: %w", err)
	}
	return nil
}

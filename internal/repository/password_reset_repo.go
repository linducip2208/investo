package repository

import (
	"database/sql"
	"fmt"
	"time"

	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type PasswordResetRepository struct {
	DB *sqlx.DB
}

func NewPasswordResetRepository(db *sqlx.DB) *PasswordResetRepository {
	return &PasswordResetRepository{DB: db}
}

func (r *PasswordResetRepository) Create(userID int64, tokenHash string, expiresAt time.Time) error {
	query := `INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES (?, ?, ?)`
	if _, err := r.DB.Exec(query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("PasswordResetRepository.Create: %w", err)
	}
	return nil
}

func (r *PasswordResetRepository) FindValid(tokenHash string) (*model.PasswordReset, error) {
	var pr model.PasswordReset
	query := `SELECT * FROM password_resets WHERE token_hash = ? AND used_at IS NULL AND expires_at > NOW() LIMIT 1`
	if err := r.DB.Get(&pr, query, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("PasswordResetRepository.FindValid: not found")
		}
		return nil, fmt.Errorf("PasswordResetRepository.FindValid: %w", err)
	}
	return &pr, nil
}

func (r *PasswordResetRepository) MarkUsed(id int64) error {
	query := `UPDATE password_resets SET used_at = NOW() WHERE id = ?`
	if _, err := r.DB.Exec(query, id); err != nil {
		return fmt.Errorf("PasswordResetRepository.MarkUsed: %w", err)
	}
	return nil
}

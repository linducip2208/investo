package repository

import (
	"database/sql"
	"fmt"

	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type SessionRepository struct {
	DB *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{DB: db}
}

func (r *SessionRepository) Create(userID int64, tokenHash, userAgent, ip string) error {
	query := `INSERT INTO sessions (user_id, token_hash, user_agent, ip) VALUES (?, ?, ?, ?)`
	if _, err := r.DB.Exec(query, userID, tokenHash, userAgent, ip); err != nil {
		return fmt.Errorf("SessionRepository.Create: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindValid(tokenHash string) (*model.Session, error) {
	var s model.Session
	query := `SELECT * FROM sessions WHERE token_hash = ? AND revoked_at IS NULL LIMIT 1`
	if err := r.DB.Get(&s, query, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("SessionRepository.FindValid: not found")
		}
		return nil, fmt.Errorf("SessionRepository.FindValid: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) Touch(tokenHash string) error {
	query := `UPDATE sessions SET last_seen_at = NOW() WHERE token_hash = ? AND revoked_at IS NULL`
	if _, err := r.DB.Exec(query, tokenHash); err != nil {
		return fmt.Errorf("SessionRepository.Touch: %w", err)
	}
	return nil
}

func (r *SessionRepository) Revoke(tokenHash string) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE token_hash = ?`
	if _, err := r.DB.Exec(query, tokenHash); err != nil {
		return fmt.Errorf("SessionRepository.Revoke: %w", err)
	}
	return nil
}

func (r *SessionRepository) RevokeAllForUser(userID int64) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE user_id = ? AND revoked_at IS NULL`
	if _, err := r.DB.Exec(query, userID); err != nil {
		return fmt.Errorf("SessionRepository.RevokeAllForUser: %w", err)
	}
	return nil
}

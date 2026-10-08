package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AuditLogRepository struct {
	DB *sqlx.DB
}

func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{DB: db}
}

func (r *AuditLogRepository) Write(actorUserID *int64, action, entity, entityID, meta, ip string) error {
	query := `INSERT INTO audit_logs (actor_user_id, action, entity, entity_id, meta, ip) VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := r.DB.Exec(query, actorUserID, action, entity, entityID, meta, ip); err != nil {
		return fmt.Errorf("AuditLogRepository.Write: %w", err)
	}
	return nil
}

package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type AlertRepository struct {
	DB *sqlx.DB
}

func (r *AlertRepository) Create(a *model.Alert) (int64, error) {
	query := `INSERT INTO alerts (user_id, stock_id, ` + "`condition`" + `, target_price, is_active)
		VALUES (:user_id, :stock_id, :condition, :target_price, :is_active)`
	result, err := r.DB.NamedExec(query, a)
	if err != nil {
		return 0, fmt.Errorf("AlertRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("AlertRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *AlertRepository) FindByID(id int64) (*model.Alert, error) {
	var alert model.Alert
	query := `SELECT * FROM alerts WHERE id = ?`
	if err := r.DB.Get(&alert, query, id); err != nil {
		return nil, fmt.Errorf("AlertRepository.FindByID: %w", err)
	}
	return &alert, nil
}

func (r *AlertRepository) FindByUserID(userID int64) ([]model.Alert, error) {
	var alerts []model.Alert
	query := `SELECT * FROM alerts WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&alerts, query, userID); err != nil {
		return nil, fmt.Errorf("AlertRepository.FindByUserID: %w", err)
	}
	return alerts, nil
}

func (r *AlertRepository) FindActive() ([]model.Alert, error) {
	var alerts []model.Alert
	query := `SELECT * FROM alerts WHERE is_active = true`
	if err := r.DB.Select(&alerts, query); err != nil {
		return nil, fmt.Errorf("AlertRepository.FindActive: %w", err)
	}
	return alerts, nil
}

func (r *AlertRepository) Update(a *model.Alert) error {
	query := `UPDATE alerts SET user_id = :user_id, stock_id = :stock_id, ` + "`condition`" + ` = :condition,
		target_price = :target_price, is_active = :is_active WHERE id = :id`
	_, err := r.DB.NamedExec(query, a)
	if err != nil {
		return fmt.Errorf("AlertRepository.Update: %w", err)
	}
	return nil
}

func (r *AlertRepository) ToggleActive(id int64, active bool) error {
	query := `UPDATE alerts SET is_active = ? WHERE id = ?`
	_, err := r.DB.Exec(query, active, id)
	if err != nil {
		return fmt.Errorf("AlertRepository.ToggleActive: %w", err)
	}
	return nil
}

func (r *AlertRepository) Delete(id int64) error {
	query := `DELETE FROM alerts WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("AlertRepository.Delete: %w", err)
	}
	return nil
}

func (r *AlertRepository) UpdateTrigger(a *model.Alert) error {
	query := `UPDATE alerts SET trigger_count = :trigger_count, last_triggered_at = :last_triggered_at WHERE id = :id`
	_, err := r.DB.NamedExec(query, a)
	if err != nil {
		return fmt.Errorf("AlertRepository.UpdateTrigger: %w", err)
	}
	return nil
}

func (r *AlertRepository) FindTriggeredByUserID(userID int64) ([]model.Alert, error) {
	var alerts []model.Alert
	query := `SELECT * FROM alerts WHERE user_id = ? AND trigger_count > 0 ORDER BY last_triggered_at DESC`
	if err := r.DB.Select(&alerts, query, userID); err != nil {
		return nil, fmt.Errorf("AlertRepository.FindTriggeredByUserID: %w", err)
	}
	return alerts, nil
}

func (r *AlertRepository) FindByUserIDAndStockID(userID, stockID int64) ([]model.Alert, error) {
	var alerts []model.Alert
	query := `SELECT * FROM alerts WHERE user_id = ? AND stock_id = ? AND is_active = true ORDER BY target_price ASC`
	if err := r.DB.Select(&alerts, query, userID, stockID); err != nil {
		return nil, fmt.Errorf("AlertRepository.FindByUserIDAndStockID: %w", err)
	}
	return alerts, nil
}

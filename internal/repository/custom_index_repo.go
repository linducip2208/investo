package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type CustomIndexRow struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	Name        string    `db:"name"`
	StocksJSON  string    `db:"stocks_json"`
	WeightsJSON string    `db:"weights_json"`
	BaseValue   float64   `db:"base_value"`
	CreatedAt   time.Time `db:"created_at"`
}

type CustomIndexRepository struct {
	DB *sqlx.DB
}

func (r *CustomIndexRepository) Create(userID int64, name string, stocksJSON string, weightsJSON string, baseValue float64) (int64, error) {
	query := `INSERT INTO custom_indices (user_id, name, stocks_json, weights_json, base_value) VALUES (?, ?, ?, ?, ?)`
	result, err := r.DB.Exec(query, userID, name, stocksJSON, weightsJSON, baseValue)
	if err != nil {
		return 0, fmt.Errorf("CustomIndexRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CustomIndexRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *CustomIndexRepository) FindByID(id int64) (*CustomIndexRow, error) {
	var row CustomIndexRow
	query := `SELECT * FROM custom_indices WHERE id = ?`
	if err := r.DB.Get(&row, query, id); err != nil {
		return nil, fmt.Errorf("CustomIndexRepository.FindByID: %w", err)
	}
	return &row, nil
}

func (r *CustomIndexRepository) FindByUserID(userID int64) ([]CustomIndexRow, error) {
	var rows []CustomIndexRow
	query := `SELECT * FROM custom_indices WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&rows, query, userID); err != nil {
		return nil, fmt.Errorf("CustomIndexRepository.FindByUserID: %w", err)
	}
	return rows, nil
}

func (r *CustomIndexRepository) Delete(id int64) error {
	query := `DELETE FROM custom_indices WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("CustomIndexRepository.Delete: %w", err)
	}
	return nil
}

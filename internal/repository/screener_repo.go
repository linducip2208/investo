package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type ScreenerRepository struct {
	DB *sqlx.DB
}

func (r *ScreenerRepository) Create(s *model.Screener) (int64, error) {
	query := `INSERT INTO screeners (user_id, name, filters_json) VALUES (:user_id, :name, :filters_json)`
	result, err := r.DB.NamedExec(query, s)
	if err != nil {
		return 0, fmt.Errorf("ScreenerRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ScreenerRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *ScreenerRepository) FindByID(id int64) (*model.Screener, error) {
	var screener model.Screener
	query := `SELECT * FROM screeners WHERE id = ?`
	if err := r.DB.Get(&screener, query, id); err != nil {
		return nil, fmt.Errorf("ScreenerRepository.FindByID: %w", err)
	}
	return &screener, nil
}

func (r *ScreenerRepository) FindByUserID(userID int64) ([]model.Screener, error) {
	var screeners []model.Screener
	query := `SELECT * FROM screeners WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&screeners, query, userID); err != nil {
		return nil, fmt.Errorf("ScreenerRepository.FindByUserID: %w", err)
	}
	return screeners, nil
}

func (r *ScreenerRepository) Delete(id int64) error {
	query := `DELETE FROM screeners WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("ScreenerRepository.Delete: %w", err)
	}
	return nil
}

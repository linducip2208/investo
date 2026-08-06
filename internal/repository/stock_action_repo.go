package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type StockActionRepository struct {
	DB *sqlx.DB
}

func (r *StockActionRepository) Create(a *model.StockAction) (int64, error) {
	query := `INSERT INTO stock_actions (stock_id, action_type, ex_date, record_date, payment_date, description, ratio, price)
		VALUES (:stock_id, :action_type, :ex_date, :record_date, :payment_date, :description, :ratio, :price)`
	result, err := r.DB.NamedExec(query, a)
	if err != nil {
		return 0, fmt.Errorf("StockActionRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("StockActionRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *StockActionRepository) FindByStockID(stockID int64) ([]model.StockAction, error) {
	var actions []model.StockAction
	query := `SELECT * FROM stock_actions WHERE stock_id = ? ORDER BY ex_date DESC`
	if err := r.DB.Select(&actions, query, stockID); err != nil {
		return nil, fmt.Errorf("StockActionRepository.FindByStockID: %w", err)
	}
	return actions, nil
}

func (r *StockActionRepository) FindUpcoming(limit int) ([]model.StockAction, error) {
	var actions []model.StockAction
	query := `SELECT * FROM stock_actions WHERE ex_date >= CURDATE() ORDER BY ex_date ASC LIMIT ?`
	if err := r.DB.Select(&actions, query, limit); err != nil {
		return nil, fmt.Errorf("StockActionRepository.FindUpcoming: %w", err)
	}
	return actions, nil
}

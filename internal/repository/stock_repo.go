package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type StockRepository struct {
	DB *sqlx.DB
}

func (r *StockRepository) Create(stock *model.Stock) (int64, error) {
	query := `INSERT INTO stocks (code, name, sector_id, subsector, listing_date, shares_outstanding, description, logo_url, website)
		VALUES (:code, :name, :sector_id, :subsector, :listing_date, :shares_outstanding, :description, :logo_url, :website)`
	result, err := r.DB.NamedExec(query, stock)
	if err != nil {
		return 0, fmt.Errorf("StockRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("StockRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *StockRepository) FindByID(id int64) (*model.Stock, error) {
	var stock model.Stock
	query := `SELECT * FROM stocks WHERE id = ?`
	if err := r.DB.Get(&stock, query, id); err != nil {
		return nil, fmt.Errorf("StockRepository.FindByID: %w", err)
	}
	return &stock, nil
}

func (r *StockRepository) FindByCode(code string) (*model.Stock, error) {
	var stock model.Stock
	query := `SELECT * FROM stocks WHERE code = ?`
	if err := r.DB.Get(&stock, query, code); err != nil {
		return nil, fmt.Errorf("StockRepository.FindByCode: %w", err)
	}
	return &stock, nil
}

func (r *StockRepository) Update(stock *model.Stock) error {
	query := `UPDATE stocks SET code = :code, name = :name, sector_id = :sector_id, subsector = :subsector,
		listing_date = :listing_date, shares_outstanding = :shares_outstanding, description = :description,
		logo_url = :logo_url, website = :website WHERE id = :id`
	_, err := r.DB.NamedExec(query, stock)
	if err != nil {
		return fmt.Errorf("StockRepository.Update: %w", err)
	}
	return nil
}

func (r *StockRepository) Delete(id int64) error {
	query := `DELETE FROM stocks WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("StockRepository.Delete: %w", err)
	}
	return nil
}

func (r *StockRepository) List(offset, limit int, search string, sectorID int64) ([]model.Stock, int, error) {
	var stocks []model.Stock
	var total int
	args := []interface{}{}

	where := "WHERE 1=1"
	if search != "" {
		where += " AND (code LIKE ? OR name LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if sectorID > 0 {
		where += " AND sector_id = ?"
		args = append(args, sectorID)
	}

	countQuery := "SELECT COUNT(*) FROM stocks " + where
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("StockRepository.List count: %w", err)
	}

	dataArgs := append(args, limit, offset)
	dataQuery := "SELECT * FROM stocks " + where + " ORDER BY code ASC LIMIT ? OFFSET ?"
	if err := r.DB.Select(&stocks, dataQuery, dataArgs...); err != nil {
		return nil, 0, fmt.Errorf("StockRepository.List: %w", err)
	}

	return stocks, total, nil
}

func (r *StockRepository) ListActive() ([]model.Stock, error) {
	var stocks []model.Stock
	query := `SELECT id, code, name,
		COALESCE(sector_id, 0) AS sector_id,
		COALESCE(subsector, '') AS subsector,
		COALESCE(listing_date, DATE('2000-01-01')) AS listing_date,
		COALESCE(shares_outstanding, 0) AS shares_outstanding,
		COALESCE(description, '') AS description,
		COALESCE(logo_url, '') AS logo_url,
		COALESCE(website, '') AS website,
		COALESCE(created_at, NOW()) AS created_at,
		COALESCE(updated_at, NOW()) AS updated_at
		FROM stocks ORDER BY code ASC`
	if err := r.DB.Select(&stocks, query); err != nil {
		return nil, fmt.Errorf("StockRepository.ListActive: %w", err)
	}
	return stocks, nil
}

func (r *StockRepository) Search(query string, limit int) ([]model.Stock, error) {
	var stocks []model.Stock
	sql := `SELECT * FROM stocks WHERE code LIKE ? OR name LIKE ? ORDER BY code ASC LIMIT ?`
	like := "%" + query + "%"
	if err := r.DB.Select(&stocks, sql, like, like, limit); err != nil {
		return nil, fmt.Errorf("StockRepository.Search: %w", err)
	}
	return stocks, nil
}

func (r *StockRepository) GetAllCodes() ([]string, error) {
	var codes []string
	query := `SELECT code FROM stocks ORDER BY code ASC`
	if err := r.DB.Select(&codes, query); err != nil {
		return nil, fmt.Errorf("StockRepository.GetAllCodes: %w", err)
	}
	return codes, nil
}

package repository

import (
	"database/sql"
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type PortfolioRepository struct {
	DB *sqlx.DB
}

func (r *PortfolioRepository) Create(p *model.Portfolio) (int64, error) {
	query := `INSERT INTO portfolios (user_id, name, description, is_default) VALUES (:user_id, :name, :description, :is_default)`
	result, err := r.DB.NamedExec(query, p)
	if err != nil {
		return 0, fmt.Errorf("PortfolioRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("PortfolioRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *PortfolioRepository) FindByID(id int64) (*model.Portfolio, error) {
	var portfolio model.Portfolio
	query := `SELECT * FROM portfolios WHERE id = ?`
	if err := r.DB.Get(&portfolio, query, id); err != nil {
		return nil, fmt.Errorf("PortfolioRepository.FindByID: %w", err)
	}
	return &portfolio, nil
}

func (r *PortfolioRepository) FindByUserID(userID int64) ([]model.Portfolio, error) {
	var portfolios []model.Portfolio
	query := `SELECT * FROM portfolios WHERE user_id = ? ORDER BY is_default DESC, created_at DESC`
	if err := r.DB.Select(&portfolios, query, userID); err != nil {
		return nil, fmt.Errorf("PortfolioRepository.FindByUserID: %w", err)
	}
	return portfolios, nil
}

func (r *PortfolioRepository) Update(p *model.Portfolio) error {
	query := `UPDATE portfolios SET user_id = :user_id, name = :name, description = :description, is_default = :is_default WHERE id = :id`
	_, err := r.DB.NamedExec(query, p)
	if err != nil {
		return fmt.Errorf("PortfolioRepository.Update: %w", err)
	}
	return nil
}

func (r *PortfolioRepository) Delete(id int64) error {
	query := `DELETE FROM portfolios WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("PortfolioRepository.Delete: %w", err)
	}
	return nil
}

type PortfolioItemRepository struct {
	DB *sqlx.DB
}

func (r *PortfolioItemRepository) Create(item *model.PortfolioItem) (int64, error) {
	query := `INSERT INTO portfolio_items (portfolio_id, stock_id, type, quantity, avg_price, notes)
		VALUES (:portfolio_id, :stock_id, :type, :quantity, :avg_price, :notes)`
	result, err := r.DB.NamedExec(query, item)
	if err != nil {
		return 0, fmt.Errorf("PortfolioItemRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("PortfolioItemRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *PortfolioItemRepository) FindByPortfolioID(portfolioID int64) ([]model.PortfolioItem, error) {
	var items []model.PortfolioItem
	query := `SELECT * FROM portfolio_items WHERE portfolio_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&items, query, portfolioID); err != nil {
		return nil, fmt.Errorf("PortfolioItemRepository.FindByPortfolioID: %w", err)
	}
	return items, nil
}

type PortfolioItemWithStock struct {
	Item  model.PortfolioItem `db:"item"`
	Stock model.Stock         `db:"stock"`
}

func (r *PortfolioItemRepository) GetWithStock(portfolioID int64) ([]PortfolioItemWithStock, error) {
	query := `SELECT pi.id AS 'item.id', pi.portfolio_id AS 'item.portfolio_id', pi.stock_id AS 'item.stock_id',
		pi.type AS 'item.type', pi.quantity AS 'item.quantity', pi.avg_price AS 'item.avg_price',
		pi.notes AS 'item.notes', pi.created_at AS 'item.created_at', pi.updated_at AS 'item.updated_at',
		s.id AS 'stock.id', s.code AS 'stock.code', s.name AS 'stock.name', s.sector_id AS 'stock.sector_id',
		s.subsector AS 'stock.subsector', s.listing_date AS 'stock.listing_date',
		s.shares_outstanding AS 'stock.shares_outstanding', s.description AS 'stock.description',
		s.logo_url AS 'stock.logo_url', s.website AS 'stock.website',
		s.created_at AS 'stock.created_at', s.updated_at AS 'stock.updated_at'
		FROM portfolio_items pi
		INNER JOIN stocks s ON pi.stock_id = s.id
		WHERE pi.portfolio_id = ?
		ORDER BY pi.created_at DESC`

	rows, err := r.DB.Queryx(query, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("PortfolioItemRepository.GetWithStock: %w", err)
	}
	defer rows.Close()

	var results []PortfolioItemWithStock
	for rows.Next() {
		var r PortfolioItemWithStock
		if err := rows.Scan(
			&r.Item.ID, &r.Item.PortfolioID, &r.Item.StockID, &r.Item.Type,
			&r.Item.Quantity, &r.Item.AvgPrice, &r.Item.Notes,
			&r.Item.CreatedAt, &r.Item.UpdatedAt,
			&r.Stock.ID, &r.Stock.Code, &r.Stock.Name, &r.Stock.SectorID,
			&r.Stock.Subsector, &r.Stock.ListingDate, &r.Stock.SharesOutstanding,
			&r.Stock.Description, &r.Stock.LogoURL, &r.Stock.Website,
			&r.Stock.CreatedAt, &r.Stock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("PortfolioItemRepository.GetWithStock scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PortfolioItemRepository.GetWithStock rows: %w", err)
	}

	return results, nil
}

func (r *PortfolioItemRepository) Update(item *model.PortfolioItem) error {
	query := `UPDATE portfolio_items SET portfolio_id = :portfolio_id, stock_id = :stock_id, type = :type,
		quantity = :quantity, avg_price = :avg_price, notes = :notes WHERE id = :id`
	_, err := r.DB.NamedExec(query, item)
	if err != nil {
		return fmt.Errorf("PortfolioItemRepository.Update: %w", err)
	}
	return nil
}

func (r *PortfolioItemRepository) Delete(id int64) error {
	query := `DELETE FROM portfolio_items WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("PortfolioItemRepository.Delete: %w", err)
	}
	return nil
}

// UpdateForPortfolio prevents an item ID from being moved out of, or updated
// through, a portfolio the caller does not own. Immutable stock/type fields are
// intentionally left untouched.
func (r *PortfolioItemRepository) UpdateForPortfolio(item *model.PortfolioItem) error {
	query := `UPDATE portfolio_items SET quantity = ?, avg_price = ?, notes = ? WHERE id = ? AND portfolio_id = ?`
	result, err := r.DB.Exec(query, item.Quantity, item.AvgPrice, item.Notes, item.ID, item.PortfolioID)
	if err != nil {
		return fmt.Errorf("PortfolioItemRepository.UpdateForPortfolio: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("PortfolioItemRepository.UpdateForPortfolio RowsAffected: %w", err)
	}
	if affected != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteForPortfolio binds the item ID to its parent portfolio in one atomic
// statement, closing the cross-portfolio IDOR window.
func (r *PortfolioItemRepository) DeleteForPortfolio(id, portfolioID int64) error {
	result, err := r.DB.Exec(`DELETE FROM portfolio_items WHERE id = ? AND portfolio_id = ?`, id, portfolioID)
	if err != nil {
		return fmt.Errorf("PortfolioItemRepository.DeleteForPortfolio: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("PortfolioItemRepository.DeleteForPortfolio RowsAffected: %w", err)
	}
	if affected != 1 {
		return sql.ErrNoRows
	}
	return nil
}

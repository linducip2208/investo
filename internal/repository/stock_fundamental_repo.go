package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type StockFundamentalRepository struct {
	DB *sqlx.DB
}

func (r *StockFundamentalRepository) Create(f *model.StockFundamental) (int64, error) {
	query := `INSERT INTO stock_fundamentals (stock_id, period, report_type, source, revenue, net_income, eps, bvps,
		total_assets, total_liabilities, equity, roe, roa, per, pbv, der, net_profit_margin, dividend_yield)
		VALUES (:stock_id, :period, :report_type, :source, :revenue, :net_income, :eps, :bvps,
		:total_assets, :total_liabilities, :equity, :roe, :roa, :per, :pbv, :der, :net_profit_margin, :dividend_yield)`
	result, err := r.DB.NamedExec(query, f)
	if err != nil {
		return 0, fmt.Errorf("StockFundamentalRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("StockFundamentalRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *StockFundamentalRepository) Upsert(f *model.StockFundamental) error {
	query := `INSERT INTO stock_fundamentals (stock_id, period, report_type, source, revenue, net_income, eps, bvps,
		total_assets, total_liabilities, equity, roe, roa, per, pbv, der, net_profit_margin, dividend_yield)
		VALUES (:stock_id, :period, :report_type, :source, :revenue, :net_income, :eps, :bvps,
		:total_assets, :total_liabilities, :equity, :roe, :roa, :per, :pbv, :der, :net_profit_margin, :dividend_yield)
		ON DUPLICATE KEY UPDATE
		source = VALUES(source),
		revenue = VALUES(revenue), net_income = VALUES(net_income), eps = VALUES(eps), bvps = VALUES(bvps),
		total_assets = VALUES(total_assets), total_liabilities = VALUES(total_liabilities), equity = VALUES(equity),
		roe = VALUES(roe), roa = VALUES(roa), per = VALUES(per), pbv = VALUES(pbv), der = VALUES(der),
		net_profit_margin = VALUES(net_profit_margin), dividend_yield = VALUES(dividend_yield)`
	_, err := r.DB.NamedExec(query, f)
	if err != nil {
		return fmt.Errorf("StockFundamentalRepository.Upsert: %w", err)
	}
	return nil
}

func (r *StockFundamentalRepository) FindByStockID(stockID int64, limit int) ([]model.StockFundamental, error) {
	var fundamentals []model.StockFundamental
	query := `SELECT * FROM stock_fundamentals WHERE stock_id = ? ORDER BY period DESC LIMIT ?`
	if err := r.DB.Select(&fundamentals, query, stockID, limit); err != nil {
		return nil, fmt.Errorf("StockFundamentalRepository.FindByStockID: %w", err)
	}
	return fundamentals, nil
}

func (r *StockFundamentalRepository) FindByStockPeriod(stockID int64, period, reportType string) (*model.StockFundamental, error) {
	var f model.StockFundamental
	query := `SELECT * FROM stock_fundamentals WHERE stock_id = ? AND period = ? AND report_type = ?`
	if err := r.DB.Get(&f, query, stockID, period, reportType); err != nil {
		return nil, fmt.Errorf("StockFundamentalRepository.FindByStockPeriod: %w", err)
	}
	return &f, nil
}

func (r *StockFundamentalRepository) FindLatest(stockID int64) (*model.StockFundamental, error) {
	var f model.StockFundamental
	query := `SELECT * FROM stock_fundamentals WHERE stock_id = ? ORDER BY period DESC LIMIT 1`
	if err := r.DB.Get(&f, query, stockID); err != nil {
		return nil, fmt.Errorf("StockFundamentalRepository.FindLatest: %w", err)
	}
	return &f, nil
}

func (r *StockFundamentalRepository) FindLatestByCode(code string) (*model.StockFundamental, error) {
	var f model.StockFundamental
	query := `SELECT sf.* FROM stock_fundamentals sf
		JOIN stocks s ON s.id = sf.stock_id
		WHERE s.code = ?
		ORDER BY sf.period DESC LIMIT 1`
	if err := r.DB.Get(&f, query, code); err != nil {
		return nil, fmt.Errorf("StockFundamentalRepository.FindLatestByCode: %w", err)
	}
	return &f, nil
}

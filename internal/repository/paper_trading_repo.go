package repository

import (
	"fmt"

	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type PaperTradingRepository struct {
	DB *sqlx.DB
}

func (r *PaperTradingRepository) CreatePortfolio(p *model.PaperPortfolio) (int64, error) {
	query := `INSERT INTO paper_portfolios (user_id, name, initial_balance, cash_balance) VALUES (?, ?, ?, ?)`
	result, err := r.DB.Exec(query, p.UserID, p.Name, p.InitialBalance, p.CashBalance)
	if err != nil {
		return 0, fmt.Errorf("CreatePortfolio: %w", err)
	}
	return result.LastInsertId()
}

func (r *PaperTradingRepository) FindPortfolioByID(id int64) (*model.PaperPortfolio, error) {
	var p model.PaperPortfolio
	query := `SELECT id, user_id, name, initial_balance, cash_balance, created_at FROM paper_portfolios WHERE id = ?`
	if err := r.DB.Get(&p, query, id); err != nil {
		return nil, fmt.Errorf("FindPortfolioByID: %w", err)
	}
	return &p, nil
}

func (r *PaperTradingRepository) FindPortfoliosByUserID(userID int64) ([]model.PaperPortfolio, error) {
	var portfolios []model.PaperPortfolio
	query := `SELECT id, user_id, name, initial_balance, cash_balance, created_at FROM paper_portfolios WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&portfolios, query, userID); err != nil {
		return nil, fmt.Errorf("FindPortfoliosByUserID: %w", err)
	}
	return portfolios, nil
}

func (r *PaperTradingRepository) UpdateCashBalance(id int64, balance float64) error {
	query := `UPDATE paper_portfolios SET cash_balance = ? WHERE id = ?`
	_, err := r.DB.Exec(query, balance, id)
	if err != nil {
		return fmt.Errorf("UpdateCashBalance: %w", err)
	}
	return nil
}

func (r *PaperTradingRepository) InsertTrade(trade *model.PaperTrade) (int64, error) {
	query := `INSERT INTO paper_trades (portfolio_id, stock_code, type, quantity, price, total, fee) VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.DB.Exec(query, trade.PortfolioID, trade.StockCode, trade.Type, trade.Quantity, trade.Price, trade.Total, trade.Fee)
	if err != nil {
		return 0, fmt.Errorf("InsertTrade: %w", err)
	}
	return result.LastInsertId()
}

func (r *PaperTradingRepository) FindTradesByPortfolioID(portfolioID int64, limit int) ([]model.PaperTrade, error) {
	if limit <= 0 {
		limit = 50
	}
	var trades []model.PaperTrade
	query := `SELECT id, portfolio_id, stock_code, type, quantity, price, total, fee, executed_at FROM paper_trades WHERE portfolio_id = ? ORDER BY executed_at DESC LIMIT ?`
	if err := r.DB.Select(&trades, query, portfolioID, limit); err != nil {
		return nil, fmt.Errorf("FindTradesByPortfolioID: %w", err)
	}
	return trades, nil
}

func (r *PaperTradingRepository) FindHolding(portfolioID int64, code string) (*model.PaperHolding, error) {
	var h model.PaperHolding
	query := `SELECT id, portfolio_id, stock_code, quantity, avg_price FROM paper_holdings WHERE portfolio_id = ? AND stock_code = ?`
	if err := r.DB.Get(&h, query, portfolioID, code); err != nil {
		return nil, fmt.Errorf("FindHolding: %w", err)
	}
	return &h, nil
}

func (r *PaperTradingRepository) UpsertHolding(portfolioID int64, code string, quantity, avgPrice float64) error {
	query := `INSERT INTO paper_holdings (portfolio_id, stock_code, quantity, avg_price) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE quantity = VALUES(quantity), avg_price = VALUES(avg_price)`
	_, err := r.DB.Exec(query, portfolioID, code, quantity, avgPrice)
	if err != nil {
		return fmt.Errorf("UpsertHolding: %w", err)
	}
	return nil
}

func (r *PaperTradingRepository) DeleteHolding(portfolioID int64, code string) error {
	query := `DELETE FROM paper_holdings WHERE portfolio_id = ? AND stock_code = ?`
	_, err := r.DB.Exec(query, portfolioID, code)
	if err != nil {
		return fmt.Errorf("DeleteHolding: %w", err)
	}
	return nil
}

func (r *PaperTradingRepository) FindHoldingsByPortfolioID(portfolioID int64) ([]model.PaperHolding, error) {
	var holdings []model.PaperHolding
	query := `SELECT id, portfolio_id, stock_code, quantity, avg_price FROM paper_holdings WHERE portfolio_id = ?`
	if err := r.DB.Select(&holdings, query, portfolioID); err != nil {
		return nil, fmt.Errorf("FindHoldingsByPortfolioID: %w", err)
	}
	return holdings, nil
}

func (r *PaperTradingRepository) GetAllPortfolios() ([]model.PaperPortfolio, error) {
	var portfolios []model.PaperPortfolio
	query := `SELECT id, user_id, name, initial_balance, cash_balance, created_at FROM paper_portfolios`
	if err := r.DB.Select(&portfolios, query); err != nil {
		return nil, fmt.Errorf("GetAllPortfolios: %w", err)
	}
	return portfolios, nil
}

func (r *PaperTradingRepository) FindSubscriptionByUserID(userID int64) (*model.Subscription, error) {
	var sub model.Subscription
	query := `SELECT id, user_id, plan, status, started_at, expires_at FROM subscriptions WHERE user_id = ? ORDER BY id DESC LIMIT 1`
	if err := r.DB.Get(&sub, query, userID); err != nil {
		return nil, fmt.Errorf("FindSubscriptionByUserID: %w", err)
	}
	return &sub, nil
}

func (r *PaperTradingRepository) UpsertSubscription(sub *model.Subscription) error {
	query := `INSERT INTO subscriptions (user_id, plan, status, started_at, expires_at) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE plan = VALUES(plan), status = VALUES(status), expires_at = VALUES(expires_at)`
	_, err := r.DB.Exec(query, sub.UserID, sub.Plan, sub.Status, sub.StartedAt, sub.ExpiresAt)
	if err != nil {
		return fmt.Errorf("UpsertSubscription: %w", err)
	}
	return nil
}

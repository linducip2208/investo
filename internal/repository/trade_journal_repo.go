package repository

import (
	"fmt"
	"investo/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type TradeJournalRepository struct {
	DB *sqlx.DB
}

func (r *TradeJournalRepository) Create(j *model.TradeJournal) (int64, error) {
	query := `INSERT INTO trade_journal (user_id, stock_code, entry_date, exit_date, entry_price, exit_price,
		quantity, direction, strategy_used, notes, emotions, outcome, profit_loss, profit_loss_pct)
		VALUES (:user_id, :stock_code, :entry_date, :exit_date, :entry_price, :exit_price,
		:quantity, :direction, :strategy_used, :notes, :emotions, :outcome, :profit_loss, :profit_loss_pct)`
	result, err := r.DB.NamedExec(query, j)
	if err != nil {
		return 0, fmt.Errorf("TradeJournalRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("TradeJournalRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *TradeJournalRepository) FindByUserID(userID int64) ([]model.TradeJournal, error) {
	var journals []model.TradeJournal
	query := `SELECT * FROM trade_journal WHERE user_id = ? ORDER BY entry_date DESC, created_at DESC`
	if err := r.DB.Select(&journals, query, userID); err != nil {
		return nil, fmt.Errorf("TradeJournalRepository.FindByUserID: %w", err)
	}
	return journals, nil
}

func (r *TradeJournalRepository) FindByID(id int64) (*model.TradeJournal, error) {
	var journal model.TradeJournal
	query := `SELECT * FROM trade_journal WHERE id = ?`
	if err := r.DB.Get(&journal, query, id); err != nil {
		return nil, fmt.Errorf("TradeJournalRepository.FindByID: %w", err)
	}
	return &journal, nil
}

func (r *TradeJournalRepository) Update(j *model.TradeJournal) error {
	query := `UPDATE trade_journal SET exit_date = :exit_date, exit_price = :exit_price,
		outcome = :outcome, profit_loss = :profit_loss, profit_loss_pct = :profit_loss_pct,
		notes = :notes, emotions = :emotions WHERE id = :id`
	_, err := r.DB.NamedExec(query, j)
	if err != nil {
		return fmt.Errorf("TradeJournalRepository.Update: %w", err)
	}
	return nil
}

func (r *TradeJournalRepository) Delete(id int64) error {
	query := `DELETE FROM trade_journal WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("TradeJournalRepository.Delete: %w", err)
	}
	return nil
}

func (r *TradeJournalRepository) GetStats(userID int64) (*model.TradeJournalStats, error) {
	stats := &model.TradeJournalStats{}

	countQuery := `SELECT COUNT(*) FROM trade_journal WHERE user_id = ?`
	if err := r.DB.Get(&stats.TotalTrades, countQuery, userID); err != nil {
		return nil, fmt.Errorf("TradeJournalRepository.GetStats count: %w", err)
	}

	if stats.TotalTrades == 0 {
		return stats, nil
	}

	type plRow struct {
		PL float64 `db:"profit_loss"`
	}
	var pls []plRow
	if err := r.DB.Select(&pls, `SELECT COALESCE(profit_loss, 0) AS profit_loss FROM trade_journal WHERE user_id = ?`, userID); err != nil {
		return nil, fmt.Errorf("TradeJournalRepository.GetStats pl: %w", err)
	}

	var totalPL float64
	var bestTrade, worstTrade float64
	bestTrade = pls[0].PL
	worstTrade = pls[0].PL
	wins := 0
	for _, p := range pls {
		totalPL += p.PL
		if p.PL > 0 {
			wins++
		}
		if p.PL > bestTrade {
			bestTrade = p.PL
		}
		if p.PL < worstTrade {
			worstTrade = p.PL
		}
	}
	stats.TotalPL = totalPL
	stats.AvgPL = totalPL / float64(stats.TotalTrades)
	stats.WinRate = float64(wins) / float64(stats.TotalTrades) * 100
	stats.BestTrade = bestTrade
	stats.WorstTrade = worstTrade

	return stats, nil
}

func (r *TradeJournalRepository) FindByUserIDFiltered(userID int64, stockCode string, outcome string, startDate, endDate time.Time) ([]model.TradeJournal, error) {
	query := `SELECT * FROM trade_journal WHERE user_id = ?`
	args := []interface{}{userID}

	if stockCode != "" {
		query += ` AND stock_code = ?`
		args = append(args, stockCode)
	}
	if outcome != "" {
		query += ` AND outcome = ?`
		args = append(args, outcome)
	}
	if !startDate.IsZero() {
		query += ` AND entry_date >= ?`
		args = append(args, startDate)
	}
	if !endDate.IsZero() {
		query += ` AND entry_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY entry_date DESC, created_at DESC`

	var journals []model.TradeJournal
	if err := r.DB.Select(&journals, query, args...); err != nil {
		return nil, fmt.Errorf("TradeJournalRepository.FindByUserIDFiltered: %w", err)
	}
	return journals, nil
}

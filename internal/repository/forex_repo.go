package repository

import (
	"fmt"
	"investo/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type ForexRepository struct {
	DB *sqlx.DB
}

func (r *ForexRepository) FindAllPairs() ([]model.ForexPair, error) {
	var pairs []model.ForexPair
	query := `SELECT * FROM forex_pairs ORDER BY name ASC`
	if err := r.DB.Select(&pairs, query); err != nil {
		return nil, fmt.Errorf("ForexRepository.FindAllPairs: %w", err)
	}
	return pairs, nil
}

func (r *ForexRepository) FindPairByID(id int64) (*model.ForexPair, error) {
	var pair model.ForexPair
	query := `SELECT * FROM forex_pairs WHERE id = ?`
	if err := r.DB.Get(&pair, query, id); err != nil {
		return nil, fmt.Errorf("ForexRepository.FindPairByID: %w", err)
	}
	return &pair, nil
}

func (r *ForexRepository) FindPairByCurrencies(base, quote string) (*model.ForexPair, error) {
	var pair model.ForexPair
	query := `SELECT * FROM forex_pairs WHERE base_currency = ? AND quote_currency = ?`
	if err := r.DB.Get(&pair, query, base, quote); err != nil {
		return nil, fmt.Errorf("ForexRepository.FindPairByCurrencies: %w", err)
	}
	return &pair, nil
}

func (r *ForexRepository) BulkInsertRates(rates []model.ForexRate) error {
	if len(rates) == 0 {
		return nil
	}

	query := `INSERT IGNORE INTO forex_rates (pair_id, date, open, high, low, close)
		VALUES (:pair_id, :date, :open, :high, :low, :close)`
	_, err := r.DB.NamedExec(query, rates)
	if err != nil {
		return fmt.Errorf("ForexRepository.BulkInsertRates: %w", err)
	}
	return nil
}

func (r *ForexRepository) FindRates(pairID int64, start, end time.Time) ([]model.ForexRate, error) {
	var rates []model.ForexRate
	query := `SELECT * FROM forex_rates WHERE pair_id = ? AND date >= ? AND date <= ? ORDER BY date ASC`
	if err := r.DB.Select(&rates, query, pairID, start, end); err != nil {
		return nil, fmt.Errorf("ForexRepository.FindRates: %w", err)
	}
	return rates, nil
}

func (r *ForexRepository) FindLatestRates() ([]model.ForexRate, error) {
	var rates []model.ForexRate
	query := `SELECT fr.* FROM forex_rates fr
		INNER JOIN (
			SELECT pair_id, MAX(date) AS max_date FROM forex_rates GROUP BY pair_id
		) latest ON fr.pair_id = latest.pair_id AND fr.date = latest.max_date`
	if err := r.DB.Select(&rates, query); err != nil {
		return nil, fmt.Errorf("ForexRepository.FindLatestRates: %w", err)
	}
	return rates, nil
}

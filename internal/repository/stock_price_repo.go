package repository

import (
	"fmt"
	"investo/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type StockPriceRepository struct {
	DB *sqlx.DB
}

func (r *StockPriceRepository) BulkInsert(prices []model.StockPrice) error {
	if len(prices) == 0 {
		return nil
	}

	query := `INSERT IGNORE INTO stock_prices (stock_id, date, open, high, low, close, volume, adj_close)
		VALUES (:stock_id, :date, :open, :high, :low, :close, :volume, :adj_close)`
	_, err := r.DB.NamedExec(query, prices)
	if err != nil {
		return fmt.Errorf("StockPriceRepository.BulkInsert: %w", err)
	}
	return nil
}

func (r *StockPriceRepository) CountByStockID(stockID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM stock_prices WHERE stock_id = ?`
	if err := r.DB.Get(&count, query, stockID); err != nil {
		return 0, fmt.Errorf("StockPriceRepository.CountByStockID: %w", err)
	}
	return count, nil
}

func (r *StockPriceRepository) FindByStockDate(stockID int64, start, end time.Time) ([]model.StockPrice, error) {
	var prices []model.StockPrice
	query := `SELECT * FROM stock_prices WHERE stock_id = ? AND date >= ? AND date <= ? ORDER BY date ASC`
	if err := r.DB.Select(&prices, query, stockID, start, end); err != nil {
		return nil, fmt.Errorf("StockPriceRepository.FindByStockDate: %w", err)
	}
	return prices, nil
}

func (r *StockPriceRepository) FindLatest(stockID int64, limit int) ([]model.StockPrice, error) {
	var prices []model.StockPrice
	query := `SELECT * FROM stock_prices WHERE stock_id = ? ORDER BY date DESC LIMIT ?`
	if err := r.DB.Select(&prices, query, stockID, limit); err != nil {
		return nil, fmt.Errorf("StockPriceRepository.FindLatest: %w", err)
	}
	return prices, nil
}

func (r *StockPriceRepository) FindLatestByDate(date time.Time) ([]model.StockPrice, error) {
	var prices []model.StockPrice
	query := `SELECT sp.* FROM stock_prices sp
		INNER JOIN (
			SELECT stock_id, MAX(date) AS max_date FROM stock_prices WHERE date <= ? GROUP BY stock_id
		) latest ON sp.stock_id = latest.stock_id AND sp.date = latest.max_date`
	if err := r.DB.Select(&prices, query, date); err != nil {
		return nil, fmt.Errorf("StockPriceRepository.FindLatestByDate: %w", err)
	}
	return prices, nil
}

func (r *StockPriceRepository) FindByDateRange(stockIDs []int64, start, end time.Time) ([]model.StockPrice, error) {
	if len(stockIDs) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.Named(`SELECT * FROM stock_prices WHERE stock_id IN (:stock_ids) AND date >= :start AND date <= :end ORDER BY stock_id, date ASC`,
		map[string]interface{}{
			"stock_ids": stockIDs,
			"start":     start,
			"end":       end,
		})
	if err != nil {
		return nil, fmt.Errorf("StockPriceRepository.FindByDateRange: %w", err)
	}

	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("StockPriceRepository.FindByDateRange: %w", err)
	}
	query = r.DB.Rebind(query)

	var prices []model.StockPrice
	if err := r.DB.Select(&prices, query, args...); err != nil {
		return nil, fmt.Errorf("StockPriceRepository.FindByDateRange: %w", err)
	}
	return prices, nil
}

func (r *StockPriceRepository) GetLatestPrices(stockIDs []int64) (map[int64]float64, error) {
	if len(stockIDs) == 0 {
		return map[int64]float64{}, nil
	}

	query, args, err := sqlx.Named(`SELECT sp.stock_id, sp.close FROM stock_prices sp
		INNER JOIN (
			SELECT stock_id, MAX(date) AS max_date FROM stock_prices WHERE stock_id IN (:stock_ids) GROUP BY stock_id
		) latest ON sp.stock_id = latest.stock_id AND sp.date = latest.max_date`,
		map[string]interface{}{
			"stock_ids": stockIDs,
		})
	if err != nil {
		return nil, fmt.Errorf("StockPriceRepository.GetLatestPrices: %w", err)
	}

	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("StockPriceRepository.GetLatestPrices: %w", err)
	}
	query = r.DB.Rebind(query)

	type result struct {
		StockID int64   `db:"stock_id"`
		Close   float64 `db:"close"`
	}

	var results []result
	if err := r.DB.Select(&results, query, args...); err != nil {
		return nil, fmt.Errorf("StockPriceRepository.GetLatestPrices: %w", err)
	}

	prices := make(map[int64]float64, len(results))
	for _, res := range results {
		prices[res.StockID] = res.Close
	}
	return prices, nil
}

func (r *StockPriceRepository) GetLatestPrice(stockID int64) (float64, error) {
	prices, err := r.GetLatestPrices([]int64{stockID})
	if err != nil {
		return 0, err
	}
	if price, ok := prices[stockID]; ok {
		return price, nil
	}
	return 0, fmt.Errorf("StockPriceRepository.GetLatestPrice: no price found for stock %d", stockID)
}

type PriceWithPrev struct {
	StockID      int64   `db:"stock_id"`
	LatestClose  float64 `db:"latest_close"`
	LatestVolume int64   `db:"latest_volume"`
	PrevClose    float64 `db:"prev_close"`
}

func (r *StockPriceRepository) GetAllLatestPricesWithPrev() ([]PriceWithPrev, error) {
	query := `SELECT
		latest.stock_id,
		latest.close AS latest_close,
		latest.volume AS latest_volume,
		COALESCE(prev.close, latest.close) AS prev_close
	FROM stock_prices latest
	LEFT JOIN stock_prices prev ON prev.stock_id = latest.stock_id
		AND prev.date = (
			SELECT MAX(sp2.date) FROM stock_prices sp2
			WHERE sp2.stock_id = latest.stock_id AND sp2.date < latest.date
		)
	INNER JOIN (
		SELECT stock_id, MAX(date) AS max_date
		FROM stock_prices GROUP BY stock_id
	) lm ON latest.stock_id = lm.stock_id AND latest.date = lm.max_date`

	var results []PriceWithPrev
	if err := r.DB.Select(&results, query); err != nil {
		return nil, fmt.Errorf("StockPriceRepository.GetAllLatestPricesWithPrev: %w", err)
	}
	return results, nil
}

func (r *StockPriceRepository) GetPriceChange(stockID int64, days int) (float64, error) {
	query := `SELECT close FROM stock_prices WHERE stock_id = ? ORDER BY date DESC LIMIT ?`
	var prices []float64
	if err := r.DB.Select(&prices, query, stockID, days+1); err != nil {
		return 0, fmt.Errorf("StockPriceRepository.GetPriceChange: %w", err)
	}

	if len(prices) < 2 {
		return 0, nil
	}

	latest := prices[0]
	oldest := prices[len(prices)-1]

	if oldest == 0 {
		return 0, nil
	}

	change := ((latest - oldest) / oldest) * 100
	return change, nil
}

func (r *StockPriceRepository) GetLatestDate() (time.Time, error) {
	var latest time.Time
	err := r.DB.Get(&latest, `SELECT MAX(date) FROM stock_prices`)
	if err != nil {
		return time.Time{}, fmt.Errorf("StockPriceRepository.GetLatestDate: %w", err)
	}
	return latest, nil
}

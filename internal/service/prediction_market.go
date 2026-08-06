package service

import (
	"fmt"
	"investo/internal/model"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
)

type PredictionMarketService struct {
	DB *sqlx.DB
}

func NewPredictionMarketService(db *sqlx.DB) *PredictionMarketService {
	return &PredictionMarketService{DB: db}
}

func (s *PredictionMarketService) SubmitPrediction(userID int64, code string, price float64, dateStr string) error {
	predictedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	stock, err := s.getStockByCode(code)
	if err != nil {
		return fmt.Errorf("stock %s not found", code)
	}

	latestPrice, err := s.getLatestPrice(stock.ID)
	if err != nil {
		latestPrice = price
	}

	today := time.Now().Format("2006-01-02")
	if dateStr <= today {
		return fmt.Errorf("tanggal prediksi harus setelah hari ini")
	}

	query := `INSERT INTO price_predictions (user_id, stock_code, predicted_price, predicted_date)
		VALUES (?, ?, ?, ?)`
	_, err = s.DB.Exec(query, userID, code, price, predictedDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("submit prediction: %w", err)
	}

	_ = latestPrice
	_ = s.DB

	s.CheckAchievements(userID)

	return nil
}

func (s *PredictionMarketService) GetLeaderboard() ([]model.PredictionLeader, error) {
	query := `SELECT u.id as user_id, u.name as user_name,
		COALESCE(SUM(pp.points), 0) as total_points,
		COUNT(pp.id) as predictions,
		COALESCE(AVG(pp.accuracy * 100), 0) as avg_accuracy
		FROM users u
		LEFT JOIN price_predictions pp ON u.id = pp.user_id
		GROUP BY u.id, u.name
		HAVING predictions > 0
		ORDER BY total_points DESC
		LIMIT 10`

	var leaders []model.PredictionLeader
	if err := s.DB.Select(&leaders, query); err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}

	if leaders == nil {
		leaders = []model.PredictionLeader{}
	}
	return leaders, nil
}

func (s *PredictionMarketService) GetUserPredictions(userID int64) ([]model.PricePrediction, error) {
	query := `SELECT pp.*, u.name as user_name
		FROM price_predictions pp
		JOIN users u ON pp.user_id = u.id
		WHERE pp.user_id = ?
		ORDER BY pp.predicted_date DESC, pp.created_at DESC`

	var preds []model.PricePrediction
	if err := s.DB.Select(&preds, query, userID); err != nil {
		return nil, fmt.Errorf("get user predictions: %w", err)
	}

	for i := range preds {
		stock, err := s.getStockByCode(preds[i].StockCode)
		if err == nil {
			preds[i].StockName = stock.Name
		}
	}

	if preds == nil {
		preds = []model.PricePrediction{}
	}
	return preds, nil
}

func (s *PredictionMarketService) CalculatePoints(predicted float64, actual float64) (int, float64) {
	diff := math.Abs((predicted - actual) / actual)
	accuracy := (1 - diff) * 100

	var points int
	switch {
	case diff <= 0.01:
		points = 10
	case diff <= 0.03:
		points = 5
	case diff <= 0.05:
		points = 2
	default:
		points = 0
	}

	if accuracy > 100 {
		accuracy = 100
	}
	if accuracy < 0 {
		accuracy = 0
	}

	return points, math.Round(accuracy*100) / 100
}

func (s *PredictionMarketService) GradePredictions() error {
	today := time.Now().Format("2006-01-02")

	query := `SELECT pp.id, pp.stock_code, pp.predicted_price, pp.predicted_date
		FROM price_predictions pp
		WHERE pp.predicted_date < ? AND pp.points = 0 AND pp.actual_price IS NULL
		ORDER BY pp.predicted_date ASC`

	type rawPred struct {
		ID             int64   `db:"id"`
		StockCode      string  `db:"stock_code"`
		PredictedPrice float64 `db:"predicted_price"`
		PredictedDate  string  `db:"predicted_date"`
	}

	var preds []rawPred
	if err := s.DB.Select(&preds, query, today); err != nil {
		return fmt.Errorf("grade predictions select: %w", err)
	}

	_ = today

	for _, p := range preds {
		stock, err := s.getStockByCode(p.StockCode)
		if err != nil {
			continue
		}

		actualPrice, err := s.getPriceOnDate(stock.ID, p.PredictedDate)
		if err != nil || actualPrice <= 0 {
			continue
		}

		points, accuracy := s.CalculatePoints(p.PredictedPrice, actualPrice)

		updateQ := `UPDATE price_predictions SET actual_price = ?, accuracy = ?, points = ? WHERE id = ?`
		if _, err := s.DB.Exec(updateQ, actualPrice, accuracy, points, p.ID); err != nil {
			continue
		}
	}

	return nil
}

func (s *PredictionMarketService) getStockByCode(code string) (*struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}, error) {
	var stock struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	query := `SELECT id, name FROM stocks WHERE code = ? LIMIT 1`
	if err := s.DB.Get(&stock, query, code); err != nil {
		return nil, err
	}
	return &stock, nil
}

func (s *PredictionMarketService) getLatestPrice(stockID int64) (float64, error) {
	var price struct {
		Close float64 `db:"close"`
	}
	query := `SELECT close FROM stock_prices WHERE stock_id = ? ORDER BY date DESC LIMIT 1`
	if err := s.DB.Get(&price, query, stockID); err != nil {
		return 0, err
	}
	return price.Close, nil
}

func (s *PredictionMarketService) getPriceOnDate(stockID int64, dateStr string) (float64, error) {
	var price struct {
		Close float64 `db:"close"`
	}
	query := `SELECT close FROM stock_prices WHERE stock_id = ? AND date = ? ORDER BY date DESC LIMIT 1`
	if err := s.DB.Get(&price, query, stockID, dateStr); err != nil {
		query = `SELECT close FROM stock_prices WHERE stock_id = ? AND date <= ? ORDER BY date DESC LIMIT 1`
		if err2 := s.DB.Get(&price, query, stockID, dateStr); err2 != nil {
			return 0, err2
		}
	}
	return price.Close, nil
}

func (s *PredictionMarketService) CheckAchievements(userID int64) {
	preds, _ := s.GetUserPredictions(userID)
	checkedCount := 0
	for _, p := range preds {
		if p.Points > 0 {
			checkedCount++
		}
	}

	if checkedCount >= 5 {
		totalAccuracy := 0.0
		for _, p := range preds {
			if p.Accuracy != nil && *p.Accuracy > 0 {
				totalAccuracy += *p.Accuracy
			}
		}
		avg := totalAccuracy / float64(checkedCount)
		if avg > 80 {
			s.unlockAchievement(userID, "sharp_shooter")
		}
	}

	leaders, _ := s.GetLeaderboard()
	for i, l := range leaders {
		if l.UserID == userID && i < 10 {
			s.unlockAchievement(userID, "top_predictor")
			break
		}
	}
}

func (s *PredictionMarketService) unlockAchievement(userID int64, key string) {
	query := `INSERT IGNORE INTO user_achievements (user_id, achievement_key, progress, unlocked_at)
		VALUES (?, ?, 1, NOW())`
	s.DB.Exec(query, userID, key)
}

package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type CustomIndicatorSetting struct {
	ID        int64  `json:"id" db:"id"`
	UserID    int64  `json:"user_id" db:"user_id"`
	StockCode string `json:"stock_code" db:"stock_code"`
	Name      string `json:"name" db:"name"`
	Formula   string `json:"formula" db:"formula"`
}

type CustomIndicatorRepo struct {
	DB *sqlx.DB
}

func EnsureCustomIndicatorTable(db *sqlx.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS custom_indicators (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		stock_code VARCHAR(20) NOT NULL,
		name VARCHAR(100) NOT NULL,
		formula TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_ci_user (user_id),
		INDEX idx_ci_user_stock (user_id, stock_code)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func (r *CustomIndicatorRepo) Save(indicator *CustomIndicatorSetting) (int64, error) {
	result, err := r.DB.NamedExec(`INSERT INTO custom_indicators (user_id, stock_code, name, formula)
		VALUES (:user_id, :stock_code, :name, :formula)
		ON DUPLICATE KEY UPDATE formula = :formula, name = :name`,
		indicator)
	if err != nil {
		return 0, fmt.Errorf("Save custom indicator: %w", err)
	}
	return result.LastInsertId()
}

func (r *CustomIndicatorRepo) ListByUserAndStock(userID int64, stockCode string) ([]CustomIndicatorSetting, error) {
	var items []CustomIndicatorSetting
	err := r.DB.Select(&items,
		"SELECT * FROM custom_indicators WHERE user_id = ? AND stock_code = ? ORDER BY id DESC",
		userID, stockCode)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []CustomIndicatorSetting{}
	}
	return items, nil
}

func (r *CustomIndicatorRepo) Delete(id, userID int64) error {
	_, err := r.DB.Exec("DELETE FROM custom_indicators WHERE id = ? AND user_id = ?", id, userID)
	return err
}

func (s *CustomIndicatorService) ValidateFormula(formula string) error {
	if strings.TrimSpace(formula) == "" {
		return fmt.Errorf("formula tidak boleh kosong")
	}

	formula = strings.ToUpper(strings.TrimSpace(formula))

	allowed := []string{"SMA(", "EMA(", "RSI(", "MACD(", "BB(", "CLOSE", "C", "+", "-", "*", "/", "(", ")", ">", "<", ">=", "<=", "==", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ".", ",", " "}

	isAllowed := func(chunk string) bool {
		for _, a := range allowed {
			if strings.Contains(chunk, a) || strings.Contains(a, chunk) {
				return true
			}
		}
		return false
	}

	_ = isAllowed
	return nil
}

type SavedIndicator struct {
	Name    string    `json:"name"`
	Formula string    `json:"formula"`
	Data    []float64 `json:"data"`
}

type SavedIndicatorResponse struct {
	Name    string          `json:"name"`
	Formula string          `json:"formula"`
	Data    []interface{}   `json:"data"`
}

func ToInterfaceSlice(arr []float64) []interface{} {
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		result[i] = v
	}
	return result
}

func MarshalSavedIndicatorsJSON(saved []SavedIndicator) string {
	type resp struct {
		Name    string        `json:"name"`
		Formula string        `json:"formula"`
		Data    []interface{} `json:"data"`
	}
	var items []resp
	for _, s := range saved {
		items = append(items, resp{
			Name:    s.Name,
			Formula: s.Formula,
			Data:    ToInterfaceSlice(s.Data),
		})
	}
	b, _ := json.Marshal(items)
	return string(b)
}

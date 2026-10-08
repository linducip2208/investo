package service

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"investo/internal/repository"
)

type BasketItem struct {
	Code     string  `json:"code"`
	Quantity float64 `json:"quantity"`
	Type     string  `json:"type"`
}

type Basket struct {
	ID        int64        `json:"id" db:"id"`
	UserID    int64        `json:"user_id" db:"user_id"`
	Name      string       `json:"name" db:"name"`
	Status    string       `json:"status" db:"status"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	Items     []BasketItem `json:"items"`
}

type ConditionalOrder struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Code       string    `json:"code" db:"code"`
	Quantity   float64   `json:"quantity" db:"quantity"`
	EntryPrice float64   `json:"entry_price" db:"entry_price"`
	StopLoss   float64   `json:"stop_loss" db:"stop_loss"`
	TakeProfit float64   `json:"take_profit" db:"take_profit"`
	Status     string    `json:"status" db:"status"`
	Triggered  string    `json:"triggered" db:"triggered"`
	ExecutedAt *time.Time `json:"executed_at" db:"executed_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

type BasketTradingService struct {
	DB             *sqlx.DB
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func NewBasketTradingService(db *sqlx.DB, stockRepo *repository.StockRepository, priceRepo *repository.StockPriceRepository) *BasketTradingService {
	return &BasketTradingService{
		DB:             db,
		StockRepo:      stockRepo,
		StockPriceRepo: priceRepo,
	}
}

func EnsureBasketTables(db *sqlx.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS baskets (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		name VARCHAR(255) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

	db.Exec(`CREATE TABLE IF NOT EXISTS basket_items (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		basket_id BIGINT NOT NULL,
		stock_code VARCHAR(20) NOT NULL,
		quantity DOUBLE NOT NULL,
		type VARCHAR(10) NOT NULL,
		executed BOOLEAN NOT NULL DEFAULT FALSE,
		executed_price DOUBLE DEFAULT NULL,
		FOREIGN KEY (basket_id) REFERENCES baskets(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

	db.Exec(`CREATE TABLE IF NOT EXISTS conditional_orders (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		code VARCHAR(20) NOT NULL,
		quantity DOUBLE NOT NULL,
		entry_price DOUBLE NOT NULL,
		stop_loss DOUBLE NOT NULL,
		take_profit DOUBLE NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		triggered VARCHAR(10) DEFAULT NULL,
		executed_at TIMESTAMP NULL DEFAULT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
}

func (s *BasketTradingService) CreateBasket(userID int64, name string, stocks []BasketItem) (int64, error) {
	tx, err := s.DB.Beginx()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`INSERT INTO baskets (user_id, name, status) VALUES (?, ?, 'pending')`,
		userID, name,
	)
	if err != nil {
		return 0, fmt.Errorf("insert basket: %w", err)
	}

	basketID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get basket id: %w", err)
	}

	for _, item := range stocks {
		if item.Quantity <= 0 {
			continue
		}
		t := item.Type
		if t != "buy" && t != "sell" {
			t = "buy"
		}
		_, err := tx.Exec(
			`INSERT INTO basket_items (basket_id, stock_code, quantity, type) VALUES (?, ?, ?, ?)`,
			basketID, item.Code, item.Quantity, t,
		)
		if err != nil {
			return 0, fmt.Errorf("insert basket item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return basketID, nil
}

func (s *BasketTradingService) ExecuteBasket(basketID int64) error {
	var basket struct {
		ID     int64  `db:"id"`
		UserID int64  `db:"user_id"`
		Status string `db:"status"`
	}
	err := s.DB.Get(&basket, `SELECT id, user_id, status FROM baskets WHERE id = ?`, basketID)
	if err != nil {
		return fmt.Errorf("basket not found: %w", err)
	}
	if basket.Status != "pending" {
		return fmt.Errorf("basket already executed")
	}

	var items []struct {
		ID        int64   `db:"id"`
		StockCode string  `db:"stock_code"`
		Quantity  float64 `db:"quantity"`
		Type      string  `db:"type"`
	}
	err = s.DB.Select(&items, `SELECT id, stock_code, quantity, type FROM basket_items WHERE basket_id = ? AND executed = FALSE`, basketID)
	if err != nil {
		return fmt.Errorf("get basket items: %w", err)
	}

	if len(items) == 0 {
		return fmt.Errorf("no items to execute")
	}

	for _, item := range items {
		stock, err := s.StockRepo.FindByCode(item.StockCode)
		if err != nil {
			continue
		}

		latestPrices, err := s.StockPriceRepo.FindLatest(stock.ID, 1)
		if err != nil || len(latestPrices) == 0 {
			continue
		}

		price := latestPrices[0].Close

		_, err = s.DB.Exec(
			`UPDATE basket_items SET executed = TRUE, executed_price = ? WHERE id = ?`,
			price, item.ID,
		)
		if err != nil {
			continue
		}
	}

	_, err = s.DB.Exec(`UPDATE baskets SET status = 'executed' WHERE id = ?`, basketID)
	if err != nil {
		return fmt.Errorf("update basket status: %w", err)
	}

	return nil
}

func (s *BasketTradingService) GetBasket(basketID int64) (*Basket, error) {
	var basket Basket
	err := s.DB.Get(&basket, `SELECT id, user_id, name, status, created_at FROM baskets WHERE id = ?`, basketID)
	if err != nil {
		return nil, err
	}

	err = s.DB.Select(&basket.Items, `SELECT stock_code AS code, quantity, type FROM basket_items WHERE basket_id = ?`, basketID)
	if err != nil {
		basket.Items = []BasketItem{}
	}

	return &basket, nil
}

func (s *BasketTradingService) ListBaskets(userID int64) ([]Basket, error) {
	var baskets []Basket
	err := s.DB.Select(&baskets, `SELECT id, user_id, name, status, created_at FROM baskets WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}

	for i := range baskets {
		err = s.DB.Select(&baskets[i].Items, `SELECT stock_code AS code, quantity, type FROM basket_items WHERE basket_id = ?`, baskets[i].ID)
		if err != nil {
			baskets[i].Items = []BasketItem{}
		}
	}

	return baskets, nil
}

func (s *BasketTradingService) CreateOCO(userID int64, code string, quantity float64, entryPrice, stopLoss, takeProfit float64) (int64, error) {
	result, err := s.DB.Exec(
		`INSERT INTO conditional_orders (user_id, code, quantity, entry_price, stop_loss, take_profit, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'active')`,
		userID, code, quantity, entryPrice, stopLoss, takeProfit,
	)
	if err != nil {
		return 0, fmt.Errorf("insert oco: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *BasketTradingService) GetActiveOCO(userID int64) ([]ConditionalOrder, error) {
	var orders []ConditionalOrder
	err := s.DB.Select(&orders,
		`SELECT id, user_id, code, quantity, entry_price, stop_loss, take_profit, status, triggered, executed_at, created_at
		 FROM conditional_orders WHERE user_id = ? AND status = 'active' ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *BasketTradingService) GetOCOHistory(userID int64) ([]ConditionalOrder, error) {
	var orders []ConditionalOrder
	err := s.DB.Select(&orders,
		`SELECT id, user_id, code, quantity, entry_price, stop_loss, take_profit, status, triggered, executed_at, created_at
		 FROM conditional_orders WHERE user_id = ? AND status != 'active' ORDER BY executed_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *BasketTradingService) CheckAndExecuteOCO() error {
	var orders []ConditionalOrder
	err := s.DB.Select(&orders,
		`SELECT id, user_id, code, quantity, entry_price, stop_loss, take_profit, status FROM conditional_orders WHERE status = 'active'`)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, order := range orders {
		stock, err := s.StockRepo.FindByCode(order.Code)
		if err != nil {
			continue
		}

		latestPrices, err := s.StockPriceRepo.FindLatest(stock.ID, 1)
		if err != nil || len(latestPrices) == 0 {
			continue
		}

		currentPrice := latestPrices[0].Close

		if currentPrice >= order.TakeProfit {
			_, err = s.DB.Exec(
				`UPDATE conditional_orders SET status = 'executed', triggered = 'tp', executed_at = ? WHERE id = ?`,
				now, order.ID,
			)
		} else if currentPrice <= order.StopLoss {
			_, err = s.DB.Exec(
				`UPDATE conditional_orders SET status = 'executed', triggered = 'sl', executed_at = ? WHERE id = ?`,
				now, order.ID,
			)
		}

		if err != nil {
			continue
		}
	}

	return nil
}



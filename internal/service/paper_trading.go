package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type PaperTradingService struct {
	Repo           *repository.PaperTradingRepository
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func (s *PaperTradingService) CreatePortfolio(userID int64, name string, balance float64) (int64, error) {
	if balance <= 0 {
		balance = 100_000_000
	}
	p := &model.PaperPortfolio{
		UserID:         userID,
		Name:           name,
		InitialBalance: balance,
		CashBalance:    balance,
	}
	id, err := s.Repo.CreatePortfolio(p)
	if err != nil {
		return 0, fmt.Errorf("CreatePortfolio: %w", err)
	}
	return id, nil
}

func (s *PaperTradingService) GetPortfolios(userID int64) ([]model.PaperPortfolio, error) {
	return s.Repo.FindPortfoliosByUserID(userID)
}

func (s *PaperTradingService) ExecuteTrade(portfolioID int64, code string, tradeType string, quantity float64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if tradeType != "buy" && tradeType != "sell" {
		return fmt.Errorf("trade type must be buy or sell")
	}

	portfolio, err := s.Repo.FindPortfolioByID(portfolioID)
	if err != nil {
		return fmt.Errorf("portfolio not found: %w", err)
	}

	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return fmt.Errorf("stock %s not found: %w", code, err)
	}

	price, err := s.StockPriceRepo.GetLatestPrice(stock.ID)
	if err != nil || price <= 0 {
		return fmt.Errorf("no price data for %s", code)
	}

	total := price * quantity
	fee := total * 0.0015

	if tradeType == "buy" {
		requiredCash := total + fee
		if portfolio.CashBalance < requiredCash {
			return fmt.Errorf("insufficient cash: need Rp %.0f, have Rp %.0f", requiredCash, portfolio.CashBalance)
		}

		newCash := portfolio.CashBalance - requiredCash
		if err := s.Repo.UpdateCashBalance(portfolioID, newCash); err != nil {
			return err
		}

		existing, err := s.Repo.FindHolding(portfolioID, code)
		if err == nil && existing != nil {
			newQty := existing.Quantity + quantity
			newAvg := ((existing.AvgPrice * existing.Quantity) + (price * quantity)) / newQty
			if err := s.Repo.UpsertHolding(portfolioID, code, newQty, math.Round(newAvg)); err != nil {
				return err
			}
		} else {
			if err := s.Repo.UpsertHolding(portfolioID, code, quantity, price); err != nil {
				return err
			}
		}
	} else {
		existing, err := s.Repo.FindHolding(portfolioID, code)
		if err != nil {
			return fmt.Errorf("no holding for %s", code)
		}
		if existing.Quantity < quantity {
			return fmt.Errorf("insufficient quantity: have %.4f, want %.4f", existing.Quantity, quantity)
		}

		proceeds := total - fee
		newCash := portfolio.CashBalance + proceeds
		if err := s.Repo.UpdateCashBalance(portfolioID, newCash); err != nil {
			return err
		}

		newQty := existing.Quantity - quantity
		if newQty <= 0 {
			if err := s.Repo.DeleteHolding(portfolioID, code); err != nil {
				return fmt.Errorf("failed to clear zero holding: %w", err)
			}
		} else {
			if err := s.Repo.UpsertHolding(portfolioID, code, newQty, existing.AvgPrice); err != nil {
				return err
			}
		}
	}

	trade := &model.PaperTrade{
		PortfolioID: portfolioID,
		StockCode:   code,
		Type:        tradeType,
		Quantity:    quantity,
		Price:       price,
		Total:       total,
		Fee:         math.Round(fee),
	}
	if _, err := s.Repo.InsertTrade(trade); err != nil {
		return fmt.Errorf("failed to record trade: %w", err)
	}

	return nil
}

func (s *PaperTradingService) GetPortfolio(portfolioID int64) (*model.PaperPortfolio, error) {
	return s.Repo.FindPortfolioByID(portfolioID)
}

func (s *PaperTradingService) GetPortfolioSummary(portfolioID int64) (map[string]interface{}, error) {
	p, err := s.Repo.FindPortfolioByID(portfolioID)
	if err != nil {
		return nil, err
	}

	holdings, _ := s.GetHoldings(portfolioID)

	var marketValue float64
	for _, h := range holdings {
		marketValue += h.MarketValue
	}

	totalValue := p.CashBalance + marketValue
	totalPL := totalValue - p.InitialBalance
	totalPLPct := float64(0)
	if p.InitialBalance > 0 {
		totalPLPct = (totalPL / p.InitialBalance) * 100
	}

	return map[string]interface{}{
		"id":              p.ID,
		"name":            p.Name,
		"initial_balance": p.InitialBalance,
		"cash_balance":    p.CashBalance,
		"market_value":    marketValue,
		"total_value":     totalValue,
		"total_pl":        totalPL,
		"total_pl_pct":    totalPLPct,
		"holdings":        holdings,
	}, nil
}

func (s *PaperTradingService) GetHoldings(portfolioID int64) ([]model.PaperHolding, error) {
	holdings, err := s.Repo.FindHoldingsByPortfolioID(portfolioID)
	if err != nil {
		return nil, err
	}
	if len(holdings) == 0 {
		return []model.PaperHolding{}, nil
	}

	for i, h := range holdings {
		stock, err := s.StockRepo.FindByCode(h.StockCode)
		if err == nil {
			holdings[i].StockName = stock.Name
			price, err := s.StockPriceRepo.GetLatestPrice(stock.ID)
			if err == nil && price > 0 {
				holdings[i].CurrentPrice = price
				holdings[i].MarketValue = price * h.Quantity
				holdings[i].PL = (price - h.AvgPrice) * h.Quantity
				if h.AvgPrice > 0 {
					holdings[i].PLPercent = ((price - h.AvgPrice) / h.AvgPrice) * 100
				}
			}
		}
	}

	return holdings, nil
}

func (s *PaperTradingService) GetTradeHistory(portfolioID int64) ([]model.PaperTrade, error) {
	return s.Repo.FindTradesByPortfolioID(portfolioID, 100)
}

func (s *PaperTradingService) GetLeaderboard() ([]model.PaperLeader, error) {
	portfolios, err := s.Repo.GetAllPortfolios()
	if err != nil {
		return nil, err
	}
	if len(portfolios) == 0 {
		return []model.PaperLeader{}, nil
	}

	var leaders []model.PaperLeader
	for _, p := range portfolios {
		holdings, _ := s.Repo.FindHoldingsByPortfolioID(p.ID)
		var marketValue float64
		for _, h := range holdings {
			stock, err := s.StockRepo.FindByCode(h.StockCode)
			if err == nil {
				price, err := s.StockPriceRepo.GetLatestPrice(stock.ID)
				if err == nil {
					marketValue += price * h.Quantity
				}
			}
		}

		totalValue := p.CashBalance + marketValue
		returnPct := float64(0)
		if p.InitialBalance > 0 {
			returnPct = ((totalValue - p.InitialBalance) / p.InitialBalance) * 100
		}

		leaders = append(leaders, model.PaperLeader{
			UserID:         p.UserID,
			PortfolioID:    p.ID,
			InitialBalance: p.InitialBalance,
			CurrentValue:   totalValue,
			ReturnPct:      returnPct,
		})
	}

	sort.Slice(leaders, func(i, j int) bool {
		return leaders[i].ReturnPct > leaders[j].ReturnPct
	})

	if len(leaders) > 20 {
		leaders = leaders[:20]
	}

	return leaders, nil
}

func (s *PaperTradingService) GetUserPortfolio(userID int64) (*model.PaperPortfolio, error) {
	portfolios, err := s.Repo.FindPortfoliosByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(portfolios) == 0 {
		id, err := s.CreatePortfolio(userID, "Simulasi", 100_000_000)
		if err != nil {
			return nil, err
		}
		return s.Repo.FindPortfolioByID(id)
	}
	return &portfolios[0], nil
}

type PaymentService struct {
	Repo        *repository.PaperTradingRepository
	SettingRepo *repository.SettingRepository
	UserRepo    *repository.UserRepository
	StockRepo   *repository.StockRepository
}

func NewPaymentService(repo *repository.PaperTradingRepository, settingRepo *repository.SettingRepository, userRepo *repository.UserRepository, stockRepo *repository.StockRepository) *PaymentService {
	return &PaymentService{
		Repo:        repo,
		SettingRepo: settingRepo,
		UserRepo:    userRepo,
		StockRepo:   stockRepo,
	}
}

func (s *PaymentService) CreateTransaction(userID int64, plan string, amount float64) (string, error) {
	serverKey, err := s.SettingRepo.Get("payment_midtrans_server")
	if err != nil || serverKey == "" {
		return "", fmt.Errorf("payment gateway not configured")
	}

	orderID := fmt.Sprintf("INV-%d-%d", userID, time.Now().Unix())
	txRef := map[string]interface{}{
		"order_id": orderID,
		"plan":     plan,
		"amount":   amount,
		"user_id":  float64(userID),
		"status":   "pending",
	}

	txJSON, _ := json.Marshal(txRef)
	s.SettingRepo.Set("tx_"+orderID, string(txJSON))

	_ = serverKey
	return orderID, nil
}

func (s *PaymentService) HandleCallback(payload map[string]interface{}) error {
	orderID, ok := payload["order_id"].(string)
	if !ok {
		return fmt.Errorf("missing order_id")
	}

	status, _ := payload["transaction_status"].(string)
	if status != "settlement" && status != "capture" {
		return nil
	}

	txJSON, err := s.SettingRepo.Get("tx_" + orderID)
	if err != nil || txJSON == "" {
		return fmt.Errorf("transaction not found: %s", orderID)
	}

	var tx map[string]interface{}
	if err := json.Unmarshal([]byte(txJSON), &tx); err != nil {
		return fmt.Errorf("invalid transaction data")
	}

	plan, _ := tx["plan"].(string)
	userIDFloat, _ := tx["user_id"].(float64)
	userID := int64(userIDFloat)

	if plan != "" && userID > 0 {
		return s.ActivateSubscription(userID, plan)
	}

	return nil
}

func (s *PaymentService) ActivateSubscription(userID int64, plan string) error {
	now := time.Now()
	var expiresAt *time.Time
	if plan == "pro" {
		t := now.AddDate(0, 1, 0)
		expiresAt = &t
	} else if plan == "whitelabel" {
		t := now.AddDate(100, 0, 0)
		expiresAt = &t
	} else if plan == "free" {
		expiresAt = nil
	}

	sub := &model.Subscription{
		UserID:    userID,
		Plan:      plan,
		Status:    "active",
		StartedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := s.Repo.UpsertSubscription(sub); err != nil {
		return err
	}

	return nil
}

func (s *PaymentService) GetUserPlan(userID int64) (string, error) {
	sub, err := s.Repo.FindSubscriptionByUserID(userID)
	if err != nil {
		return "free", nil
	}

	if sub.ExpiresAt != nil && time.Now().After(*sub.ExpiresAt) {
		return "free", nil
	}

	return sub.Plan, nil
}

func (s *PaymentService) CheckAccess(userID int64, feature string) (bool, error) {
	plan, err := s.GetUserPlan(userID)
	if err != nil {
		return false, err
	}

	proFeatures := map[string]bool{
		"ai_agents":      true,
		"ai_signals":     true,
		"backtesting":    true,
		"screener_adv":   true,
		"api_access":     true,
		"all_indicators": true,
		"telegram_bot":   true,
		"real_time":      true,
		"paper_trading":  true,
	}
	whitelabelFeatures := map[string]bool{
		"source_code":      true,
		"api_key_manage":   true,
		"custom_domain":    true,
		"white_label":      true,
		"priority_support": true,
	}

	switch plan {
	case "whitelabel":
		return true, nil
	case "pro":
		if whitelabelFeatures[feature] {
			return false, nil
		}
		return true, nil
	default:
		if proFeatures[feature] || whitelabelFeatures[feature] {
			return false, nil
		}
		return true, nil
	}
}

func (s *PaymentService) UpgradePlan(userID int64, plan string) error {
	validPlans := map[string]bool{"free": true, "pro": true, "whitelabel": true}
	if !validPlans[plan] {
		return fmt.Errorf("invalid plan: %s", plan)
	}
	return s.ActivateSubscription(userID, plan)
}

func (s *PaymentService) GenerateInvoice(userID int64, transactionID string) (*model.Invoice, error) {
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	txJSON, err := s.SettingRepo.Get("tx_" + transactionID)
	if err != nil || txJSON == "" {
		return nil, fmt.Errorf("transaction not found")
	}

	var tx map[string]interface{}
	if err := json.Unmarshal([]byte(txJSON), &tx); err != nil {
		return nil, fmt.Errorf("invalid transaction data")
	}

	plan, _ := tx["plan"].(string)
	amount, _ := tx["amount"].(float64)
	status, _ := tx["status"].(string)

	planNames := map[string]string{"free": "Free", "pro": "Pro (Bulanan)", "whitelabel": "Whitelabel (Sekali Bayar)"}
	planName := planNames[plan]
	if planName == "" {
		planName = plan
	}

	inv := &model.Invoice{
		Number:        transactionID,
		Date:          time.Now().Format("02 Jan 2006"),
		CustomerName:  user.Name,
		CustomerEmail: user.Email,
		Plan:          plan,
		Status:        status,
		Items: []model.InvoiceItem{
			{Description: "Paket Investo " + planName, Qty: 1, Amount: amount},
		},
		Total: amount,
	}

	return inv, nil
}

package service

import (
	"bytes"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

var (
	ErrPaymentNotConfigured = errors.New("payment provider not configured")
	ErrInvalidSignature     = errors.New("invalid callback signature")
)

var orderIDPattern = regexp.MustCompile(`^INV-\d+-\d+$`)

var fallbackPlans = map[string]bool{"free": true, "pro": true, "enterprise": true, "whitelabel": true}

const (
	midtransSandboxURL    = "https://app.sandbox.midtrans.com/snap/v1/transactions"
	midtransProductionURL = "https://app.midtrans.com/snap/v1/transactions"
)

func ParseOrderID(orderID string) (int64, int64, error) {
	if !orderIDPattern.MatchString(orderID) {
		return 0, 0, fmt.Errorf("invalid order_id format")
	}
	parts := strings.Split(orderID, "-")
	userID, err1 := strconv.ParseInt(parts[1], 10, 64)
	ts, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || userID <= 0 || ts <= 0 {
		return 0, 0, fmt.Errorf("invalid order_id format")
	}
	return userID, ts, nil
}

func ComputeMidtransSignature(orderID, statusCode, grossAmount, serverKey string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	return hex.EncodeToString(sum[:])
}

func VerifyMidtransSignature(orderID, statusCode, grossAmount, serverKey, signature string) bool {
	if orderID == "" || serverKey == "" || signature == "" {
		return false
	}
	expected := ComputeMidtransSignature(orderID, statusCode, grossAmount, serverKey)
	got := strings.ToLower(strings.TrimSpace(signature))
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}

func (s *PaymentService) midtransServerKey() string {
	if s.SettingRepo == nil {
		return ""
	}
	if v, err := s.SettingRepo.Get("payment_midtrans_server_key"); err == nil && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if v, err := s.SettingRepo.Get("payment_midtrans_server"); err == nil {
		return strings.TrimSpace(v)
	}
	return ""
}

func (s *PaymentService) midtransMode() string {
	if s.SettingRepo == nil {
		return "sandbox"
	}
	if v, err := s.SettingRepo.Get("payment_midtrans_mode"); err == nil && strings.EqualFold(strings.TrimSpace(v), "production") {
		return "production"
	}
	return "sandbox"
}

func (s *PaymentService) activePlans() map[string]bool {
	if s.Repo != nil && s.Repo.DB != nil {
		var codes []string
		if err := s.Repo.DB.Select(&codes, `SELECT code FROM subscription_plans WHERE is_active = 1`); err == nil && len(codes) > 0 {
			plans := make(map[string]bool, len(codes))
			for _, c := range codes {
				if c = strings.ToLower(strings.TrimSpace(c)); c != "" {
					plans[c] = true
				}
			}
			if len(plans) > 0 {
				return plans
			}
		}
	}
	plans := make(map[string]bool, len(fallbackPlans))
	for k, v := range fallbackPlans {
		plans[k] = v
	}
	return plans
}

func (s *PaymentService) isValidPlan(plan string) bool {
	return s.activePlans()[strings.ToLower(strings.TrimSpace(plan))]
}

func (s *PaymentService) loadTx(orderID string) (map[string]interface{}, error) {
	if s.SettingRepo == nil {
		return nil, fmt.Errorf("transaction store not available")
	}
	txJSON, err := s.SettingRepo.Get("tx_" + orderID)
	if err != nil || txJSON == "" {
		return nil, fmt.Errorf("transaction not found: %s", orderID)
	}
	var tx map[string]interface{}
	if err := json.Unmarshal([]byte(txJSON), &tx); err != nil {
		return nil, fmt.Errorf("invalid transaction data")
	}
	return tx, nil
}

func (s *PaymentService) storeTx(orderID string, tx map[string]interface{}) error {
	if s.SettingRepo == nil {
		return fmt.Errorf("transaction store not available")
	}
	txJSON, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("encode transaction: %w", err)
	}
	return s.SettingRepo.Set("tx_"+orderID, string(txJSON))
}

func (s *PaymentService) GetSnapDetails(orderID string) (string, string) {
	tx, err := s.loadTx(orderID)
	if err != nil {
		return "", ""
	}
	token, _ := tx["snap_token"].(string)
	redirectURL, _ := tx["snap_redirect_url"].(string)
	return token, redirectURL
}

func planDisplayName(plan string) string {
	switch plan {
	case "pro":
		return "Pro (Bulanan)"
	case "whitelabel":
		return "Whitelabel (Sekali Bayar)"
	case "enterprise":
		return "Enterprise (Bulanan)"
	case "free":
		return "Free"
	default:
		return plan
	}
}

func truncateForError(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		s = s[:n] + "..."
	}
	return s
}

func (s *PaymentService) chargeSnap(orderID, plan string, amount float64, userID int64, serverKey string) (string, string, error) {
	snapURL := midtransSandboxURL
	if s.midtransMode() == "production" {
		snapURL = midtransProductionURL
	}

	gross := int64(math.Round(amount))
	if gross <= 0 {
		return "", "", fmt.Errorf("invalid amount for snap charge")
	}

	firstName, email := "", ""
	if s.UserRepo != nil {
		if u, err := s.UserRepo.FindByID(userID); err == nil && u != nil {
			firstName, email = u.Name, u.Email
		}
	}

	reqBody := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     orderID,
			"gross_amount": gross,
		},
		"item_details": []map[string]interface{}{
			{"id": plan, "price": gross, "quantity": 1, "name": "Paket Investo " + planDisplayName(plan)},
		},
		"customer_details": map[string]interface{}{
			"first_name": firstName,
			"email":      email,
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("snap charge encode: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, snapURL, bytes.NewReader(raw))
	if err != nil {
		return "", "", fmt.Errorf("snap charge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Investo/1.0")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(serverKey+":")))

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > 2 {
				return fmt.Errorf("snap: too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("snap charge failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", fmt.Errorf("snap charge read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("snap charge rejected: status %d: %s", resp.StatusCode, truncateForError(respBody, 300))
	}

	var snapResp struct {
		Token       string `json:"token"`
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return "", "", fmt.Errorf("snap charge decode: %w", err)
	}
	if snapResp.Token == "" || snapResp.RedirectURL == "" {
		return "", "", fmt.Errorf("snap charge returned incomplete response")
	}
	return snapResp.Token, snapResp.RedirectURL, nil
}

func callbackString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		return ""
	}
}

func txUserID(tx map[string]interface{}) int64 {
	switch v := tx["user_id"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func txAmount(tx map[string]interface{}) float64 {
	switch v := tx["amount"].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return 0
}

func (s *PaymentService) txPlanAmount(orderID string) (string, float64) {
	tx, err := s.loadTx(orderID)
	if err != nil {
		return "", 0
	}
	plan, _ := tx["plan"].(string)
	return strings.ToLower(strings.TrimSpace(plan)), txAmount(tx)
}

func marshalCallback(payload map[string]interface{}) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	if len(raw) > 32768 {
		raw = raw[:32768]
	}
	return string(raw)
}

func (s *PaymentService) paymentRepo() *repository.PaymentRepository {
	if s.Repo == nil || s.Repo.DB == nil {
		return nil
	}
	return repository.NewPaymentRepository(s.Repo.DB)
}

func (s *PaymentService) recordPayment(orderID, plan string, userID int64, amount float64, status, raw string, paidAt *time.Time) {
	repo := s.paymentRepo()
	if repo == nil {
		return
	}
	_ = repo.UpsertByOrderID(&model.Payment{
		UserID:      userID,
		OrderID:     orderID,
		Plan:        plan,
		Amount:      int64(math.Round(amount)),
		Status:      status,
		RawCallback: raw,
		PaidAt:      paidAt,
	})
}

func (s *PaymentService) CreateTransaction(userID int64, plan string, amount float64) (string, error) {
	plan = strings.ToLower(strings.TrimSpace(plan))
	if !s.isValidPlan(plan) {
		return "", fmt.Errorf("invalid plan: %s", plan)
	}
	if userID <= 0 {
		return "", fmt.Errorf("invalid user")
	}
	if amount < 0 {
		return "", fmt.Errorf("invalid amount")
	}

	serverKey := s.midtransServerKey()
	orderID := fmt.Sprintf("INV-%d-%d", userID, time.Now().Unix())
	txRef := map[string]interface{}{
		"order_id":   orderID,
		"plan":       plan,
		"amount":     amount,
		"user_id":    float64(userID),
		"status":     "pending",
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	if serverKey == "" {
		txRef["status"] = "pending_setup"
		_ = s.storeTx(orderID, txRef)
		return "", fmt.Errorf("%w: snap charge unavailable", ErrPaymentNotConfigured)
	}

	if err := s.storeTx(orderID, txRef); err != nil {
		return "", fmt.Errorf("failed to persist transaction: %w", err)
	}

	token, redirectURL, err := s.chargeSnap(orderID, plan, amount, userID, serverKey)
	if err != nil {
		txRef["status"] = "snap_failed"
		_ = s.storeTx(orderID, txRef)
		return "", err
	}

	txRef["snap_token"] = token
	txRef["snap_redirect_url"] = redirectURL
	_ = s.storeTx(orderID, txRef)
	return orderID, nil
}

func (s *PaymentService) HandleCallback(payload map[string]interface{}) error {
	if payload == nil {
		return fmt.Errorf("missing order_id")
	}
	orderID := callbackString(payload["order_id"])
	orderUserID, _, err := ParseOrderID(orderID)
	if err != nil {
		return fmt.Errorf("invalid order_id")
	}

	serverKey := s.midtransServerKey()
	if serverKey == "" {
		s.recordPayment(orderID, "", orderUserID, 0, "failed", marshalCallback(payload), nil)
		return ErrPaymentNotConfigured
	}

	statusCode := callbackString(payload["status_code"])
	grossAmount := callbackString(payload["gross_amount"])
	signature := callbackString(payload["signature_key"])
	if statusCode == "" || grossAmount == "" {
		return fmt.Errorf("incomplete callback payload")
	}
	if !VerifyMidtransSignature(orderID, statusCode, grossAmount, serverKey, signature) {
		plan, amount := s.txPlanAmount(orderID)
		s.recordPayment(orderID, plan, orderUserID, amount, "failed", marshalCallback(payload), nil)
		return ErrInvalidSignature
	}

	tx, err := s.loadTx(orderID)
	if err != nil {
		return fmt.Errorf("transaction not found: %s", orderID)
	}

	plan, _ := tx["plan"].(string)
	plan = strings.ToLower(strings.TrimSpace(plan))
	if !s.isValidPlan(plan) {
		s.recordPayment(orderID, plan, orderUserID, txAmount(tx), "failed", marshalCallback(payload), nil)
		return fmt.Errorf("invalid plan: %s", plan)
	}

	if txUID := txUserID(tx); txUID != orderUserID {
		return fmt.Errorf("order user mismatch")
	}

	if g, err := strconv.ParseFloat(strings.TrimSpace(grossAmount), 64); err == nil {
		if amt, ok := tx["amount"].(float64); ok && math.Abs(g-amt) > 0.5 {
			s.recordPayment(orderID, plan, orderUserID, amt, "failed", marshalCallback(payload), nil)
			return fmt.Errorf("amount mismatch")
		}
	}

	txnStatus := callbackString(payload["transaction_status"])
	fraudStatus := callbackString(payload["fraud_status"])

	payStatus := "pending"
	activate := false
	switch txnStatus {
	case "settlement":
		payStatus = "paid"
		activate = true
	case "capture":
		switch fraudStatus {
		case "accept":
			payStatus = "paid"
			activate = true
		case "deny":
			payStatus = "failed"
		default:
			payStatus = "pending"
		}
	case "pending":
		payStatus = "pending"
	case "deny", "expire", "cancel", "failure":
		payStatus = "failed"
	default:
		payStatus = "pending"
	}

	tx["status"] = payStatus
	tx["transaction_status"] = txnStatus
	tx["fraud_status"] = fraudStatus
	_ = s.storeTx(orderID, tx)

	amount := txAmount(tx)
	var paidAt *time.Time
	if payStatus == "paid" {
		now := time.Now()
		paidAt = &now
	}
	s.recordPayment(orderID, plan, orderUserID, amount, payStatus, marshalCallback(payload), paidAt)

	if activate {
		return s.ActivateSubscription(orderUserID, plan)
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
	plan = strings.ToLower(strings.TrimSpace(plan))
	if !s.isValidPlan(plan) {
		return fmt.Errorf("invalid plan: %s", plan)
	}
	return s.ActivateSubscription(userID, plan)
}

func (s *PaymentService) GenerateInvoice(requesterID int64, transactionID string, isAdmin bool) (*model.Invoice, error) {
	ownerID, _, err := ParseOrderID(transactionID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found")
	}
	if !isAdmin && ownerID != requesterID {
		return nil, fmt.Errorf("transaction not found")
	}

	if s.UserRepo == nil {
		return nil, fmt.Errorf("user not found")
	}
	user, err := s.UserRepo.FindByID(ownerID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	tx, err := s.loadTx(transactionID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found")
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

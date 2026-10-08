package service

import (
	"fmt"
	"math"

	"investo/internal/model"
	"investo/internal/repository"
)

// SimulatedExchangeService auto-executes multi-agent decisions into a paper-trading
// portfolio, mirroring TradingAgents' simulated-exchange execution loop.
type SimulatedExchangeService struct {
	PaperRepo    *repository.PaperTradingRepository
	SettingRepo  *repository.SettingRepository
	ApprovalRepo *repository.ApprovalRepository
}

func NewSimulatedExchangeService(paperRepo *repository.PaperTradingRepository, settingRepo *repository.SettingRepository) *SimulatedExchangeService {
	return &SimulatedExchangeService{PaperRepo: paperRepo, SettingRepo: settingRepo}
}

// approvalThreshold returns the amount (IDR) above which a BUY requires approval.
func (s *SimulatedExchangeService) approvalThreshold() float64 {
	if s.SettingRepo != nil {
		if v, err := s.SettingRepo.Get("approval_threshold"); err == nil && v != "" {
			var t float64
			if _, err := fmt.Sscanf(v, "%f", &t); err == nil && t > 0 {
				return t
			}
		}
	}
	return 100_000_000
}

func (s *SimulatedExchangeService) IsEnabled() bool {
	if s.SettingRepo == nil {
		return false
	}
	v, err := s.SettingRepo.Get("ai_auto_execute")
	return err == nil && v == "1"
}

func (s *SimulatedExchangeService) ExecuteDecision(decision *MultiAgentDecision, userID int64) (*model.PaperTrade, error) {
	if !s.IsEnabled() || s.PaperRepo == nil {
		return nil, nil
	}
	if decision == nil || (decision.FinalSignal != "BUY" && decision.FinalSignal != "SELL") {
		return nil, nil
	}
	if decision.PositionPct <= 0 {
		return nil, nil
	}

	portfolio, err := s.getOrCreatePortfolio(userID)
	if err != nil {
		return nil, err
	}

	if decision.FinalSignal == "BUY" {
		return s.executeBuy(portfolio, decision)
	}
	return s.executeSell(portfolio, decision)
}

func (s *SimulatedExchangeService) getOrCreatePortfolio(userID int64) (*model.PaperPortfolio, error) {
	portfolios, err := s.PaperRepo.FindPortfoliosByUserID(userID)
	if err == nil && len(portfolios) > 0 {
		return &portfolios[0], nil
	}

	p := &model.PaperPortfolio{
		UserID:         userID,
		Name:           "AI Auto-Trading",
		InitialBalance: 100_000_000,
		CashBalance:    100_000_000,
	}
	id, err := s.PaperRepo.CreatePortfolio(p)
	if err != nil {
		return nil, fmt.Errorf("SimulatedExchangeService: create portfolio: %w", err)
	}
	p.ID = id
	return p, nil
}

func (s *SimulatedExchangeService) executeBuy(portfolio *model.PaperPortfolio, d *MultiAgentDecision) (*model.PaperTrade, error) {
	price := d.EntryPrice
	if price <= 0 {
		price = d.TargetPrice * 0.9
	}
	if price <= 0 {
		return nil, nil
	}

	allocated := portfolio.CashBalance * (d.PositionPct / 100.0)
	if allocated <= 0 {
		return nil, nil
	}

	// Approval workflow: large orders require manual approval.
	if s.ApprovalRepo != nil && allocated > s.approvalThreshold() {
		req := &model.ApprovalRequest{
			UserID:      portfolio.UserID,
			Type:        "paper_buy",
			Amount:      allocated,
			Description: fmt.Sprintf("Auto-BUY %s senilai Rp %.0f (entry Rp %.0f)", d.Ticker, allocated, price),
			Status:      "pending",
		}
		if _, err := s.ApprovalRepo.Create(req); err != nil {
			return nil, fmt.Errorf("SimulatedExchangeService: create approval: %w", err)
		}
		return nil, nil
	}

	fee := allocated * 0.0015
	net := allocated - fee
	quantity := net / price
	if quantity <= 0 {
		return nil, nil
	}

	trade := &model.PaperTrade{
		PortfolioID: portfolio.ID,
		StockCode:   d.Ticker,
		Type:        "BUY",
		Quantity:    math.Floor(quantity*100) / 100,
		Price:       price,
		Total:       allocated,
		Fee:         fee,
	}

	// Deduct cash, record trade, upsert holding.
	if _, err := s.PaperRepo.InsertTrade(trade); err != nil {
		return nil, err
	}

	holding, _ := s.PaperRepo.FindHolding(portfolio.ID, d.Ticker)
	var newQty, newAvg float64
	if holding != nil && holding.Quantity > 0 {
		newQty = holding.Quantity + trade.Quantity
		newAvg = ((holding.Quantity * holding.AvgPrice) + (trade.Quantity * trade.Price)) / newQty
	} else {
		newQty = trade.Quantity
		newAvg = trade.Price
	}
	if err := s.PaperRepo.UpsertHolding(portfolio.ID, d.Ticker, newQty, newAvg); err != nil {
		return nil, err
	}

	newCash := portfolio.CashBalance - trade.Total
	if err := s.PaperRepo.UpdateCashBalance(portfolio.ID, newCash); err != nil {
		return nil, err
	}

	return trade, nil
}

func (s *SimulatedExchangeService) executeSell(portfolio *model.PaperPortfolio, d *MultiAgentDecision) (*model.PaperTrade, error) {
	holding, err := s.PaperRepo.FindHolding(portfolio.ID, d.Ticker)
	if err != nil || holding == nil || holding.Quantity <= 0 {
		return nil, nil
	}

	price := d.EntryPrice
	if price <= 0 {
		price = d.StopLoss
	}
	if price <= 0 {
		return nil, nil
	}

	gross := holding.Quantity * price
	fee := gross * 0.0015
	trade := &model.PaperTrade{
		PortfolioID: portfolio.ID,
		StockCode:   d.Ticker,
		Type:        "SELL",
		Quantity:    holding.Quantity,
		Price:       price,
		Total:       gross - fee,
		Fee:         fee,
	}

	if _, err := s.PaperRepo.InsertTrade(trade); err != nil {
		return nil, err
	}
	if err := s.PaperRepo.DeleteHolding(portfolio.ID, d.Ticker); err != nil {
		return nil, err
	}
	if err := s.PaperRepo.UpdateCashBalance(portfolio.ID, portfolio.CashBalance+(gross-fee)); err != nil {
		return nil, err
	}

	return trade, nil
}

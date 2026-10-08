package service

import (
	"encoding/json"
	"fmt"
	"time"

	"investo/internal/repository"
)

type SignalManager struct {
	AIService   *AIService
	BEIPipeline *BEIAgentPipeline
	ForexPipeline *ForexAgentPipeline
	RiskManager *UnifiedRiskManager
	SignalRepo  *repository.AgentDecisionRepository
}

type FinalSignal struct {
	Instrument   string             `json:"instrument"`
	MarketType   string             `json:"market_type"`
	Signal       string             `json:"signal"`
	Confidence   float64            `json:"confidence"`
	PositionPct  float64            `json:"position_pct"`
	EntryPrice   float64            `json:"entry_price"`
	StopLoss     float64            `json:"stop_loss"`
	TakeProfit   float64            `json:"take_profit"`
	Rationale    string             `json:"rationale"`
	AgentScores  map[string]float64 `json:"agent_scores"`
	RiskWarnings []string           `json:"risk_warnings"`
	Timestamp    time.Time          `json:"timestamp"`
}

func NewSignalManager(
	ai *AIService,
	beiPipeline *BEIAgentPipeline,
	forexPipeline *ForexAgentPipeline,
	riskManager *UnifiedRiskManager,
	signalRepo *repository.AgentDecisionRepository,
) *SignalManager {
	return &SignalManager{
		AIService:     ai,
		BEIPipeline:   beiPipeline,
		ForexPipeline: forexPipeline,
		RiskManager:   riskManager,
		SignalRepo:    signalRepo,
	}
}

func (sm *SignalManager) ProcessSignal(instrument string, marketType string) (*FinalSignal, error) {
	var signal *FinalSignal
	var err error

	switch marketType {
	case "BEI":
		signal, err = sm.processBEISignal(instrument)
	case "FOREX":
		signal, err = sm.processForexSignal(instrument)
	default:
		return nil, fmt.Errorf("market type tidak dikenal: %s", marketType)
	}

	if err != nil {
		return nil, err
	}

	signal.Timestamp = time.Now()

	sm.saveToDecisionLog(signal)

	return signal, nil
}

func (sm *SignalManager) processBEISignal(code string) (*FinalSignal, error) {
	result, err := sm.BEIPipeline.Analyze(code)
	if err != nil {
		return nil, fmt.Errorf("BEI analysis failed: %w", err)
	}

	risk, err := sm.RiskManager.AssessRisk(code, result.Confidence, "BEI", 0)
	if err != nil {
		risk = &RiskAssessment{Warnings: []string{}}
	}

	signal := &FinalSignal{
		Instrument:  code,
		MarketType:  "BEI",
		Signal:      result.CompositeSignal,
		Confidence:  result.Confidence,
		PositionPct: risk.MaxPositionPct,
		EntryPrice:  result.CurrentPrice,
		StopLoss:    result.StopLoss,
		TakeProfit:  result.TargetPrice,
		Rationale:   result.Rationale,
		AgentScores: map[string]float64{
			"fundamental":   result.FundamentalScore,
			"foreign_flow":  result.ForeignFlowScore,
			"sentiment":     result.SentimentScore,
			"technical":     result.TechnicalScore,
		},
		RiskWarnings: risk.Warnings,
	}

	return signal, nil
}

func (sm *SignalManager) processForexSignal(pair string) (*FinalSignal, error) {
	result, err := sm.ForexPipeline.Analyze(pair)
	if err != nil {
		return nil, fmt.Errorf("forex analysis failed: %w", err)
	}

	risk, err := sm.RiskManager.AssessRisk(pair, result.Confidence, "FOREX", 0)
	if err != nil {
		risk = &RiskAssessment{Warnings: []string{}}
	}

	signal := &FinalSignal{
		Instrument:  pair,
		MarketType:  "FOREX",
		Signal:      result.CompositeSignal,
		Confidence:  result.Confidence,
		PositionPct: risk.MaxPositionPct,
		EntryPrice:  result.CurrentRate,
		StopLoss:    result.StopLoss,
		TakeProfit:  result.TargetRate,
		Rationale:   result.Rationale,
		AgentScores: map[string]float64{
			"technical": result.TechnicalScore,
			"macro":     result.MacroScore,
			"sentiment": result.SentimentScore,
		},
		RiskWarnings: risk.Warnings,
	}

	return signal, nil
}

func (sm *SignalManager) saveToDecisionLog(signal *FinalSignal) {
	if sm.SignalRepo == nil {
		return
	}

	scoresJSON, _ := json.Marshal(signal.AgentScores)

	decision := map[string]interface{}{
		"ticker":       signal.Instrument,
		"decision_date": signal.Timestamp,
		"final_signal": signal.Signal,
		"entry_price":  signal.EntryPrice,
		"target_price": signal.TakeProfit,
		"stop_loss":    signal.StopLoss,
		"position_pct": signal.PositionPct * 100,
		"confidence":   signal.Confidence,
		"risk_score":   50.0,
		"reports_json": string(scoresJSON),
		"rationale":    signal.Rationale,
	}
	_ = decision
}

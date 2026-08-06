package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"investo/internal/model"
)

type AgentDecisionRepository struct {
	DB *sqlx.DB
}

func (r *AgentDecisionRepository) Save(d *model.AgentDecision) error {
	query := `INSERT INTO agent_decisions
		(ticker, decision_date, final_signal, entry_price, target_price, stop_loss, position_pct, confidence, risk_score, reports_json, debate_log, rationale, outcome, realized_return, reflection)
		VALUES (:ticker, :decision_date, :final_signal, :entry_price, :target_price, :stop_loss, :position_pct, :confidence, :risk_score, :reports_json, :debate_log, :rationale, :outcome, :realized_return, :reflection)
		ON DUPLICATE KEY UPDATE
			final_signal = VALUES(final_signal),
			entry_price = VALUES(entry_price),
			target_price = VALUES(target_price),
			stop_loss = VALUES(stop_loss),
			position_pct = VALUES(position_pct),
			confidence = VALUES(confidence),
			risk_score = VALUES(risk_score),
			reports_json = VALUES(reports_json),
			debate_log = VALUES(debate_log),
			rationale = VALUES(rationale),
			outcome = VALUES(outcome),
			realized_return = VALUES(realized_return),
			reflection = VALUES(reflection)`

	if d.DecisionDate.IsZero() {
		d.DecisionDate = time.Now()
	}

	_, err := r.DB.NamedExec(query, d)
	if err != nil {
		return fmt.Errorf("AgentDecisionRepository.Save: %w", err)
	}
	return nil
}

func (r *AgentDecisionRepository) FindByTicker(ticker string, limit int) ([]model.AgentDecision, error) {
	var decisions []model.AgentDecision
	query := `SELECT * FROM agent_decisions WHERE ticker = ? ORDER BY decision_date DESC LIMIT ?`
	if err := r.DB.Select(&decisions, query, ticker, limit); err != nil {
		return nil, fmt.Errorf("AgentDecisionRepository.FindByTicker: %w", err)
	}
	return decisions, nil
}

func (r *AgentDecisionRepository) GetPastReflections(ticker string) ([]string, error) {
	var reflections []string
	query := `SELECT reflection FROM agent_decisions WHERE ticker = ? AND reflection IS NOT NULL AND reflection != '' ORDER BY decision_date DESC LIMIT 5`
	if err := r.DB.Select(&reflections, query, ticker); err != nil {
		return nil, fmt.Errorf("AgentDecisionRepository.GetPastReflections: %w", err)
	}
	return reflections, nil
}

func (r *AgentDecisionRepository) UpdateOutcome(id int64, outcome string, realizedReturn float64, reflection string) error {
	query := `UPDATE agent_decisions SET outcome = ?, realized_return = ?, reflection = ? WHERE id = ?`
	_, err := r.DB.Exec(query, outcome, realizedReturn, reflection, id)
	if err != nil {
		return fmt.Errorf("AgentDecisionRepository.UpdateOutcome: %w", err)
	}
	return nil
}

func (r *AgentDecisionRepository) FindLatestByTicker(ticker string) (*model.AgentDecision, error) {
	var d model.AgentDecision
	query := `SELECT * FROM agent_decisions WHERE ticker = ? ORDER BY decision_date DESC LIMIT 1`
	if err := r.DB.Get(&d, query, ticker); err != nil {
		return nil, fmt.Errorf("AgentDecisionRepository.FindLatestByTicker: %w", err)
	}
	return &d, nil
}

package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"investo/internal/model"
)

type AgentMandateRepository struct {
	DB *sqlx.DB
}

func (r *AgentMandateRepository) Create(m *model.AgentMandate) (int64, error) {
	query := `INSERT INTO agent_mandates (user_id, name, persona_style, debate_rounds, risk_profile, max_position, tickers_json)
		VALUES (:user_id, :name, :persona_style, :debate_rounds, :risk_profile, :max_position, :tickers_json)`
	result, err := r.DB.NamedExec(query, m)
	if err != nil {
		return 0, fmt.Errorf("AgentMandateRepository.Create: %w", err)
	}
	return result.LastInsertId()
}

func (r *AgentMandateRepository) FindByID(id int64) (*model.AgentMandate, error) {
	var m model.AgentMandate
	if err := r.DB.Get(&m, `SELECT * FROM agent_mandates WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("AgentMandateRepository.FindByID: %w", err)
	}
	return &m, nil
}

func (r *AgentMandateRepository) FindByUserID(userID int64) ([]model.AgentMandate, error) {
	var mandates []model.AgentMandate
	query := `SELECT * FROM agent_mandates WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&mandates, query, userID); err != nil {
		return nil, fmt.Errorf("AgentMandateRepository.FindByUserID: %w", err)
	}
	return mandates, nil
}

func (r *AgentMandateRepository) Delete(id int64) error {
	_, err := r.DB.Exec(`DELETE FROM agent_mandates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("AgentMandateRepository.Delete: %w", err)
	}
	return nil
}

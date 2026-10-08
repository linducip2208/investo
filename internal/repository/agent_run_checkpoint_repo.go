package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"investo/internal/model"
)

type AgentRunCheckpointRepository struct {
	DB *sqlx.DB
}

func (r *AgentRunCheckpointRepository) Save(ticker string, runDate time.Time, phase, reportsJSON, debateLog string) error {
	query := `INSERT INTO agent_run_checkpoints (ticker, run_date, phase, reports_json, debate_log)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE reports_json = VALUES(reports_json), debate_log = VALUES(debate_log)`

	if runDate.IsZero() {
		runDate = time.Now()
	}

	if _, err := r.DB.Exec(query, ticker, runDate.Format("2006-01-02"), phase, reportsJSON, debateLog); err != nil {
		return fmt.Errorf("AgentRunCheckpointRepository.Save: %w", err)
	}
	return nil
}

func (r *AgentRunCheckpointRepository) Find(ticker string, runDate time.Time, phase string) (*model.AgentRunCheckpoint, error) {
	var cp model.AgentRunCheckpoint
	query := `SELECT * FROM agent_run_checkpoints WHERE ticker = ? AND run_date = ? AND phase = ? LIMIT 1`
	if err := r.DB.Get(&cp, query, ticker, runDate.Format("2006-01-02"), phase); err != nil {
		return nil, fmt.Errorf("AgentRunCheckpointRepository.Find: %w", err)
	}
	return &cp, nil
}

func (r *AgentRunCheckpointRepository) Clear(ticker string, runDate time.Time) error {
	_, err := r.DB.Exec(`DELETE FROM agent_run_checkpoints WHERE ticker = ? AND run_date = ?`, ticker, runDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("AgentRunCheckpointRepository.Clear: %w", err)
	}
	return nil
}

package service

import (
	"testing"
)

func TestSimulatedExchangeDefaults(t *testing.T) {
	s := &SimulatedExchangeService{} // nil repos

	if s.IsEnabled() {
		t.Error("IsEnabled with nil SettingRepo should be false")
	}
	if s.approvalThreshold() != 100_000_000 {
		t.Errorf("approvalThreshold default = %v, want 100000000", s.approvalThreshold())
	}
}

func TestExecuteDecisionNilGuards(t *testing.T) {
	s := &SimulatedExchangeService{} // nil repos -> not enabled

	if trade, err := s.ExecuteDecision(nil, 1); trade != nil || err != nil {
		t.Errorf("ExecuteDecision(nil) should return nil,nil, got %v,%v", trade, err)
	}

	d := &MultiAgentDecision{FinalSignal: "BUY", PositionPct: 10}
	if trade, err := s.ExecuteDecision(d, 1); trade != nil || err != nil {
		t.Errorf("ExecuteDecision with disabled exchange should return nil,nil, got %v,%v", trade, err)
	}
}

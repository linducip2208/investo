package service

import (
	"math"
	"testing"
)

func TestCalcGraham(t *testing.T) {
	s := &ValuationService{}
	v := s.calcGraham(500, 2000)
	// sqrt(22.5 * 500 * 2000) = sqrt(22,500,000) = 4743.42
	expected := math.Sqrt(22.5 * 500 * 2000)
	if math.Abs(v-expected) > 0.01 {
		t.Errorf("calcGraham = %f, want %f", v, expected)
	}

	if s.calcGraham(0, 2000) != 0 {
		t.Error("calcGraham with eps=0 should return 0")
	}
	if s.calcGraham(500, 0) != 0 {
		t.Error("calcGraham with bvps=0 should return 0")
	}
}

func TestCalcLynch(t *testing.T) {
	s := &ValuationService{}
	if v := s.calcLynch(100); v != 1200 {
		t.Errorf("calcLynch(100) = %f, want 1200", v)
	}
	if s.calcLynch(0) != 0 {
		t.Error("calcLynch with eps=0 should return 0")
	}
}

func TestCalcDDM(t *testing.T) {
	s := &ValuationService{}
	v := s.calcDDM(100, 0)
	// payout 0.4 -> dividend 40, required 0.12, growth 0.05
	// value = 40 * 1.05 / 0.07 = 600
	if math.Abs(v-600) > 0.01 {
		t.Errorf("calcDDM(100, 0) = %f, want 600", v)
	}

	if s.calcDDM(0, 0) != 0 {
		t.Error("calcDDM with eps=0 should return 0")
	}

	// with high dividend yield, payout ratio is clamped to 0.8
	vHigh := s.calcDDM(100, 0.10)
	if vHigh <= 0 {
		t.Error("calcDDM with high yield should be positive")
	}
}

func TestCalcDCFWithParams(t *testing.T) {
	s := &ValuationService{}
	// 1e6 shares, 1e6 net income -> fcf = 700k
	v := s.calcDCFWithParams(1_000_000, 1_000_000, 0.10, 0.10, 0.03)
	if v <= 0 {
		t.Fatalf("calcDCF should be positive, got %f", v)
	}

	if s.calcDCFWithParams(1_000_000, 0, 0.10, 0.10, 0.03) != 0 {
		t.Error("calcDCF with 0 shares should return 0")
	}

	if s.calcDCFWithParams(0, 1_000_000, 0.10, 0.10, 0.03) != 0 {
		t.Error("calcDCF with 0 income should return 0")
	}

	// wacc <= terminalGrowth should not divide by zero
	v2 := s.calcDCFWithParams(1_000_000, 1_000_000, 0.10, 0.02, 0.05)
	if v2 <= 0 {
		t.Error("calcDCF with wacc < terminalGrowth should still be positive (wacc adjusted)")
	}
}

func TestCalcMonteCarloDCF(t *testing.T) {
	s := &ValuationService{}
	v := s.calcMonteCarloDCF(1_000_000, 1_000_000)
	if v <= 0 {
		t.Fatalf("monte carlo mean should be positive, got %f", v)
	}
	// should be deterministic given fixed seed
	v2 := s.calcMonteCarloDCF(1_000_000, 1_000_000)
	if math.Abs(v-v2) > 0.0001 {
		t.Errorf("monte carlo should be deterministic, got %f vs %f", v, v2)
	}

	if s.calcMonteCarloDCF(0, 1_000_000) != 0 {
		t.Error("monte carlo with 0 income should return 0")
	}
}

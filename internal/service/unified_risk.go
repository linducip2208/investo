package service

import (
	"fmt"
	"math"
)

type UnifiedRiskManager struct {
	StockPriceRepo interface{}
	ForexRepo      interface{}
}

type RiskAssessment struct {
	Instrument    string   `json:"instrument"`
	MarketType    string   `json:"market_type"`
	MaxPositionPct float64 `json:"max_position_pct"`
	StopLoss      float64  `json:"stop_loss"`
	TakeProfit    float64  `json:"take_profit"`
	RiskScore     float64  `json:"risk_score"`
	Volatility    float64  `json:"volatility"`
	MaxDrawdown   float64  `json:"max_drawdown"`
	Warnings      []string `json:"warnings"`
}

func NewUnifiedRiskManager() *UnifiedRiskManager {
	return &UnifiedRiskManager{}
}

func (r *UnifiedRiskManager) AssessRisk(instrument string, signalScore float64, marketType string, portfolioValue float64) (*RiskAssessment, error) {
	assessment := &RiskAssessment{
		Instrument: instrument,
		MarketType: marketType,
		Warnings:   []string{},
	}

	baseRisk := 50.0

	switch marketType {
	case "BEI":
		baseRisk = r.assessBEIRisk(instrument, signalScore, assessment)
	case "FOREX":
		baseRisk = r.assessForexRisk(instrument, signalScore, assessment)
	default:
		baseRisk = r.assessGenericRisk(signalScore, assessment)
	}

	assessment.RiskScore = math.Round(math.Min(math.Max(baseRisk, 0), 100)*10) / 10

	if portfolioValue > 0 {
		switch {
		case assessment.RiskScore < 30:
			assessment.MaxPositionPct = 0.15
		case assessment.RiskScore < 50:
			assessment.MaxPositionPct = 0.10
		case assessment.RiskScore < 70:
			assessment.MaxPositionPct = 0.05
		default:
			assessment.MaxPositionPct = 0.02
		}
	} else {
		assessment.MaxPositionPct = 0.05
	}

	if signalScore >= 80 {
		assessment.StopLoss = -0.03
		assessment.TakeProfit = 0.08
	} else if signalScore >= 60 {
		assessment.StopLoss = -0.04
		assessment.TakeProfit = 0.06
	} else {
		assessment.StopLoss = -0.05
		assessment.TakeProfit = 0.04
	}

	assessment.Volatility = baseRisk / 100 * 0.05
	assessment.MaxDrawdown = math.Abs(assessment.StopLoss) * 2

	if assessment.RiskScore > 70 {
		assessment.Warnings = append(assessment.Warnings, "Risiko tinggi: pertimbangkan posisi kecil atau skip.")
	}
	if assessment.Volatility > 0.03 {
		assessment.Warnings = append(assessment.Warnings, "Volatilitas tinggi — perketat stop loss.")
	}
	if signalScore < 40 {
		assessment.Warnings = append(assessment.Warnings, "Sinyal lemah — konfirmasi ulang sebelum entry.")
	}

	return assessment, nil
}

func (r *UnifiedRiskManager) assessBEIRisk(code string, signalScore float64, a *RiskAssessment) float64 {
	risk := 50.0

	if len(code) == 4 {
		if code[0] == 'A' || code[0] == 'B' {
			risk -= 5
		}
		if code[0] == 'G' || code[0] == 'Z' {
			risk += 10
		}
	}

	if signalScore > 70 {
		risk -= 10
	} else if signalScore < 40 {
		risk += 10
	}

	araArbProximity := 0.0
	if araArbProximity > 0.9 {
		risk += 15
		a.Warnings = append(a.Warnings, "Harga mendekati batas ARA/ARB — risiko auto-reject.")
	}

	return risk
}

func (r *UnifiedRiskManager) assessForexRisk(pair string, signalScore float64, a *RiskAssessment) float64 {
	risk := 50.0

	if signalScore > 70 {
		risk -= 8
	} else if signalScore < 40 {
		risk += 12
	}

	volatilePairs := map[string]bool{
		"GBPJPY": true, "EURJPY": true, "GBPAUD": true,
		"USDZAR": true, "USDTRY": true, "USDMXN": true,
	}
	if volatilePairs[pair] {
		risk += 10
		a.Warnings = append(a.Warnings, fmt.Sprintf("%s adalah pair volatil — gunakan manajemen risiko ketat.", pair))
	}

	return risk
}

func (r *UnifiedRiskManager) assessGenericRisk(signalScore float64, a *RiskAssessment) float64 {
	risk := 55.0
	if signalScore > 70 {
		risk -= 5
	}
	return risk
}

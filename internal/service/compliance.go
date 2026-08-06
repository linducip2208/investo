package service

import (
	"crypto/rand"
	"fmt"
	"time"
)

type ComplianceService struct {
	MarketType string
}

type ComplianceCheck struct {
	HasDisclaimer bool        `json:"has_disclaimer"`
	RiskWarning   string      `json:"risk_warning"`
	SignalAudit   SignalAudit `json:"signal_audit"`
	Passed        bool        `json:"passed"`
}

type SignalAudit struct {
	SignalID    string      `json:"signal_id"`
	GeneratedAt time.Time   `json:"generated_at"`
	GeneratedBy string      `json:"generated_by"`
	AgentTrail  []AgentStep `json:"agent_trail"`
	Reviewed    bool        `json:"reviewed"`
	ApprovedBy  string      `json:"approved_by"`
}

type AgentStep struct {
	AgentName string    `json:"agent_name"`
	Action    string    `json:"action"`
	Score     float64   `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}

func NewComplianceService(marketType string) *ComplianceService {
	return &ComplianceService{MarketType: marketType}
}

var defaultDisclaimer = `
⚠️ Disclaimer: Informasi ini bukan rekomendasi investasi. Semua keputusan trading dan investasi sepenuhnya menjadi tanggung jawab Anda sendiri. Kinerja masa lalu tidak menjamin hasil di masa depan. Pastikan Anda memahami risiko sebelum berinvestasi di pasar modal.`

var foreDisclaimer = `
⚠️ Disclaimer: Trading forex memiliki risiko tinggi. Informasi ini bukan rekomendasi dan disediakan untuk tujuan edukasi. Leverage dapat memperbesar keuntungan maupun kerugian. Pastikan Anda memahami risiko leverage, margin, dan volatilitas pasar forex sebelum bertransaksi. Semua keputusan trading sepenuhnya tanggung jawab Anda.`

func (c *ComplianceService) AddDisclaimer(signalText string) string {
	disclaimer := defaultDisclaimer
	if c.MarketType == "FOREX" {
		disclaimer = foreDisclaimer
	}
	return signalText + "\n\n---\n" + disclaimer
}

func (c *ComplianceService) AuditSignal(instrument string, marketType string, agentScores map[string]float64) *SignalAudit {
	auditID := generateUUID()

	var trail []AgentStep
	now := time.Now()
	for agentName, score := range agentScores {
		trail = append(trail, AgentStep{
			AgentName: agentName,
			Action:    fmt.Sprintf("Analyze %s %s", instrument, marketType),
			Score:     score,
			Timestamp: now,
		})
	}

	return &SignalAudit{
		SignalID:    auditID,
		GeneratedAt: now,
		GeneratedBy: "Investo Multi-Agent Trading System",
		AgentTrail:  trail,
		Reviewed:    false,
		ApprovedBy:  "",
	}
}

func (c *ComplianceService) CheckRegulatory(marketType string) ([]string, error) {
	var rules []string

	switch marketType {
	case "BEI":
		rules = []string{
			"OJK Regulation No. 11/POJK.04/2022: Investment analysis reports must include disclaimer",
			"IDX Rule II-A: Auto rejection limits (ARA 25%, ARB 7% for stocks > Rp 5,000)",
			"OJK Circular: No guaranteed return promises allowed",
			"IDX: Trading signals must not constitute investment advice without license",
		}
	case "FOREX":
		rules = []string{
			"BAPPEBTI Regulation: Forex brokers must be licensed by BAPPEBTI",
			"BAPPEBTI: Maximum leverage 1:400 for Indonesian retail forex traders",
			"BAPPEBTI: Mandatory risk disclosure for all marketing materials",
			"BAPPEBTI: No guaranteed profit claims allowed in any communication",
		}
	default:
		return nil, fmt.Errorf("market type tidak dikenal: %s", marketType)
	}

	return rules, nil
}

func (c *ComplianceService) RunComplianceCheck(signalText string, marketType string, agentScores map[string]float64, instrument string) *ComplianceCheck {
	hasDisclaimer := containsDisclaimerKeywords(signalText)
	audit := c.AuditSignal(instrument, marketType, agentScores)

	riskWarning := defaultDisclaimer
	if marketType == "FOREX" {
		riskWarning = foreDisclaimer
	}

	passed := hasDisclaimer

	return &ComplianceCheck{
		HasDisclaimer: hasDisclaimer,
		RiskWarning:   riskWarning,
		SignalAudit:   *audit,
		Passed:        passed,
	}
}

func containsDisclaimerKeywords(text string) bool {
	keywords := []string{
		"bukan rekomendasi",
		"tanggung jawab",
		"disclaimer",
		"sepenuhnya tanggung jawab",
	}
	for _, kw := range keywords {
		if contains(text, kw) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

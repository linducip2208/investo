package service

import (
	"fmt"
	"time"
)

type WebhookIntelConfig struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	EventType   string `json:"event_type"`
	Condition   string `json:"condition"`
	IsActive    bool   `json:"is_active"`
}

type WebhookLog struct {
	ID        int64     `json:"id"`
	ConfigID  int64     `json:"config_id"`
	Event     string    `json:"event"`
	Payload   string    `json:"payload"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type AIWebhookIntelService struct {
	AI *AIService
}

func (s *AIWebhookIntelService) GetDefaultConfigs() []WebhookIntelConfig {
	return []WebhookIntelConfig{
		{
			Name:        "Signal Confidence > 80%",
			Description: "Ketika confidence AI signal > 80 untuk saham bank",
			EventType:   "ai_signal",
			Condition:   "signal_confidence > 80 AND sector = bank",
		},
		{
			Name:        "Black Swan Detection",
			Description: "Ketika terdeteksi potensi black swan event",
			EventType:   "black_swan",
			Condition:   "anomaly_score > 0.9",
		},
		{
			Name:        "Portfolio Drop > 5%",
			Description: "Ketika portofolio turun > 5% dalam sehari",
			EventType:   "portfolio_alert",
			Condition:   "daily_change_pct < -5",
		},
	}
}

func (s *AIWebhookIntelService) TestWebhook(url string) (string, error) {
	testPayload := fmt.Sprintf(`{"event":"test","message":"Webhook test from Investo AI","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	return testPayload, nil
}

func (s *AIWebhookIntelService) GenerateWebhookDescription(config WebhookIntelConfig) string {
	descriptions := map[string]string{
		"ai_signal":      fmt.Sprintf("Trigger: Ketika confidence sinyal AI > threshold untuk saham tertentu. URL: %s", config.URL),
		"black_swan":     fmt.Sprintf("Trigger: Deteksi anomali pasar level black swan. URL: %s", config.URL),
		"portfolio_alert": fmt.Sprintf("Trigger: Portofolio turun > threshold harian. URL: %s", config.URL),
	}

	if desc, ok := descriptions[config.EventType]; ok {
		return desc
	}
	return fmt.Sprintf("Trigger: %s. URL: %s", config.EventType, config.URL)
}

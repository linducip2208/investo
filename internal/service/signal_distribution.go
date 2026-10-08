package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SignalDistributor struct {
	WAClient     *WABusinessClient
	WebhookSender *WebhookService
}

type WABusinessClient struct {
	PhoneNumberID string
	AccessToken   string
	client        *http.Client
}

type DistributionResult struct {
	Success   bool     `json:"success"`
	Channels  []string `json:"channels"`
	Errors    []string `json:"errors"`
	Timestamp string   `json:"timestamp"`
}

func NewSignalDistributor(waClient *WABusinessClient, webhookSvc *WebhookService) *SignalDistributor {
	return &SignalDistributor{
		WAClient:     waClient,
		WebhookSender: webhookSvc,
	}
}

func NewWABusinessClient(phoneNumberID, accessToken string) *WABusinessClient {
	return &WABusinessClient{
		PhoneNumberID: phoneNumberID,
		AccessToken:   accessToken,
		client:        &http.Client{Timeout: 15 * time.Second},
	}
}

func (d *SignalDistributor) DistributeSignal(signal FinalSignal, targets []string) *DistributionResult {
	result := &DistributionResult{
		Success:   true,
		Channels:  []string{},
		Errors:    []string{},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	message := d.FormatSignalMessage(signal)

	if d.WAClient != nil && d.WAClient.IsConfigured() {
		for _, target := range targets {
			if strings.HasPrefix(target, "62") || strings.HasPrefix(target, "+62") {
				if err := d.WAClient.SendMessage(target, message); err != nil {
					result.Errors = append(result.Errors, "WA: "+err.Error())
				} else {
					result.Channels = append(result.Channels, "whatsapp")
				}
			}
		}
	}

	if d.WebhookSender != nil {
		payload := map[string]interface{}{
			"type":    "signal_distribution",
			"signal":  signal,
			"message": message,
		}
		payloadJSON, _ := json.Marshal(payload)
		if err := d.WebhookSender.Send(string(payloadJSON)); err != nil {
			result.Errors = append(result.Errors, "Webhook: "+err.Error())
		} else {
			result.Channels = append(result.Channels, "webhook")
		}
	}

	if len(result.Errors) > 0 && len(result.Channels) == 0 {
		result.Success = false
	}

	return result
}

func (d *SignalDistributor) FormatSignalMessage(signal FinalSignal) string {
	var emoji string
	switch signal.Signal {
	case "STRONG_BUY":
		emoji = "🔥"
	case "BUY":
		emoji = "📈"
	case "HOLD":
		emoji = "⏸️"
	case "SELL":
		emoji = "📉"
	case "STRONG_SELL":
		emoji = "🚨"
	default:
		emoji = "📊"
	}

	marketLabel := "IHSG / BEI"
	if signal.MarketType == "FOREX" {
		marketLabel = "Forex"
	}

	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("🔔 *SINYAL INVESTOC*\n\n"))
	sb.WriteString(fmt.Sprintf("%s %s — %s\n", emoji, signal.Instrument, marketLabel))
	sb.WriteString(fmt.Sprintf("🎯 *%s* (Confidence: %.0f%%)\n\n", signal.Signal, signal.Confidence))

	if scores, ok := signal.AgentScores["fundamental"]; ok {
		sb.WriteString(fmt.Sprintf("📊 Fundamental: %.0f/100\n", scores))
	}
	if scores, ok := signal.AgentScores["foreign_flow"]; ok {
		sb.WriteString(fmt.Sprintf("💵 Foreign Flow: %.0f/100\n", scores))
	}
	if scores, ok := signal.AgentScores["sentiment"]; ok {
		sb.WriteString(fmt.Sprintf("📰 Sentiment: %.0f/100\n", scores))
	}
	if scores, ok := signal.AgentScores["technical"]; ok {
		sb.WriteString(fmt.Sprintf("📉 Technical: %.0f/100\n", scores))
	}
	if scores, ok := signal.AgentScores["macro"]; ok {
		sb.WriteString(fmt.Sprintf("🏛️ Macro: %.0f/100\n", scores))
	}

	sb.WriteString(fmt.Sprintf("\n💰 Entry: %.2f\n", signal.EntryPrice))
	sb.WriteString(fmt.Sprintf("🛑 Stop Loss: %.2f\n", signal.StopLoss))
	sb.WriteString(fmt.Sprintf("✅ Target: %.2f\n", signal.TakeProfit))

	if len(signal.RiskWarnings) > 0 {
		sb.WriteString("\n⚠️ *Peringatan Risiko:*\n")
		for _, w := range signal.RiskWarnings {
			sb.WriteString(fmt.Sprintf("- %s\n", w))
		}
	}

	sb.WriteString("\n⚠️ *Disclaimer:* Bukan rekomendasi investasi. Keputusan trading sepenuhnya tanggung jawab Anda.\n")
	sb.WriteString("\n#InvestoC #SahamIDX")

	return sb.String()
}

func (w *WABusinessClient) IsConfigured() bool {
	return w.PhoneNumberID != "" && w.AccessToken != ""
}

func (w *WABusinessClient) SendMessage(to string, message string) error {
	if !w.IsConfigured() {
		return fmt.Errorf("WA Business API tidak dikonfigurasi")
	}

	to = strings.TrimPrefix(to, "+")

	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", w.PhoneNumberID)

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal WA message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create WA request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.AccessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("WA API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("WA API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"investo/internal/model"
)

type WebhookService struct {
	HTTPClient *http.Client
}

type WebhookConfig struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

type AlertWebhookPayload struct {
	Event       string  `json:"event"`
	Timestamp   string  `json:"timestamp"`
	AlertID     int64   `json:"alert_id"`
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Condition   string  `json:"condition"`
	TargetPrice float64 `json:"target_price"`
	CurrentPrice float64 `json:"current_price"`
	Message     string  `json:"message"`
}

type DailyRecapWebhookPayload struct {
	Event     string  `json:"event"`
	Timestamp string  `json:"timestamp"`
	Date      string  `json:"date"`
	Advance   int     `json:"advance"`
	Decline   int     `json:"decline"`
	Unchanged int     `json:"unchanged"`
	AdvancePercent float64 `json:"advance_percent"`
	DeclinePercent float64 `json:"decline_percent"`
}

func NewWebhookService() *WebhookService {
	return &WebhookService{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *WebhookService) SendAlertWebhook(config WebhookConfig, alert model.Alert, stock model.Stock, currentPrice float64) error {
	if config.URL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	payload := AlertWebhookPayload{
		Event:        "alert.triggered",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		AlertID:      alert.ID,
		StockCode:    stock.Code,
		StockName:    stock.Name,
		Condition:    alert.Condition,
		TargetPrice:  alert.TargetPrice,
		CurrentPrice: currentPrice,
		Message:      fmt.Sprintf("Alert %s %s: %s saat ini Rp %.0f (target: Rp %.0f)", stock.Code, stock.Name, alert.Condition, currentPrice, alert.TargetPrice),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("SendAlertWebhook marshal: %w", err)
	}

	return s.send(config.URL, body, config.Secret)
}

func (s *WebhookService) SendDailyRecapWebhook(config WebhookConfig, recap interface{}) error {
	if config.URL == "" {
		return nil
	}

	body, err := json.Marshal(recap)
	if err != nil {
		return fmt.Errorf("SendDailyRecapWebhook marshal: %w", err)
	}

	return s.send(config.URL, body, config.Secret)
}

func (s *WebhookService) Send(payloadJSON string) error {
	config := WebhookConfig{URL: "", Secret: ""}
	_ = config
	return s.send("", []byte(payloadJSON), "")
}

func (s *WebhookService) send(targetURL string, body []byte, secret string) error {
	if strings.TrimSpace(targetURL) == "" {
		return fmt.Errorf("webhook URL is empty")
	}
	if err := ValidateOutboundURL(targetURL); err != nil {
		return err
	}
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("webhook: invalid URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ResolveAndCheckHost(ctx, parsed.Hostname()); err != nil {
		return err
	}

	client := NewOutboundClient(10 * time.Second)
	client.HTTP.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) > 2 {
			return fmt.Errorf("webhook: too many redirects")
		}
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook send create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Investo-Webhook/1.0")

	if secret != "" {
		sig := s.computeSignature(body, secret)
		req.Header.Set("X-Investo-Signature", sig)
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("webhook send request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *WebhookService) VerifySignature(body []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	expected := s.computeSignature(body, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *WebhookService) computeSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

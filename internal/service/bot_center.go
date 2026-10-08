package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"investo/internal/model"
)

type BotCenterService struct {
	HTTPClient *http.Client
}

func NewBotCenterService() *BotCenterService {
	return &BotCenterService{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *BotCenterService) ForwardSignal(signal TradingSignal, bot model.Bot) error {
	var cfg model.BotConfig
	if err := json.Unmarshal([]byte(bot.Config), &cfg); err != nil {
		return fmt.Errorf("ForwardSignal parse config: %w", err)
	}

	switch bot.Type {
	case "telegram":
		return s.forwardTelegram(signal, cfg)
	case "webhook":
		return s.forwardWebhook(signal, cfg)
	case "email":
		return s.forwardEmail(signal, cfg)
	default:
		return fmt.Errorf("unknown bot type: %s", bot.Type)
	}
}

func (s *BotCenterService) forwardTelegram(signal TradingSignal, cfg model.BotConfig) error {
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("telegram: bot token or chat ID missing")
	}

	text := fmt.Sprintf(
		"\U0001F4C8 *Sinyal %s*\n\n"+
			"\U0001F4B0 *%s* \u2014 %s\n"+
			"\U0001F3AF Tipe: %s\n"+
			"\U0001F4CA Confidence: %d%%\n"+
			"\U0001F4DD %s\n\n"+
			"\U0001F4B5 Harga: Rp %s\n"+
			"\U0001F3AF Target: Rp %s\n"+
			"\U0001F6D1 Stop Loss: Rp %s\n"+
			"\U000023F0 %s",
		signal.Type,
		signal.StockCode,
		signal.StockName,
		signal.Type,
		signal.Confidence,
		signal.Reason,
		formatRupiah(signal.Price),
		formatRupiah(signal.TargetPrice),
		formatRupiah(signal.StopLoss),
		time.Now().Format("02/01/2006 15:04 WIB"),
	)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	body := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("telegram marshal: %w", err)
	}

	resp, err := s.HTTPClient.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("telegram API error: %d — %v", resp.StatusCode, result)
	}

	return nil
}

func (s *BotCenterService) forwardWebhook(signal TradingSignal, cfg model.BotConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("webhook: URL missing")
	}

	payload := map[string]interface{}{
		"event":        "signal.generated",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"stock_code":   signal.StockCode,
		"stock_name":   signal.StockName,
		"type":         signal.Type,
		"confidence":   signal.Confidence,
		"reason":       signal.Reason,
		"indicators":   signal.Indicators,
		"price":        signal.Price,
		"target_price": signal.TargetPrice,
		"stop_loss":    signal.StopLoss,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Investo-BotCenter/1.0")

	if cfg.Secret != "" {
		req.Header.Set("X-Investo-Signature", cfg.Secret)
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *BotCenterService) forwardEmail(signal TradingSignal, cfg model.BotConfig) error {
	log.Printf("[bot-center] email would be sent: signal=%s type=%s confidence=%d host=%s to=%s",
		signal.StockCode, signal.Type, signal.Confidence, cfg.SMTPHost, cfg.SMTPUser)
	return nil
}

func formatRupiah(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

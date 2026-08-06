package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type TelegramService struct {
	BotToken string
	ChatIDs  map[int64]string
	mu       sync.RWMutex
	client   *http.Client
}

func NewTelegramService(botToken string) *TelegramService {
	return &TelegramService{
		BotToken: botToken,
		ChatIDs:  make(map[int64]string),
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramService) SetChatID(userID int64, chatID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ChatIDs[userID] = chatID
}

func (t *TelegramService) SendAlert(userID int64, message string) error {
	t.mu.RLock()
	chatID, ok := t.ChatIDs[userID]
	t.mu.RUnlock()

	if !ok || chatID == "" {
		log.Printf("telegram: skip alert for user %d — no chat ID registered", userID)
		return nil
	}

	if t.BotToken == "" {
		log.Printf("telegram: skip alert — no bot token configured")
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)

	body := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("telegram marshal: %w", err)
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewReader(payload))
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

func (t *TelegramService) FormatAlertMessage(result AlertResult) string {
	now := time.Now().Format("02/01/2006 15:04 WIB")

	msg := fmt.Sprintf(
		"\U0001F514 *Alert Investo*\n\n"+
			"\U0001F4C8 *%s* \u2014 %s\n"+
			"\U0001F4B0 Harga: Rp %s\n"+
			"\U0001F4CA %s\n\n"+
			"\u23F0 %s",
		result.StockCode,
		result.StockName,
		formatPrice(result.CurrentPrice),
		result.Message,
		now,
	)

	return msg
}

func formatPrice(price float64) string {
	s := fmt.Sprintf("%.0f", price)
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

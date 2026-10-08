package service

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type WhatsAppBotService struct {
	DB *sqlx.DB
}

func NewWhatsAppBotService(db *sqlx.DB) *WhatsAppBotService {
	return &WhatsAppBotService{DB: db}
}

func (s *WhatsAppBotService) ProcessCommand(phone string, message string) (string, error) {
	msg := strings.TrimSpace(message)
	upper := strings.ToUpper(msg)

	if strings.HasPrefix(upper, "INFO ") {
		code := strings.TrimSpace(msg[5:])
		return s.handleInfo(code)
	}

	if strings.HasPrefix(upper, "HARGA ") {
		code := strings.TrimSpace(msg[6:])
		return s.handlePrice(code)
	}

	if strings.HasPrefix(upper, "CHART ") {
		code := strings.TrimSpace(msg[6:])
		return s.handleChart(code)
	}

	if strings.HasPrefix(upper, "FOREX ") {
		pair := strings.TrimSpace(msg[6:])
		return s.handleForex(pair)
	}

	if strings.HasPrefix(upper, "ALERT ") {
		parts := strings.Fields(msg[6:])
		if len(parts) >= 2 {
			code := parts[0]
			var price float64
			fmt.Sscanf(parts[1], "%f", &price)
			return s.handleAlert(phone, code, price)
		}
		return "Format salah. Gunakan: ALERT {CODE} {PRICE}\nContoh: ALERT BBCA 12000", nil
	}

	if upper == "HELP" || upper == "BANTUAN" {
		return s.handleHelp(), nil
	}

	return s.handleHelp(), nil
}

func (s *WhatsAppBotService) handleInfo(code string) (string, error) {
	type stockInfo struct {
		Name  string  `db:"name"`
		Price float64 `db:"price"`
		Chg   float64 `db:"change_percent"`
		PER   float64 `db:"per"`
		PBV   float64 `db:"pbv"`
	}

	var info stockInfo
	query := `SELECT s.name,
		COALESCE((SELECT close FROM stock_prices WHERE stock_id = s.id ORDER BY date DESC LIMIT 1), 0) as price,
		COALESCE((SELECT ((curr.close - prev.close) / prev.close * 100)
			FROM (SELECT close FROM stock_prices WHERE stock_id = s.id ORDER BY date DESC LIMIT 1) curr
			CROSS JOIN (SELECT close FROM stock_prices WHERE stock_id = s.id ORDER BY date DESC LIMIT 1 OFFSET 1) prev), 0) as change_percent,
		COALESCE((SELECT per FROM stock_fundamentals WHERE stock_id = s.id ORDER BY period DESC LIMIT 1), 0) as per,
		COALESCE((SELECT pbv FROM stock_fundamentals WHERE stock_id = s.id ORDER BY period DESC LIMIT 1), 0) as pbv
		FROM stocks s WHERE UPPER(s.code) = ? LIMIT 1`

	if err := s.DB.Get(&info, query, strings.ToUpper(code)); err != nil {
		return fmt.Sprintf("Saham %s tidak ditemukan.", code), nil
	}

	chgEmoji := "📈"
	chgSign := "+"
	if info.Chg < 0 {
		chgEmoji = "📉"
		chgSign = ""
	}

	priceStr := fmt.Sprintf("Rp %s", formatNum(int64(info.Price)))

	return fmt.Sprintf(`%s *%s - %s*
💰 %s
%s %s%.2f%%
📊 PER: %.1f | PBV: %.1f`, chgEmoji, strings.ToUpper(code), info.Name, priceStr, chgEmoji, chgSign, info.Chg, info.PER, info.PBV), nil
}

func (s *WhatsAppBotService) handlePrice(code string) (string, error) {
	type priceInfo struct {
		Name  string  `db:"name"`
		Price float64 `db:"price"`
	}
	var info priceInfo
	query := `SELECT s.name, COALESCE((SELECT close FROM stock_prices WHERE stock_id = s.id ORDER BY date DESC LIMIT 1), 0) as price
		FROM stocks s WHERE UPPER(s.code) = ? LIMIT 1`

	if err := s.DB.Get(&info, query, strings.ToUpper(code)); err != nil {
		return fmt.Sprintf("Saham %s tidak ditemukan.", code), nil
	}

	return fmt.Sprintf("💰 *%s*: Rp %s", strings.ToUpper(code), formatNum(int64(info.Price))), nil
}

func (s *WhatsAppBotService) handleChart(code string) (string, error) {
	parts := strings.Fields(code)
	stockCode := parts[0]

	type stockCheck struct {
		ID int64 `db:"id"`
	}
	var sc stockCheck
	query := `SELECT id FROM stocks WHERE UPPER(code) = ? LIMIT 1`
	if err := s.DB.Get(&sc, query, strings.ToUpper(stockCode)); err != nil {
		return fmt.Sprintf("Saham %s tidak ditemukan.", stockCode), nil
	}

	return fmt.Sprintf("📊 *Chart %s*\nSilakan buka: /saham/%s", strings.ToUpper(stockCode), strings.ToLower(stockCode)), nil
}

func (s *WhatsAppBotService) handleForex(pair string) (string, error) {
	pair = strings.ToUpper(strings.ReplaceAll(pair, "/", ""))
	if len(pair) != 6 {
		return "Format salah. Gunakan: FOREX {PAIR}\nContoh: FOREX USDIDR", nil
	}

	base := pair[:3]
	quote := pair[3:]

	type forexInfo struct {
		Name  string  `db:"name"`
		Rate  float64 `db:"rate"`
		Chg   float64 `db:"change"`
	}
	var info forexInfo
	query := `SELECT fp.name,
		COALESCE((SELECT close FROM forex_rates WHERE pair_id = fp.id ORDER BY date DESC LIMIT 1), 0) as rate,
		COALESCE((SELECT ((curr.close - prev.close) / prev.close * 100)
			FROM (SELECT close FROM forex_rates WHERE pair_id = fp.id ORDER BY date DESC LIMIT 1) curr
			CROSS JOIN (SELECT close FROM forex_rates WHERE pair_id = fp.id ORDER BY date DESC LIMIT 1 OFFSET 1) prev), 0) as change
		FROM forex_pairs fp WHERE UPPER(fp.base_currency) = ? AND UPPER(fp.quote_currency) = ? LIMIT 1`

	if err := s.DB.Get(&info, query, base, quote); err != nil {
		return fmt.Sprintf("Pair %s/%s tidak ditemukan.", base, quote), nil
	}

	chgSign := "+"
	if info.Chg < 0 {
		chgSign = ""
	}

	return fmt.Sprintf(`💱 *%s/%s* - %s
💰 Rate: %.4f
📊 Perubahan: %s%.2f%%`, base, quote, info.Name, info.Rate, chgSign, info.Chg), nil
}

func (s *WhatsAppBotService) handleAlert(phone, code string, price float64) (string, error) {
	type stockCheck struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	var sc stockCheck
	query := `SELECT id, name FROM stocks WHERE UPPER(code) = ? LIMIT 1`
	if err := s.DB.Get(&sc, query, strings.ToUpper(code)); err != nil {
		return fmt.Sprintf("Saham %s tidak ditemukan.", code), nil
	}

	currentPrice, _ := s.getLatestPrice(sc.ID)

	cond := "naik_ke"
	if currentPrice > price {
		cond = "turun_ke"
	}

	insertQ := `INSERT INTO alerts (user_id, stock_id, alert_type, condition_type, condition_value, is_active, whatsapp_phone)
		VALUES ((SELECT id FROM users LIMIT 1), ?, 'price', ?, ?, 1, ?)`
	_, err := s.DB.Exec(insertQ, sc.ID, cond, price, phone)
	if err != nil {
		_ = err
	}

	return fmt.Sprintf("✅ *Alert dibuat:* %s > %s\nKondisi: %s mencapai Rp %s",
		strings.ToUpper(code), formatNum(int64(price)), strings.ReplaceAll(cond, "_", " "), formatNum(int64(price))), nil
}

func (s *WhatsAppBotService) getLatestPrice(stockID int64) (float64, error) {
	var price float64
	query := `SELECT close FROM stock_prices WHERE stock_id = ? ORDER BY date DESC LIMIT 1`
	if err := s.DB.Get(&price, query, stockID); err != nil {
		return 0, err
	}
	return price, nil
}

func (s *WhatsAppBotService) handleHelp() string {
	return `🤖 *Investo WhatsApp Bot*

Perintah yang tersedia:

📊 *INFO {CODE}* - Info lengkap saham
  Contoh: INFO BBCA

💰 *HARGA {CODE}* - Harga terkini
  Contoh: HARGA TLKM

📈 *CHART {CODE}* - Link chart
  Contoh: CHART BBCA

💱 *FOREX {PAIR}* - Info forex
  Contoh: FOREX USDIDR

🔔 *ALERT {CODE} {PRICE}* - Buat alert
  Contoh: ALERT BBCA 12000

❓ *HELP* - Tampilkan bantuan ini`
}

func formatNum(n int64) string {
	if n < 0 {
		return "-" + formatNum(-n)
	}
	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

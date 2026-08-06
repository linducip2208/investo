package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type VoiceParseResult struct {
	Success   bool    `json:"success"`
	Action    string  `json:"action"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	StockCode string  `json:"stock_code"`
	Condition string  `json:"condition"`
	Price     float64 `json:"price"`
	RawText   string  `json:"raw_text"`
	Message   string  `json:"message"`
}

type AIVoiceService struct {
	AI *AIService
}

func NewAIVoiceService(ai *AIService) *AIVoiceService {
	return &AIVoiceService{AI: ai}
}

func (s *AIVoiceService) ParseVoiceCommand(transcript string) (*VoiceParseResult, error) {
	transcript = strings.TrimSpace(strings.ToLower(transcript))

	if transcript == "" {
		return &VoiceParseResult{Success: false, Message: "Tidak ada suara yang terdeteksi. Coba lagi."}, nil
	}

	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	if aiAvailable {
		result, err := s.parseWithAI(transcript)
		if err == nil && result.Success {
			return result, nil
		}
	}

	return s.parseWithRegex(transcript)
}

func (s *AIVoiceService) parseWithAI(transcript string) (*VoiceParseResult, error) {
	prompt := fmt.Sprintf(`Kamu adalah parser perintah trading suara. Parse perintah berikut ke JSON:
"%s"

Aturan:
- action: "beli" atau "jual"
- quantity: angka jumlah lot/saham
- unit: "lot" atau "lembar"
- stock_code: kode saham (contoh: BBCA, TLKM, ASII) - uppercase
- condition: kondisi seperti "tembus", "turun ke", "naik ke", "cross" - atau kosong jika tidak ada
- price: harga target jika ada, 0 jika tidak ada
- message: konfirmasi dalam bahasa Indonesia

Response HARUS JSON valid.`, transcript)

	resp, err := s.AI.Chat("Kamu adalah parser perintah trading. Response HARUS JSON valid. Jangan tambahkan teks lain.", prompt)
	if err != nil {
		return nil, err
	}

	var result VoiceParseResult
	if err := json.Unmarshal([]byte(cleanJSON(resp)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	result.Success = result.Action != "" && result.StockCode != ""
	result.RawText = transcript
	return &result, nil
}

func (s *AIVoiceService) parseWithRegex(transcript string) (*VoiceParseResult, error) {
	result := &VoiceParseResult{RawText: transcript}

	transcript = strings.ToUpper(transcript)

	actionRe := regexp.MustCompile(`(?i)\b(BELI|JUAL|BUY|SELL)\b`)
	if match := actionRe.FindString(transcript); match != "" {
		upper := strings.ToUpper(match)
		if upper == "BELI" || upper == "BUY" {
			result.Action = "beli"
		} else {
			result.Action = "jual"
		}
	}

	qtyRe := regexp.MustCompile(`(\d+)\s*(LOT|LEMBAR|LOTS|SHARES)`)
	if match := qtyRe.FindStringSubmatch(transcript); len(match) >= 3 {
		result.Quantity, _ = strconv.ParseFloat(match[1], 64)
		result.Unit = strings.ToLower(match[2])
		if result.Unit == "shares" || result.Unit == "lembar" {
			result.Unit = "lembar"
		} else {
			result.Unit = "lot"
		}
	}

	stockRe := regexp.MustCompile(`\b([A-Z]{4})\b`)
	if matches := stockRe.FindAllStringSubmatch(transcript, -1); len(matches) > 0 {
		for _, m := range matches {
			code := m[1]
			if code != "BELI" && code != "JUAL" && code != "LOTS" {
				result.StockCode = code
				break
			}
		}
	}

	condRe := regexp.MustCompile(`(?i)(TEMBUS|TURUN\s*KE|NAIK\s*KE|BREAK|CROSS|DI\s*ATAS|DI\s*BAWAH|KE|PADA)\s*(\d[\d.]*)`)
	if match := condRe.FindStringSubmatch(transcript); len(match) >= 3 {
		condWord := strings.ToLower(strings.TrimSpace(match[1]))
		result.Price, _ = strconv.ParseFloat(match[2], 64)

		switch condWord {
		case "tembus", "break", "di atas":
			result.Condition = "tembus"
		case "turun ke", "di bawah":
			result.Condition = "turun ke"
		case "naik ke":
			result.Condition = "naik ke"
		default:
			result.Condition = strings.TrimSpace(match[1])
		}
	}

	if result.Quantity == 0 {
		qtySimpleRe := regexp.MustCompile(`(\d+)\s*(LOT|LEMBAR|SAHAM)`)
		if match := qtySimpleRe.FindStringSubmatch(strings.ToLower(transcript)); len(match) >= 3 {
			result.Quantity, _ = strconv.ParseFloat(match[1], 64)
			result.Unit = "lot"
		}
	}

	if result.Action != "" && result.StockCode != "" {
		result.Success = true
		result.Message = s.buildConfirmation(result)
	} else {
		result.Message = "Maaf, tidak dapat memahami perintah. Coba: \"Beli 10 lot BBCA kalau tembus 6500\""
	}

	return result, nil
}

func (s *AIVoiceService) buildConfirmation(r *VoiceParseResult) string {
	actionText := "Membeli"
	if r.Action == "jual" {
		actionText = "Menjual"
	}

	qtyText := fmt.Sprintf("%.0f %s", r.Quantity, r.Unit)

	msg := fmt.Sprintf("%s %s saham %s", actionText, qtyText, r.StockCode)

	if r.Condition != "" && r.Price > 0 {
		msg += fmt.Sprintf(" dengan kondisi %s %d", r.Condition, int(r.Price))
	}

	return msg
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

package service

import (
	"fmt"
	"strings"
	"time"

	"investo/internal/repository"
)

type AIPersonalizedService struct {
	AI               *AIService
	PortfolioRepo    *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	TradeJournalRepo *repository.TradeJournalRepository
	StockRepo        *repository.StockRepository
	StockPriceRepo   *repository.StockPriceRepository
	SectorRepo       *repository.SectorRepository
}

type RiskProfileResult struct {
	Profile              string            `json:"profile"`
	Score                int               `json:"score"`
	Allocation           map[string]int    `json:"allocation"`
	Description          string            `json:"description"`
	Recommendation       string            `json:"recommendation"`
}

func (s *AIPersonalizedService) GeneratePortfolioAdvice(portfolioID int64) (string, error) {
	portfolio, err := s.PortfolioRepo.FindByID(portfolioID)
	if err != nil {
		return "", fmt.Errorf("portfolio not found: %w", err)
	}

	items, err := s.PortfolioItemRepo.GetWithStock(portfolioID)
	if err != nil {
		return "", fmt.Errorf("portfolio items: %w", err)
	}

	var holdingLines string
	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.Item.StockID)
		holdingLines += fmt.Sprintf("- %s (%s): %v %.0f lot @ Rp %.0f | Notes: %s\n",
			item.Stock.Code, item.Stock.Name, item.Item.Type, item.Item.Quantity,
			item.Item.AvgPrice, item.Item.Notes)
	}

	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	var totalValue, totalCost float64
	for _, item := range items {
		price := priceMap[item.Item.StockID]
		totalCost += item.Item.Quantity * item.Item.AvgPrice * 100
		totalValue += item.Item.Quantity * price * 100
	}

	pnl := totalValue - totalCost
	pnlPct := 0.0
	if totalCost > 0 {
		pnlPct = (pnl / totalCost) * 100
	}

	systemPrompt := "Kamu adalah penasihat investasi (investment advisor) profesional di Indonesia. Berikan saran yang personal, data-driven, dan mudah dipahami dalam Bahasa Indonesia."

	userMessage := fmt.Sprintf(`Berikan saran investasi personal untuk portofolio "%s" dalam Bahasa Indonesia.

📊 HOLDINGS:
%s

💰 RINGKASAN:
- Total Modal: Rp %.0f
- Nilai Saat Ini: Rp %.0f
- P/L: Rp %.0f (%.2f%%)

Tolong berikan saran yang mencakup:
1. Evaluasi alokasi aset saat ini
2. Saham yang perlu dipertimbangkan untuk ditahan/dijual
3. Saran diversifikasi
4. Strategi untuk 3-6 bulan ke depan

Gunakan Bahasa Indonesia. Format dalam HTML yang rapi.`,
		portfolio.Name,
		holdingLines,
		totalCost, totalValue, pnl, pnlPct)

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func (s *AIPersonalizedService) GenerateRiskProfile(answers map[string]string) (*RiskProfileResult, error) {
	var answerLines string
	for q, a := range answers {
		answerLines += fmt.Sprintf("%s: %s\n", q, a)
	}

	systemPrompt := `Kamu adalah risk assessment specialist untuk investor pasar modal Indonesia. 
Berdasarkan jawaban risk questionnaire, tentukan profil risiko investor dan berikan rekomendasi alokasi aset.
Kembalikan HASIL dalam format JSON dengan struktur:
{
  "profile": "Conservative/Moderate/Aggressive",
  "score": 0-100,
  "allocation": {"Saham": 30, "Obligasi": 40, "Reksadana": 20, "Emas": 5, "Kas": 5},
  "description": "deskripsi profil risiko",
  "recommendation": "rekomendasi strategi investasi"
}
HANYA kembalikan JSON yang valid, tanpa teks lain.`

	userMessage := fmt.Sprintf(`Berikut adalah jawaban risk assessment questionnaire dari seorang investor:
%s

Tentukan profil risiko (Conservative/Moderate/Aggressive) dan berikan rekomendasi alokasi aset.
Kembalikan dalam format JSON yang valid.`,
		answerLines)

	result, err := s.AI.QuickChat(systemPrompt, userMessage)
	if err != nil {
		return nil, err
	}

	riskResult := &RiskProfileResult{
		Profile:        "Moderate",
		Score:          50,
		Description:    "Berdasarkan jawaban Anda, profil risiko Anda adalah Moderate.",
		Recommendation: result,
		Allocation: map[string]int{
			"Saham": 50, "Obligasi": 30, "Reksadana": 10, "Emas": 5, "Kas": 5,
		},
	}

	return riskResult, nil
}

func (s *AIPersonalizedService) GenerateLearningPath(topic, level string) (string, error) {
	systemPrompt := "Kamu adalah edukator pasar modal dan mentor trading berpengalaman di Indonesia. Buat kurikulum belajar yang terstruktur, praktis, dan sesuai level investor Indonesia."

	userMessage := fmt.Sprintf(`Buatkan learning path (kurikulum belajar) untuk topik "%s" dengan level "%s" dalam Bahasa Indonesia.

Tolong buat kurikulum yang mencakup:
1. Deskripsi learning path
2. Prasyarat (jika ada)
3. Modul-modul pembelajaran (minimal 5-6 modul), masing-masing dengan:
   - Judul modul
   - Sub-topik yang dipelajari
   - Estimasi waktu
   - Sumber belajar yang direkomendasikan
4. Proyek akhir / evaluasi
5. Rekomendasi next step setelah menyelesaikan learning path

Format dalam HTML yang rapi. Gunakan Bahasa Indonesia.`,
		topic, level)

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func (s *AIPersonalizedService) AnalyzeTradeJournal(userID int64) (string, error) {
	now := time.Now()
	startDate := now.AddDate(0, -6, 0)
	journals, err := s.TradeJournalRepo.FindByUserIDFiltered(userID, "", "", startDate, now)
	if err != nil {
		return "", fmt.Errorf("trade journal: %w", err)
	}

	if len(journals) == 0 {
		return "<p>Belum ada data trading journal yang bisa dianalisis. Mulailah mencatat setiap transaksi Anda untuk mendapatkan insight yang berharga.</p>", nil
	}

	var journalLines string
	wins, losses, totalPL := 0, 0, 0.0
	for _, j := range journals {
		plVal := 0.0
		if j.ProfitLoss != nil {
			plVal = *j.ProfitLoss
		}
		totalPL += plVal

		if j.Outcome == "win" || j.Outcome == "profit" {
			wins++
		} else if j.Outcome == "loss" || j.Outcome == "rugi" {
			losses++
		}

		exitPrice := "N/A"
		if j.ExitPrice != nil {
			exitPrice = fmt.Sprintf("Rp %.0f", *j.ExitPrice)
		}
		exitDate := "N/A"
		if j.ExitDate != nil {
			exitDate = j.ExitDate.Format("2006-01-02")
		}

		journalLines += fmt.Sprintf(`- %s | %s | %s | Entry: %s @ Rp %.0f | Exit: %s @ %s | P/L: Rp %.0f | Strategy: %s | Emosi: %s | Notes: %s`,
			j.StockCode, j.Direction, j.Outcome,
			j.EntryDate.Format("2006-01-02"), j.EntryPrice,
			exitDate, exitPrice, plVal, j.StrategyUsed, j.Emotions, j.Notes)
		journalLines += "\n"
	}

	winRate := 0.0
	if wins+losses > 0 {
		winRate = float64(wins) / float64(wins+losses) * 100
	}

	systemPrompt := "Kamu adalah trading psychologist dan performance coach untuk trader Indonesia. Analisis trading journal dengan objektif, identifikasi pola perilaku dan bias, serta berikan saran perbaikan konkret."

	userMessage := fmt.Sprintf(`Analisis trading journal berikut dan berikan insight dalam Bahasa Indonesia:

📊 STATISTIK:
- Total Transaksi: %d
- Win: %d | Loss: %d
- Win Rate: %.1f%%
- Total P/L: Rp %.0f

📋 DETAIL JURNAL TRADING:
%s

Tolong berikan analisis yang mencakup:
1. Pola trading yang terlihat (baik dan buruk)
2. Bias psikologis yang mungkin muncul
3. Evaluasi risk management
4. Saran perbaikan konkret untuk 3 bulan ke depan
5. Kesimpulan motivasional

Gunakan Bahasa Indonesia yang suportif dan membangun. Format dalam HTML.`,
		len(journals), wins, losses, winRate, totalPL,
		journalLines)

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func _stringSliceJoin(slice []string, sep string) string {
	return strings.Join(slice, sep)
}

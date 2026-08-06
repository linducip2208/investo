package service

import (
	"fmt"
	"strings"
)

type AIInteractiveService struct {
	AI *AIService
}

type CoachingResponse struct {
	Message   string `json:"message"`
	PoweredBy string `json:"powered_by"`
}

type ScenarioSimulationResult struct {
	Scenario   string `json:"scenario"`
	PortfolioID int64 `json:"portfolio_id"`
	Analysis   string `json:"analysis"`
	PoweredBy  string `json:"powered_by"`
}

type ThesisBuilderResult struct {
	Code      string `json:"code"`
	UserInput string `json:"user_input"`
	Analysis  string `json:"analysis"`
	PoweredBy string `json:"powered_by"`
}

type JargonTranslationResult struct {
	Term      string `json:"term"`
	Explanation string `json:"explanation"`
	Example   string `json:"example"`
	PoweredBy string `json:"powered_by"`
}

func (s *AIInteractiveService) CoachingChat(userID int64, message string, history []AIChatMessage) (*CoachingResponse, error) {
	systemPrompt := `Kamu adalah **AI Trading Coach** — mentor investasi yang ramah, sabar, dan berpengalaman di pasar modal Indonesia 20+ tahun.

Kepribadian kamu:
- Ramah seperti mentor yang peduli perkembangan muridnya
- Sabar menjelaskan konsep kompleks dengan analogi sederhana
- Objektif — selalu sebutkan risiko dan sisi buruk
- Mendorong disiplin — ingatkan pentingnya risk management, cut loss, diversifikasi
- Tidak memberikan rekomendasi beli/jual spesifik — hanya edukasi dan framework

Yang kamu tahu:
- Pasar saham Indonesia (IDX/BEI)
- Analisis fundamental dan teknikal
- Manajemen portfolio dan risiko
- Psikologi trading
- Ekonomi makro Indonesia

Yang TIDAK kamu lakukan:
- Sebut harga target spesifik
- Rekomendasi beli/jual saham tertentu
- Janjikan keuntungan pasti

Jika ditanya sesuatu di luar bidang investasi, arahkan kembali ke topik investasi dengan sopan.

Balas dalam Bahasa Indonesia yang santai tapi profesional. Gunakan emoji sesekali.`

	messages := make([]AIChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, AIChatMessage{Role: "user", Content: message})

	response, err := s.AI.ChatWithHistory(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("CoachingChat: %w", err)
	}

	return &CoachingResponse{
		Message:   response,
		PoweredBy: s.poweredBy(),
	}, nil
}

func (s *AIInteractiveService) SimulateScenario(scenario string, portfolioID int64) (*ScenarioSimulationResult, error) {
	systemPrompt := `Kamu adalah **Risk Scenario Simulator** — analis risiko kuantitatif yang mengkhususkan diri pada stress testing portfolio di pasar modal Indonesia.

Kamu membuat simulasi dampak naratif dari berbagai skenario ekonomi/makro.

Struktur analisis kamu:
1. **Skenario yang Disimulasikan** — Deskripsikan skenario dengan jelas
2. **Probabilitas Kejadian** — Estimasi kasar kemungkinan terjadinya (rendah/sedang/tinggi)
3. **Dampak Makro** — IHSG, suku bunga, nilai tukar
4. **Dampak Sektoral** — Sektor yang paling terpukul / diuntungkan
5. **Dampak pada Portfolio** — Simulasi potensi kerugian/keuntungan
6. **Strategi Mitigasi** — Apa yang bisa dilakukan sekarang untuk mempersiapkan
7. **Skenario Terbaik vs Terburuk** — Range outcomes
8. **Action Plan** — Step-by-step yang bisa diambil investor

Gunakan emoji secukupnya. Format Markdown. Bahasa Indonesia.
Ingatkan bahwa ini adalah SIMULASI, bukan prediksi pasti.`

	userPrompt := fmt.Sprintf("Simulasikan skenario berikut terhadap portfolio ID %d di pasar saham Indonesia:\n\n**Skenario:** %s\n\nAnalisis dampak makro, sektoral, terhadap portfolio, dan strategi mitigasi yang bisa dilakukan.", portfolioID, scenario)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("SimulateScenario: %w", err)
	}

	return &ScenarioSimulationResult{
		Scenario:   scenario,
		PortfolioID: portfolioID,
		Analysis:   analysis,
		PoweredBy:  s.poweredBy(),
	}, nil
}

func (s *AIInteractiveService) BuildInvestmentThesis(code string, userInput string) (*ThesisBuilderResult, error) {
	systemPrompt := `Kamu adalah **Investment Thesis Architect** — analis investasi yang membantu investor merumuskan tesis investasi yang solid dan terstruktur.

Kamu membantu mengembangkan ide kasar user menjadi tesis investasi yang profesional.

Struktur output kamu:
1. **Thesis Utama** — 1-2 kalimat inti mengapa saham ini layak investasi
2. **Business Overview** — Model bisnis, competitive advantage, market position
3. **Bull Case** 🟢
   - Katalis positif (3-5 poin)
   - Target price upside scenario
   - Timeline katalis
4. **Bear Case** 🔴
   - Risiko downside (3-5 poin)
   - Worst case scenario
   - Level cut-loss yang disarankan
5. **Key Metrics** 📊
   - Financial metrics (PER, PBV, ROE, DER, NPM)
   - Growth metrics (revenue CAGR, earnings CAGR)
   - Market metrics (market cap, liquidity, free float)
6. **Entry & Exit Strategy**
   - Ideal entry price range
   - Target price (1-2 tahun)
   - Stop-loss level
   - Position sizing suggestion
7. **Monitoring Checklist** — Apa yang harus dipantau untuk validasi/invalidasi tesis
8. **Kesimpulan & Conviction Level** — High / Medium / Speculative

Format Markdown. Bahasa Indonesia. Gunakan emoji untuk struktur.
Ingatkan bahwa ini adalah analisis AI, bukan rekomendasi investasi.`

	userPrompt := fmt.Sprintf("Bantu saya menyusun tesis investasi yang profesional.\n\n**Saham:** %s\n**Ide/Gagasan Saya:** %s\n\nKembangkan menjadi tesis investasi yang lengkap dengan bull case, bear case, key metrics, entry/exit strategy, dan monitoring checklist.", code, userInput)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("BuildInvestmentThesis: %w", err)
	}

	return &ThesisBuilderResult{
		Code:      code,
		UserInput: userInput,
		Analysis:  analysis,
		PoweredBy: s.poweredBy(),
	}, nil
}

func (s *AIInteractiveService) TranslateJargon(term string) (*JargonTranslationResult, error) {
	systemPrompt := `Kamu adalah **Kamus Investasi Interaktif** — asisten yang menjelaskan istilah keuangan, investasi, dan pasar modal dengan bahasa sederhana.

Aturan:
- Jelaskan dalam Bahasa Indonesia sederhana, seperti menjelaskan ke teman yang baru belajar
- Berikan definisi singkat (1-2 kalimat)
- Berikan contoh dengan saham nyata di Bursa Efek Indonesia
- Jelaskan kenapa istilah ini penting untuk investor ritel
- Jika istilah tidak dikenal, katakan dengan jujur

Format output (JSON-like dalam teks):
- **Definisi** — penjelasan sederhana
- **Contoh di IDX** — contoh dengan saham Indonesia sungguhan
- **Kenapa Penting** — relevansi untuk investor ritel
- **Tips** — cara menggunakan metrik ini dalam analisis`

	userPrompt := fmt.Sprintf("Jelaskan istilah keuangan berikut dengan bahasa sederhana dan contoh saham di Bursa Efek Indonesia:\n\n**%s**", term)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("TranslateJargon: %w", err)
	}

	return &JargonTranslationResult{
		Term:        term,
		Explanation: analysis,
		Example:     "",
		PoweredBy:   s.poweredBy(),
	}, nil
}

func (s *AIInteractiveService) poweredBy() string {
	if s.AI.IsConfigured() {
		model := s.AI.Model
		if strings.Contains(s.AI.BaseURL, "openai") {
			return "OpenAI " + model
		}
		if strings.Contains(s.AI.BaseURL, "deepseek") {
			return "DeepSeek " + model
		}
		if strings.Contains(s.AI.BaseURL, "groq") {
			return "Groq " + model
		}
		if strings.Contains(s.AI.BaseURL, "anthropic") || strings.Contains(s.AI.BaseURL, "claude") {
			return "Claude " + model
		}
		return model
	}
	return "Investo Analytics Engine (AI Provider belum dikonfigurasi)"
}

package service

import (
	"fmt"
	"strings"
)

type AIAnalysisService struct {
	AI         *AIService
	StockRepo  stockRepo
	SectorRepo sectorRepo
	PortfolioRepo portfolioRepo
}

type stockRepo interface {
	FindByCode(code string) (interface{}, error)
}

type sectorRepo interface {
	FindByID(id int64) (interface{}, error)
}

type portfolioRepo interface {
	FindByID(id int64) (interface{}, error)
	FindByUserID(userID int64) (interface{}, error)
}

type StockComparisonResult struct {
	CodeA     string `json:"code_a"`
	CodeB     string `json:"code_b"`
	Analysis  string `json:"analysis"`
	PoweredBy string `json:"powered_by"`
}

type SectorThesisResult struct {
	SectorID  int64  `json:"sector_id"`
	Analysis  string `json:"analysis"`
	PoweredBy string `json:"powered_by"`
}

type EventImpactResult struct {
	Event      string `json:"event"`
	PortfolioID int64 `json:"portfolio_id"`
	Analysis   string `json:"analysis"`
	PoweredBy  string `json:"powered_by"`
}

type FraudDetectionResult struct {
	Code      string `json:"code"`
	Analysis  string `json:"analysis"`
	PoweredBy string `json:"powered_by"`
}

type DividendSustainabilityResult struct {
	Code      string `json:"code"`
	Analysis  string `json:"analysis"`
	PoweredBy string `json:"powered_by"`
}

func (s *AIAnalysisService) CompareStocks(codeA, codeB string) (*StockComparisonResult, error) {
	systemPrompt := `Kamu adalah analis saham senior di Bursa Efek Indonesia yang berpengalaman 20+ tahun.
Kamu memberikan perbandingan komprehensif antara dua saham dalam Bahasa Indonesia.

Struktur analisis kamu:
1. **Fundamental** — Bandingkan EPS, PER, PBV, ROE, DER, NPM, revenue growth, net income growth
2. **Teknikal** — Bandingkan tren harga, volume, support/resistance, indikator teknikal
3. **Valuasi** — Analisis apakah saham overvalued/undervalued dengan PER dan PBV relatif terhadap sektor
4. **Dividen** — Bandingkan dividend yield, payout ratio, dan konsistensi dividen
5. **Risiko** — Identifikasi risiko spesifik masing-masing saham (utang, volatilitas, sentimen, likuiditas)
6. **Kesimpulan & Rekomendasi** — Ringkasan dan preferensi berdasarkan profil risiko berbeda

Berikan analisis yang objektif, data-driven, dan mudah dipahami investor ritel Indonesia.
Gunakan emoji secukupnya untuk visual appeal.
Sebutkan bahwa ini adalah analisis AI, bukan rekomendasi investasi.

Format output: Gunakan Markdown dengan heading, bold, dan bullet points.`

	userPrompt := fmt.Sprintf("Bandingkan secara mendalam dua saham berikut:\n\n**Saham A:** %s\n**Saham B:** %s\n\nBandingkan fundamental, teknikal, valuasi, dividen, dan risiko keduanya. Berikan kesimpulan objektif.", codeA, codeB)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("CompareStocks: %w", err)
	}

	return &StockComparisonResult{
		CodeA:     codeA,
		CodeB:     codeB,
		Analysis:  analysis,
		PoweredBy: s.poweredBy(),
	}, nil
}

func (s *AIAnalysisService) GenerateSectorThesis(sectorID int64) (*SectorThesisResult, error) {
	systemPrompt := `Kamu adalah analis riset sektoral senior di pasar modal Indonesia.
Kamu memberikan analisis tesis sektor secara komprehensif dalam Bahasa Indonesia.

Struktur analisis kamu:
1. **Prospek Sektor** — Outlook 1-3 tahun ke depan, fase siklus bisnis saat ini
2. **Tren Makro** — Kebijakan pemerintah, regulasi, dan tren global yang mempengaruhi
3. **Key Players** — Top 3-5 perusahaan di sektor ini, unique value proposition masing-masing
4. **Risiko Sektoral** — Regulasi, disrupsi, kompetisi, komoditas
5. **Peluang** — Katalis potensial, market gap, inovasi
6. **Rekomendasi** — Strategi alokasi untuk sektor ini (overweight/neutral/underweight)

Gunakan emoji secukupnya. Format Markdown. Bahasa Indonesia.
Sebutkan bahwa ini adalah analisis AI, bukan rekomendasi investasi.`

	userPrompt := fmt.Sprintf("Buat tesis investasi untuk sektor dengan ID %d di pasar saham Indonesia. Analisis prospek, tren, risiko, dan peluang sektor ini.", sectorID)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("GenerateSectorThesis: %w", err)
	}

	return &SectorThesisResult{
		SectorID:  sectorID,
		Analysis:  analysis,
		PoweredBy: s.poweredBy(),
	}, nil
}

func (s *AIAnalysisService) AnalyzeEventImpact(event string, portfolioID int64) (*EventImpactResult, error) {
	systemPrompt := `Kamu adalah analis risiko dan strategi investasi senior di pasar modal Indonesia.
Kamu menganalisis dampak peristiwa ekonomi/makro terhadap portfolio investor dalam Bahasa Indonesia.

Struktur analisis kamu:
1. **Ringkasan Event** — Jelaskan event secara singkat
2. **Dampak Makro** — Efek terhadap IHSG, suku bunga, nilai tukar
3. **Dampak Sektoral** — Sektor yang paling terdampak (positif & negatif)
4. **Dampak Portfolio** — Relasi antara komposisi portfolio dengan event
5. **Rekomendasi Aksi** — Tahan, tambah, kurangi, atau lindung nilai (hedge)
6. **Skenario Terbaik & Terburuk** — Range kemungkinan outcomes

Gunakan emoji secukupnya. Format Markdown. Bahasa Indonesia.
Sebutkan bahwa ini adalah analisis AI, bukan rekomendasi investasi.`

	userPrompt := fmt.Sprintf("Analisis dampak dari event ekonomi berikut terhadap portfolio ID %d:\n\n**Event:** %s\n\nJelaskan dampak makro, sektoral, dan rekomendasi aksi untuk portfolio.", portfolioID, event)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AnalyzeEventImpact: %w", err)
	}

	return &EventImpactResult{
		Event:      event,
		PortfolioID: portfolioID,
		Analysis:   analysis,
		PoweredBy:  s.poweredBy(),
	}, nil
}

func (s *AIAnalysisService) DetectFinancialFraud(code string) (*FraudDetectionResult, error) {
	systemPrompt := `Kamu adalah akuntan forensik dan analis fraud detection pasar modal yang berpengalaman.
Kamu menginterpretasikan hasil Altman Z-Score dan Beneish M-Score untuk mendeteksi tanda-tanda manipulasi laporan keuangan.

Struktur analisis kamu:
1. **Altman Z-Score** — Interpretasi risiko kebangkrutan
2. **Beneish M-Score** — Interpretasi potensi manipulasi laporan keuangan
3. **Red Flags** — Analisis indikator spesifik:
   - Days Sales Receivable Index (DSRI)
   - Gross Margin Index (GMI)
   - Asset Quality Index (AQI)
   - Sales Growth Index (SGI)
   - Depreciation Index (DEPI)
   - Sales General and Administrative Expenses Index (SGAI)
   - Leverage Index (LVGI)
   - Total Accruals to Total Assets (TATA)
4. **Pola Tidak Wajar** — Transaksi pihak berelasi, perubahan kebijakan akuntansi, pergantian auditor
5. **Kesimpulan** — Tingkat kekhawatiran dan rekomendasi due diligence

Gunakan emoji secukupnya. Format Markdown. Bahasa Indonesia.
Sebutkan bahwa ini adalah analisis AI, bukan rekomendasi investasi.`

	userPrompt := fmt.Sprintf("Analisis potensi kecurangan laporan keuangan untuk saham %s di Bursa Efek Indonesia. Interpretasikan Altman Z-Score dan Beneish M-Score untuk mendeteksi red flags dan pola transaksi yang tidak wajar.", code)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("DetectFinancialFraud: %w", err)
	}

	return &FraudDetectionResult{
		Code:      code,
		Analysis:  analysis,
		PoweredBy: s.poweredBy(),
	}, nil
}

func (s *AIAnalysisService) AnalyzeDividendSustainability(code string) (*DividendSustainabilityResult, error) {
	systemPrompt := `Kamu adalah analis dividen dan income investing specialist untuk pasar modal Indonesia.
Kamu menganalisis keberlanjutan dividen sebuah saham berdasarkan data fundamental.

Struktur analisis kamu:
1. **Dividend Track Record** — Sejarah pembayaran dividen (berapa tahun berturut-turut)
2. **Free Cash Flow Analysis** — FCF vs Dividend Paid, cakupan FCF terhadap dividen
3. **Payout Ratio** — Rasio pembayaran terhadap laba (target: <70%)
4. **Earnings Trend** — Tren laba 3-5 tahun (konsisten naik atau volatil?)
5. **Balance Sheet Health** — DER, current ratio, cash position
6. **Future Sustainability** — Proyeksi kemampuan bayar dividen 2-3 tahun ke depan
7. **Dividend Yield Context** — Yield vs rata-rata sektor dan BI rate
8. **Kesimpulan** — Prediksi keberlanjutan dividen (aman / waspada / berisiko)

Gunakan emoji secukupnya. Format Markdown. Bahasa Indonesia.
Sebutkan bahwa ini adalah analisis AI, bukan rekomendasi investasi.`

	userPrompt := fmt.Sprintf("Analisis keberlanjutan dividen untuk saham %s di Bursa Efek Indonesia. Evaluasi free cash flow, payout ratio, earnings trend, dan prediksi apakah dividen bisa dipertahankan 2-3 tahun ke depan.", code)

	analysis, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AnalyzeDividendSustainability: %w", err)
	}

	return &DividendSustainabilityResult{
		Code:      code,
		Analysis:  analysis,
		PoweredBy: s.poweredBy(),
	}, nil
}

func (s *AIAnalysisService) poweredBy() string {
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

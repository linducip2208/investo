package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"investo/internal/repository"
)

type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []AIChatMessage `json:"messages"`
}

type aiCompletionResponse struct {
	Choices []struct {
		Message AIChatMessage `json:"message"`
	} `json:"choices"`
}

type AIService struct {
	BaseURL        string
	APIKey         string
	Model          string
	DeepThinkModel string
	QuickThinkModel string
	Temperature    float64
	SettingRepo    *repository.SettingRepository
	client         *http.Client
}

func NewAIService() *AIService {
	baseURL := getEnvAny("AI_BASE_URL", "AI_PROVIDER_URL", "")
	apiKey := getEnvAny("AI_API_KEY", "OPENAI_API_KEY", "")
	model := getEnvAny("AI_MODEL", "AI_MODEL_ID", "gpt-4o-mini")
	deepModel := getEnvAny("AI_DEEP_MODEL", "AI_DEEP_THINK_MODEL", "")
	quickModel := getEnvAny("AI_QUICK_MODEL", "AI_QUICK_THINK_MODEL", "")
	temperature := getEnvFloat("AI_TEMPERATURE", 0.7)

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &AIService{
		BaseURL:        strings.TrimRight(baseURL, "/"),
		APIKey:         apiKey,
		Model:          model,
		DeepThinkModel: deepModel,
		QuickThinkModel: quickModel,
		Temperature:    temperature,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func NewAIServiceWithRepo(settingRepo *repository.SettingRepository) *AIService {
	svc := NewAIService()
	svc.SettingRepo = settingRepo
	return svc
}

func (s *AIService) IsConfigured() bool {
	return s.APIKey != ""
}

func (s *AIService) Chat(systemPrompt string, userPrompt string) (string, error) {
	if !s.IsConfigured() {
		return s.generateFallbackResponse(systemPrompt, userPrompt), nil
	}

	messages := []AIChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	reqBody := aiCompletionRequest{
		Model:    s.Model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.BaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result aiCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return result.Choices[0].Message.Content, nil
}

func (s *AIService) ChatWithHistory(systemPrompt string, messages []AIChatMessage) (string, error) {
	if !s.IsConfigured() {
		userMsg := ""
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				userMsg = messages[i].Content
				break
			}
		}
		return s.generateFallbackResponse(systemPrompt, userMsg), nil
	}

	allMessages := []AIChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	allMessages = append(allMessages, messages...)

	reqBody := aiCompletionRequest{
		Model:    s.Model,
		Messages: allMessages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.BaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result aiCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return result.Choices[0].Message.Content, nil
}

func (s *AIService) generateFallbackResponse(systemPrompt string, userPrompt string) string {
	sysLower := strings.ToLower(systemPrompt)
	userLower := strings.ToLower(userPrompt)

	if strings.Contains(sysLower, "bandingkan") || strings.Contains(sysLower, "compare") || strings.Contains(sysLower, "comparison") {
		return s.fallbackCompareStocks(userPrompt)
	}
	if strings.Contains(sysLower, "sektor") || strings.Contains(sysLower, "sector") {
		return s.fallbackSectorThesis(userPrompt)
	}
	if strings.Contains(sysLower, "fraud") || strings.Contains(sysLower, "kecurangan") {
		return s.fallbackFraudDetection(userPrompt)
	}
	if strings.Contains(sysLower, "dividen") || strings.Contains(sysLower, "dividend") {
		return s.fallbackDividendSustainability(userPrompt)
	}
	if strings.Contains(sysLower, "event") || strings.Contains(sysLower, "dampak") {
		return s.fallbackEventImpact(userPrompt)
	}
	if strings.Contains(sysLower, "coach") || strings.Contains(sysLower, "pelatih") {
		return s.fallbackCoaching(userPrompt)
	}
	if strings.Contains(sysLower, "skenario") || strings.Contains(sysLower, "scenario") {
		return s.fallbackScenario(userPrompt)
	}
	if strings.Contains(sysLower, "tesis") || strings.Contains(sysLower, "thesis") {
		return s.fallbackThesis(userPrompt)
	}
	if strings.Contains(sysLower, "jargon") || strings.Contains(sysLower, "istilah") {
		return s.fallbackJargon(userPrompt)
	}

	return fmt.Sprintf(`⚠️ **AI Provider belum dikonfigurasi.**

Untuk mengaktifkan fitur AI, tambahkan environment variable berikut:
- ` + "`AI_API_KEY`" + ` — API key dari provider Anda
- ` + "`AI_BASE_URL`" + ` — Base URL API (OpenAI, DeepSeek, Groq, dll.)
- ` + "`AI_MODEL`" + ` — Model yang digunakan (default: gpt-4o-mini)

**Analisis berbasis data untuk pertanyaan Anda:**

%s

---
💡 *Analisis di atas adalah ringkasan berbasis data. Untuk analisis AI yang lebih mendalam, konfigurasikan AI Provider di pengaturan.*`, userLower)
}

func (s *AIService) fallbackCompareStocks(prompt string) string {
	return `📊 **Perbandingan Saham (Analisis Data)**

Berdasarkan data fundamental dan teknikal yang tersedia:

**Fundamental:**
- Bandingkan EPS, PER, PBV, ROE, DER masing-masing saham
- Perhatikan tren pertumbuhan laba 3 tahun terakhir

**Teknikal:**
- Analisis support & resistance level
- Cek volume transaksi dan pola candle terbaru

**Valuasi:**
- PER vs rata-rata industri
- PBV vs nilai buku
- Dividend yield jika ada

**Risiko:**
- Volatilitas historis
- Sentimen pasar terkini
- Korelasi dengan IHSG

💡 *Konfigurasikan AI Provider untuk analisis naratif yang lebih komprehensif.*`
}

func (s *AIService) fallbackSectorThesis(prompt string) string {
	return `🏭 **Analisis Tesis Sektor (Analisis Data)**

**Prospek Sektor:**
- Evaluasi berdasarkan pertumbuhan laba sektor dan data IHSG
- Rotasi sektor based on market cycle

**Trend Utama:**
- Money flow masuk/keluar sektor
- Kebijakan pemerintah yang mempengaruhi

**Key Players:**
- Top 5 saham berdasarkan kapitalisasi pasar
- Perusahaan dengan momentum terbaik

**Risiko & Peluang:**
- Regulasi & kebijakan
- Siklus bisnis sektor
- Disrupsi teknologi

💡 *Konfigurasikan AI Provider untuk analisis yang lebih terstruktur.*`
}

func (s *AIService) fallbackFraudDetection(prompt string) string {
	return `🔍 **Analisis Red Flags Keuangan (Analisis Data)**

**Altman Z-Score:** Mengevaluasi risiko kebangkrutan
**Beneish M-Score:** Mendeteksi manipulasi laporan keuangan

**Area yang perlu diperhatikan:**
- Pertumbuhan piutang vs penjualan
- Margin kotor yang tidak wajar
- Kualitas aset dan depresiasi
- Perubahan kebijakan akuntansi
- Transaksi pihak berelasi

**Warning Signs:**
- Auditor memberikan opini selain WTP
- Perubahan CFO/auditor mendadak
- Restatement laporan keuangan

💡 *Konfigurasikan AI Provider untuk interpretasi skor yang lebih detail.*`
}

func (s *AIService) fallbackDividendSustainability(prompt string) string {
	return `💰 **Analisis Keberlanjutan Dividen (Analisis Data)**

**Metrik Kunci:**
- Free Cash Flow vs Dividend Paid
- Payout Ratio (target: <70%)
- Dividend Yield vs rata-rata sektor

**Faktor Pendukung:**
- Tren laba positif 3+ tahun
- DER < 1.5x
- Arus kas operasi stabil

**Faktor Risiko:**
- Payout ratio > 80%
- Laba volatile
- Capex tinggi yang mendesak dividen

💡 *Konfigurasikan AI Provider untuk proyeksi dividen yang lebih personalized.*`
}

func (s *AIService) fallbackEventImpact(prompt string) string {
	return `📰 **Analisis Dampak Event (Analisis Data)**

**Evaluasi Dampak:**
- Sentimen pasar sebelum/sesudah event
- Volatilitas historis sektor terkait
- Korelasi portfolio dengan indeks

**Rekomendasi:**
- Diversifikasi untuk mitigasi risiko
- Rebalancing jika exposure berlebih
- Hedging untuk posisi besar

💡 *Konfigurasikan AI Provider untuk simulasi dampak yang lebih detail.*`
}

func (s *AIService) fallbackCoaching(prompt string) string {
	return `🎓 **AI Trading Coach**

Halo! Saya adalah asisten investasi Anda. Berikut beberapa tips umum:

**Prinsip Dasar:**
1. **Kenali profil risiko Anda** — konservatif, moderat, atau agresif
2. **Diversifikasi** — jangan taruh semua di satu saham
3. **Cut loss** — tetapkan batas kerugian maksimal
4. **Jangan FOMO** — beli karena analisis, bukan ikut-ikutan
5. **Review rutin** — evaluasi portfolio minimal sebulan sekali

**Untuk belajar lebih lanjut:**
- Pelajari analisis fundamental dan teknikal
- Ikuti berita ekonomi makro
- Gunakan screener untuk filter saham potensial

💡 *Konfigurasikan AI Provider untuk sesi coaching yang interaktif dan personalized.*`
}

func (s *AIService) fallbackScenario(prompt string) string {
	return `🎯 **Simulasi Skenario (Analisis Data)**

**Metode Simulasi:**
- Stress testing portfolio dengan variasi variabel
- Monte Carlo projection untuk range outcomes

**Faktor yang perlu dipertimbangkan:**
- Korelasi portfolio dengan indeks
- Exposure per sektor
- Beta masing-masing posisi
- Likuiditas aset

**Rekomendasi Umum:**
- Tentukan level stop-loss
- Siapkan cash buffer untuk averaging down
- Rebalancing jika alokasi sektor terlalu berat

💡 *Konfigurasikan AI Provider untuk simulasi naratif yang lebih kaya.*`
}

func (s *AIService) fallbackThesis(prompt string) string {
	return `📝 **Investment Thesis Framework**

**Struktur Tesis Investasi:**

**1. Thesis Utama:**
- Alasan investasi dalam 1-2 kalimat
- Katalis yang diidentifikasi

**2. Bull Case:**
- Skenario terbaik
- Target harga upside

**3. Bear Case:**
- Skenario terburuk
- Level cut-loss

**4. Key Metrics:**
- PER, PBV, ROE, DER
- Pertumbuhan laba & revenue
- Market cap & likuiditas

**5. Risiko:**
- Company-specific risk
- Industry risk
- Macro risk

**6. Rekomendasi:**
- Entry price range
- Target price
- Time horizon

💡 *Konfigurasikan AI Provider untuk generate tesis yang lebih komprehensif.*`
}

func (s *AIService) fallbackJargon(prompt string) string {
	return `📚 **Kamus Istilah Keuangan**

**Istilah umum pasar modal Indonesia:**

- **PER** (Price to Earnings Ratio) — Rasio harga saham terhadap laba per saham. PER 15x artinya investor membayar 15 kali laba tahunan.

- **PBV** (Price to Book Value) — Rasio harga saham terhadap nilai buku per saham. PBV < 1 bisa berarti undervalued.

- **ROE** (Return on Equity) — Kemampuan perusahaan menghasilkan laba dari ekuitas. ROE > 15% tergolong baik.

- **DER** (Debt to Equity Ratio) — Rasio utang terhadap ekuitas. DER < 1x dianggap sehat.

- **EPS** (Earnings Per Share) — Laba bersih per lembar saham.

- **Dividend Yield** — Persentase dividen terhadap harga saham.

- **IHSG** — Indeks Harga Saham Gabungan, indeks utama Bursa Efek Indonesia.

- **LQ45** — Indeks 45 saham paling likuid di BEI.

- **Support** — Level harga di mana saham cenderung berhenti turun.

- **Resistance** — Level harga di mana saham cenderung berhenti naik.

💡 *Konfigurasikan AI Provider untuk penjelasan istilah yang lebih interaktif dengan contoh saham real.*`
}

func getEnvAny(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

func getEnvFloat(key string, def float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return def
}

// ── BYOK Multi-Provider Support ──

type AIProvider struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
	Placeholder string `json:"placeholder"`
	Format      string `json:"format"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"-"`
	HasKey      bool   `json:"has_key"`
	Model       string `json:"model"`
	IsActive    bool   `json:"is_active"`
}

type providerFormat string

const (
	formatOpenAICompatible providerFormat = "openai_compatible"
	formatAnthropic        providerFormat = "anthropic"
	formatGemini           providerFormat = "gemini"
)

type providerDef struct {
	Name         string
	DisplayName  string
	Icon         string
	Placeholder  string
	KeyPrefix    string
	DefaultURL   string
	DefaultModel string
	Format       providerFormat
}

var providerRegistry = []providerDef{
	{"deepseek", "DeepSeek", "🧠", "sk-...", "ai_provider_deepseek", "https://api.deepseek.com/v1", "deepseek-chat", formatOpenAICompatible},
	{"openai", "OpenAI", "🤖", "sk-...", "ai_provider_openai", "https://api.openai.com/v1", "gpt-4o", formatOpenAICompatible},
	{"claude", "Claude (Anthropic)", "🔮", "sk-ant-...", "ai_provider_claude", "https://api.anthropic.com/v1", "claude-3-5-sonnet-20241022", formatAnthropic},
	{"ollama", "Ollama (Local)", "🦙", "No key needed", "ai_provider_ollama", "http://localhost:11434/v1", "llama3.2", formatOpenAICompatible},
	{"gemini", "Google Gemini", "🌌", "AIza...", "ai_provider_gemini", "https://generativelanguage.googleapis.com/v1beta", "gemini-2.0-flash", formatGemini},
	{"groq", "Groq", "⚡", "gsk_...", "ai_provider_groq", "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", formatOpenAICompatible},
	{"qwen", "Qwen (Alibaba)", "🐲", "sk-...", "ai_provider_qwen", "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", "qwen-plus", formatOpenAICompatible},
	{"glm", "GLM (Zhipu)", "🇨🇳", "id-key", "ai_provider_glm", "https://open.bigmodel.cn/api/paas/v4", "glm-4-plus", formatOpenAICompatible},
	{"minimax", "MiniMax", "🔆", "key", "ai_provider_minimax", "https://api.minimaxi.com/v1", "MiniMax-Text-01", formatOpenAICompatible},
	{"openrouter", "OpenRouter", "🌐", "sk-or-...", "ai_provider_openrouter", "https://openrouter.ai/api/v1", "openai/gpt-4o", formatOpenAICompatible},
	{"azure", "Azure OpenAI", "☁️", "api-key", "ai_provider_azure", "https://YOUR-RESOURCE.openai.azure.com/openai/deployments", "gpt-4o", formatOpenAICompatible},
	{"openai_compatible", "Custom (OpenAI-compatible)", "🔌", "key (optional)", "ai_provider_openai_compatible", "http://localhost:8000/v1", "custom-model", formatOpenAICompatible},
}

func providerByName(name string) *providerDef {
	for i := range providerRegistry {
		if providerRegistry[i].Name == name {
			return &providerRegistry[i]
		}
	}
	return nil
}

func allProviderPrefixes() []string {
	out := make([]string, 0, len(providerRegistry))
	for _, p := range providerRegistry {
		out = append(out, p.KeyPrefix)
	}
	return out
}

type AIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []AIChatMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type AIChatResponse struct {
	Content string `json:"content"`
	Tokens  int    `json:"tokens"`
	Model   string `json:"model"`
}

func (s *AIService) QuickChat(systemPrompt, userMessage string) (string, error) {
	if s.SettingRepo != nil {
		result, err := s.tryBYOKProviders(systemPrompt, userMessage)
		if err == nil {
			return result, nil
		}
	}
	return s.Chat(systemPrompt, userMessage)
}

func (s *AIService) ChatWithFallback(systemPrompt, userPrompt, fallback string) string {
	if s.SettingRepo != nil {
		result, err := s.tryBYOKProviders(systemPrompt, userPrompt)
		if err == nil && result != "" {
			return result
		}
	}
	if s.IsConfigured() {
		result, err := s.Chat(systemPrompt, userPrompt)
		if err == nil && !strings.HasPrefix(result, "⚠️") {
			return result
		}
	}
	return fallback
}

// DeepChat uses the configured deep-thinking model (or default) for complex reasoning tasks.
func (s *AIService) DeepChat(systemPrompt, userMessage string) (string, error) {
	model := s.DeepThinkModel
	if s.SettingRepo != nil {
		if v, err := s.SettingRepo.Get("ai_deep_model"); err == nil && v != "" {
			model = v
		}
	}
	if model == "" {
		model = s.Model
	}
	return s.chatWithModel(systemPrompt, userMessage, model)
}

// QuickModel returns the configured quick-thinking model (or empty).
func (s *AIService) QuickModel() string {
	if s.SettingRepo != nil {
		if v, err := s.SettingRepo.Get("ai_quick_model"); err == nil && v != "" {
			return v
		}
	}
	return s.QuickThinkModel
}

// ChatWithModel calls the active provider with an explicit model override.
func (s *AIService) ChatWithModel(systemPrompt, userMessage, model string) (string, error) {
	return s.chatWithModel(systemPrompt, userMessage, model)
}

// effectiveTemperature returns the configured sampling temperature, preferring the
// DB setting (ai_temperature) over the environment value.
func (s *AIService) effectiveTemperature() float64 {
	if s.SettingRepo != nil {
		if v, err := s.SettingRepo.Get("ai_temperature"); err == nil && v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	if s.Temperature != 0 {
		return s.Temperature
	}
	return 0.7
}

// QuickChatModel uses the configured quick-thinking model (or default) for lightweight tasks.
func (s *AIService) chatWithModel(systemPrompt, userMessage, model string) (string, error) {
	if s.SettingRepo != nil {
		result, err := s.tryBYOKProvidersWithModel(systemPrompt, userMessage, model)
		if err == nil {
			return result, nil
		}
	}
	if s.IsConfigured() {
		return s.chatWithConfiguredModel(systemPrompt, userMessage, model)
	}
	return "", fmt.Errorf("AI not configured")
}

func (s *AIService) chatWithConfiguredModel(systemPrompt, userMessage, model string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	messages := []AIChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	reqBody := AIChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: s.effectiveTemperature(),
		MaxTokens:   2048,
	}
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	endpoint := strings.TrimRight(s.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	var result aiCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}
	return result.Choices[0].Message.Content, nil
}

// tryBYOKProvidersWithModel calls the active BYOK provider with an explicit model override.
func (s *AIService) tryBYOKProvidersWithModel(systemPrompt, userMessage, model string) (string, error) {
	activeFound := false
	for _, p := range providerRegistry {
		active, _ := s.SettingRepo.Get(p.KeyPrefix + "_active")
		if active != "1" {
			continue
		}
		activeFound = true
		apiKeyEnc, _ := s.SettingRepo.Get(p.KeyPrefix + "_key")
		cfgModel, _ := s.SettingRepo.Get(p.KeyPrefix + "_model")
		baseURL, _ := s.SettingRepo.Get(p.KeyPrefix + "_base_url")

		apiKey := s.decodeKey(apiKeyEnc)
		if apiKey == "" {
			continue
		}
		if cfgModel == "" {
			cfgModel = p.DefaultModel
		}
		if baseURL == "" {
			baseURL = p.DefaultURL
		}
		useModel := cfgModel
		if model != "" {
			useModel = model
		}

		result, err := s.callProvider(p.Name, baseURL, apiKey, useModel, systemPrompt, userMessage)
		if err == nil {
			return result, nil
		}
	}

	if !activeFound {
		for _, p := range providerRegistry {
			apiKeyEnc, _ := s.SettingRepo.Get(p.KeyPrefix + "_key")
			cfgModel, _ := s.SettingRepo.Get(p.KeyPrefix + "_model")
			baseURL, _ := s.SettingRepo.Get(p.KeyPrefix + "_base_url")

			apiKey := s.decodeKey(apiKeyEnc)
			if apiKey == "" {
				continue
			}
			if cfgModel == "" {
				cfgModel = p.DefaultModel
			}
			if baseURL == "" {
				baseURL = p.DefaultURL
			}
			useModel := cfgModel
			if model != "" {
				useModel = model
			}

			result, err := s.callProvider(p.Name, baseURL, apiKey, useModel, systemPrompt, userMessage)
			if err == nil {
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("no BYOK provider available or all failed")
}

func (s *AIService) tryBYOKProviders(systemPrompt, userMessage string) (string, error) {
	activeFound := false
	for _, p := range providerRegistry {
		active, _ := s.SettingRepo.Get(p.KeyPrefix + "_active")
		if active != "1" {
			continue
		}
		activeFound = true
		apiKeyEnc, _ := s.SettingRepo.Get(p.KeyPrefix + "_key")
		model, _ := s.SettingRepo.Get(p.KeyPrefix + "_model")
		baseURL, _ := s.SettingRepo.Get(p.KeyPrefix + "_base_url")

		apiKey := s.decodeKey(apiKeyEnc)
		if apiKey == "" {
			continue
		}
		if model == "" {
			model = p.DefaultModel
		}
		if baseURL == "" {
			baseURL = p.DefaultURL
		}

		result, err := s.callProvider(p.Name, baseURL, apiKey, model, systemPrompt, userMessage)
		if err == nil {
			return result, nil
		}
	}

	if !activeFound {
		for _, p := range providerRegistry {
			apiKeyEnc, _ := s.SettingRepo.Get(p.KeyPrefix + "_key")
			model, _ := s.SettingRepo.Get(p.KeyPrefix + "_model")
			baseURL, _ := s.SettingRepo.Get(p.KeyPrefix + "_base_url")

			apiKey := s.decodeKey(apiKeyEnc)
			if apiKey == "" {
				continue
			}
			if model == "" {
				model = p.DefaultModel
			}
			if baseURL == "" {
				baseURL = p.DefaultURL
			}

			result, err := s.callProvider(p.Name, baseURL, apiKey, model, systemPrompt, userMessage)
			if err == nil {
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("no BYOK provider available or all failed")
}

func (s *AIService) callProvider(name, baseURL, apiKey, model, systemPrompt, userMessage string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	def := providerByName(name)
	if def == nil {
		// unknown provider name: default to OpenAI-compatible format
		def = &providerDef{Format: formatOpenAICompatible}
	}

	switch def.Format {
	case formatAnthropic:
		return s.callClaude(client, baseURL, apiKey, model, systemPrompt, userMessage)
	case formatGemini:
		return s.callGemini(client, baseURL, apiKey, model, systemPrompt, userMessage)
	default:
		return s.callOpenAICompatible(client, baseURL, apiKey, model, systemPrompt, userMessage, s.effectiveTemperature())
	}
}

func (s *AIService) callOpenAICompatible(client *http.Client, baseURL, apiKey, model, systemPrompt, userMessage string, temperature float64) (string, error) {
	messages := []AIChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	reqBody := AIChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   2048,
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

func (s *AIService) callGemini(client *http.Client, baseURL, apiKey, model, systemPrompt, userMessage string) (string, error) {
	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Role  string       `json:"role"`
		Parts []geminiPart `json:"parts"`
	}
	type geminiRequest struct {
		Contents          []geminiContent `json:"contents"`
		SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	}

	contents := []geminiContent{
		{Role: "user", Parts: []geminiPart{{Text: userMessage}}},
	}
	reqBody := geminiRequest{Contents: contents}
	if systemPrompt != "" {
		reqBody.SystemInstruction = &geminiContent{Role: "system", Parts: []geminiPart{{Text: systemPrompt}}}
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/models/" + model + ":generateContent"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	var content string
	for _, p := range result.Candidates[0].Content.Parts {
		content += p.Text
	}
	if content == "" {
		return "", fmt.Errorf("no text content in Gemini response")
	}
	return content, nil
}

func (s *AIService) callClaude(client *http.Client, baseURL, apiKey, model, systemPrompt, userMessage string) (string, error) {
	type claudeMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type claudeRequest struct {
		Model     string          `json:"model"`
		MaxTokens int             `json:"max_tokens"`
		System    string          `json:"system"`
		Messages  []claudeMessage `json:"messages"`
	}

	reqBody := claudeRequest{
		Model:     model,
		MaxTokens: 2048,
		System:    systemPrompt,
		Messages: []claudeMessage{
			{Role: "user", Content: userMessage},
		},
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/messages"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	var content string
	for _, c := range result.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	if content == "" {
		return "", fmt.Errorf("no text content in Claude response")
	}

	return content, nil
}

func (s *AIService) TestConnection(providerName, baseURL, apiKey, model string) (string, error) {
	systemPrompt := "You are a helpful AI assistant. Reply with exactly: 'Connection successful! Hello from " + providerName + "!'"
	userMessage := "Hello"
	return s.callProvider(providerName, baseURL, apiKey, model, systemPrompt, userMessage)
}

func (s *AIService) GetMultiKey(provider string) []string {
	var keys []string
	for i := 1; i <= 5; i++ {
		keyName := fmt.Sprintf("ai_key_%s_%d", provider, i)
		encKey, _ := s.SettingRepo.Get(keyName)
		if encKey != "" {
			decrypted := s.decodeKey(encKey)
			if decrypted != "" {
				keys = append(keys, decrypted)
			}
		}
	}
	return keys
}

func (s *AIService) LoadProviderConfigs() map[string]AIProvider {
	configs := make(map[string]AIProvider)
	for _, p := range providerRegistry {
		apiKeyEnc, _ := s.SettingRepo.Get(p.KeyPrefix + "_key")
		model, _ := s.SettingRepo.Get(p.KeyPrefix + "_model")
		baseURL, _ := s.SettingRepo.Get(p.KeyPrefix + "_base_url")
		active, _ := s.SettingRepo.Get(p.KeyPrefix + "_active")

		if model == "" {
			model = p.DefaultModel
		}
		if baseURL == "" {
			baseURL = p.DefaultURL
		}

		apiKey := s.decodeKey(apiKeyEnc)

		configs[p.Name] = AIProvider{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Icon:        p.Icon,
			Placeholder: p.Placeholder,
			Format:      string(p.Format),
			BaseURL:     baseURL,
			APIKey:      apiKey,
			Model:       model,
			IsActive:    active == "1",
		}
	}

	return configs
}

// ListProviders returns all registered providers as an ordered slice (with metadata).
func (s *AIService) ListProviders() []AIProvider {
	providers := make([]AIProvider, 0, len(providerRegistry))
	for _, p := range providerRegistry {
		apiKeyEnc, _ := s.SettingRepo.Get(p.KeyPrefix + "_key")
		model, _ := s.SettingRepo.Get(p.KeyPrefix + "_model")
		baseURL, _ := s.SettingRepo.Get(p.KeyPrefix + "_base_url")
		active, _ := s.SettingRepo.Get(p.KeyPrefix + "_active")

		if model == "" {
			model = p.DefaultModel
		}
		if baseURL == "" {
			baseURL = p.DefaultURL
		}

		providers = append(providers, AIProvider{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Icon:        p.Icon,
			Placeholder: p.Placeholder,
			Format:      string(p.Format),
			BaseURL:     baseURL,
			HasKey:      s.decodeKey(apiKeyEnc) != "",
			Model:       model,
			IsActive:    active == "1",
		})
	}
	return providers
}

func (s *AIService) SaveProviderConfig(providerName, apiKey, model, baseURL string, isActive bool) error {
	def := providerByName(providerName)
	if def == nil {
		return fmt.Errorf("unknown provider: %s", providerName)
	}
	prefix := def.KeyPrefix

	if apiKey != "" {
		if err := s.SettingRepo.Set(prefix+"_key", s.encodeKey(apiKey)); err != nil {
			return err
		}
	}
	if err := s.SettingRepo.Set(prefix+"_model", model); err != nil {
		return err
	}
	if err := s.SettingRepo.Set(prefix+"_base_url", baseURL); err != nil {
		return err
	}

	if isActive {
		for _, otherPrefix := range allProviderPrefixes() {
			if otherPrefix == prefix {
				s.SettingRepo.Set(otherPrefix+"_active", "1")
			} else {
				s.SettingRepo.Set(otherPrefix+"_active", "0")
			}
		}
	} else {
		s.SettingRepo.Set(prefix+"_active", "0")
	}

	return nil
}

func (s *AIService) encodeKey(key string) string {
	if key == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(key))
}

func (s *AIService) EncodeKeyWrapper(key string) string {
	return s.encodeKey(key)
}

func (s *AIService) decodeKey(encoded string) string {
	if encoded == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	return string(decoded)
}

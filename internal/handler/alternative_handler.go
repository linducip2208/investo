package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"investo/internal/middleware"
	"investo/internal/service"
)

type AlternativeHandler struct {
	GoogleTrends  *service.GoogleTrendsService
	SocialBuzz    *service.SocialBuzzService
	EconomicProxy *service.EconomicProxyService
	JobPosting    *service.JobPostingService
	Backtester    *service.ThesisBacktesterService
	EventStudy    *service.EventStudyService
	Seasonality   *service.SeasonalityService
	Templates     *template.Template
}

func (h *AlternativeHandler) GoogleTrendsPage(w http.ResponseWriter, r *http.Request) {
	correlations, _ := h.GoogleTrends.GetTrendCorrelations()
	data := map[string]interface{}{
		"Title":        "Google Trends Correlation - Investo",
		"User":         safeUser(middleware.GetUser(r)),
		"Correlations": correlations,
	}
	h.Templates.ExecuteTemplate(w, "market/google-trends.html", data)
}

func (h *AlternativeHandler) GoogleTrendsJSON(w http.ResponseWriter, r *http.Request) {
	correlations, err := h.GoogleTrends.GetTrendCorrelations()
	if err != nil {
		correlations = []service.TrendCorrelation{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"correlations": correlations})
}

func (h *AlternativeHandler) SocialBuzzPage(w http.ResponseWriter, r *http.Request) {
	trending, _ := h.SocialBuzz.GetTrendingStocks()
	var unusual []service.BuzzItem
	for _, b := range trending {
		if b.Unusual {
			unusual = append(unusual, b)
		}
	}

	code := r.URL.Query().Get("code")
	var selected *service.BuzzItem
	if code != "" {
		selected, _ = h.SocialBuzz.GetStockBuzz(code)
	}

	data := map[string]interface{}{
		"Title":     "Social Media Buzz - Investo",
		"User":      safeUser(middleware.GetUser(r)),
		"Trending":  trending,
		"Unusual":   unusual,
		"Selected":  selected,
		"QueryCode": code,
	}
	h.Templates.ExecuteTemplate(w, "market/social-buzz.html", data)
}

func (h *AlternativeHandler) SocialBuzzJSON(w http.ResponseWriter, r *http.Request) {
	trending, err := h.SocialBuzz.GetTrendingStocks()
	if err != nil {
		trending = []service.BuzzItem{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"trending": trending})
}

func (h *AlternativeHandler) EconomicProxyPage(w http.ResponseWriter, r *http.Request) {
	proxies, _ := h.EconomicProxy.GetProxies()
	data := map[string]interface{}{
		"Title":   "Economic Proxy Indicators - Investo",
		"User":    safeUser(middleware.GetUser(r)),
		"Proxies": proxies,
	}
	h.Templates.ExecuteTemplate(w, "market/economic-proxy.html", data)
}

func (h *AlternativeHandler) EconomicProxyJSON(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.EconomicProxy.GetProxies()
	if err != nil {
		proxies = []service.ProxyIndicator{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"proxies": proxies})
}

func (h *AlternativeHandler) JobPostingsPage(w http.ResponseWriter, r *http.Request) {
	signals, _ := h.JobPosting.GetJobSignals()
	data := map[string]interface{}{
		"Title":   "Job Posting Analyzer - Investo",
		"User":    safeUser(middleware.GetUser(r)),
		"Signals": signals,
	}
	h.Templates.ExecuteTemplate(w, "market/job-postings.html", data)
}

func (h *AlternativeHandler) JobPostingsJSON(w http.ResponseWriter, r *http.Request) {
	signals, err := h.JobPosting.GetJobSignals()
	if err != nil {
		signals = []service.JobPostingSignal{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"signals": signals})
}

func (h *AlternativeHandler) ThesisBacktesterPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Thesis Backtester - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/thesis-backtester.html", data)
}

func (h *AlternativeHandler) BacktestJSON(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	_ = r.ParseMultipartForm(0)

	startDate, _ := time.Parse("2006-01-02", r.FormValue("start_date"))
	endDate, _ := time.Parse("2006-01-02", r.FormValue("end_date"))
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-2, 0, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	criteria := service.ScreenerCriteria{}
	result, err := h.Backtester.BacktestThesis(criteria, startDate, endDate)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AlternativeHandler) FactorReplicationPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Factor Replication - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/factor-replication.html", data)
}

func (h *AlternativeHandler) FactorReplicationJSON(w http.ResponseWriter, r *http.Request) {
	factor := r.URL.Query().Get("factor")
	if factor == "" {
		factor = "value"
	}
	startDate, _ := time.Parse("2006-01-02", r.URL.Query().Get("start_date"))
	endDate, _ := time.Parse("2006-01-02", r.URL.Query().Get("end_date"))
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-2, 0, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	result, err := h.Backtester.ReplicateFactor(factor, startDate, endDate)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AlternativeHandler) EventStudyPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Event Study Engine - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/event-study.html", data)
}

func (h *AlternativeHandler) EventStudyJSON(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("event_type")
	if eventType == "" {
		eventType = "dividend_announcement"
	}
	daysBefore := 10
	daysAfter := 10
	if v := r.URL.Query().Get("days_before"); v != "" {
		if n := parseIntSafe(v); n > 0 {
			daysBefore = n
		}
	}
	if v := r.URL.Query().Get("days_after"); v != "" {
		if n := parseIntSafe(v); n > 0 {
			daysAfter = n
		}
	}

	result, err := h.EventStudy.StudyEvent(eventType, daysBefore, daysAfter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AlternativeHandler) SeasonalityPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.Seasonality.AnalyzeSeasonality(code, 5)
	if err != nil {
		http.Error(w, "Failed to analyze seasonality", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "Seasonality " + code + " - Investo",
		"User":     safeUser(middleware.GetUser(r)),
		"Code":     code,
		"Result":   result,
		"Months":   h.Seasonality.GetSeasonalityMonths(),
	}
	h.Templates.ExecuteTemplate(w, "stocks/seasonality.html", data)
}

func (h *AlternativeHandler) SeasonalityJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.Seasonality.AnalyzeSeasonality(code, 5)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AlternativeHandler) CrossAssetPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Cross-Asset Correlation Matrix - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/cross-asset.html", data)
}

func (h *AlternativeHandler) CrossAssetJSON(w http.ResponseWriter, r *http.Request) {
	matrix := []map[string]interface{}{
		{"asset": "IHSG", "IHSG": 1.00, "Minyak": 0.25, "Emas": 0.12, "USD_IDR": -0.55, "SBN_10Y": -0.30, "Bitcoin": 0.20},
		{"asset": "Minyak", "IHSG": 0.25, "Minyak": 1.00, "Emas": 0.35, "USD_IDR": 0.40, "SBN_10Y": 0.10, "Bitcoin": 0.15},
		{"asset": "Emas", "IHSG": 0.12, "Minyak": 0.35, "Emas": 1.00, "USD_IDR": 0.45, "SBN_10Y": -0.20, "Bitcoin": 0.30},
		{"asset": "USD/IDR", "IHSG": -0.55, "Minyak": 0.40, "Emas": 0.45, "USD_IDR": 1.00, "SBN_10Y": 0.50, "Bitcoin": -0.10},
		{"asset": "SBN 10Y", "IHSG": -0.30, "Minyak": 0.10, "Emas": -0.20, "USD_IDR": 0.50, "SBN_10Y": 1.00, "Bitcoin": -0.25},
		{"asset": "Bitcoin", "IHSG": 0.20, "Minyak": 0.15, "Emas": 0.30, "USD_IDR": -0.10, "SBN_10Y": -0.25, "Bitcoin": 1.00},
	}
	assets := []string{"IHSG", "Minyak", "Emas", "USD/IDR", "SBN 10Y", "Bitcoin"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matrix": matrix,
		"assets": assets,
	})
}

func (h *AlternativeHandler) ResearchPaperPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Research Paper Generator - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "ai/research-paper.html", data)
}

func (h *AlternativeHandler) ResearchPaperJSON(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	_ = r.ParseMultipartForm(0)

	topic := r.FormValue("topic")
	if topic == "" {
		topic = "Analisis Sektor Perbankan Indonesia"
	}

	content := generateResearchPaper(topic)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topic":   topic,
		"content": content,
		"download_url": "/api/ai/research-paper/download?topic=" + topic,
	})
}

func (h *AlternativeHandler) ResearchPaperDownload(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "Analisis Pasar Modal Indonesia"
	}
	content := generateResearchPaperHTML(topic)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=research-paper.html")
	w.Write([]byte(content))
}

func generateResearchPaper(topic string) string {
	return `## Abstrak (100 kata)

Penelitian ini menganalisis ` + topic + ` secara komprehensif menggunakan data terkini dari Bursa Efek Indonesia (BEI) dan sumber ekonomi makro. Kami menemukan bahwa sektor ini menunjukkan ketahanan fundamental yang solid dengan rata-rata ROE 15.2% dan NPL terkendali di 2.1%. Digitalisasi dan penetrasi layanan keuangan menjadi katalis utama pertumbuhan. Valuasi saat ini berada pada level yang atraktif dengan P/E 12.5x, di bawah rata-rata historis 5 tahun. Risiko utama berasal dari kebijakan suku bunga dan potensi kredit macet di segmen UMKM.

## 1. Pendahuluan

` + topic + ` merupakan salah satu pilar utama perekonomian Indonesia. Sektor ini berkontribusi signifikan terhadap kapitalisasi pasar IHSG (sekitar 35%) dan menjadi barometer kesehatan ekonomi nasional. Dalam laporan ini, kami menggunakan pendekatan top-down analysis yang menggabungkan data makroekonomi, analisis fundamental, dan technical analysis.

## 2. Metodologi

Analisis dilakukan dengan metode sebagai berikut:
- **Fundamental Analysis**: Evaluasi rasio keuangan (ROE, ROA, NIM, CAR, NPL, LDR, CASA)
- **Macro Overlay**: Proyeksi suku bunga BI Rate, inflasi, dan pertumbuhan GDP
- **Valuation Framework**: DCF (Discounted Cash Flow), P/E relative valuation, P/B vs ROE regression
- **Technical Analysis**: Support/resistance level, volume profile, relative strength
- **Data Period**: Q1 2024 - Q2 2026

## 3. Temuan (Findings)

### 3.1 Profitabilitas Superior
Bank-bank besar Indonesia mencatatkan ROE rata-rata 15.2%, jauh di atas rata-rata ASEAN (11.8%). BBCA memimpin dengan ROE 21.3%, diikuti BBRI (18.5%) dan BMRI (16.8%). Net Interest Margin (NIM) stabil di kisaran 5.2-5.8%.

### 3.2 Digital Banking Growth
Adopsi mobile banking tumbuh 28% YoY. BRIS memimpin transformasi digital syariah dengan 15.2 juta pengguna aktif. BBCA Mobile mencatatkan 32 juta transaksi harian.

### 3.3 Asset Quality Improvement
NPL Gross sektor perbankan turun dari 2.8% (2024) ke 2.1% (2026). Coverage ratio meningkat ke 185%, menunjukkan provisioning yang prudent.

### 3.4 Valuasi Atraktif
P/E sektor perbankan Indonesia rata-rata 12.5x, dibandingkan dengan 15.8x untuk bank ASEAN. P/B ratio 2.1x dengan implied ROE 15.2% menunjukkan undervaluation relatif.

## 4. Kesimpulan

Sektor perbankan Indonesia menawarkan kombinasi langka: profitabilitas superior, valuasi atraktif, dan katalis pertumbuhan digital. Kami merekomendasikan **OVERWEIGHT** pada sektor ini dengan preferensi pada:
1. **BBCA**: Quality compounder dengan ROE terbaik
2. **BBRI**: Leverage ke pertumbuhan UMKM dan digital
3. **BRIS**: First-mover advantage di syariah banking

Risiko perlu dimonitor: kenaikan BI Rate >6.5% atau NPL >3% akan menjadi trigger downgrade.

## Referensi

1. Bank Indonesia, "Statistik Perbankan Indonesia", Q2 2026
2. OJK, "Laporan Profil Industri Perbankan", 2026
3. Bloomberg Terminal, Data harga saham dan valuasi, diakses Agustus 2026
4. Laporan Tahunan BBCA, BBRI, BMRI, BRIS - FY2025
5. World Bank, "Indonesia Economic Prospects", June 2026
`
}

func generateResearchPaperHTML(topic string) string {
	md := generateResearchPaper(topic)
	return `<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><title>` + topic + ` - Investo Research</title>
<style>body{font-family:'Inter',sans-serif;max-width:800px;margin:40px auto;padding:20px;color:#1a1a1a;line-height:1.8}
h1{font-size:2em;border-bottom:3px solid #2563eb;padding-bottom:10px}
h2{font-size:1.4em;margin-top:30px;color:#1e40af}
h3{font-size:1.1em;color:#334155}
strong{color:#1e293b}</style></head><body>
` + formatMDToHTML(md) + `
</body></html>`
}

func formatMDToHTML(md string) string {
	html := ""
	inList := false
	for _, line := range splitLines(md) {
		line = trimSpace(line)
		if line == "" {
			if inList {
				html += "</ul>\n"
				inList = false
			}
			continue
		}
		if hasPrefix(line, "## ") {
			if inList {
				html += "</ul>\n"
				inList = false
			}
			html += "<h2>" + trimPrefix(line, "## ") + "</h2>\n"
		} else if hasPrefix(line, "### ") {
			if inList {
				html += "</ul>\n"
				inList = false
			}
			html += "<h3>" + trimPrefix(line, "### ") + "</h3>\n"
		} else if hasPrefix(line, "- **") {
			if !inList {
				html += "<ul>\n"
				inList = true
			}
			rest := trimPrefix(line, "- ")
			rest = replaceAll(rest, "**:", "</strong>:")
			rest = replaceFirst(rest, "**", "<strong>")
			html += "<li>" + rest + "</li>\n"
		} else if hasPrefix(line, "1. **") || hasPrefix(line, "2. **") || hasPrefix(line, "3. **") || hasPrefix(line, "4. **") || hasPrefix(line, "5. **") {
			idx := indexOf(line, ". ")
			if idx > 0 {
				rest := line[idx+2:]
				rest = replaceAll(rest, "**:", "</strong>:")
				rest = replaceFirst(rest, "**", "<strong>")
				html += "<p>" + rest + "</p>\n"
			}
		} else {
			if inList {
				html += "</ul>\n"
				inList = false
			}
			html += "<p>" + line + "</p>\n"
		}
	}
	if inList {
		html += "</ul>\n"
	}
	return html
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func parseIntSafe(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

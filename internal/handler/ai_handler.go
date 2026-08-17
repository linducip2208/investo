package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"
)

type AIHandler struct {
	AIService     *service.AIService
	AIAnalysis    *service.AIAnalysisService
	AIInteractive *service.AIInteractiveService
	AITools       *service.AIToolsService
	AIData        *service.AIDataService
	ContentSvc    *service.AIContentService
	Personalized  *service.AIPersonalizedService
	MultiAgent    *service.MultiAgentService
	SignalSvc     *service.AISignalService
	ForecastSvc   *service.AIForecastService
	StockRepo     *repository.StockRepository
	PortfolioRepo *repository.PortfolioRepository
	Templates     *template.Template

	BEIPipeline       *service.BEIAgentPipeline
	ForexPipeline     *service.ForexAgentPipeline
	SignalManager     *service.SignalManager
	SignalDistributor *service.SignalDistributor
	ComplianceSvc     *service.ComplianceService
	MCPConnector      *service.MCPConnector
	SettingRepo       *repository.SettingRepository
	AIUsageSvc        *service.AIUsageService
	SimExchange       *service.SimulatedExchangeService
	MandateRepo       *repository.AgentMandateRepository
	ApprovalRepo      *repository.ApprovalRepository
}

// ── AI Tools Pages ──

func (h *AIHandler) ReportGeneratorPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title":      "AI Report Generator - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai",
	}
	h.Templates.ExecuteTemplate(w, "ai/report-generator.html", data)
}

func (h *AIHandler) SearchPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title":      "AI Smart Search - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai",
	}
	h.Templates.ExecuteTemplate(w, "ai/search.html", data)
}

func (h *AIHandler) SocialPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title":      "AI Social Post Generator - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai",
	}
	h.Templates.ExecuteTemplate(w, "ai/social.html", data)
}

// ── AI Tools JSON APIs ──

func (h *AIHandler) GenerateReportJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid portfolio ID"})
		return
	}

	html, err := h.AITools.GenerateReport(portfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]string{"html": html})
}

func (h *AIHandler) AlertMessageJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code      string  `json:"code"`
		Price     float64 `json:"price"`
		Condition string  `json:"condition"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Code == "" {
		writeJSONSimple(w, map[string]string{"error": "stock code required"})
		return
	}

	message := h.AITools.GenerateAlertMessage(req.Code, req.Price, req.Condition)
	writeJSONSimple(w, map[string]string{"message": message})
}

func (h *AIHandler) SocialPostJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code     string `json:"code"`
		Theme    string `json:"theme"`
		Platform string `json:"platform"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Code == "" {
		writeJSONSimple(w, map[string]string{"error": "stock code required"})
		return
	}
	if req.Theme == "" {
		req.Theme = "analysis"
	}
	if req.Platform == "" {
		req.Platform = "twitter"
	}

	post := h.AITools.GenerateSocialPost(req.Code, req.Theme, req.Platform)
	charCount := len([]rune(post))

	writeJSONSimple(w, map[string]interface{}{
		"post":       post,
		"char_count": charCount,
		"platform":   req.Platform,
	})
}

func (h *AIHandler) SemanticSearchJSON(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSONSimple(w, map[string]string{"error": "query parameter 'q' required"})
		return
	}

	results, err := h.AITools.SemanticSearch(query)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	if results == nil {
		results = []service.SemanticSearchResult{}
	}

	writeJSONSimple(w, map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

func (h *AIHandler) ImputeFundamentalsJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONSimple(w, map[string]string{"error": "stock code required"})
		return
	}

	result, err := h.AIData.ImputeMissingFundamentals(code)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, result)
}

func (h *AIHandler) AnomalyExplanationJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONSimple(w, map[string]string{"error": "stock code required"})
		return
	}

	anomalyType := r.URL.Query().Get("type")
	if anomalyType == "" {
		anomalyType = "price_spike"
	}

	result, err := h.AIData.ExplainAnomaly(code, anomalyType)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, result)
}

// ── AI Settings (BYOK Multi-Provider) ──

func (h *AIHandler) AISettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	providers := h.AIService.LoadProviderConfigs()

	data := map[string]interface{}{
		"Title":      "Pengaturan AI - Investo",
		"User":       user,
		"ActivePage": "pengaturan-ai",
		"Providers":  providers,
	}
	h.Templates.ExecuteTemplate(w, "settings/ai.html", data)
}

func (h *AIHandler) SaveAISettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
		BaseURL  string `json:"base_url"`
		Active   bool   `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	if req.Provider == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "provider is required"})
		return
	}

	if err := h.AIService.SaveProviderConfig(req.Provider, req.APIKey, req.Model, req.BaseURL, req.Active); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "message": "Pengaturan AI berhasil disimpan"})
}

func (h *AIHandler) AIProvidersJSON(w http.ResponseWriter, r *http.Request) {
	providers := h.AIService.ListProviders()
	advanced := map[string]string{}
	if h.SettingRepo != nil {
		for _, k := range []string{"temperature", "deep_model", "quick_model", "debate_rounds", "auto_execute"} {
			if v, err := h.SettingRepo.Get("ai_" + k); err == nil {
				advanced[k] = v
			}
		}
	}
	writeJSONSimple(w, map[string]interface{}{"providers": providers, "advanced": advanced})
}

func (h *AIHandler) SaveAdvancedSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Temperature  string `json:"temperature"`
		DeepModel    string `json:"deep_model"`
		QuickModel   string `json:"quick_model"`
		DebateRounds string `json:"debate_rounds"`
		AutoExecute  string `json:"auto_execute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if h.SettingRepo == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "settings repo not available"})
		return
	}

	_ = h.SettingRepo.Set("ai_temperature", req.Temperature)
	_ = h.SettingRepo.Set("ai_deep_model", req.DeepModel)
	_ = h.SettingRepo.Set("ai_quick_model", req.QuickModel)
	_ = h.SettingRepo.Set("ai_debate_rounds", req.DebateRounds)
	_ = h.SettingRepo.Set("ai_auto_execute", req.AutoExecute)

	writeJSONSimple(w, map[string]interface{}{"success": true, "message": "Pengaturan lanjutan AI berhasil disimpan"})
}

func (h *AIHandler) MandatesJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}
	mandates, err := h.MandateRepo.FindByUserID(user.ID)
	if err != nil {
		mandates = []model.AgentMandate{}
	}
	writeJSONSimple(w, map[string]interface{}{"mandates": mandates})
}

func (h *AIHandler) SaveMandate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Name         string   `json:"name"`
		PersonaStyle string   `json:"persona_style"`
		DebateRounds int      `json:"debate_rounds"`
		RiskProfile  string   `json:"risk_profile"`
		MaxPosition  float64  `json:"max_position"`
		Tickers      []string `json:"tickers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}
	if req.Name == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "name is required"})
		return
	}
	if req.DebateRounds < 1 {
		req.DebateRounds = 1
	}
	if req.DebateRounds > 3 {
		req.DebateRounds = 3
	}
	if req.MaxPosition <= 0 {
		req.MaxPosition = 15
	}
	if req.PersonaStyle == "" {
		req.PersonaStyle = "balanced"
	}
	if req.RiskProfile == "" {
		req.RiskProfile = "moderate"
	}

	tickersJSON, _ := json.Marshal(req.Tickers)
	m := &model.AgentMandate{
		UserID:       user.ID,
		Name:         req.Name,
		PersonaStyle: req.PersonaStyle,
		DebateRounds: req.DebateRounds,
		RiskProfile:  req.RiskProfile,
		MaxPosition:  req.MaxPosition,
		TickersJSON:  string(tickersJSON),
	}
	id, err := h.MandateRepo.Create(m)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "id": id})
}

func (h *AIHandler) DeleteMandate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}
	if err := h.MandateRepo.Delete(req.ID); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true})
}

func (h *AIHandler) RunMandate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}

	mandate, err := h.MandateRepo.FindByID(req.ID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "mandate not found"})
		return
	}

	var tickers []string
	_ = json.Unmarshal([]byte(mandate.TickersJSON), &tickers)
	if len(tickers) == 0 {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "mandate has no tickers"})
		return
	}

	var decisions []*service.MultiAgentDecision
	for _, code := range tickers {
		code = strings.TrimSpace(strings.ToUpper(code))
		if code == "" {
			continue
		}
		d, err := h.MultiAgent.RunAnalysisWithPersona(code, mandate.PersonaStyle)
		if err != nil {
			continue
		}
		if h.SimExchange != nil {
			_, _ = h.SimExchange.ExecuteDecision(d, user.ID)
		}
		decisions = append(decisions, d)
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "mandate": mandate.Name, "decisions": decisions})
}

func (h *AIHandler) ApprovalsJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil || (user.Role != "admin" && user.Role != "manager") {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}
	reqs, err := h.ApprovalRepo.FindPending()
	if err != nil {
		reqs = []model.ApprovalRequest{}
	}
	writeJSONSimple(w, map[string]interface{}{"approvals": reqs})
}

func (h *AIHandler) ReviewApproval(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil || (user.Role != "admin" && user.Role != "manager") {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		ID     int64  `json:"id"`
		Action string `json:"action"` // "approve" or "reject"
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}

	status := "approved"
	if req.Action == "reject" {
		status = "rejected"
	} else if req.Action != "approve" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "action must be approve or reject"})
		return
	}

	if err := h.ApprovalRepo.UpdateStatus(req.ID, status, user.ID, req.Reason); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "status": status})
}

func (h *AIHandler) TestAIProvider(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Provider string `json:"provider"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	if req.Provider == "" || req.APIKey == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "provider and api_key are required"})
		return
	}

	response, err := h.AIService.TestConnection(req.Provider, req.BaseURL, req.APIKey, req.Model)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "response": response})
}

func (h *AIHandler) QuickAsk(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	if req.Message == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "message is required"})
		return
	}

	systemPrompt := "You are Investo AI, a financial analyst assistant for the Indonesian stock market and forex. Provide concise, data-driven analysis in Bahasa Indonesia. Be helpful, accurate, and always remind users that this is not financial advice."
	response, err := h.AIService.QuickChat(systemPrompt, req.Message)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "response": response})
}

// ── AI Analysis Pages ──

func (h *AIHandler) ComparePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title":      "AI Bandingkan Saham - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai-compare",
	}
	h.Templates.ExecuteTemplate(w, "ai/compare.html", data)
}

func (h *AIHandler) CoachPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title":      "AI Trading Coach - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai-coach",
	}
	h.Templates.ExecuteTemplate(w, "ai/coach.html", data)
}

func (h *AIHandler) ThesisBuilderPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title":      "AI Thesis Builder - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai-thesis",
	}
	h.Templates.ExecuteTemplate(w, "ai/thesis.html", data)
}

// ── AI Content Generation ──

func (h *AIHandler) DailyBriefingPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	now := time.Now().Format("02 January 2006")
	briefing, err := h.ContentSvc.GenerateDailyBriefing()
	if err != nil {
		briefing = "<p class=\"text-red-400\">Gagal menghasilkan briefing: " + err.Error() + "</p>"
	}

	data := map[string]interface{}{
		"Title":      "AI Market Briefing - " + now + " - Investo",
		"User":       user,
		"Briefing":   briefing,
		"Date":       now,
		"ActivePage": "ai-briefing",
	}
	h.Templates.ExecuteTemplate(w, "ai/briefing.html", data)
}

func (h *AIHandler) StockReportPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	report, err := h.ContentSvc.GenerateStockReport(code)
	if err != nil {
		report = "<p class=\"text-red-400\">Gagal menghasilkan laporan: " + err.Error() + "</p>"
	}

	data := map[string]interface{}{
		"Title":      "AI Stock Report - " + stock.Code + " - Investo",
		"User":       user,
		"Stock":      stock,
		"Report":     report,
		"Code":       stock.Code,
		"ActivePage": "ai-report",
	}
	h.Templates.ExecuteTemplate(w, "ai/report.html", data)
}

func (h *AIHandler) EarningsSummaryJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "2024"
	}

	summary, err := h.ContentSvc.SummarizeEarnings(code, period)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]string{"summary": summary, "code": code, "period": period})
}

func (h *AIHandler) NewsSummaryJSON(w http.ResponseWriter, r *http.Request) {
	var body struct {
		NewsIDs []int64 `json:"news_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid request body"})
		return
	}

	if len(body.NewsIDs) == 0 {
		writeJSONSimple(w, map[string]string{"error": "news_ids is required"})
		return
	}

	summary, err := h.ContentSvc.SummarizeNews(body.NewsIDs)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]string{"summary": summary})
}

func (h *AIHandler) IPOAIAnalysisJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	analysis, err := h.ContentSvc.AnalyzeIPO(code)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]string{"analysis": analysis, "code": code})
}

// ── AI Personalized ──

func (h *AIHandler) PortfolioAdviceJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	portfolioID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid portfolio id"})
		return
	}

	advice, err := h.Personalized.GeneratePortfolioAdvice(portfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{
		"advice":       advice,
		"portfolio_id": portfolioID,
	})
}

func (h *AIHandler) RiskTestPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":      "Risk Assessment Test - Investo",
		"User":       user,
		"ActivePage": "risk-test",
	}
	h.Templates.ExecuteTemplate(w, "ai/risk-test.html", data)
}

func (h *AIHandler) RiskTestJSON(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid form data"})
		return
	}

	answers := make(map[string]string)
	for i := 1; i <= 10; i++ {
		key := "q" + strconv.Itoa(i)
		if val, ok := r.Form[key]; ok && len(val) > 0 {
			answers[key] = val[0]
		}
	}

	if len(answers) < 10 {
		writeJSONSimple(w, map[string]string{"error": "semua 10 pertanyaan harus dijawab"})
		return
	}

	result, err := h.Personalized.GenerateRiskProfile(answers)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, result)
}

func (h *AIHandler) LearningPathJSON(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid form data"})
		return
	}

	topic := strings.TrimSpace(r.FormValue("topic"))
	level := strings.TrimSpace(r.FormValue("level"))

	if topic == "" || level == "" {
		writeJSONSimple(w, map[string]string{"error": "topic and level are required"})
		return
	}

	path, err := h.Personalized.GenerateLearningPath(topic, level)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]string{
		"learning_path": path,
		"topic":         topic,
		"level":         level,
	})
}

func (h *AIHandler) JournalAnalysisJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]string{"error": "unauthorized"})
		return
	}

	analysis, err := h.Personalized.AnalyzeTradeJournal(user.ID)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]string{"analysis": analysis})
}

// ── AI Analysis JSON APIs ──

func (h *AIHandler) CompareStocksJSON(w http.ResponseWriter, r *http.Request) {
	codeA := chi.URLParam(r, "codeA")
	codeB := chi.URLParam(r, "codeB")

	analysis, err := h.AIAnalysis.CompareStocks(codeA, codeB)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "analysis": analysis})
}

func (h *AIHandler) SectorThesisJSON(w http.ResponseWriter, r *http.Request) {
	sectorIDStr := chi.URLParam(r, "sectorID")
	sectorID, err := strconv.ParseInt(sectorIDStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid sector ID"})
		return
	}

	thesis, err := h.AIAnalysis.GenerateSectorThesis(sectorID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "thesis": thesis})
}

func (h *AIHandler) EventImpactJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Event       string `json:"event"`
		PortfolioID int64  `json:"portfolio_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	analysis, err := h.AIAnalysis.AnalyzeEventImpact(req.Event, req.PortfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "analysis": analysis})
}

func (h *AIHandler) FraudDetectionJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.AIAnalysis.DetectFinancialFraud(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

func (h *AIHandler) DividendSustainabilityJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.AIAnalysis.AnalyzeDividendSustainability(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

func (h *AIHandler) CoachingChatJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string                  `json:"message"`
		History []service.AIChatMessage `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	response, err := h.AIInteractive.CoachingChat(0, req.Message, req.History)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "response": response})
}

func (h *AIHandler) ScenarioSimulationJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario    string `json:"scenario"`
		PortfolioID int64  `json:"portfolio_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	result, err := h.AIInteractive.SimulateScenario(req.Scenario, req.PortfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

func (h *AIHandler) ThesisBuilderJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code      string `json:"code"`
		UserInput string `json:"user_input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	result, err := h.AIInteractive.BuildInvestmentThesis(req.Code, req.UserInput)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

func (h *AIHandler) JargonTranslatorJSON(w http.ResponseWriter, r *http.Request) {
	term := r.URL.Query().Get("term")
	if term == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "term parameter required"})
		return
	}

	result, err := h.AIInteractive.TranslateJargon(term)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Multi-Agent Trading System ──

func (h *AIHandler) AgentAnalysisPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		stocks = []model.Stock{}
	}

	data := map[string]interface{}{
		"Title":      "Multi-Agent Trading System - Investo",
		"User":       user,
		"Stocks":     stocks,
		"ActivePage": "ai-agents",
	}
	h.Templates.ExecuteTemplate(w, "ai/agents.html", data)
}

func (h *AIHandler) RunAgentAnalysis(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))
	if req.Code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code is required"})
		return
	}

	decision, err := h.MultiAgent.RunAnalysis(req.Code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	if h.SimExchange != nil {
		_, _ = h.SimExchange.ExecuteDecision(decision, user.ID)
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "decision": decision})
}

func (h *AIHandler) AgentAnalysisJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	decision, err := h.MultiAgent.RunAnalysis(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "decision": decision})
}

func (h *AIHandler) AgentHistoryJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	decisions, err := h.MultiAgent.DecisionRepo.FindByTicker(code, limit)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	if decisions == nil {
		decisions = []model.AgentDecision{}
	}

	writeJSONSimple(w, map[string]interface{}{
		"success":   true,
		"ticker":    code,
		"decisions": decisions,
		"count":     len(decisions),
	})
}

// ─────────── Voice Trade ───────────

func (h *AIHandler) VoiceTradePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "Voice-to-Trade - Investo",
		"User":       user,
		"ActivePage": "ai-voice",
	}
	h.Templates.ExecuteTemplate(w, "ai/voice-trade.html", data)
}

func (h *AIHandler) VoiceParseJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transcript string `json:"transcript"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if req.Transcript == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "transcript required"})
		return
	}

	voiceSvc := service.NewAIVoiceService(h.AIService)
	result, err := voiceSvc.ParseVoiceCommand(req.Transcript)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ─────────── Daily Standup ───────────

var standupSvc *service.AIStandupService

func (h *AIHandler) SetStandupService(svc *service.AIStandupService) {
	standupSvc = svc
}

func (h *AIHandler) StandupPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	briefing, data, err := standupSvc.GenerateStandup(user.ID)
	if err != nil {
		briefing = "<p class=\"text-red-400\">Gagal menghasilkan standup: " + err.Error() + "</p>"
		data = &service.StandupData{Date: time.Now().Format("02 January 2006")}
	}
	tplData := map[string]interface{}{
		"Title":      "AI Daily Standup - Investo",
		"User":       user,
		"Briefing":   briefing,
		"Date":       data.Date,
		"ActivePage": "ai-standup",
	}
	h.Templates.ExecuteTemplate(w, "ai/daily-standup.html", tplData)
}

func (h *AIHandler) StandupJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}
	briefing, data, err := standupSvc.GenerateStandup(user.ID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "briefing": briefing, "data": data})
}

// ─────────── Tax Optimizer ───────────

var taxSvc *service.AITaxService

func (h *AIHandler) SetTaxServiceVar(svc *service.AITaxService) {
	taxSvc = svc
}

func (h *AIHandler) TaxOptimizerPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "AI Tax Optimizer - Investo",
		"User":       user,
		"ActivePage": "ai-tax",
	}
	h.Templates.ExecuteTemplate(w, "ai/tax-optimizer.html", data)
}

func (h *AIHandler) TaxOptimizerJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid portfolio ID"})
		return
	}
	result, err := taxSvc.OptimizeTax(portfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ─────────── Comic Generator ───────────

var comicSvc *service.AIComicService

func (h *AIHandler) SetComicService(svc *service.AIComicService) {
	comicSvc = svc
}

func (h *AIHandler) ComicPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	comic, err := comicSvc.GenerateComic()
	if err != nil {
		comic = &service.ComicResult{Title: "Error", Date: time.Now().Format("02 January 2006")}
	}
	data := map[string]interface{}{
		"Title":      "AI Market Comic - Investo",
		"User":       user,
		"Comic":      comic,
		"ActivePage": "ai-comic",
	}
	h.Templates.ExecuteTemplate(w, "ai/comic.html", data)
}

func (h *AIHandler) ComicJSON(w http.ResponseWriter, r *http.Request) {
	comic, err := comicSvc.GenerateComic()
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "comic": comic})
}

func (h *AIHandler) HaikuJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid portfolio ID"})
		return
	}
	haiku, err := comicSvc.GenerateHaiku(portfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "haiku": haiku})
}

// ─────────── Stock Story ───────────

var storySvc *service.AIStoryService

func (h *AIHandler) SetStoryService(svc *service.AIStoryService) {
	storySvc = svc
}

func (h *AIHandler) StockStoryPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	code := chi.URLParam(r, "code")
	story, err := storySvc.GenerateStory(code)
	if err != nil {
		story = &service.StockStory{Code: code, Title: "Error: " + err.Error()}
	}
	data := map[string]interface{}{
		"Title":      "AI Stock Story - " + code + " - Investo",
		"User":       user,
		"Story":      story,
		"Code":       code,
		"ActivePage": "ai-story",
	}
	h.Templates.ExecuteTemplate(w, "ai/stock-story.html", data)
}

func (h *AIHandler) StockStoryJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	story, err := storySvc.GenerateStory(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "story": story})
}

// ─────────── Compliance Checker ───────────

var complianceSvc *service.AIComplianceService

func (h *AIHandler) SetComplianceService(svc *service.AIComplianceService) {
	complianceSvc = svc
}

func (h *AIHandler) CompliancePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "AI Compliance Checker - Investo",
		"User":       user,
		"ActivePage": "ai-compliance",
	}
	h.Templates.ExecuteTemplate(w, "ai/compliance.html", data)
}

func (h *AIHandler) ComplianceJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid portfolio ID"})
		return
	}
	report, err := complianceSvc.CheckCompliance(portfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "report": report})
}

// ─────────── Client Report ───────────

var clientReportSvc *service.AIClientReportService

func (h *AIHandler) SetClientReportService(svc *service.AIClientReportService) {
	clientReportSvc = svc
}

func (h *AIHandler) ClientReportPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "AI Client Report - Investo",
		"User":       user,
		"ActivePage": "ai-client-report",
	}
	h.Templates.ExecuteTemplate(w, "ai/client-report.html", data)
}

func (h *AIHandler) ClientReportJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid portfolio ID"})
		return
	}
	html, data, err := clientReportSvc.GenerateClientReport(portfolioID)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "html": html, "data": data})
}

// ─────────── Webhook Intelligence ───────────

var webhookIntelSvc *service.AIWebhookIntelService

func (h *AIHandler) SetWebhookIntelService(svc *service.AIWebhookIntelService) {
	webhookIntelSvc = svc
}

func (h *AIHandler) WebhookIntelPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	configs := webhookIntelSvc.GetDefaultConfigs()
	data := map[string]interface{}{
		"Title":      "AI Webhook Intelligence - Investo",
		"User":       user,
		"Configs":    configs,
		"ActivePage": "ai-webhooks",
	}
	h.Templates.ExecuteTemplate(w, "ai/webhook-intel.html", data)
}

func (h *AIHandler) WebhookConfigJSON(w http.ResponseWriter, r *http.Request) {
	var req service.WebhookIntelConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	testPayload, _ := webhookIntelSvc.TestWebhook(req.URL)
	writeJSONSimple(w, map[string]interface{}{"success": true, "message": "Webhook configured", "test_payload": testPayload})
}

// ─────────── Calendar Sync ───────────

var calendarSvc *service.AICalendarService

func (h *AIHandler) SetCalendarService(svc *service.AICalendarService) {
	calendarSvc = svc
}

func (h *AIHandler) CalendarSyncJSON(w http.ResponseWriter, r *http.Request) {
	events, err := calendarSvc.GenerateCalendarSync()
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	ics := calendarSvc.GenerateICS(events)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=investo-calendar.ics")
	w.Write([]byte(ics))
}

func (h *AIHandler) CalendarSyncPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	events, _ := calendarSvc.GenerateCalendarSync()
	data := map[string]interface{}{
		"Title":      "AI Calendar Sync - Investo",
		"User":       user,
		"Events":     events,
		"ActivePage": "ai-calendar",
	}
	h.Templates.ExecuteTemplate(w, "ai/calendar-sync.html", data)
}

// ─────────── Bot Format ───────────

var botFormatSvc *service.AIBotFormatService

func (h *AIHandler) SetBotFormatService(svc *service.AIBotFormatService) {
	botFormatSvc = svc
}

func (h *AIHandler) BotFormatPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "AI Bot Format - Investo",
		"User":       user,
		"ActivePage": "ai-bot-format",
	}
	h.Templates.ExecuteTemplate(w, "ai/bot-format.html", data)
}

func (h *AIHandler) BotFormatJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platform string `json:"platform"`
		Command  string `json:"command"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if req.Command == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "command required"})
		return
	}
	result, err := botFormatSvc.FormatBotMessage(req.Platform, req.Command, req.Code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Agent Streaming (SSE) ──

func (h *AIHandler) StreamAgentAnalysis(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "streaming not supported"})
		return
	}

	h.MultiAgent.StreamAnalysis(w, code)
	flusher.Flush()
}

// ── Agent Consensus ──

func (h *AIHandler) AgentConsensusJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	consensus, err := h.MultiAgent.GetConsensusVote(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "consensus": consensus})
}

// ── Agent Analysis with Persona ──

func (h *AIHandler) AgentAnalysisWithPersona(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Code     string `json:"code"`
		Persona  string `json:"persona"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))
	if req.Code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code is required"})
		return
	}

	validPersonas := map[string]bool{"buffett": true, "soros": true, "tudor_jones": true, "cathie_wood": true}
	if !validPersonas[req.Persona] {
		req.Persona = "buffett"
	}

	decision, err := h.MultiAgent.RunAnalysisWithPersona(req.Code, req.Persona)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "persona": req.Persona, "decision": decision})
}

// ── Signal Confidence ──

func (h *AIHandler) SignalConfidenceJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	result, err := h.SignalSvc.GetSignalWithConfidence(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Signal Backtest ──

func (h *AIHandler) SignalBacktestJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 365
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
	}

	result, err := h.SignalSvc.BacktestSignal(code, days)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error(), "result": result})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Market Regime ──

func (h *AIHandler) MarketRegimeJSON(w http.ResponseWriter, r *http.Request) {
	result, err := h.SignalSvc.DetectMarketRegime()
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	if result == nil {
		result = &service.RegimeResult{
			Regime:             "tidak_diketahui",
			Confidence:         0,
			RecommendedStrategy: "hold",
			Description:        "Tidak dapat mendeteksi regime pasar.",
		}
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Optimal Entry ──

func (h *AIHandler) OptimalEntryJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	result, err := h.SignalSvc.FindOptimalEntry(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Price Forecast ──

func (h *AIHandler) PriceForecastJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
	}

	result, err := h.ForecastSvc.ForecastPriceRange(code, days)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Dividend Prediction ──

func (h *AIHandler) DividendPredictJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	result, err := h.ForecastSvc.DetectDividendChange(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "result": result})
}

// ── Black Swan Scan ──

func (h *AIHandler) BlackSwanScanJSON(w http.ResponseWriter, r *http.Request) {
	result, err := h.ForecastSvc.ScanBlackSwanRisks()
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	if result == nil {
		result = []service.RiskAlert{}
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "alerts": result, "count": len(result)})
}

// ── Insider Signal Interpreter ──

func (h *AIHandler) InsiderInterpretJSON(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	analysis, err := h.SignalSvc.InterpretInsiderTransaction(code)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "analysis": analysis, "code": code})
}

func (h *AIHandler) SignalCenterPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "Signal Center - Investo",
		"User":       safeUser(user),
		"ActivePage": "signal-center",
	}
	h.Templates.ExecuteTemplate(w, "ai/signal-center.html", data)
}

func (h *AIHandler) DataSourcesPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "Data Sources - Investo",
		"User":       safeUser(user),
		"ActivePage": "data-sources",
	}
	h.Templates.ExecuteTemplate(w, "settings/data-sources.html", data)
}

func (h *AIHandler) GenerateBEISignal(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	if h.SignalManager == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "signal manager not configured"})
		return
	}

	signal, err := h.SignalManager.ProcessSignal(code, "BEI")
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "signal": signal})
}

func (h *AIHandler) GenerateForexSignal(w http.ResponseWriter, r *http.Request) {
	pair := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "pair")))
	if pair == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "forex pair required"})
		return
	}

	if h.SignalManager == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "signal manager not configured"})
		return
	}

	signal, err := h.SignalManager.ProcessSignal(pair, "FOREX")
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "signal": signal})
}

func (h *AIHandler) DistributeSignal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Signal  service.FinalSignal `json:"signal"`
		Targets []string            `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if h.SignalDistributor == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "signal distributor not configured"})
		return
	}

	result := h.SignalDistributor.DistributeSignal(req.Signal, req.Targets)
	writeJSONSimple(w, map[string]interface{}{
		"success":   result.Success,
		"channels":  result.Channels,
		"errors":    result.Errors,
		"timestamp": result.Timestamp,
	})
}

func (h *AIHandler) SaveDataSource(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Source      string `json:"source"`
		APIKey      string `json:"api_key"`
		BaseURL     string `json:"base_url"`
		AccountID   string `json:"account_id"`
		Environment string `json:"environment"`
		AccessToken string `json:"access_token"`
		PhoneID     string `json:"phone_id"`
		VerifyToken string `json:"verify_token"`
		Active      bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}

	if h.SettingRepo == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "setting repo not available"})
		return
	}

	switch req.Source {
	case "invezgo":
		h.SettingRepo.Set("datasource_invezgo_key", h.AIService.EncodeKeyWrapper(req.APIKey))
		h.SettingRepo.Set("datasource_invezgo_base_url", req.BaseURL)
		h.SettingRepo.Set("datasource_invezgo_active", boolToString(req.Active))
	case "oanda":
		h.SettingRepo.Set("datasource_oanda_key", h.AIService.EncodeKeyWrapper(req.APIKey))
		h.SettingRepo.Set("datasource_oanda_account", req.AccountID)
		h.SettingRepo.Set("datasource_oanda_environment", req.Environment)
		h.SettingRepo.Set("datasource_oanda_active", boolToString(req.Active))
	case "wa":
		h.SettingRepo.Set("datasource_wa_phone_id", req.PhoneID)
		h.SettingRepo.Set("datasource_wa_token", h.AIService.EncodeKeyWrapper(req.AccessToken))
		h.SettingRepo.Set("datasource_wa_verify_token", req.VerifyToken)
		h.SettingRepo.Set("datasource_wa_active", boolToString(req.Active))
	default:
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unknown source: " + req.Source})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "message": req.Source + " settings saved"})
}

func (h *AIHandler) TestDataSource(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Source      string `json:"source"`
		APIKey      string `json:"api_key"`
		BaseURL     string `json:"base_url"`
		AccountID   string `json:"account_id"`
		AccessToken string `json:"access_token"`
		PhoneID     string `json:"phone_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}

	switch req.Source {
	case "invezgo":
		if req.APIKey == "" {
			writeJSONSimple(w, map[string]interface{}{"success": false, "error": "API key required"})
			return
		}
		writeJSONSimple(w, map[string]interface{}{"success": true, "response": "Invezgo connection test: OK (simulated)"})
	case "oanda":
		if req.APIKey == "" || req.AccountID == "" {
			writeJSONSimple(w, map[string]interface{}{"success": false, "error": "API key and Account ID required"})
			return
		}
		writeJSONSimple(w, map[string]interface{}{"success": true, "response": "OANDA connection test: OK (simulated)"})
	case "wa":
		if req.AccessToken == "" || req.PhoneID == "" {
			writeJSONSimple(w, map[string]interface{}{"success": false, "error": "Access Token and Phone ID required"})
			return
		}
		writeJSONSimple(w, map[string]interface{}{"success": true, "response": "WA Business API connection test: OK (simulated)"})
	default:
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unknown source"})
	}
}

func (h *AIHandler) ComplianceCheckJSON(w http.ResponseWriter, r *http.Request) {
	marketType := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "marketType")))
	if marketType == "" {
		marketType = "BEI"
	}

	if h.ComplianceSvc == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "compliance service not configured"})
		return
	}

	rules, err := h.ComplianceSvc.CheckRegulatory(marketType)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{
		"success":    true,
		"market":     marketType,
		"rules":      rules,
		"count":      len(rules),
		"disclaimer": h.ComplianceSvc.AddDisclaimer(""),
	})
}

func (h *AIHandler) ValidateMCP(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(strings.ToUpper(chi.URLParam(r, "code")))
	if code == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "stock code required"})
		return
	}

	if h.MCPConnector == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "MCP connector not configured"})
		return
	}

	result, err := h.MCPConnector.ValidateSignal(code, 0)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "validation": result})
}

func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (h *AIHandler) AIUsagePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]interface{}{
		"Title":      "AI Usage Dashboard - Investo",
		"User":       safeUser(user),
		"ActivePage": "ai-usage",
	}
	h.Templates.ExecuteTemplate(w, "ai/usage.html", data)
}

func (h *AIHandler) AIUsageJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"error": "unauthorized"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
	}

	svc := h.AIUsageSvc
	if svc == nil {
		writeJSONSimple(w, map[string]interface{}{"error": "AI usage service not initialized"})
		return
	}

	stats, err := svc.GetUsageStats(days)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"error": err.Error()})
		return
	}

	recentLogs, _ := svc.GetRecentLogs(50, 0, 0)
	if recentLogs == nil {
		recentLogs = []map[string]interface{}{}
	}

	var callsToday int
	today := time.Now().Format("2006-01-02")
	for _, dl := range recentLogs {
		if ca, ok := dl["created_at"].(string); ok && strings.HasPrefix(ca, today) {
			callsToday++
		}
	}

	fastestProvider := svc.GetFastestProvider()

	writeJSONSimple(w, map[string]interface{}{
		"stats":            stats,
		"recent_logs":      recentLogs,
		"calls_today":      callsToday,
		"fastest_provider": fastestProvider,
	})
}

func (h *AIHandler) AIHealthCheckJSON(w http.ResponseWriter, r *http.Request) {
	svc := h.AIUsageSvc
	if svc == nil {
		writeJSONSimple(w, map[string]interface{}{"error": "AI usage service not initialized"})
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider != "" {
		healthy, err := svc.CheckKeyHealth(provider)
		writeJSONSimple(w, map[string]interface{}{
			"provider": provider,
			"healthy":  healthy,
			"status":   map[bool]string{true: "healthy", false: "unhealthy"}[healthy],
			"error":    map[bool]string{true: "", false: err.Error()}[healthy],
		})
		return
	}

	result := svc.GetKeyHealthAll()
	writeJSONSimple(w, result)
}

func (h *AIHandler) AIBudgetJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"error": "unauthorized"})
		return
	}

	var req struct {
		Provider     string  `json:"provider"`
		MonthlyLimit float64 `json:"monthly_limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	if req.Provider == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "provider is required"})
		return
	}

	svc := h.AIUsageSvc
	if svc == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "AI usage service not initialized"})
		return
	}

	if err := svc.SetBudget(req.Provider, req.MonthlyLimit); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "message": "Budget updated"})
}

func (h *AIHandler) AIRateResponseJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"error": "unauthorized"})
		return
	}

	var req struct {
		UsageLogID int64  `json:"usage_log_id"`
		Rating     int    `json:"rating"`
		Comment    string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	if req.UsageLogID == 0 || req.Rating < 1 || req.Rating > 5 {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "usage_log_id and rating (1-5) are required"})
		return
	}

	svc := h.AIUsageSvc
	if svc == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "AI usage service not initialized"})
		return
	}

	if err := svc.RateResponse(req.UsageLogID, user.ID, req.Rating, req.Comment); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "message": "Rating saved"})
}

func (h *AIHandler) AIReportJSON(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	month := 0
	year := 0
	if monthStr != "" {
		month, _ = strconv.Atoi(monthStr)
	}
	if yearStr != "" {
		year, _ = strconv.Atoi(yearStr)
	}

	svc := h.AIUsageSvc
	if svc == nil {
		writeJSONSimple(w, map[string]interface{}{"error": "AI usage service not initialized"})
		return
	}

	report, err := svc.GenerateSpendReport(month, year)
	if err != nil {
		writeJSONSimple(w, map[string]interface{}{"error": err.Error()})
		return
	}

	writeJSONSimple(w, report)
}

func (h *AIHandler) SaveAISettingsEnhanced(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	var req struct {
		Provider  string `json:"provider"`
		APIKey    string `json:"api_key"`
		Model     string `json:"model"`
		BaseURL   string `json:"base_url"`
		Active    bool   `json:"active"`
		Budget    string `json:"budget"`
		MultiKeys []struct {
			Num int    `json:"num"`
			Key string `json:"key"`
		} `json:"multi_keys"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	if req.Provider == "" {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": "provider is required"})
		return
	}

	if err := h.AIService.SaveProviderConfig(req.Provider, req.APIKey, req.Model, req.BaseURL, req.Active); err != nil {
		writeJSONSimple(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	for _, mk := range req.MultiKeys {
		if mk.Key != "" {
			keyName := fmt.Sprintf("ai_key_%s_%d", req.Provider, mk.Num)
			if h.SettingRepo != nil {
				usageSvc := h.AIUsageSvc
				if usageSvc != nil {
					h.SettingRepo.Set(keyName, usageSvc.EncryptKey(mk.Key))
				} else {
					h.SettingRepo.Set(keyName, h.AIService.EncodeKeyWrapper(mk.Key))
				}
			}
		}
	}

	writeJSONSimple(w, map[string]interface{}{"success": true, "message": "Pengaturan AI berhasil disimpan"})
}

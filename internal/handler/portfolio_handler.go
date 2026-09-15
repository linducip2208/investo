package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type PortfolioHandler struct {
	PortfolioRepo      *repository.PortfolioRepository
	PortfolioItemRepo  *repository.PortfolioItemRepository
	Templates          *template.Template
	PortfolioAnalytics *service.PortfolioAnalytics
	DRIPService        *service.DRIPService
	RiskAnalyzer       *service.RiskAnalyzer
	TradeJournalRepo   *repository.TradeJournalRepository
}

func (h *PortfolioHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	portfolios, err := h.PortfolioRepo.FindByUserID(user.ID)
	if err != nil {
		http.Error(w, "Gagal memuat portfolio", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":        "Portfolio Saya - Investo",
		"Portfolios":   portfolios,
		"User":         user,
		"UserInitials": string([]rune(user.Name)[:1]),
		"UserName":     user.Name,
		"UserEmail":    user.Email,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/list.html", data)
}

func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	isDefault := r.FormValue("is_default") == "1"

	if name == "" {
		session, _ := r.Context().Value("session").(interface{ Save(*http.Request, http.ResponseWriter) error })
		_ = session
		http.Redirect(w, r, "/dashboard/portfolios", http.StatusSeeOther)
		return
	}

	portfolio := &model.Portfolio{
		UserID:      user.ID,
		Name:        name,
		Description: description,
		IsDefault:   isDefault,
	}

	_, err := h.PortfolioRepo.Create(portfolio)
	if err != nil {
		http.Error(w, "Gagal membuat portfolio", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/portfolios", http.StatusSeeOther)
}

// CreateJSON is the JSON API for creating a portfolio (used by frontend fetch).
func (h *PortfolioHandler) CreateJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsDefault   bool   `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}
	if req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	p := &model.Portfolio{
		UserID:      user.ID,
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   req.IsDefault,
	}
	id, err := h.PortfolioRepo.Create(p)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": id})
}

func (h *PortfolioHandler) Detail(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil || portfolio.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	items, _ := h.PortfolioItemRepo.GetWithStock(portfolio.ID)

	data := map[string]interface{}{
		"Title":     portfolio.Name + " - Portfolio - Investo",
		"Portfolio": portfolio,
		"Items":     items,
		"User":      user,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/detail.html", data)
}

func (h *PortfolioHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	portfolioID := int64(0)
	if pid := r.FormValue("portfolio_id"); pid != "" {
		portfolioID, _ = strconv.ParseInt(pid, 10, 64)
	}
	stockID, _ := strconv.ParseInt(r.FormValue("stock_id"), 10, 64)
	itemType := r.FormValue("type")
	quantity := parseFloat(r.FormValue("quantity"))
	avgPrice := parseFloat(r.FormValue("avg_price"))
	notes := r.FormValue("notes")

	if portfolioID == 0 || stockID == 0 || quantity <= 0 || itemType == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(portfolioID)
	if err != nil || portfolio.UserID != user.ID {
		http.Error(w, "Portfolio tidak ditemukan", http.StatusNotFound)
		return
	}

	if itemType == "" {
		itemType = "buy"
	}

	item := &model.PortfolioItem{
		PortfolioID: portfolioID,
		StockID:     stockID,
		Type:        itemType,
		Quantity:    quantity,
		AvgPrice:    avgPrice,
		Notes:       notes,
	}

	_, err = h.PortfolioItemRepo.Create(item)
	if err != nil {
		http.Error(w, "Gagal menambahkan item", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/portfolios/"+strconv.FormatInt(portfolioID, 10), http.StatusSeeOther)
}

func (h *PortfolioHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.ParseInt(r.FormValue("item_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	portfolioID, _ := strconv.ParseInt(r.FormValue("portfolio_id"), 10, 64)

	portfolio, err := h.PortfolioRepo.FindByID(portfolioID)
	if err != nil || portfolio.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := h.PortfolioItemRepo.DeleteForPortfolio(itemID, portfolioID); err != nil {
		http.Error(w, "Item tidak ditemukan", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/dashboard/portfolios/"+strconv.FormatInt(portfolioID, 10), http.StatusSeeOther)
}

func (h *PortfolioHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	itemID, _ := strconv.ParseInt(r.FormValue("item_id"), 10, 64)
	portfolioID, _ := strconv.ParseInt(r.FormValue("portfolio_id"), 10, 64)
	quantity := parseFloat(r.FormValue("quantity"))
	avgPrice := parseFloat(r.FormValue("avg_price"))
	notes := r.FormValue("notes")

	portfolio, err := h.PortfolioRepo.FindByID(portfolioID)
	if err != nil || portfolio.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	item := &model.PortfolioItem{
		ID:      itemID,
		PortfolioID: portfolioID,
		Quantity: quantity,
		AvgPrice: avgPrice,
		Notes:    notes,
	}

	if err := h.PortfolioItemRepo.UpdateForPortfolio(item); err != nil {
		http.Error(w, "Item tidak ditemukan", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/dashboard/portfolios/"+strconv.FormatInt(portfolioID, 10), http.StatusSeeOther)
}

func (h *PortfolioHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil || portfolio.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	perf, _ := h.PortfolioAnalytics.CalcPerformance(id)
	frontier, _ := h.PortfolioAnalytics.CalcEfficientFrontier(id)
	attribution, _ := h.PortfolioAnalytics.CalcAttribution(id)

	data := map[string]interface{}{
		"Title":       portfolio.Name + " - Analytics - Investo",
		"Portfolio":   portfolio,
		"Performance": perf,
		"Frontier":    frontier,
		"Attribution": attribution,
		"User":        user,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/analytics.html", data)
}

func (h *PortfolioHandler) PerformanceJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	perf, err := h.PortfolioAnalytics.CalcPerformance(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perf)
}

func (h *PortfolioHandler) FrontierJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	frontier, err := h.PortfolioAnalytics.CalcEfficientFrontier(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(frontier)
}

func (h *PortfolioHandler) AttributionJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	att, err := h.PortfolioAnalytics.CalcAttribution(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(att)
}

func (h *PortfolioHandler) ToolsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	portfolios, err := h.PortfolioRepo.FindByUserID(user.ID)
	if err != nil {
		http.Error(w, "Gagal memuat portfolio", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":      "Portfolio Tools - Investo",
		"Portfolios": portfolios,
		"User":       user,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/tools.html", data)
}

func (h *PortfolioHandler) MonteCarloJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid portfolio ID"})
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid form data"})
		return
	}

	simulations := 1000
	if s := r.FormValue("simulations"); s != "" {
		simulations, _ = strconv.Atoi(s)
	}
	years := 1
	if y := r.FormValue("years"); y != "" {
		years, _ = strconv.Atoi(y)
	}

	result, err := h.PortfolioAnalytics.RunMonteCarlo(id, simulations, years)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PortfolioHandler) StressTestJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid portfolio ID"})
		return
	}

	scenario := r.URL.Query().Get("scenario")
	if scenario == "" {
		scenario = "2008_crash"
	}

	result, err := h.PortfolioAnalytics.RunStressTest(id, scenario)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PortfolioHandler) TaxLossJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid portfolio ID"})
		return
	}

	result, err := h.PortfolioAnalytics.FindTaxLossOpportunities(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PortfolioHandler) DRIPJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Stock code is required"})
		return
	}

	investment := 10000000.0
	if inv := r.URL.Query().Get("investment"); inv != "" {
		investment, _ = strconv.ParseFloat(inv, 64)
	}
	years := 10
	if y := r.URL.Query().Get("years"); y != "" {
		years, _ = strconv.Atoi(y)
	}
	reinvest := true
	if reinv := r.URL.Query().Get("reinvest"); reinv == "false" || reinv == "0" {
		reinvest = false
	}

	stockRepo := h.PortfolioAnalytics.StockRepo
	stockPriceRepo := h.PortfolioAnalytics.StockPriceRepo
	fundamentalRepo := h.PortfolioAnalytics.StockFundamentalRepo

	stock, err := stockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Saham tidak ditemukan"})
		return
	}

	dripService := &service.DRIPService{
		StockFundamentalRepo: fundamentalRepo,
		StockPriceRepo:       stockPriceRepo,
		StockRepo:            stockRepo,
	}

	result, err := dripService.Calculate(stock.ID, investment, years, reinvest)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PortfolioHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Gagal membaca file", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("csv_file")
	if err != nil {
		http.Error(w, "File CSV tidak ditemukan", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Gagal parsing CSV", http.StatusBadRequest)
		return
	}

	if len(records) < 2 {
		http.Error(w, "CSV kosong atau hanya berisi header", http.StatusBadRequest)
		return
	}

	dateStr := time.Now().Format("02 Jan 2006 15:04")
	portfolio := &model.Portfolio{
		UserID:      user.ID,
		Name:        fmt.Sprintf("Import %s", dateStr),
		Description: fmt.Sprintf("Diimpor dari CSV pada %s", dateStr),
		IsDefault:   false,
	}

	portfolioID, err := h.PortfolioRepo.Create(portfolio)
	if err != nil {
		http.Error(w, "Gagal membuat portfolio", http.StatusInternalServerError)
		return
	}

	imported := 0
	skipped := 0

	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 3 {
			skipped++
			continue
		}

		code := strings.TrimSpace(strings.ToUpper(row[0]))
		quantity, err := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		if err != nil || quantity <= 0 {
			skipped++
			continue
		}
		avgPrice, err := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		if err != nil || avgPrice <= 0 {
			skipped++
			continue
		}

		stock, err := h.PortfolioAnalytics.StockRepo.FindByCode(code)
		if err != nil {
			skipped++
			continue
		}

		notes := ""
		if len(row) >= 4 {
			notes = strings.TrimSpace(row[3])
		}

		item := &model.PortfolioItem{
			PortfolioID: portfolioID,
			StockID:     stock.ID,
			Type:        "buy",
			Quantity:    quantity,
			AvgPrice:    avgPrice,
			Notes:       notes,
		}

		if _, err := h.PortfolioItemRepo.Create(item); err != nil {
			skipped++
			continue
		}
		imported++
	}

	_ = skipped
	http.Redirect(w, r, fmt.Sprintf("/dashboard/portfolios/%d", portfolioID), http.StatusSeeOther)
}

func (h *PortfolioHandler) ProjectionJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Stock code is required"})
		return
	}

	growthRate := 10.0
	if g := r.URL.Query().Get("growth"); g != "" {
		growthRate, _ = strconv.ParseFloat(g, 64)
	}
	discountRate := 8.0
	if d := r.URL.Query().Get("discount"); d != "" {
		discountRate, _ = strconv.ParseFloat(d, 64)
	}
	years := 5
	if y := r.URL.Query().Get("years"); y != "" {
		years, _ = strconv.Atoi(y)
	}

	stockRepo := h.PortfolioAnalytics.StockRepo
	stock, err := stockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Saham tidak ditemukan"})
		return
	}

	result, err := h.PortfolioAnalytics.ProjectStock(stock.ID, growthRate, discountRate, years)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PortfolioHandler) RiskAnalysis(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil || portfolio.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	risk, _ := h.RiskAnalyzer.AnalyzePortfolio(id)

	data := map[string]interface{}{
		"Title":     portfolio.Name + " - Risk Analysis - Investo",
		"Portfolio": portfolio,
		"Risk":      risk,
		"User":      user,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/risk.html", data)
}

func (h *PortfolioHandler) RiskJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid portfolio ID"})
		return
	}

	risk, err := h.RiskAnalyzer.AnalyzePortfolio(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(risk)
}

func (h *PortfolioHandler) JournalList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	journals, err := h.TradeJournalRepo.FindByUserID(user.ID)
	if err != nil {
		http.Error(w, "Gagal memuat trade journal", http.StatusInternalServerError)
		return
	}

	stats, _ := h.TradeJournalRepo.GetStats(user.ID)

	data := map[string]interface{}{
		"Title":    "Trade Journal - Investo",
		"Journals": journals,
		"Stats":    stats,
		"User":     user,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/journal.html", data)
}

func (h *PortfolioHandler) JournalCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	stockCode := strings.ToUpper(strings.TrimSpace(r.FormValue("stock_code")))
	entryDate, _ := time.Parse("2006-01-02", r.FormValue("entry_date"))
	entryPrice := parseFloat(r.FormValue("entry_price"))
	quantity := parseFloat(r.FormValue("quantity"))
	direction := r.FormValue("direction")
	strategyUsed := r.FormValue("strategy_used")
	notes := r.FormValue("notes")
	emotions := r.FormValue("emotions")
	outcome := r.FormValue("outcome")

	if stockCode == "" || entryPrice <= 0 || quantity <= 0 {
		http.Error(w, "Data tidak lengkap", http.StatusBadRequest)
		return
	}
	if direction == "" {
		direction = "buy"
	}

	journal := &model.TradeJournal{
		UserID:       user.ID,
		StockCode:    stockCode,
		EntryDate:    entryDate,
		EntryPrice:   entryPrice,
		Quantity:     quantity,
		Direction:    direction,
		StrategyUsed: strategyUsed,
		Notes:        notes,
		Emotions:     emotions,
		Outcome:      outcome,
	}

	exitDateStr := r.FormValue("exit_date")
	if exitDateStr != "" {
		exitDate, err := time.Parse("2006-01-02", exitDateStr)
		if err == nil {
			journal.ExitDate = &exitDate
		}
	}

	exitPriceStr := r.FormValue("exit_price")
	if exitPriceStr != "" {
		ep := parseFloat(exitPriceStr)
		if ep > 0 {
			journal.ExitPrice = &ep
			pl := (ep - entryPrice) * quantity
			journal.ProfitLoss = &pl
			if entryPrice > 0 {
				plPct := (ep - entryPrice) / entryPrice * 100
				journal.ProfitLossPct = &plPct
			}
		}
	}

	if _, err := h.TradeJournalRepo.Create(journal); err != nil {
		http.Error(w, "Gagal menyimpan trade journal", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/journal", http.StatusSeeOther)
}

func (h *PortfolioHandler) JournalDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	journal, err := h.TradeJournalRepo.FindByID(id)
	if err != nil || journal.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	if err := h.TradeJournalRepo.Delete(id); err != nil {
		http.Error(w, "Gagal menghapus entry", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/journal", http.StatusSeeOther)
}

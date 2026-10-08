package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type AdvancedHandler struct {
	FearGreedSvc      *service.FearGreedService
	SectorRotationSvc *service.SectorRotationService
	PairTradingSvc    *service.PairTradingService
	TradeIdeaSvc      *service.TradeIdeaService
	CustomIndexSvc    *service.CustomIndexService
	CustomIndexRepo   *repository.CustomIndexRepository
	StockRepo         *repository.StockRepository
	Templates         *template.Template
	FactorExposureSvc *service.FactorExposureService
	RiskDecompSvc     *service.RiskDecompositionService
	SupplyChainSvc    *service.SupplyChainService
}

func (h *AdvancedHandler) FearGreedPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Market Fear & Greed Index - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/fear-greed.html", data)
}

func (h *AdvancedHandler) FearGreedJSON(w http.ResponseWriter, r *http.Request) {
	if h.FearGreedSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	result, err := h.FearGreedSvc.Calculate()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	history, _ := h.FearGreedSvc.History(30)

	resp := map[string]interface{}{
		"current": result,
		"history": history,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdvancedHandler) SectorRotationPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Sector Rotation Radar - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/sector-rotation.html", data)
}

func (h *AdvancedHandler) SectorRotationJSON(w http.ResponseWriter, r *http.Request) {
	if h.SectorRotationSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	rotations, err := h.SectorRotationSvc.Analyze()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sectors": rotations,
	})
}

func (h *AdvancedHandler) PairTradingPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Pair Trading Finder - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/pair-trading.html", data)
}

func (h *AdvancedHandler) PairTradingJSON(w http.ResponseWriter, r *http.Request) {
	if h.PairTradingSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	pairs, err := h.PairTradingSvc.FindPairs()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pairs": pairs,
	})
}

func (h *AdvancedHandler) TradeIdeasPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Trade Ideas Generator - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/trade-ideas.html", data)
}

func (h *AdvancedHandler) TradeIdeasJSON(w http.ResponseWriter, r *http.Request) {
	if h.TradeIdeaSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	ideas, err := h.TradeIdeaSvc.Generate()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ideas": ideas,
	})
}

func (h *AdvancedHandler) CustomIndexListPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	indices, _ := h.CustomIndexRepo.FindByUserID(user.ID)

	type IndexSummary struct {
		ID          int64   `json:"id"`
		Name        string  `json:"name"`
		StockCount  int     `json:"stock_count"`
		BaseValue   float64 `json:"base_value"`
		CreatedAt   string  `json:"created_at"`
	}

	var summaries []IndexSummary
	for _, idx := range indices {
		var stocks []string
		json.Unmarshal([]byte(idx.StocksJSON), &stocks)
		summaries = append(summaries, IndexSummary{
			ID:         idx.ID,
			Name:       idx.Name,
			StockCount: len(stocks),
			BaseValue:  idx.BaseValue,
			CreatedAt:  idx.CreatedAt.Format("02 Jan 2006"),
		})
	}

	activeStocks, _ := h.StockRepo.ListActive()
	type StockOption struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	var stockOptions []StockOption
	for _, s := range activeStocks {
		stockOptions = append(stockOptions, StockOption{Code: s.Code, Name: s.Name})
	}

	data := map[string]interface{}{
		"Title":      "Custom Indices - Investo",
		"User":       user,
		"Indices":    summaries,
		"Stocks":     stockOptions,
		"ActivePage": "indices",
	}
	h.Templates.ExecuteTemplate(w, "portfolio/custom-index.html", data)
}

func (h *AdvancedHandler) CustomIndexCreate(w http.ResponseWriter, r *http.Request) {
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
	baseValueStr := r.FormValue("base_value")
	baseValue := 100.0
	if baseValueStr != "" {
		if v, err := strconv.ParseFloat(baseValueStr, 64); err == nil {
			baseValue = v
		}
	}

	stocksJSON := r.FormValue("stocks_json")
	weightsJSON := r.FormValue("weights_json")

	if name == "" || stocksJSON == "" {
		http.Redirect(w, r, "/dashboard/indices", http.StatusSeeOther)
		return
	}

	if weightsJSON == "" {
		weightsJSON = "{}"
	}

	_, err := h.CustomIndexRepo.Create(user.ID, name, stocksJSON, weightsJSON, baseValue)
	if err != nil {
		http.Error(w, "Failed to create index", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/indices", http.StatusSeeOther)
}

func (h *AdvancedHandler) CustomIndexDetailJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid id"})
		return
	}

	result, err := h.CustomIndexSvc.CalculateValue(id)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AdvancedHandler) CustomIndexDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/dashboard/indices", http.StatusSeeOther)
		return
	}

	idx, err := h.CustomIndexRepo.FindByID(id)
	if err != nil || idx.UserID != user.ID {
		http.Redirect(w, r, "/dashboard/indices", http.StatusSeeOther)
		return
	}

	h.CustomIndexRepo.Delete(id)
	http.Redirect(w, r, "/dashboard/indices", http.StatusSeeOther)
}

type CompetitorSummary struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	MarketShare    float64 `json:"market_share"`
	RevenueGrowth  float64 `json:"revenue_growth"`
	Differentiator string  `json:"differentiator"`
	Rivalry        float64 `json:"rivalry"`
	Barrier        float64 `json:"barrier"`
	SupplierPower  float64 `json:"supplier_power"`
	BuyerPower     float64 `json:"buyer_power"`
	Substitute     float64 `json:"substitute"`
}

var competitiveData = map[string][]CompetitorSummary{
	"Banking": {
		{Code: "BBCA", Name: "Bank Central Asia", MarketShare: 28, RevenueGrowth: 12.5, Differentiator: "Largest private bank by market cap, strong digital banking ecosystem", Rivalry: 8, Barrier: 9, SupplierPower: 3, BuyerPower: 7, Substitute: 4},
		{Code: "BBRI", Name: "Bank Rakyat Indonesia", MarketShare: 22, RevenueGrowth: 15.2, Differentiator: "Dominant in micro/SME lending, largest branch network", Rivalry: 8, Barrier: 9, SupplierPower: 3, BuyerPower: 7, Substitute: 4},
		{Code: "BMRI", Name: "Bank Mandiri", MarketShare: 18, RevenueGrowth: 10.8, Differentiator: "State-owned enterprise banking, wholesale dominance", Rivalry: 8, Barrier: 9, SupplierPower: 3, BuyerPower: 7, Substitute: 4},
		{Code: "BBNI", Name: "Bank Negara Indonesia", MarketShare: 12, RevenueGrowth: 8.5, Differentiator: "Government transaction banking, international trade finance", Rivalry: 8, Barrier: 8, SupplierPower: 3, BuyerPower: 7, Substitute: 4},
	},
	"Consumer Goods": {
		{Code: "INDF", Name: "Indofood Sukses Makmur", MarketShare: 25, RevenueGrowth: 8.2, Differentiator: "Vertical integration from wheat to instant noodles", Rivalry: 7, Barrier: 7, SupplierPower: 5, BuyerPower: 6, Substitute: 5},
		{Code: "ICBP", Name: "Indofood CBP", MarketShare: 18, RevenueGrowth: 10.5, Differentiator: "Strong brand portfolio in packaged food", Rivalry: 7, Barrier: 6, SupplierPower: 5, BuyerPower: 6, Substitute: 5},
		{Code: "UNVR", Name: "Unilever Indonesia", MarketShare: 30, RevenueGrowth: 5.1, Differentiator: "Global FMCG brand power, extensive distribution network", Rivalry: 8, Barrier: 8, SupplierPower: 4, BuyerPower: 5, Substitute: 6},
	},
	"Telecommunication": {
		{Code: "TLKM", Name: "Telkom Indonesia", MarketShare: 45, RevenueGrowth: 6.3, Differentiator: "Broadband infrastructure backbone, government ties", Rivalry: 6, Barrier: 9, SupplierPower: 4, BuyerPower: 5, Substitute: 3},
	},
	"Mining": {
		{Code: "ADRO", Name: "Adaro Energy", MarketShare: 20, RevenueGrowth: 15.8, Differentiator: "Low-cost thermal coal producer, strong ESG transition", Rivalry: 6, Barrier: 8, SupplierPower: 3, BuyerPower: 8, Substitute: 5},
	},
	"Automotive": {
		{Code: "ASII", Name: "Astra International", MarketShare: 35, RevenueGrowth: 7.1, Differentiator: "Conglomerate with automotive, heavy equipment, and finance", Rivalry: 7, Barrier: 8, SupplierPower: 5, BuyerPower: 6, Substitute: 4},
		{Code: "AUTO", Name: "Astra Otoparts", MarketShare: 12, RevenueGrowth: 8.0, Differentiator: "OEM parts supplier for Japanese car manufacturers", Rivalry: 7, Barrier: 7, SupplierPower: 6, BuyerPower: 5, Substitute: 4},
	},
	"Infrastructure": {
		{Code: "SMGR", Name: "Semen Indonesia", MarketShare: 28, RevenueGrowth: 4.5, Differentiator: "Cement market leader with national distribution", Rivalry: 7, Barrier: 9, SupplierPower: 5, BuyerPower: 5, Substitute: 3},
	},
}

var competitiveDifferentiators = map[string]string{
	"ADRO": "Low-cost thermal coal producer, strong ESG transition toward renewables",
	"ASII": "Conglomerate with automotive, heavy equipment, agribusiness, and financial services",
	"AUTO": "OEM parts supplier for major Japanese automotive manufacturers in Indonesia",
	"BBCA": "Largest private bank by market cap, strong digital banking ecosystem (BCA Mobile, myBCA)",
	"BBRI": "Dominant in micro/SME lending through BRI Unit network across all Indonesian villages",
	"BMRI": "State-owned enterprise banking, wholesale banking dominance, strong digital push (Livin')",
	"BBNI": "Government transaction banking, international trade finance, SOE ecosystem",
	"INDF": "Vertical integration from wheat flour production to consumer branded products",
	"ICBP": "Strong brand portfolio: Indomie, Chitato, Pop Mie, with nationwide distribution",
	"SMGR": "Cement market leader with national production and distribution network",
	"TLKM": "Telecommunications infrastructure backbone, broadband monopoly, data center expansion",
	"UNVR": "Global FMCG brand portfolio with localized production and extensive rural distribution",
	"UNTR": "Heavy equipment distributor for Komatsu, mining contractor with diversified customer base",
}

func (h *AdvancedHandler) FactorExposurePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Factor Exposure Report - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/factor-exposure.html", data)
}

func (h *AdvancedHandler) FactorExposureJSON(w http.ResponseWriter, r *http.Request) {
	if h.FactorExposureSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	exposures, err := h.FactorExposureSvc.GetAllExposures()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exposures": exposures,
	})
}

func (h *AdvancedHandler) FactorTimingPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Factor Timing Model - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/factor-timing.html", data)
}

func (h *AdvancedHandler) FactorTimingJSON(w http.ResponseWriter, r *http.Request) {
	if h.FactorExposureSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	timings, err := h.FactorExposureSvc.GetFactorTiming()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timings": timings,
	})
}

func (h *AdvancedHandler) RiskDecompJSON(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": "invalid portfolio id"})
		return
	}

	if h.RiskDecompSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	decomp, err := h.RiskDecompSvc.DecomposePortfolio(id)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decomp)
}

func (h *AdvancedHandler) SmartBetaPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Smart Beta Screener - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/smart-beta.html", data)
}

func (h *AdvancedHandler) SupplyChainPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Supply Chain Mapper - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/supply-chain.html", data)
}

func (h *AdvancedHandler) SupplyChainJSON(w http.ResponseWriter, r *http.Request) {
	if h.SupplyChainSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	links, err := h.SupplyChainSvc.GetAllLinks()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"links": links,
	})
}

func (h *AdvancedHandler) IndustryLifecyclePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Industry Lifecycle Analyzer - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/industry-lifecycle.html", data)
}

func (h *AdvancedHandler) IndustryLifecycleJSON(w http.ResponseWriter, r *http.Request) {
	if h.SectorRotationSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	stages, err := h.SectorRotationSvc.AnalyzeLifecycle()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stages": stages,
	})
}

func (h *AdvancedHandler) CompetitivePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Competitive Landscape - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/competitive.html", data)
}

func (h *AdvancedHandler) CompetitiveJSON(w http.ResponseWriter, r *http.Request) {
	sectorName := r.URL.Query().Get("sector")

	if sectorName != "" {
		companies, ok := competitiveData[sectorName]
		if !ok {
			companies = []CompetitorSummary{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"companies": companies,
		})
		return
	}

	type sectorOption struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	var sectors []sectorOption
	for name, companies := range competitiveData {
		sectors = append(sectors, sectorOption{Name: name, Count: len(companies)})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sectors": sectors,
	})
}

func (h *AdvancedHandler) ConcentrationJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if h.SupplyChainSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	risk, err := h.SupplyChainSvc.AnalyzeConcentration(code)
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(risk)
}

func (h *AdvancedHandler) ArbOpportunitiesJSON(w http.ResponseWriter, r *http.Request) {
	if h.PairTradingSvc == nil {
		writeJSONSimple(w, map[string]string{"error": "service not available"})
		return
	}

	opps, err := h.PairTradingSvc.FindArbitrageOpportunities()
	if err != nil {
		writeJSONSimple(w, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"opportunities": opps,
	})
}

func writeJSONSimple(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

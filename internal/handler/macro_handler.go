package handler

import (
	"encoding/json"
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type MacroHandler struct {
	MacroDashboardSvc     *service.MacroDashboardService
	YieldCurveSvc         *service.YieldCurveService
	LiquidityFlowSvc      *service.LiquidityFlowService
	LeadingIndicatorsSvc  *service.LeadingIndicatorsService
	MacroSpilloverSvc     *service.MacroSpilloverService
	MAScreenerSvc         *service.MAScreenerService
	EarningsQualitySvc    *service.EarningsQualityService
	ManagementQualitySvc  *service.ManagementQualityService
	MoatSvc               *service.MoatService
	Templates             *template.Template
}

func (h *MacroHandler) MacroDashboardPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Macro Factor Dashboard - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/macro-dashboard.html", data)
}

func (h *MacroHandler) MacroDashboardJSON(w http.ResponseWriter, r *http.Request) {
	indicators, err := h.MacroDashboardSvc.GetMacroSnapshot()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load macro data"})
		return
	}
	if indicators == nil {
		indicators = []service.MacroIndicator{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(indicators)
}

func (h *MacroHandler) YieldCurvePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Yield Curve Analyzer - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/yield-curve.html", data)
}

func (h *MacroHandler) YieldCurveJSON(w http.ResponseWriter, r *http.Request) {
	curve, err := h.YieldCurveSvc.GetYieldCurve()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load yield curve"})
		return
	}
	analysis, err := h.YieldCurveSvc.AnalyzeCurve()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to analyze curve"})
		return
	}
	if curve == nil {
		curve = []service.YieldPoint{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"curve":    curve,
		"analysis": analysis,
	})
}

func (h *MacroHandler) LiquidityFlowPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Liquidity Flow Tracker - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/liquidity-flow.html", data)
}

func (h *MacroHandler) LiquidityFlowJSON(w http.ResponseWriter, r *http.Request) {
	entries, _ := h.LiquidityFlowSvc.GetFlowHistory(30)
	summary, _ := h.LiquidityFlowSvc.GetCurrentFlow()
	if entries == nil {
		entries = []service.FlowEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"summary": summary,
	})
}

func (h *MacroHandler) LeadingIndicatorsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Leading Indicator Dashboard - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/leading-indicators.html", data)
}

func (h *MacroHandler) LeadingIndicatorsJSON(w http.ResponseWriter, r *http.Request) {
	indicators, _ := h.LeadingIndicatorsSvc.GetLeadingIndicators()
	prediction, _ := h.LeadingIndicatorsSvc.PredictGDP()
	if indicators == nil {
		indicators = []service.LeadingIndicator{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"indicators": indicators,
		"prediction": prediction,
	})
}

func (h *MacroHandler) MacroSpilloverPage(w http.ResponseWriter, r *http.Request) {
	scenarios := h.MacroSpilloverSvc.GetScenarios()
	data := map[string]interface{}{
		"Title":     "Global Macro Spillover - Investo",
		"User":      safeUser(middleware.GetUser(r)),
		"Scenarios": scenarios,
	}
	h.Templates.ExecuteTemplate(w, "market/macro-spillover.html", data)
}

func (h *MacroHandler) MacroSpilloverJSON(w http.ResponseWriter, r *http.Request) {
	event := r.URL.Query().Get("event")
	if event == "" {
		event = "Fed Hike 25bps"
	}
	result, err := h.MacroSpilloverSvc.AnalyzeSpillover(event)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "analysis failed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *MacroHandler) MATargetsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "M&A Target Screener - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/ma-targets.html", data)
}

func (h *MacroHandler) MATargetsJSON(w http.ResponseWriter, r *http.Request) {
	targets, err := h.MAScreenerSvc.ScreenMATargets()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "screening failed"})
		return
	}
	if targets == nil {
		targets = []service.MATarget{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}

func (h *MacroHandler) MoatPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	data := map[string]interface{}{
		"Title":  code + " - Moat Analysis - Investo",
		"User":   safeUser(middleware.GetUser(r)),
		"Code":   code,
	}
	h.Templates.ExecuteTemplate(w, "stocks/moat.html", data)
}

func (h *MacroHandler) MoatJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.MoatSvc.AnalyzeMoat(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "moat analysis failed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *MacroHandler) EarningsQualityJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.EarningsQualitySvc.CalculateQualityScore(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "quality analysis failed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *MacroHandler) ManagementQualityJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.ManagementQualitySvc.AssessManagement(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "management assessment failed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *MacroHandler) DCFBuilderPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "DCF Scenario Builder - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "calculators/dcf-builder.html", data)
}

func (h *MacroHandler) ConglomeratePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Conglomerate Discount Calculator - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "calculators/conglomerate.html", data)
}

type EarningsQualityHandler struct {
	EarningsQualitySvc *service.EarningsQualityService
	StockRepo          *repository.StockRepository
}

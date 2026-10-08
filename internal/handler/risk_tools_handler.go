package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type RiskToolsHandler struct {
	Templates           *template.Template
	WashSaleService     *service.WashSaleService
	BetaWeightedService *service.BetaWeightedService
	PortfolioRepo       *repository.PortfolioRepository
}

// ownedPortfolioRepo prefers the explicitly wired PortfolioRepo and falls back
// to the repository held by BetaWeightedService so ownership can always be
// verified without changing handler construction.
func (h *RiskToolsHandler) ownedPortfolioRepo() *repository.PortfolioRepository {
	if h.PortfolioRepo != nil {
		return h.PortfolioRepo
	}
	if h.BetaWeightedService != nil {
		return h.BetaWeightedService.PortfolioRepo
	}
	return nil
}

func (h *RiskToolsHandler) WashSalePage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))
	if user.ID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	washSales, _ := h.WashSaleService.DetectWashSales(user.ID)

	var totalLossDisallowed float64
	for _, ws := range washSales {
		totalLossDisallowed += ws.LossDisallowed
	}

	h.Templates.ExecuteTemplate(w, "risk/wash-sale.html", map[string]interface{}{
		"Title":              "Wash Sale Detector - Investo",
		"User":               user,
		"WashSales":          washSales,
		"TotalLossDisallowed": totalLossDisallowed,
		"Count":              len(washSales),
	})
}

func (h *RiskToolsHandler) WashSaleJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	washSales, err := h.WashSaleService.DetectWashSales(user.ID)
	if err != nil {
		log.Printf("washsale user=%d: %v", user.ID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Terjadi kesalahan internal"})
		return
	}

	if washSales == nil {
		washSales = []service.WashSale{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(washSales)
}

func (h *RiskToolsHandler) BetaWeightedPage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))
	if user.ID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.Templates.ExecuteTemplate(w, "risk/beta-weighted.html", map[string]interface{}{
		"Title": "Beta-Weighted Portfolio - Investo",
		"User":  user,
	})
}

func (h *RiskToolsHandler) BetaJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "id")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid portfolio id"})
		return
	}

	if loadOwnedPortfolio(w, r, h.ownedPortfolioRepo(), portfolioID) == nil {
		return
	}

	result, err := h.BetaWeightedService.CalcBetaWeighted(portfolioID)
	if err != nil {
		log.Printf("beta portfolio=%d: %v", portfolioID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Terjadi kesalahan internal"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *RiskToolsHandler) HedgeFinderPage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))
	if user.ID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.Templates.ExecuteTemplate(w, "risk/hedge-finder.html", map[string]interface{}{
		"Title": "Correlation Hedge Finder - Investo",
		"User":  user,
	})
}

func (h *RiskToolsHandler) HedgeJSON(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "id")
	portfolioID, err := strconv.ParseInt(portfolioIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid portfolio id"})
		return
	}

	if loadOwnedPortfolio(w, r, h.ownedPortfolioRepo(), portfolioID) == nil {
		return
	}

	suggestions, err := h.BetaWeightedService.FindHedge(portfolioID)
	if err != nil {
		log.Printf("hedge portfolio=%d: %v", portfolioID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Terjadi kesalahan internal"})
		return
	}

	if suggestions == nil {
		suggestions = []service.HedgeSuggestion{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

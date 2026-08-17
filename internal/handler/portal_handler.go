package handler

import (
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"
)

// PortalHandler is the B2C self-service portal (clean layout, separate from admin).
type PortalHandler struct {
	PaperSvc   *service.PaperTradingService
	PaymentSvc *service.PaymentService
	Templates  *template.Template
}

func (h *PortalHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	plan, _ := h.PaymentSvc.GetUserPlan(user.ID)
	portfolios, _ := h.PaperSvc.GetPortfolios(user.ID)

	totalValue := 0.0
	for _, p := range portfolios {
		totalValue += p.CashBalance
	}

	data := map[string]interface{}{
		"Title":       "Portal — Investo",
		"User":        user,
		"Plan":        plan,
		"Portfolios":  portfolios,
		"TotalValue":  totalValue,
		"ActivePage":  "dashboard",
	}
	h.Templates.ExecuteTemplate(w, "portal/dashboard.html", data)
}

func (h *PortalHandler) Portfolios(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	portfolios, _ := h.PaperSvc.GetPortfolios(user.ID)

	data := map[string]interface{}{
		"Title":      "Portofolio — Portal Investo",
		"User":       user,
		"Portfolios": portfolios,
		"ActivePage": "portfolios",
	}
	h.Templates.ExecuteTemplate(w, "portal/portfolios.html", data)
}

func (h *PortalHandler) Subscription(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	plan, _ := h.PaymentSvc.GetUserPlan(user.ID)

	data := map[string]interface{}{
		"Title":      "Langganan — Portal Investo",
		"User":       user,
		"Plan":       plan,
		"ActivePage": "subscription",
	}
	h.Templates.ExecuteTemplate(w, "portal/subscription.html", data)
}

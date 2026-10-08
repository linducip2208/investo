package handler

import (
	"html/template"
	"net/http"

	"investo/internal/middleware"
)

type CalculatorHandler struct {
	Templates *template.Template
}

func (h *CalculatorHandler) Hub(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Kalkulator Investasi - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "calculators/hub.html", data)
}

func (h *CalculatorHandler) RightIssue(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Kalkulator Right Issue - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "calculators/right-issue.html", data)
}

func (h *CalculatorHandler) TaxCalendar(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Kalender Pajak & Tax Loss Harvesting - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "calculators/tax-calendar.html", data)
}

package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type PaperTradingHandler struct {
	Service    *service.PaperTradingService
	PaymentSvc *service.PaymentService
	Templates  *template.Template
}

func (h *PaperTradingHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	plan, _ := h.PaymentSvc.GetUserPlan(user.ID)
	badge := middleware.GetPlanBadge(plan)

	data := map[string]interface{}{
		"Title":      "Paper Trading - Investo",
		"User":       user,
		"ActivePage": "paper-trading",
		"Plan":       plan,
		"PlanBadge":  badge,
	}
	h.Templates.ExecuteTemplate(w, "paper-trading/index.html", data)
}

func (h *PaperTradingHandler) Trade(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		PortfolioID int64   `json:"portfolio_id"`
		StockCode   string  `json:"stock_code"`
		TradeType   string  `json:"type"`
		Quantity    float64 `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.PortfolioID == 0 {
		p, err := h.Service.GetUserPortfolio(user.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get portfolio"})
			return
		}
		req.PortfolioID = p.ID
	}

	if err := h.Service.ExecuteTrade(req.PortfolioID, req.StockCode, req.TradeType, req.Quantity); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	summary, _ := h.Service.GetPortfolioSummary(req.PortfolioID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"portfolio": summary,
	})
}

func (h *PaperTradingHandler) PortfolioJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	if idStr == "" || idStr == "my" {
		p, err := h.Service.GetUserPortfolio(user.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		idStr = strconv.FormatInt(p.ID, 10)
	}

	portfolioID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid portfolio id"})
		return
	}

	summary, err := h.Service.GetPortfolioSummary(portfolioID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	trades, _ := h.Service.GetTradeHistory(portfolioID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"portfolio": summary,
		"trades":    trades,
	})
}

func (h *PaperTradingHandler) LeaderboardJSON(w http.ResponseWriter, r *http.Request) {
	leaders, err := h.Service.GetLeaderboard()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"leaderboard": leaders,
	})
}

type PaymentHandler struct {
	Service   *service.PaymentService
	Templates *template.Template
}

func (h *PaymentHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Plan   string  `json:"plan"`
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	if req.Amount == 0 {
		switch req.Plan {
		case "free":
			req.Amount = 0
		case "pro":
			req.Amount = 99000
		case "whitelabel":
			req.Amount = 4999000
		default:
			req.Amount = 99000
		}
	}

	txID, err := h.Service.CreateTransaction(user.ID, req.Plan, req.Amount)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction_id": txID,
		"redirect_url":   "/api/invoice/" + txID,
	})
}

func (h *PaymentHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	if err := h.Service.HandleCallback(payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *PaymentHandler) Invoice(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	txID := chi.URLParam(r, "txID")
	inv, err := h.Service.GenerateInvoice(user.ID, txID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Invoice not found"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(renderInvoiceHTML(inv)))
}

func renderInvoiceHTML(inv *model.Invoice) string {
	itemsHTML := ""
	for _, item := range inv.Items {
		itemsHTML += fmt.Sprintf(`<tr><td style="padding:12px;border-bottom:1px solid #334155">%s</td><td style="padding:12px;border-bottom:1px solid #334155;text-align:center">%d</td><td style="padding:12px;border-bottom:1px solid #334155;text-align:right;font-family:'JetBrains Mono',monospace">Rp %s</td></tr>`,
			item.Description, item.Qty, formatNum(item.Amount))
	}
	return fmt.Sprintf(`<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Invoice %s - Investo</title><style>body{font-family:'Inter',system-ui,sans-serif;background:#0b1120;color:#e2e8f0;display:flex;justify-content:center;padding:40px 20px;margin:0}*{margin:0;box-sizing:border-box}.inv{max-width:700px;width:100%%;background:#1e293b;border:1px solid #334155;border-radius:16px;padding:40px}.head{display:flex;justify-content:space-between;margin-bottom:32px}.head h1{font-size:24px;color:#fff}.head .meta{text-align:right;color:#94a3b8;font-size:13px}.table{width:100%%;border-collapse:collapse;margin:24px 0}.table th{text-align:left;padding:12px;color:#94a3b8;font-size:11px;text-transform:uppercase;letter-spacing:.05em;border-bottom:2px solid #334155}.total{text-align:right;padding:12px;font-weight:700;font-size:16px;color:#fff}.status{display:inline-block;padding:4px 12px;border-radius:20px;font-size:12px;font-weight:600}.status-pending{background:#713f12;color:#fbbf24}.status-active{background:#064e3b;color:#34d399}.footer{margin-top:32px;text-align:center;color:#475569;font-size:12px}@media print{body{background:#fff;color:#000}.inv{background:#fff;border:1px solid #ccc}}</style></head><body><div class="inv"><div class="head"><div><h1>INVOICE</h1><p style="color:#94a3b8;font-size:14px">%s</p></div><div class="meta"><p>%s</p><p>Status: <span class="status status-%s">%s</span></p></div></div><div style="margin-bottom:24px"><p style="color:#94a3b8;font-size:12px">Kepada:</p><p style="color:#fff;font-weight:600">%s</p><p style="color:#94a3b8;font-size:13px">%s</p></div><table class="table"><thead><tr><th>Deskripsi</th><th style="text-align:center">Qty</th><th style="text-align:right">Jumlah</th></tr></thead><tbody>%s</tbody><tfoot><tr><td colspan="2" style="text-align:right;padding:12px;font-weight:600">Total</td><td class="total">Rp %s</td></tr></tfoot></table><div class="footer"><p>Terima kasih telah menggunakan Investo.</p><p>Investo - Platform Analisa Saham & Forex Indonesia</p></div></div></body></html>`,
		inv.Number, inv.Number, inv.Date, inv.Status, inv.Status, inv.CustomerName, inv.CustomerEmail, itemsHTML, formatNum(inv.Total))
}

func formatNum(v float64) string {
	s := strconv.FormatInt(int64(v), 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func (h *PaymentHandler) SubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	plan, err := h.Service.GetUserPlan(user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plan":  plan,
		"badge": middleware.GetPlanBadge(plan),
	})
}

func (h *PaymentHandler) UpgradeSubscription(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	if err := h.Service.UpgradePlan(user.ID, req.Plan); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"plan":    req.Plan,
	})
}

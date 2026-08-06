package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"
)

type AlertHandler struct {
	AlertRepo      *repository.AlertRepository
	AlertEngine    *service.AlertEngine
	TelegramSvc    *service.TelegramService
	StockPriceRepo *repository.StockPriceRepository
	StockRepo      *repository.StockRepository
	Templates      *template.Template
}

func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	alerts, err := h.AlertRepo.FindByUserID(user.ID)
	if err != nil {
		http.Error(w, "Gagal memuat alert", http.StatusInternalServerError)
		return
	}

	triggeredAlerts, _ := h.AlertRepo.FindTriggeredByUserID(user.ID)

	templates := []service.AlertTemplate{}
	if h.AlertEngine != nil {
		templates = h.AlertEngine.GetTemplates()
	}

	data := map[string]interface{}{
		"Title":           "Alert Harga - Investo",
		"Alerts":          alerts,
		"TriggeredAlerts": triggeredAlerts,
		"Templates":       templates,
		"User":            user,
	}
	h.Templates.ExecuteTemplate(w, "alerts/list.html", data)
}

func (h *AlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	stockID, _ := strconv.ParseInt(r.FormValue("stock_id"), 10, 64)
	condition := r.FormValue("condition")
	targetPrice := parseFloat(r.FormValue("target_price"))

	if stockID == 0 || condition == "" || targetPrice <= 0 {
		http.Redirect(w, r, "/dashboard/alerts", http.StatusSeeOther)
		return
	}

	alert := &model.Alert{
		UserID:      user.ID,
		StockID:     stockID,
		Condition:   condition,
		TargetPrice: targetPrice,
		IsActive:    true,
	}

	_, err := h.AlertRepo.Create(alert)
	if err != nil {
		http.Error(w, "Gagal membuat alert", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/alerts", http.StatusSeeOther)
}

func (h *AlertHandler) CreateAdvanced(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		StockID          int64                   `json:"stock_id"`
		Conditions       []model.AlertCondition   `json:"conditions"`
		NotificationType string                  `json:"notification_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.StockID == 0 || len(req.Conditions) == 0 {
		http.Error(w, "stock_id dan conditions wajib diisi", http.StatusBadRequest)
		return
	}

	nt := req.NotificationType
	if nt == "" {
		nt = "web"
	}

	conditionsJSON, err := json.Marshal(req.Conditions)
	if err != nil {
		http.Error(w, "Gagal encode conditions", http.StatusInternalServerError)
		return
	}

	alert := &model.Alert{
		UserID:           user.ID,
		StockID:          req.StockID,
		Condition:        "advanced",
		TargetPrice:      0,
		ConditionsJSON:   string(conditionsJSON),
		NotificationType: nt,
		IsActive:         true,
	}

	id, err := h.AlertRepo.Create(alert)
	if err != nil {
		http.Error(w, "Gagal membuat alert", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func (h *AlertHandler) AlertTemplates(w http.ResponseWriter, r *http.Request) {
	if h.AlertEngine == nil {
		json.NewEncoder(w).Encode([]service.AlertTemplate{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.AlertEngine.GetTemplates())
}

func (h *AlertHandler) History(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	triggeredAlerts, err := h.AlertRepo.FindTriggeredByUserID(user.ID)
	if err != nil {
		http.Error(w, "Gagal memuat history", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "History Alert - Investo",
		"TriggeredAlerts": triggeredAlerts,
		"User":            user,
	}
	h.Templates.ExecuteTemplate(w, "alerts/list.html", data)
}

func (h *AlertHandler) TestAlert(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	alertIDStr := chi.URLParam(r, "id")
	alertID, err := strconv.ParseInt(alertIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	alert, err := h.AlertRepo.FindByID(alertID)
	if err != nil || alert.UserID != user.ID {
		http.Error(w, "Alert tidak ditemukan", http.StatusNotFound)
		return
	}

	if h.AlertEngine == nil {
		http.Error(w, "Alert engine tidak tersedia", http.StatusInternalServerError)
		return
	}

	triggered, msg, err := h.AlertEngine.EvaluateAlert(alert)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"triggered": triggered,
		"message":   msg,
	})
}

func (h *AlertHandler) TelegramConnect(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ChatID string `json:"chat_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ChatID == "" {
		http.Error(w, "chat_id wajib diisi", http.StatusBadRequest)
		return
	}

	if h.TelegramSvc == nil {
		http.Error(w, "Telegram service tidak tersedia", http.StatusInternalServerError)
		return
	}

	h.TelegramSvc.SetChatID(user.ID, req.ChatID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Telegram chat ID berhasil disimpan",
	})
}

func (h *AlertHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	alertID, _ := strconv.ParseInt(r.FormValue("alert_id"), 10, 64)
	active := r.FormValue("active") == "1"

	alert, err := h.AlertRepo.FindByID(alertID)
	if err != nil || alert.UserID != user.ID {
		http.Error(w, "Alert tidak ditemukan", http.StatusNotFound)
		return
	}

	if err := h.AlertRepo.ToggleActive(alertID, active); err != nil {
		http.Error(w, "Gagal mengubah status alert", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/alerts", http.StatusSeeOther)
}

func (h *AlertHandler) AlertsJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stockCode := r.URL.Query().Get("stock_code")
	var alerts []model.Alert

	if stockCode != "" {
		stock, err := h.StockRepo.FindByCode(stockCode)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		alerts, err = h.AlertRepo.FindByUserIDAndStockID(user.ID, stock.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
	} else {
		var err error
		alerts, err = h.AlertRepo.FindByUserID(user.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
	}

	type AlertJSON struct {
		ID          int64   `json:"id"`
		StockID     int64   `json:"stock_id"`
		Condition   string  `json:"condition"`
		TargetPrice float64 `json:"target_price"`
		IsActive    bool    `json:"is_active"`
	}

	result := make([]AlertJSON, 0, len(alerts))
	for _, a := range alerts {
		result = append(result, AlertJSON{
			ID:          a.ID,
			StockID:     a.StockID,
			Condition:   a.Condition,
			TargetPrice: a.TargetPrice,
			IsActive:    a.IsActive,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AlertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	alertID, _ := strconv.ParseInt(r.FormValue("alert_id"), 10, 64)

	alert, err := h.AlertRepo.FindByID(alertID)
	if err != nil || alert.UserID != user.ID {
		http.Error(w, "Alert tidak ditemukan", http.StatusNotFound)
		return
	}

	if err := h.AlertRepo.Delete(alertID); err != nil {
		http.Error(w, "Gagal menghapus alert", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/alerts", http.StatusSeeOther)
}

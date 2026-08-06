package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type EngagementHandler struct {
	PredictionService  *service.PredictionMarketService
	AchievementService *service.AchievementService
	WhatsAppService    *service.WhatsAppBotService
	WeatherService     *service.PortfolioWeatherService
	Templates          *template.Template
}

func (h *EngagementHandler) PredictionPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Prediction Market - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/prediction.html", data)
}

func (h *EngagementHandler) PredictionsJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	predictions, err := h.PredictionService.GetUserPredictions(user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(predictions)
}

func (h *EngagementHandler) PredictionLeaderboardJSON(w http.ResponseWriter, r *http.Request) {
	leaders, err := h.PredictionService.GetLeaderboard()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaders)
}

func (h *EngagementHandler) PredictionSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Harus login"})
		return
	}

	var req struct {
		Code  string  `json:"code"`
		Price float64 `json:"price"`
		Date  string  `json:"date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	if req.Code == "" || req.Price <= 0 || req.Date == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Semua field wajib diisi"})
		return
	}

	if err := h.PredictionService.SubmitPrediction(user.ID, req.Code, req.Price, req.Date); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Prediksi berhasil disimpan"})
}

func (h *EngagementHandler) PredictionGrade(w http.ResponseWriter, r *http.Request) {
	if err := h.PredictionService.GradePredictions(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Predictions graded successfully"})
}

func (h *EngagementHandler) AchievementPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Achievements - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/achievements.html", data)
}

func (h *EngagementHandler) AchievementsJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	achievements, err := h.AchievementService.CheckAchievements(user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(achievements)
}

func (h *EngagementHandler) WhatsAppBotPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "WhatsApp Bot - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/whatsapp-bot.html", data)
}

func (h *EngagementHandler) WhatsAppBotTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	response, err := h.WhatsAppService.ProcessCommand("test-user", req.Command)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"response": "Error: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

func (h *EngagementHandler) PortfolioWeatherJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	idStr := chi.URLParam(r, "id")

	var portfolioID int64
	if _, err := fmt.Sscanf(idStr, "%d", &portfolioID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid portfolio ID"})
		return
	}

	result, err := h.WeatherService.GetWeather(portfolioID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = user

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

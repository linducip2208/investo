package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"
)

type BotHandler struct {
	SettingRepo      *repository.SettingRepository
	BotCenterService *service.BotCenterService
	SignalService    *service.SignalGeneratorService
	Templates        *template.Template
}

func (h *BotHandler) BotCenterPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title": "Bot Center - Investo",
		"User":  user,
	}
	h.Templates.ExecuteTemplate(w, "market/bot-center.html", data)
}

func (h *BotHandler) ListBots(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	botsJSON, err := h.SettingRepo.Get("bots_" + strconv.FormatInt(user.ID, 10))
	if err != nil || botsJSON == "" {
		botsJSON = "[]"
	}

	var bots []model.Bot
	if err := json.Unmarshal([]byte(botsJSON), &bots); err != nil {
		bots = []model.Bot{}
	}

	writeJSON(w, http.StatusOK, bots, nil)
}

func (h *BotHandler) CreateBot(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var req struct {
		Name   string            `json:"name"`
		Type   string            `json:"type"`
		Config map[string]string `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}, nil)
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"}, nil)
		return
	}
	if req.Type != "telegram" && req.Type != "webhook" && req.Type != "email" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be telegram, webhook, or email"}, nil)
		return
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode config"}, nil)
		return
	}

	bots, _ := h.loadUserBots(user.ID)
	bot := model.Bot{
		ID:        time.Now().UnixNano(),
		UserID:    user.ID,
		Name:      req.Name,
		Type:      req.Type,
		Config:    string(configJSON),
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bots = append(bots, bot)

	if err := h.saveUserBots(user.ID, bots); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save bot"}, nil)
		return
	}

	writeJSON(w, http.StatusCreated, bot, nil)
}

func (h *BotHandler) TestBot(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var req struct {
		BotID int64 `json:"bot_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}, nil)
		return
	}

	bot, err := h.findUserBot(user.ID, req.BotID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bot not found"}, nil)
		return
	}

	testSignal := service.TradingSignal{
		StockCode:   "BBRI",
		StockName:   "Bank Rakyat Indonesia",
		Type:        "BUY",
		Confidence:  85,
		Reason:      "Ini adalah sinyal test dari Bot Center",
		Indicators:  []string{"RSI", "MACD"},
		Price:       4500,
		TargetPrice: 5000,
		StopLoss:    4200,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := h.BotCenterService.ForwardSignal(testSignal, *bot); err != nil {
		log.Printf("[bot-center] test bot %d failed: %v", req.BotID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "test failed: " + err.Error()}, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Test signal sent to " + bot.Name,
	}, nil)
}

func (h *BotHandler) ToggleBot(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var req struct {
		BotID int64 `json:"bot_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}, nil)
		return
	}

	bots, err := h.loadUserBots(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load bots"}, nil)
		return
	}

	found := false
	for i := range bots {
		if bots[i].ID == req.BotID {
			bots[i].Active = !bots[i].Active
			bots[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bot not found"}, nil)
		return
	}

	if err := h.saveUserBots(user.ID, bots); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"}, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true}, nil)
}

func (h *BotHandler) DeleteBot(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var req struct {
		BotID int64 `json:"bot_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}, nil)
		return
	}

	bots, err := h.loadUserBots(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load bots"}, nil)
		return
	}

	var filtered []model.Bot
	found := false
	for _, b := range bots {
		if b.ID == req.BotID {
			found = true
			continue
		}
		filtered = append(filtered, b)
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bot not found"}, nil)
		return
	}

	if err := h.saveUserBots(user.ID, filtered); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"}, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true}, nil)
}

func (h *BotHandler) ForwardRulesPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title": "Auto-Forward Rules - Investo",
		"User":  user,
	}
	h.Templates.ExecuteTemplate(w, "market/bot-forward-rules.html", data)
}

func (h *BotHandler) SaveForwardRules(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var rules map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}, nil)
		return
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode rules"}, nil)
		return
	}

	if err := h.SettingRepo.Set("forward_rules_"+strconv.FormatInt(user.ID, 10), string(rulesJSON)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save rules"}, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true}, nil)
}

func (h *BotHandler) GetForwardRules(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	rulesJSON, err := h.SettingRepo.Get("forward_rules_" + strconv.FormatInt(user.ID, 10))
	if err != nil || rulesJSON == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"confidence_min": 70,
			"pattern_match":  "",
			"signal_types":   []string{"BUY", "SELL"},
			"target_bot_id":  0,
		}, nil)
		return
	}

	var rules map[string]interface{}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"confidence_min": 70,
			"pattern_match":  "",
			"signal_types":   []string{"BUY", "SELL"},
			"target_bot_id":  0,
		}, nil)
		return
	}

	writeJSON(w, http.StatusOK, rules, nil)
}

func (h *BotHandler) ExchangesPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	data := map[string]interface{}{
		"Title": "Connected Exchanges - Investo",
		"User":  user,
	}
	h.Templates.ExecuteTemplate(w, "market/exchanges.html", data)
}

func (h *BotHandler) PricingPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	currentPlan := "free"
	if user != nil {
		subJSON, err := h.SettingRepo.Get("subscription_" + strconv.FormatInt(user.ID, 10))
		if err == nil && subJSON != "" {
			var sub model.Subscription
			if json.Unmarshal([]byte(subJSON), &sub) == nil {
				currentPlan = sub.Plan
			}
		}
	}

	data := map[string]interface{}{
		"Title":       "Harga - Investo",
		"User":        user,
		"CurrentPlan": currentPlan,
	}
	h.Templates.ExecuteTemplate(w, "market/pricing.html", data)
}

func (h *BotHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"}, nil)
		return
	}

	if req.Plan != "free" && req.Plan != "pro" && req.Plan != "whitelabel" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid plan"}, nil)
		return
	}

	now := time.Now()
	var expiresAt *time.Time
	if req.Plan == "pro" {
		exp := now.AddDate(0, 1, 0)
		expiresAt = &exp
	}

	sub := model.Subscription{
		UserID:    user.ID,
		Plan:      req.Plan,
		Status:    "active",
		StartedAt: now,
		ExpiresAt: expiresAt,
	}

	subJSON, err := json.Marshal(sub)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode subscription"}, nil)
		return
	}

	if err := h.SettingRepo.Set("subscription_"+strconv.FormatInt(user.ID, 10), string(subJSON)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save subscription"}, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"plan":    req.Plan,
	}, nil)
}

func (h *BotHandler) loadUserBots(userID int64) ([]model.Bot, error) {
	botsJSON, err := h.SettingRepo.Get("bots_" + strconv.FormatInt(userID, 10))
	if err != nil || botsJSON == "" {
		return []model.Bot{}, nil
	}

	var bots []model.Bot
	if err := json.Unmarshal([]byte(botsJSON), &bots); err != nil {
		return []model.Bot{}, nil
	}
	return bots, nil
}

func (h *BotHandler) saveUserBots(userID int64, bots []model.Bot) error {
	botsJSON, err := json.Marshal(bots)
	if err != nil {
		return err
	}
	return h.SettingRepo.Set("bots_"+strconv.FormatInt(userID, 10), string(botsJSON))
}

func (h *BotHandler) findUserBot(userID, botID int64) (*model.Bot, error) {
	bots, err := h.loadUserBots(userID)
	if err != nil {
		return nil, err
	}
	for i := range bots {
		if bots[i].ID == botID {
			return &bots[i], nil
		}
	}
	return nil, http.ErrNotSupported
}

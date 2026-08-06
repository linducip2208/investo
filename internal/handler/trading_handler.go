package handler

import (
	"encoding/json"
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"
)

type TradingHandler struct {
	BasketSvc *service.BasketTradingService
	Templates *template.Template
}

func (h *TradingHandler) BasketPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":      "Basket Trading - Investo",
		"User":       user,
		"ActivePage": "trading",
	}
	h.Templates.ExecuteTemplate(w, "trading/basket.html", data)
}

func (h *TradingHandler) CreateBasket(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Name   string               `json:"name"`
		Stocks []service.BasketItem `json:"stocks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	if req.Name == "" || len(req.Stocks) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name and stocks required"})
		return
	}

	id, err := h.BasketSvc.CreateBasket(user.ID, req.Name, req.Stocks)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "Basket created"})
}

func (h *TradingHandler) ExecuteBasket(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		BasketID int64 `json:"basket_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	if err := h.BasketSvc.ExecuteBasket(req.BasketID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Basket executed"})
}

func (h *TradingHandler) ListBaskets(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	baskets, err := h.BasketSvc.ListBaskets(user.ID)
	if err != nil {
		baskets = []service.Basket{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"baskets": baskets})
}

func (h *TradingHandler) OrdersPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":      "Conditional Orders (OCO) - Investo",
		"User":       user,
		"ActivePage": "trading",
	}
	h.Templates.ExecuteTemplate(w, "trading/orders.html", data)
}

func (h *TradingHandler) CreateOCO(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Code       string  `json:"code"`
		Quantity   float64 `json:"quantity"`
		EntryPrice float64 `json:"entry_price"`
		StopLoss   float64 `json:"stop_loss"`
		TakeProfit float64 `json:"take_profit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	if req.Code == "" || req.Quantity <= 0 || req.TakeProfit <= 0 || req.StopLoss <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "code, quantity, stop_loss, take_profit required"})
		return
	}

	id, err := h.BasketSvc.CreateOCO(user.ID, req.Code, req.Quantity, req.EntryPrice, req.StopLoss, req.TakeProfit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "OCO order created"})
}

func (h *TradingHandler) ActiveOCO(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	orders, err := h.BasketSvc.GetActiveOCO(user.ID)
	if err != nil {
		orders = []service.ConditionalOrder{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"orders": orders})
}

func (h *TradingHandler) OCOHistory(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	orders, err := h.BasketSvc.GetOCOHistory(user.ID)
	if err != nil {
		orders = []service.ConditionalOrder{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"orders": orders})
}

func (h *TradingHandler) KellyCalculate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PortfolioValue float64 `json:"portfolio_value"`
		StockPrice     float64 `json:"stock_price"`
		WinRate        float64 `json:"win_rate"`
		AvgWin         float64 `json:"avg_win"`
		AvgLoss        float64 `json:"avg_loss"`
		Capital        float64 `json:"capital"`
		RiskPerTrade   float64 `json:"risk_per_trade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	kelly := service.CalcKelly(req.WinRate, req.AvgWin, req.AvgLoss)
	position, err := service.CalcOptimalPosition(req.PortfolioValue, req.StockPrice, req.WinRate, req.AvgWin, req.AvgLoss)
	riskOfRuin := service.CalcRiskOfRuin(req.WinRate, req.AvgWin, req.AvgLoss, req.Capital, req.RiskPerTrade)

	resp := map[string]interface{}{
		"kelly_pct":       kelly * 100,
		"half_kelly_pct":  kelly * 50,
		"optimal_lots":    position,
		"risk_of_ruin_pct": riskOfRuin * 100,
		"warning":         "",
	}
	if riskOfRuin > 0.05 {
		resp["warning"] = "WARNING: Risk of ruin > 5%. Reduce position size."
	}
	if err != nil {
		resp["optimal_lots"] = 0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

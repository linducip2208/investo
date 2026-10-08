package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"investo/internal/middleware"
	"investo/internal/service"
)

type StrategyHandler struct {
	StrategyBuilder *service.StrategyBuilder
	Templates       *template.Template
}

func (h *StrategyHandler) BuilderPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":      "AI Strategy Builder - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Strategies": service.PrebuiltStrategies,
	}
	h.Templates.ExecuteTemplate(w, "market/strategy-builder.html", data)
}

func (h *StrategyHandler) BacktestJSON(w http.ResponseWriter, r *http.Request) {
	stockCode := r.URL.Query().Get("stock_code")
	strategyName := r.URL.Query().Get("strategy")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if stockCode == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "stock_code is required"})
		return
	}
	if startStr == "" || endStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "start and end dates required"})
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid start date format"})
		return
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid end date format"})
		return
	}

	var strategy service.Strategy
	if strategyName != "" {
		found := false
		for _, s := range service.PrebuiltStrategies {
			if s.Name == strategyName {
				strategy = s
				found = true
				break
			}
		}
		if !found {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "strategy not found"})
			return
		}
	} else {
		var custom struct {
			Name       string                    `json:"name"`
			Conditions []service.StrategyCondition `json:"conditions"`
			StopLoss   float64                   `json:"stop_loss"`
			TakeProfit float64                   `json:"take_profit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&custom); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid custom strategy JSON"})
			return
		}
		strategy = service.Strategy{
			Name:       custom.Name,
			Conditions: custom.Conditions,
			StopLoss:   custom.StopLoss,
			TakeProfit: custom.TakeProfit,
		}
	}

	result, err := h.StrategyBuilder.Backtest(stockCode, strategy, start, end)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

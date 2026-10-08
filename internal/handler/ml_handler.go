package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"investo/internal/middleware"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

func (h *StockHandler) MLPredictPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sectors, _ := h.SectorRepo.FindAll()
	sectorMap := make(map[int64]string)
	for _, s := range sectors {
		sectorMap[s.ID] = s.Name
	}

	data := map[string]interface{}{
		"Title":       "ML Price Prediction: " + stock.Code + " - Investo",
		"User":        safeUser(middleware.GetUser(r)),
		"StockCode":   stock.Code,
		"StockName":   stock.Name,
		"SectorName":  sectorMap[stock.SectorID],
		"Sectors":     sectors,
	}
	h.Templates.ExecuteTemplate(w, "stocks/ml-predict.html", data)
}

func (h *StockHandler) MLPredictJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 30 {
			days = parsed
		}
	}

	predictor := &service.MLPredictor{
		StockPriceRepo: h.StockPriceRepo,
		StockRepo:      h.StockRepo,
	}

	prediction, err := predictor.PredictPrice(code, days)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prediction)
}

package handler

import (
	"encoding/json"
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"
)

type IntelligenceHandler struct {
	RecapService     *service.RecapService
	AnomalyService   *service.AnomalyService
	SentimentService *service.SentimentService
	BreadthService   *service.MarketBreadthService
	Templates        *template.Template
}

func (h *IntelligenceHandler) DailyRecapPage(w http.ResponseWriter, r *http.Request) {
	recap, err := h.RecapService.Generate()
	if err != nil {
		http.Error(w, "Gagal generate market recap", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":  "Rekap Pasar Harian - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Recap":  recap,
	}
	h.Templates.ExecuteTemplate(w, "market/recap.html", data)
}

func (h *IntelligenceHandler) DailyRecapJSON(w http.ResponseWriter, r *http.Request) {
	recap, err := h.RecapService.Generate()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "gagal generate recap"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recap)
}

func (h *IntelligenceHandler) AnomaliesPage(w http.ResponseWriter, r *http.Request) {
	anomalyService := h.AnomalyService

	priceAnomalies, _ := anomalyService.DetectPriceAnomalies()
	gapAnomalies, _ := anomalyService.DetectGapAnomalies()

	allAnomalies := append(priceAnomalies, gapAnomalies...)

	anomalyType := r.URL.Query().Get("type")

	type anomalyView struct {
		StockCode   string
		StockName   string
		Type        string
		Severity    string
		Description string
		Value       float64
		Threshold   float64
	}

	var filtered []anomalyView
	for _, a := range allAnomalies {
		if anomalyType != "" && a.Type != anomalyType {
			continue
		}

		typeLabel := ""
		switch a.Type {
		case "price":
			typeLabel = "Harga"
		case "volume":
			typeLabel = "Volume"
		case "gap":
			typeLabel = "Gap"
		}

		filtered = append(filtered, anomalyView{
			StockCode:   a.StockCode,
			StockName:   a.StockName,
			Type:        typeLabel,
			Severity:    a.Severity,
			Description: a.Description,
			Value:       a.Value,
			Threshold:   a.Threshold,
		})
	}

	data := map[string]interface{}{
		"Title":     "Deteksi Anomali Pasar - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Anomalies": filtered,
		"SelType":   anomalyType,
	}
	h.Templates.ExecuteTemplate(w, "market/anomalies.html", data)
}

func (h *IntelligenceHandler) AnomaliesJSON(w http.ResponseWriter, r *http.Request) {
	priceAnomalies, _ := h.AnomalyService.DetectPriceAnomalies()
	gapAnomalies, _ := h.AnomalyService.DetectGapAnomalies()

	allAnomalies := append(priceAnomalies, gapAnomalies...)
	if allAnomalies == nil {
		allAnomalies = []service.AnomalyResult{}
	}

	anomalyType := r.URL.Query().Get("type")
	var filtered []service.AnomalyResult
	for _, a := range allAnomalies {
		if anomalyType != "" && a.Type != anomalyType {
			continue
		}
		filtered = append(filtered, a)
	}
	if filtered == nil {
		filtered = []service.AnomalyResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func (h *IntelligenceHandler) MarketSentimentJSON(w http.ResponseWriter, r *http.Request) {
	recap, err := h.RecapService.Generate()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "gagal generate sentiment"})
		return
	}

	sentimentScore := 0.0
	if recap.BreadthAdvance > recap.BreadthDecline {
		sentimentScore = recap.BreadthAdvancePct / 100.0
	} else if recap.BreadthDecline > recap.BreadthAdvance {
		sentimentScore = -recap.BreadthDeclinePct / 100.0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sentiment":       recap.MarketSentiment,
		"score":           sentimentScore,
		"breadth_advance": recap.BreadthAdvance,
		"breadth_decline": recap.BreadthDecline,
		"breadth_unchanged": recap.BreadthUnchanged,
		"highlights":      recap.Highlights,
		"date":            recap.Date,
	})
}

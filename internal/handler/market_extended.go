package handler

import (
	"encoding/json"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"
)

func (h *MarketHandler) SectorHealthPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Sector Health Index - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/sector-health.html", data)
}

func (h *MarketHandler) SectorHealthJSON(w http.ResponseWriter, r *http.Request) {
	svc := &service.SectorHealthService{
		StockRepo:            h.StockRepo,
		StockPriceRepo:       h.StockPriceRepo,
		StockFundamentalRepo: h.StockFundamentalRepo,
		SectorRepo:           h.SectorRepo,
	}
	w.Header().Set("Content-Type", "application/json")
	results, err := svc.CalculateAll()
	if err != nil || results == nil {
		results = []service.SectorHealth{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"sectors": results})
}

func (h *MarketHandler) WaranPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Waran Tracker - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/waran.html", data)
}

func (h *MarketHandler) WaranJSON(w http.ResponseWriter, r *http.Request) {
	waranSvc := &service.WaranService{}
	warans := waranSvc.GetAllWaran()
	if warans == nil {
		warans = []service.WaranData{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"warans": warans})
}

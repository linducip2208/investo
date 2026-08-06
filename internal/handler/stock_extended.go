package handler

import (
	"encoding/json"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

func (h *StockHandler) MultiTimeframePage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	prices, _ := h.StockPriceRepo.FindLatest(stock.ID, 1)
	priceFormatted := "Rp 0"
	if len(prices) > 0 && prices[0].Close > 0 {
		priceFormatted = "Rp " + humanizeNum(int64(prices[0].Close))
	}

	data := map[string]interface{}{
		"Title":          stock.Code + " - Multi-Timeframe - Investo",
		"Stock":          stock,
		"LatestPrice":    map[string]interface{}{"PriceFormatted": priceFormatted},
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "stocks/multi-tf.html", data)
}

func (h *APIHandler) SimilarStocks(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	svc := &service.StockSimilarityService{
		StockRepo:            h.StockRepo,
		StockPriceRepo:       h.StockPriceRepo,
		StockFundamentalRepo: h.StockFundamentalRepo,
		SectorRepo:           h.SectorRepo,
	}

	similar, err := svc.FindSimilar(code)
	if err != nil || similar == nil {
		similar = []service.SimilarStock{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"similar": similar, "code": code})
}

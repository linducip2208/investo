package handler

import (
	"encoding/json"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/service"
)

func (h *MarketHandler) GapScannerPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Gap Scanner - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/gap-scanner.html", data)
}

func (h *MarketHandler) GapsJSON(w http.ResponseWriter, r *http.Request) {
	gapScanner := &service.GapScanner{
		StockRepo:      h.StockRepo,
		StockPriceRepo: h.StockPriceRepo,
	}

	gaps, err := gapScanner.ScanGaps()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"gaps":  []service.GapResult{},
		})
		return
	}

	if gaps == nil {
		gaps = []service.GapResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gaps": gaps,
	})
}

func (h *MarketHandler) RSRankingPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Relative Strength Ranking - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/rs-ranking.html", data)
}

func (h *MarketHandler) RSRankingJSON(w http.ResponseWriter, r *http.Request) {
	rsRanking := &service.RSRanking{
		StockRepo:      h.StockRepo,
		StockPriceRepo: h.StockPriceRepo,
		SectorRepo:     h.SectorRepo,
	}

	results, err := rsRanking.CalcRSRanking()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   err.Error(),
			"results": []service.RSResult{},
		})
		return
	}

	if results == nil {
		results = []service.RSResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

func (h *MarketHandler) NewHighLowPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "New High / New Low - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/new-high-low.html", data)
}

func (h *MarketHandler) NewHighsJSON(w http.ResponseWriter, r *http.Request) {
	rsRanking := &service.RSRanking{
		StockRepo:      h.StockRepo,
		StockPriceRepo: h.StockPriceRepo,
		SectorRepo:     h.SectorRepo,
	}

	highs, err := rsRanking.ScanNewHighs()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"highs":  []service.HLResult{},
			"lows":   []service.HLResult{},
		})
		return
	}

	lows, err := rsRanking.ScanNewLows()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"highs": highs,
			"lows":  []service.HLResult{},
		})
		return
	}

	if highs == nil {
		highs = []service.HLResult{}
	}
	if lows == nil {
		lows = []service.HLResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"highs": highs,
		"lows":  lows,
	})
}

func (h *MarketHandler) NewLowsJSON(w http.ResponseWriter, r *http.Request) {
	rsRanking := &service.RSRanking{
		StockRepo:      h.StockRepo,
		StockPriceRepo: h.StockPriceRepo,
		SectorRepo:     h.SectorRepo,
	}

	lows, err := rsRanking.ScanNewLows()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"lows":  []service.HLResult{},
		})
		return
	}

	if lows == nil {
		lows = []service.HLResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lows": lows,
	})
}

func (h *MarketHandler) PreMarketPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Pre-Market Movers - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/pre-market.html", data)
}

func (h *MarketHandler) PreMarketJSON(w http.ResponseWriter, r *http.Request) {
	gapScanner := &service.GapScanner{
		StockRepo:      h.StockRepo,
		StockPriceRepo: h.StockPriceRepo,
	}

	movers, err := gapScanner.ScanPreMarketMovers()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"movers": []service.PreMarketMover{},
		})
		return
	}

	if movers == nil {
		movers = []service.PreMarketMover{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"movers": movers,
	})
}

func (h *MarketHandler) BlockTradesPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Block Trade Scanner - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "market/block-trades.html", data)
}

func (h *MarketHandler) BlockTradesJSON(w http.ResponseWriter, r *http.Request) {
	minValue := 100_000_000.0
	if v := r.URL.Query().Get("min_value"); v != "" {
		if parsed := atoi(v); parsed > 0 {
			minValue = float64(parsed)
		}
	}

	gapScanner := &service.GapScanner{
		StockRepo:      h.StockRepo,
		StockPriceRepo: h.StockPriceRepo,
	}

	trades, err := gapScanner.ScanBlockTrades(minValue)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"trades": []service.BlockTrade{},
		})
		return
	}

	if trades == nil {
		trades = []service.BlockTrade{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trades": trades,
	})
}

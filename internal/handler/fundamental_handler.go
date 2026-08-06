package handler

import (
	"encoding/json"
	"html/template"
	"net/http"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type FundamentalHandler struct {
	ValuationService      *service.ValuationService
	PeerComparisonService *service.ValuationService
	DividendService       *service.DividendService
	StockRepo             *repository.StockRepository
	StockFundamentalRepo  *repository.StockFundamentalRepository
	StockPriceRepo        *repository.StockPriceRepository
	SectorRepo            *repository.SectorRepository
	Templates             *template.Template
}

func (h *FundamentalHandler) ValuationPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	valuation, err := h.ValuationService.Calculate(stock.ID)
	if err != nil {
		valuation = &service.ValuationResult{
			Recommendation: "HOLD",
		}
	}

	peerComp, _ := h.PeerComparisonService.ComparePeers(stock.ID)

	fundamentals, _ := h.StockFundamentalRepo.FindByStockID(stock.ID, 10)

	dividends, _ := h.DividendService.GetStockDividends(stock.ID)

	sector, _ := h.SectorRepo.FindByID(stock.SectorID)

	data := map[string]interface{}{
		"Title":        stock.Code + " - Valuasi - Investo",
		"Stock":        stock,
		"Valuation":    valuation,
		"PeerComp":     peerComp,
		"Fundamentals": fundamentals,
		"Dividends":    dividends,
		"Sector":       sector,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "stocks/valuation.html", data)
}

func (h *FundamentalHandler) PeerComparisonJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	comp, err := h.PeerComparisonService.ComparePeers(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comp)
}

func (h *FundamentalHandler) ValuationJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	valuation, err := h.ValuationService.Calculate(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(valuation)
}

func (h *FundamentalHandler) DividendCalendarPage(w http.ResponseWriter, r *http.Request) {
	calendar, err := h.DividendService.GetCalendar()
	if err != nil {
		calendar = &service.DividendCalendar{}
	}

	stocks, _ := h.StockRepo.ListActive()

	type yieldLeader struct {
		Code  string
		Name  string
		Yield float64
		Price float64
	}

	var yieldLeaders []yieldLeader
	for _, st := range stocks {
		fund, err := h.StockFundamentalRepo.FindLatest(st.ID)
		if err != nil || fund.DividendYield <= 0 {
			continue
		}
		prices, err := h.StockPriceRepo.FindLatest(st.ID, 1)
		price := 0.0
		if err == nil && len(prices) > 0 {
			price = prices[0].Close
		}

		yieldLeaders = append(yieldLeaders, yieldLeader{
			Code:  st.Code,
			Name:  st.Name,
			Yield: fund.DividendYield,
			Price: price,
		})
	}

	for i := 0; i < len(yieldLeaders); i++ {
		for j := i + 1; j < len(yieldLeaders); j++ {
			if yieldLeaders[j].Yield > yieldLeaders[i].Yield {
				yieldLeaders[i], yieldLeaders[j] = yieldLeaders[j], yieldLeaders[i]
			}
		}
	}

	if len(yieldLeaders) > 10 {
		yieldLeaders = yieldLeaders[:10]
	}

	data := map[string]interface{}{
		"Title":        "Kalender Dividen - Investo",
		"Calendar":     calendar,
		"YieldLeaders": yieldLeaders,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "dividen/calendar.html", data)
}

func (h *FundamentalHandler) DividendJSON(w http.ResponseWriter, r *http.Request) {
	calendar, err := h.DividendService.GetCalendar()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

func (h *FundamentalHandler) StockDividendJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	dividends, err := h.DividendService.GetStockDividends(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dividends)
}

func (h *FundamentalHandler) DuPontJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	result, err := h.ValuationService.CalcDuPont(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *FundamentalHandler) PiotroskiJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	result, err := h.ValuationService.CalcPiotroski(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *FundamentalHandler) AltmanJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	result, err := h.ValuationService.CalcAltmanZ(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *FundamentalHandler) BeneishJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	result, err := h.ValuationService.CalcBeneish(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *FundamentalHandler) HealthScoreJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	result, err := h.ValuationService.CalcHealthScore(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

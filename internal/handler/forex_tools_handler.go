package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type ForexToolsHandler struct {
	Templates       *template.Template
	ForexTools      *service.ForexTools
	MarketProfile   *service.MarketProfileService
	COTData         *service.COTDataService
	StockPriceRepo  interface {
		FindLatestByStockID(stockID int64) (*model.StockPrice, error)
		FindByStockIDRange(stockID int64, start, end interface{}) ([]model.StockPrice, error)
	}
}

func (h *ForexToolsHandler) SwapCalcPage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))
	ft := service.NewForexTools()
	rates := ft.GetAllSwapRates(100000)

	currencies := []string{"USD", "EUR", "JPY", "GBP", "AUD", "NZD", "CAD", "CHF", "IDR", "SGD"}

	h.Templates.ExecuteTemplate(w, "forex/swap-calc.html", map[string]interface{}{
		"Title":      "Swap/Rollover Calculator - Investo",
		"User":       user,
		"Rates":      rates,
		"Currencies": currencies,
	})
}

func (h *ForexToolsHandler) SwapCalcJSON(w http.ResponseWriter, r *http.Request) {
	baseCurrency := strings.ToUpper(r.FormValue("base_currency"))
	quoteCurrency := strings.ToUpper(r.FormValue("quote_currency"))
	positionSize, _ := strconv.ParseFloat(r.FormValue("position_size"), 64)
	direction := strings.ToUpper(r.FormValue("direction"))

	if baseCurrency == "" || quoteCurrency == "" || positionSize <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "parameter tidak lengkap"})
		return
	}

	result, err := h.ForexTools.CalcSwap(baseCurrency, quoteCurrency, positionSize, direction)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ForexToolsHandler) MarginCalcPage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))

	pairs := []string{
		"EUR/USD", "USD/JPY", "GBP/USD", "USD/CHF", "AUD/USD",
		"NZD/USD", "USD/CAD", "EUR/JPY", "GBP/JPY", "EUR/GBP",
		"USD/IDR", "EUR/IDR", "SGD/IDR",
	}

	leverages := []int{1, 10, 20, 50, 100}

	h.Templates.ExecuteTemplate(w, "forex/margin-calc.html", map[string]interface{}{
		"Title":      "Margin Calculator - Investo",
		"User":       user,
		"Pairs":      pairs,
		"Leverages":  leverages,
	})
}

func (h *ForexToolsHandler) MarginCalcJSON(w http.ResponseWriter, r *http.Request) {
	pair := strings.ToUpper(r.FormValue("pair"))
	positionSize, _ := strconv.ParseFloat(r.FormValue("position_size"), 64)
	leverage, _ := strconv.Atoi(r.FormValue("leverage"))

	if pair == "" || positionSize <= 0 || leverage <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "parameter tidak lengkap"})
		return
	}

	result, err := h.ForexTools.CalcMargin(pair, positionSize, leverage)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ForexToolsHandler) PipCalcPage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))
	ft := service.NewForexTools()
	refs := ft.GetAllPipReferences(1.0, "USD")

	accountCurrencies := []string{"USD", "EUR", "IDR", "SGD"}

	h.Templates.ExecuteTemplate(w, "forex/pip-calc.html", map[string]interface{}{
		"Title":             "Pip Value Calculator - Investo",
		"User":              user,
		"Refs":              refs,
		"AccountCurrencies": accountCurrencies,
	})
}

func (h *ForexToolsHandler) PipValueJSON(w http.ResponseWriter, r *http.Request) {
	pair := strings.ToUpper(r.URL.Query().Get("pair"))
	lotSize, _ := strconv.ParseFloat(r.URL.Query().Get("lot_size"), 64)
	accountCurrency := strings.ToUpper(r.URL.Query().Get("account_currency"))

	if pair == "" || lotSize <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "parameter tidak lengkap"})
		return
	}
	if accountCurrency == "" {
		accountCurrency = "USD"
	}

	pv, err := h.ForexTools.CalcPipValue(pair, lotSize, accountCurrency)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pair":             pair,
		"lot_size":         lotSize,
		"account_currency": accountCurrency,
		"pip_value":        pv,
		"value_10_pip":     pv * 10,
		"value_100_pip":    pv * 100,
	})
}

func (h *ForexToolsHandler) VolumeProfilePage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))

	h.Templates.ExecuteTemplate(w, "forex/volume-profile.html", map[string]interface{}{
		"Title": "Volume Profile - Investo",
		"User":  user,
	})
}

func (h *ForexToolsHandler) COTPage(w http.ResponseWriter, r *http.Request) {
	user := safeUser(middleware.GetUser(r))
	reports := h.COTData.GetAllCOTData()

	h.Templates.ExecuteTemplate(w, "forex/cot.html", map[string]interface{}{
		"Title":   "COT Data (Commitment of Traders) - Investo",
		"User":    user,
		"Reports": reports,
	})
}

func (h *ForexToolsHandler) COTJSON(w http.ResponseWriter, r *http.Request) {
	pair := chi.URLParam(r, "pair")

	report, err := h.COTData.GetCOTData(pair)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type ForexHandler struct {
	ForexRepo      *repository.ForexRepository
	Templates      *template.Template
	ForexAnalytics *service.ForexAnalytics
}

func (h *ForexHandler) List(w http.ResponseWriter, r *http.Request) {
	pairs, err := h.ForexRepo.FindAllPairs()
	if err != nil {
		http.Error(w, "Gagal memuat data forex", http.StatusInternalServerError)
		return
	}

	latestRates, _ := h.ForexRepo.FindLatestRates()
	rateMap := make(map[int64]float64)
	for _, rate := range latestRates {
		rateMap[rate.PairID] = rate.Close
	}

	type PairRow struct {
		Base           string
		Quote          string
		BaseName       string
		QuoteName      string
		Name           string
		Group          string
		Rate           float64
		RateFormatted  string
		ChangePercent  float64
		Sparkline      string
	}

	var results []PairRow
	for _, p := range pairs {
		rate := rateMap[p.ID]
		baseName := p.BaseCurrency
		quoteName := p.QuoteCurrency
		parts := strings.SplitN(p.Name, " / ", 2)
		if len(parts) == 2 {
			baseName = parts[0]
			quoteName = parts[1]
		}
		results = append(results, PairRow{
			Base:          p.BaseCurrency,
			Quote:         p.QuoteCurrency,
			BaseName:      baseName,
			QuoteName:     quoteName,
			Name:          p.Name,
			Group:         p.Group,
			Rate:          rate,
			RateFormatted: fmt.Sprintf("%.4f", rate),
			ChangePercent: 0,
		})
	}

	groups := []map[string]string{
		{"Slug": "major", "Name": "Major"},
		{"Slug": "exotic", "Name": "Exotic"},
	}

	user := middleware.GetUser(r)
	if user == nil {
		user = &model.User{Name: "Guest", Email: "", Role: "visitor"}
	}

	data := map[string]interface{}{
		"Title":          "Forex - Investo",
		"Pairs":          results,
		"Groups":         groups,
		"User":           user,
		"TotalStocks":    0,
		"PortfolioCount": 0,
		"WatchlistCount": 0,
	}
	if err := h.Templates.ExecuteTemplate(w, "forex/list.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *ForexHandler) Detail(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")

	pair, err := h.ForexRepo.FindPairByCurrencies(base, quote)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rates, _ := h.ForexRepo.FindRates(pair.ID, time.Now().AddDate(0, -3, 0), time.Now())

	var latestRate, highRate, lowRate float64
	if len(rates) > 0 {
		latestRate = rates[len(rates)-1].Close
		highRate = rates[0].High
		lowRate = rates[0].Low
		for _, r := range rates {
			if r.High > highRate { highRate = r.High }
			if r.Low < lowRate { lowRate = r.Low }
		}
	}

	baseName := pair.BaseCurrency
	quoteName := pair.QuoteCurrency
	parts := strings.SplitN(pair.Name, " / ", 2)
	if len(parts) == 2 {
		baseName = parts[0]
		quoteName = parts[1]
	}

	prevRate := 0.0
	changePct := 0.0
	if len(rates) > 1 {
		prevRate = rates[len(rates)-2].Close
		if prevRate > 0 {
			changePct = ((latestRate - prevRate) / prevRate) * 100
		}
	}

	type pairView struct {
		Base      string
		Quote     string
		BaseName  string
		QuoteName string
		Name      string
		Group     string
	}
	type priceView struct {
		RateFormatted   string
		ChangePercent   float64
		ChangeFormatted string
		HighFormatted   string
		LowFormatted    string
	}

	data := map[string]interface{}{
		"Title": base + "/" + quote + " - Forex - Investo",
		"Pair": pairView{Base: pair.BaseCurrency, Quote: pair.QuoteCurrency, BaseName: baseName, QuoteName: quoteName, Name: pair.Name, Group: pair.Group},
		"Rates":      rates,
		"LatestPrice": priceView{RateFormatted: fmt.Sprintf("%.4f", latestRate), ChangePercent: changePct, ChangeFormatted: fmt.Sprintf("%.4f", latestRate-prevRate), HighFormatted: fmt.Sprintf("%.4f", highRate), LowFormatted: fmt.Sprintf("%.4f", lowRate)},
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "forex/detail.html", data)
}

func (h *ForexHandler) ChartData(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")

	pair, err := h.ForexRepo.FindPairByCurrencies(base, quote)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "pair not found"})
		return
	}

	q := r.URL.Query()
	startStr := q.Get("start")
	endStr := q.Get("end")

	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()

	if startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			start = parsed
		}
	}
	if endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			end = parsed
		}
	}

	rates, err := h.ForexRepo.FindRates(pair.ID, start, end)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to get rates"})
		return
	}

	type ChartPoint struct {
		Date string  `json:"date"`
		Open float64 `json:"open"`
		High float64 `json:"high"`
		Low  float64 `json:"low"`
		Close float64 `json:"close"`
	}

	var data []ChartPoint
	for _, rate := range rates {
		data = append(data, ChartPoint{
			Date:  rate.Date.Format("2006-01-02"),
			Open:  rate.Open,
			High:  rate.High,
			Low:   rate.Low,
			Close: rate.Close,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *ForexHandler) ProTerminal(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	strength, _ := h.ForexAnalytics.CalcStrengthMeter()
	arbitrage, _ := h.ForexAnalytics.FindArbitrage()

	pairs, err := h.ForexRepo.FindAllPairs()
	if err != nil {
		pairs = nil
	}

	latestRates, _ := h.ForexRepo.FindLatestRates()
	rateMap := make(map[string]float64)
	pairIDMap := make(map[string]int64)
	for _, rate := range latestRates {
		pair, err := h.ForexRepo.FindPairByID(rate.PairID)
		if err != nil {
			continue
		}
		key := pair.BaseCurrency + "/" + pair.QuoteCurrency
		rateMap[key] = rate.Close
		pairIDMap[key] = rate.PairID
	}

	currencies := []string{"USD", "EUR", "JPY", "GBP", "AUD", "NZD", "CAD", "CHF", "IDR", "SGD"}

	type rateGrid struct {
		Base    string
		Rates   []float64
		Changes []float64
	}

	var grid []rateGrid
	for _, base := range currencies {
		var rates []float64
		var changes []float64
		for _, quote := range currencies {
			if base == quote {
				rates = append(rates, 1.0)
				changes = append(changes, 0)
			} else {
				r := rateMap[base+"/"+quote]
				if r <= 0 {
					inv := rateMap[quote+"/"+base]
					if inv > 0 {
						r = 1.0 / inv
					}
				}
				rates = append(rates, r)
				changes = append(changes, 0)
			}
		}
		grid = append(grid, rateGrid{Base: base, Rates: rates, Changes: changes})
	}

	type carryResult struct {
		Pair      string
		Direction string
		Spread    float64
		Return    float64
	}

	var carryTrades []carryResult
	checked := make(map[string]bool)
	for _, c1 := range currencies {
		for _, c2 := range currencies {
			if c1 == c2 {
				continue
			}
			key := c1 + "/" + c2
			if checked[key] || checked[c2+"/"+c1] {
				continue
			}
			checked[key] = true
			if ct, err := h.ForexAnalytics.CalcCarryTrade(c1, c2); err == nil {
				carryTrades = append(carryTrades, carryResult{
					Pair:      ct.Pair,
					Direction: ct.Direction,
					Spread:    ct.Spread,
					Return:    ct.AnnualReturn,
				})
			}
		}
	}

	for i := 0; i < len(carryTrades); i++ {
		for j := i + 1; j < len(carryTrades); j++ {
			if carryTrades[i].Return < carryTrades[j].Return {
				carryTrades[i], carryTrades[j] = carryTrades[j], carryTrades[i]
			}
		}
	}

	type PairInfo struct {
		ID        int64
		Name      string
		Base      string
		Quote     string
		Rate      float64
	}

	var pairList []PairInfo
	for _, p := range pairs {
		key := p.BaseCurrency + "/" + p.QuoteCurrency
		pairList = append(pairList, PairInfo{
			ID:    p.ID,
			Name:  p.Name,
			Base:  p.BaseCurrency,
			Quote: p.QuoteCurrency,
			Rate:  rateMap[key],
		})
	}

	data := map[string]interface{}{
		"Title":       "Forex Pro Terminal - Investo",
		"Strength":    strength,
		"Arbitrage":   arbitrage,
		"Currencies":  currencies,
		"Grid":        grid,
		"CarryTrades": carryTrades,
		"Pairs":       pairList,
		"PairIDs":     pairIDMap,
		"User":        user,
	}
	h.Templates.ExecuteTemplate(w, "forex/pro.html", data)
}

func (h *ForexHandler) StrengthJSON(w http.ResponseWriter, r *http.Request) {
	strength, err := h.ForexAnalytics.CalcStrengthMeter()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strength)
}

func (h *ForexHandler) CarryTradeJSON(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")

	ct, err := h.ForexAnalytics.CalcCarryTrade(base, quote)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ct)
}

func (h *ForexHandler) CorrelationJSON(w http.ResponseWriter, r *http.Request) {
	p1Str := chi.URLParam(r, "p1")
	p2Str := chi.URLParam(r, "p2")

	p1, err := strconv.ParseInt(p1Str, 10, 64)
	if err != nil {
		http.Error(w, "Invalid pair ID", http.StatusBadRequest)
		return
	}
	p2, err := strconv.ParseInt(p2Str, 10, 64)
	if err != nil {
		http.Error(w, "Invalid pair ID", http.StatusBadRequest)
		return
	}

	corr, err := h.ForexAnalytics.CalcCorrelation(p1, p2)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{"correlation": corr})
}

func (h *ForexHandler) ArbitrageJSON(w http.ResponseWriter, r *http.Request) {
	arb, err := h.ForexAnalytics.FindArbitrage()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arb)
}

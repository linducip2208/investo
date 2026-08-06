package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"time"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"
	"investo/internal/service/indicator"
	"investo/internal/service/pattern"

	"github.com/go-chi/chi/v5"
)

type AnalysisHandler struct {
	StockRepo             *repository.StockRepository
	StockPriceRepo        *repository.StockPriceRepository
	PatternService        *pattern.PatternService
	ChartService          *service.ChartService
	PatternScanner        *service.PatternScannerService
	PatternAnalytics      *service.PatternAnalyticsService
	AnnotationService     *service.AnnotationService
	Templates             *template.Template
	CustomIndicatorSvc    *service.CustomIndicatorService
	CustomIndicatorRepo   *service.CustomIndicatorRepo
	FeatureFlagService    *service.FeatureFlagService
}

func (h *AnalysisHandler) TechnicalPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	latestPrices, _ := h.StockPriceRepo.FindLatest(stock.ID, 1)
	var latestPrice float64
	if len(latestPrices) > 0 {
		latestPrice = latestPrices[0].Close
	}

	data := map[string]interface{}{
		"Title":       stock.Code + " - Analisis Teknikal - Investo",
		"Stock":       stock,
		"LatestPrice": latestPrice,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "stocks/teknikal.html", data)
}

func (h *AnalysisHandler) PatternsJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	results := h.PatternService.DetectAll(prices)
	if results == nil {
		results = []pattern.PatternResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *AnalysisHandler) SRLevelsJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"supports": []interface{}{}, "resistances": []interface{}{}})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	supports, resistances := h.PatternService.FindSupportResistance(prices, 20)
	if supports == nil {
		supports = []pattern.SRLevel{}
	}
	if resistances == nil {
		resistances = []pattern.SRLevel{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"supports":     supports,
		"resistances":  resistances,
	})
}

func (h *AnalysisHandler) VolumeProfileJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	vp := h.PatternService.CalcVolumeProfile(prices)
	if vp == nil {
		vp = []pattern.VolumeProfile{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vp)
}

type IndicatorsResponse struct {
	Dates      []string        `json:"dates"`
	SMA        map[int][]interface{} `json:"sma"`
	EMA        map[int][]interface{} `json:"ema"`
	RSI        map[int][]interface{} `json:"rsi"`
	MACD       *IndicatorMACDData   `json:"macd"`
	BB         *IndicatorBBData      `json:"bb"`
	Stochastic *IndicatorStochData  `json:"stochastic"`
	ATR        map[int][]interface{} `json:"atr"`
}

type IndicatorMACDData struct {
	MACDLine   []interface{} `json:"macd_line"`
	SignalLine []interface{} `json:"signal_line"`
	Histogram  []interface{} `json:"histogram"`
}

type IndicatorBBData struct {
	Upper  []interface{} `json:"upper"`
	Middle []interface{} `json:"middle"`
	Lower  []interface{} `json:"lower"`
}

type IndicatorStochData struct {
	K []interface{} `json:"k"`
	D []interface{} `json:"d"`
}

func f64ToInterface(arr []float64) []interface{} {
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		if math.IsNaN(v) {
			result[i] = nil
		} else {
			result[i] = v
		}
	}
	return result
}

func (h *AnalysisHandler) IndicatorsJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	closePrices := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		closePrices[i] = p.Close
		highs[i] = p.High
		lows[i] = p.Low
	}

	resp := IndicatorsResponse{
		Dates: dates,
		SMA: map[int][]interface{}{
			20:  f64ToInterface(indicator.CalcSMA(closePrices, 20)),
			50:  f64ToInterface(indicator.CalcSMA(closePrices, 50)),
			200: f64ToInterface(indicator.CalcSMA(closePrices, 200)),
		},
		EMA: map[int][]interface{}{
			12: f64ToInterface(indicator.CalcEMA(closePrices, 12)),
			26: f64ToInterface(indicator.CalcEMA(closePrices, 26)),
		},
		RSI: map[int][]interface{}{
			14: f64ToInterface(indicator.CalcRSI(closePrices, 14)),
		},
	}

	macdLine, signalLine, hist := indicator.CalcMACD(closePrices, 12, 26, 9)
	resp.MACD = &IndicatorMACDData{
		MACDLine:   f64ToInterface(macdLine),
		SignalLine: f64ToInterface(signalLine),
		Histogram:  f64ToInterface(hist),
	}

	upper, middle, lower := indicator.CalcBollingerBands(closePrices, 20, 2.0)
	resp.BB = &IndicatorBBData{
		Upper:  f64ToInterface(upper),
		Middle: f64ToInterface(middle),
		Lower:  f64ToInterface(lower),
	}

	stochK, stochD := indicator.CalcStochastic(highs, lows, closePrices, 14, 3)
	resp.Stochastic = &IndicatorStochData{
		K: f64ToInterface(stochK),
		D: f64ToInterface(stochD),
	}

	resp.ATR = map[int][]interface{}{
		14: f64ToInterface(indicator.CalcATR(highs, lows, closePrices, 14)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AnalysisHandler) ChartPro(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title": stock.Code + " - Chart Pro - Investo",
		"Stock": stock,
		"User":  safeUser(middleware.GetUser(r)),
	}

	if h.Templates != nil {
		if err := h.Templates.ExecuteTemplate(w, "stocks/chart-pro.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	}
}

func (h *AnalysisHandler) FibonacciJSON(w http.ResponseWriter, r *http.Request) {
	highStr := r.URL.Query().Get("high")
	lowStr := r.URL.Query().Get("low")

	high, err := strconv.ParseFloat(highStr, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid high value"}`, http.StatusBadRequest)
		return
	}
	low, err := strconv.ParseFloat(lowStr, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid low value"}`, http.StatusBadRequest)
		return
	}

	levels := service.CalcRetracement(high, low)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"retracement": levels,
	})
}

func (h *AnalysisHandler) PatternReliabilityJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	rel, err := h.PatternAnalytics.AnalyzeReliability(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if rel == nil {
		rel = []service.PatternReliability{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rel)
}

func (h *AnalysisHandler) MarketPatternScan(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "1D"
	}

	scans, err := h.PatternScanner.ScanMarket(timeframe)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if scans == nil {
		scans = []service.TimeframeScan{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func (h *AnalysisHandler) AnnotationsJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	annotations := h.AnnotationService.GenerateAutoAnnotations(code, prices)

	divAnnotations := h.AnnotationService.GetDividendAnnotations(stock.ID)
	annotations = append(annotations, divAnnotations...)

	if annotations == nil {
		annotations = []service.ChartAnnotation{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(annotations)
}

func (h *AnalysisHandler) MultiTimeframeJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	scans, err := h.PatternScanner.ScanAllTimeframes(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if scans == nil {
		scans = []service.TimeframeScan{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func (h *AnalysisHandler) MarketPatternScanPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Pattern Scanner Pasar - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}

	if h.Templates != nil {
		if err := h.Templates.ExecuteTemplate(w, "market/pattern-scan.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	}
}

func (h *AnalysisHandler) CustomIndicatorPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user := middleware.GetUser(r)
	if h.FeatureFlagService != nil && user != nil {
		if !h.FeatureFlagService.CanAccess(user.ID, "custom_indicators") {
			http.Error(w, "Fitur ini memerlukan langganan Pro atau Whitelabel", http.StatusForbidden)
			return
		}
	}

	data := map[string]interface{}{
		"Title": stock.Code + " - Custom Indicator - Investo",
		"Stock": stock,
		"User":  safeUser(user),
	}

	if h.Templates != nil {
		if err := h.Templates.ExecuteTemplate(w, "stocks/custom-indicator.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	}
}

func (h *AnalysisHandler) CustomIndicatorJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	formula := r.URL.Query().Get("formula")
	w.Header().Set("Content-Type", "application/json")

	if formula == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "formula parameter required"})
		return
	}

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			json.NewEncoder(w).Encode(map[string]string{"error": "no price data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	closePrices := make([]float64, len(prices))
	for i, p := range prices {
		closePrices[i] = p.Close
	}

	result, err := h.CustomIndicatorSvc.BuildIndicator(formula, closePrices)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if result == nil {
		result = []float64{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":    service.ToInterfaceSlice(result),
		"formula": formula,
	})
}

func (h *AnalysisHandler) CustomIndicatorSaveJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	w.Header().Set("Content-Type", "application/json")

	user := middleware.GetUser(r)
	if user == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "login required"})
		return
	}

	var payload struct {
		Name    string `json:"name"`
		Formula string `json:"formula"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	indicator := &service.CustomIndicatorSetting{
		UserID:    user.ID,
		StockCode: code,
		Name:      payload.Name,
		Formula:   payload.Formula,
	}

	id, err := h.CustomIndicatorRepo.Save(indicator)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "saved",
		"id":     id,
	})
}

func (h *AnalysisHandler) CustomIndicatorSavedJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	w.Header().Set("Content-Type", "application/json")

	user := middleware.GetUser(r)
	if user == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	items, err := h.CustomIndicatorRepo.ListByUserAndStock(user.ID, code)
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	json.NewEncoder(w).Encode(items)
}

func (h *AnalysisHandler) CustomIndicatorDeleteJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user := middleware.GetUser(r)
	if user == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "login required"})
		return
	}

	var payload struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.CustomIndicatorRepo.Delete(payload.ID, user.ID); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *AnalysisHandler) PatternConfidenceJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	w.Header().Set("Content-Type", "application/json")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	results := h.PatternService.DetectAll(prices)
	if results == nil {
		results = []pattern.PatternResult{}
	}

	type ConfidenceEntry struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Reliability int    `json:"reliability"`
		Confidence  int    `json:"confidence"`
		Label       string `json:"label"`
		Color       string `json:"color"`
	}

	var entries []ConfidenceEntry
	for _, r := range results {
		color := "#ef4444"
		switch {
		case r.Confidence >= 70:
			color = "#10b981"
		case r.Confidence >= 40:
			color = "#f59e0b"
		}
		entries = append(entries, ConfidenceEntry{
			Name:        r.Name,
			Type:        r.Type,
			Reliability: r.Reliability,
			Confidence:  r.Confidence,
			Label:       fmt.Sprintf("%d%%", r.Confidence),
			Color:       color,
		})
	}

	json.NewEncoder(w).Encode(entries)
}

func (h *AnalysisHandler) IchimokuJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		highs[i] = p.High
		lows[i] = p.Low
		closes[i] = p.Close
	}

	tenkan, kijun, senkouA, senkouB, chikou := indicator.CalcIchimoku(highs, lows, closes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates":    dates,
		"tenkan":   f64ToInterface(tenkan),
		"kijun":    f64ToInterface(kijun),
		"senkou_a": f64ToInterface(senkouA),
		"senkou_b": f64ToInterface(senkouB),
		"chikou":   f64ToInterface(chikou),
	})
}

func (h *AnalysisHandler) SARJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	highs := make([]float64, n)
	lows := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		highs[i] = p.High
		lows[i] = p.Low
	}

	sar := indicator.CalcParabolicSAR(highs, lows, 0.02, 0.20)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates": dates,
		"sar":   f64ToInterface(sar),
	})
}

func (h *AnalysisHandler) ADXJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		highs[i] = p.High
		lows[i] = p.Low
		closes[i] = p.Close
	}

	plusDI, minusDI, adx := indicator.CalcADX(highs, lows, closes, 14)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates":    dates,
		"plus_di":  f64ToInterface(plusDI),
		"minus_di": f64ToInterface(minusDI),
		"adx":      f64ToInterface(adx),
	})
}

func (h *AnalysisHandler) VWAPJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		highs[i] = p.High
		lows[i] = p.Low
		closes[i] = p.Close
		volumes[i] = float64(p.Volume)
	}

	vwap := indicator.CalcVWAP(highs, lows, closes, volumes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates": dates,
		"vwap":  f64ToInterface(vwap),
	})
}

func (h *AnalysisHandler) OBVJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		closes[i] = p.Close
		volumes[i] = float64(p.Volume)
	}

	obv := indicator.CalcOBV(closes, volumes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates": dates,
		"obv":   f64ToInterface(obv),
	})
}

func (h *AnalysisHandler) CMFJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		highs[i] = p.High
		lows[i] = p.Low
		closes[i] = p.Close
		volumes[i] = float64(p.Volume)
	}

	cmf := indicator.CalcCMF(highs, lows, closes, volumes, 20)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates": dates,
		"cmf":   f64ToInterface(cmf),
	})
}

func (h *AnalysisHandler) HeikinAshiJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	opens := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		opens[i] = p.Open
		highs[i] = p.High
		lows[i] = p.Low
		closes[i] = p.Close
	}

	haOpen, haHigh, haLow, haClose := indicator.CalcHeikinAshi(opens, highs, lows, closes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates":      dates,
		"ha_open":    f64ToInterface(haOpen),
		"ha_high":    f64ToInterface(haHigh),
		"ha_low":     f64ToInterface(haLow),
		"ha_close":   f64ToInterface(haClose),
	})
}

func (h *AnalysisHandler) KeltnerJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "no data"})
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	n := len(prices)
	dates := make([]string, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		highs[i] = p.High
		lows[i] = p.Low
		closes[i] = p.Close
	}

	middle, upper, lower := indicator.CalcKeltner(highs, lows, closes, 20, 2.0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates":  dates,
		"upper":  f64ToInterface(upper),
		"middle": f64ToInterface(middle),
		"lower":  f64ToInterface(lower),
	})
}

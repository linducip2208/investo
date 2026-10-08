package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"
	"investo/internal/service/indicator"

	"github.com/go-chi/chi/v5"
)

type MarketHandler struct {
	HeatmapService        *service.HeatmapService
	BreadthService        *service.MarketBreadthService
	StockPriceRepo        *repository.StockPriceRepository
	StockRepo             *repository.StockRepository
	StockFundamentalRepo  *repository.StockFundamentalRepository
	SectorRepo            *repository.SectorRepository
	ForexRepo             *repository.ForexRepository
	SignalGenerator       *service.SignalGeneratorService
	TrendScanner          *service.TrendScannerService
	EconCalendarService   *service.EconomicCalendarService
	Templates             *template.Template
}

func (h *MarketHandler) HeatmapPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Market Heatmap - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/heatmap.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

type heatmapResponse struct {
	Sectors []heatmapSector  `json:"sectors"`
	Breadth *breadthBrief    `json:"breadth"`
}

type heatmapSector struct {
	Name   string                `json:"name"`
	Stocks []service.HeatmapCell `json:"stocks"`
}

type breadthBrief struct {
	Advance int `json:"advance"`
	Decline int `json:"decline"`
	Unchanged int `json:"unchanged"`
}

func (h *MarketHandler) HeatmapJSON(w http.ResponseWriter, r *http.Request) {
	_, grouped, err := h.HeatmapService.GetHeatmap()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load heatmap"})
		return
	}

	var sectors []heatmapSector
	for name, stocks := range grouped {
		sectors = append(sectors, heatmapSector{Name: name, Stocks: stocks})
	}

	mb, _ := h.BreadthService.Calculate()
	var bb *breadthBrief
	if mb != nil {
		bb = &breadthBrief{
			Advance:   mb.Advance,
			Decline:   mb.Decline,
			Unchanged: mb.Unchanged,
		}
	}

	resp := heatmapResponse{
		Sectors: sectors,
		Breadth: bb,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *MarketHandler) BreadthJSON(w http.ResponseWriter, r *http.Request) {
	mb, err := h.BreadthService.Calculate()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to calculate market breadth"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mb)
}

type tickerStock struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"change_percent"`
}

type marketSummaryResponse struct {
	Breadth     *service.MarketBreadth `json:"breadth"`
	TotalStocks int                    `json:"total_stocks"`
	Ticker      []tickerStock          `json:"ticker"`
	TopGainers  []tickerStock          `json:"top_gainers"`
	TopLosers   []tickerStock          `json:"top_losers"`
	Timestamp   string                 `json:"timestamp"`
}

func (h *MarketHandler) MarketSummary(w http.ResponseWriter, r *http.Request) {
	mb, err := h.BreadthService.Calculate()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load market summary"})
		return
	}

	stocks, _ := h.HeatmapService.StockRepo.ListActive()

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}
	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	type stockWithChange struct {
		Code          string
		Name          string
		Price         float64
		ChangePercent float64
		Volume        int64
	}
	var tickerData []stockWithChange
	for _, s := range stocks {
		price := prices[s.ID]
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 1)
		var vol int64
		if price > 0 {
			vol = int64(price * 1000)
		}
		tickerData = append(tickerData, stockWithChange{
			Code:          s.Code,
			Name:          s.Name,
			Price:         price,
			ChangePercent: change,
			Volume:        vol,
		})
	}

	sort.Slice(tickerData, func(i, j int) bool {
		return tickerData[i].Volume > tickerData[j].Volume
	})

	var ticker []tickerStock
	limit := 10
	if len(tickerData) < limit {
		limit = len(tickerData)
	}
	for i := 0; i < limit; i++ {
		ticker = append(ticker, tickerStock{
			Code:          tickerData[i].Code,
			Name:          tickerData[i].Name,
			Price:         tickerData[i].Price,
			ChangePercent: tickerData[i].ChangePercent,
		})
	}

	sort.Slice(tickerData, func(i, j int) bool {
		return tickerData[i].ChangePercent > tickerData[j].ChangePercent
	})
	var gainers []tickerStock
	for i := 0; i < 8 && i < len(tickerData); i++ {
		gainers = append(gainers, tickerStock{Code: tickerData[i].Code, Name: tickerData[i].Name, Price: tickerData[i].Price, ChangePercent: tickerData[i].ChangePercent})
	}

	sort.Slice(tickerData, func(i, j int) bool {
		return tickerData[i].ChangePercent < tickerData[j].ChangePercent
	})
	var losers []tickerStock
	for i := 0; i < 8 && i < len(tickerData); i++ {
		losers = append(losers, tickerStock{Code: tickerData[i].Code, Name: tickerData[i].Name, Price: tickerData[i].Price, ChangePercent: tickerData[i].ChangePercent})
	}

	resp := marketSummaryResponse{
		Breadth:     mb,
		TotalStocks: len(stocks),
		Ticker:      ticker,
		TopGainers:  gainers,
		TopLosers:   losers,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *MarketHandler) EconomicCalendarPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Economic Calendar - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/economic-calendar.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *MarketHandler) EconomicCalendarJSON(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	days := 60
	if d := q.Get("days"); d != "" {
		if parsed, err := time.ParseDuration(d + "h"); err == nil {
			days = int(parsed.Hours() / 24)
		}
	}

	events := h.EconCalendarService.GetUpcomingEvents(days)

	importance := q.Get("importance")
	currency := q.Get("currency")
	dateFrom := q.Get("from")
	dateTo := q.Get("to")

	if importance != "" || currency != "" || dateFrom != "" || dateTo != "" {
		events = h.EconCalendarService.FilterEvents(events, importance, currency, dateFrom, dateTo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (h *MarketHandler) CorrelationPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Correlation Matrix - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/correlation.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

type correlationCell struct {
	CodeA       string  `json:"code_a"`
	CodeB       string  `json:"code_b"`
	NameA       string  `json:"name_a"`
	NameB       string  `json:"name_b"`
	Correlation float64 `json:"correlation"`
}

func pearson(x, y []float64) float64 {
	n := len(x)
	if n < 3 || len(y) != n {
		return 0
	}
	var sx, sy, sxx, syy, sxy float64
	for i := 0; i < n; i++ {
		sx += x[i]
		sy += y[i]
		sxx += x[i] * x[i]
		syy += y[i] * y[i]
		sxy += x[i] * y[i]
	}
	denom := math.Sqrt((float64(n)*sxx - sx*sx) * (float64(n)*syy - sy*sy))
	if denom == 0 {
		return 0
	}
	return (float64(n)*sxy - sx*sy) / denom
}

func (h *MarketHandler) CorrelationJSON(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	universe := q.Get("universe")
	if universe == "" {
		universe = "lq45"
	}

	stocks, err := h.StockRepo.ListActive()
	if err != nil || len(stocks) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "not enough stocks"})
		return
	}

	limit := 20
	if len(stocks) < limit {
		limit = len(stocks)
	}

	selected := stocks[:limit]
	var stockIDs []int64
	for _, s := range selected {
		stockIDs = append(stockIDs, s.ID)
	}

	end := time.Now()
	start := end.AddDate(0, -3, 0)
	allPrices, err := h.StockPriceRepo.FindByDateRange(stockIDs, start, end)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "failed to load prices"})
		return
	}

	priceSeries := make(map[int64][]float64)
	for _, p := range allPrices {
		priceSeries[p.StockID] = append(priceSeries[p.StockID], p.Close)
	}

	type stockInfo struct {
		Code   string    `json:"code"`
		Name   string    `json:"name"`
		Prices []float64 `json:"prices"`
	}

	var stockData []stockInfo
	for _, s := range selected {
		if prices, ok := priceSeries[s.ID]; ok && len(prices) >= 3 {
			stockData = append(stockData, stockInfo{
				Code:   s.Code,
				Name:   s.Name,
				Prices: prices,
			})
		}
	}

	if len(stockData) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "insufficient data"})
		return
	}

	var cells []correlationCell
	for i := 0; i < len(stockData); i++ {
		for j := i; j < len(stockData); j++ {
			if i == j {
				cells = append(cells, correlationCell{
					CodeA: stockData[i].Code, CodeB: stockData[j].Code,
					NameA: stockData[i].Name, NameB: stockData[j].Name,
					Correlation: 1.0,
				})
			} else {
				minLen := len(stockData[i].Prices)
				if len(stockData[j].Prices) < minLen {
					minLen = len(stockData[j].Prices)
				}
				corr := pearson(stockData[i].Prices[:minLen], stockData[j].Prices[:minLen])
				cells = append(cells, correlationCell{
					CodeA: stockData[i].Code, CodeB: stockData[j].Code,
					NameA: stockData[i].Name, NameB: stockData[j].Name,
					Correlation: math.Round(corr*10000) / 10000,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stocks": stockData,
		"matrix": cells,
		"count":  len(stockData),
	})
}

type forexHeatmapCell struct {
	Pair   string  `json:"pair"`
	Name   string  `json:"name"`
	Rate   float64 `json:"rate"`
	Change float64 `json:"change"`
	Group  string  `json:"group"`
}

func (h *MarketHandler) ForexHeatmapJSON(w http.ResponseWriter, r *http.Request) {
	pairs, err := h.ForexRepo.FindAllPairs()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load forex pairs"})
		return
	}

	latestRates, _ := h.ForexRepo.FindLatestRates()
	rateMap := make(map[int64]float64)
	for _, rate := range latestRates {
		rateMap[rate.PairID] = rate.Close
	}

	var cells []forexHeatmapCell
	for _, pair := range pairs {
		rate := rateMap[pair.ID]
		change := 0.0
		if rate > 0 {
			change = 0.5
		}
		cells = append(cells, forexHeatmapCell{
			Pair:   pair.BaseCurrency + "/" + pair.QuoteCurrency,
			Name:   pair.Name,
			Rate:   rate,
			Change: change,
			Group:  pair.Group,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pairs": cells,
		"count": len(cells),
	})
}

func (h *MarketHandler) StockDetailLiquidityJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "stock not found"})
		return
	}

	liquiditySvc := &service.LiquidityService{StockPriceRepo: h.StockPriceRepo}
	zones, err := liquiditySvc.FindLiquidityZones(stock.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if zones == nil {
		zones = []service.LiquidityZone{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stock_code": stock.Code,
		"stock_name": stock.Name,
		"zones":      zones,
		"count":      len(zones),
	})
}

func (h *MarketHandler) LiveSignalsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Live Trading Signals - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/live-signals.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *MarketHandler) SignalsJSON(w http.ResponseWriter, r *http.Request) {
	if h.SignalGenerator == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
		return
	}

	signals, err := h.SignalGenerator.GenerateSignals()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if signals == nil {
		signals = []service.TradingSignal{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

func (h *MarketHandler) SignalGeneratorPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "AI Signal Generator - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/signal-generator.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *MarketHandler) SignalByStockJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if h.SignalGenerator == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "service not available"})
		return
	}

	signal, err := h.SignalGenerator.GetSignalsByStock(code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signal)
}

func (h *MarketHandler) TrendScannerPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Trend Scanner - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/trend-scanner.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *MarketHandler) TrendsJSON(w http.ResponseWriter, r *http.Request) {
	if h.TrendScanner == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
		return
	}

	trends, err := h.TrendScanner.ScanTrends()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if trends == nil {
		trends = []service.TrendResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}

type PriceActionEvent struct {
	StockCode  string  `json:"stock_code"`
	StockName  string  `json:"stock_name"`
	EventType  string  `json:"event_type"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
	Volume     int64   `json:"volume"`
	Detail     string  `json:"detail"`
	CreatedAt  string  `json:"created_at"`
}

func (h *MarketHandler) PriceActionPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Price Action Monitor - Investo",
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "market/price-action.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *MarketHandler) PriceActionJSON(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PriceActionEvent{})
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	priceMap, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var events []PriceActionEvent
	var mu sync.Mutex

	for _, st := range stocks {
		stock := st
		price := priceMap[stock.ID]
		if price <= 0 {
			continue
		}

		prices, err := h.StockPriceRepo.FindLatest(stock.ID, 260)
		if err != nil || len(prices) < 5 {
			continue
		}

		closes := make([]float64, len(prices))
		highs := make([]float64, len(prices))
		lows := make([]float64, len(prices))
		volumes := make([]int64, len(prices))
		for i := 0; i < len(prices); i++ {
			j := len(prices) - 1 - i
			closes[i] = prices[j].Close
			highs[i] = prices[j].High
			lows[i] = prices[j].Low
			volumes[i] = prices[j].Volume
		}

		n := len(closes)
		currentPrice := closes[n-1]
		currentVolume := volumes[n-1]
		currTime := now.Format(time.RFC3339)

		prevDayPrice := closes[0]
		if n >= 2 {
			prevDayPrice = closes[n-2]
		}

		changePct := 0.0
		if prevDayPrice > 0 {
			changePct = ((currentPrice - prevDayPrice) / prevDayPrice) * 100
		}

		high52 := currentPrice
		low52 := currentPrice
		for _, c := range closes {
			if c > high52 {
				high52 = c
			}
			if c < low52 {
				low52 = c
			}
		}

		if high52 > 0 && currentPrice >= high52*0.995 {
			mu.Lock()
			events = append(events, PriceActionEvent{
				StockCode: stock.Code, StockName: stock.Name,
				EventType: "new_high", Price: math.Round(currentPrice*100) / 100,
				ChangePct: math.Round(changePct*100) / 100,
				Detail:    "New 52-week high",
				CreatedAt: currTime,
			})
			mu.Unlock()
		}

		if low52 > 0 && currentPrice <= low52*1.005 {
			mu.Lock()
			events = append(events, PriceActionEvent{
				StockCode: stock.Code, StockName: stock.Name,
				EventType: "new_low", Price: math.Round(currentPrice*100) / 100,
				ChangePct: math.Round(changePct*100) / 100,
				Detail:    "New 52-week low",
				CreatedAt: currTime,
			})
			mu.Unlock()
		}

		if prevDayPrice > 0 && changePct > 3.0 {
			mu.Lock()
			events = append(events, PriceActionEvent{
				StockCode: stock.Code, StockName: stock.Name,
				EventType: "gap_up", Price: math.Round(currentPrice*100) / 100,
				ChangePct: math.Round(changePct*100) / 100,
				Volume:    currentVolume,
				Detail:    fmt.Sprintf("Gap up %.2f%% dari harga sebelumnya", changePct),
				CreatedAt: currTime,
			})
			mu.Unlock()
		}

		if prevDayPrice > 0 && changePct < -3.0 {
			mu.Lock()
			events = append(events, PriceActionEvent{
				StockCode: stock.Code, StockName: stock.Name,
				EventType: "gap_down", Price: math.Round(currentPrice*100) / 100,
				ChangePct: math.Round(changePct*100) / 100,
				Volume:    currentVolume,
				Detail:    fmt.Sprintf("Gap down %.2f%% dari harga sebelumnya", changePct),
				CreatedAt: currTime,
			})
			mu.Unlock()
		}

		if n >= 21 {
			avgVol := int64(0)
			for i := n - 21; i < n-1; i++ {
				avgVol += volumes[i]
			}
			avgVol /= 20
			if avgVol > 0 && currentVolume > 3*avgVol {
				mu.Lock()
				events = append(events, PriceActionEvent{
					StockCode: stock.Code, StockName: stock.Name,
					EventType: "volume_spike", Price: math.Round(currentPrice*100) / 100,
					ChangePct: math.Round(changePct*100) / 100,
					Volume:    currentVolume,
					Detail:    fmt.Sprintf("Volume %.0fx dari rata-rata 20 hari", float64(currentVolume)/float64(avgVol)),
					CreatedAt: currTime,
				})
				mu.Unlock()
			}
		}

		if n >= 20 {
			volCloses := make([]float64, n)
			for i := 0; i < n; i++ {
				volCloses[i] = closes[i]
			}
			rsi := indicator.CalcRSI(volCloses, 14)
			lastRSI := lastValid(rsi)
			if !math.IsNaN(lastRSI) {
				if lastRSI > 70 {
					mu.Lock()
					events = append(events, PriceActionEvent{
						StockCode: stock.Code, StockName: stock.Name,
						EventType: "rsi_overbought", Price: math.Round(currentPrice*100) / 100,
						ChangePct: math.Round(changePct*100) / 100,
						Detail:    fmt.Sprintf("RSI overbought: %.1f", lastRSI),
						CreatedAt: currTime,
					})
					mu.Unlock()
				} else if lastRSI < 30 {
					mu.Lock()
					events = append(events, PriceActionEvent{
						StockCode: stock.Code, StockName: stock.Name,
						EventType: "rsi_oversold", Price: math.Round(currentPrice*100) / 100,
						ChangePct: math.Round(changePct*100) / 100,
						Detail:    fmt.Sprintf("RSI oversold: %.1f", lastRSI),
						CreatedAt: currTime,
					})
					mu.Unlock()
				}
			}
		}

		if n >= 30 {
			support := closes[n-1]
			resistance := closes[n-1]
			for i := maxInt(0, n-30); i < n; i++ {
				if closes[i] < support {
					support = closes[i]
				}
				if closes[i] > resistance {
					resistance = closes[i]
				}
			}

			if currentPrice < support*1.01 && currentPrice > support*0.98 {
				mu.Lock()
				events = append(events, PriceActionEvent{
					StockCode: stock.Code, StockName: stock.Name,
					EventType: "support_break", Price: math.Round(currentPrice*100) / 100,
					ChangePct: math.Round(changePct*100) / 100,
					Detail:    fmt.Sprintf("Mendekati support %.0f", math.Round(support)),
					CreatedAt: currTime,
				})
				mu.Unlock()
			}

			if currentPrice > resistance*0.99 && currentPrice < resistance*1.02 {
				mu.Lock()
				events = append(events, PriceActionEvent{
					StockCode: stock.Code, StockName: stock.Name,
					EventType: "resistance_break", Price: math.Round(currentPrice*100) / 100,
					ChangePct: math.Round(changePct*100) / 100,
					Detail:    fmt.Sprintf("Mendekati resistance %.0f", math.Round(resistance)),
					CreatedAt: currTime,
				})
				mu.Unlock()
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt > events[j].CreatedAt
	})

	if events == nil {
		events = []PriceActionEvent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func lastValid(arr []float64) float64 {
	for i := len(arr) - 1; i >= 0; i-- {
		if !math.IsNaN(arr[i]) && !math.IsInf(arr[i], 0) {
			return arr[i]
		}
	}
	return math.NaN()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}


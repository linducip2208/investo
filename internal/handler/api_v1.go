package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"
	"investo/internal/service/indicator"
	"investo/internal/service/pattern"

	"github.com/go-chi/chi/v5"
)

type APIv1Handler struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
	NewsRepo             *repository.NewsRepository
	ForexRepo            *repository.ForexRepository
	HeatmapService       *service.HeatmapService
	BreadthService       *service.MarketBreadthService
	ChartService         *service.ChartService
	ValuationService     *service.ValuationService
	PatternService       *pattern.PatternService
	ForexAnalytics       *service.ForexAnalytics
	AuthService          *service.AuthService
	JWTService           *service.JWTService
	SettingRepo          *repository.SettingRepository
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Meta      interface{} `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
}

type PaginationMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success:   status < 400,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message}, nil)
}

type StockListItem struct {
	ID            int64   `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	SectorName    string  `json:"sector_name"`
	SectorID      int64   `json:"sector_id"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"change_percent"`
	Volume        int64   `json:"volume"`
	MarketCap     float64 `json:"market_cap"`
}

func sortStockListItems(results []StockListItem, sortBy, sortOrder string) {
	asc := strings.ToLower(sortOrder) != "desc"
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			var less bool
			switch strings.ToLower(sortBy) {
			case "price":
				less = results[i].Price < results[j].Price
			case "change", "change_percent":
				less = results[i].ChangePercent < results[j].ChangePercent
			case "market_cap":
				less = results[i].MarketCap < results[j].MarketCap
			default:
				less = results[i].Code < results[j].Code
			}
			swap := false
			if asc && less {
				swap = true
			}
			if !asc && !less {
				swap = true
			}
			if swap {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func (h *APIv1Handler) ListStocks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sector := q.Get("sector")
	search := q.Get("search")
	page := 1
	limit := 50
	sortBy := q.Get("sort_by")
	sortOrder := q.Get("sort_order")

	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sectorID := int64(0)
	if sector != "" {
		if v, err := strconv.ParseInt(sector, 10, 64); err == nil {
			sectorID = v
		} else {
			sec, err := h.SectorRepo.FindBySlug(sector)
			if err == nil {
				sectorID = sec.ID
			}
		}
	}

	offset := (page - 1) * limit
	stocks, total, err := h.StockRepo.List(offset, limit, search, sectorID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load stocks")
		return
	}

	sectors, _ := h.SectorRepo.FindAll()
	sectorMap := make(map[int64]string)
	for _, s := range sectors {
		sectorMap[s.ID] = s.Name
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	priceMap, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var results []StockListItem
	for _, s := range stocks {
		price := priceMap[s.ID]
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 1)
		results = append(results, StockListItem{
			ID:            s.ID,
			Code:          s.Code,
			Name:          s.Name,
			SectorName:    sectorMap[s.SectorID],
			SectorID:      s.SectorID,
			Price:         math.Round(price*100) / 100,
			ChangePercent: math.Round(change*100) / 100,
			Volume:        0,
			MarketCap:     price * float64(s.SharesOutstanding),
		})
	}

	if sortBy != "" {
		sortStockListItems(results, sortBy, sortOrder)
	}

	writeJSON(w, http.StatusOK, results, PaginationMeta{
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (h *APIv1Handler) StockDetail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	prices, _ := h.StockPriceRepo.FindLatest(stock.ID, 1)
	var latestPrice float64
	var latestVolume int64
	if len(prices) > 0 {
		latestPrice = math.Round(prices[0].Close*100) / 100
		latestVolume = prices[0].Volume
	}

	fundamental, _ := h.StockFundamentalRepo.FindLatest(stock.ID)
	sector, _ := h.SectorRepo.FindByID(stock.SectorID)
	change, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)

	type StockDetailResponse struct {
		ID                int64                    `json:"id"`
		Code              string                   `json:"code"`
		Name              string                   `json:"name"`
		SectorName        string                   `json:"sector_name"`
		SectorID          int64                    `json:"sector_id"`
		Subsector         string                   `json:"subsector"`
		ListingDate       string                   `json:"listing_date"`
		SharesOutstanding int64                    `json:"shares_outstanding"`
		Website           string                   `json:"website"`
		Description       string                   `json:"description"`
		Price             float64                  `json:"price"`
		ChangePercent     float64                  `json:"change_percent"`
		Volume            int64                    `json:"volume"`
		MarketCap         float64                  `json:"market_cap"`
		Fundamental       *model.StockFundamental `json:"fundamental,omitempty"`
	}

	resp := StockDetailResponse{
		ID:                stock.ID,
		Code:              stock.Code,
		Name:              stock.Name,
		SectorID:          stock.SectorID,
		Subsector:         stock.Subsector,
		ListingDate:       stock.ListingDate.Format("2006-01-02"),
		SharesOutstanding: stock.SharesOutstanding,
		Website:           stock.Website,
		Description:       stock.Description,
		Price:             latestPrice,
		ChangePercent:     math.Round(change*100) / 100,
		Volume:            latestVolume,
		MarketCap:         latestPrice * float64(stock.SharesOutstanding),
		Fundamental:       fundamental,
	}
	if sector != nil {
		resp.SectorName = sector.Name
	}

	writeJSON(w, http.StatusOK, resp, nil)
}

func (h *APIv1Handler) StockPrices(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	q := r.URL.Query()
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()

	if s := q.Get("start"); s != "" {
		if parsed, err := time.Parse("2006-01-02", s); err == nil {
			start = parsed
		}
	}
	if e := q.Get("end"); e != "" {
		if parsed, err := time.Parse("2006-01-02", e); err == nil {
			end = parsed
		}
	}

	interval := q.Get("interval")
	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, start, end)
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			writeJSON(w, http.StatusOK, []interface{}{}, nil)
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	aggregated := aggregateByInterval(prices, interval)

	writeJSON(w, http.StatusOK, aggregated, nil)
}

func aggregateByInterval(prices []model.StockPrice, interval string) []model.StockPrice {
	if interval == "" {
		return prices
	}

	var result []model.StockPrice
	var current *model.StockPrice

	for i, p := range prices {
		shouldGroup := false
		if current != nil {
			switch interval {
			case "1wk":
				_, cw := current.Date.ISOWeek()
				_, pw := p.Date.ISOWeek()
				cy, py := current.Date.Year(), p.Date.Year()
				shouldGroup = (cw == pw && cy == py)
			case "1mo":
				shouldGroup = (current.Date.Year() == p.Date.Year() && current.Date.Month() == p.Date.Month())
			}
		}

		if shouldGroup {
			if p.High > current.High {
				current.High = p.High
			}
			if p.Low < current.Low || current.Low == 0 {
				current.Low = p.Low
			}
			current.Close = p.Close
			current.Volume += p.Volume
		} else {
			if current != nil {
				result = append(result, *current)
			}
			cp := p
			current = &cp
		}

		if i == len(prices)-1 && current != nil {
			result = append(result, *current)
		}
	}

	return result
}

func (h *APIv1Handler) StockFundamentals(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	limit := 40
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	fundamentals, _ := h.StockFundamentalRepo.FindByStockID(stock.ID, limit)

	type FundamentalResponse struct {
		StockCode    string                   `json:"stock_code"`
		StockName    string                   `json:"stock_name"`
		Fundamentals []model.StockFundamental `json:"fundamentals"`
	}

	if fundamentals == nil {
		fundamentals = []model.StockFundamental{}
	}

	writeJSON(w, http.StatusOK, FundamentalResponse{
		StockCode:    stock.Code,
		StockName:    stock.Name,
		Fundamentals: fundamentals,
	}, nil)
}

func (h *APIv1Handler) StockIndicators(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			writeJSONError(w, http.StatusNotFound, "no price data")
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

	writeJSON(w, http.StatusOK, resp, nil)
}

func (h *APIv1Handler) StockPatterns(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	prices, err := h.StockPriceRepo.FindByStockDate(stock.ID, time.Now().AddDate(-2, 0, 0), time.Now())
	if err != nil || len(prices) == 0 {
		latest, err := h.StockPriceRepo.FindLatest(stock.ID, 365)
		if err != nil || len(latest) == 0 {
			writeJSON(w, http.StatusOK, []interface{}{}, nil)
			return
		}
		service.ReversePrices(latest)
		prices = latest
	}

	results := h.PatternService.DetectAll(prices)
	if results == nil {
		results = []pattern.PatternResult{}
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = v
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	writeJSON(w, http.StatusOK, results, nil)
}

type ForexPairData struct {
	ID            int64   `json:"id"`
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
	Name          string  `json:"name"`
	Group         string  `json:"group"`
	Rate          float64 `json:"rate"`
	ChangePercent float64 `json:"change_percent"`
}

func (h *APIv1Handler) ForexPairs(w http.ResponseWriter, r *http.Request) {
	pairs, err := h.ForexRepo.FindAllPairs()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load forex pairs")
		return
	}

	latestRates, _ := h.ForexRepo.FindLatestRates()
	rateMap := make(map[int64]model.ForexRate)
	for _, rate := range latestRates {
		rateMap[rate.PairID] = rate
	}

	var results []ForexPairData
	for _, p := range pairs {
		rate := rateMap[p.ID]
		results = append(results, ForexPairData{
			ID:            p.ID,
			BaseCurrency:  p.BaseCurrency,
			QuoteCurrency: p.QuoteCurrency,
			Name:          p.Name,
			Group:         p.Group,
			Rate:          math.Round(rate.Close*10000) / 10000,
			ChangePercent: 0,
		})
	}

	if results == nil {
		results = []ForexPairData{}
	}

	writeJSON(w, http.StatusOK, results, nil)
}

func (h *APIv1Handler) ForexPairDetail(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")

	pair, err := h.ForexRepo.FindPairByCurrencies(base, quote)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "pair not found")
		return
	}

	rates, _ := h.ForexRepo.FindRates(pair.ID, time.Now().AddDate(-3, 0, 0), time.Now())

	var latestRate float64
	if len(rates) > 0 {
		latestRate = math.Round(rates[len(rates)-1].Close*10000) / 10000
	}

	type PairDetailResponse struct {
		Pair  ForexPairData     `json:"pair"`
		Rates []model.ForexRate `json:"rates"`
	}

	resp := PairDetailResponse{
		Pair: ForexPairData{
			ID:            pair.ID,
			BaseCurrency:  pair.BaseCurrency,
			QuoteCurrency: pair.QuoteCurrency,
			Name:          pair.Name,
			Group:         pair.Group,
			Rate:          latestRate,
			ChangePercent: 0,
		},
		Rates: rates,
	}
	if resp.Rates == nil {
		resp.Rates = []model.ForexRate{}
	}

	writeJSON(w, http.StatusOK, resp, nil)
}

type MarketSummaryResponse struct {
	Breadth     *service.MarketBreadth `json:"breadth"`
	TotalStocks int                    `json:"total_stocks"`
	TotalVolume int64                  `json:"total_volume"`
	TotalValue  float64                `json:"total_value"`
	TopGainers  []Mover                `json:"top_gainers"`
	TopLosers   []Mover                `json:"top_losers"`
	Timestamp   string                 `json:"timestamp"`
}

func (h *APIv1Handler) MarketSummary(w http.ResponseWriter, r *http.Request) {
	mb, _ := h.BreadthService.Calculate()

	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load stocks")
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var gainers, losers []Mover
	totalVolume := int64(0)
	totalValue := 0.0

	for _, stock := range stocks {
		price := prices[stock.ID]
		change, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)

		mover := Mover{
			Code:   stock.Code,
			Name:   stock.Name,
			Price:  math.Round(price*100) / 100,
			Change: math.Round(change*100) / 100,
		}

		if change > 0 {
			gainers = append(gainers, mover)
		} else if change < 0 {
			losers = append(losers, mover)
		}
	}

	sortMoversDesc(gainers)
	sortMoversAsc(losers)

	if len(gainers) > 10 {
		gainers = gainers[:10]
	}
	if len(losers) > 10 {
		losers = losers[:10]
	}

	if gainers == nil {
		gainers = []Mover{}
	}
	if losers == nil {
		losers = []Mover{}
	}

	if mb != nil {
		totalVolume = mb.TotalVolume
		totalValue = mb.TotalValue
	}

	resp := MarketSummaryResponse{
		Breadth:     mb,
		TotalStocks: len(stocks),
		TotalVolume: totalVolume,
		TotalValue:  totalValue,
		TopGainers:  gainers,
		TopLosers:   losers,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, resp, nil)
}

func (h *APIv1Handler) MarketHeatmap(w http.ResponseWriter, r *http.Request) {
	_, grouped, err := h.HeatmapService.GetHeatmap()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load heatmap")
		return
	}

	type HeatmapSector struct {
		Name   string                `json:"name"`
		Stocks []service.HeatmapCell `json:"stocks"`
	}

	var sectors []HeatmapSector
	for name, stocks := range grouped {
		sectors = append(sectors, HeatmapSector{Name: name, Stocks: stocks})
	}

	if sectors == nil {
		sectors = []HeatmapSector{}
	}

	writeJSON(w, http.StatusOK, sectors, nil)
}

type DailyRecapData struct {
	Date           string     `json:"date"`
	Advance        int        `json:"advance"`
	Decline        int        `json:"decline"`
	Unchanged      int        `json:"unchanged"`
	AdvancePercent float64    `json:"advance_percent"`
	DeclinePercent float64    `json:"decline_percent"`
	TotalVolume    int64      `json:"total_volume"`
	TotalValue     float64    `json:"total_value"`
	ForeignBuy     float64    `json:"foreign_buy"`
	ForeignSell    float64    `json:"foreign_sell"`
	ForeignNet     float64    `json:"foreign_net"`
	TopGainers     []Mover `json:"top_gainers"`
	TopLosers      []Mover `json:"top_losers"`
}

func (h *APIv1Handler) MarketRecap(w http.ResponseWriter, r *http.Request) {
	mb, _ := h.BreadthService.Calculate()

	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load stocks")
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)
	var gainers, losers []Mover

	for _, stock := range stocks {
		price := prices[stock.ID]
		change, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)

		mover := Mover{
			Code:   stock.Code,
			Name:   stock.Name,
			Price:  math.Round(price*100) / 100,
			Change: math.Round(change*100) / 100,
		}

		if change > 0 {
			gainers = append(gainers, mover)
		} else if change < 0 {
			losers = append(losers, mover)
		}
	}

	sortMoversDesc(gainers)
	sortMoversAsc(losers)

	if len(gainers) > 10 {
		gainers = gainers[:10]
	}
	if len(losers) > 10 {
		losers = losers[:10]
	}

	if gainers == nil {
		gainers = []Mover{}
	}
	if losers == nil {
		losers = []Mover{}
	}

	recap := DailyRecapData{
		Date:       time.Now().Format("2006-01-02"),
		TopGainers: gainers,
		TopLosers:  losers,
	}

	if mb != nil {
		recap.Advance = mb.Advance
		recap.Decline = mb.Decline
		recap.Unchanged = mb.Unchanged
		recap.AdvancePercent = mb.AdvancePercent
		recap.DeclinePercent = mb.DeclinePercent
		recap.TotalVolume = mb.TotalVolume
		recap.TotalValue = mb.TotalValue
		recap.ForeignBuy = mb.ForeignBuy
		recap.ForeignSell = mb.ForeignSell
		recap.ForeignNet = mb.ForeignNet
	}

	writeJSON(w, http.StatusOK, recap, nil)
}

type SectorData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	StockCount  int    `json:"stock_count"`
}

func (h *APIv1Handler) Sectors(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.SectorRepo.FindAll()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load sectors")
		return
	}

	stocks, _ := h.StockRepo.ListActive()
	stockCountMap := make(map[int64]int)
	for _, s := range stocks {
		stockCountMap[s.SectorID]++
	}

	result := make([]SectorData, 0, len(sectors))
	for _, sec := range sectors {
		result = append(result, SectorData{
			ID:          sec.ID,
			Name:        sec.Name,
			Slug:        sec.Slug,
			Description: sec.Description,
			StockCount:  stockCountMap[sec.ID],
		})
	}

	writeJSON(w, http.StatusOK, result, nil)
}

func (h *APIv1Handler) News(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	limit := 20
	stockCode := q.Get("stock_code")

	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	offset := (page - 1) * limit

	var news []model.News
	var total int
	var err error

	if stockCode != "" {
		stock, stockErr := h.StockRepo.FindByCode(stockCode)
		if stockErr != nil {
			writeJSONError(w, http.StatusNotFound, "stock not found")
			return
		}
		news, err = h.NewsRepo.FindByStockID(stock.ID, limit)
		total = len(news)
	} else {
		news, total, err = h.NewsRepo.List(offset, limit)
	}

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load news")
		return
	}

	if news == nil {
		news = []model.News{}
	}

	writeJSON(w, http.StatusOK, news, PaginationMeta{
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (h *APIv1Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" || len(strings.TrimSpace(query)) == 0 {
		writeJSON(w, http.StatusOK, []interface{}{}, nil)
		return
	}

	stocks, err := h.StockRepo.Search(query, 20)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "search failed")
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	type SearchResult struct {
		ID            int64   `json:"id"`
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		SectorID      int64   `json:"sector_id"`
		Price         float64 `json:"price"`
		ChangePercent float64 `json:"change_percent"`
	}

	results := make([]SearchResult, 0, len(stocks))
	for _, s := range stocks {
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 1)
		results = append(results, SearchResult{
			ID:            s.ID,
			Code:          s.Code,
			Name:          s.Name,
			SectorID:      s.SectorID,
			Price:         math.Round(prices[s.ID]*100) / 100,
			ChangePercent: math.Round(change*100) / 100,
		})
	}

	writeJSON(w, http.StatusOK, results, nil)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *APIv1Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.AuthService.Login(req.Email, req.Password)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.JWTService.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"token": token,
		"name":  user.Name,
		"role":  user.Role,
	}, nil)
}

func (h *APIv1Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.URL.Query().Get("api_key")
		if apiKey != "" {
			savedKey, err := h.SettingRepo.Get("api_key")
			if err == nil && savedKey == apiKey {
				next.ServeHTTP(w, r)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "invalid api_key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

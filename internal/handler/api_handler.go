package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
	NewsRepo             *repository.NewsRepository
	ForexRepo            *repository.ForexRepository
	ChartService         *service.ChartService
}

type Mover struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

func (h *APIHandler) MarketOverview(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load data"})
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}

	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var gainers, losers []Mover
	totalStocks := len(stocks)

	for _, stock := range stocks {
		price := prices[stock.ID]
		change, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)

		mover := Mover{
			Code:   stock.Code,
			Name:   stock.Name,
			Price:  price,
			Change: change,
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

	resp := map[string]interface{}{
		"total_stocks": totalStocks,
		"gainers":      gainers,
		"losers":       losers,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func sortMoversDesc(movers []Mover) {
	for i := 0; i < len(movers); i++ {
		for j := i + 1; j < len(movers); j++ {
			if movers[i].Change < movers[j].Change {
				movers[i], movers[j] = movers[j], movers[i]
			}
		}
	}
}

func sortMoversAsc(movers []Mover) {
	for i := 0; i < len(movers); i++ {
		for j := i + 1; j < len(movers); j++ {
			if movers[i].Change > movers[j].Change {
				movers[i], movers[j] = movers[j], movers[i]
			}
		}
	}
}

func (h *APIHandler) StockSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" || len(strings.TrimSpace(q)) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	stocks, err := h.StockRepo.Search(q, 20)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "search failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stocks)
}

func (h *APIHandler) StockChart(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id := int64(atoi(idStr))
	if id == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid stock id"})
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

	indicators := q["indicators"]

	chartData, err := h.ChartService.GetStockChartData(id, start, end, indicators)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to get chart data"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartData)
}

func (h *APIHandler) StockLatestPrice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id := int64(atoi(idStr))
	if id == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid stock id"})
		return
	}

	prices, err := h.StockPriceRepo.FindLatest(id, 1)
	if err != nil || len(prices) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "no price data"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prices[0])
}

func (h *APIHandler) SectorsData(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.SectorRepo.FindAll()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load sectors"})
		return
	}

	type SectorData struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		StockCount  int    `json:"stock_count"`
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *APIHandler) NewsFeed(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 20
	if l := q.Get("limit"); l != "" {
		limit = atoi(l)
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	news, _, err := h.NewsRepo.List(0, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load news"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

func (h *APIHandler) ForexRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.ForexRepo.FindLatestRates()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load rates"})
		return
	}

	pairs, _ := h.ForexRepo.FindAllPairs()
	pairMap := make(map[int64]string)
	for _, p := range pairs {
		pairMap[p.ID] = p.BaseCurrency + "/" + p.QuoteCurrency
	}

	type RateData struct {
		Pair  string  `json:"pair"`
		Date  string  `json:"date"`
		Open  float64 `json:"open"`
		High  float64 `json:"high"`
		Low   float64 `json:"low"`
		Close float64 `json:"close"`
	}

	result := make([]RateData, 0, len(rates))
	for _, rate := range rates {
		result = append(result, RateData{
			Pair:  pairMap[rate.PairID],
			Date:  rate.Date.Format("2006-01-02"),
			Open:  rate.Open,
			High:  rate.High,
			Low:   rate.Low,
			Close: rate.Close,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *APIHandler) TopGainers(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load data"})
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}
	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var gainers []Mover
	for _, stock := range stocks {
		change, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)
		if change > 0 {
			gainers = append(gainers, Mover{
				Code:   stock.Code,
				Name:   stock.Name,
				Price:  prices[stock.ID],
				Change: change,
			})
		}
	}

	sortMoversDesc(gainers)
	if len(gainers) > 10 {
		gainers = gainers[:10]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gainers)
}

func (h *APIHandler) TopLosers(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.StockRepo.ListActive()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to load data"})
		return
	}

	var stockIDs []int64
	for _, s := range stocks {
		stockIDs = append(stockIDs, s.ID)
	}
	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var losers []Mover
	for _, stock := range stocks {
		change, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)
		if change < 0 {
			losers = append(losers, Mover{
				Code:   stock.Code,
				Name:   stock.Name,
				Price:  prices[stock.ID],
				Change: change,
			})
		}
	}

	sortMoversAsc(losers)
	if len(losers) > 10 {
		losers = losers[:10]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(losers)
}

package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type ExportHandler struct {
	ExportService    *service.ExportService
	PDFReportService *service.PDFReportService
	PDFExportService *service.PDFExportService
	WebhookService   *service.WebhookService

	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
	PortfolioRepo        *repository.PortfolioRepository
	PortfolioItemRepo    *repository.PortfolioItemRepository
	PortfolioAnalytics   *service.PortfolioAnalytics
	ScreenerService      *service.ScreenerService
	ValuationService     *service.ValuationService
}

func (h *ExportHandler) ExportStocksCSV(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.StockRepo.ListActive()
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

	prices, _ := h.StockPriceRepo.GetLatestPrices(stockIDs)

	var rows []service.StockCSVRow
	for _, s := range stocks {
		price := prices[s.ID]
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 1)

		var per, pbv, roe, der float64
		fund, err := h.StockFundamentalRepo.FindLatest(s.ID)
		if err == nil && fund != nil {
			per = fund.PER
			pbv = fund.PBV
			roe = fund.ROE
			der = fund.DER
		}

		rows = append(rows, service.StockCSVRow{
			Code:          s.Code,
			Name:          s.Name,
			SectorName:    sectorMap[s.SectorID],
			Price:         math.Round(price*100) / 100,
			ChangePercent: math.Round(change*100) / 100,
			PER:           math.Round(per*100) / 100,
			PBV:           math.Round(pbv*100) / 100,
			ROE:           math.Round(roe*100) / 100,
			DER:           math.Round(der*100) / 100,
			MarketCap:     price * float64(s.SharesOutstanding),
		})
	}

	csvData, err := h.ExportService.ExportStocksCSV(rows)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate CSV")
		return
	}

	filename := "investo_stocks_" + time.Now().Format("20060102") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Write(csvData)
}

func (h *ExportHandler) ExportPortfolioCSV(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	perf, err := h.PortfolioAnalytics.CalcPerformance(id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to calculate performance")
		return
	}

	var rows []service.PortfolioCSVRow
	for _, hp := range perf.Holdings {
		rows = append(rows, service.PortfolioCSVRow{
			Code:         hp.Stock.Code,
			Name:         hp.Stock.Name,
			Quantity:     hp.Quantity,
			AvgPrice:     hp.AvgPrice,
			CurrentPrice: hp.CurrentPrice,
			MarketValue:  hp.MarketValue,
			Gain:         hp.Gain,
			GainPercent:  hp.GainPercent,
		})
	}

	csvData, err := h.ExportService.ExportPortfolioCSV(rows)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate CSV")
		return
	}

	filename := "investo_portfolio_" + portfolio.Name + "_" + time.Now().Format("20060102") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Write(csvData)
}

func (h *ExportHandler) ExportScreenerCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	preset := q.Get("preset")

	var sr []service.ScreenerResult
	var err error

	if preset != "" {
		sr, err = h.ScreenerService.QuickScreen(preset)
	} else {
		criteria := parseScreenerCriteria(r)
		sr, err = h.ScreenerService.Screen(criteria)
	}

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to run screener")
		return
	}

	var rows []service.ScreenerCSVRow
	for _, r := range sr {
		rows = append(rows, service.ScreenerCSVRow{
			Code:          r.Code,
			Name:          r.Name,
			SectorName:    r.SectorName,
			Price:         r.Price,
			ChangePercent: r.ChangePercent,
			PER:           r.PER,
			PBV:           r.PBV,
			ROE:           r.ROE,
			DER:           r.DER,
			NPM:           r.NPM,
			DivYield:      r.DivYield,
			MarketCap:     r.MarketCap,
			Score:         r.Score,
		})
	}

	csvData, err := h.ExportService.ExportScreenerCSV(rows)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate CSV")
		return
	}

	filename := "investo_screener_" + time.Now().Format("20060102") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Write(csvData)
}

func parseScreenerCriteria(r *http.Request) service.ScreenerCriteria {
	q := r.URL.Query()
	c := service.ScreenerCriteria{}

	if v := q.Get("min_per"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MinPER = f
		}
	}
	if v := q.Get("max_per"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MaxPER = f
		}
	}
	if v := q.Get("min_pbv"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MinPBV = f
		}
	}
	if v := q.Get("max_pbv"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MaxPBV = f
		}
	}
	if v := q.Get("min_roe"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MinROE = f
		}
	}
	if v := q.Get("min_der"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MinDER = f
		}
	}
	if v := q.Get("max_der"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MaxDER = f
		}
	}
	if v := q.Get("min_div_yield"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.MinDivYield = f
		}
	}
	if v := q.Get("sector_id"); v != "" {
		if sid, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.SectorID = sid
		}
	}
	if v := q.Get("sort_by"); v != "" {
		c.SortBy = v
	}
	if v := q.Get("sort_order"); v != "" {
		c.SortOrder = v
	}

	return c
}

func (h *ExportHandler) StockReport(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	fund, _ := h.StockFundamentalRepo.FindLatest(stock.ID)

	valuation, _ := h.ValuationService.Calculate(stock.ID)

	var peers []service.PeerData
	var industryAvg map[string]float64
	if peerComp, err := h.ValuationService.ComparePeers(stock.ID); err == nil && peerComp != nil {
		peers = peerComp.Peers
		industryAvg = peerComp.IndustryAvg
	}

	htmlData, err := h.PDFReportService.GenerateStockReport(*stock, fund, valuation, peers, industryAvg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate report")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlData)
}

func (h *ExportHandler) PortfolioReport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	perf, err := h.PortfolioAnalytics.CalcPerformance(id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to calculate performance")
		return
	}

	attr, _ := h.PortfolioAnalytics.CalcAttribution(id)

	htmlData, err := h.PDFReportService.GeneratePortfolioReport(*portfolio, perf, attr)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate report")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlData)
}

func (h *ExportHandler) WebhookAlertReceiver(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	resp := map[string]interface{}{
		"received": true,
		"event":    payload["event"],
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, resp, nil)
}

type WebhookTestPayload struct {
	URL     string `json:"url"`
	Event   string `json:"event"`
	Message string `json:"message"`
}

func (h *ExportHandler) WebhookTest(w http.ResponseWriter, r *http.Request) {
	var payload WebhookTestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if payload.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}

	config := service.WebhookConfig{
		URL:    payload.URL,
		Events: []string{payload.Event},
		Secret: "",
	}

	alert := model.Alert{
		ID:          0,
		StockID:     0,
		Condition:   "test",
		TargetPrice: 0,
	}

	stock := model.Stock{
		Code: "TEST",
		Name: "Test Stock",
	}

	err := h.WebhookService.SendAlertWebhook(config, alert, stock, 0)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "webhook test failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"message": "Webhook test sent successfully",
	}, nil)
}

func (h *ExportHandler) ExportStockPDF(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "stock not found")
		return
	}

	fund, _ := h.StockFundamentalRepo.FindLatest(stock.ID)

	price := 0.0
	change := 0.0
	priceMap, _ := h.StockPriceRepo.GetLatestPrices([]int64{stock.ID})
	if p, ok := priceMap[stock.ID]; ok {
		price = p
	}
	prev, _ := h.StockPriceRepo.GetPriceChange(stock.ID, 1)
	if prev > 0 {
		change = ((price - prev) / prev) * 100
	}

	pdfData, err := h.PDFExportService.GenerateStockReportPDF(*stock, price, change, fund)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\""+stock.Code+"_report.html\"")
	w.Write(pdfData)
}

func (h *ExportHandler) ExportPortfolioPDF(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	perf, err := h.PortfolioAnalytics.CalcPerformance(id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to calculate performance")
		return
	}

	var holdings []service.PortfolioHoldingExport
	for _, hp := range perf.Holdings {
		holdings = append(holdings, service.PortfolioHoldingExport{
			Code:         hp.Stock.Code,
			Name:         hp.Stock.Name,
			Quantity:     hp.Quantity,
			AvgPrice:     hp.AvgPrice,
			CurrentPrice: hp.CurrentPrice,
			MarketValue:  hp.MarketValue,
			Gain:         hp.Gain,
			GainPercent:  hp.GainPercent,
		})
	}

	pdfData, err := h.PDFExportService.GeneratePortfolioPDF(*portfolio, holdings, perf.TotalValue, perf.TotalCost, perf.TotalGain, perf.TotalGainPercent)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"portfolio_"+portfolio.Name+"_report.html\"")
	w.Write(pdfData)
}

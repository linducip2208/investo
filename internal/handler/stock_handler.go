package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"investo/internal/model"
	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type StockListRow struct {
	Code          string
	Name          string
	SectorName    string
	Price         float64
	ChangePercent float64
	Volume        int64
}

type StockHandler struct {
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
	NewsRepo             *repository.NewsRepository
	ChartService         *service.ChartService
	Templates            *template.Template
}

func (h *StockHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := q.Get("search")
	sectorID := int64(0)
	if sid := q.Get("sector"); sid != "" {
		sectorID = int64(atoi(sid))
	}
	page := 1
	if p := q.Get("page"); p != "" {
		page = atoi(p)
	}
	if page < 1 {
		page = 1
	}

	perPage := 50
	offset := (page - 1) * perPage

	stocks, total, err := h.StockRepo.List(offset, perPage, search, sectorID)
	if err != nil {
		log.Printf("StockHandler.List error: %v", err)
		http.Error(w, "Gagal memuat data saham", http.StatusInternalServerError)
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

	var rows []StockListRow
	for _, s := range stocks {
		price := priceMap[s.ID]
		prevPrice := 0.0
		change := 0.0
		if price > 0 {
			prevPrice, _ = h.StockPriceRepo.GetPriceChange(s.ID, 1)
			if prevPrice > 0 {
				change = ((price - prevPrice) / prevPrice) * 100
			}
		}
		rows = append(rows, StockListRow{
			Code:          s.Code,
			Name:          s.Name,
			SectorName:    sectorMap[s.SectorID],
			Price:         price,
			ChangePercent: change,
		})
	}

	var currentSector *model.Sector
	if sectorID > 0 {
		currentSector, _ = h.SectorRepo.FindByID(sectorID)
	}

	data := map[string]interface{}{
		"Title":         "Daftar Saham - Investo",
		"Stocks":        rows,
		"Sectors":       sectors,
		"Search":        search,
		"CurrentSector": currentSector,
		"TotalCount":    total,
		"Page":          page,
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "stocks/list.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *StockHandler) Detail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	latestPrices, _ := h.StockPriceRepo.FindLatest(stock.ID, 30)
	var latestPrice float64
	var priceData map[string]interface{}
	if len(latestPrices) > 0 {
		latestPrice = latestPrices[0].Close
		prevClose := 0.0
		if len(latestPrices) > 1 {
			prevClose = latestPrices[1].Close
		}
		change := 0.0
		changeAmt := 0.0
		if prevClose > 0 {
			change = ((latestPrice - prevClose) / prevClose) * 100
			changeAmt = latestPrice - prevClose
		}
		vol := int64(0)
		if latestPrices[0].Volume > 0 {
			vol = latestPrices[0].Volume
		}
		priceData = map[string]interface{}{
			"PriceFormatted":  fmt.Sprintf("Rp %s", humanizeNum(int64(latestPrice))),
			"ChangePercent":   change,
			"ChangeFormatted": fmt.Sprintf("Rp %s", humanizeNum(int64(changeAmt))),
			"VolumeFormatted": fmt.Sprintf("%d", vol),
			"Open":            latestPrices[0].Open,
			"OpenFormatted":   fmt.Sprintf("Rp %s", humanizeNum(int64(latestPrices[0].Open))),
			"PrevFormatted":   fmt.Sprintf("Rp %s", humanizeNum(int64(prevClose))),
		}
	} else {
		priceData = map[string]interface{}{
			"PriceFormatted":  "Rp 0",
			"ChangePercent":   0.0,
			"ChangeFormatted": "Rp 0",
			"VolumeFormatted": "0",
		}
	}

	fundamental, _ := h.StockFundamentalRepo.FindLatest(stock.ID)
	news, _ := h.NewsRepo.FindByStockID(stock.ID, 5)
	sector, _ := h.SectorRepo.FindByID(stock.SectorID)

	type FundSummary struct {
		Source                     string
		MarketCapFormatted          string
		PER                         float64
		PBV                         float64
		EPSFormatted                string
		ROE                         float64
		ROA                         float64
		DER                         float64
		NPM                         float64
		DividendYield               float64
		RevenueFormatted            string
		NetIncomeFormatted          string
		BVPSFormatted               string
		SharesOutstandingFormatted  string
		Low52WFormatted             string
		High52WFormatted            string
	}

	var fundSummary FundSummary
	if fundamental != nil {
		marketCap := float64(stock.SharesOutstanding) * latestPrice
		low52 := latestPrice * 0.7
		high52 := latestPrice * 1.3
		if len(latestPrices) > 0 {
			minP := latestPrices[0].Low
			maxP := latestPrices[0].High
			for _, p := range latestPrices {
				if p.Low < minP { minP = p.Low }
				if p.High > maxP { maxP = p.High }
			}
			low52 = minP
			high52 = maxP
		}
		fundSummary = FundSummary{
			Source:                     fundamental.Source,
			MarketCapFormatted:         fmt.Sprintf("Rp %s", humanizeNum(int64(marketCap))),
			PER:                        fundamental.PER,
			PBV:                        fundamental.PBV,
			EPSFormatted:               fmt.Sprintf("Rp %.2f", fundamental.EPS),
			ROE:                        fundamental.ROE,
			ROA:                        fundamental.ROA,
			DER:                        fundamental.DER,
			NPM:                        fundamental.NetProfitMargin,
			DividendYield:              fundamental.DividendYield,
			RevenueFormatted:           fmt.Sprintf("Rp %s", humanizeNum(int64(fundamental.Revenue))),
			NetIncomeFormatted:         fmt.Sprintf("Rp %s", humanizeNum(int64(fundamental.NetIncome))),
			BVPSFormatted:              fmt.Sprintf("Rp %.2f", fundamental.BVPS),
			SharesOutstandingFormatted: humanizeNum(stock.SharesOutstanding),
			Low52WFormatted:            fmt.Sprintf("Rp %s", humanizeNum(int64(low52))),
			High52WFormatted:           fmt.Sprintf("Rp %s", humanizeNum(int64(high52))),
		}
	}

	historyDays := 0
	if earliest, err := h.StockPriceRepo.GetEarliestDate(stock.ID); err == nil && !earliest.IsZero() {
		historyDays = int(time.Since(earliest).Hours() / 24)
	}

	data := map[string]interface{}{
		"Title":         stock.Code + " — " + stock.Name + " - Investo",
		"Stock":         stock,
		"LatestPrice":   priceData,
		"Fundamentals":  fundSummary,
		"News":          news,
		"Sector":        sector,
		"HistoryDays":   historyDays,
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "stocks/detail.html", data); err != nil {
		log.Printf("StockHandler.Detail: ExecuteTemplate error for %s: %v", code, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *StockHandler) ChartData(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.Error(w, `{"error":"stock not found"}`, http.StatusNotFound)
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

	chartData, err := h.ChartService.GetStockChartData(stock.ID, start, end, indicators)
	if err != nil {
		log.Printf("ChartData: GetStockChartData error for %s: %v", code, err)
		http.Error(w, `{"error":"failed to get chart data"}`, http.StatusInternalServerError)
		return
	}
	if chartData == nil {
		log.Printf("ChartData: nil chartData for %s", code)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chartData); err != nil {
		log.Printf("ChartData: encode error for %s: %v", code, err)
	}
}

func (h *StockHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]model.Stock{})
		return
	}

	stocks, err := h.StockRepo.Search(q, 20)
	if err != nil {
		http.Error(w, `{"error":"search failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stocks)
}

func (h *StockHandler) Fundamentals(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fundamentals, _ := h.StockFundamentalRepo.FindByStockID(stock.ID, 40)
	sector, _ := h.SectorRepo.FindByID(stock.SectorID)

	sectorName := ""
	if sector != nil {
		sectorName = sector.Name
	}

	type FinSummary struct {
		RevenueFormatted      string
		NetIncomeFormatted    string
		EPSFormatted          string
		BVPSFormatted         string
		TotalAssetsFormatted  string
		EquityFormatted       string
	}
	type Ratios struct {
		ROE           float64
		ROA           float64
		PER           float64
		PBV           float64
		DER           float64
		NPM           float64
		DividendYield float64
	}

	var finSummary FinSummary
	var ratios Ratios
	if len(fundamentals) > 0 {
		f := fundamentals[0]
		finSummary = FinSummary{
			RevenueFormatted:     fmt.Sprintf("Rp %s", humanizeNum(int64(f.Revenue))),
			NetIncomeFormatted:   fmt.Sprintf("Rp %s", humanizeNum(int64(f.NetIncome))),
			EPSFormatted:         fmt.Sprintf("Rp %.2f", f.EPS),
			BVPSFormatted:        fmt.Sprintf("Rp %.2f", f.BVPS),
			TotalAssetsFormatted: fmt.Sprintf("Rp %s", humanizeNum(int64(f.TotalAssets))),
			EquityFormatted:      fmt.Sprintf("Rp %s", humanizeNum(int64(f.Equity))),
		}
		ratios = Ratios{
			ROE:           f.ROE,
			ROA:           f.ROA,
			PER:           f.PER,
			PBV:           f.PBV,
			DER:           f.DER,
			NPM:           f.NetProfitMargin,
			DividendYield: f.DividendYield,
		}
	}

	type PeriodRow struct {
		PeriodLabel      string
		RevenueFormatted string
		NetIncome        float64
		NetIncomeFormatted string
		EPSFormatted     string
		BVPSFormatted    string
		ROE              float64
		DER              float64
	}
	var periods []PeriodRow
	for _, f := range fundamentals {
		periods = append(periods, PeriodRow{
			PeriodLabel:        f.Period,
			RevenueFormatted:   fmt.Sprintf("Rp %s", humanizeNum(int64(f.Revenue))),
			NetIncome:          f.NetIncome,
			NetIncomeFormatted: fmt.Sprintf("Rp %s", humanizeNum(int64(f.NetIncome))),
			EPSFormatted:       fmt.Sprintf("%.2f", f.EPS),
			BVPSFormatted:      fmt.Sprintf("%.2f", f.BVPS),
			ROE:                f.ROE,
			DER:                f.DER,
		})
	}

	data := map[string]interface{}{
		"Title":            stock.Code + " - Fundamental - Investo",
		"Stock":            stock,
		"SectorName":        sectorName,
		"Fundamentals":     fundamentals,
		"FinancialSummary": finSummary,
		"Ratios":           ratios,
		"Periods":          periods,
		"CurrentYear":      fmt.Sprintf("%d", time.Now().Year()),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "stocks/fundamental.html", data)
}

func (h *StockHandler) SectorList(w http.ResponseWriter, r *http.Request) {
	sectorSlug := chi.URLParam(r, "sector")

	sector, err := h.SectorRepo.FindBySlug(sectorSlug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stocks, total, _ := h.StockRepo.List(0, 100, "", sector.ID)

	type StockWithPrice struct {
		Stock model.Stock
		Price float64
		Change float64
	}

	var results []StockWithPrice
	for _, s := range stocks {
		prices, _ := h.StockPriceRepo.FindLatest(s.ID, 1)
		var price float64
		if len(prices) > 0 {
			price = prices[0].Close
		}
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 30)
		results = append(results, StockWithPrice{Stock: s, Price: price, Change: change})
	}

	data := map[string]interface{}{
		"Title":    "Saham Terbaik Sektor " + sector.Name + " - Investo",
		"Sector":   sector,
		"Stocks":   results,
		"Total":    total,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "stocks/sector.html", data)
}

func (h *StockHandler) Compare(w http.ResponseWriter, r *http.Request) {
	a := chi.URLParam(r, "a")
	b := chi.URLParam(r, "b")

	stockA, errA := h.StockRepo.FindByCode(a)
	stockB, errB := h.StockRepo.FindByCode(b)

	if errA != nil || errB != nil {
		http.NotFound(w, r)
		return
	}

	fundA, _ := h.StockFundamentalRepo.FindLatest(stockA.ID)
	fundB, _ := h.StockFundamentalRepo.FindLatest(stockB.ID)

	pricesA, _ := h.StockPriceRepo.FindLatest(stockA.ID, 30)
	pricesB, _ := h.StockPriceRepo.FindLatest(stockB.ID, 30)

	var priceA, priceB, prevA, prevB float64
	var volA, volB int64
	if len(pricesA) > 0 {
		priceA = pricesA[0].Close
		volA = pricesA[0].Volume
		if len(pricesA) > 1 {
			prevA = pricesA[1].Close
		}
	}
	if len(pricesB) > 0 {
		priceB = pricesB[0].Close
		volB = pricesB[0].Volume
		if len(pricesB) > 1 {
			prevB = pricesB[1].Close
		}
	}

	changeA := 0.0
	changeAmtA := 0.0
	if prevA > 0 {
		changeA = ((priceA - prevA) / prevA) * 100
		changeAmtA = priceA - prevA
	}
	changeB := 0.0
	changeAmtB := 0.0
	if prevB > 0 {
		changeB = ((priceB - prevB) / prevB) * 100
		changeAmtB = priceB - prevB
	}

	buildLatestPrice := func(price, prev, change, changeAmt float64, vol int64) map[string]interface{} {
		return map[string]interface{}{
			"PriceFormatted":  fmt.Sprintf("Rp %s", humanizeNum(int64(price))),
			"ChangePercent":   change,
			"ChangeFormatted": fmt.Sprintf("Rp %s", humanizeNum(int64(changeAmt))),
			"VolumeFormatted": fmt.Sprintf("%d", vol),
		}
	}

	buildFundSummary := func(stock *model.Stock, fund *model.StockFundamental, price float64) map[string]interface{} {
		if fund == nil {
			return map[string]interface{}{
				"MarketCapFormatted": "Rp 0",
				"PER":                "N/A",
				"PBV":                "N/A",
				"ROE":                "N/A",
				"DividendYield":      "N/A",
				"RevenueFormatted":   "Rp 0",
				"NetIncomeFormatted": "Rp 0",
			}
		}
		marketCap := float64(stock.SharesOutstanding) * price
		return map[string]interface{}{
			"MarketCapFormatted": fmt.Sprintf("Rp %s", humanizeNum(int64(marketCap))),
			"PER":                fmt.Sprintf("%.1fx", fund.PER),
			"PBV":                fmt.Sprintf("%.1fx", fund.PBV),
			"ROE":                fmt.Sprintf("%.1f%%", fund.ROE),
			"DividendYield":      fmt.Sprintf("%.2f%%", fund.DividendYield),
			"RevenueFormatted":   fmt.Sprintf("Rp %s", humanizeNum(int64(fund.Revenue))),
			"NetIncomeFormatted": fmt.Sprintf("Rp %s", humanizeNum(int64(fund.NetIncome))),
		}
	}

	comp := map[string]interface{}{}
	if fundA != nil && fundB != nil {
		marketCapA := float64(stockA.SharesOutstanding) * priceA
		marketCapB := float64(stockB.SharesOutstanding) * priceB
		capDiff := ""
		if marketCapA > marketCapB {
			capDiff = fmt.Sprintf("%s lebih besar", stockA.Code)
		} else if marketCapB > marketCapA {
			capDiff = fmt.Sprintf("%s lebih besar", stockB.Code)
		} else {
			capDiff = "Seimbang"
		}
		comp = map[string]interface{}{
			"MarketCapDiff":    capDiff,
			"PERDiff":          fmt.Sprintf("%.1fx vs %.1fx", fundA.PER, fundB.PER),
			"PERHigher":        fundA.PER > fundB.PER,
			"PBVDiff":          fmt.Sprintf("%.1fx vs %.1fx", fundA.PBV, fundB.PBV),
			"PBVHigher":        fundA.PBV > fundB.PBV,
			"ROEDiff":          fmt.Sprintf("%.1f%% vs %.1f%%", fundA.ROE, fundB.ROE),
			"ROEHigher":        fundA.ROE > fundB.ROE,
			"DivYieldDiff":     fmt.Sprintf("%.2f%% vs %.2f%%", fundA.DividendYield, fundB.DividendYield),
			"DivYieldHigher":   fundA.DividendYield > fundB.DividendYield,
			"RevenueDiff":      fmt.Sprintf("%s vs %s", humanizeNum(int64(fundA.Revenue)), humanizeNum(int64(fundB.Revenue))),
			"NetIncomeDiff":    fmt.Sprintf("%s vs %s", humanizeNum(int64(fundA.NetIncome)), humanizeNum(int64(fundB.NetIncome))),
			"NetIncomeHigher":  fundA.NetIncome > fundB.NetIncome,
		}
	}

	keyDiffs := []map[string]interface{}{
		{
			"Title":  "Sektor & Skala Bisnis",
			"Detail": fmt.Sprintf("%s bergerak di %s. %s bergerak di %s.", stockA.Name, a, stockB.Name, b),
		},
		{
			"Title":  "Harga Saham",
			"Detail": fmt.Sprintf("%s: Rp %s/lembar. %s: Rp %s/lembar.", stockA.Code, humanizeNum(int64(priceA)), stockB.Code, humanizeNum(int64(priceB))),
		},
	}
	if fundA != nil && fundB != nil {
		keyDiffs = append(keyDiffs, map[string]interface{}{
			"Title":  "Profitabilitas",
			"Detail": fmt.Sprintf("ROE %s: %.1f%%. ROE %s: %.1f%%.", stockA.Code, fundA.ROE, stockB.Code, fundB.ROE),
		})
	}

	type stockWithExtras struct {
		*model.Stock
		LatestPrice  map[string]interface{}
		Fundamentals map[string]interface{}
	}

	enrichedData := map[string]interface{}{
		"Title":  stockA.Code + " vs " + stockB.Code + " - Perbandingan Saham - Investo",
		"StockA": stockWithExtras{Stock: stockA, LatestPrice: buildLatestPrice(priceA, prevA, changeA, changeAmtA, volA), Fundamentals: buildFundSummary(stockA, fundA, priceA)},
		"StockB": stockWithExtras{Stock: stockB, LatestPrice: buildLatestPrice(priceB, prevB, changeB, changeAmtB, volB), Fundamentals: buildFundSummary(stockB, fundB, priceB)},
		"Comparison":     comp,
		"KeyDifferences": keyDiffs,
		"CurrentYear":    fmt.Sprintf("%d", time.Now().Year()),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "stocks/compare.html", enrichedData)
}

func humanizeNum(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func (h *StockHandler) LiquidityPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	latestPrices, _ := h.StockPriceRepo.FindLatest(stock.ID, 30)
	var latestPrice float64
	var priceFormatted string
	if len(latestPrices) > 0 {
		latestPrice = latestPrices[0].Close
		priceFormatted = fmt.Sprintf("Rp %s", humanizeNum(int64(latestPrice)))
	} else {
		priceFormatted = "Rp 0"
	}

	data := map[string]interface{}{
		"Title":  stock.Code + " - Liquidity Zones - Investo",
		"Stock":  stock,
		"LatestPrice": map[string]interface{}{
			"PriceFormatted": priceFormatted,
		},
		"User":  safeUser(middleware.GetUser(r)),
	}
	if err := h.Templates.ExecuteTemplate(w, "stocks/liquidity.html", data); err != nil {
		log.Printf("StockHandler.LiquidityPage: ExecuteTemplate error for %s: %v", code, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

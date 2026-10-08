package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"investo/internal/model"
)

// FMPScraper is an optional fallback data provider using Financial Modeling Prep.
// It activates only when FMP_API_KEY is configured.
type FMPScraper struct {
	apiKey string
	client *http.Client
}

func NewFMPScraper() *FMPScraper {
	key := os.Getenv("FMP_API_KEY")
	if key == "" {
		key = os.Getenv("FINANCIALMODELINGPREP_API_KEY")
	}
	return &FMPScraper{
		apiKey: key,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *FMPScraper) IsConfigured() bool {
	return f.apiKey != ""
}

type fmpHistorical struct {
	Symbol     string `json:"symbol"`
	Historical []struct {
		Date     string  `json:"date"`
		Open     float64 `json:"open"`
		High     float64 `json:"high"`
		Low      float64 `json:"low"`
		Close    float64 `json:"close"`
		AdjClose float64 `json:"adjClose"`
		Volume   int64   `json:"volume"`
	} `json:"historical"`
}

// toFMPSymbol converts a Yahoo-style symbol to FMP's format.
func toFMPSymbol(yahooSymbol string) string {
	s := yahooSymbol
	// Crypto: BTC-USD -> BTCUSD
	if strings.Contains(s, "-USD") && !strings.HasSuffix(s, ".JK") {
		return strings.ReplaceAll(s, "-USD", "USD")
	}
	// IDX stocks already use .JK suffix which FMP also understands.
	return s
}

func (f *FMPScraper) FetchHistorical(symbol string, start, end time.Time) ([]model.StockPrice, error) {
	if !f.IsConfigured() {
		return nil, fmt.Errorf("fmp scraper: not configured (FMP_API_KEY empty)")
	}

	fmpSymbol := toFMPSymbol(symbol)
	from := start.Format("2006-01-02")
	to := end.Format("2006-01-02")

	url := fmt.Sprintf(
		"https://financialmodelingprep.com/api/v3/historical-price-full/%s?from=%s&to=%s&apikey=%s",
		fmpSymbol, from, to, f.apiKey,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fmp scraper: create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fmp scraper: fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fmp scraper: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fmp scraper: status %d for %s: %s", resp.StatusCode, symbol, truncate(string(body), 200))
	}

	var data fmpHistorical
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("fmp scraper: parse json: %w", err)
	}

	prices := make([]model.StockPrice, 0, len(data.Historical))
	// FMP returns newest-first; reverse to chronological.
	for i := len(data.Historical) - 1; i >= 0; i-- {
		h := data.Historical[i]
		if h.Close <= 0 {
			continue
		}
		date, err := time.Parse("2006-01-02", h.Date)
		if err != nil {
			continue
		}
		prices = append(prices, model.StockPrice{
			Date:     date.In(jakartaLoc),
			Open:     h.Open,
			High:     h.High,
			Low:      h.Low,
			Close:    h.Close,
			Volume:   h.Volume,
			AdjClose: h.AdjClose,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("fmp scraper: no data for %s", symbol)
	}

	return prices, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

type fmpRatios struct {
	Date                    string  `json:"date"`
	NetProfitMargin         float64 `json:"netProfitMargin"`
	ReturnOnAssets          float64 `json:"returnOnAssets"`
	ReturnOnEquity          float64 `json:"returnOnEquity"`
	DebtEquityRatio         float64 `json:"debtEquityRatio"`
	PriceToBookRatio        float64 `json:"priceToBookRatio"`
	PriceBookValueRatio     float64 `json:"priceBookValueRatio"`
	PriceEarningsRatio      float64 `json:"priceEarningsRatio"`
	DividendYield           float64 `json:"dividendYield"`
	DividendYieldPercentage float64 `json:"dividendYieldPercentage"`
	EPS                     float64 `json:"eps"`
	BookValuePerShare       float64 `json:"bookValuePerShare"`
}

type fmpIncome struct {
	Date      string  `json:"date"`
	Revenue   float64 `json:"revenue"`
	NetIncome float64 `json:"netIncome"`
	EPS       float64 `json:"eps"`
}

// FetchFundamentalsFromFMP pulls real fundamental ratios + income statement from
// Financial Modeling Prep, giving a deterministic alternative to Yahoo estimation.
func (f *FMPScraper) FetchFundamentalsFromFMP(symbol string) (*model.StockFundamental, error) {
	if !f.IsConfigured() {
		return nil, fmt.Errorf("fmp scraper: not configured")
	}
	fmpSymbol := toFMPSymbol(symbol)

	var ratios fmpRatios
	var income fmpIncome

	ratiosURL := fmt.Sprintf("https://financialmodelingprep.com/api/v3/ratios/%s?limit=1&apikey=%s", fmpSymbol, f.apiKey)
	if err := f.getJSON(ratiosURL, &ratios); err != nil {
		return nil, fmt.Errorf("fmp fundamental ratios: %w", err)
	}

	incomeURL := fmt.Sprintf("https://financialmodelingprep.com/api/v3/income-statement/%s?limit=1&apikey=%s", fmpSymbol, f.apiKey)
	// Income statement is optional — derive from ratios if unavailable.
	_ = f.getJSON(incomeURL, &income)

	pbv := ratios.PriceToBookRatio
	if pbv <= 0 {
		pbv = ratios.PriceBookValueRatio
	}

	roe := ratios.ReturnOnEquity
	roa := ratios.ReturnOnAssets
	npm := ratios.NetProfitMargin
	// FMP returns ROE/ROA/NPM as ratios (0.15 = 15%); convert to percent.
	if roe > 0 && roe < 1 {
		roe *= 100
	}
	if roa > 0 && roa < 1 {
		roa *= 100
	}
	if npm > 0 && npm < 1 {
		npm *= 100
	}

	divYield := ratios.DividendYieldPercentage
	if divYield <= 0 {
		divYield = ratios.DividendYield * 100
	}

	eps := ratios.EPS
	if eps <= 0 {
		eps = income.EPS
	}
	revenue := income.Revenue
	netIncome := income.NetIncome
	if netIncome <= 0 && eps > 0 && revenue > 0 {
		netIncome = revenue * (npm / 100)
	}

	// Derive balance-sheet items from ratios (same approach as Yahoo path).
	equity := 0.0
	if roe > 0 && netIncome > 0 {
		equity = netIncome / (roe / 100)
	}
	if equity <= 0 && ratios.BookValuePerShare > 0 && eps > 0 {
		equity = ratios.BookValuePerShare * (netIncome / eps)
	}
	assets := equity * (1 + ratios.DebtEquityRatio)
	liabilities := assets - equity

	return &model.StockFundamental{
		Period:           "FY",
		ReportType:       "annual",
		Source:           "fmp",
		Revenue:          revenue,
		NetIncome:        netIncome,
		EPS:              eps,
		BVPS:             ratios.BookValuePerShare,
		TotalAssets:      assets,
		TotalLiabilities: liabilities,
		Equity:           equity,
		ROE:              roe,
		ROA:              roa,
		PER:              ratios.PriceEarningsRatio,
		PBV:              pbv,
		DER:              ratios.DebtEquityRatio,
		NetProfitMargin:  npm,
		DividendYield:    divYield,
	}, nil
}

func (f *FMPScraper) getJSON(url string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	// FMP returns an array; decode into a slice then take first element.
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty response")
	}
	return json.Unmarshal(raw[0], out)
}

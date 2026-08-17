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

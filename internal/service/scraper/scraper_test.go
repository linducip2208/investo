package scraper

import (
	"testing"
	"time"
)

func TestForexSymbol(t *testing.T) {
	cases := []struct {
		base, quote, want string
	}{
		{"EUR", "USD", "EURUSD=X"},
		{"USD", "IDR", "USDIDR=X"},
		{"BTC", "USD", "BTC-USD"},
		{"ETH", "USD", "ETH-USD"},
		{"SOL", "USD", "SOL-USD"},
	}
	for _, c := range cases {
		if got := forexSymbol(c.base, c.quote); got != c.want {
			t.Errorf("forexSymbol(%s,%s) = %q, want %q", c.base, c.quote, got, c.want)
		}
	}
}

func TestToFMPSymbol(t *testing.T) {
	cases := []struct{ in, want string }{
		{"BTC-USD", "BTCUSD"},
		{"AALI.JK", "AALI.JK"},
		{"EURUSD=X", "EURUSD=X"},
	}
	for _, c := range cases {
		if got := toFMPSymbol(c.in); got != c.want {
			t.Errorf("toFMPSymbol(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFMPScraperNotConfigured(t *testing.T) {
	f := &FMPScraper{apiKey: ""}
	if f.IsConfigured() {
		t.Error("FMPScraper without key should not be configured")
	}
	if _, err := f.FetchHistorical("AAPL", time.Now().AddDate(0, 0, -5), time.Now()); err == nil {
		t.Error("FMPScraper without key should return error on fetch")
	}
}

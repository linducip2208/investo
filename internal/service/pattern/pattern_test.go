package pattern

import (
	"testing"
	"time"

	"investo/internal/model"
)

func TestDoji(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now(), Open: 1000, High: 1100, Low: 900, Close: 1000},
	}

	result := detectDoji(prices)
	if result == nil {
		t.Fatal("expected Doji pattern detected")
	}
	if result.Name != "Doji" {
		t.Errorf("expected name 'Doji', got '%s'", result.Name)
	}
	if result.Type != "neutral" {
		t.Errorf("expected type 'neutral', got '%s'", result.Type)
	}
}

func TestDojiNotFound(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now(), Open: 1000, High: 1500, Low: 900, Close: 1400},
	}

	result := detectDoji(prices)
	if result != nil {
		t.Errorf("expected no Doji, got %v", result)
	}
}

func TestBullishEngulfing(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now().Add(-24 * time.Hour), Open: 1100, High: 1150, Low: 900, Close: 950},
		{Date: time.Now(), Open: 940, High: 1200, Low: 920, Close: 1150},
	}

	result := detectBullishEngulfing(prices)
	if result == nil {
		t.Fatal("expected Bullish Engulfing pattern detected")
	}
	if result.Name != "Bullish Engulfing" {
		t.Errorf("expected 'Bullish Engulfing', got '%s'", result.Name)
	}
	if result.Type != "bullish" {
		t.Errorf("expected type 'bullish', got '%s'", result.Type)
	}
	if result.Reliability != 5 {
		t.Errorf("expected reliability 5, got %d", result.Reliability)
	}
}

func TestBearishEngulfing(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now().Add(-24 * time.Hour), Open: 900, High: 1150, Low: 890, Close: 1100},
		{Date: time.Now(), Open: 1110, High: 1120, Low: 850, Close: 880},
	}

	result := detectBearishEngulfing(prices)
	if result == nil {
		t.Fatal("expected Bearish Engulfing pattern detected")
	}
	if result.Name != "Bearish Engulfing" {
		t.Errorf("expected 'Bearish Engulfing', got '%s'", result.Name)
	}
	if result.Type != "bearish" {
		t.Errorf("expected type 'bearish', got '%s'", result.Type)
	}
}

func TestEngulfingNotFound(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now().Add(-24 * time.Hour), Open: 900, High: 950, Low: 890, Close: 950},
		{Date: time.Now(), Open: 960, High: 1000, Low: 940, Close: 980},
	}

	result := detectBullishEngulfing(prices)
	if result != nil {
		t.Errorf("expected no Bullish Engulfing on all-bullish, got %v", result)
	}

	result = detectBearishEngulfing(prices)
	if result != nil {
		t.Errorf("expected no Bearish Engulfing on all-bullish, got %v", result)
	}
}

func TestDetectAll(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now().Add(-72 * time.Hour), Open: 1100, High: 1150, Low: 900, Close: 950},
		{Date: time.Now().Add(-48 * time.Hour), Open: 1000, High: 1050, Low: 990, Close: 1005},
		{Date: time.Now().Add(-24 * time.Hour), Open: 1000, High: 1100, Low: 900, Close: 1000},
		{Date: time.Now(), Open: 940, High: 1200, Low: 920, Close: 1150},
	}

	s := &PatternService{}
	results := s.DetectAll(prices)

	t.Logf("Detected %d patterns:", len(results))
	for _, r := range results {
		t.Logf("  %s (type=%s, reliability=%d)", r.Name, r.Type, r.Reliability)
	}

	if len(results) == 0 {
		t.Error("expected at least one pattern detected")
	}
}

func TestDetectAllEmpty(t *testing.T) {
	prices := []model.StockPrice{
		{Date: time.Now(), Open: 1000, High: 1000, Low: 1000, Close: 1000},
	}

	s := &PatternService{}
	results := s.DetectAll(prices)

	if len(results) != 0 {
		t.Errorf("expected no patterns on single candle, got %d", len(results))
	}
}

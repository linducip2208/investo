package service

import (
	"testing"
	"time"

	"investo/internal/model"
)

func TestFilterValidPrices(t *testing.T) {
	now := time.Now()
	past := now.AddDate(0, 0, -5)
	future := now.AddDate(0, 0, 5)

	prices := []model.StockPrice{
		{Date: past, Close: 100},
		{Date: past.AddDate(0, 0, 1), Close: 0}, // zero close -> filtered
		{Date: past.AddDate(0, 0, 2), Close: 105},
		{Date: future, Close: 110}, // future -> filtered
		{Date: past.AddDate(0, 0, 3), Close: 108},
	}

	got := filterValidPrices(prices)
	if len(got) != 3 {
		t.Fatalf("expected 3 valid prices, got %d", len(got))
	}
	if got[0].Close != 100 || got[1].Close != 105 || got[2].Close != 108 {
		t.Errorf("unexpected filtered result: %+v", got)
	}
}

func TestFilterValidPricesEmpty(t *testing.T) {
	if got := filterValidPrices(nil); len(got) != 0 {
		t.Errorf("nil input should return empty, got %d", len(got))
	}
}

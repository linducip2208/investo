package service

import (
	"fmt"
	"math"
	"strings"
)

var centralBankRates = map[string]float64{
	"USD": 5.50,
	"EUR": 3.75,
	"JPY": 0.50,
	"GBP": 5.25,
	"AUD": 4.35,
	"NZD": 5.50,
	"CAD": 4.75,
	"CHF": 1.75,
	"IDR": 5.75,
	"SGD": 3.50,
}

type SwapResult struct {
	Pair              string  `json:"pair"`
	BaseCurrency      string  `json:"base_currency"`
	QuoteCurrency     string  `json:"quote_currency"`
	Direction         string  `json:"direction"`
	PositionSize      float64 `json:"position_size"`
	BaseRate          float64 `json:"base_rate"`
	QuoteRate         float64 `json:"quote_rate"`
	DailySwap         float64 `json:"daily_swap"`
	AnnualSwap        float64 `json:"annual_swap"`
	InterestDiff      float64 `json:"interest_differential"`
}

type MarginResult struct {
	Pair             string  `json:"pair"`
	PositionSize     float64 `json:"position_size"`
	Leverage         int     `json:"leverage"`
	ContractSize     float64 `json:"contract_size"`
	RequiredMargin   float64 `json:"required_margin"`
	MarginPercent    float64 `json:"margin_percent"`
	AccountBalance   float64 `json:"account_balance"`
	FreeMargin       float64 `json:"free_margin"`
	MarginCallLevel  float64 `json:"margin_call_level"`
	MarginCallPrice  float64 `json:"margin_call_price"`
	StopOutPrice     float64 `json:"stop_out_price"`
}

type ForexTools struct{}

func NewForexTools() *ForexTools {
	return &ForexTools{}
}

func (t *ForexTools) CalcSwap(baseCurrency, quoteCurrency string, positionSize float64, direction string) (*SwapResult, error) {
	baseCurrency = strings.ToUpper(baseCurrency)
	quoteCurrency = strings.ToUpper(quoteCurrency)
	direction = strings.ToUpper(direction)

	baseRate, ok := centralBankRates[baseCurrency]
	if !ok {
		return nil, fmt.Errorf("suku bunga untuk %s tidak tersedia", baseCurrency)
	}
	quoteRate, ok := centralBankRates[quoteCurrency]
	if !ok {
		return nil, fmt.Errorf("suku bunga untuk %s tidak tersedia", quoteCurrency)
	}

	pair := baseCurrency + "/" + quoteCurrency
	interestDiff := baseRate - quoteRate

	var dailySwap float64
	switch direction {
	case "LONG":
		dailySwap = (positionSize * interestDiff / 100) / 365
	case "SHORT":
		dailySwap = (positionSize * -interestDiff / 100) / 365
	default:
		return nil, fmt.Errorf("direction harus LONG atau SHORT")
	}

	return &SwapResult{
		Pair:         pair,
		BaseCurrency: baseCurrency,
		QuoteCurrency: quoteCurrency,
		Direction:    direction,
		PositionSize: positionSize,
		BaseRate:     baseRate,
		QuoteRate:    quoteRate,
		DailySwap:    math.Round(dailySwap*100) / 100,
		AnnualSwap:   math.Round(dailySwap*365*100) / 100,
		InterestDiff: math.Round(interestDiff*100) / 100,
	}, nil
}

type SwapPairRate struct {
	Pair         string  `json:"pair"`
	BaseRate     float64 `json:"base_rate"`
	QuoteRate    float64 `json:"quote_rate"`
	DailyLong    float64 `json:"daily_long"`
	AnnualLong   float64 `json:"annual_long"`
	DailyShort   float64 `json:"daily_short"`
	AnnualShort  float64 `json:"annual_short"`
	Spread       float64 `json:"spread"`
}

func (t *ForexTools) GetAllSwapRates(standardLot float64) []SwapPairRate {
	currencies := []string{"USD", "EUR", "JPY", "GBP", "AUD", "NZD", "CAD", "CHF", "IDR", "SGD"}
	var rates []SwapPairRate
	checked := make(map[string]bool)

	for _, base := range currencies {
		for _, quote := range currencies {
			if base == quote {
				continue
			}
			key := base + "/" + quote
			revKey := quote + "/" + base
			if checked[key] || checked[revKey] {
				continue
			}
			checked[key] = true

			baseRate := centralBankRates[base]
			quoteRate := centralBankRates[quote]
			spread := baseRate - quoteRate
			dailyLong := (standardLot * spread / 100) / 365
			dailyShort := (standardLot * -spread / 100) / 365

			rates = append(rates, SwapPairRate{
				Pair:        key,
				BaseRate:    baseRate,
				QuoteRate:   quoteRate,
				DailyLong:   math.Round(dailyLong*100) / 100,
				AnnualLong:  math.Round(dailyLong*365*100) / 100,
				DailyShort:  math.Round(dailyShort*100) / 100,
				AnnualShort: math.Round(dailyShort*365*100) / 100,
				Spread:      math.Round(spread*100) / 100,
			})
		}
	}
	return rates
}

func (t *ForexTools) CalcMargin(pair string, positionSize float64, leverage int) (*MarginResult, error) {
	if leverage <= 0 {
		return nil, fmt.Errorf("leverage tidak boleh 0")
	}
	contractSize := 100000.0
	if strings.Contains(pair, "JPY") || strings.Contains(pair, "IDR") {
		contractSize = 10000000.0
	}
	requiredMargin := (positionSize * contractSize) / float64(leverage)
	accountBalance := requiredMargin * 3
	freeMargin := accountBalance - requiredMargin
	marginPercent := (requiredMargin / accountBalance) * 100
	marginCallLevel := requiredMargin * 0.5
	marginCallPrice := 0.0
	stopOutPrice := requiredMargin * 0.2

	return &MarginResult{
		Pair:            pair,
		PositionSize:    positionSize,
		Leverage:        leverage,
		ContractSize:    contractSize,
		RequiredMargin:  math.Round(requiredMargin*100) / 100,
		MarginPercent:   math.Round(marginPercent*100) / 100,
		AccountBalance:  math.Round(accountBalance*100) / 100,
		FreeMargin:      math.Round(freeMargin*100) / 100,
		MarginCallLevel: math.Round(marginCallLevel*100) / 100,
		MarginCallPrice: math.Round(marginCallPrice*100) / 100,
		StopOutPrice:    math.Round(stopOutPrice*100) / 100,
	}, nil
}

func (t *ForexTools) CalcPipValue(pair string, lotSize float64, accountCurrency string) (float64, error) {
	pipSize := 0.0001
	if strings.Contains(pair, "JPY") {
		pipSize = 0.01
	}
	if strings.Contains(pair, "IDR") {
		pipSize = 1.0
	}

	parts := strings.Split(pair, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("format pair tidak valid: %s", pair)
	}

	quoteCurrency := parts[1]
	contractSize := 100000.0
	if strings.Contains(pair, "JPY") || strings.Contains(pair, "IDR") {
		contractSize = 10000000.0
	}

	pipValueInQuote := lotSize * contractSize * pipSize

	if strings.ToUpper(accountCurrency) == quoteCurrency {
		return math.Round(pipValueInQuote*100) / 100, nil
	}

	if rate, ok := centralBankRates[quoteCurrency]; ok {
		if accRate, ok2 := centralBankRates[strings.ToUpper(accountCurrency)]; ok2 && accRate > 0 {
			converted := pipValueInQuote * (rate / accRate)
			return math.Round(converted*100) / 100, nil
		}
	}

	return math.Round(pipValueInQuote*100) / 100, nil
}

type PipRef struct {
	Pair      string  `json:"pair"`
	PipSize   float64 `json:"pip_size"`
	PipValue  float64 `json:"pip_value"`
	Value10   float64 `json:"value_10_pip"`
	Value100  float64 `json:"value_100_pip"`
}

func (t *ForexTools) GetAllPipReferences(lotSize float64, accountCurrency string) []PipRef {
	pairs := []string{
		"EUR/USD", "USD/JPY", "GBP/USD", "USD/CHF", "AUD/USD",
		"USD/CAD", "NZD/USD", "EUR/JPY", "GBP/JPY", "EUR/GBP",
		"USD/IDR", "EUR/IDR", "SGD/IDR", "AUD/IDR", "USD/SGD",
	}
	var refs []PipRef
	for _, pair := range pairs {
		pipSize := 0.0001
		if strings.Contains(pair, "JPY") {
			pipSize = 0.01
		}
		if strings.Contains(pair, "IDR") {
			pipSize = 1.0
		}
		pv, _ := t.CalcPipValue(pair, lotSize, accountCurrency)
		refs = append(refs, PipRef{
			Pair:     pair,
			PipSize:  pipSize,
			PipValue: pv,
			Value10:  math.Round(pv*10*100) / 100,
			Value100: math.Round(pv*100*100) / 100,
		})
	}
	return refs
}

package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service/indicator"
)

func cleanNaN(s []float64) {
	for i := range s {
		if math.IsNaN(s[i]) || math.IsInf(s[i], 0) {
			s[i] = 0
		}
	}
}

type ChartResponse struct {
	Dates  []string           `json:"dates"`
	Prices []OHLC             `json:"prices"`
	Volume []int64            `json:"volume"`
	SMA    map[int][]float64  `json:"sma"`
	EMA    map[int][]float64  `json:"ema"`
	RSI    map[int][]float64  `json:"rsi"`
	MACD   *MACDData          `json:"macd"`
	BB     *BBData            `json:"bb"`
}

type OHLC struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

type MACDData struct {
	MACDLine   []float64 `json:"macd_line"`
	SignalLine []float64 `json:"signal_line"`
	Histogram  []float64 `json:"histogram"`
}

type BBData struct {
	Upper  []float64 `json:"upper"`
	Middle []float64 `json:"middle"`
	Lower  []float64 `json:"lower"`
}

type ChartService struct {
	StockPriceRepo *repository.StockPriceRepository
}

func (s *ChartService) GetChartData(stockID int64, start, end time.Time) (ChartResponse, error) {
	prices, err := s.StockPriceRepo.FindByStockDate(stockID, start, end)
	if err != nil {
		return ChartResponse{}, fmt.Errorf("ChartService.GetChartData: %w", err)
	}

	prices = filterValidPrices(prices)

	if len(prices) == 0 {
		latest, err := s.StockPriceRepo.FindLatest(stockID, 365)
		if err != nil {
			return ChartResponse{}, fmt.Errorf("ChartService.GetChartData fallback: %w", err)
		}
		ReversePrices(latest)
		prices = filterValidPrices(latest)
	}

	return s.buildChartResponse(prices)
}

func (s *ChartService) GetStockChartData(stockID int64, start, end time.Time, indicators []string) (*ChartResponse, error) {
	prices, err := s.StockPriceRepo.FindByStockDate(stockID, start, end)
	if err != nil {
		return nil, fmt.Errorf("ChartService.GetStockChartData: %w", err)
	}

	prices = filterValidPrices(prices)

	if len(prices) == 0 {
		latest, err := s.StockPriceRepo.FindLatest(stockID, 365)
		if err != nil {
			return nil, fmt.Errorf("ChartService.GetStockChartData fallback: %w", err)
		}
		ReversePrices(latest)
		prices = filterValidPrices(latest)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("ChartService.GetStockChartData: no price data for stock %d", stockID)
	}

	n := len(prices)
	dates := make([]string, n)
	ohlc := make([]OHLC, n)
	volumes := make([]int64, n)
	closePrices := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		ohlc[i] = OHLC{Open: p.Open, High: p.High, Low: p.Low, Close: p.Close}
		volumes[i] = p.Volume
		closePrices[i] = p.Close
	}

	resp := &ChartResponse{
		Dates:  dates,
		Prices: ohlc,
		Volume: volumes,
	}

	includeIndicator := func(name string) bool {
		if len(indicators) == 0 {
			return true
		}
		for _, ind := range indicators {
			if ind == name || ind == "all" {
				return true
			}
		}
		return false
	}

	if includeIndicator("sma") {
		resp.SMA = map[int][]float64{
			20:  indicator.CalcSMA(closePrices, 20),
			50:  indicator.CalcSMA(closePrices, 50),
			200: indicator.CalcSMA(closePrices, 200),
		}
	}

	if includeIndicator("ema") {
		resp.EMA = map[int][]float64{
			12: indicator.CalcEMA(closePrices, 12),
			26: indicator.CalcEMA(closePrices, 26),
		}
	}

	if includeIndicator("rsi") {
		resp.RSI = map[int][]float64{
			14: indicator.CalcRSI(closePrices, 14),
		}
	}

	if includeIndicator("macd") {
		macdLine, signalLine, hist := indicator.CalcMACD(closePrices, 12, 26, 9)
		resp.MACD = &MACDData{
			MACDLine:   macdLine,
			SignalLine: signalLine,
			Histogram:  hist,
		}
	}

	if includeIndicator("bb") || includeIndicator("bollinger") {
		upper, middle, lower := indicator.CalcBollingerBands(closePrices, 20, 2.0)
		resp.BB = &BBData{
			Upper:  upper,
			Middle: middle,
			Lower:  lower,
		}
	}

	for _, v := range resp.SMA {
		cleanNaN(v)
	}
	for _, v := range resp.EMA {
		cleanNaN(v)
	}
	for _, v := range resp.RSI {
		cleanNaN(v)
	}
	if resp.MACD != nil {
		cleanNaN(resp.MACD.MACDLine)
		cleanNaN(resp.MACD.SignalLine)
		cleanNaN(resp.MACD.Histogram)
	}
	if resp.BB != nil {
		cleanNaN(resp.BB.Upper)
		cleanNaN(resp.BB.Middle)
		cleanNaN(resp.BB.Lower)
	}

	return resp, nil
}

func (s *ChartService) buildChartResponse(prices []model.StockPrice) (ChartResponse, error) {
	n := len(prices)
	if n == 0 {
		return ChartResponse{}, fmt.Errorf("ChartService.buildChartResponse: no data")
	}

	dates := make([]string, n)
	ohlc := make([]OHLC, n)
	volumes := make([]int64, n)
	closePrices := make([]float64, n)

	for i, p := range prices {
		dates[i] = p.Date.Format("2006-01-02")
		ohlc[i] = OHLC{Open: p.Open, High: p.High, Low: p.Low, Close: p.Close}
		volumes[i] = p.Volume
		closePrices[i] = p.Close
	}

	sma := map[int][]float64{
		20:  indicator.CalcSMA(closePrices, 20),
		50:  indicator.CalcSMA(closePrices, 50),
		200: indicator.CalcSMA(closePrices, 200),
	}

	ema := map[int][]float64{
		12: indicator.CalcEMA(closePrices, 12),
		26: indicator.CalcEMA(closePrices, 26),
	}

	rsi := map[int][]float64{
		14: indicator.CalcRSI(closePrices, 14),
	}

	macdLine, signalLine, hist := indicator.CalcMACD(closePrices, 12, 26, 9)

	upper, middle, lower := indicator.CalcBollingerBands(closePrices, 20, 2.0)

	return ChartResponse{
		Dates:  dates,
		Prices: ohlc,
		Volume: volumes,
		SMA:    sma,
		EMA:    ema,
		RSI:    rsi,
		MACD: &MACDData{
			MACDLine:   macdLine,
			SignalLine: signalLine,
			Histogram:  hist,
		},
		BB: &BBData{
			Upper:  upper,
			Middle: middle,
			Lower:  lower,
		},
	}, nil
}

func ReversePrices(prices []model.StockPrice) {
	for i, j := 0, len(prices)-1; i < j; i, j = i+1, j-1 {
		prices[i], prices[j] = prices[j], prices[i]
	}
}

// filterValidPrices removes zero/placeholder rows and future-dated rows so charts
// and indicators are computed only on real, non-look-ahead data.
func filterValidPrices(prices []model.StockPrice) []model.StockPrice {
	now := time.Now()
	out := make([]model.StockPrice, 0, len(prices))
	for _, p := range prices {
		if p.Close <= 0 {
			continue
		}
		if p.Date.After(now) {
			continue
		}
		out = append(out, p)
	}
	return out
}

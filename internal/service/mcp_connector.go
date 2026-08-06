package service

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

type MCPConnector struct {
	ServerURL string
	APIKey    string
	client    *http.Client
}

type OHLCV struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type ValidationResult struct {
	Code           string  `json:"code"`
	Accuracy       float64 `json:"accuracy"`
	TotalSignals   int     `json:"total_signals"`
	CorrectSignals int     `json:"correct_signals"`
	AvgReturn      float64 `json:"avg_return"`
	TestedDays     int     `json:"tested_days"`
	Message        string  `json:"message"`
}

func NewMCPConnector(serverURL, apiKey string) *MCPConnector {
	return &MCPConnector{
		ServerURL: serverURL,
		APIKey:    apiKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MCPConnector) IsConfigured() bool {
	return m.ServerURL != "" && m.APIKey != ""
}

func (m *MCPConnector) FetchHistoricalData(code string, start, end time.Time) ([]OHLCV, error) {
	if !m.IsConfigured() {
		return m.fetchYahooFallback(code, start, end)
	}

	url := fmt.Sprintf("%s/historical?code=%s&start=%s&end=%s",
		m.ServerURL, code, start.Format("2006-01-02"), end.Format("2006-01-02"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return m.fetchYahooFallback(code, start, end)
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return m.fetchYahooFallback(code, start, end)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return m.fetchYahooFallback(code, start, end)
	}

	body, _ := io.ReadAll(resp.Body)
	var ohlcv []OHLCV
	if err := json.Unmarshal(body, &ohlcv); err != nil {
		return nil, fmt.Errorf("parse MCP response: %w", err)
	}

	return ohlcv, nil
}

func (m *MCPConnector) fetchYahooFallback(code string, start, end time.Time) ([]OHLCV, error) {
	return nil, fmt.Errorf("MCP server tidak tersedia dan Yahoo Finance fallback belum dikonfigurasi. Code: %s", code)
}

func (m *MCPConnector) ValidateSignal(code string, signalStrength float64) (*ValidationResult, error) {
	end := time.Now()
	start := end.AddDate(0, -6, 0)

	history, err := m.FetchHistoricalData(code, start, end)
	if err != nil || len(history) < 30 {
		return &ValidationResult{
			Code:         code,
			Message:      "Data historis tidak mencukupi untuk validasi. Minimal 30 hari data diperlukan.",
			TestedDays:   len(history),
		}, nil
	}

	var totalSignals, correctSignals int
	var returns []float64

	for i := 14; i < len(history)-5; i++ {
		sma := calcSMA(history, i, 14)
		rsi := calcRSI(history, i, 14)
		current := history[i].Close

		var signal string
		if current > sma*1.02 && rsi > 50 && rsi < 70 {
			signal = "BUY"
		} else if current < sma*0.98 && rsi < 50 {
			signal = "SELL"
		} else {
			continue
		}

		totalSignals++
		future := history[minInt(i+5, len(history)-1)].Close
		ret := (future - current) / current * 100

		if (signal == "BUY" && ret > 0) || (signal == "SELL" && ret < 0) {
			correctSignals++
		}
		returns = append(returns, ret)
	}

	var accuracy, avgReturn float64
	if totalSignals > 0 {
		accuracy = float64(correctSignals) / float64(totalSignals) * 100
	}

	if len(returns) > 0 {
		var sum float64
		for _, r := range returns {
			sum += r
		}
		avgReturn = sum / float64(len(returns))
	}

	msg := fmt.Sprintf("Validasi %s selesai. Akurasi sinyal: %.1f%% (%d/%d) selama %d hari. Avg return: %.2f%%.",
		code, accuracy, correctSignals, totalSignals, len(history), avgReturn)

	return &ValidationResult{
		Code:           code,
		Accuracy:       math.Round(accuracy*10) / 10,
		TotalSignals:   totalSignals,
		CorrectSignals: correctSignals,
		AvgReturn:      math.Round(avgReturn*100) / 100,
		TestedDays:     len(history),
		Message:        msg,
	}, nil
}

func calcSMA(data []OHLCV, currentIdx, period int) float64 {
	if currentIdx < period {
		return 0
	}
	var sum float64
	for i := currentIdx - period + 1; i <= currentIdx; i++ {
		sum += data[i].Close
	}
	return sum / float64(period)
}

func calcRSI(data []OHLCV, currentIdx, period int) float64 {
	if currentIdx < period+1 {
		return 50
	}
	var gains, losses float64
	for i := currentIdx - period + 1; i <= currentIdx; i++ {
		change := data[i].Close - data[i-1].Close
		if change >= 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

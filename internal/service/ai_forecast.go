package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type PriceForecast struct {
	Code       string  `json:"code"`
	Low70      float64 `json:"low_70"`
	High70     float64 `json:"high_70"`
	Low90      float64 `json:"low_90"`
	High90     float64 `json:"high_90"`
	Direction  string  `json:"direction"`
	Confidence float64 `json:"confidence"`
	Rationale  string  `json:"rationale"`
	ForecastDays int   `json:"forecast_days"`
}

type DividendPrediction struct {
	Code       string  `json:"code"`
	Prediction string  `json:"prediction"`
	Confidence float64 `json:"confidence"`
	Rationale  string  `json:"rationale"`
	KeyFactors []string `json:"key_factors"`
}

type RiskAlert struct {
	Code     string  `json:"code"`
	RiskType string  `json:"risk_type"`
	Severity string  `json:"severity"`
	Message  string  `json:"message"`
	Score    float64 `json:"score"`
	Timestamp string `json:"timestamp"`
}

type AIForecastService struct {
	AI                   *AIService
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
}

func NewAIForecastService(
	ai *AIService,
	stockRepo *repository.StockRepository,
	stockPriceRepo *repository.StockPriceRepository,
	stockFundamentalRepo *repository.StockFundamentalRepository,
	sectorRepo *repository.SectorRepository,
) *AIForecastService {
	return &AIForecastService{
		AI:                   ai,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
	}
}

func (s *AIForecastService) ForecastPriceRange(code string, days int) (*PriceForecast, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	if days <= 0 {
		days = 30
	}

	currentPrice, _ := s.StockPriceRepo.GetLatestPrice(stock.ID)
	prices, _ := s.StockPriceRepo.FindLatest(stock.ID, 90)

	if currentPrice == 0 {
		currentPrice = 1000
	}

	var volatility float64 = 0.02
	if len(prices) >= 20 {
		volatility = calcHistoricalVolatility(prices, 20)
	}

	z70 := 1.036
	z90 := 1.645

	annualVol := volatility * math.Sqrt(252)
	periodVol := annualVol * math.Sqrt(float64(days)/252)

	up70 := currentPrice * (1 + periodVol*z70)
	down70 := currentPrice * (1 - periodVol*z70)
	up90 := currentPrice * (1 + periodVol*z90)
	down90 := currentPrice * (1 - periodVol*z90)

	direction := "sideways"
	sma20 := calcSMAFromPrices(prices, 20)
	if len(prices) >= 20 && sma20 > 0 {
		trendStrength := (currentPrice - sma20) / sma20
		if trendStrength > 0.03 {
			direction = "bullish"
		} else if trendStrength < -0.03 {
			direction = "bearish"
		}
	}

	var directionConfidence float64 = 50
	if direction == "bullish" {
		directionConfidence = 65
	} else if direction == "bearish" {
		directionConfidence = 60
	}

	rationale := fmt.Sprintf("Forecast %d hari ke depan untuk %s (%s) berdasarkan volatilitas historis %.1f%%. Range 70%%: Rp %.0f - Rp %.0f. Range 90%%: Rp %.0f - Rp %.0f. Arah: %s.",
		days, stock.Code, stock.Name, volatility*100, down70, up70, down90, up90, direction)

	if s.AI.IsConfigured() {
		sysPrompt := fmt.Sprintf(`Kamu adalah quantitative analyst untuk pasar saham Indonesia. Buat rationale forecast harga %d hari ke depan untuk %s (%s).
Harga saat ini: Rp %.0f
Volatilitas: %.1f%%
Range 70%%: Rp %.0f - Rp %.0f
Range 90%%: Rp %.0f - Rp %.0f
Arah: %s

Buat rationale 2-3 kalimat dalam bahasa Indonesia. Output JSON: {"rationale": "..."}`, days, stock.Name, stock.Code, currentPrice, volatility*100, down70, up70, down90, up90, direction)

		response, err := s.AI.Chat(sysPrompt, "Buat rationale forecast harga.")
		if err == nil {
			jsonStr := extractJSON(response)
			var parsed struct {
				Rationale string `json:"rationale"`
			}
			if json.Unmarshal([]byte(jsonStr), &parsed) == nil && parsed.Rationale != "" {
				rationale = parsed.Rationale
			}
		}
	}

	return &PriceForecast{
		Code:         code,
		Low70:        math.Round(down70),
		High70:       math.Round(up70),
		Low90:        math.Round(down90),
		High90:       math.Round(up90),
		Direction:    direction,
		Confidence:   directionConfidence,
		Rationale:    rationale,
		ForecastDays: days,
	}, nil
}

func (s *AIForecastService) DetectDividendChange(code string) (*DividendPrediction, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	funds, _ := s.StockFundamentalRepo.FindByStockID(stock.ID, 4)
	price, _ := s.StockPriceRepo.GetLatestPrice(stock.ID)

	prediction := "stable"
	confidence := 50.0
	var keyFactors []string

	if len(funds) == 0 {
		prediction = "tidak_diketahui"
		confidence = 0
		keyFactors = []string{"Data fundamental tidak tersedia"}
	} else {
		latest := funds[0]
		var payoutRatio float64
		if latest.EPS > 0 && price > 0 {
			if latest.DividendYield > 0 {
				dps := latest.DividendYield / 100 * price
				payoutRatio = dps / latest.EPS * 100
			}
		}

		if payoutRatio > 0 {
			if payoutRatio < 40 && latest.ROE > 10 {
				prediction = "likely_increase"
				confidence = 70
				keyFactors = append(keyFactors, fmt.Sprintf("Payout ratio rendah (%.0f%%)", payoutRatio))
			} else if payoutRatio > 80 {
				prediction = "likely_cut"
				confidence = 65
				keyFactors = append(keyFactors, fmt.Sprintf("Payout ratio tinggi (%.0f%%)", payoutRatio))
			} else {
				prediction = "stable"
				confidence = 55
				keyFactors = append(keyFactors, fmt.Sprintf("Payout ratio normal (%.0f%%)", payoutRatio))
			}
		}

		if latest.ROE > 15 {
			keyFactors = append(keyFactors, fmt.Sprintf("ROE kuat %.1f%%", latest.ROE))
		}
		if latest.DER < 1.5 {
			keyFactors = append(keyFactors, fmt.Sprintf("DER sehat %.2fx", latest.DER))
		}
		if latest.DER > 3 {
			keyFactors = append(keyFactors, fmt.Sprintf("DER tinggi %.2fx — risiko", latest.DER))
		}
		if latest.DividendYield > 0 {
			keyFactors = append(keyFactors, fmt.Sprintf("Dividend yield saat ini %.2f%%", latest.DividendYield))
		}
	}

	rationale := fmt.Sprintf("Prediksi dividen %s: %s dengan confidence %.0f%% berdasarkan analisis fundamental terkini.", stock.Code, predictionToLabel(prediction), confidence)

	if s.AI.IsConfigured() {
		sysPrompt := fmt.Sprintf(`Kamu adalah analis dividen untuk pasar saham Indonesia. Prediksi perubahan dividen untuk %s (%s).
Payout ratio: %.0f%%. ROE: %.1f%%. DER: %.2fx. Dividend yield: %.2f%%.

Beri prediksi: "likely_increase", "stable", atau "likely_cut". Output JSON: {"prediction": "...", "rationale": "2-3 kalimat bahasa Indonesia"}`, stock.Name, stock.Code, 0.0, 0.0, 0.0, 0.0)

		if len(funds) > 0 {
			sysPrompt = fmt.Sprintf(`Kamu adalah analis dividen untuk pasar saham Indonesia. Prediksi perubahan dividen untuk %s (%s).
Berdasarkan data fundamental terbaru, berikan prediksi: "likely_increase", "stable", atau "likely_cut".
Output JSON: {"prediction": "...", "rationale": "2-3 kalimat bahasa Indonesia menjelaskan mengapa"}`, stock.Name, stock.Code)
		}

		response, err := s.AI.Chat(sysPrompt, fmt.Sprintf("Prediksi dividen %s.", stock.Code))
		if err == nil {
			jsonStr := extractJSON(response)
			var parsed struct {
				Prediction string `json:"prediction"`
				Rationale  string `json:"rationale"`
			}
			if json.Unmarshal([]byte(jsonStr), &parsed) == nil {
				if parsed.Prediction != "" {
					prediction = parsed.Prediction
				}
				if parsed.Rationale != "" {
					rationale = parsed.Rationale
				}
			}
		}
	}

	if len(keyFactors) == 0 {
		keyFactors = append(keyFactors, "Evaluasi berbasis data fundamental")
	}

	return &DividendPrediction{
		Code:       code,
		Prediction: prediction,
		Confidence: math.Round(confidence*10) / 10,
		Rationale:  rationale,
		KeyFactors: keyFactors,
	}, nil
}

func (s *AIForecastService) ScanBlackSwanRisks() ([]RiskAlert, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil || len(stocks) == 0 {
		return nil, fmt.Errorf("no active stocks found")
	}

	var alerts []RiskAlert
	now := time.Now().Format(time.RFC3339)

	for _, stock := range stocks[:minInt(30, len(stocks))] {
		prices, err := s.StockPriceRepo.FindLatest(stock.ID, 20)
		if err != nil || len(prices) < 10 {
			continue
		}

		currentPrice := prices[0].Close
		avgPrice20 := calcSMAFromPrices(prices, 20)

		if avgPrice20 > 0 {
			deviation := math.Abs(currentPrice-avgPrice20) / avgPrice20
			if deviation > 0.08 {
				severity := "medium"
				score := 60.0
				if deviation > 0.15 {
					severity = "high"
					score = 80.0
				}
				direction := "naik"
				if currentPrice < avgPrice20 {
					direction = "turun"
				}
				alerts = append(alerts, RiskAlert{
					Code:      stock.Code,
					RiskType:  "extreme_volatility",
					Severity:  severity,
					Message:   fmt.Sprintf("%s (%s): Pergerakan harga ekstrem %s %.0f%% dari rata-rata 20 hari. Volatilitas tidak normal terdeteksi.", stock.Code, stock.Name, direction, deviation*100),
					Score:     score,
					Timestamp: now,
				})
			}
		}

		if len(prices) >= 5 {
			avgVol := int64(0)
			for _, p := range prices {
				avgVol += p.Volume
			}
			avgVol /= int64(len(prices))
			if avgVol > 0 && prices[0].Volume > avgVol*3 {
				alerts = append(alerts, RiskAlert{
					Code:      stock.Code,
					RiskType:  "unusual_volume",
					Severity:  "medium",
					Message:   fmt.Sprintf("%s (%s): Lonjakan volume tidak wajar %dx dari rata-rata. Mungkin ada akumulasi atau distribusi besar.", stock.Code, stock.Name, prices[0].Volume/avgVol),
					Score:     55.0,
					Timestamp: now,
				})
			}
		}
	}

	var correlatedDropAlerts int
	var totalChecked int
	for i := 0; i < len(stocks)-1 && totalChecked < 50; i++ {
		pi, _ := s.StockPriceRepo.GetLatestPrice(stocks[i].ID)
		pricesI, _ := s.StockPriceRepo.FindLatest(stocks[i].ID, 5)
		if len(pricesI) < 5 {
			continue
		}
		totalChecked++
		iChange := (pricesI[0].Close - pricesI[4].Close) / pricesI[4].Close
		if iChange < -0.03 {
			for j := i + 1; j < len(stocks); j++ {
				pricesJ, _ := s.StockPriceRepo.FindLatest(stocks[j].ID, 5)
				if len(pricesJ) < 5 {
					continue
				}
				jChange := (pricesJ[0].Close - pricesJ[4].Close) / pricesJ[4].Close
				if jChange < -0.03 {
					correlatedDropAlerts++
					break
				}
			}
		}
		if pi > 0 {
			_ = pi
		}
	}

	if correlatedDropAlerts >= 5 {
		alerts = append(alerts, RiskAlert{
			Code:      "IHSG",
			RiskType:  "correlated_selling",
			Severity:  "high",
			Message:   fmt.Sprintf("Terdeteksi %d saham dengan penurunan simultan >3%% dalam 5 hari terakhir. Kemungkinan aksi jual terkoordinasi atau sentimen negatif menyeluruh.", correlatedDropAlerts),
			Score:     75.0,
			Timestamp: now,
		})
	}

	if len(alerts) == 0 {
		alerts = append(alerts, RiskAlert{
			Code:      "IHSG",
			RiskType:  "all_clear",
			Severity:  "low",
			Message:   "Tidak terdeteksi risiko black swan signifikan pada saham-saham yang dipantau. Pasar dalam kondisi normal.",
			Score:     10.0,
			Timestamp: now,
		})
	}

	return alerts, nil
}

func (s *AIForecastService) analyzeWithAI(stockCode, prompt string) string {
	if !s.AI.IsConfigured() {
		return ""
	}

	sysPrompt := "Kamu adalah analis pasar modal Indonesia. Jawab dalam bahasa Indonesia, 2-3 kalimat maksimal."
	response, err := s.AI.Chat(sysPrompt, prompt)
	if err != nil {
		return ""
	}
	return response
}

func calcHistoricalVolatility(prices []model.StockPrice, period int) float64 {
	if len(prices) < period+1 {
		return 0.02
	}

	var returns []float64
	for i := 0; i < period && i+1 < len(prices); i++ {
		if prices[i+1].Close > 0 {
			r := math.Log(prices[i].Close / prices[i+1].Close)
			returns = append(returns, r)
		}
	}

	if len(returns) == 0 {
		return 0.02
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	var variance float64
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance)
}

func predictionToLabel(pred string) string {
	switch pred {
	case "likely_increase":
		return "kemungkinan naik"
	case "likely_cut":
		return "kemungkinan turun"
	case "stable":
		return "stabil"
	default:
		return "tidak diketahui"
	}
}

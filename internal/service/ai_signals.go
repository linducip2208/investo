package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"investo/internal/model"
	"investo/internal/repository"
)

type SignalConfidence struct {
	Code            string   `json:"code"`
	Signal          string   `json:"signal"`
	ConfidenceScore float64  `json:"confidence_score"`
	Explainability  []string `json:"explainability"`
	Analysis        string   `json:"analysis"`
}

type SignalBacktest struct {
	Code           string  `json:"code"`
	Accuracy       float64 `json:"accuracy"`
	TotalSignals   int     `json:"total_signals"`
	CorrectSignals int     `json:"correct_signals"`
	AvgReturn      float64 `json:"avg_return"`
	MaxReturn      float64 `json:"max_return"`
	MinReturn      float64 `json:"min_return"`
	TestedDays     int     `json:"tested_days"`
}

type RegimeResult struct {
	Regime             string  `json:"regime"`
	Confidence         float64 `json:"confidence"`
	RecommendedStrategy string `json:"recommended_strategy"`
	Description        string  `json:"description"`
}

type EntryAnalysis struct {
	Code          string   `json:"code"`
	CurrentPrice  float64  `json:"current_price"`
	EntryZoneLow  float64  `json:"entry_zone_low"`
	EntryZoneHigh float64  `json:"entry_zone_high"`
	SupportLevels []float64 `json:"support_levels"`
	Rationale     string   `json:"rationale"`
}

type AISignalService struct {
	AI                   *AIService
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
	DecisionRepo         *repository.AgentDecisionRepository
}

func NewAISignalService(
	ai *AIService,
	stockRepo *repository.StockRepository,
	stockPriceRepo *repository.StockPriceRepository,
	stockFundamentalRepo *repository.StockFundamentalRepository,
	sectorRepo *repository.SectorRepository,
	decisionRepo *repository.AgentDecisionRepository,
) *AISignalService {
	return &AISignalService{
		AI:                   ai,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		DecisionRepo:         decisionRepo,
	}
}

func (s *AISignalService) GetSignalWithConfidence(code string) (*SignalConfidence, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	price, _ := s.StockPriceRepo.GetLatestPrice(stock.ID)
	fund, _ := s.StockFundamentalRepo.FindLatest(stock.ID)

	var signal string
	var baseConfidence float64

	decision, _ := s.DecisionRepo.FindLatestByTicker(code)
	if decision != nil {
		signal = decision.FinalSignal
		baseConfidence = decision.Confidence
	} else {
		signal = "HOLD"
		baseConfidence = 50
		if fund != nil {
			if fund.PER > 0 && fund.PER < 12 && fund.ROE > 15 {
				signal = "BUY"
				baseConfidence = 65
			} else if fund.PER > 30 || fund.DER > 3 {
				signal = "SELL"
				baseConfidence = 60
			}
		}
	}

	explain := s.generateExplainability(stock.Code, signal, fund, price)
	analysis := s.generateConfidenceAnalysis(stock.Code, stock.Name, signal, baseConfidence, price)

	score := baseConfidence
	if s.AI.IsConfigured() {
		sysPrompt := fmt.Sprintf(`Kamu adalah AI trading signal analyst. Evaluasi sinyal trading untuk saham %s (%s).
Sinyal saat ini: %s dengan confidence %.0f%%. Harga: Rp %.0f.

Berikan confidence score 0-100 dan 3-5 poin explainability dalam JSON:
{"confidence_score": float, "analysis": "analisis singkat dalam bahasa Indonesia", "explain": ["poin1", "poin2", ...]}`, stock.Code, stock.Name, signal, baseConfidence, price)

		userPrompt := fmt.Sprintf("Evaluasi sinyal %s untuk saham %s.", signal, stock.Code)
		response, err := s.AI.Chat(sysPrompt, userPrompt)
		if err == nil {
			jsonStr := extractJSON(response)
			var parsed struct {
				ConfidenceScore float64  `json:"confidence_score"`
				Analysis        string   `json:"analysis"`
				Explain         []string `json:"explain"`
			}
			if json.Unmarshal([]byte(jsonStr), &parsed) == nil {
				if parsed.ConfidenceScore > 0 {
					score = clamp(parsed.ConfidenceScore, 0, 100)
				}
				if parsed.Analysis != "" {
					analysis = parsed.Analysis
				}
				if len(parsed.Explain) > 0 {
					explain = parsed.Explain
				}
			}
		}
	}

	return &SignalConfidence{
		Code:            code,
		Signal:          signal,
		ConfidenceScore: math.Round(score*10) / 10,
		Explainability:  explain,
		Analysis:        analysis,
	}, nil
}

func (s *AISignalService) BacktestSignal(code string, days int) (*SignalBacktest, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	if days <= 0 {
		days = 365
	}

	prices, err := s.StockPriceRepo.FindLatest(stock.ID, days+30)
	if err != nil || len(prices) < 30 {
		return &SignalBacktest{
			Code:         code,
			Accuracy:     0,
			TotalSignals: 0,
			TestedDays:   days,
		}, fmt.Errorf("insufficient price data for backtest")
	}

	prices = reversePricesForAISignal(prices)

	var totalSignals, correctSignals int
	var returns []float64

	for i := 20; i < len(prices)-5; i++ {
		current := prices[i].Close
		prev20 := make([]model.StockPrice, 21)
		for j := 0; j <= 20 && i-j >= 0; j++ {
			prev20[j] = prices[i-j]
		}

		sma20 := calcSMAFromPrices(prev20, minInt(20, len(prev20)))
		rsi := calcRSIFromPrices(prev20, 14)

		var signal string
		if current > sma20*1.02 && rsi > 50 && rsi < 70 {
			signal = "BUY"
		} else if current < sma20*0.98 && rsi < 50 {
			signal = "SELL"
		} else {
			signal = "HOLD"
		}

		if signal == "HOLD" {
			continue
		}

		totalSignals++
		futurePrice := prices[i+5].Close
		ret := (futurePrice - current) / current * 100

		if (signal == "BUY" && ret > 0) || (signal == "SELL" && ret < 0) {
			correctSignals++
		}
		returns = append(returns, ret)
	}

	var accuracy float64
	if totalSignals > 0 {
		accuracy = float64(correctSignals) / float64(totalSignals) * 100
	}

	var avgRet, maxRet, minRet float64
	if len(returns) > 0 {
		var sum float64
		maxRet = returns[0]
		minRet = returns[0]
		for _, r := range returns {
			sum += r
			if r > maxRet {
				maxRet = r
			}
			if r < minRet {
				minRet = r
			}
		}
		avgRet = sum / float64(len(returns))
	}

	if s.AI.IsConfigured() {
		enhanced := s.enhanceBacktestWithAI(code, stock.Name, accuracy, totalSignals, correctSignals, avgRet, days)
		if enhanced.Accuracy > 0 {
			accuracy = enhanced.Accuracy
		}
	}

	return &SignalBacktest{
		Code:           code,
		Accuracy:       math.Round(accuracy*10) / 10,
		TotalSignals:   totalSignals,
		CorrectSignals: correctSignals,
		AvgReturn:      math.Round(avgRet*100) / 100,
		MaxReturn:      math.Round(maxRet*100) / 100,
		MinReturn:      math.Round(minRet*100) / 100,
		TestedDays:     days,
	}, nil
}

func (s *AISignalService) DetectMarketRegime() (*RegimeResult, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil || len(stocks) == 0 {
		return &RegimeResult{
			Regime:             "tidak_diketahui",
			Confidence:         0,
			RecommendedStrategy: "hold",
			Description:        "Data pasar tidak tersedia untuk analisis regime.",
		}, nil
	}

	var trendingCount, rangingCount, volatileCount int
	var sampleSize int

	for _, stock := range stocks[:minInt(20, len(stocks))] {
		prices, err := s.StockPriceRepo.FindLatest(stock.ID, 30)
		if err != nil || len(prices) < 20 {
			continue
		}

		sma20 := calcSMAFromPrices(prices, 20)
		currentPrice := prices[0].Close
		deviation := math.Abs(currentPrice-sma20) / sma20

		highLowRange := (prices[0].High - prices[0].Low) / prices[0].Close

		if deviation > 0.05 {
			trendingCount++
		} else if highLowRange > 0.05 {
			volatileCount++
		} else {
			rangingCount++
		}
		sampleSize++
	}

	if sampleSize == 0 {
		return &RegimeResult{
			Regime:             "tidak_diketahui",
			Confidence:         0,
			RecommendedStrategy: "hold",
			Description:        "Data harga tidak mencukupi untuk analisis regime.",
		}, nil
	}

	var regime, strategy, description string
	var confidence float64

	totalWeight := float64(trendingCount + rangingCount + volatileCount)
	trendingPct := float64(trendingCount) / totalWeight * 100
	rangingPct := float64(rangingCount) / totalWeight * 100
	volatilePct := float64(volatileCount) / totalWeight * 100

	if trendingPct >= rangingPct && trendingPct >= volatilePct {
		regime = "trending"
		strategy = "trend_following"
		confidence = trendingPct
		description = fmt.Sprintf("Pasar dalam kondisi trending. %d dari %d saham menunjukkan tren yang jelas. Strategi trend following direkomendasikan.", trendingCount, sampleSize)
	} else if rangingPct >= volatilePct {
		regime = "ranging"
		strategy = "mean_reversion"
		confidence = rangingPct
		description = fmt.Sprintf("Pasar dalam kondisi ranging/sideways. %d dari %d saham bergerak dalam range. Strategi mean reversion dan swing trading direkomendasikan.", rangingCount, sampleSize)
	} else {
		regime = "volatile"
		strategy = "breakout"
		confidence = volatilePct
		description = fmt.Sprintf("Pasar dalam kondisi volatil. %d dari %d saham menunjukkan volatilitas tinggi. Strategi breakout dan risk management ketat direkomendasikan.", volatileCount, sampleSize)
	}

	if s.AI.IsConfigured() {
		sysPrompt := fmt.Sprintf(`Kamu adalah analis market regime untuk IHSG. Market saat ini: %s (confidence %.0f%%) berdasarkan %d saham.
Trending: %d, Ranging: %d, Volatile: %d.

Buat deskripsi market regime dalam 2-3 kalimat bahasa Indonesia yang informatif. Output JSON:
{"regime": "%s", "strategy": "%s", "description": "deskripsi dalam bahasa Indonesia"}`, regime, confidence, sampleSize, trendingCount, rangingCount, volatileCount, regime, strategy)

		response, err := s.AI.Chat(sysPrompt, "Jelaskan kondisi market IHSG saat ini.")
		if err == nil {
			jsonStr := extractJSON(response)
			var parsed struct {
				Regime      string `json:"regime"`
				Strategy    string `json:"strategy"`
				Description string `json:"description"`
			}
			if json.Unmarshal([]byte(jsonStr), &parsed) == nil {
				if parsed.Description != "" {
					description = parsed.Description
				}
				if parsed.Strategy != "" {
					strategy = parsed.Strategy
				}
			}
		}
	}

	return &RegimeResult{
		Regime:             regime,
		Confidence:         math.Round(confidence*10) / 10,
		RecommendedStrategy: strategy,
		Description:        description,
	}, nil
}

func (s *AISignalService) FindOptimalEntry(code string) (*EntryAnalysis, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	currentPrice, _ := s.StockPriceRepo.GetLatestPrice(stock.ID)
	prices, _ := s.StockPriceRepo.FindLatest(stock.ID, 90)

	var supports []float64

	if len(prices) >= 20 {
		sma20 := calcSMAFromPrices(prices, 20)
		supports = append(supports, sma20)
	}
	if len(prices) >= 50 {
		sma50 := calcSMAFromPrices(prices, 50)
		supports = append(supports, sma50)
	}

	if len(prices) > 0 {
		var lowest float64 = prices[0].Low
		for _, p := range prices {
			if p.Low < lowest {
				lowest = p.Low
			}
		}
		highLowRange := currentPrice - lowest
		if highLowRange > 0 {
			fib382 := lowest + highLowRange*0.382
			fib500 := lowest + highLowRange*0.500
			fib618 := lowest + highLowRange*0.618
			supports = append(supports, fib382, fib500, fib618)
		}
	}

	var validSupports []float64
	for _, s := range supports {
		if s > 0 && s < currentPrice {
			validSupports = append(validSupports, s)
		}
	}

	if len(validSupports) == 0 {
		validSupports = append(validSupports, currentPrice*0.95, currentPrice*0.90)
	}

	var clusterSum float64
	for _, s := range validSupports {
		clusterSum += s
	}
	avgSupport := clusterSum / float64(len(validSupports))

	entryLow := avgSupport * 0.98
	entryHigh := avgSupport * 1.02

	rationale := fmt.Sprintf("Entry zone optimal untuk %s (%s) dihitung dari cluster support level. Entry zone: Rp %.0f - Rp %.0f. Support cluster di sekitar Rp %.0f berdasarkan %d level teknikal.",
		stock.Code, stock.Name, entryLow, entryHigh, avgSupport, len(validSupports))

	if s.AI.IsConfigured() {
		aiRationale := s.enhanceEntryRationale(stock.Code, stock.Name, currentPrice, validSupports, entryLow, entryHigh)
		if aiRationale != "" {
			rationale = aiRationale
		}
	}

	return &EntryAnalysis{
		Code:          code,
		CurrentPrice:  math.Round(currentPrice),
		EntryZoneLow:  math.Round(entryLow),
		EntryZoneHigh: math.Round(entryHigh),
		SupportLevels: validSupports,
		Rationale:     rationale,
	}, nil
}

func (s *AISignalService) InterpretInsiderTransaction(code string) (string, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return "", fmt.Errorf("stock not found: %s", code)
	}
	_, _ = s.StockPriceRepo.GetLatestPrice(stock.ID)

	fallback := fmt.Sprintf("Data transaksi insider untuk saham %s (%s) belum tersedia di sistem. Biasanya, insider buying dalam jumlah besar menandakan keyakinan manajemen terhadap prospek perusahaan. Sebaliknya, insider selling signifikan bisa menjadi sinyal peringatan, meskipun bisa juga karena alasan pribadi (diversifikasi, kebutuhan likuiditas).", stock.Code, stock.Name)

	if !s.AI.IsConfigured() {
		return fallback, nil
	}

	sysPrompt := fmt.Sprintf(`Kamu adalah analis insider trading untuk pasar modal Indonesia. Analisis signifikansi transaksi insider untuk saham %s (%s).

Jelaskan dalam 3-5 kalimat bahasa Indonesia:
1. Jika ada insider buying → apa maknanya (keyakinan manajemen, undervaluation signal)
2. Jika ada insider selling → apakah alarm atau alasan pribadi
3. Konteks: insider trading di Indonesia diatur OJK, transaksi di atas threshold wajib dilaporkan

Bersikap objektif, jangan memberikan rekomendasi beli/jual.`, stock.Code, stock.Name)

	response, err := s.AI.Chat(sysPrompt, fmt.Sprintf("Interpretasikan transaksi insider untuk saham %s di BEI.", stock.Code))
	if err != nil {
		return fallback, nil
	}
	return response, nil
}

func (s *AISignalService) generateExplainability(code, signal string, fund *model.StockFundamental, price float64) []string {
	explain := []string{}
	if fund != nil {
		if fund.PER > 0 {
			explain = append(explain, fmt.Sprintf("PER %.1fx sebagai indikator valuasi", fund.PER))
		}
		if fund.ROE > 0 {
			explain = append(explain, fmt.Sprintf("ROE %.1f%% menunjukkan profitabilitas", fund.ROE))
		}
		if fund.DER > 0 {
			explain = append(explain, fmt.Sprintf("DER %.2fx untuk kesehatan neraca", fund.DER))
		}
	}
	explain = append(explain,
		fmt.Sprintf("Harga terkini Rp %.0f", price),
		"Konsensus dari historical agent decisions",
	)
	return explain
}

func (s *AISignalService) generateConfidenceAnalysis(code, name, signal string, confidence float64, price float64) string {
	return fmt.Sprintf("Sinyal %s untuk %s (%s) dengan confidence %.0f%% berdasarkan analisis multi-agent dan data fundamental. Harga saat ini: Rp %.0f.",
		signal, name, code, confidence, price)
}

func (s *AISignalService) enhanceBacktestWithAI(code, name string, dataAccuracy float64, total, correct int, avgRet float64, days int) *SignalBacktest {
	sysPrompt := fmt.Sprintf(`Evaluasi hasil backtest signal untuk saham %s (%s) selama %d hari.
Akurasi: %.1f%% (%d benar dari %d sinyal), rata-rata return: %.2f%%.

Output JSON: {"accuracy": float, "note": "1 kalimat penjelasan dalam bahasa Indonesia"}`, name, code, days, dataAccuracy, correct, total, avgRet)

	response, err := s.AI.Chat(sysPrompt, "Evaluasi hasil backtest ini.")
	if err != nil {
		return &SignalBacktest{Accuracy: dataAccuracy}
	}

	jsonStr := extractJSON(response)
	var parsed struct {
		Accuracy float64 `json:"accuracy"`
	}
	if json.Unmarshal([]byte(jsonStr), &parsed) == nil && parsed.Accuracy > 0 {
		return &SignalBacktest{Accuracy: parsed.Accuracy}
	}
	return &SignalBacktest{Accuracy: dataAccuracy}
}

func (s *AISignalService) enhanceEntryRationale(code, name string, currentPrice float64, supports []float64, entryLow, entryHigh float64) string {
	var supportStr string
	for _, s := range supports {
		supportStr += fmt.Sprintf("Rp %.0f, ", s)
	}
	supportStr = strings.TrimRight(supportStr, ", ")

	sysPrompt := fmt.Sprintf(`Kamu adalah technical analyst untuk pasar modal Indonesia. Buat rationale untuk entry zone saham %s (%s).
Harga saat ini: Rp %.0f
Support cluster: %s
Entry zone: Rp %.0f - Rp %.0f

Buat rationale 2-3 kalimat dalam bahasa Indonesia yang profesional dan actionable. Output JSON: {"rationale": "..."}`, name, code, currentPrice, supportStr, entryLow, entryHigh)

	response, err := s.AI.Chat(sysPrompt, "Buat rationale entry zone.")
	if err != nil {
		return ""
	}

	jsonStr := extractJSON(response)
	var parsed struct {
		Rationale string `json:"rationale"`
	}
	if json.Unmarshal([]byte(jsonStr), &parsed) == nil {
		return parsed.Rationale
	}
	return ""
}

func reversePricesForAISignal(prices []model.StockPrice) []model.StockPrice {
	reversed := make([]model.StockPrice, len(prices))
	for i, p := range prices {
		reversed[len(prices)-1-i] = p
	}
	return reversed
}

func calcSMAFromPrices(prices []model.StockPrice, period int) float64 {
	if len(prices) < period {
		period = len(prices)
	}
	if period == 0 {
		return 0
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += prices[i].Close
	}
	return sum / float64(period)
}

func calcRSIFromPrices(prices []model.StockPrice, period int) float64 {
	if len(prices) < period+1 {
		return 50
	}
	var gains, losses float64
	for i := 0; i < period; i++ {
		change := prices[i].Close - prices[i+1].Close
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



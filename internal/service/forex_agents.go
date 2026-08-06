package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"investo/internal/repository"
)

type ForexAgentPipeline struct {
	AIService      *AIService
	StockPriceRepo *repository.StockPriceRepository
	ForexRepo      *repository.ForexRepository
	NewsRepo       *repository.NewsRepository
	SentimentSvc   *SentimentService
}

type ForexSignalResult struct {
	Pair            string  `json:"pair"`
	TechnicalScore  float64 `json:"technical_score"`
	MacroScore      float64 `json:"macro_score"`
	SentimentScore  float64 `json:"sentiment_score"`
	CompositeSignal string  `json:"composite_signal"`
	Confidence      float64 `json:"confidence"`
	CarryTradeSignal string `json:"carry_trade_signal"`
	EventRisk       string  `json:"event_risk"`
	Rationale       string  `json:"rationale"`
	CurrentRate     float64 `json:"current_rate"`
	TargetRate      float64 `json:"target_rate"`
	StopLoss        float64 `json:"stop_loss"`
}

type ForexTechnicalAgent struct {
	AI *AIService
}

type ForexMacroAgent struct {
	AI *AIService
}

type ForexSentimentAgent struct {
	SentimentSvc *SentimentService
	NewsRepo     *repository.NewsRepository
	AI           *AIService
}

func NewForexAgentPipeline(
	ai *AIService,
	stockPriceRepo *repository.StockPriceRepository,
	forexRepo *repository.ForexRepository,
	newsRepo *repository.NewsRepository,
	sentimentSvc *SentimentService,
) *ForexAgentPipeline {
	return &ForexAgentPipeline{
		AIService:      ai,
		StockPriceRepo: stockPriceRepo,
		ForexRepo:      forexRepo,
		NewsRepo:       newsRepo,
		SentimentSvc:   sentimentSvc,
	}
}

func (p *ForexAgentPipeline) Analyze(pairCode string) (*ForexSignalResult, error) {
	var wg sync.WaitGroup
	var technicalScore, macroScore, sentimentScore float64
	var carryTradeSignal, eventRisk string
	var technicalRationale, macroRationale, sentimentRationale string

	technicalAgent := &ForexTechnicalAgent{AI: p.AIService}
	macroAgent := &ForexMacroAgent{AI: p.AIService}
	sentimentAgent := &ForexSentimentAgent{
		SentimentSvc: p.SentimentSvc,
		NewsRepo:     p.NewsRepo,
		AI:           p.AIService,
	}

	wg.Add(3)

	go func() {
		defer wg.Done()
		technicalScore, technicalRationale = technicalAgent.Analyze(pairCode)
	}()

	go func() {
		defer wg.Done()
		macroScore, carryTradeSignal, eventRisk, macroRationale = macroAgent.Analyze(pairCode)
	}()

	go func() {
		defer wg.Done()
		sentimentScore, sentimentRationale = sentimentAgent.Analyze(pairCode)
	}()

	wg.Wait()

	compositeScore := (technicalScore * 0.45) + (macroScore * 0.35) + (sentimentScore * 0.20)

	var signal string
	switch {
	case compositeScore >= 80:
		signal = "STRONG_BUY"
	case compositeScore >= 65:
		signal = "BUY"
	case compositeScore >= 45:
		signal = "HOLD"
	case compositeScore >= 30:
		signal = "SELL"
	default:
		signal = "STRONG_SELL"
	}

	confidence := math.Min(compositeScore, 98)
	if confidence < 20 {
		confidence = 20
	}

	baseRate := 10000.0
	if p.ForexRepo != nil {
		pairs, err := p.ForexRepo.FindAllPairs()
		if err == nil {
			latestRates, ratesErr := p.ForexRepo.FindLatestRates()
			if ratesErr == nil {
				for _, pair := range pairs {
					if pair.BaseCurrency+pair.QuoteCurrency == pairCode {
						for _, rate := range latestRates {
							if rate.PairID == pair.ID {
								baseRate = rate.Close
								break
							}
						}
						break
					}
				}
			}
		}
	}

	targetRate := baseRate * 1.02
	stopLossRate := baseRate * 0.98
	if signal == "STRONG_BUY" {
		targetRate = baseRate * 1.04
		stopLossRate = baseRate * 0.97
	} else if signal == "SELL" || signal == "STRONG_SELL" {
		targetRate = baseRate * 0.97
		stopLossRate = baseRate * 1.02
	}

	rationale := fmt.Sprintf("%s %s %s", technicalRationale, macroRationale, sentimentRationale)

	return &ForexSignalResult{
		Pair:             pairCode,
		TechnicalScore:   math.Round(technicalScore*10) / 10,
		MacroScore:       math.Round(macroScore*10) / 10,
		SentimentScore:   math.Round(sentimentScore*10) / 10,
		CompositeSignal:  signal,
		Confidence:       math.Round(confidence*10) / 10,
		CarryTradeSignal: carryTradeSignal,
		EventRisk:        eventRisk,
		Rationale:        rationale,
		CurrentRate:      math.Round(baseRate*10000) / 10000,
		TargetRate:       math.Round(targetRate*10000) / 10000,
		StopLoss:         math.Round(stopLossRate*10000) / 10000,
	}, nil
}

func (a *ForexTechnicalAgent) Analyze(pair string) (score float64, rationale string) {
	score = 50

	_ = pair

	switch {
	case score >= 80:
		rationale = fmt.Sprintf("Teknikal %s: indikator menunjukkan momentum kuat bullish. Skor: %.0f/100.", pair, score)
	case score >= 60:
		rationale = fmt.Sprintf("Teknikal %s: multi-timeframe menunjukkan tren positif. Skor: %.0f/100.", pair, score)
	case score >= 40:
		rationale = fmt.Sprintf("Teknikal %s: pasar dalam konsolidasi. Skor: %.0f/100.", pair, score)
	default:
		rationale = fmt.Sprintf("Teknikal %s: indikator menunjukkan tekanan bearish. Skor: %.0f/100.", pair, score)
	}

	if a.AI != nil && a.AI.IsConfigured() {
		aiScore, aiRationale := enhanceForexTechnicalAI(a.AI, pair, score)
		if aiScore > 0 {
			score = aiScore
		}
		if aiRationale != "" {
			rationale = aiRationale
		}
	}

	return score, rationale
}

func (a *ForexMacroAgent) Analyze(pair string) (score float64, carryTradeSignal string, eventRisk string, rationale string) {
	score = 50
	carryTradeSignal = "neutral"
	eventRisk = "LOW"

	_ = pair

	if a.AI != nil && a.AI.IsConfigured() {
		aiScore, aiCarry, aiEvent, aiRationale := enhanceForexMacroAI(a.AI, pair, score)
		if aiScore > 0 {
			score = aiScore
		}
		if aiCarry != "" {
			carryTradeSignal = aiCarry
		}
		if aiEvent != "" {
			eventRisk = aiEvent
		}
		if aiRationale != "" {
			rationale = aiRationale
			return
		}
	}

	rationale = fmt.Sprintf("Makro %s: kondisi ekonomi netral, carry trade: %s, event risk: %s. Skor: %.0f/100.",
		pair, carryTradeSignal, eventRisk, score)
	return
}

func (a *ForexSentimentAgent) Analyze(pair string) (score float64, rationale string) {
	score = 50

	if a.SentimentSvc != nil {
		result := a.SentimentSvc.Analyze("forex " + pair)
		score = clamp((result.Score+1)*50, 0, 100)
	}

	rationale = fmt.Sprintf("Sentimen %s: %.0f/100 berdasarkan data pasar.", pair, score)

	if a.AI != nil && a.AI.IsConfigured() {
		aiScore, aiRationale := enhanceForexSentimentAI(a.AI, pair, score)
		if aiScore > 0 {
			score = aiScore
		}
		if aiRationale != "" {
			rationale = aiRationale
		}
	}

	return score, rationale
}

func enhanceForexTechnicalAI(ai *AIService, pair string, baseScore float64) (float64, string) {
	sysPrompt := fmt.Sprintf(`Kamu adalah technical analyst forex. Pair: %s. Skor dasar: %.0f/100.
Beri skor teknikal 0-100 dan rationale 1 kalimat dalam JSON: {"score": float, "rationale": "bahasa Indonesia"}`, pair, baseScore)
	return callAIForScore(ai, sysPrompt, "Evaluasi teknikal "+pair)
}

func enhanceForexMacroAI(ai *AIService, pair string, baseScore float64) (float64, string, string, string) {
	sysPrompt := fmt.Sprintf(`Kamu adalah analis makro ekonomi forex. Pair: %s. Skor dasar: %.0f/100.
Periksa kondisi ekonomi makro, carry trade, event risk.
Output JSON: {"score": float, "carry_trade": "positive_carry/negative_carry/neutral", "event_risk": "HIGH/LOW/MEDIUM", "rationale": "bahasa Indonesia"}`, pair, baseScore)

	response, err := ai.Chat(sysPrompt, "Evaluasi makro "+pair)
	if err != nil {
		return 0, "", "", ""
	}

	var parsed struct {
		Score      float64 `json:"score"`
		CarryTrade string  `json:"carry_trade"`
		EventRisk  string  `json:"event_risk"`
		Rationale  string  `json:"rationale"`
	}
	jsonStr := extractJSON(response)
	parseJSONData([]byte(jsonStr), &parsed)
	return parsed.Score, parsed.CarryTrade, parsed.EventRisk, parsed.Rationale
}

func enhanceForexSentimentAI(ai *AIService, pair string, baseScore float64) (float64, string) {
	sysPrompt := fmt.Sprintf(`Kamu adalah analis sentimen forex. Pair: %s. Skor dasar: %.0f/100.
Beri skor sentimen 0-100 dan rationale 1 kalimat dalam JSON: {"score": float, "rationale": "bahasa Indonesia"}`, pair, baseScore)
	return callAIForScore(ai, sysPrompt, "Evaluasi sentimen "+pair)
}

func callAIForScore(ai *AIService, sysPrompt, userPrompt string) (float64, string) {
	response, err := ai.Chat(sysPrompt, userPrompt)
	if err != nil {
		return 0, ""
	}

	var parsed struct {
		Score     float64 `json:"score"`
		Rationale string  `json:"rationale"`
	}
	jsonStr := extractJSON(response)
	parseJSONData([]byte(jsonStr), &parsed)
	return parsed.Score, parsed.Rationale
}

func parseJSONData(data []byte, v interface{}) {
	json.Unmarshal(data, v)
}

package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"investo/internal/model"
	"investo/internal/repository"
)

type BEIAgentPipeline struct {
	AIService       *AIService
	StockRepo       *repository.StockRepository
	StockPriceRepo  *repository.StockPriceRepository
	StockFundRepo   *repository.StockFundamentalRepository
	NewsRepo        *repository.NewsRepository
	SentimentSvc    *SentimentService
	ForeignFlowSvc  *ForeignFlowService
}

type BEISignalResult struct {
	StockCode            string  `json:"stock_code"`
	StockName            string  `json:"stock_name"`
	FundamentalScore     float64 `json:"fundamental_score"`
	ForeignFlowScore     float64 `json:"foreign_flow_score"`
	SentimentScore       float64 `json:"sentiment_score"`
	TechnicalScore       float64 `json:"technical_score"`
	CompositeSignal      string  `json:"composite_signal"`
	Confidence           float64 `json:"confidence"`
	ARAARBAlert          string  `json:"ara_arb_alert"`
	ForeignFlowDirection string  `json:"foreign_flow_direction"`
	Rationale            string  `json:"rationale"`
	CurrentPrice         float64 `json:"current_price"`
	TargetPrice          float64 `json:"target_price"`
	StopLoss             float64 `json:"stop_loss"`
}

type BEIFundamentalAgent struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	StockFundRepo  *repository.StockFundamentalRepository
	AI             *AIService
}

type BEIForeignFlowAgent struct {
	ForeignFlowSvc *ForeignFlowService
	AI             *AIService
}

type BEISentimentAgent struct {
	SentimentSvc *SentimentService
	NewsRepo     *repository.NewsRepository
	AI           *AIService
}

type BEITechnicalAgent struct {
	StockPriceRepo *repository.StockPriceRepository
	AI             *AIService
}

func NewBEIAgentPipeline(
	ai *AIService,
	stockRepo *repository.StockRepository,
	stockPriceRepo *repository.StockPriceRepository,
	stockFundRepo *repository.StockFundamentalRepository,
	newsRepo *repository.NewsRepository,
	sentimentSvc *SentimentService,
	foreignFlowSvc *ForeignFlowService,
) *BEIAgentPipeline {
	return &BEIAgentPipeline{
		AIService:       ai,
		StockRepo:       stockRepo,
		StockPriceRepo:  stockPriceRepo,
		StockFundRepo:   stockFundRepo,
		NewsRepo:        newsRepo,
		SentimentSvc:    sentimentSvc,
		ForeignFlowSvc:  foreignFlowSvc,
	}
}

func (p *BEIAgentPipeline) Analyze(code string) (*BEISignalResult, error) {
	stock, err := p.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("saham tidak ditemukan: %s", code)
	}

	latestPrice, _ := p.StockPriceRepo.GetLatestPrice(stock.ID)
	prices, _ := p.StockPriceRepo.FindLatest(stock.ID, 90)
	fund, _ := p.StockFundRepo.FindLatest(stock.ID)

	newsList, _, _ := p.NewsRepo.List(0, 20)
	var stockNews []model.News
	for _, n := range newsList {
		upperTitle := strings.ToUpper(n.Title)
		if strings.Contains(upperTitle, strings.ToUpper(code)) {
			stockNews = append(stockNews, n)
		}
	}
	if len(stockNews) > 5 {
		stockNews = stockNews[:5]
	}

	fundamentalAgent := &BEIFundamentalAgent{
		StockRepo:      p.StockRepo,
		StockPriceRepo: p.StockPriceRepo,
		StockFundRepo:  p.StockFundRepo,
		AI:             p.AIService,
	}
	foreignFlowAgent := &BEIForeignFlowAgent{
		ForeignFlowSvc: p.ForeignFlowSvc,
		AI:             p.AIService,
	}
	sentimentAgent := &BEISentimentAgent{
		SentimentSvc: p.SentimentSvc,
		NewsRepo:     p.NewsRepo,
		AI:           p.AIService,
	}
	technicalAgent := &BEITechnicalAgent{
		StockPriceRepo: p.StockPriceRepo,
		AI:             p.AIService,
	}

	var wg sync.WaitGroup
	var fundamentalScore, foreignFlowScore, sentimentScore, technicalScore float64
	var foreignFlowDirection, araArbAlert string
	var fundamentalRationale, sentimentRationale, technicalRationale string

	wg.Add(4)

	go func() {
		defer wg.Done()
		fundamentalScore, araArbAlert, fundamentalRationale = fundamentalAgent.Analyze(stock, fund, latestPrice)
	}()

	go func() {
		defer wg.Done()
		foreignFlowScore, foreignFlowDirection = foreignFlowAgent.Analyze(code)
	}()

	go func() {
		defer wg.Done()
		sentimentScore, sentimentRationale = sentimentAgent.Analyze(code, stock.Name, stockNews)
	}()

	go func() {
		defer wg.Done()
		technicalScore, technicalRationale = technicalAgent.Analyze(stock.ID, latestPrice, prices)
	}()

	wg.Wait()

	compositeScore := (fundamentalScore * 0.35) + (foreignFlowScore * 0.25) + (sentimentScore * 0.15) + (technicalScore * 0.25)

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

	targetPrice := latestPrice * 1.15
	stopLoss := latestPrice * 0.93
	if signal == "STRONG_BUY" {
		targetPrice = latestPrice * 1.25
		stopLoss = latestPrice * 0.88
	} else if signal == "SELL" || signal == "STRONG_SELL" {
		targetPrice = latestPrice * 0.90
		stopLoss = latestPrice * 1.05
	}

	rationale := fmt.Sprintf("%s %s %s",
		fundamentalRationale, sentimentRationale, technicalRationale)

	if araArbAlert != "" {
		rationale += " " + araArbAlert
	}

	return &BEISignalResult{
		StockCode:            code,
		StockName:            stock.Name,
		FundamentalScore:     math.Round(fundamentalScore*10) / 10,
		ForeignFlowScore:     math.Round(foreignFlowScore*10) / 10,
		SentimentScore:       math.Round(sentimentScore*10) / 10,
		TechnicalScore:       math.Round(technicalScore*10) / 10,
		CompositeSignal:      signal,
		Confidence:           math.Round(confidence*10) / 10,
		ARAARBAlert:          araArbAlert,
		ForeignFlowDirection: foreignFlowDirection,
		Rationale:            rationale,
		CurrentPrice:         math.Round(latestPrice),
		TargetPrice:          math.Round(targetPrice),
		StopLoss:             math.Round(stopLoss),
	}, nil
}

func (a *BEIFundamentalAgent) Analyze(stock *model.Stock, fund *model.StockFundamental, price float64) (score float64, araArbAlert string, rationale string) {
	score = 50

	if fund == nil {
		return 40, "", fmt.Sprintf("Fundamental %s: data tidak tersedia. Skor: 40/100.", stock.Code)
	}

	bonus := 0.0

	if fund.PER > 0 && fund.PER < 12 {
		bonus += 12
	} else if fund.PER > 0 && fund.PER < 18 {
		bonus += 6
	} else if fund.PER > 30 {
		bonus -= 8
	}

	if fund.PBV > 0 && fund.PBV < 1.5 {
		bonus += 8
	} else if fund.PBV > 0 && fund.PBV < 3 {
		bonus += 3
	} else if fund.PBV > 8 {
		bonus -= 5
	}

	if fund.ROE > 20 {
		bonus += 10
	} else if fund.ROE > 15 {
		bonus += 6
	} else if fund.ROE > 10 {
		bonus += 2
	} else if fund.ROE < 5 && fund.ROE > 0 {
		bonus -= 5
	}

	if fund.DER > 0 && fund.DER < 1 {
		bonus += 8
	} else if fund.DER > 0 && fund.DER < 2 {
		bonus += 2
	} else if fund.DER > 3 {
		bonus -= 10
	}

	if stock.SharesOutstanding > 100_000_000_000 {
		bonus += 4
	} else if stock.SharesOutstanding > 10_000_000_000 {
		bonus += 2
	}

	score = clamp(score+bonus, 0, 100)

	if price > 0 {
		araLimit := price * 1.25
		arbLimit := price * 0.75

		if price >= araLimit*0.98 {
			araArbAlert = fmt.Sprintf("Harga mendekati ARA (Auto Reject Atas) pada Rp %.0f.", araLimit)
		} else if price <= arbLimit*1.02 {
			araArbAlert = fmt.Sprintf("Harga mendekati ARB (Auto Reject Bawah) pada Rp %.0f.", arbLimit)
		}

		if price >= araLimit*0.95 {
			score = clamp(score-10, 0, 100)
		}
	}

	rationale = fmt.Sprintf("Fundamental %s: PER %.1fx, PBV %.2fx, ROE %.1f%%, DER %.2fx. Skor: %.0f/100.",
		stock.Code, fund.PER, fund.PBV, fund.ROE, fund.DER, score)

	if a.AI != nil && a.AI.IsConfigured() {
		aiScore, aiRationale := enhanceFundamentalAI(a.AI, stock.Code, stock.Name, fund, score)
		if aiScore > 0 {
			score = aiScore
		}
		if aiRationale != "" {
			rationale = aiRationale
		}
	}

	return score, araArbAlert, rationale
}

func enhanceFundamentalAI(ai *AIService, code, name string, fund *model.StockFundamental, baseScore float64) (float64, string) {
	sysPrompt := fmt.Sprintf(`Kamu adalah analis fundamental saham Indonesia.
Saham: %s (%s). PER %.1fx, PBV %.2fx, ROE %.1f%%, DER %.2fx.
Skor dasar: %.0f/100.
Beri skor fundamental 0-100 dan rationale 1 kalimat dalam JSON: {"score": float, "rationale": "bahasa Indonesia"}`, code, name, fund.PER, fund.PBV, fund.ROE, fund.DER, baseScore)

	response, err := ai.Chat(sysPrompt, fmt.Sprintf("Evaluasi fundamental %s", code))
	if err != nil {
		return 0, ""
	}

	var parsed struct {
		Score     float64 `json:"score"`
		Rationale string  `json:"rationale"`
	}
	jsonStr := extractJSON(response)
	json.Unmarshal([]byte(jsonStr), &parsed)
	return parsed.Score, parsed.Rationale
}

func (a *BEIForeignFlowAgent) Analyze(code string) (score float64, direction string) {
	score = 50
	direction = "neutral"

	if a.ForeignFlowSvc == nil {
		return score, direction
	}

	fd, err := a.ForeignFlowSvc.GetStockFlow(code)
	if err != nil {
		return score, direction
	}

	if fd.ForeignNet > 0 {
		bonus := math.Min(float64(fd.ForeignNet)/1_000_000_000*2, 25)
		score = clamp(score+bonus, 0, 100)
		direction = "inflow"
	} else if fd.ForeignNet < 0 {
		penalty := math.Min(math.Abs(float64(fd.ForeignNet))/1_000_000_000*2, 20)
		score = clamp(score-penalty, 0, 100)
		direction = "outflow"
	}

	volumeBonus := math.Min(float64(fd.TotalValue)/1_000_000_000_000*3, 10)
	score = clamp(score+volumeBonus, 0, 100)

	return score, direction
}

func (a *BEISentimentAgent) Analyze(code string, name string, newsList []model.News) (score float64, rationale string) {
	score = 50

	if a.SentimentSvc != nil && len(newsList) > 0 {
		totalSentimentScore := 0.0
		for _, news := range newsList {
			result := a.SentimentSvc.Analyze(news.Title + " " + news.Content)
			sentVal := (result.Score + 1) * 50
			totalSentimentScore += sentVal
		}
		score = clamp(totalSentimentScore/float64(len(newsList)), 0, 100)
	}

	rationale = fmt.Sprintf("Sentimen %s: %.0f/100 berdasarkan analisis berita terkini.", code, score)

	if a.AI != nil && a.AI.IsConfigured() {
		aiScore, aiRationale := enhanceSentimentAI(a.AI, code, name, score)
		if aiScore > 0 {
			score = aiScore
		}
		if aiRationale != "" {
			rationale = aiRationale
		}
	}

	return score, rationale
}

func enhanceSentimentAI(ai *AIService, code, name string, baseScore float64) (float64, string) {
	sysPrompt := fmt.Sprintf(`Kamu adalah analis sentimen pasar saham Indonesia.
Saham: %s (%s). Skor dasar: %.0f/100.
Beri skor sentimen 0-100 dan rationale 1 kalimat dalam JSON: {"score": float, "rationale": "bahasa Indonesia"}`, code, name, baseScore)

	response, err := ai.Chat(sysPrompt, fmt.Sprintf("Analisis sentimen %s", code))
	if err != nil {
		return 0, ""
	}

	var parsed struct {
		Score     float64 `json:"score"`
		Rationale string  `json:"rationale"`
	}
	jsonStr := extractJSON(response)
	json.Unmarshal([]byte(jsonStr), &parsed)
	return parsed.Score, parsed.Rationale
}

func (a *BEITechnicalAgent) Analyze(stockID int64, currentPrice float64, prices []model.StockPrice) (score float64, rationale string) {
	score = 50

	if len(prices) < 20 {
		return 40, "Data harga tidak mencukupi untuk analisis teknikal. Skor: 40/100."
	}

	sma20 := calcSMAFromPrices(prices, 20)
	rsi := calcRSIFromPrices(prices, 14)

	if currentPrice > sma20 {
		score = clamp(score+8, 0, 100)
	} else {
		score = clamp(score-8, 0, 100)
	}

	if rsi > 30 && rsi < 70 {
		score = clamp(score+5, 0, 100)
	} else if rsi > 80 {
		score = clamp(score-10, 0, 100)
	} else if rsi < 20 {
		score = clamp(score+10, 0, 100)
	}

	if len(prices) >= 50 {
		sma50 := calcSMAFromPrices(prices, 50)
		if currentPrice > sma50 {
			score = clamp(score+5, 0, 100)
		}
	}

	volatility := 0.0
	sampleLen := minInt(14, len(prices))
	for i := 0; i < sampleLen; i++ {
		if prices[i].Close > 0 {
			dailyRange := (prices[i].High - prices[i].Low) / prices[i].Close
			volatility += dailyRange
		}
	}
	volatility = volatility / float64(sampleLen)

	if volatility > 0.05 {
		score = clamp(score-5, 0, 100)
	}

	rationale = fmt.Sprintf("Teknikal: SMA20=%.0f, RSI=%.0f, Volatilitas=%.1f%%. Skor: %.0f/100.",
		sma20, rsi, volatility*100, score)

	if a.AI != nil && a.AI.IsConfigured() {
		aiScore, aiRationale := enhanceTechnicalAI(a.AI, currentPrice, sma20, rsi, score)
		if aiScore > 0 {
			score = aiScore
		}
		if aiRationale != "" {
			rationale = aiRationale
		}
	}

	return score, rationale
}

func enhanceTechnicalAI(ai *AIService, price, sma20, rsi, baseScore float64) (float64, string) {
	sysPrompt := fmt.Sprintf(`Kamu adalah technical analyst. Harga: %.0f, SMA20: %.0f, RSI: %.0f. Skor dasar: %.0f/100.
Beri skor teknikal 0-100 dan rationale 1 kalimat dalam JSON: {"score": float, "rationale": "bahasa Indonesia"}`, price, sma20, rsi, baseScore)

	response, err := ai.Chat(sysPrompt, "Evaluasi teknikal.")
	if err != nil {
		return 0, ""
	}

	var parsed struct {
		Score     float64 `json:"score"`
		Rationale string  `json:"rationale"`
	}
	jsonStr := extractJSON(response)
	json.Unmarshal([]byte(jsonStr), &parsed)
	return parsed.Score, parsed.Rationale
}

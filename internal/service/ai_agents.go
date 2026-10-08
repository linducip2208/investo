package service

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type AgentRole string

const (
	AgentFundamental AgentRole = "fundamental_analyst"
	AgentTechnical   AgentRole = "technical_analyst"
	AgentSentiment   AgentRole = "sentiment_analyst"
	AgentNews        AgentRole = "news_analyst"
	AgentBullish     AgentRole = "bullish_researcher"
	AgentBearish     AgentRole = "bearish_researcher"
	AgentTrader      AgentRole = "trader"
	AgentRisk        AgentRole = "risk_manager"
	AgentPortfolio   AgentRole = "portfolio_manager"
)

type AgentReport struct {
	Agent      AgentRole `json:"agent"`
	Role       string    `json:"role"`
	Analysis   string    `json:"analysis"`
	Signal     string    `json:"signal"`
	Confidence float64   `json:"confidence"`
	KeyPoints  []string  `json:"key_points"`
	RiskFlags  []string  `json:"risk_flags"`
}

type MultiAgentDecision struct {
	Ticker          string        `json:"ticker"`
	Date            string        `json:"date"`
	Reports         []AgentReport `json:"reports"`
	DebateLog       string        `json:"debate_log"`
	FinalSignal     string        `json:"final_signal"`
	EntryPrice      float64       `json:"entry_price"`
	TargetPrice     float64       `json:"target_price"`
	StopLoss        float64       `json:"stop_loss"`
	PositionPct     float64       `json:"position_pct"`
	Rationale       string        `json:"rationale"`
	RiskScore       float64       `json:"risk_score"`
	PastReflections []string      `json:"past_reflections"`
}

type MultiAgentService struct {
	AI         *AIService
	DecisionRepo *repository.AgentDecisionRepository
	CheckpointRepo *repository.AgentRunCheckpointRepository
	StockRepo    *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	NewsRepo     *repository.NewsRepository
	SentimentSvc *SentimentService
}

func NewMultiAgentService(
	ai *AIService,
	decisionRepo *repository.AgentDecisionRepository,
	checkpointRepo *repository.AgentRunCheckpointRepository,
	stockRepo *repository.StockRepository,
	stockPriceRepo *repository.StockPriceRepository,
	stockFundamentalRepo *repository.StockFundamentalRepository,
	newsRepo *repository.NewsRepository,
	sentimentSvc *SentimentService,
) *MultiAgentService {
	return &MultiAgentService{
		AI:                   ai,
		DecisionRepo:         decisionRepo,
		CheckpointRepo:       checkpointRepo,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		NewsRepo:             newsRepo,
		SentimentSvc:         sentimentSvc,
	}
}

func (s *MultiAgentService) RunAnalysis(code string) (*MultiAgentDecision, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	today := time.Now()

	pastReflections, _ := s.GetAgentMemory(code)

	fundamentalData := s.fetchFundamentalData(stock.ID)
	technicalData := s.fetchTechnicalData(stock.ID)
	sentimentData := s.fetchSentimentData(code)
	newsData := s.fetchNewsData(stock.ID)
	currentPrice := s.getCurrentPrice(stock.ID)

	var analystReports []AgentReport

	// Phase 1: Analyst Team (checkpoint-resumable)
	if cp := s.loadCheckpoint(code, today, "analysts"); cp != nil && len(cp) > 0 {
		analystReports = cp
	} else {
		fundamentalReport := s.runFundamentalAnalyst(stock.Code, stock.Name, fundamentalData, currentPrice)
		technicalReport := s.runTechnicalAnalyst(stock.Code, stock.Name, technicalData, currentPrice)
		sentimentReport := s.runSentimentAnalyst(stock.Code, stock.Name, sentimentData)
		newsReport := s.runNewsAnalyst(stock.Code, stock.Name, newsData)

		analystReports = []AgentReport{fundamentalReport, technicalReport, sentimentReport, newsReport}
		s.saveCheckpoint(code, today, "analysts", analystReports, "")
	}

	var debateLog string

	// Phase 2: Research Debate (multi-round, checkpoint-resumable)
	if debateReports, storedLog := s.loadCheckpointFull(code, today, "debate"); debateReports != nil {
		debateLog = storedLog
		analystReports = append(analystReports, debateReports...)
	} else {
		bullishReport := s.runBullishResearcher(stock.Code, stock.Name, analystReports, currentPrice)
		bearishReport := s.runBearishResearcher(stock.Code, stock.Name, analystReports, bullishReport, currentPrice)

		debateRoundLogs := []string{s.buildDebateLog(bullishReport, bearishReport)}
		for round := 2; round <= s.debateRounds(); round++ {
			bullishReport = s.runBullishResearcherRebuttal(stock.Code, stock.Name, analystReports, bearishReport, currentPrice, round)
			bearishReport = s.runBearishResearcherRebuttal(stock.Code, stock.Name, analystReports, bullishReport, currentPrice, round)
			debateRoundLogs = append(debateRoundLogs, s.buildDebateLog(bullishReport, bearishReport))
		}
		debateLog = strings.Join(debateRoundLogs, "\n\n")

		s.saveCheckpoint(code, today, "debate", []AgentReport{bullishReport, bearishReport}, debateLog)
		analystReports = append(analystReports, bullishReport, bearishReport)
	}

	allReports := analystReports

	// Phase 3: Trader Synthesis
	if cp := s.loadCheckpoint(code, today, "trader"); cp != nil {
		allReports = append(allReports, cp...)
	} else {
		traderReport := s.runTraderAgent(stock.Code, stock.Name, allReports, currentPrice)
		s.saveCheckpoint(code, today, "trader", []AgentReport{traderReport}, "")
		allReports = append(allReports, traderReport)
	}

	// Phase 4: Risk Management
	if cp := s.loadCheckpoint(code, today, "risk"); cp != nil {
		allReports = append(allReports, cp...)
	} else {
		riskReport := s.runRiskManager(stock.Code, stock.Name, allReports, currentPrice)
		s.saveCheckpoint(code, today, "risk", []AgentReport{riskReport}, "")
		allReports = append(allReports, riskReport)
	}

	// Phase 5: Portfolio Manager
	if cp := s.loadCheckpoint(code, today, "portfolio"); cp != nil {
		allReports = append(allReports, cp...)
	} else {
		pmReport := s.runPortfolioManager(stock.Code, stock.Name, allReports, debateLog, pastReflections, currentPrice)
		s.saveCheckpoint(code, today, "portfolio", []AgentReport{pmReport}, "")
		allReports = append(allReports, pmReport)
	}

	traderReport := s.findReport(allReports, AgentTrader)
	riskReport := s.findReport(allReports, AgentRisk)
	pmReport := s.findReport(allReports, AgentPortfolio)

	entryPrice, targetPrice, stopLoss, positionPct := s.parseTraderParams(traderReport, currentPrice)
	riskScore := s.parseRiskScore(riskReport)
	finalSignal := s.parseSignal(pmReport)

	decision := &MultiAgentDecision{
		Ticker:          code,
		Date:            today.Format("2006-01-02"),
		Reports:         allReports,
		DebateLog:       debateLog,
		FinalSignal:     finalSignal,
		EntryPrice:      entryPrice,
		TargetPrice:     targetPrice,
		StopLoss:        stopLoss,
		PositionPct:     positionPct,
		Rationale:       pmReport.Analysis,
		RiskScore:       riskScore,
		PastReflections: pastReflections,
	}

	if err := s.SaveDecision(decision); err != nil {
		return decision, fmt.Errorf("saved but failed to persist: %w", err)
	}

	// Successful run — clear checkpoints for this ticker/day.
	if s.CheckpointRepo != nil {
		_ = s.CheckpointRepo.Clear(code, today)
	}

	return decision, nil
}

func (s *MultiAgentService) loadCheckpoint(code string, runDate time.Time, phase string) []AgentReport {
	reports, _ := s.loadCheckpointFull(code, runDate, phase)
	return reports
}

func (s *MultiAgentService) loadCheckpointFull(code string, runDate time.Time, phase string) ([]AgentReport, string) {
	if s.CheckpointRepo == nil {
		return nil, ""
	}
	cp, err := s.CheckpointRepo.Find(code, runDate, phase)
	if err != nil || cp == nil || cp.ReportsJSON == "" {
		return nil, ""
	}
	var reports []AgentReport
	if err := json.Unmarshal([]byte(cp.ReportsJSON), &reports); err != nil {
		return nil, ""
	}
	return reports, cp.DebateLog
}

func (s *MultiAgentService) saveCheckpoint(code string, runDate time.Time, phase string, reports []AgentReport, debateLog string) {
	if s.CheckpointRepo == nil {
		return
	}
	reportsJSON, _ := json.Marshal(reports)
	_ = s.CheckpointRepo.Save(code, runDate, phase, string(reportsJSON), debateLog)
}

func (s *MultiAgentService) findReport(reports []AgentReport, role AgentRole) AgentReport {
	for i := len(reports) - 1; i >= 0; i-- {
		if reports[i].Agent == role {
			return reports[i]
		}
	}
	return AgentReport{}
}

func (s *MultiAgentService) RunBatch(codes []string) ([]*MultiAgentDecision, error) {
	var decisions []*MultiAgentDecision
	for _, code := range codes {
		d, err := s.RunAnalysis(code)
		if err != nil {
			decisions = append(decisions, &MultiAgentDecision{
				Ticker: code,
				Date:   time.Now().Format("2006-01-02"),
				Rationale: fmt.Sprintf("Error: %v", err),
			})
			continue
		}
		decisions = append(decisions, d)
	}
	return decisions, nil
}

func (s *MultiAgentService) GetAgentMemory(code string) ([]string, error) {
	return s.DecisionRepo.GetPastReflections(code)
}

func (s *MultiAgentService) SaveDecision(decision *MultiAgentDecision) error {
	reportsJSON, _ := json.Marshal(decision.Reports)

	d := &model.AgentDecision{
		Ticker:       decision.Ticker,
		DecisionDate: time.Now(),
		FinalSignal:  decision.FinalSignal,
		EntryPrice:   decision.EntryPrice,
		TargetPrice:  decision.TargetPrice,
		StopLoss:     decision.StopLoss,
		PositionPct:  decision.PositionPct,
		Confidence:   s.parseConfidenceFromReports(decision.Reports),
		RiskScore:    decision.RiskScore,
		ReportsJSON:  string(reportsJSON),
		DebateLog:    decision.DebateLog,
		Rationale:    decision.Rationale,
	}
	return s.DecisionRepo.Save(d)
}

// ── Phase 1: Analyst Team ──

func (s *MultiAgentService) runFundamentalAnalyst(code, name string, data map[string]interface{}, price float64) AgentReport {
	systemPrompt := `Kamu adalah Fundamental Analyst senior di pasar modal Indonesia dengan pengalaman 20+ tahun.
Kamu menganalisis fundamental perusahaan berdasarkan data keuangan yang diberikan.

Tugas kamu:
1. Analisis valuasi (PER, PBV, EV/EBITDA)
2. Analisis profitabilitas (ROE, ROA, NPM, EPS growth)
3. Analisis kesehatan keuangan (DER, Current Ratio, FCF)
4. Bandingkan dengan rata-rata industri
5. Beri sinyal: BUY jika fundamental sehat + valuasi wajar, SELL jika fundamental buruk + mahal, HOLD jika mixed

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`Analisis fundamental untuk saham %s (%s).

Harga saat ini: Rp %.2f

Data Fundamental:
%s

Berikan analisis fundamental komprehensif dalam format JSON.`, code, name, price, s.formatMap(data))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentFundamental, "Fundamental Analyst", response)
}

func (s *MultiAgentService) runTechnicalAnalyst(code, name string, data map[string]interface{}, price float64) AgentReport {
	systemPrompt := `Kamu adalah Technical Analyst profesional dengan keahlian analisis teknikal multi-timeframe.

Tugas kamu:
1. Analisis tren (MA crossover, ADX, MACD)
2. Analisis momentum (RSI, Stochastic, CCI)
3. Analisis volatility (Bollinger Bands, ATR)
4. Identifikasi pola candlestick dan chart pattern
5. Tentukan support & resistance level
6. Beri sinyal: BUY jika uptrend + momentum kuat, SELL jika downtrend, HOLD jika sideways

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`Analisis teknikal untuk saham %s (%s).

Harga saat ini: Rp %.2f

Data Indikator Teknikal:
%s

Berikan analisis teknikal komprehensif dalam format JSON.`, code, name, price, s.formatMap(data))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentTechnical, "Technical Analyst", response)
}

func (s *MultiAgentService) runSentimentAnalyst(code, name string, data map[string]interface{}) AgentReport {
	systemPrompt := `Kamu adalah Sentiment Analyst yang menganalisis sentimen pasar dan media sosial untuk saham Indonesia.

Tugas kamu:
1. Analisis sentimen berita dan media sosial
2. Evaluasi foreign flow (buy/sell)
3. Analisis broker summary
4. Evaluasi social media buzz dan trending topics
5. Beri sinyal: BUY jika sentimen positif kuat, SELL jika sentimen negatif dominan, HOLD jika mixed/netral

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`Analisis sentimen untuk saham %s (%s).

Data Sentimen:
%s

Berikan analisis sentimen dalam format JSON.`, code, name, s.formatMap(data))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentSentiment, "Sentiment Analyst", response)
}

func (s *MultiAgentService) runNewsAnalyst(code, name string, headlines []string) AgentReport {
	systemPrompt := `Kamu adalah News Analyst yang menganalisis dampak berita terhadap pergerakan saham Indonesia.

Tugas kamu:
1. Ringkas berita-berita terbaru terkait saham
2. Kategorikan dampak: positif, negatif, atau netral
3. Identifikasi katalis jangka pendek
4. Evaluasi risiko dari sisi berita (regulasi, lawsuit, restrukturisasi)
5. Beri sinyal berdasarkan sentimen berita

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`Analisis berita untuk saham %s (%s).

Headlines:
%s

Berikan analisis berita dalam format JSON.`, code, name, strings.Join(headlines, "\n"))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentNews, "News Analyst", response)
}

// ── Phase 2: Research Debate ──

func (s *MultiAgentService) runBullishResearcher(code, name string, analystReports []AgentReport, price float64) AgentReport {
	var reportsText string
	for _, r := range analystReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\nKey Points: %s\nRisk Flags: %s\n",
			r.Role, r.Agent, r.Signal, r.Analysis,
			strings.Join(r.KeyPoints, ", "),
			strings.Join(r.RiskFlags, ", "))
	}

	systemPrompt := `Kamu adalah Bullish Researcher — tugasmu membangun kasus terkuat MENGAPA saham ini layak DIBELI.

Baca laporan dari 4 analis (Fundamental, Technical, Sentiment, News). Cari semua argumen positif, bahkan dari analis yang memberi sinyal SELL.

Tugas kamu:
1. Bangun bull case yang komprehensif berdasarkan data dari semua analis
2. Counter bearish arguments dengan data dan logika
3. Identifikasi katalis potensial yang belum disebutkan
4. Beri target harga upside dan timeframe
5. Evaluasi peluang vs risiko dari perspektif bullish

TETAP OBJEKTIF — jangan mengabaikan risiko. Akui risiko, tapi jelaskan mengapa reward lebih besar.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`BANGUN BULL CASE untuk saham %s (%s) — harga saat ini: Rp %.2f

LAPORAN ANALIS:
%s

Bangun kasus bullish terkuat dalam format JSON.`, code, name, price, reportsText)

	response := s.callAgent(systemPrompt, userPrompt)
	report := s.parseAgentReport(AgentBullish, "Bullish Researcher", response)
	report.Signal = "BUY"
	return report
}

func (s *MultiAgentService) runBearishResearcher(code, name string, analystReports []AgentReport, bullishReport AgentReport, price float64) AgentReport {
	var reportsText string
	for _, r := range analystReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\nKey Points: %s\nRisk Flags: %s\n",
			r.Role, r.Agent, r.Signal, r.Analysis,
			strings.Join(r.KeyPoints, ", "),
			strings.Join(r.RiskFlags, ", "))
	}

	systemPrompt := `Kamu adalah Bearish Researcher — tugasmu membangun kasus terkuat MENGAPA saham ini BERISIKO dan layak DIJUAL/DIHINDARI.

Baca laporan dari 4 analis + Bullish Researcher. Cari semua risiko, red flags, dan kelemahan — bahkan dari analis yang memberi sinyal BUY.

Tugas kamu:
1. Bangun bear case yang komprehensif
2. Counter bullish arguments satu per satu dengan data
3. Identifikasi risiko tersembunyi yang belum disebutkan
4. Beri skenario worst-case dengan target harga downside
5. Evaluasi apakah risk/reward ratio tidak favorable

TETAP RASIONAL — jangan takut-takuti. Gunakan data untuk mendukung argumen.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`BANGUN BEAR CASE untuk saham %s (%s) — harga saat ini: Rp %.2f

LAPORAN ANALIS:
%s

LAPORAN BULLISH RESEARCHER:
Signal: %s | Confidence: %.0f
%s
Key Points: %s
Risk Flags: %s

Bangun kasus bearish terkuat dalam format JSON. COUNTER setiap argumen bullish.`, code, name, price, reportsText,
		bullishReport.Signal, bullishReport.Confidence, bullishReport.Analysis,
		strings.Join(bullishReport.KeyPoints, ", "),
		strings.Join(bullishReport.RiskFlags, ", "))

	response := s.callAgent(systemPrompt, userPrompt)
	report := s.parseAgentReport(AgentBearish, "Bearish Researcher", response)
	report.Signal = "SELL"
	return report
}

// ── Phase 2b: Multi-round debate rebuttals ──

func (s *MultiAgentService) runBullishResearcherRebuttal(code, name string, analystReports []AgentReport, bearishReport AgentReport, price float64, round int) AgentReport {
	var reportsText string
	for _, r := range analystReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\n",
			r.Role, r.Agent, r.Signal, r.Analysis)
	}

	systemPrompt := fmt.Sprintf(`Kamu adalah Bullish Researcher pada RONDE DEBAT #%d.

Baca argumen Bearish Researcher terbaru dan COUNTER satu per satu dengan data dan logika. Perkuat bull case, akui risiko yang valid, tapi tunjukkan mengapa kasus bullish masih lebih kuat.

TETAP OBJEKTIF.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "rebuttal lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`, round)

	userPrompt := fmt.Sprintf(`COUNTER BEAR CASE untuk saham %s (%s) — harga saat ini: Rp %.2f

LAPORAN ANALIS:
%s

LAPORAN BEARISH RESEARCHER (ronde sebelumnya):
Signal: %s | Confidence: %.0f
%s
Key Points: %s
Risk Flags: %s

Bangun rebuttal bullish terkuat dalam format JSON. COUNTER setiap argumen bearish.`, code, name, price, reportsText,
		bearishReport.Signal, bearishReport.Confidence, bearishReport.Analysis,
		strings.Join(bearishReport.KeyPoints, ", "),
		strings.Join(bearishReport.RiskFlags, ", "))

	response := s.callAgent(systemPrompt, userPrompt)
	report := s.parseAgentReport(AgentBullish, "Bullish Researcher", response)
	report.Signal = "BUY"
	return report
}

func (s *MultiAgentService) runBearishResearcherRebuttal(code, name string, analystReports []AgentReport, bullishReport AgentReport, price float64, round int) AgentReport {
	var reportsText string
	for _, r := range analystReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\n",
			r.Role, r.Agent, r.Signal, r.Analysis)
	}

	systemPrompt := fmt.Sprintf(`Kamu adalah Bearish Researcher pada RONDE DEBAT #%d.

Baca argumen Bullish Researcher terbaru dan COUNTER satu per satu dengan data. Perkuat bear case, akui kekuatan bull yang valid, tapi tunjukkan risiko yang tidak bisa diabaikan.

TETAP RASIONAL.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "rebuttal lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`, round)

	userPrompt := fmt.Sprintf(`COUNTER BULL CASE untuk saham %s (%s) — harga saat ini: Rp %.2f

LAPORAN ANALIS:
%s

LAPORAN BULLISH RESEARCHER (ronde sebelumnya):
Signal: %s | Confidence: %.0f
%s
Key Points: %s
Risk Flags: %s

Bangun rebuttal bearish terkuat dalam format JSON. COUNTER setiap argumen bullish.`, code, name, price, reportsText,
		bullishReport.Signal, bullishReport.Confidence, bullishReport.Analysis,
		strings.Join(bullishReport.KeyPoints, ", "),
		strings.Join(bullishReport.RiskFlags, ", "))

	response := s.callAgent(systemPrompt, userPrompt)
	report := s.parseAgentReport(AgentBearish, "Bearish Researcher", response)
	report.Signal = "SELL"
	return report
}

// ── Phase 3: Trader Synthesis ──

func (s *MultiAgentService) runTraderAgent(code, name string, allReports []AgentReport, price float64) AgentReport {
	var reportsText string
	for _, r := range allReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s, Confidence: %.0f)\n%s\n",
			r.Role, r.Agent, r.Signal, r.Confidence, r.Analysis)
	}

	systemPrompt := `Kamu adalah Senior Trader profesional dengan pengalaman 15+ tahun di bursa saham Indonesia.

Kamu MEMBACA SEMUA laporan analis + debat bullish/bearish dan membuat KEPUTUSAN TRADING.

Tugas kamu:
1. Evaluasi SEMUA laporan (fundamental, teknikal, sentimen, news, bullish, bearish)
2. Tetapkan sinyal final: BUY, SELL, atau HOLD
3. Tetapkan ENTRY PRICE (harga masuk yang ideal — harus LEBIH RENDAH dari harga saat ini untuk BUY)
4. Tetapkan TARGET PRICE (harga jual/take profit)
5. Tetapkan STOP LOSS (batas kerugian maksimal)
6. Jelaskan rationale keputusan

KAMU MEMBUAT KEPUTUSAN — tidak ragu-ragu. Pilih BUY, SELL, atau HOLD dengan jelas.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "entry_price": float (harga dalam Rupiah),
  "target_price": float (harga dalam Rupiah — harus lebih tinggi dari entry untuk BUY),
  "stop_loss": float (harga dalam Rupiah — harus lebih rendah dari entry untuk BUY),
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`BUAT KEPUTUSAN TRADING untuk saham %s (%s) — harga saat ini: Rp %.2f

SEMUA LAPORAN:
%s

Buat keputusan trading final. Tetapkan entry, target, stop loss. Dalam format JSON.`, code, name, price, reportsText)

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentTrader, "Trader", response)
}

// ── Phase 4: Risk Management ──

func (s *MultiAgentService) runRiskManager(code, name string, allReports []AgentReport, price float64) AgentReport {
	var reportsText string
	for _, r := range allReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\nRisk Flags: %s\n",
			r.Role, r.Agent, r.Signal, r.Analysis,
			strings.Join(r.RiskFlags, ", "))
	}

	systemPrompt := `Kamu adalah Risk Manager profesional. Tugasmu mengevaluasi RISIKO dari keputusan trading.

Tugas kamu:
1. Hitung risk score 0-100 (semakin tinggi = semakin berisiko)
2. Evaluasi position sizing yang aman (% dari portfolio)
3. Identifikasi semua risk factors (market risk, liquidity risk, company-specific risk)
4. Evaluasi apakah stop loss realistis
5. Rekomendasi mitigasi risiko (hedging, diversification, staggered entry)

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "risk_score": 0-100,
  "position_pct": 5-25 (persentase maksimum dari portfolio),
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	userPrompt := fmt.Sprintf(`EVALUASI RISIKO untuk saham %s (%s) — harga saat ini: Rp %.2f

SEMUA LAPORAN:
%s

Hitung risk score, position sizing, dan rekomendasi mitigasi. Dalam format JSON.`, code, name, price, reportsText)

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentRisk, "Risk Manager", response)
}

// ── Phase 5: Portfolio Manager ──

func (s *MultiAgentService) runPortfolioManager(code, name string, allReports []AgentReport, debateLog string, pastReflections []string, price float64) AgentReport {
	var reportsText string
	for _, r := range allReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s, Confidence: %.0f)\n%s\n",
			r.Role, r.Agent, r.Signal, r.Confidence, r.Analysis)
	}

	systemPrompt := `Kamu adalah Portfolio Manager senior dengan AUM >Rp 5 Triliun di pasar modal Indonesia.

Kamu adalah PENGAMBIL KEPUTUSAN FINAL. Baca semua laporan + debat + past reflections, lalu buat keputusan investasi.

Tugas kamu:
1. Evaluasi SEMUA laporan dari seluruh tim (analis + researcher + trader + risk manager)
2. Baca PAST REFLECTIONS — pelajari dari keputusan sebelumnya
3. Baca DEBATE LOG — pahami argumen bullish vs bearish
4. Buat KEPUTUSAN FINAL: BUY, SELL, atau HOLD
5. Berikan RATIONALE yang komprehensif — termasuk lessons learned dari past reflections
6. Tentukan CONFIDENCE level

KAMU ADALAH DECISION MAKER FINAL. Tidak ada yang di atas kamu. Putuskan dengan tegas.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "keputusan final lengkap dengan rationale dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}`

	pastRefText := "Belum ada refleksi sebelumnya."
	if len(pastReflections) > 0 {
		pastRefText = strings.Join(pastReflections, "\n---\n")
	}

	userPrompt := fmt.Sprintf(`BUAT KEPUTUSAN FINAL untuk saham %s (%s) — harga saat ini: Rp %.2f

SEMUA LAPORAN:
%s

DEBATE LOG:
%s

PAST REFLECTIONS (pembelajaran dari keputusan sebelumnya):
%s

Buat keputusan final sebagai Portfolio Manager. Dalam format JSON.`, code, name, price, reportsText, debateLog, pastRefText)

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentPortfolio, "Portfolio Manager", response)
}

// ── Helpers ──

func (s *MultiAgentService) buildDebateLog(bullish, bearish AgentReport) string {
	return fmt.Sprintf(`=== DEBATE: BULLISH VS BEARISH ===

>> BULLISH RESEARCHER <<
Signal: %s | Confidence: %.0f
%s

Key Points:
%s

Risk Flags:
%s

---

>> BEARISH RESEARCHER <<
Signal: %s | Confidence: %.0f
%s

Key Points:
%s

Risk Flags:
%s

=== END DEBATE ===`,
		bullish.Signal, bullish.Confidence, bullish.Analysis,
		strings.Join(bullish.KeyPoints, "\n- "),
		strings.Join(bullish.RiskFlags, "\n- "),
		bearish.Signal, bearish.Confidence, bearish.Analysis,
		strings.Join(bearish.KeyPoints, "\n- "),
		strings.Join(bearish.RiskFlags, "\n- "))
}

func (s *MultiAgentService) callAgent(systemPrompt, userPrompt string) string {
	if !s.AI.IsConfigured() {
		return s.generateFallback(systemPrompt)
	}

	if s.isDeepAgent(systemPrompt) && (s.AI.DeepThinkModel != "" || s.AI.Model != "") {
		if response, err := s.AI.DeepChat(systemPrompt, userPrompt); err == nil {
			return response
		}
	} else if qm := s.AI.QuickModel(); qm != "" {
		if response, err := s.AI.ChatWithModel(systemPrompt, userPrompt, qm); err == nil {
			return response
		}
	}

	response, err := s.AI.Chat(systemPrompt, userPrompt)
	if err != nil {
		return s.generateFallback(systemPrompt)
	}
	return response
}

// isDeepAgent identifies complex-reasoning agents that benefit from a stronger model.
func (s *MultiAgentService) isDeepAgent(systemPrompt string) bool {
	l := strings.ToLower(systemPrompt)
	for _, kw := range []string{
		"bullish researcher", "bearish researcher", "senior trader",
		"risk manager", "portfolio manager",
	} {
		if strings.Contains(l, kw) {
			return true
		}
	}
	return false
}

// callAgentDeep routes complex-reasoning agents (researchers, trader, risk, PM)
// through the deep-think model when configured.
func (s *MultiAgentService) callAgentDeep(systemPrompt, userPrompt string) string {
	if !s.AI.IsConfigured() {
		return s.generateFallback(systemPrompt)
	}

	response, err := s.AI.DeepChat(systemPrompt, userPrompt)
	if err != nil {
		return s.generateFallback(systemPrompt)
	}
	return response
}

// debateRounds returns the configured number of bull/bear debate rounds (default 1, max 3).
func (s *MultiAgentService) debateRounds() int {
	rounds := 1
	if s.AI.SettingRepo != nil {
		if v, err := s.AI.SettingRepo.Get("ai_debate_rounds"); err == nil && v != "" {
			if n := atoiSafe(v); n >= 1 {
				rounds = n
			}
		}
	}
	if rounds > 3 {
		rounds = 3
	}
	return rounds
}

func atoiSafe(s string) int {
	var n int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n
}

func (s *MultiAgentService) generateFallback(systemPrompt string) string {
	sysLower := strings.ToLower(systemPrompt)

	if strings.Contains(sysLower, "fundamental analyst") {
		return s.fallbackFundamental()
	}
	if strings.Contains(sysLower, "technical analyst") {
		return s.fallbackTechnical()
	}
	if strings.Contains(sysLower, "sentiment analyst") {
		return s.fallbackSentiment()
	}
	if strings.Contains(sysLower, "news analyst") {
		return s.fallbackNews()
	}
	if strings.Contains(sysLower, "bullish researcher") || strings.Contains(sysLower, "bull case") {
		return s.fallbackBullish()
	}
	if strings.Contains(sysLower, "bearish researcher") || strings.Contains(sysLower, "bear case") {
		return s.fallbackBearish()
	}
	if strings.Contains(sysLower, "senior trader") || strings.Contains(sysLower, "keputusan trading") {
		return s.fallbackTrader()
	}
	if strings.Contains(sysLower, "risk manager") {
		return s.fallbackRisk()
	}
	if strings.Contains(sysLower, "portfolio manager") {
		return s.fallbackPortfolio()
	}

	return fmt.Sprintf(`{
  "signal": "HOLD",
  "confidence": 50,
  "analysis": "**Agen tidak dapat menghasilkan analisis.** AI Provider belum dikonfigurasi. Silakan tambahkan API key di pengaturan AI untuk mengaktifkan analisis multi-agent.\n\n💡 *Konfigurasikan AI provider (OpenAI, DeepSeek, Claude, dll.) untuk analisis lengkap oleh tim agen AI.*",
  "key_points": ["AI Provider belum dikonfigurasi", "Analisis tidak dapat dilakukan"],
  "risk_flags": ["AI tidak tersedia"]
}`)
}

func (s *MultiAgentService) parseAgentReport(agent AgentRole, roleName, rawJSON string) AgentReport {
	report := AgentReport{
		Agent:      agent,
		Role:       roleName,
		Signal:     "HOLD",
		Confidence: 50,
		KeyPoints:  []string{},
		RiskFlags:  []string{},
	}

	jsonStr := extractJSON(rawJSON)
	if jsonStr == "" {
		report.Analysis = rawJSON
		report.KeyPoints = []string{"Analisis tidak dapat diparse sebagai JSON"}
		report.RiskFlags = []string{"Format respons tidak valid"}
		return report
	}

	var parsed struct {
		Signal     string   `json:"signal"`
		Confidence float64  `json:"confidence"`
		Analysis   string   `json:"analysis"`
		KeyPoints  []string `json:"key_points"`
		RiskFlags  []string `json:"risk_flags"`
		RiskScore  float64  `json:"risk_score"`
		PositionPct float64 `json:"position_pct"`
		EntryPrice float64  `json:"entry_price"`
		TargetPrice float64 `json:"target_price"`
		StopLoss   float64  `json:"stop_loss"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		report.Analysis = rawJSON
		report.KeyPoints = []string{"Gagal parse response"}
		report.RiskFlags = []string{"JSON parse error"}
		return report
	}

	report.Signal = strings.ToUpper(parsed.Signal)
	if report.Signal != "BUY" && report.Signal != "SELL" {
		report.Signal = "HOLD"
	}
	report.Confidence = clamp(parsed.Confidence, 0, 100)
	report.Analysis = parsed.Analysis
	report.KeyPoints = parsed.KeyPoints
	report.RiskFlags = parsed.RiskFlags

	if parsed.Analysis == "" {
		report.Analysis = rawJSON
	}

	return report
}

func (s *MultiAgentService) parseTraderParams(traderReport AgentReport, currentPrice float64) (entry, target, stop, pct float64) {
	rawJSON := extractJSON(traderReport.Analysis)

	var parsed struct {
		EntryPrice  float64 `json:"entry_price"`
		TargetPrice float64 `json:"target_price"`
		StopLoss    float64 `json:"stop_loss"`
		PositionPct float64 `json:"position_pct"`
	}

	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		entry = currentPrice * 0.98
		target = currentPrice * 1.10
		stop = currentPrice * 0.95
		pct = 10
		return
	}

	entry = parsed.EntryPrice
	target = parsed.TargetPrice
	stop = parsed.StopLoss
	pct = parsed.PositionPct

	if entry <= 0 {
		entry = currentPrice
	}
	if target <= 0 {
		target = currentPrice * 1.10
	}
	if stop <= 0 {
		stop = currentPrice * 0.95
	}
	if pct <= 0 {
		pct = 10
	}
	pct = clamp(pct, 1, 30)

	return
}

func (s *MultiAgentService) parseRiskScore(riskReport AgentReport) float64 {
	rawJSON := extractJSON(riskReport.Analysis)

	var parsed struct {
		RiskScore float64 `json:"risk_score"`
	}

	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return 50
	}

	return clamp(parsed.RiskScore, 0, 100)
}

func (s *MultiAgentService) parseSignal(pmReport AgentReport) string {
	sig := strings.ToUpper(pmReport.Signal)
	if sig != "BUY" && sig != "SELL" {
		return "HOLD"
	}
	return sig
}

func (s *MultiAgentService) parseConfidence(pmReport AgentReport) float64 {
	return clamp(pmReport.Confidence, 0, 100)
}

func (s *MultiAgentService) parseConfidenceFromReports(reports []AgentReport) float64 {
	if len(reports) == 0 {
		return 50
	}
	var sum float64
	for _, r := range reports {
		sum += r.Confidence
	}
	return clamp(sum/float64(len(reports)), 0, 100)
}

// ── Consensus & Voting ──

type Vote struct {
	Agent      string  `json:"agent"`
	Vote       string  `json:"vote"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

type ConsensusResult struct {
	BuyVotes    int     `json:"buy_votes"`
	SellVotes   int     `json:"sell_votes"`
	HoldVotes   int     `json:"hold_votes"`
	Consensus   string  `json:"consensus"`
	Confidence  float64 `json:"confidence"`
	VoteDetails []Vote  `json:"vote_details"`
}

func (s *MultiAgentService) GetConsensusVote(code string) (*ConsensusResult, error) {
	decision, err := s.RunAnalysis(code)
	if err != nil {
		return nil, err
	}

	var buyVotes, sellVotes, holdVotes int
	var votes []Vote
	var totalConfidence float64
	reportCount := 0

	for _, report := range decision.Reports {
		vote := Vote{
			Agent:      string(report.Agent),
			Vote:       report.Signal,
			Confidence: report.Confidence,
		}

		if report.Analysis != "" {
			if len(report.Analysis) > 120 {
				vote.Reason = report.Analysis[:120] + "..."
			} else {
				vote.Reason = report.Analysis
			}
		}
		if len(report.KeyPoints) > 0 {
			vote.Reason = report.KeyPoints[0]
		}

		switch report.Signal {
		case "BUY":
			buyVotes++
		case "SELL":
			sellVotes++
		default:
			holdVotes++
		}

		totalConfidence += report.Confidence
		reportCount++
		votes = append(votes, vote)
	}

	var consensus string
	var avgConfidence float64
	if reportCount > 0 {
		avgConfidence = totalConfidence / float64(reportCount)
	}

	buyPct := float64(buyVotes) / float64(len(votes)) * 100
	sellPct := float64(sellVotes) / float64(len(votes)) * 100

	switch {
	case buyPct >= 60:
		consensus = "STRONG_BUY"
	case buyPct >= 40 && buyPct > sellPct:
		consensus = "BUY"
	case sellPct >= 60:
		consensus = "STRONG_SELL"
	case sellPct >= 40 && sellPct > buyPct:
		consensus = "SELL"
	default:
		consensus = "HOLD"
	}

	consensusConfidence := avgConfidence
	if buyPct >= 70 || sellPct >= 70 {
		consensusConfidence = math.Max(avgConfidence, 75)
	} else if buyPct < 40 && sellPct < 40 {
		consensusConfidence = math.Min(avgConfidence, 55)
	}

	return &ConsensusResult{
		BuyVotes:    buyVotes,
		SellVotes:   sellVotes,
		HoldVotes:   holdVotes,
		Consensus:   consensus,
		Confidence:  math.Round(consensusConfidence*10) / 10,
		VoteDetails: votes,
	}, nil
}

// ── Persona Packs ──

var personaPrompts = map[string]string{
	"buffett": `Kamu menganalisis dengan filosofi Warren Buffett — value investing, margin of safety, long-term compounding.
Prinsip:
1. Fokus pada bisnis yang mudah dipahami (circle of competence)
2. Cari economic moat — keunggulan kompetitif yang bertahan
3. Management integrity — manajemen yang jujur dan kompeten
4. Valuasi — beli dengan diskon signifikan dari intrinsic value (margin of safety)
5. Long-term horizon — minimal 5-10 tahun, abaikan noise jangka pendek
6. Hindari spekulasi, leverage, dan market timing
7. Favoritkan perusahaan dengan ROE konsisten >15%, DER rendah, dan FCF kuat
Gaya: Konservatif, sabar, fokus fundamental. Sering gunakan analogi sederhana.
Frasa khas: "Price is what you pay, value is what you get." "Rule #1: never lose money."`,

	"soros": `Kamu menganalisis dengan filosofi George Soros — reflexivity theory, macro-driven, asymmetric bets.
Prinsip:
1. Reflexivity — harga mempengaruhi fundamental dan sebaliknya (feedback loop)
2. Boom-bust cycle — identifikasi gelembung sebelum pecah
3. Global macro — perhatikan suku bunga, nilai tukar, kebijakan moneter global
4. Asymmetric risk/reward — cari trade dengan upside besar dan downside terbatas
5. Decisive — ketika yakin, posisi besar (no half measures)
6. Adaptable — ubah pandangan cepat jika evidence berubah
7. Political economy — kebijakan pemerintah dan geopolitik sangat mempengaruhi pasar
Gaya: Agresif, makro-global, fleksibel. Sering referensi siklus ekonomi dan politik.
Frasa khas: "I'm only rich because I know when I'm wrong." "The worse a situation becomes, the less it takes to turn it around."`,

	"tudor_jones": `Kamu menganalisis dengan filosofi Paul Tudor Jones — macro trading, technical precision, risk-first.
Prinsip:
1. Risk management FIRST — defense wins championships
2. Technical analysis — price action dan market structure lebih penting dari narasi
3. Macro overlay — suku bunga resiko, yield curve, credit spreads
4. Momentum — ikuti tren sampai bukti reversal jelas
5. Position sizing dinamis — kecil saat uncertainty, besar saat konfirmasi
6. 5:1 risk/reward minimum — setiap trade harus punya asymmetric payoff
7. Preservation of capital — jangan pernah biarkan satu trade menghancurkan portfolio
8. Multi-timeframe — konfirmasi dari daily, weekly, monthly chart
Gaya: Disiplin, risk-managed, technical-macro hybrid. Fokus pada risk/reward.
Frasa khas: "The most important rule of trading is to play great defense, not great offense."`,

	"cathie_wood": `Kamu menganalisis dengan filosofi Cathie Wood — disruptive innovation, growth, high-conviction.
Prinsip:
1. Disruptive innovation — identifikasi teknologi yang mengubah industri (AI, genomics, fintech, EV, space)
2. Exponential growth — cari perusahaan dengan TAM besar dan growth >30% YoY
3. Long innovation cycles — S-curve adoption, 5+ year time horizon
4. Conviction-weighted — posisi terbesar di ide dengan keyakinan tertinggi
5. Ignore short-term noise — fokus pada 5-year CAGR, bukan quarterly earnings
6. Network effects & platform economics — bisnis yang semakin kuat seiring pertumbuhan
7. Open-source research — transparansi analisis, sharing ideas
8. Valuation context — high growth justifies high multiples in early innings
Gaya: Optimis, growth-oriented, tech-forward. Fokus pada inovasi dan disruption.
Frasa khas: "Innovation solves problems." "We invest in the future, not the past."`,
}

func (s *MultiAgentService) RunAnalysisWithPersona(code string, persona string) (*MultiAgentDecision, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	personaPrompt, ok := personaPrompts[persona]
	if !ok {
		return s.RunAnalysis(code)
	}

	pastReflections, _ := s.GetAgentMemory(code)

	fundamentalData := s.fetchFundamentalData(stock.ID)
	technicalData := s.fetchTechnicalData(stock.ID)
	sentimentData := s.fetchSentimentData(code)
	newsData := s.fetchNewsData(stock.ID)
	currentPrice := s.getCurrentPrice(stock.ID)

	personaStyle := fmt.Sprintf("\n\n=== PERSONA: %s ===\n%s\n\nANALYZE WITH THIS PERSONA'S PHILOSOPHY AND STYLE.", persona, personaPrompt)

	fundamentalReport := s.runFundamentalAnalystWithPersona(stock.Code, stock.Name, fundamentalData, currentPrice, personaStyle)
	technicalReport := s.runTechnicalAnalystWithPersona(stock.Code, stock.Name, technicalData, currentPrice, personaStyle)
	sentimentReport := s.runSentimentAnalystWithPersona(stock.Code, stock.Name, sentimentData, personaStyle)
	newsReport := s.runNewsAnalystWithPersona(stock.Code, stock.Name, newsData, personaStyle)

	analystReports := []AgentReport{fundamentalReport, technicalReport, sentimentReport, newsReport}

	bullishReport := s.runBullishResearcherWithPersona(stock.Code, stock.Name, analystReports, currentPrice, personaStyle)
	bearishReport := s.runBearishResearcherWithPersona(stock.Code, stock.Name, analystReports, bullishReport, currentPrice, personaStyle)

	debateLog := s.buildDebateLog(bullishReport, bearishReport)

	allReports := append(analystReports, bullishReport, bearishReport)

	traderReport := s.runTraderAgentWithPersona(stock.Code, stock.Name, allReports, currentPrice, personaStyle)
	allReports = append(allReports, traderReport)

	riskReport := s.runRiskManagerWithPersona(stock.Code, stock.Name, allReports, currentPrice, personaStyle)
	allReports = append(allReports, riskReport)

	pmReport := s.runPortfolioManagerWithPersona(stock.Code, stock.Name, allReports, debateLog, pastReflections, currentPrice, personaStyle)
	allReports = append(allReports, pmReport)

	entryPrice, targetPrice, stopLoss, positionPct := s.parseTraderParams(traderReport, currentPrice)
	riskScore := s.parseRiskScore(riskReport)
	finalSignal := s.parseSignal(pmReport)

	decision := &MultiAgentDecision{
		Ticker:          code,
		Date:            time.Now().Format("2006-01-02"),
		Reports:         allReports,
		DebateLog:       debateLog,
		FinalSignal:     finalSignal,
		EntryPrice:      entryPrice,
		TargetPrice:     targetPrice,
		StopLoss:        stopLoss,
		PositionPct:     positionPct,
		Rationale:       fmt.Sprintf("[Persona: %s]\n\n%s", persona, pmReport.Analysis),
		RiskScore:       riskScore,
		PastReflections: pastReflections,
	}

	_ = s.SaveDecision(decision)

	return decision, nil
}

// ── Persona-specific agent runners ──

func (s *MultiAgentService) runFundamentalAnalystWithPersona(code, name string, data map[string]interface{}, price float64, personaStyle string) AgentReport {
	systemPrompt := `Kamu adalah Fundamental Analyst senior di pasar modal Indonesia dengan pengalaman 20+ tahun.

Tugas kamu:
1. Analisis valuasi (PER, PBV, EV/EBITDA)
2. Analisis profitabilitas (ROE, ROA, NPM, EPS growth)
3. Analisis kesehatan keuangan (DER, Current Ratio, FCF)
4. Bandingkan dengan rata-rata industri
5. Beri sinyal: BUY jika fundamental sehat + valuasi wajar, SELL jika fundamental buruk + mahal, HOLD jika mixed

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`Analisis fundamental untuk saham %s (%s).

Harga saat ini: Rp %.2f

Data Fundamental:
%s

Berikan analisis fundamental komprehensif dalam format JSON.`, code, name, price, s.formatMap(data))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentFundamental, "Fundamental Analyst", response)
}

func (s *MultiAgentService) runTechnicalAnalystWithPersona(code, name string, data map[string]interface{}, price float64, personaStyle string) AgentReport {
	systemPrompt := `Kamu adalah Technical Analyst profesional dengan keahlian analisis teknikal multi-timeframe.

Tugas kamu:
1. Analisis tren (MA crossover, ADX, MACD)
2. Analisis momentum (RSI, Stochastic, CCI)
3. Analisis volatility (Bollinger Bands, ATR)
4. Identifikasi pola candlestick dan chart pattern
5. Tentukan support & resistance level
6. Beri sinyal: BUY jika uptrend + momentum kuat, SELL jika downtrend, HOLD jika sideways

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`Analisis teknikal untuk saham %s (%s).

Harga saat ini: Rp %.2f

Data Indikator Teknikal:
%s

Berikan analisis teknikal komprehensif dalam format JSON.`, code, name, price, s.formatMap(data))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentTechnical, "Technical Analyst", response)
}

func (s *MultiAgentService) runSentimentAnalystWithPersona(code, name string, data map[string]interface{}, personaStyle string) AgentReport {
	systemPrompt := `Kamu adalah Sentiment Analyst yang menganalisis sentimen pasar dan media sosial untuk saham Indonesia.

Tugas kamu:
1. Analisis sentimen berita dan media sosial
2. Evaluasi foreign flow (buy/sell)
3. Analisis broker summary
4. Evaluasi social media buzz dan trending topics
5. Beri sinyal: BUY jika sentimen positif kuat, SELL jika sentimen negatif dominan, HOLD jika mixed/netral

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`Analisis sentimen untuk saham %s (%s).

Data Sentimen:
%s

Berikan analisis sentimen dalam format JSON.`, code, name, s.formatMap(data))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentSentiment, "Sentiment Analyst", response)
}

func (s *MultiAgentService) runNewsAnalystWithPersona(code, name string, headlines []string, personaStyle string) AgentReport {
	systemPrompt := `Kamu adalah News Analyst yang menganalisis dampak berita terhadap pergerakan saham Indonesia.

Tugas kamu:
1. Ringkas berita-berita terbaru terkait saham
2. Kategorikan dampak: positif, negatif, atau netral
3. Identifikasi katalis jangka pendek
4. Evaluasi risiko dari sisi berita (regulasi, lawsuit, restrukturisasi)
5. Beri sinyal berdasarkan sentimen berita

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`Analisis berita untuk saham %s (%s).

Headlines:
%s

Berikan analisis berita dalam format JSON.`, code, name, strings.Join(headlines, "\n"))

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentNews, "News Analyst", response)
}

func (s *MultiAgentService) runBullishResearcherWithPersona(code, name string, analystReports []AgentReport, price float64, personaStyle string) AgentReport {
	var reportsText string
	for _, r := range analystReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\nKey Points: %s\nRisk Flags: %s\n",
			r.Role, r.Agent, r.Signal, r.Analysis,
			strings.Join(r.KeyPoints, ", "),
			strings.Join(r.RiskFlags, ", "))
	}

	systemPrompt := `Kamu adalah Bullish Researcher — tugasmu membangun kasus terkuat MENGAPA saham ini layak DIBELI.

Baca laporan dari 4 analis (Fundamental, Technical, Sentiment, News). Cari semua argumen positif, bahkan dari analis yang memberi sinyal SELL.

Tugas kamu:
1. Bangun bull case yang komprehensif berdasarkan data dari semua analis
2. Counter bearish arguments dengan data dan logika
3. Identifikasi katalis potensial yang belum disebutkan
4. Beri target harga upside dan timeframe
5. Evaluasi peluang vs risiko dari perspektif bullish

TETAP OBJEKTIF — jangan mengabaikan risiko. Akui risiko, tapi jelaskan mengapa reward lebih besar.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`BANGUN BULL CASE untuk saham %s (%s) — harga saat ini: Rp %.2f

LAPORAN ANALIS:
%s

Bangun kasus bullish terkuat dalam format JSON.`, code, name, price, reportsText)

	response := s.callAgent(systemPrompt, userPrompt)
	report := s.parseAgentReport(AgentBullish, "Bullish Researcher", response)
	report.Signal = "BUY"
	return report
}

func (s *MultiAgentService) runBearishResearcherWithPersona(code, name string, analystReports []AgentReport, bullishReport AgentReport, price float64, personaStyle string) AgentReport {
	var reportsText string
	for _, r := range analystReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\nKey Points: %s\nRisk Flags: %s\n",
			r.Role, r.Agent, r.Signal, r.Analysis,
			strings.Join(r.KeyPoints, ", "),
			strings.Join(r.RiskFlags, ", "))
	}

	systemPrompt := `Kamu adalah Bearish Researcher — tugasmu membangun kasus terkuat MENGAPA saham ini BERISIKO dan layak DIJUAL/DIHINDARI.

Baca laporan dari 4 analis + Bullish Researcher. Cari semua risiko, red flags, dan kelemahan — bahkan dari analis yang memberi sinyal BUY.

Tugas kamu:
1. Bangun bear case yang komprehensif
2. Counter bullish arguments satu per satu dengan data
3. Identifikasi risiko tersembunyi yang belum disebutkan
4. Beri skenario worst-case dengan target harga downside
5. Evaluasi apakah risk/reward ratio tidak favorable

TETAP RASIONAL — jangan takut-takuti. Gunakan data untuk mendukung argumen.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`BANGUN BEAR CASE untuk saham %s (%s) — harga saat ini: Rp %.2f

LAPORAN ANALIS:
%s

LAPORAN BULLISH RESEARCHER:
Signal: %s | Confidence: %.0f
%s
Key Points: %s
Risk Flags: %s

Bangun kasus bearish terkuat dalam format JSON. COUNTER setiap argumen bullish.`, code, name, price, reportsText,
		bullishReport.Signal, bullishReport.Confidence, bullishReport.Analysis,
		strings.Join(bullishReport.KeyPoints, ", "),
		strings.Join(bullishReport.RiskFlags, ", "))

	response := s.callAgent(systemPrompt, userPrompt)
	report := s.parseAgentReport(AgentBearish, "Bearish Researcher", response)
	report.Signal = "SELL"
	return report
}

func (s *MultiAgentService) runTraderAgentWithPersona(code, name string, allReports []AgentReport, price float64, personaStyle string) AgentReport {
	var reportsText string
	for _, r := range allReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s, Confidence: %.0f)\n%s\n",
			r.Role, r.Agent, r.Signal, r.Confidence, r.Analysis)
	}

	systemPrompt := `Kamu adalah Senior Trader profesional dengan pengalaman 15+ tahun di bursa saham Indonesia.

Kamu MEMBACA SEMUA laporan analis + debat bullish/bearish dan membuat KEPUTUSAN TRADING.

Tugas kamu:
1. Evaluasi SEMUA laporan (fundamental, teknikal, sentimen, news, bullish, bearish)
2. Tetapkan sinyal final: BUY, SELL, atau HOLD
3. Tetapkan ENTRY PRICE (harga masuk yang ideal)
4. Tetapkan TARGET PRICE (harga jual/take profit)
5. Tetapkan STOP LOSS (batas kerugian maksimal)
6. Jelaskan rationale keputusan

KAMU MEMBUAT KEPUTUSAN — tidak ragu-ragu.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "entry_price": float,
  "target_price": float,
  "stop_loss": float,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`BUAT KEPUTUSAN TRADING untuk saham %s (%s) — harga saat ini: Rp %.2f

SEMUA LAPORAN:
%s

Buat keputusan trading final. Tetapkan entry, target, stop loss. Dalam format JSON.`, code, name, price, reportsText)

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentTrader, "Trader", response)
}

func (s *MultiAgentService) runRiskManagerWithPersona(code, name string, allReports []AgentReport, price float64, personaStyle string) AgentReport {
	var reportsText string
	for _, r := range allReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s)\n%s\nRisk Flags: %s\n",
			r.Role, r.Agent, r.Signal, r.Analysis,
			strings.Join(r.RiskFlags, ", "))
	}

	systemPrompt := `Kamu adalah Risk Manager profesional. Tugasmu mengevaluasi RISIKO dari keputusan trading.

Tugas kamu:
1. Hitung risk score 0-100 (semakin tinggi = semakin berisiko)
2. Evaluasi position sizing yang aman (% dari portfolio)
3. Identifikasi semua risk factors (market risk, liquidity risk, company-specific risk)
4. Evaluasi apakah stop loss realistis
5. Rekomendasi mitigasi risiko

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "risk_score": 0-100,
  "position_pct": 5-25,
  "analysis": "analisis lengkap dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	userPrompt := fmt.Sprintf(`EVALUASI RISIKO untuk saham %s (%s) — harga saat ini: Rp %.2f

SEMUA LAPORAN:
%s

Hitung risk score, position sizing, dan rekomendasi mitigasi. Dalam format JSON.`, code, name, price, reportsText)

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentRisk, "Risk Manager", response)
}

func (s *MultiAgentService) runPortfolioManagerWithPersona(code, name string, allReports []AgentReport, debateLog string, pastReflections []string, price float64, personaStyle string) AgentReport {
	var reportsText string
	for _, r := range allReports {
		reportsText += fmt.Sprintf("\n### %s (%s — Signal: %s, Confidence: %.0f)\n%s\n",
			r.Role, r.Agent, r.Signal, r.Confidence, r.Analysis)
	}

	systemPrompt := `Kamu adalah Portfolio Manager senior dengan AUM >Rp 5 Triliun di pasar modal Indonesia.

Kamu adalah PENGAMBIL KEPUTUSAN FINAL.

Tugas kamu:
1. Evaluasi SEMUA laporan dari seluruh tim
2. Baca PAST REFLECTIONS
3. Baca DEBATE LOG
4. Buat KEPUTUSAN FINAL: BUY, SELL, atau HOLD
5. Berikan RATIONALE yang komprehensif

KAMU ADALAH DECISION MAKER FINAL.

Output WAJIB dalam format JSON:
{
  "signal": "BUY/SELL/HOLD",
  "confidence": 0-100,
  "analysis": "keputusan final lengkap dengan rationale dalam markdown bahasa Indonesia",
  "key_points": ["poin1", "poin2", ...],
  "risk_flags": ["flag1", "flag2", ...]
}` + personaStyle

	pastRefText := "Belum ada refleksi sebelumnya."
	if len(pastReflections) > 0 {
		pastRefText = strings.Join(pastReflections, "\n---\n")
	}

	userPrompt := fmt.Sprintf(`BUAT KEPUTUSAN FINAL untuk saham %s (%s) — harga saat ini: Rp %.2f

SEMUA LAPORAN:
%s

DEBATE LOG:
%s

PAST REFLECTIONS:
%s

Buat keputusan final sebagai Portfolio Manager. Dalam format JSON.`, code, name, price, reportsText, debateLog, pastRefText)

	response := s.callAgent(systemPrompt, userPrompt)
	return s.parseAgentReport(AgentPortfolio, "Portfolio Manager", response)
}

// ── Streaming Analysis (SSE) ──

func (s *MultiAgentService) StreamAnalysis(w io.Writer, code string) error {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		s.sendSSE(w, "error", map[string]string{"message": fmt.Sprintf("stock not found: %s", code)})
		return err
	}

	s.sendSSE(w, "status", map[string]string{"phase": "init", "message": fmt.Sprintf("Memulai analisis multi-agent untuk %s (%s)...", stock.Code, stock.Name)})

	pastReflections, _ := s.GetAgentMemory(code)

	fundamentalData := s.fetchFundamentalData(stock.ID)
	technicalData := s.fetchTechnicalData(stock.ID)
	sentimentData := s.fetchSentimentData(code)
	newsData := s.fetchNewsData(stock.ID)
	currentPrice := s.getCurrentPrice(stock.ID)

	s.sendSSE(w, "status", map[string]string{"phase": "fundamental_analyst", "status": "analyzing", "agent": "Fundamental Analyst"})
	fundamentalReport := s.runFundamentalAnalyst(stock.Code, stock.Name, fundamentalData, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "fundamental_analyst",
		"status": "complete",
		"agent":  fundamentalReport,
	})

	s.sendSSE(w, "status", map[string]string{"phase": "technical_analyst", "status": "analyzing", "agent": "Technical Analyst"})
	technicalReport := s.runTechnicalAnalyst(stock.Code, stock.Name, technicalData, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "technical_analyst",
		"status": "complete",
		"agent":  technicalReport,
	})

	s.sendSSE(w, "status", map[string]string{"phase": "sentiment_analyst", "status": "analyzing", "agent": "Sentiment Analyst"})
	sentimentReport := s.runSentimentAnalyst(stock.Code, stock.Name, sentimentData)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "sentiment_analyst",
		"status": "complete",
		"agent":  sentimentReport,
	})

	s.sendSSE(w, "status", map[string]string{"phase": "news_analyst", "status": "analyzing", "agent": "News Analyst"})
	newsReport := s.runNewsAnalyst(stock.Code, stock.Name, newsData)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "news_analyst",
		"status": "complete",
		"agent":  newsReport,
	})

	analystReports := []AgentReport{fundamentalReport, technicalReport, sentimentReport, newsReport}

	s.sendSSE(w, "status", map[string]string{"phase": "bullish_researcher", "status": "analyzing", "agent": "Bullish Researcher"})
	bullishReport := s.runBullishResearcher(stock.Code, stock.Name, analystReports, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "bullish_researcher",
		"status": "complete",
		"agent":  bullishReport,
	})

	s.sendSSE(w, "status", map[string]string{"phase": "bearish_researcher", "status": "analyzing", "agent": "Bearish Researcher"})
	bearishReport := s.runBearishResearcher(stock.Code, stock.Name, analystReports, bullishReport, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "bearish_researcher",
		"status": "complete",
		"agent":  bearishReport,
	})

	debateLog := s.buildDebateLog(bullishReport, bearishReport)
	allReports := append(analystReports, bullishReport, bearishReport)

	s.sendSSE(w, "status", map[string]string{"phase": "trader", "status": "analyzing", "agent": "Trader"})
	traderReport := s.runTraderAgent(stock.Code, stock.Name, allReports, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "trader",
		"status": "complete",
		"agent":  traderReport,
	})
	allReports = append(allReports, traderReport)

	s.sendSSE(w, "status", map[string]string{"phase": "risk_manager", "status": "analyzing", "agent": "Risk Manager"})
	riskReport := s.runRiskManager(stock.Code, stock.Name, allReports, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "risk_manager",
		"status": "complete",
		"agent":  riskReport,
	})
	allReports = append(allReports, riskReport)

	s.sendSSE(w, "status", map[string]string{"phase": "portfolio_manager", "status": "analyzing", "agent": "Portfolio Manager"})
	pmReport := s.runPortfolioManager(stock.Code, stock.Name, allReports, debateLog, pastReflections, currentPrice)
	s.sendSSE(w, "partial", map[string]interface{}{
		"phase":  "portfolio_manager",
		"status": "complete",
		"agent":  pmReport,
	})
	allReports = append(allReports, pmReport)

	entryPrice, targetPrice, stopLoss, positionPct := s.parseTraderParams(traderReport, currentPrice)
	riskScore := s.parseRiskScore(riskReport)
	finalSignal := s.parseSignal(pmReport)

	decision := &MultiAgentDecision{
		Ticker:          code,
		Date:            time.Now().Format("2006-01-02"),
		Reports:         allReports,
		DebateLog:       debateLog,
		FinalSignal:     finalSignal,
		EntryPrice:      entryPrice,
		TargetPrice:     targetPrice,
		StopLoss:        stopLoss,
		PositionPct:     positionPct,
		Rationale:       pmReport.Analysis,
		RiskScore:       riskScore,
		PastReflections: pastReflections,
	}

	s.sendSSE(w, "complete", map[string]interface{}{
		"phase":    "complete",
		"status":   "done",
		"decision": decision,
	})

	_ = s.SaveDecision(decision)

	s.sendSSE(w, "done", map[string]string{"message": "Stream selesai"})

	return nil
}

func (s *MultiAgentService) sendSSE(w io.Writer, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))

	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
}

// ── Parallel Agent Analysis ──

func (s *MultiAgentService) RunAnalysisParallel(code string) (*MultiAgentDecision, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", code)
	}

	pastReflections, _ := s.GetAgentMemory(code)

	fundamentalData := s.fetchFundamentalData(stock.ID)
	technicalData := s.fetchTechnicalData(stock.ID)
	sentimentData := s.fetchSentimentData(code)
	newsData := s.fetchNewsData(stock.ID)
	currentPrice := s.getCurrentPrice(stock.ID)

	var (
		fundamentalReport, technicalReport, sentimentReport, newsReport AgentReport
		wg                                                             sync.WaitGroup
	)

	wg.Add(4)
	go func() { defer wg.Done(); fundamentalReport = s.runFundamentalAnalyst(stock.Code, stock.Name, fundamentalData, currentPrice) }()
	go func() { defer wg.Done(); technicalReport = s.runTechnicalAnalyst(stock.Code, stock.Name, technicalData, currentPrice) }()
	go func() { defer wg.Done(); sentimentReport = s.runSentimentAnalyst(stock.Code, stock.Name, sentimentData) }()
	go func() { defer wg.Done(); newsReport = s.runNewsAnalyst(stock.Code, stock.Name, newsData) }()
	wg.Wait()

	analystReports := []AgentReport{fundamentalReport, technicalReport, sentimentReport, newsReport}

	bullishReport := s.runBullishResearcher(stock.Code, stock.Name, analystReports, currentPrice)
	bearishReport := s.runBearishResearcher(stock.Code, stock.Name, analystReports, bullishReport, currentPrice)

	debateLog := s.buildDebateLog(bullishReport, bearishReport)

	allReports := append(analystReports, bullishReport, bearishReport)

	traderReport := s.runTraderAgent(stock.Code, stock.Name, allReports, currentPrice)
	allReports = append(allReports, traderReport)

	riskReport := s.runRiskManager(stock.Code, stock.Name, allReports, currentPrice)
	allReports = append(allReports, riskReport)

	pmReport := s.runPortfolioManager(stock.Code, stock.Name, allReports, debateLog, pastReflections, currentPrice)
	allReports = append(allReports, pmReport)

	entryPrice, targetPrice, stopLoss, positionPct := s.parseTraderParams(traderReport, currentPrice)
	riskScore := s.parseRiskScore(riskReport)
	finalSignal := s.parseSignal(pmReport)

	decision := &MultiAgentDecision{
		Ticker:          code,
		Date:            time.Now().Format("2006-01-02"),
		Reports:         allReports,
		DebateLog:       debateLog,
		FinalSignal:     finalSignal,
		EntryPrice:      entryPrice,
		TargetPrice:     targetPrice,
		StopLoss:        stopLoss,
		PositionPct:     positionPct,
		Rationale:       pmReport.Analysis,
		RiskScore:       riskScore,
		PastReflections: pastReflections,
	}

	_ = s.SaveDecision(decision)

	return decision, nil
}

// ── Data Fetchers ──

func (s *MultiAgentService) fetchFundamentalData(stockID int64) map[string]interface{} {
	fundamentals, err := s.StockFundamentalRepo.FindByStockID(stockID, 4)
	if err != nil || len(fundamentals) == 0 {
		return map[string]interface{}{
			"status": "Data fundamental tidak tersedia",
		}
	}

	latest := fundamentals[0]
	data := map[string]interface{}{
		"revenue":          latest.Revenue,
		"net_income":       latest.NetIncome,
		"eps":              latest.EPS,
		"bvps":             latest.BVPS,
		"per":              latest.PER,
		"pbv":              latest.PBV,
		"roe":              latest.ROE,
		"roa":              latest.ROA,
		"der":              latest.DER,
		"net_profit_margin": latest.NetProfitMargin,
		"dividend_yield":   latest.DividendYield,
		"total_assets":     latest.TotalAssets,
		"total_liabilities": latest.TotalLiabilities,
		"equity":           latest.Equity,
		"period":           latest.Period,
	}

	return data
}

func (s *MultiAgentService) fetchTechnicalData(stockID int64) map[string]interface{} {
	prices, err := s.StockPriceRepo.FindLatest(stockID, 90)
	if err != nil || len(prices) < 20 {
		return map[string]interface{}{
			"status": "Data harga tidak mencukupi (minimal 20 hari)",
		}
	}

	latest := prices[0]
	var highest, lowest float64 = latest.Close, latest.Close
	var totalVolume int64
	for _, p := range prices {
		if p.High > highest {
			highest = p.High
		}
		if p.Low < lowest {
			lowest = p.Low
		}
		totalVolume += p.Volume
	}

	avgVolume := totalVolume / int64(len(prices))

	sma20 := s.calcSMA(prices, 20)
	sma50 := s.calcSMA(prices, 50)
	rsi := s.calcRSI(prices, 14)

	trend := "sideways"
	if len(prices) >= 50 {
		startPrice := prices[len(prices)-1].Close
		if sma20 > startPrice*1.1 && sma50 > startPrice*1.05 {
			trend = "uptrend"
		} else if sma20 < startPrice*0.9 && sma50 < startPrice*0.95 {
			trend = "downtrend"
		}
	}

	return map[string]interface{}{
		"latest_close":  latest.Close,
		"latest_high":   latest.High,
		"latest_low":    latest.Low,
		"latest_volume": latest.Volume,
		"highest_90d":   highest,
		"lowest_90d":    lowest,
		"avg_volume_90d": avgVolume,
		"sma_20":        sma20,
		"sma_50":        sma50,
		"rsi_14":        rsi,
		"trend":         trend,
		"data_points":   len(prices),
	}
}

func (s *MultiAgentService) fetchSentimentData(code string) map[string]interface{} {
	return map[string]interface{}{
		"market_sentiment":   "netral",
		"social_media_buzz":  "low",
		"foreign_flow":       "balanced",
		"broker_consensus":   "hold",
		"analyst_recommendations": "mixed",
		"sentiment_score":    0.0,
		"note":              "Sentimen real-time memerlukan konfigurasi API",
	}
}

func (s *MultiAgentService) fetchNewsData(stockID int64) []string {
	news, err := s.NewsRepo.FindByStockID(stockID, 5)
	if err != nil || len(news) == 0 {
		return []string{"Tidak ada berita terbaru untuk saham ini."}
	}

	var headlines []string
	for _, n := range news {
		headlines = append(headlines, fmt.Sprintf("- [%s] %s (Sumber: %s)", n.PublishedAt.Format("2006-01-02"), n.Title, n.Source))
	}
	return headlines
}

func (s *MultiAgentService) getCurrentPrice(stockID int64) float64 {
	price, err := s.StockPriceRepo.GetLatestPrice(stockID)
	if err != nil {
		return 0
	}
	return price
}

// ── Technical Calculator Helpers ──

func (s *MultiAgentService) calcSMA(prices []model.StockPrice, period int) float64 {
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

func (s *MultiAgentService) calcRSI(prices []model.StockPrice, period int) float64 {
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

// ── Fallback Responses ──

func (s *MultiAgentService) fallbackFundamental() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "analysis": "**Analisis Fundamental (Data-Based)**\n\nAnalisis fundamental dilakukan berdasarkan data keuangan yang tersedia di database.\n\nMetrik yang dievaluasi:\n- **Valuasi**: PER, PBV dibandingkan rata-rata sektor\n- **Profitabilitas**: ROE, ROA, Net Profit Margin\n- **Kesehatan**: DER, Current Ratio\n- **Pertumbuhan**: EPS growth, Revenue growth\n\n💡 Untuk analisis fundamental yang lebih mendalam dengan reasoning naratif, konfigurasikan AI Provider di pengaturan.",
  "key_points": ["Evaluasi PER dan PBV terhadap sektor", "Analisis ROE dan profitabilitas", "Cek DER dan kesehatan neraca"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackTechnical() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "analysis": "**Analisis Teknikal (Data-Based)**\n\nIndikator teknikal dihitung berdasarkan data harga historis.\n\nIndikator yang dihitung:\n- **SMA 20 & 50**: Untuk identifikasi tren\n- **RSI 14**: Untuk momentum dan overbought/oversold\n- **Support & Resistance**: Level kunci\n- **Volume Analysis**: Konfirmasi pergerakan\n\n💡 Untuk identifikasi pola candlestick dan interpretasi teknikal yang lebih canggih, konfigurasikan AI Provider.",
  "key_points": ["SMA crossover untuk sinyal tren", "RSI untuk momentum", "Support/resistance level"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackSentiment() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "analysis": "**Analisis Sentimen (Data-Based)**\n\nSentimen diukur dari analisis teks berita dan data pasar.\n\nSumber data:\n- Berita pasar modal\n- Foreign flow (net buy/sell)\n- Broker summary\n\n💡 Untuk analisis sentimen yang lebih nuanced dengan NLP, konfigurasikan AI Provider.",
  "key_points": ["Sentimen berita", "Foreign flow analysis", "Broker consensus"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackNews() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "analysis": "**Analisis Berita (Data-Based)**\n\nRingkasan berita terbaru terkait saham.\n\nFokus analisis:\n- Katalis jangka pendek\n- Risiko regulasi\n- Perubahan fundamental bisnis\n\n💡 Untuk analisis dampak berita yang lebih detail, konfigurasikan AI Provider.",
  "key_points": ["Berita terbaru", "Katalis potensial", "Risiko dari berita"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackBullish() string {
	return `{
  "signal": "BUY",
  "confidence": 50,
  "analysis": "**Bull Case (Data-Based)**\n\nBerdasarkan data yang tersedia:\n- Valuasi dibandingkan peers\n- Momentum teknikal\n- Sentimen pasar\n\nArgumen bullish dibangun dari konsensus data yang tersedia.\n\n💡 Untuk bull case yang lebih komprehensif dengan debate synthesis, konfigurasikan AI Provider.",
  "key_points": ["Valuasi relatif", "Momentum positif", "Katalis potensial"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackBearish() string {
	return `{
  "signal": "SELL",
  "confidence": 50,
  "analysis": "**Bear Case (Data-Based)**\n\nBerdasarkan data yang tersedia:\n- Risiko valuasi\n- Sinyal teknikal negatif\n- Sentimen pasar\n\nArgumen bearish dibangun dari konsensus data yang tersedia.\n\n💡 Untuk bear case yang lebih komprehensif dengan counterpoints, konfigurasikan AI Provider.",
  "key_points": ["Risiko valuasi", "Momentum negatif", "Red flags"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackTrader() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "entry_price": 0,
  "target_price": 0,
  "stop_loss": 0,
  "analysis": "**Keputusan Trading (Data-Based)**\n\nBerdasarkan konsensus analis:\n- Rekomendasi: HOLD sampai konfirmasi lebih jelas\n- Entry point: menunggu pullback ke support\n- Risk/reward: perlu evaluasi lebih lanjut\n\n💡 Untuk trading plan yang detail dengan level harga spesifik, konfigurasikan AI Provider.",
  "key_points": ["HOLD untuk saat ini", "Tunggu konfirmasi sinyal", "Evaluasi risk/reward"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackRisk() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "risk_score": 50,
  "position_pct": 10,
  "analysis": "**Risk Assessment (Data-Based)**\n\nEvaluasi risiko berdasarkan data yang tersedia:\n- **Risk Score**: 50/100 (moderate)\n- **Position Limit**: 10% dari portfolio\n- **Stop Loss**: 5% dari entry\n\n💡 Untuk risk assessment yang lebih granular dengan scenario analysis, konfigurasikan AI Provider.",
  "key_points": ["Risk score moderate", "Posisi maksimal 10%", "Stop loss 5%"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

func (s *MultiAgentService) fallbackPortfolio() string {
	return `{
  "signal": "HOLD",
  "confidence": 50,
  "analysis": "**Portfolio Manager Decision**\n\nSetelah mengevaluasi semua laporan analis:\n\n**Keputusan: HOLD**\n\nAlasan:\n- Data analis tidak memberikan sinyal yang cukup kuat\n- Risk/reward ratio perlu evaluasi lebih detail\n- Menunggu konfirmasi dari multiple timeframe\n\n**Rencana Aksi:**\n1. Monitor pergerakan harga harian\n2. Review ulang dalam 1 minggu\n3. Siapkan buy plan jika harga turun ke support\n\n💡 Untuk portfolio management decision yang komprehensif dengan multi-agent debate, konfigurasikan AI Provider di pengaturan.",
  "key_points": ["HOLD dengan monitoring ketat", "Review mingguan", "Siapkan buy plan di support"],
  "risk_flags": ["AI Provider belum dikonfigurasi"]
}`
}

// ── Utility Functions ──

func (s *MultiAgentService) formatMap(data map[string]interface{}) string {
	var lines []string
	for k, v := range data {
		lines = append(lines, fmt.Sprintf("- **%s**: %v", k, v))
	}
	return strings.Join(lines, "\n")
}

func extractJSON(raw string) string {
	raw = strings.TrimSpace(raw)

	start := strings.Index(raw, "{")
	if start == -1 {
		return ""
	}

	end := strings.LastIndex(raw, "}")
	if end == -1 || end <= start {
		return ""
	}

	return raw[start : end+1]
}

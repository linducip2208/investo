package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"

	"investo/internal/config"
	"investo/internal/database"
	"investo/internal/handler"
	mw "investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/pseo"
	"investo/internal/repository"
	"investo/internal/service"
	"investo/internal/service/pattern"
	"investo/internal/service/scraper"
	"investo/internal/service/seo"
	"investo/web"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	db, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	} else {
		log.Println("Migrations completed")
	}

	stockRepo := &repository.StockRepository{DB: db}
	stockPriceRepo := &repository.StockPriceRepository{DB: db}
	stockFundamentalRepo := &repository.StockFundamentalRepository{DB: db}

	// Auto-seed if no stocks exist — run in background so server starts immediately
	allStocks, _ := stockRepo.ListActive()
	if len(allStocks) == 0 {
		log.Println("[AutoSeed] No stocks found — seeding in background...")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[AutoSeed] PANIC: %v", r)
				}
			}()
			autoSeed(db)
		}()
	}

	wsHub := NewWSHub()
	sectorRepo := &repository.SectorRepository{DB: db}
	userRepo := &repository.UserRepository{DB: db}
	portfolioRepo := &repository.PortfolioRepository{DB: db}
	portfolioItemRepo := &repository.PortfolioItemRepository{DB: db}
	watchlistRepo := &repository.WatchlistRepository{DB: db}
	watchlistItemRepo := &repository.WatchlistItemRepository{DB: db}
	alertRepo := &repository.AlertRepository{DB: db}
	newsRepo := &repository.NewsRepository{DB: db}
	forexRepo := &repository.ForexRepository{DB: db}
	blogRepo := &repository.BlogRepository{DB: db}
	screenerRepo := &repository.ScreenerRepository{DB: db}
	settingRepo := &repository.SettingRepository{DB: db}
	indexNowSvc := seo.NewIndexNowService(settingRepo, cfg.AppURL)
	communityRepo := &repository.CommunityRepository{DB: db}
	stockActionRepo := &repository.StockActionRepository{DB: db}
	tradeJournalRepo := &repository.TradeJournalRepository{DB: db}
	agentDecisionRepo := &repository.AgentDecisionRepository{DB: db}
	agentRunCheckpointRepo := &repository.AgentRunCheckpointRepository{DB: db}
	agentMandateRepo := &repository.AgentMandateRepository{DB: db}
	approvalRepo := &repository.ApprovalRepository{DB: db}
	paperTradingRepo := &repository.PaperTradingRepository{DB: db}

	authService := &service.AuthService{UserRepo: userRepo}
	emailService := service.NewEmailService(db, cfg.AppName, cfg.AppURL)
	totpService := service.NewTOTPService(db, cfg.AppName)
	chatService := service.NewChatService(db)
	pdfExportService := service.NewPDFExportService(cfg.AppName)
	chartService := &service.ChartService{StockPriceRepo: stockPriceRepo}
	patternService := &pattern.PatternService{}
	patternScannerService := &service.PatternScannerService{
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
		PatternService: patternService,
	}
	patternAnalyticsService := &service.PatternAnalyticsService{
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
		PatternService: patternService,
	}
	annotationService := &service.AnnotationService{
		StockActionRepo: stockActionRepo,
	}
	heatmapService := &service.HeatmapService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
		SectorRepo:     sectorRepo,
	}
	breadthService := &service.MarketBreadthService{
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
	}

	sentimentService := &service.SentimentService{}

	anomalyService := &service.AnomalyService{
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
	}

	alertChecker := &service.AlertChecker{
		AlertRepo:         alertRepo,
		StockPriceRepo:    stockPriceRepo,
		StockRepo:         stockRepo,
		WatchlistRepo:     watchlistRepo,
		WatchlistItemRepo: watchlistItemRepo,
		PatternService:    patternService,
		AnomalyService:    anomalyService,
	}

	recapService := &service.RecapService{
		StockRepo:        stockRepo,
		StockPriceRepo:   stockPriceRepo,
		SectorRepo:       sectorRepo,
		ForexRepo:        forexRepo,
		SentimentService: sentimentService,
		BreadthService:   breadthService,
	}

	valuationService := &service.ValuationService{
		StockFundamentalRepo: stockFundamentalRepo,
		StockPriceRepo:       stockPriceRepo,
		StockRepo:            stockRepo,
		SectorRepo:           sectorRepo,
	}

	dividendService := &service.DividendService{
		StockRepo:            stockRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		StockPriceRepo:       stockPriceRepo,
	}

	screenerService := &service.ScreenerService{
		StockRepo:            stockRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		StockPriceRepo:       stockPriceRepo,
		SectorRepo:           sectorRepo,
	}

	notifRepo := &repository.NotificationRepository{DB: db}
	notifService := service.NewNotificationService(notifRepo)

	pseoService := &pseo.Service{
		StockRepo:  stockRepo,
		SectorRepo: sectorRepo,
		BlogRepo:   blogRepo,
		NewsRepo:   newsRepo,
		AppURL:     cfg.AppURL,
	}

	sessionHash := sha256.Sum256([]byte(cfg.SessionSecret))
	sessionStore := sessions.NewCookieStore(sessionHash[:], sessionHash[:16])
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   cfg.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
	}

	authMiddleware := &mw.AuthMiddleware{
		SessionStore: sessionStore,
		UserRepo:     userRepo,
	}

	tpl, err := parseTemplates(cfg.AppURL)
	if err != nil {
		log.Printf("Warning: template parsing failed: %v", err)
		tpl = nil
	} else {
		log.Println("Templates parsed successfully")
	}

	pageHandler := &handler.PageHandler{
		StockRepo:         stockRepo,
		StockPriceRepo:    stockPriceRepo,
		WatchlistRepo:     watchlistRepo,
		WatchlistItemRepo: watchlistItemRepo,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		NewsRepo:          newsRepo,
		SectorRepo:        sectorRepo,
		PSEOService:       pseoService,
		Templates:         tpl,
		NotifService:      notifService,
	}
	authHandler := &handler.AuthHandler{
		AuthService:  authService,
		SessionStore: sessionStore,
		Templates:    tpl,
		EmailService: emailService,
		TOTPService:  totpService,
		SettingRepo:  settingRepo,
		DB:           db,
	}
	stockHandler := &handler.StockHandler{
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		NewsRepo:             newsRepo,
		ChartService:         chartService,
		Templates:            tpl,
	}
	forexAnalytics := &service.ForexAnalytics{
		ForexRepo: forexRepo,
	}

	forexHandler := &handler.ForexHandler{
		ForexRepo:      forexRepo,
		Templates:      tpl,
		ForexAnalytics: forexAnalytics,
	}

	forexToolsService := service.NewForexTools()
	marketProfileService := service.NewMarketProfileService()
	cotDataService := service.NewCOTDataService()

	forexToolsHandler := &handler.ForexToolsHandler{
		Templates:     tpl,
		ForexTools:    forexToolsService,
		MarketProfile: marketProfileService,
		COTData:       cotDataService,
	}

	washSaleService := service.NewWashSaleService(
		paperTradingRepo,
		portfolioItemRepo,
		portfolioRepo,
	)

	betaWeightedService := service.NewBetaWeightedService(
		stockPriceRepo,
		portfolioRepo,
		portfolioItemRepo,
		stockRepo,
		sectorRepo,
	)

	riskToolsHandler := &handler.RiskToolsHandler{
		Templates:           tpl,
		WashSaleService:     washSaleService,
		BetaWeightedService: betaWeightedService,
	}

	predictionService := service.NewPredictionMarketService(db)
	achievementService := service.NewAchievementService(db)
	whatsappService := service.NewWhatsAppBotService(db)
	weatherService := service.NewPortfolioWeatherService(db)

	engagementHandler := &handler.EngagementHandler{
		PredictionService:  predictionService,
		AchievementService: achievementService,
		WhatsAppService:    whatsappService,
		WeatherService:     weatherService,
		Templates:          tpl,
	}
	screenerHandler := &handler.ScreenerHandler{
		ScreenerRepo:         screenerRepo,
		ScreenerService:      screenerService,
		StockRepo:            stockRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		StockPriceRepo:       stockPriceRepo,
		SectorRepo:           sectorRepo,
		DB:                   db,
		Templates:            tpl,
	}
	newsHandler := &handler.NewsHandler{
		NewsRepo:  newsRepo,
		StockRepo: stockRepo,
		Templates: tpl,
	}
	blogHandler := &handler.BlogHandler{
		BlogRepo:  blogRepo,
		Templates: tpl,
	}
	pseoHandler := &handler.PSEOHandler{
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		PSEOService:          pseoService,
		Templates:            tpl,
	}
	portfolioAnalytics := &service.PortfolioAnalytics{
		StockPriceRepo:       stockPriceRepo,
		PortfolioItemRepo:    portfolioItemRepo,
		PortfolioRepo:        portfolioRepo,
		SectorRepo:           sectorRepo,
		StockRepo:            stockRepo,
		StockFundamentalRepo: stockFundamentalRepo,
	}

	riskAnalyzer := &service.RiskAnalyzer{
		StockPriceRepo:    stockPriceRepo,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		SectorRepo:        sectorRepo,
		StockRepo:         stockRepo,
	}

	strategyBuilder := &service.StrategyBuilder{
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
	}

	signalGenerator := &service.SignalGeneratorService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}

	trendScanner := &service.TrendScannerService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}

	portfolioHandler := &handler.PortfolioHandler{
		PortfolioRepo:      portfolioRepo,
		PortfolioItemRepo:  portfolioItemRepo,
		Templates:          tpl,
		PortfolioAnalytics: portfolioAnalytics,
		RiskAnalyzer:       riskAnalyzer,
		TradeJournalRepo:   tradeJournalRepo,
	}
	watchlistHandler := &handler.WatchlistHandler{
		WatchlistRepo:     watchlistRepo,
		WatchlistItemRepo: watchlistItemRepo,
		Templates:         tpl,
		StockRepo:         stockRepo,
	}
	alertEngine := &service.AlertEngine{
		AlertRepo:      alertRepo,
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
		PatternService: patternService,
	}

	adminAnalyticsService := &service.AdminAnalyticsService{DB: db}
	featureFlagService := &service.FeatureFlagService{DB: db}
	customIndicatorService := &service.CustomIndicatorService{}
	customIndicatorRepo := &service.CustomIndicatorRepo{DB: db}

	service.EnsureSessionTable(db)
	service.EnsureCustomIndicatorTable(db)

	telegramService := service.NewTelegramService(cfg.TelegramBotToken)

	alertHandler := &handler.AlertHandler{
		AlertRepo:      alertRepo,
		AlertEngine:    alertEngine,
		TelegramSvc:    telegramService,
		StockPriceRepo: stockPriceRepo,
		StockRepo:      stockRepo,
		Templates:      tpl,
	}
	adminHandler := &handler.AdminHandler{
		UserRepo:             userRepo,
		StockRepo:            stockRepo,
		NewsRepo:             newsRepo,
		BlogRepo:             blogRepo,
		SectorRepo:           sectorRepo,
		SettingRepo:          settingRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		Templates:            tpl,
		FreshnessService: &service.DataFreshnessService{
			StockRepo:      stockRepo,
			StockPriceRepo: stockPriceRepo,
		},
		IntegrityService: &service.DataIntegrityService{
			StockRepo:      stockRepo,
			StockPriceRepo: stockPriceRepo,
		},
		DataMerger: &service.DataMerger{
			StockRepo:       stockRepo,
			StockPriceRepo:  stockPriceRepo,
			StockActionRepo: stockActionRepo,
		},
		AnalyticsService:   adminAnalyticsService,
		FeatureFlagService: featureFlagService,
	}
	apiHandler := &handler.APIHandler{
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		NewsRepo:             newsRepo,
		ForexRepo:            forexRepo,
		ChartService:         chartService,
	}
	analysisHandler := &handler.AnalysisHandler{
		StockRepo:           stockRepo,
		StockPriceRepo:      stockPriceRepo,
		PatternService:      patternService,
		ChartService:        chartService,
		PatternScanner:      patternScannerService,
		PatternAnalytics:    patternAnalyticsService,
		AnnotationService:   annotationService,
		Templates:           tpl,
		CustomIndicatorSvc:  customIndicatorService,
		CustomIndicatorRepo: customIndicatorRepo,
		FeatureFlagService:  featureFlagService,
	}
	marketClock := service.NewMarketClock()
	_ = marketClock

	marketHandler := &handler.MarketHandler{
		HeatmapService:       heatmapService,
		BreadthService:       breadthService,
		StockPriceRepo:       stockPriceRepo,
		StockRepo:            stockRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		ForexRepo:            forexRepo,
		SignalGenerator:      signalGenerator,
		TrendScanner:         trendScanner,
		EconCalendarService:  &service.EconomicCalendarService{},
		Templates:            tpl,
	}

	strategyHandler := &handler.StrategyHandler{
		StrategyBuilder: strategyBuilder,
		Templates:       tpl,
	}

	intelligenceHandler := &handler.IntelligenceHandler{
		RecapService:     recapService,
		AnomalyService:   anomalyService,
		SentimentService: sentimentService,
		BreadthService:   breadthService,
		Templates:        tpl,
	}

	bandarmologiService := &service.BandarmologiService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}
	foreignFlowService := &service.ForeignFlowService{
		DB:        db,
		StockRepo: stockRepo,
	}
	syariahService := &service.SyariahService{
		StockRepo:            stockRepo,
		StockFundamentalRepo: stockFundamentalRepo,
	}
	ipoService := &service.IPOService{}
	rebalanceService := &service.RebalanceService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}
	insiderService := &service.InsiderService{}

	idxHandler := &handler.IDXHandler{
		Bandarmologi: bandarmologiService,
		ForeignFlow:  foreignFlowService,
		Syariah:      syariahService,
		IPO:          ipoService,
		Rebalance:    rebalanceService,
		Insider:      insiderService,
		Templates:    tpl,
	}

	fundamentalHandler := &handler.FundamentalHandler{
		ValuationService:      valuationService,
		PeerComparisonService: valuationService,
		DividendService:       dividendService,
		StockRepo:             stockRepo,
		StockFundamentalRepo:  stockFundamentalRepo,
		StockPriceRepo:        stockPriceRepo,
		SectorRepo:            sectorRepo,
		Templates:             tpl,
	}

	communityHandler := &handler.CommunityHandler{
		CommunityRepo:     communityRepo,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		StockRepo:         stockRepo,
		Templates:         tpl,
		ChatService:       chatService,
	}

	paperTradingSvc := &service.PaperTradingService{
		Repo:           paperTradingRepo,
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}

	paymentSvc := service.NewPaymentService(paperTradingRepo, settingRepo, userRepo, stockRepo)

	portalHandler := &handler.PortalHandler{
		PaperSvc:   paperTradingSvc,
		PaymentSvc: paymentSvc,
		Templates:  tpl,
	}

	exportService := service.NewExportService()
	pdfReportService := &service.PDFReportService{}
	webhookService := service.NewWebhookService()
	jwtService := service.NewJWTService(cfg.JWTSecret)

	apiV1Handler := &handler.APIv1Handler{
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		NewsRepo:             newsRepo,
		ForexRepo:            forexRepo,
		HeatmapService:       heatmapService,
		BreadthService:       breadthService,
		ChartService:         chartService,
		ValuationService:     valuationService,
		PatternService:       patternService,
		ForexAnalytics:       forexAnalytics,
		AuthService:          authService,
		JWTService:           jwtService,
		SettingRepo:          settingRepo,
	}

	exportHandler := &handler.ExportHandler{
		ExportService:        exportService,
		PDFReportService:     pdfReportService,
		PDFExportService:     pdfExportService,
		WebhookService:       webhookService,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
		PortfolioRepo:        portfolioRepo,
		PortfolioItemRepo:    portfolioItemRepo,
		PortfolioAnalytics:   portfolioAnalytics,
		ScreenerService:      screenerService,
		ValuationService:     valuationService,
	}

	calculatorHandler := &handler.CalculatorHandler{Templates: tpl}

	macroDashboardSvc := service.NewMacroDashboardService()
	yieldCurveSvc := service.NewYieldCurveService()
	liquidityFlowSvc := service.NewLiquidityFlowService(db)
	leadingIndicatorsSvc := service.NewLeadingIndicatorsService()
	macroSpilloverSvc := service.NewMacroSpilloverService()
	maScreenerSvc := service.NewMAScreenerService()
	earningsQualitySvc := service.NewEarningsQualityService(stockFundamentalRepo)
	managementQualitySvc := service.NewManagementQualityService(stockFundamentalRepo, stockRepo)
	moatSvc := service.NewMoatService()

	macroHandler := &handler.MacroHandler{
		MacroDashboardSvc:    macroDashboardSvc,
		YieldCurveSvc:        yieldCurveSvc,
		LiquidityFlowSvc:     liquidityFlowSvc,
		LeadingIndicatorsSvc: leadingIndicatorsSvc,
		MacroSpilloverSvc:    macroSpilloverSvc,
		MAScreenerSvc:        maScreenerSvc,
		EarningsQualitySvc:   earningsQualitySvc,
		ManagementQualitySvc: managementQualitySvc,
		MoatSvc:              moatSvc,
		Templates:            tpl,
	}

	botCenterService := service.NewBotCenterService()
	botHandler := &handler.BotHandler{
		SettingRepo:      settingRepo,
		BotCenterService: botCenterService,
		SignalService:    signalGenerator,
		Templates:        tpl,
	}

	googleTrendsSvc := &service.GoogleTrendsService{}
	socialBuzzSvc := &service.SocialBuzzService{}
	economicProxySvc := &service.EconomicProxyService{}
	jobPostingSvc := &service.JobPostingService{}
	backtesterSvc := &service.ThesisBacktesterService{}
	eventStudySvc := &service.EventStudyService{}
	seasonalitySvc := &service.SeasonalityService{}

	alternativeHandler := &handler.AlternativeHandler{
		GoogleTrends:  googleTrendsSvc,
		SocialBuzz:    socialBuzzSvc,
		EconomicProxy: economicProxySvc,
		JobPosting:    jobPostingSvc,
		Backtester:    backtesterSvc,
		EventStudy:    eventStudySvc,
		Seasonality:   seasonalitySvc,
		Templates:     tpl,
	}

	customIndexRepo := &repository.CustomIndexRepository{DB: db}

	fearGreedService := &service.FearGreedService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}

	sectorRotationService := &service.SectorRotationService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
		SectorRepo:     sectorRepo,
	}

	aiService := service.NewAIServiceWithRepo(settingRepo)

	aiUsageSvc := service.InitAIUsageService(db, settingRepo)

	aiAnalysisService := &service.AIAnalysisService{
		AI: aiService,
	}

	aiInteractiveService := &service.AIInteractiveService{
		AI: aiService,
	}

	aiContentSvc := &service.AIContentService{
		AI:             aiService,
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
		StockFundRepo:  stockFundamentalRepo,
		SectorRepo:     sectorRepo,
		NewsRepo:       newsRepo,
		ForexRepo:      forexRepo,
		BreadthSvc:     breadthService,
	}

	aiPersonalized := &service.AIPersonalizedService{
		AI:                aiService,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		TradeJournalRepo:  tradeJournalRepo,
		StockRepo:         stockRepo,
		StockPriceRepo:    stockPriceRepo,
		SectorRepo:        sectorRepo,
	}

	aiToolsSvc := &service.AIToolsService{
		AI:                   aiService,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		PortfolioRepo:        portfolioRepo,
		PortfolioItemRepo:    portfolioItemRepo,
		SectorRepo:           sectorRepo,
	}

	aiDataSvc := &service.AIDataService{
		AI:                   aiService,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		SectorRepo:           sectorRepo,
	}

	multiAgentSvc := service.NewMultiAgentService(
		aiService,
		agentDecisionRepo,
		agentRunCheckpointRepo,
		stockRepo,
		stockPriceRepo,
		stockFundamentalRepo,
		newsRepo,
		sentimentService,
	)

	aiSignalSvc := service.NewAISignalService(
		aiService,
		stockRepo,
		stockPriceRepo,
		stockFundamentalRepo,
		sectorRepo,
		agentDecisionRepo,
	)

	aiForecastSvc := service.NewAIForecastService(
		aiService,
		stockRepo,
		stockPriceRepo,
		stockFundamentalRepo,
		sectorRepo,
	)

	pairTradingService := &service.PairTradingService{
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
		SectorRepo:     sectorRepo,
	}

	factorExposureService := &service.FactorExposureService{
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
	}

	riskDecompService := &service.RiskDecompositionService{
		StockPriceRepo:    stockPriceRepo,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		StockRepo:         stockRepo,
		SectorRepo:        sectorRepo,
	}

	supplyChainService := &service.SupplyChainService{
		StockRepo:  stockRepo,
		SectorRepo: sectorRepo,
	}

	tradeIdeaService := &service.TradeIdeaService{
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
	}

	basketTradingSvc := service.NewBasketTradingService(db, stockRepo, stockPriceRepo)
	service.EnsureBasketTables(db)

	customIndexService := &service.CustomIndexService{
		CustomIndexRepo: customIndexRepo,
		StockRepo:       stockRepo,
		StockPriceRepo:  stockPriceRepo,
	}

	voiceSvc := service.NewAIVoiceService(aiService)
	standupSvc := &service.AIStandupService{
		AI:             aiService,
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
		ForexRepo:      forexRepo,
		PortfolioRepo:  portfolioRepo,
		BreadthSvc:     breadthService,
	}
	taxSvc := &service.AITaxService{
		AI:                aiService,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		StockRepo:         stockRepo,
		StockPriceRepo:    stockPriceRepo,
		StockFundRepo:     stockFundamentalRepo,
	}
	comicSvc := &service.AIComicService{
		AI:        aiService,
		StockRepo: stockRepo,
	}
	storySvc := &service.AIStoryService{
		AI:             aiService,
		StockRepo:      stockRepo,
		StockPriceRepo: stockPriceRepo,
	}
	complianceSvc := &service.AIComplianceService{
		AI:                aiService,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		StockRepo:         stockRepo,
		StockPriceRepo:    stockPriceRepo,
		SectorRepo:        sectorRepo,
	}
	clientReportSvc := &service.AIClientReportService{
		AI:                aiService,
		PortfolioRepo:     portfolioRepo,
		PortfolioItemRepo: portfolioItemRepo,
		StockRepo:         stockRepo,
		StockPriceRepo:    stockPriceRepo,
		StockFundRepo:     stockFundamentalRepo,
	}
	webhookIntelSvc := &service.AIWebhookIntelService{AI: aiService}
	calendarSvc := &service.AICalendarService{AI: aiService}
	botFormatSvc := &service.AIBotFormatService{
		AI:                   aiService,
		StockRepo:            stockRepo,
		StockPriceRepo:       stockPriceRepo,
		StockFundamentalRepo: stockFundamentalRepo,
		PortfolioRepo:        portfolioRepo,
		AppURL:               cfg.AppURL,
	}

	_ = voiceSvc

	aiHandler := &handler.AIHandler{
		AIService:     aiService,
		AIAnalysis:    aiAnalysisService,
		AIInteractive: aiInteractiveService,
		AITools:       aiToolsSvc,
		AIData:        aiDataSvc,
		ContentSvc:    aiContentSvc,
		Personalized:  aiPersonalized,
		MultiAgent:    multiAgentSvc,
		SignalSvc:     aiSignalSvc,
		ForecastSvc:   aiForecastSvc,
		StockRepo:     stockRepo,
		PortfolioRepo: portfolioRepo,
		Templates:     tpl,
		SettingRepo:   settingRepo,
		AIUsageSvc:    aiUsageSvc,
		SimExchange:   newSimExchange(paperTradingRepo, settingRepo, approvalRepo),
		MandateRepo:   agentMandateRepo,
		ApprovalRepo:  approvalRepo,
	}
	aiHandler.SetStandupService(standupSvc)
	aiHandler.SetTaxServiceVar(taxSvc)
	aiHandler.SetComicService(comicSvc)
	aiHandler.SetStoryService(storySvc)
	aiHandler.SetComplianceService(complianceSvc)
	aiHandler.SetClientReportService(clientReportSvc)
	aiHandler.SetWebhookIntelService(webhookIntelSvc)
	aiHandler.SetCalendarService(calendarSvc)
	aiHandler.SetBotFormatService(botFormatSvc)

	beiPipeline := service.NewBEIAgentPipeline(
		aiService, stockRepo, stockPriceRepo, stockFundamentalRepo,
		newsRepo, sentimentService, foreignFlowService,
	)
	forexPipeline := service.NewForexAgentPipeline(
		aiService, stockPriceRepo, forexRepo, newsRepo, sentimentService,
	)
	unifiedRisk := service.NewUnifiedRiskManager()
	signalManager := service.NewSignalManager(
		aiService, beiPipeline, forexPipeline, unifiedRisk, agentDecisionRepo,
	)
	waClient := service.NewWABusinessClient("", "")
	signalDistributor := service.NewSignalDistributor(waClient, webhookService)
	regulatoryComplianceSvc := service.NewComplianceService("BEI")
	mcpConnector := service.NewMCPConnector("", "")

	aiHandler.BEIPipeline = beiPipeline
	aiHandler.ForexPipeline = forexPipeline
	aiHandler.SignalManager = signalManager
	aiHandler.SignalDistributor = signalDistributor
	aiHandler.ComplianceSvc = regulatoryComplianceSvc
	aiHandler.MCPConnector = mcpConnector

	paperTradingHandler := &handler.PaperTradingHandler{
		Service:    paperTradingSvc,
		PaymentSvc: paymentSvc,
		Templates:  tpl,
	}

	paymentHandler := &handler.PaymentHandler{
		Service:   paymentSvc,
		Templates: tpl,
	}

	proMW := mw.RequirePro(paymentSvc)
	whitelabelMW := mw.RequireWhitelabel(paymentSvc)
	_ = whitelabelMW

	advancedHandler := &handler.AdvancedHandler{
		FearGreedSvc:      fearGreedService,
		SectorRotationSvc: sectorRotationService,
		PairTradingSvc:    pairTradingService,
		TradeIdeaSvc:      tradeIdeaService,
		CustomIndexSvc:    customIndexService,
		CustomIndexRepo:   customIndexRepo,
		StockRepo:         stockRepo,
		Templates:         tpl,
		FactorExposureSvc: factorExposureService,
		RiskDecompSvc:     riskDecompService,
		SupplyChainSvc:    supplyChainService,
	}

	tradingHandler := &handler.TradingHandler{
		BasketSvc: basketTradingSvc,
		Templates: tpl,
	}

	r := chi.NewRouter()

	apiRateLimiter := mw.NewRateLimiter(60, 1*time.Minute)

	r.Use(chimw.RealIP)
	r.Use(mw.StackRecoverer)
	r.Use(mw.Logger)
	r.Use(mw.CORS)
	r.Use(mw.CSRFOrigin)

	r.Get("/", pageHandler.Home)
	r.Get("/saham", stockHandler.List)
	r.Get("/saham/{code}", stockHandler.Detail)
	r.Get("/saham/{code}/fundamental", stockHandler.Fundamentals)
	r.Get("/saham/{code}/teknikal", analysisHandler.TechnicalPage)
	r.Get("/saham/{code}/chart-pro", analysisHandler.ChartPro)
	r.Get("/saham/{code}/custom-indicator", analysisHandler.CustomIndicatorPage)
	r.Get("/saham/{code}/berita", newsHandler.ByStock)
	r.Get("/best-saham-{sector}", stockHandler.SectorList)
	r.Get("/best-saham-{sector}-{year}", pseoHandler.SectorYearPage)
	r.Get("/compare/{a}-vs-{b}", stockHandler.Compare)
	r.Get("/alternatives-to-{code}", pseoHandler.AlternativesPage)
	r.Get("/forex", forexHandler.List)
	r.Get("/forex/pro", forexHandler.ProTerminal)
	r.Get("/forex/{base}-{quote}", forexHandler.Detail)
	r.Get("/forex/swap-calc", forexToolsHandler.SwapCalcPage)
	r.Get("/forex/margin-calc", forexToolsHandler.MarginCalcPage)
	r.Get("/forex/pip-calc", forexToolsHandler.PipCalcPage)
	r.Get("/forex/volume-profile", forexToolsHandler.VolumeProfilePage)
	r.Get("/forex/cot", forexToolsHandler.COTPage)
	r.Get("/screener", screenerHandler.Page)
	r.Get("/berita", newsHandler.List)
	r.Get("/berita/{slug}", newsHandler.Detail)
	r.Get("/blog", blogHandler.ListPublic)
	r.Get("/blog/{slug}", blogHandler.DetailPublic)
	r.Get("/blog/category/{slug}", blogHandler.CategoryList)
	r.Get("/docs", pageHandler.Docs)
	r.Get("/faq", pageHandler.FAQ)
	r.Get("/kontak", pageHandler.Contact)
	r.Get("/sitemap.xml", pageHandler.Sitemap)
	r.Get("/robots.txt", pageHandler.Robots)

	r.Get("/verify-email", authHandler.VerifyEmail)
	r.Get("/forgot-password", authHandler.ForgotPasswordPage)
	r.Post("/forgot-password", authHandler.ForgotPassword)
	r.Get("/reset-password", authHandler.ResetPasswordPage)
	r.Post("/reset-password", authHandler.ResetPassword)
	r.Get("/api/auth/google/callback", authHandler.GoogleOAuthCallback)
	r.Get("/api/auth/google", authHandler.GoogleOAuthStart)

	r.Get("/beli-aplikasi-saham", pseoHandler.SourceCodePage)
	r.Get("/beli-aplikasi-forex", pseoHandler.SourceCodePage)
	r.Get("/source-code-trading", pseoHandler.SourceCodePage)
	r.Get("/aplikasi-analisa-saham", pseoHandler.SourceCodePage)
	r.Get("/istilah/{slug}", pseoHandler.GlossaryPage)
	r.Get("/belajar-saham-{city}", pseoHandler.CityPage)
	r.Get("/komunitas-saham-{city}", pseoHandler.CityCommunityPage)
	r.Get("/aplikasi-saham-{city}", pseoHandler.CityAppPage)
	r.Get("/cara/{slug}", pseoHandler.HowToPage)
	r.Get("/market/heatmap", marketHandler.HeatmapPage)
	r.Get("/market/prediction", engagementHandler.PredictionPage)
	r.Get("/market/pattern-scan", analysisHandler.MarketPatternScanPage)
	r.Get("/market/strategy-builder", strategyHandler.BuilderPage)
	r.Get("/market/economic-calendar", marketHandler.EconomicCalendarPage)
	r.Get("/market/correlation", marketHandler.CorrelationPage)
	r.Get("/market/live-signals", marketHandler.LiveSignalsPage)
	r.Get("/market/signal-generator", marketHandler.SignalGeneratorPage)
	r.Get("/market/trend-scanner", marketHandler.TrendScannerPage)
	r.Get("/market/price-action", marketHandler.PriceActionPage)
	r.Get("/market/sector-health", marketHandler.SectorHealthPage)
	r.Get("/market/waran", marketHandler.WaranPage)
	r.Get("/market/fear-greed", advancedHandler.FearGreedPage)
	r.Get("/market/sector-rotation", advancedHandler.SectorRotationPage)
	r.Get("/market/pair-trading", advancedHandler.PairTradingPage)
	r.Get("/market/trade-ideas", advancedHandler.TradeIdeasPage)
	r.Get("/market/gap-scanner", marketHandler.GapScannerPage)
	r.Get("/market/rs-ranking", marketHandler.RSRankingPage)
	r.Get("/market/new-high-low", marketHandler.NewHighLowPage)
	r.Get("/market/pre-market", marketHandler.PreMarketPage)
	r.Get("/market/block-trades", marketHandler.BlockTradesPage)

	r.Get("/market/macro-dashboard", macroHandler.MacroDashboardPage)
	r.Get("/market/yield-curve", macroHandler.YieldCurvePage)
	r.Get("/market/liquidity-flow", macroHandler.LiquidityFlowPage)
	r.Get("/market/leading-indicators", macroHandler.LeadingIndicatorsPage)
	r.Get("/market/macro-spillover", macroHandler.MacroSpilloverPage)
	r.Get("/market/ma-targets", macroHandler.MATargetsPage)
	r.Get("/saham/{code}/moat", macroHandler.MoatPage)
	r.Get("/market/factor-exposure", advancedHandler.FactorExposurePage)
	r.Get("/market/factor-timing", advancedHandler.FactorTimingPage)
	r.Get("/market/smart-beta", advancedHandler.SmartBetaPage)
	r.Get("/market/supply-chain", advancedHandler.SupplyChainPage)
	r.Get("/market/industry-lifecycle", advancedHandler.IndustryLifecyclePage)
	r.Get("/market/competitive", advancedHandler.CompetitivePage)
	r.Get("/api/market/arbitrage", advancedHandler.ArbOpportunitiesJSON)
	r.Get("/ai/compare", aiHandler.ComparePage)
	r.Get("/ai/signal-center", aiHandler.SignalCenterPage)
	r.Get("/ai/coach", aiHandler.CoachPage)
	r.Get("/ai/risk-test", aiHandler.RiskTestPage)
	r.Get("/ai/briefing", aiHandler.DailyBriefingPage)
	r.Get("/ai/thesis", aiHandler.ThesisBuilderPage)
	r.Get("/ai/agents", aiHandler.AgentAnalysisPage)
	r.Get("/saham/{code}/liquidity", stockHandler.LiquidityPage)
	r.Get("/saham/{code}/multi-tf", stockHandler.MultiTimeframePage)
	r.Get("/saham/{code}/ml-predict", stockHandler.MLPredictPage)
	r.Get("/dividen", fundamentalHandler.DividendCalendarPage)
	r.Get("/saham/{code}/valuasi", fundamentalHandler.ValuationPage)
	r.Get("/calculators", calculatorHandler.Hub)
	r.Get("/calculators/right-issue", calculatorHandler.RightIssue)
	r.Get("/calculators/tax", calculatorHandler.TaxCalendar)
	r.Get("/calculators/dcf-builder", macroHandler.DCFBuilderPage)
	r.Get("/calculators/conglomerate", macroHandler.ConglomeratePage)

	r.Get("/market/google-trends", alternativeHandler.GoogleTrendsPage)
	r.Get("/market/social-buzz", alternativeHandler.SocialBuzzPage)
	r.Get("/market/economic-proxy", alternativeHandler.EconomicProxyPage)
	r.Get("/market/job-postings", alternativeHandler.JobPostingsPage)
	r.Get("/market/thesis-backtester", alternativeHandler.ThesisBacktesterPage)
	r.Get("/market/factor-replication", alternativeHandler.FactorReplicationPage)
	r.Get("/market/event-study", alternativeHandler.EventStudyPage)
	r.Get("/saham/{code}/seasonality", alternativeHandler.SeasonalityPage)
	r.Get("/market/cross-asset", alternativeHandler.CrossAssetPage)
	r.Get("/ai/research-paper", alternativeHandler.ResearchPaperPage)

	r.Get("/market/bandarmologi", idxHandler.BandarmologiPage)
	r.Get("/market/saham-gorengan", idxHandler.SahamGorenganPage)
	r.Get("/market/foreign-flow", idxHandler.ForeignFlowPage)
	r.Get("/market/syariah", idxHandler.SyariahPage)
	r.Get("/market/ipo", idxHandler.IPOPage)
	r.Get("/market/rebalance", idxHandler.RebalancePage)
	r.Get("/market/insider", idxHandler.InsiderPage)

	r.Get("/ideas", communityHandler.IdeasList)
	r.Get("/ideas/{id}", communityHandler.IdeasDetail)
	r.Get("/saham/{code}/diskusi", communityHandler.DiscussionList)
	r.Get("/shared/{token}", communityHandler.ViewSharedPortfolio)
	r.Get("/leaderboard", communityHandler.Leaderboard)
	r.Get("/market/recap", intelligenceHandler.DailyRecapPage)
	r.Get("/market/anomalies", intelligenceHandler.AnomaliesPage)
	r.Get("/market/exchanges", botHandler.ExchangesPage)
	r.Get("/market/signal-generator", marketHandler.SignalGeneratorPage)
	r.Get("/pricing", botHandler.PricingPage)
	r.Get("/health", pageHandler.Health)
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(web.Static, "static/favicon.svg")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write(data)
	})
	r.Get("/indexnow-key.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(indexNowSvc.Key()))
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireGuest)

		r.Get("/login", authHandler.LoginPage)
		r.Post("/login", authHandler.Login)
		r.Get("/register", authHandler.RegisterPage)
		r.Post("/register", authHandler.Register)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		r.Get("/dashboard", pageHandler.Dashboard)
		r.Get("/ws", func(w http.ResponseWriter, r *http.Request) { wsHub.HandleWS(w, r) })
		r.Post("/logout", authHandler.Logout)
		r.Get("/profile", authHandler.ProfilePage)
		r.Post("/profile", authHandler.ProfileUpdate)
		r.Get("/pengaturan", pageHandler.SettingsPage)
		r.Post("/pengaturan", pageHandler.SettingsSave)
		r.Get("/pengaturan/ai", aiHandler.AISettingsPage)
		r.Get("/pengaturan/data-sources", aiHandler.DataSourcesPage)
		r.Get("/ai/usage", aiHandler.AIUsagePage)

		r.Get("/ai/search", aiHandler.SearchPage)
		r.Get("/ai/report-generator", aiHandler.ReportGeneratorPage)
		r.Get("/ai/social", aiHandler.SocialPage)
		r.Get("/ai/voice-trade", aiHandler.VoiceTradePage)
		r.Get("/ai/standup", aiHandler.StandupPage)
		r.Get("/ai/tax-optimizer", aiHandler.TaxOptimizerPage)
		r.Get("/ai/comic", aiHandler.ComicPage)
		r.Get("/ai/stock-story/{code}", aiHandler.StockStoryPage)
		r.Get("/ai/compliance", aiHandler.CompliancePage)
		r.Get("/ai/client-report", aiHandler.ClientReportPage)
		r.Get("/ai/webhooks", aiHandler.WebhookIntelPage)
		r.Get("/ai/calendar-sync", aiHandler.CalendarSyncPage)
		r.Get("/ai/bot-format", aiHandler.BotFormatPage)

		r.Get("/trading/basket", tradingHandler.BasketPage)
		r.Get("/trading/orders", tradingHandler.OrdersPage)

		r.Get("/risk/wash-sale", riskToolsHandler.WashSalePage)
		r.Get("/risk/beta-weighted", riskToolsHandler.BetaWeightedPage)
		r.Get("/risk/hedge-finder", riskToolsHandler.HedgeFinderPage)

		r.Get("/dashboard/portfolios", portfolioHandler.List)
		r.Post("/dashboard/portfolios", portfolioHandler.Create)
		r.Get("/dashboard/portfolios/tools", portfolioHandler.ToolsPage)
		r.Get("/dashboard/portfolios/{id}", portfolioHandler.Detail)
		r.Get("/dashboard/portfolios/{id}/analytics", portfolioHandler.Analytics)
		r.Get("/dashboard/portfolios/{id}/risk", portfolioHandler.RiskAnalysis)
		r.Post("/dashboard/portfolios/{id}/items", portfolioHandler.AddItem)
		r.Post("/dashboard/portfolios/import", portfolioHandler.ImportCSV)
		r.Post("/dashboard/portfolios/{id}/items/{itemId}/remove", portfolioHandler.RemoveItem)

		r.Get("/dashboard/journal", portfolioHandler.JournalList)
		r.Post("/dashboard/journal", portfolioHandler.JournalCreate)
		r.Post("/dashboard/journal/{id}/delete", portfolioHandler.JournalDelete)

		r.Get("/dashboard/watchlists", watchlistHandler.List)
		r.Post("/dashboard/watchlists", watchlistHandler.Create)
		r.Post("/dashboard/watchlists/{id}/stocks", watchlistHandler.AddStock)
		r.Post("/dashboard/watchlists/{id}/stocks/{stockId}/remove", watchlistHandler.RemoveStock)

		r.Get("/dashboard/alerts", alertHandler.List)
		r.Post("/dashboard/alerts", alertHandler.Create)
		r.Post("/dashboard/alerts/advanced", alertHandler.CreateAdvanced)
		r.Get("/dashboard/alerts/history", alertHandler.History)
		r.Post("/dashboard/alerts/{id}/toggle", alertHandler.Toggle)
		r.Post("/dashboard/alerts/{id}/delete", alertHandler.Delete)
		r.Post("/api/alerts/{id}/test", alertHandler.TestAlert)
		r.Post("/api/telegram/connect", alertHandler.TelegramConnect)

		r.Get("/dashboard/indices", advancedHandler.CustomIndexListPage)
		r.Post("/dashboard/indices", advancedHandler.CustomIndexCreate)
		r.Get("/api/dashboard/indices/{id}", advancedHandler.CustomIndexDetailJSON)
		r.Post("/dashboard/indices/{id}/delete", advancedHandler.CustomIndexDelete)

		r.Post("/ideas", communityHandler.IdeasCreate)
		r.Post("/ideas/{id}/delete", communityHandler.IdeasDelete)
		r.Post("/ideas/{id}/like", communityHandler.IdeasLike)
		r.Post("/ideas/{id}/comment", communityHandler.IdeasComment)
		r.Post("/ideas/{id}/comments/{commentId}/delete", communityHandler.IdeasCommentDelete)
		r.Post("/saham/{code}/diskusi", communityHandler.DiscussionCreate)
		r.Post("/dashboard/portfolios/{id}/share", communityHandler.SharePortfolio)
		r.Post("/dashboard/portfolios/{id}/unshare", communityHandler.UnsharePortfolio)

		r.Get("/community/chat", communityHandler.ChatPage)
		r.Post("/api/chat/send", communityHandler.ChatSendMessage)
		r.Get("/api/chat/room/{room}", communityHandler.ChatRoomMessages)
		r.Post("/api/users/{id}/follow", communityHandler.FollowUser)
		r.Post("/api/users/{id}/unfollow", communityHandler.UnfollowUser)
		r.Get("/api/dm/{userID}", communityHandler.DMMessages)
		r.Get("/api/dm/conversations", communityHandler.DMConversationsJSON)
		r.Post("/dashboard/watchlists/import", watchlistHandler.ImportCSV)
		r.Get("/api/export/stock/{code}/pdf", exportHandler.ExportStockPDF)
		r.Get("/api/export/portfolio/{id}/pdf", exportHandler.ExportPortfolioPDF)
		r.Post("/api/auth/2fa/enable", authHandler.Enable2FA)
		r.Post("/api/auth/2fa/verify", authHandler.Verify2FA)
		r.Post("/api/auth/2fa/disable", authHandler.Disable2FA)

		r.Get("/bots", botHandler.BotCenterPage)
		r.Get("/api/bots", botHandler.ListBots)
		r.Post("/api/bots", botHandler.CreateBot)
		r.Post("/api/bots/test", botHandler.TestBot)
		r.Post("/api/bots/toggle", botHandler.ToggleBot)
		r.Post("/api/bots/delete", botHandler.DeleteBot)
		r.Get("/api/bots/forward-rules", botHandler.GetForwardRules)
		r.Post("/api/bots/forward-rules", botHandler.SaveForwardRules)
		r.Post("/api/subscribe", botHandler.Subscribe)

		r.Get("/achievements", engagementHandler.AchievementPage)
		r.Get("/market/whatsapp-bot", engagementHandler.WhatsAppBotPage)

		r.Get("/portal", portalHandler.Dashboard)
		r.Get("/portal/portfolios", portalHandler.Portfolios)
		r.Get("/portal/subscription", portalHandler.Subscription)

		r.Get("/api/predictions", engagementHandler.PredictionsJSON)
		r.Post("/api/predictions", engagementHandler.PredictionSubmit)
		r.Get("/api/predictions/leaderboard", engagementHandler.PredictionLeaderboardJSON)
		r.Get("/api/achievements", engagementHandler.AchievementsJSON)
		r.Post("/api/whatsapp/test", engagementHandler.WhatsAppBotTest)

		r.Get("/api/notifications", pageHandler.GetNotifications)
		r.Post("/api/notifications/read-all", pageHandler.MarkAllRead)
		r.Post("/api/notifications/{id}/read", pageHandler.MarkRead)
		r.Get("/api/notifications/unread-count", pageHandler.UnreadCount)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Use(proMW)

		r.Get("/paper-trading", paperTradingHandler.Page)

		r.Post("/api/paper-trading/trade", paperTradingHandler.Trade)
		r.Get("/api/paper-trading/portfolio/{id}", paperTradingHandler.PortfolioJSON)
		r.Get("/api/paper-trading/leaderboard", paperTradingHandler.LeaderboardJSON)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Use(authMiddleware.RequireAdmin)
		ipWhitelist := mw.RequireIPWhitelist(settingRepo)
		r.Use(ipWhitelist)

		r.Get("/admin", adminHandler.Dashboard)
		r.Get("/admin/users", adminHandler.UsersList)
		r.Get("/admin/users/{id}/edit", adminHandler.UsersEdit)
		r.Post("/admin/users/{id}", adminHandler.UsersUpdate)
		r.Post("/admin/users/{id}/delete", adminHandler.UsersDelete)

		r.Get("/admin/stocks", adminHandler.StocksList)
		r.Get("/admin/stocks/{id}/edit", adminHandler.StocksEdit)
		r.Post("/admin/stocks/{id}", adminHandler.StocksUpdate)

		r.Get("/admin/news", adminHandler.NewsList)
		r.Get("/admin/news/create", adminHandler.NewsCreate)
		r.Get("/admin/news/{id}/edit", adminHandler.NewsEdit)
		r.Post("/admin/news", adminHandler.NewsSave)
		r.Post("/admin/news/{id}/delete", adminHandler.NewsDelete)

		r.Get("/admin/blog", adminHandler.BlogPostsList)
		r.Get("/admin/blog/create", adminHandler.BlogPostCreate)
		r.Get("/admin/blog/{id}/edit", adminHandler.BlogPostEdit)
		r.Post("/admin/blog", adminHandler.BlogPostSave)
		r.Post("/admin/blog/{id}/delete", adminHandler.BlogPostDelete)

		r.Get("/admin/sectors", adminHandler.SectorList)
		r.Post("/admin/sectors", adminHandler.SectorSave)

		r.Get("/admin/settings", adminHandler.SettingsPage)
		r.Post("/admin/settings", adminHandler.SettingsSave)
		r.Post("/admin/settings/api-key", adminHandler.GenerateAPIKey)

		r.Get("/admin/pipeline", adminHandler.PipelinePage)
		r.Post("/admin/pipeline/fetch", adminHandler.TriggerFetch)
		r.Post("/admin/pipeline/backfill", adminHandler.TriggerBackfill)
		r.Post("/admin/backup", adminHandler.Backup)

		r.Get("/admin/analytics", adminHandler.AnalyticsPage)
		r.Get("/admin/feature-flags", adminHandler.FeatureFlagsPage)

		r.Post("/admin/backup", adminHandler.Backup)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(apiRateLimiter.Limit)
		r.Use(authMiddleware.RequireAPIAccess)

		r.Get("/ai/generate-report/{portfolioID}", aiHandler.GenerateReportJSON)
		r.Post("/ai/alert-message", aiHandler.AlertMessageJSON)
		r.Post("/ai/social-post", aiHandler.SocialPostJSON)
		r.Get("/ai/search", aiHandler.SemanticSearchJSON)
		r.Get("/ai/impute/{code}", aiHandler.ImputeFundamentalsJSON)
		r.Get("/ai/explain-anomaly/{code}", aiHandler.AnomalyExplanationJSON)

		r.Get("/alert-templates", alertHandler.AlertTemplates)
		r.Get("/predictions/leaderboard", engagementHandler.PredictionLeaderboardJSON)
		r.Get("/portfolios/{id}/weather", engagementHandler.PortfolioWeatherJSON)
		r.Get("/market/overview", apiHandler.MarketOverview)
		r.Get("/market/gainers", apiHandler.TopGainers)
		r.Get("/market/losers", apiHandler.TopLosers)
		r.Get("/market/heatmap", marketHandler.HeatmapJSON)
		r.Get("/market/breadth", marketHandler.BreadthJSON)
		r.Get("/market/summary", marketHandler.MarketSummary)
		r.Get("/market/economic-calendar", marketHandler.EconomicCalendarJSON)
		r.Get("/market/correlation", marketHandler.CorrelationJSON)
		r.Get("/market/forex-heatmap", marketHandler.ForexHeatmapJSON)
		r.Get("/market/liquidity/{code}", marketHandler.StockDetailLiquidityJSON)
		r.Get("/market/signals", marketHandler.SignalsJSON)
		r.Get("/market/signals/{code}", marketHandler.SignalByStockJSON)
		r.Get("/market/trends", marketHandler.TrendsJSON)
		r.Get("/market/price-action", marketHandler.PriceActionJSON)
		r.Get("/market/sector-health", marketHandler.SectorHealthJSON)
		r.Get("/market/waran", marketHandler.WaranJSON)
		r.Get("/market/fear-greed", advancedHandler.FearGreedJSON)
		r.Get("/market/sector-rotation", advancedHandler.SectorRotationJSON)
		r.Get("/market/pair-trading", advancedHandler.PairTradingJSON)
		r.Get("/market/trade-ideas", advancedHandler.TradeIdeasJSON)
		r.Get("/market/macro-dashboard", macroHandler.MacroDashboardJSON)
		r.Get("/market/yield-curve", macroHandler.YieldCurveJSON)
		r.Get("/market/liquidity-flow", macroHandler.LiquidityFlowJSON)
		r.Get("/market/leading-indicators", macroHandler.LeadingIndicatorsJSON)
		r.Get("/market/macro-spillover", macroHandler.MacroSpilloverJSON)
		r.Get("/market/ma-targets", macroHandler.MATargetsJSON)
		r.Get("/market/factor-exposure", advancedHandler.FactorExposureJSON)
		r.Get("/market/factor-timing", advancedHandler.FactorTimingJSON)
		r.Get("/market/supply-chain", advancedHandler.SupplyChainJSON)
		r.Get("/market/industry-lifecycle", advancedHandler.IndustryLifecycleJSON)
		r.Get("/market/competitive", advancedHandler.CompetitiveJSON)
		r.Get("/market/gaps", marketHandler.GapsJSON)
		r.Get("/market/rs-ranking", marketHandler.RSRankingJSON)
		r.Get("/market/new-highs", marketHandler.NewHighsJSON)
		r.Get("/market/new-lows", marketHandler.NewLowsJSON)
		r.Get("/market/pre-market", marketHandler.PreMarketJSON)
		r.Get("/market/block-trades", marketHandler.BlockTradesJSON)
		r.Get("/market/arbitrage", advancedHandler.ArbOpportunitiesJSON)
		r.Get("/portfolio/{id}/risk-decomp", advancedHandler.RiskDecompJSON)
		r.Get("/stocks/{code}/concentration", advancedHandler.ConcentrationJSON)
		r.Get("/market/clock", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service.NewMarketClock().GetMarketStatus())
		})
		r.Get("/market/google-trends", alternativeHandler.GoogleTrendsJSON)
		r.Get("/market/social-buzz", alternativeHandler.SocialBuzzJSON)
		r.Get("/market/cross-asset", alternativeHandler.CrossAssetJSON)
		r.Post("/thesis/backtest", alternativeHandler.BacktestJSON)
		r.Get("/factor/replication", alternativeHandler.FactorReplicationJSON)
		r.Get("/event-study", alternativeHandler.EventStudyJSON)
		r.Post("/ai/research-paper", alternativeHandler.ResearchPaperJSON)
		r.Get("/ai/research-paper/download", alternativeHandler.ResearchPaperDownload)
		r.Get("/stocks/search", apiHandler.StockSearch)
		r.Get("/stocks/{id}/chart", apiHandler.StockChart)
		r.Get("/stocks/{id}/price", apiHandler.StockLatestPrice)
		r.Get("/stocks/{code}/similar", apiHandler.SimilarStocks)
		r.Get("/stocks/{code}/predict", stockHandler.MLPredictJSON)
		r.Get("/stocks/{code}/seasonality", alternativeHandler.SeasonalityJSON)
		r.Get("/sectors", apiHandler.SectorsData)
		r.Get("/news", apiHandler.NewsFeed)
		r.Get("/forex/rates", apiHandler.ForexRates)
		r.Get("/screener/results", screenerHandler.Results)
		r.Get("/forex/{base}-{quote}", forexHandler.ChartData)
		r.Get("/stocks/{code}/chart", stockHandler.ChartData)
		r.Get("/stocks/{code}/drip", portfolioHandler.DRIPJSON)
		r.Get("/stocks/{code}/projection", portfolioHandler.ProjectionJSON)
		r.Get("/portfolios/{id}/performance", portfolioHandler.PerformanceJSON)
		r.Get("/portfolios/{id}/frontier", portfolioHandler.FrontierJSON)
		r.Get("/portfolios/{id}/attribution", portfolioHandler.AttributionJSON)
		r.Post("/portfolios/{id}/monte-carlo", portfolioHandler.MonteCarloJSON)
		r.Get("/portfolios/{id}/stress-test", portfolioHandler.StressTestJSON)
		r.Get("/portfolios/{id}/tax-loss", portfolioHandler.TaxLossJSON)
		r.Get("/portfolios/{id}/risk", portfolioHandler.RiskJSON)
		r.Get("/forex/strength", forexHandler.StrengthJSON)
		r.Get("/forex/carry/{base}/{quote}", forexHandler.CarryTradeJSON)
		r.Get("/forex/correlation/{p1}/{p2}", forexHandler.CorrelationJSON)
		r.Get("/forex/arbitrage", forexHandler.ArbitrageJSON)
		r.Post("/forex/swap", forexToolsHandler.SwapCalcJSON)
		r.Post("/forex/margin", forexToolsHandler.MarginCalcJSON)
		r.Get("/forex/pip-value", forexToolsHandler.PipValueJSON)
		r.Get("/forex/cot/{pair}", forexToolsHandler.COTJSON)
		r.Get("/risk/wash-sale/{id}", riskToolsHandler.WashSaleJSON)
		r.Get("/risk/beta/{id}", riskToolsHandler.BetaJSON)
		r.Get("/risk/hedge/{id}", riskToolsHandler.HedgeJSON)
		r.Get("/stocks/{code}/patterns", analysisHandler.PatternsJSON)
		r.Get("/stocks/{code}/sr-levels", analysisHandler.SRLevelsJSON)
		r.Get("/stocks/{code}/volume-profile", analysisHandler.VolumeProfileJSON)
		r.Get("/stocks/{code}/indicators", analysisHandler.IndicatorsJSON)
		r.Get("/stocks/{code}/fibonacci", analysisHandler.FibonacciJSON)
		r.Get("/stocks/{code}/pattern-reliability", analysisHandler.PatternReliabilityJSON)
		r.Get("/stocks/{code}/pattern-confidence", analysisHandler.PatternConfidenceJSON)
		r.Get("/stocks/{code}/annotations", analysisHandler.AnnotationsJSON)
		r.Get("/stocks/{code}/multi-tf", analysisHandler.MultiTimeframeJSON)
		r.Get("/stocks/{code}/custom-indicator", analysisHandler.CustomIndicatorJSON)
		r.Post("/stocks/{code}/custom-indicator/save", analysisHandler.CustomIndicatorSaveJSON)
		r.Get("/stocks/{code}/custom-indicator/saved", analysisHandler.CustomIndicatorSavedJSON)
		r.Post("/stocks/{code}/custom-indicator/delete", analysisHandler.CustomIndicatorDeleteJSON)
		r.Get("/stocks/{code}/ichimoku", analysisHandler.IchimokuJSON)
		r.Get("/stocks/{code}/sar", analysisHandler.SARJSON)
		r.Get("/stocks/{code}/adx", analysisHandler.ADXJSON)
		r.Get("/stocks/{code}/vwap", analysisHandler.VWAPJSON)
		r.Get("/stocks/{code}/obv", analysisHandler.OBVJSON)
		r.Get("/stocks/{code}/cmf", analysisHandler.CMFJSON)
		r.Get("/stocks/{code}/heikin-ashi", analysisHandler.HeikinAshiJSON)
		r.Get("/stocks/{code}/keltner", analysisHandler.KeltnerJSON)
		r.Get("/alerts", alertHandler.AlertsJSON)
		r.Get("/market/pattern-scan", analysisHandler.MarketPatternScan)
		r.Get("/stocks/{code}/peers", fundamentalHandler.PeerComparisonJSON)
		r.Get("/stocks/{code}/valuation", fundamentalHandler.ValuationJSON)
		r.Get("/stocks/{code}/dividends", fundamentalHandler.StockDividendJSON)
		r.Get("/stocks/{code}/dupont", fundamentalHandler.DuPontJSON)
		r.Get("/stocks/{code}/piotroski", fundamentalHandler.PiotroskiJSON)
		r.Get("/stocks/{code}/altman", fundamentalHandler.AltmanJSON)
		r.Get("/stocks/{code}/beneish", fundamentalHandler.BeneishJSON)
		r.Get("/stocks/{code}/health-score", fundamentalHandler.HealthScoreJSON)
		r.Get("/stocks/{code}/earnings-quality", macroHandler.EarningsQualityJSON)
		r.Get("/stocks/{code}/management-quality", macroHandler.ManagementQualityJSON)
		r.Get("/stocks/{code}/moat", macroHandler.MoatJSON)
		r.Get("/dividends", fundamentalHandler.DividendJSON)
		r.Get("/screener/quick", screenerHandler.QuickScreen)
		r.Get("/screener/nl", screenerHandler.NaturalLanguageScreen)
		r.Post("/screeners", screenerHandler.SaveScreener)
		r.Get("/market/recap", intelligenceHandler.DailyRecapJSON)
		r.Get("/market/anomalies", intelligenceHandler.AnomaliesJSON)
		r.Get("/market/sentiment", intelligenceHandler.MarketSentimentJSON)
		r.Post("/strategies/backtest", strategyHandler.BacktestJSON)

		r.Get("/admin/data-freshness", adminHandler.DataFreshnessJSON)
		r.Get("/admin/data-integrity", adminHandler.DataIntegrityJSON)

		r.Get("/admin/analytics", adminHandler.AnalyticsJSON)
		r.Get("/admin/analytics/users", adminHandler.AnalyticsUsersJSON)
		r.Get("/admin/analytics/features", adminHandler.AnalyticsFeatureUsageJSON)
		r.Get("/admin/feature-flags", adminHandler.FeatureFlagsListJSON)
		r.Post("/admin/feature-flags", adminHandler.FeatureFlagsSaveJSON)

		r.Get("/export/stocks/csv", exportHandler.ExportStocksCSV)
		r.Get("/export/portfolios/{id}/csv", exportHandler.ExportPortfolioCSV)
		r.Get("/export/screener/csv", exportHandler.ExportScreenerCSV)

		r.Post("/webhooks/alert", exportHandler.WebhookAlertReceiver)
		r.Post("/webhooks/test", exportHandler.WebhookTest)

		r.Get("/ai/compare/{codeA}/{codeB}", aiHandler.CompareStocksJSON)
		r.Get("/ai/sector-thesis/{sectorID}", aiHandler.SectorThesisJSON)
		r.Post("/ai/event-impact", aiHandler.EventImpactJSON)
		r.Get("/ai/fraud-check/{code}", aiHandler.FraudDetectionJSON)
		r.Get("/ai/dividend-check/{code}", aiHandler.DividendSustainabilityJSON)
		r.Post("/ai/coach", aiHandler.CoachingChatJSON)
		r.Post("/ai/scenario", aiHandler.ScenarioSimulationJSON)
		r.Post("/ai/thesis", aiHandler.ThesisBuilderJSON)
		r.Get("/ai/jargon", aiHandler.JargonTranslatorJSON)

		r.Post("/ai/agents/analyze", aiHandler.RunAgentAnalysis)
		r.Get("/ai/agents/result/{code}", aiHandler.AgentAnalysisJSON)
		r.Get("/ai/agents/history/{code}", aiHandler.AgentHistoryJSON)

		r.Post("/ai/voice-parse", aiHandler.VoiceParseJSON)
		r.Get("/ai/standup", aiHandler.StandupJSON)
		r.Get("/ai/tax-optimizer/{portfolioID}", aiHandler.TaxOptimizerJSON)
		r.Get("/ai/comic", aiHandler.ComicJSON)
		r.Get("/ai/haiku/{portfolioID}", aiHandler.HaikuJSON)
		r.Get("/ai/stock-story/{code}", aiHandler.StockStoryJSON)
		r.Get("/ai/compliance/{portfolioID}", aiHandler.ComplianceJSON)
		r.Get("/ai/client-report/{portfolioID}", aiHandler.ClientReportJSON)
		r.Post("/ai/webhooks/config", aiHandler.WebhookConfigJSON)
		r.Get("/ai/calendar-sync", aiHandler.CalendarSyncJSON)
		r.Post("/ai/bot-format", aiHandler.BotFormatJSON)
		r.Get("/ai/agents/stream/{code}", aiHandler.StreamAgentAnalysis)
		r.Get("/ai/agents/consensus/{code}", aiHandler.AgentConsensusJSON)
		r.Post("/ai/agents/analyze-persona", aiHandler.AgentAnalysisWithPersona)
		r.Get("/ai/signals/confidence/{code}", aiHandler.SignalConfidenceJSON)
		r.Get("/ai/signals/backtest/{code}", aiHandler.SignalBacktestJSON)
		r.Get("/ai/signals/entry/{code}", aiHandler.OptimalEntryJSON)
		r.Get("/ai/market/regime", aiHandler.MarketRegimeJSON)
		r.Get("/ai/forecast/{code}", aiHandler.PriceForecastJSON)
		r.Get("/ai/dividend-predict/{code}", aiHandler.DividendPredictJSON)
		r.Get("/ai/black-swan", aiHandler.BlackSwanScanJSON)
		r.Get("/ai/insider-interpret/{code}", aiHandler.InsiderInterpretJSON)

		r.Post("/ai/signals/bei/{code}", aiHandler.GenerateBEISignal)
		r.Post("/ai/signals/forex/{pair}", aiHandler.GenerateForexSignal)
		r.Post("/ai/signals/distribute", aiHandler.DistributeSignal)
		r.Post("/settings/data-sources", aiHandler.SaveDataSource)
		r.Post("/settings/data-sources/test", aiHandler.TestDataSource)
		r.Get("/ai/compliance-check/{marketType}", aiHandler.ComplianceCheckJSON)
		r.Get("/mcp/validate/{code}", aiHandler.ValidateMCP)

		r.Get("/ai/usage", aiHandler.AIUsageJSON)
		r.Get("/ai/health-check", aiHandler.AIHealthCheckJSON)
		r.Post("/ai/budget", aiHandler.AIBudgetJSON)
		r.Post("/ai/rate-response", aiHandler.AIRateResponseJSON)
		r.Get("/ai/report", aiHandler.AIReportJSON)
		r.Post("/ai/settings", aiHandler.SaveAISettingsEnhanced)
		r.Post("/ai/test", aiHandler.TestAIProvider)
		r.Get("/ai/providers", aiHandler.AIProvidersJSON)
		r.Post("/ai/advanced", aiHandler.SaveAdvancedSettings)
		r.Get("/ai/mandates", aiHandler.MandatesJSON)
		r.Post("/ai/mandates/save", aiHandler.SaveMandate)
		r.Post("/ai/mandates/delete", aiHandler.DeleteMandate)
		r.Post("/ai/mandates/run", aiHandler.RunMandate)
		r.Get("/ai/approvals", aiHandler.ApprovalsJSON)
		r.Post("/ai/approvals/review", aiHandler.ReviewApproval)
		r.Post("/ai/risk-test", aiHandler.RiskTestJSON)
		r.Post("/portfolios", portfolioHandler.CreateJSON)
		r.Post("/watchlists", watchlistHandler.CreateJSON)
		r.Delete("/watchlists/{id}/stocks/{code}", watchlistHandler.RemoveStockJSON)

		r.Post("/payment/create", paymentHandler.CreateTransaction)
		r.Post("/payment/callback", paymentHandler.Callback)
		r.Get("/invoice/{txID}", paymentHandler.Invoice)

		r.Get("/subscription/status", paymentHandler.SubscriptionStatus)
		r.Post("/subscription/upgrade", paymentHandler.UpgradeSubscription)

		r.Post("/trading/basket", tradingHandler.CreateBasket)
		r.Post("/trading/basket/execute", tradingHandler.ExecuteBasket)
		r.Get("/trading/basket/list", tradingHandler.ListBaskets)
		r.Post("/trading/oco", tradingHandler.CreateOCO)
		r.Get("/trading/oco/active", tradingHandler.ActiveOCO)
		r.Get("/trading/oco/history", tradingHandler.OCOHistory)
		r.Post("/kelly/calculate", tradingHandler.KellyCalculate)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", apiV1Handler.Login)

		r.Group(func(r chi.Router) {
			r.Use(mw.JWTAuth(jwtService))
			r.Use(apiV1Handler.AuthMiddleware)

			r.Get("/stocks", apiV1Handler.ListStocks)
			r.Get("/stocks/{code}", apiV1Handler.StockDetail)
			r.Get("/stocks/{code}/prices", apiV1Handler.StockPrices)
			r.Get("/stocks/{code}/fundamentals", apiV1Handler.StockFundamentals)
			r.Get("/stocks/{code}/indicators", apiV1Handler.StockIndicators)
			r.Get("/stocks/{code}/patterns", apiV1Handler.StockPatterns)
			r.Get("/forex/pairs", apiV1Handler.ForexPairs)
			r.Get("/forex/pairs/{base}/{quote}", apiV1Handler.ForexPairDetail)
			r.Get("/market/summary", apiV1Handler.MarketSummary)
			r.Get("/market/heatmap", apiV1Handler.MarketHeatmap)
			r.Get("/market/recap", apiV1Handler.MarketRecap)
			r.Get("/sectors", apiV1Handler.Sectors)
			r.Get("/news", apiV1Handler.News)
			r.Get("/search", apiV1Handler.Search)
		})
	})

	r.Get("/saham/{code}/report", exportHandler.StockReport)
	r.Get("/dashboard/portfolios/{id}/report", exportHandler.PortfolioReport)

	r.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(web.Static, "static/api/docs.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	staticFS, _ := fs.Sub(web.Static, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.NotFound(pageHandler.NotFound)

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdown
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
	}()

	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("  ║  %-50s  ║\n", cfg.AppName)
	fmt.Printf("  ║  Stock & Forex Market Intelligence Platform          ║\n")
	fmt.Printf("  ╠══════════════════════════════════════════════════════╣\n")
	fmt.Printf("  ║  URL:  %-44s  ║\n", fmt.Sprintf("http://localhost:%s", cfg.Port))
	fmt.Printf("  ║  Port: %-44s  ║\n", cfg.Port)
	fmt.Printf("  ║  Env:  %-44s  ║\n", cfg.AppEnv)
	fmt.Printf("  ╚══════════════════════════════════════════════════════╝\n")
	fmt.Println()

	go startScheduler(db, stockPriceRepo, forexRepo, wsHub, indexNowSvc, alertChecker)

	log.Printf("Starting HTTP server on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

func connectDB(cfg *config.Config) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		db, err = database.Connect(cfg)
		if err == nil {
			log.Println("Database connected")
			return db, nil
		}
		if attempt < 3 {
			log.Printf("DB connection attempt %d/3 failed: %v, retrying in 2s...", attempt, err)
			time.Sleep(2 * time.Second)
		}
	}
	return nil, fmt.Errorf("failed after 3 attempts: %w", err)
}

func humanizeNumber(n int64) string {
	if n < 0 {
		return "-" + humanizeNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	}
	return 0
}

func parseTemplates(appURL string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatNumber": func(v interface{}) string {
			switch val := v.(type) {
			case float64:
				return humanizeNumber(int64(val))
			case int64:
				return humanizeNumber(val)
			case int:
				return humanizeNumber(int64(val))
			}
			return fmt.Sprintf("%v", v)
		},
		"formatPrice": func(v interface{}) string {
			switch val := v.(type) {
			case float64:
				return "Rp " + humanizeNumber(int64(val))
			case int64:
				return "Rp " + humanizeNumber(val)
			case int:
				return "Rp " + humanizeNumber(int64(val))
			}
			return fmt.Sprintf("Rp %v", v)
		},
		"formatVolume": func(v interface{}) string {
			switch val := v.(type) {
			case int64:
				if val >= 1_000_000_000 {
					return fmt.Sprintf("%.1f B", float64(val)/1_000_000_000)
				}
				if val >= 1_000_000 {
					return fmt.Sprintf("%.1f M", float64(val)/1_000_000)
				}
				if val >= 1_000 {
					return fmt.Sprintf("%.0f rb", float64(val)/1_000)
				}
				return humanizeNumber(val)
			case int:
				return fmt.Sprintf("%d", val)
			case float64:
				return fmt.Sprintf("%.0f", val)
			}
			return fmt.Sprintf("%v", v)
		},
		"add": func(a, b interface{}) float64 {
			return toFloat(a) + toFloat(b)
		},
		"sub": func(a, b interface{}) float64 {
			return toFloat(a) - toFloat(b)
		},
		"subtract": func(a, b interface{}) float64 {
			return toFloat(a) - toFloat(b)
		},
		"mul": func(a, b interface{}) float64 {
			return toFloat(a) * toFloat(b)
		},
		"div": func(a, b interface{}) float64 {
			if toFloat(b) == 0 {
				return 0
			}
			return toFloat(a) / toFloat(b)
		},
		"last": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() != reflect.Slice || rv.Len() == 0 {
				return nil
			}
			return rv.Index(rv.Len() - 1).Interface()
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"toJson": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"toUpper":     strings.ToUpper,
		"currentYear": func() int { return time.Now().Year() },
		"appURL":      func() string { return appURL },
	}

	tpl := template.New("").Delims("[[", "]]").Funcs(funcMap)
	err := fs.WalkDir(web.Templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}
		name := filepath.ToSlash(strings.TrimPrefix(path, "templates/"))
		data, err := fs.ReadFile(web.Templates, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		_, err = tpl.New(name).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk templates: %w", err)
	}
	return tpl, nil
}

func autoSeed(db *sqlx.DB) {
	stockRepo := &repository.StockRepository{DB: db}
	priceRepo := &repository.StockPriceRepository{DB: db}
	fundRepo := &repository.StockFundamentalRepository{DB: db}
	idx := &scraper.IDXScraper{}
	yahoo := &scraper.YahooScraper{}

	stocks, _, err := idx.FetchStockList()
	if err != nil {
		log.Printf("[AutoSeed] Failed to fetch stock list: %v", err)
		return
	}

	for i := range stocks {
		stocks[i].ListingDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		stocks[i].SharesOutstanding = 10_000_000_000
		if _, err := stockRepo.Create(&stocks[i]); err != nil {
			log.Printf("[AutoSeed] Failed to create stock %s: %v", stocks[i].Code, err)
		}
	}
	log.Printf("[AutoSeed] %d stocks seeded", len(stocks))

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	end := time.Now()
	start := end.AddDate(0, -1, 0)

	for i := range stocks {
		wg.Add(1)
		go func(s model.Stock) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prices, err := yahoo.FetchHistorical(s.Code+".JK", start, end)
			if err != nil {
				log.Printf("[AutoSeed] Price fetch failed for %s: %v", s.Code, err)
				return
			}
			for j := range prices {
				prices[j].StockID = s.ID
			}
			if err := priceRepo.BulkInsert(prices); err != nil {
				log.Printf("[AutoSeed] Price insert failed for %s: %v", s.Code, err)
			}
		}(stocks[i])
	}
	wg.Wait()
	log.Println("[AutoSeed] Prices fetched (concurrent)")

	sem2 := make(chan struct{}, 5)
	var wg2 sync.WaitGroup
	for i := range stocks {
		wg2.Add(1)
		go func(s model.Stock) {
			defer wg2.Done()
			sem2 <- struct{}{}
			defer func() { <-sem2 }()

			fund, err := idx.FetchFundamentals(s.Code)
			if err != nil {
				return
			}
			fund.StockID = s.ID
			fundRepo.Upsert(fund)
		}(stocks[i])
	}
	wg2.Wait()
	log.Println("[AutoSeed] Fundamentals generated")
	log.Println("[AutoSeed] Complete")
}

func startScheduler(db *sqlx.DB, priceRepo *repository.StockPriceRepository, forexRepo *repository.ForexRepository, wsHub *WSHub, indexNowSvc *seo.IndexNowService, alertChecker *service.AlertChecker) {
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta == nil {
		jakarta = time.FixedZone("WIB", 7*3600)
	}

	runFetch := func(context string) {
		log.Printf("[Scheduler] Starting %s data fetch...", context)
		yahoo := &scraper.YahooScraper{}
		end := time.Now().In(jakarta)
		start := end.AddDate(0, 0, -3)

		latestDate, err := priceRepo.GetLatestDate()
		if err == nil && !latestDate.IsZero() {
			daysSince := int(end.Sub(latestDate).Hours() / 24)
			if daysSince > 3 {
				start = latestDate.AddDate(0, 0, 1)
				log.Printf("[Scheduler] Data gap detected: %d days since %s — backfilling from %s", daysSince, latestDate.Format("2006-01-02"), start.Format("2006-01-02"))
			}
		}

		stockRepo := &repository.StockRepository{DB: db}
		stocks, err := stockRepo.ListActive()
		if err != nil {
			log.Printf("[Scheduler] List stocks error: %v", err)
			return
		}

		var mu sync.Mutex
		var fetched int
		var errors int
		sem := make(chan struct{}, 5)
		var wg sync.WaitGroup

		for _, s := range stocks {
			wg.Add(1)
			go func(stock model.Stock) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				prices, err := yahoo.FetchHistorical(stock.Code+".JK", start, end)
				if err != nil {
					mu.Lock()
					errors++
					mu.Unlock()
					return
				}
				if len(prices) > 0 {
					for j := range prices {
						prices[j].StockID = stock.ID
					}
					if err := priceRepo.BulkInsert(prices); err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
						return
					}
					mu.Lock()
					fetched++
					mu.Unlock()
				}
			}(s)
		}
		wg.Wait()
		log.Printf("[Scheduler] Stock prices: %d/%d stocks updated (errors: %d)", fetched, len(stocks), errors)

		pairs, _ := forexRepo.FindAllPairs()
		ratesFetched := 0
		for _, pair := range pairs {
			open, high, low, close, err := yahoo.FetchForexRate(pair.BaseCurrency, pair.QuoteCurrency)
			if err != nil {
				continue
			}
			rate := model.ForexRate{PairID: pair.ID, Date: time.Now().In(jakarta), Open: open, High: high, Low: low, Close: close}
			if err := forexRepo.BulkInsertRates([]model.ForexRate{rate}); err == nil {
				ratesFetched++
			}
			time.Sleep(100 * time.Millisecond)
		}
		log.Printf("[Scheduler] Forex rates: %d/%d pairs updated", ratesFetched, len(pairs))
		wsHub.Broadcast([]byte(`{"type":"market_update","time":"` + time.Now().In(jakarta).Format("15:04:05") + `","stocks":` + fmt.Sprintf("%d", fetched) + `,"rates":` + fmt.Sprintf("%d", ratesFetched) + `}`))
	}

	fetchNews := func() {
		log.Printf("[Scheduler] Fetching market news...")
		newsScraper := &scraper.NewsScraper{}
		newsRepo := &repository.NewsRepository{DB: db}
		stockRepo := &repository.StockRepository{DB: db}

		newsList, err := newsScraper.FetchMarketNews(20)
		if err != nil {
			log.Printf("[Scheduler] News fetch error: %v", err)
			return
		}

		now := time.Now().In(jakarta)
		for i := range newsList {
			newsList[i].PublishedAt = newsList[i].PublishedAt.UTC()
			newsList[i].CreatedAt = now.UTC()
		}

		if err := newsRepo.BulkInsert(newsList); err != nil {
			log.Printf("[Scheduler] News insert error: %v", err)
			return
		}

		stocks, err := stockRepo.ListActive()
		if err != nil {
			log.Printf("[Scheduler] News stock list error: %v", err)
			return
		}

		var linked int
		for _, article := range newsList {
			saved, err := newsRepo.FindBySlug(article.Slug)
			if err != nil || saved == nil {
				continue
			}
			upperTitle := strings.ToUpper(article.Title)
			upperContent := strings.ToUpper(article.Content)
			for _, s := range stocks {
				code := strings.ToUpper(s.Code)
				if strings.Contains(upperTitle, code) || strings.Contains(upperContent, code) {
					if err := newsRepo.LinkStock(saved.ID, s.ID); err == nil {
						linked++
					}
				}
			}
		}

		log.Printf("[Scheduler] News: %d articles fetched, %d stock links", len(newsList), linked)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Scheduler] PANIC in startup fetch: %v", r)
			}
		}()
		runFetch("startup")
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Scheduler] PANIC in news fetch: %v", r)
			}
		}()
		time.Sleep(10 * time.Second)
		fetchNews()
	}()

	priceTicker := time.NewTicker(5 * time.Minute)
	defer priceTicker.Stop()

	newsTicker := time.NewTicker(30 * time.Minute)
	defer newsTicker.Stop()

	backupTicker := time.NewTicker(24 * time.Hour)
	defer backupTicker.Stop()

	indexNowTicker := time.NewTicker(24 * time.Hour)
	defer indexNowTicker.Stop()

	for {
		select {
		case <-priceTicker.C:
			now := time.Now().In(jakarta)
			hour := now.Hour()
			if hour >= 9 && hour < 16 {
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[Scheduler] PANIC in price fetch: %v", r)
						}
					}()
					runFetch("5min-update")
					if alertChecker != nil {
						runAlertCheck(alertChecker)
					}
				}()
			}
		case <-newsTicker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Scheduler] PANIC in news fetch: %v", r)
					}
				}()
				fetchNews()
			}()
		case <-backupTicker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Scheduler] PANIC in backup: %v", r)
					}
				}()
				runBackup(db)
			}()
		case <-indexNowTicker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Scheduler] PANIC in indexnow: %v", r)
					}
				}()
				runIndexNow(db, indexNowSvc)
			}()
		}
	}
}

// runIndexNow collects recent public URLs and submits them to IndexNow engines.
func runIndexNow(db *sqlx.DB, svc *seo.IndexNowService) {
	if svc == nil {
		return
	}

	blogRepo := &repository.BlogRepository{DB: db}
	stockRepo := &repository.StockRepository{DB: db}

	var urls []string

	posts, _, err := blogRepo.ListPublished(0, 100, 0)
	if err == nil {
		for _, p := range posts {
			urls = append(urls, "/blog/"+p.Slug)
		}
	}

	stocks, _, err := stockRepo.List(0, 200, "", 0)
	if err == nil {
		for _, s := range stocks {
			urls = append(urls, "/saham/"+s.Code)
		}
	}

	submitted, err := svc.Submit(urls)
	if err != nil {
		log.Printf("[IndexNow] submit error: %v", err)
		return
	}
	log.Printf("[IndexNow] submitted %d URLs", submitted)
}

// runAlertCheck evaluates active price alerts and marks triggered ones.
func runAlertCheck(checker *service.AlertChecker) {
	hits, err := checker.CheckAlerts()
	if err != nil {
		log.Printf("[AlertChecker] check error: %v", err)
		return
	}
	if len(hits) == 0 {
		return
	}

	for _, h := range hits {
		a := h.Alert
		a.TriggerCount++
		now := time.Now()
		a.LastTriggeredAt = &now
		if err := checker.AlertRepo.UpdateTrigger(&a); err != nil {
			log.Printf("[AlertChecker] update trigger error: %v", err)
			continue
		}
		log.Printf("[AlertChecker] ALERT triggered: %s %s @ %.2f (count=%d)", h.StockCode, a.Condition, h.CurrentPrice, a.TriggerCount)
	}
}

func runBackup(db *sqlx.DB) {
	cfg := config.Load()

	backupDir := "data/backup"
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		log.Printf("[Scheduler] Backup: mkdir error: %v", err)
		return
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(backupDir, fmt.Sprintf("investo-%s.sql", timestamp))

	args := []string{
		"-h", cfg.DBHost,
		"-P", cfg.DBPort,
		"-u", cfg.DBUser,
		"--single-transaction",
		"--routines",
		"--triggers",
	}
	if cfg.DBPass != "" {
		args = append(args, "--password="+cfg.DBPass)
	}
	args = append(args, cfg.DBName)

	cmd := exec.Command("mysqldump", args...)
	outFile, err := os.Create(filename)
	if err != nil {
		log.Printf("[Scheduler] Backup: create file error: %v", err)
		return
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[Scheduler] Backup: mysqldump error: %v", err)
		return
	}

	pruneOldBackups(backupDir, 14)

	log.Printf("[Scheduler] Backup completed: %s", filename)
}

func newSimExchange(paperRepo *repository.PaperTradingRepository, settingRepo *repository.SettingRepository, approvalRepo *repository.ApprovalRepository) *service.SimulatedExchangeService {
	s := service.NewSimulatedExchangeService(paperRepo, settingRepo)
	s.ApprovalRepo = approvalRepo
	return s
}

func pruneOldBackups(dir string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var files []os.FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err == nil && strings.HasSuffix(info.Name(), ".sql") {
			files = append(files, info)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})

	for i := keep; i < len(files); i++ {
		_ = os.Remove(filepath.Join(dir, files[i].Name()))
	}
}

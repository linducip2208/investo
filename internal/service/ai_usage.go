package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"investo/internal/repository"
)

type ProviderUsage struct {
	TokenCount int64   `json:"token_count"`
	Cost       float64 `json:"cost"`
	AvgLatency float64 `json:"avg_latency"`
	CallsCount int64   `json:"calls_count"`
}

type FeatureUsage struct {
	TokenCount int64   `json:"token_count"`
	Cost       float64 `json:"cost"`
	CallsCount int64   `json:"calls_count"`
}

type DailyUsage struct {
	Date       string  `json:"date"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	Cost       float64 `json:"cost"`
	CallsCount int64   `json:"calls_count"`
}

type UsageStats struct {
	TotalTokens    int64                    `json:"total_tokens"`
	TotalCost      float64                  `json:"total_cost"`
	ByProvider     map[string]ProviderUsage `json:"by_provider"`
	ByFeature      map[string]FeatureUsage  `json:"by_feature"`
	DailyBreakdown []DailyUsage             `json:"daily_breakdown"`
}

type SpendReport struct {
	TotalCost      float64                  `json:"total_cost"`
	ByProvider     map[string]float64       `json:"by_provider"`
	ByFeature      map[string]float64       `json:"by_feature"`
	DailyBreakdown []DailyUsage             `json:"daily_breakdown"`
	Month          int                      `json:"month"`
	Year           int                      `json:"year"`
}

type AIUsageService struct {
	DB          *sqlx.DB
	SettingRepo *repository.SettingRepository
	mu          sync.RWMutex
	failedKeys  map[string]time.Time
}

var providerCostRates = map[string]struct {
	InputPer1M  float64
	OutputPer1M float64
}{
	"deepseek": {0.14, 0.28},
	"openai":   {2.50, 2.50},
	"claude":   {3.00, 3.00},
	"ollama":   {0.00, 0.00},
}

var aiUsageServiceInstance *AIUsageService

func InitAIUsageService(db *sqlx.DB, settingRepo *repository.SettingRepository) *AIUsageService {
	svc := &AIUsageService{
		DB:          db,
		SettingRepo: settingRepo,
		failedKeys:  make(map[string]time.Time),
	}
	aiUsageServiceInstance = svc
	return svc
}

func GetAIUsageService() *AIUsageService {
	return aiUsageServiceInstance
}

func (s *AIUsageService) LogUsage(provider, feature, model string, tokensIn, tokensOut, latencyMs int, userID int64, success bool) error {
	cost := s.calculateCost(provider, tokensIn, tokensOut)
	_, err := s.DB.Exec(`INSERT INTO ai_usage_log (provider, feature, model, tokens_in, tokens_out, latency_ms, cost, user_id, success) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider, feature, model, tokensIn, tokensOut, latencyMs, cost, userID, success)
	return err
}

func (s *AIUsageService) calculateCost(provider string, tokensIn, tokensOut int) float64 {
	rates, ok := providerCostRates[provider]
	if !ok {
		rates = providerCostRates["openai"]
	}
	inputCost := (float64(tokensIn) / 1000000.0) * rates.InputPer1M
	outputCost := (float64(tokensOut) / 1000000.0) * rates.OutputPer1M
	return inputCost + outputCost
}

func (s *AIUsageService) GetUsageStats(days int) (*UsageStats, error) {
	if days <= 0 {
		days = 30
	}

	stats := &UsageStats{
		ByProvider: make(map[string]ProviderUsage),
		ByFeature:  make(map[string]FeatureUsage),
	}

	var totalTokens, totalCost float64
	s.DB.Get(&totalTokens, `SELECT COALESCE(SUM(tokens_in + tokens_out), 0) FROM ai_usage_log WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)`, days)
	s.DB.Get(&totalCost, `SELECT COALESCE(SUM(cost), 0) FROM ai_usage_log WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)`, days)
	stats.TotalTokens = int64(totalTokens)
	stats.TotalCost = totalCost

	rows, err := s.DB.Query(`SELECT provider, COALESCE(SUM(tokens_in + tokens_out), 0) as tokens, COALESCE(SUM(cost), 0) as cost, COALESCE(AVG(latency_ms), 0) as avg_latency, COUNT(*) as calls FROM ai_usage_log WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? DAY) GROUP BY provider`, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var provider string
			var pu ProviderUsage
			if err := rows.Scan(&provider, &pu.TokenCount, &pu.Cost, &pu.AvgLatency, &pu.CallsCount); err == nil {
				stats.ByProvider[provider] = pu
			}
		}
	}

	rows2, err := s.DB.Query(`SELECT feature, COALESCE(SUM(tokens_in + tokens_out), 0) as tokens, COALESCE(SUM(cost), 0) as cost, COUNT(*) as calls FROM ai_usage_log WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? DAY) GROUP BY feature`, days)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var feature string
			var fu FeatureUsage
			if err := rows2.Scan(&feature, &fu.TokenCount, &fu.Cost, &fu.CallsCount); err == nil {
				stats.ByFeature[feature] = fu
			}
		}
	}

	dailyRows, err := s.DB.Query(`SELECT DATE(created_at) as dt, COALESCE(SUM(tokens_in), 0) as ti, COALESCE(SUM(tokens_out), 0) as `+"`to`"+`, COALESCE(SUM(cost), 0) as cost, COUNT(*) as calls FROM ai_usage_log WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? DAY) GROUP BY DATE(created_at) ORDER BY dt`, days)
	if err == nil {
		defer dailyRows.Close()
		for dailyRows.Next() {
			var du DailyUsage
			if err := dailyRows.Scan(&du.Date, &du.TokensIn, &du.TokensOut, &du.Cost, &du.CallsCount); err == nil {
				stats.DailyBreakdown = append(stats.DailyBreakdown, du)
			}
		}
	}

	return stats, nil
}

func (s *AIUsageService) CheckBudget(provider string) (bool, error) {
	key := "ai_budget_" + provider
	limitStr, _ := s.SettingRepo.Get(key)
	if limitStr == "" {
		return true, nil
	}

	var limit float64
	fmt.Sscanf(limitStr, "%f", &limit)
	if limit <= 0 {
		return true, nil
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var spent float64
	if err := s.DB.Get(&spent, `SELECT COALESCE(SUM(cost), 0) FROM ai_usage_log WHERE provider = ? AND created_at >= ?`, provider, monthStart); err != nil {
		return true, err
	}

	return spent < limit, nil
}

func (s *AIUsageService) SetBudget(provider string, monthlyLimit float64) error {
	key := "ai_budget_" + provider
	return s.SettingRepo.Set(key, fmt.Sprintf("%.2f", monthlyLimit))
}

func (s *AIUsageService) CheckKeyHealth(provider string) (bool, error) {
	aiSvc := NewAIService()
	aiSvc.SettingRepo = s.SettingRepo
	configs := aiSvc.LoadProviderConfigs()

	cfg, ok := configs[provider]
	if !ok || cfg.APIKey == "" {
		return false, fmt.Errorf("provider %s not configured", provider)
	}

	start := time.Now()
	_, err := aiSvc.callProvider(provider, cfg.BaseURL, cfg.APIKey, cfg.Model, "You are a health check bot.", "ping")
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		_ = s.LogUsage(provider, "health_check", cfg.Model, 2, 4, int(elapsed), 0, false)
		return false, err
	}

	_ = s.LogUsage(provider, "health_check", cfg.Model, 2, 4, int(elapsed), 0, true)
	return true, nil
}

var keyPoolMutex sync.Mutex
var keyPoolIndex = make(map[string]int)
var keyFailedUntil = make(map[string]time.Time)

func (s *AIUsageService) GetNextKey(provider string) string {
	keyPoolMutex.Lock()
	defer keyPoolMutex.Unlock()

	now := time.Now()
	for i := 1; i <= 5; i++ {
		keyName := fmt.Sprintf("ai_key_%s_%d", provider, i)
		encKey, _ := s.SettingRepo.Get(keyName)
		if encKey == "" {
			continue
		}

		key := s.DecryptKey(encKey)
		if key == "" {
			continue
		}

		failKey := fmt.Sprintf("%s_%d", provider, i)
		if until, ok := keyFailedUntil[failKey]; ok && now.Before(until) {
			continue
		}

		keyPoolIndex[provider] = (keyPoolIndex[provider] % 5) + 1
		return key
	}

	mainKeyName := fmt.Sprintf("ai_provider_%s_key", provider)
	encKey, _ := s.SettingRepo.Get(mainKeyName)
	return s.DecryptKey(encKey)
}

func (s *AIUsageService) MarkKeyFailed(provider, keyName string) {
	keyPoolMutex.Lock()
	defer keyPoolMutex.Unlock()
	keyFailedUntil[keyName] = time.Now().Add(5 * time.Minute)
}

type SmartRouter struct {
	UsageSvc  *AIUsageService
	SettingRepo *repository.SettingRepository
}

func (sr *SmartRouter) RouteRequest(feature string, prompt string) (*AIChatResponse, error) {
	provider := sr.selectProvider(feature)

	aiSvc := NewAIService()
	aiSvc.SettingRepo = sr.SettingRepo
	configs := aiSvc.LoadProviderConfigs()

	cfg, ok := configs[provider]
	if !ok || cfg.APIKey == "" {
		for _, fallback := range []string{"deepseek", "openai", "claude"} {
			if fc, ok := configs[fallback]; ok && fc.APIKey != "" {
				provider = fallback
				cfg = fc
				ok = true
				break
			}
		}
	}

	if !ok || cfg.APIKey == "" {
		return nil, fmt.Errorf("no provider available")
	}

	start := time.Now()
	result, err := aiSvc.callProvider(provider, cfg.BaseURL, cfg.APIKey, cfg.Model, "You are Investo AI, a helpful financial analyst.", prompt)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		sr.UsageSvc.LogUsage(provider, feature, cfg.Model, len(prompt)/4, 0, int(elapsed), 0, false)
		for _, fallback := range []string{"openai", "deepseek", "claude"} {
			if fallback == provider {
				continue
			}
			if fc, ok := configs[fallback]; ok && fc.APIKey != "" {
				start2 := time.Now()
				result2, err2 := aiSvc.callProvider(fallback, fc.BaseURL, fc.APIKey, fc.Model, "You are Investo AI, a helpful financial analyst.", prompt)
				elapsed2 := time.Since(start2).Milliseconds()
				if err2 == nil {
					sr.UsageSvc.LogUsage(fallback, feature, fc.Model, len(prompt)/4, len(result2)/4, int(elapsed2), 0, true)
					estTokens := len(result2) / 4
					return &AIChatResponse{Content: result2, Tokens: estTokens, Model: fc.Model}, nil
				}
			}
		}
		return nil, fmt.Errorf("all providers failed: %w", err)
	}

	sr.UsageSvc.LogUsage(provider, feature, cfg.Model, len(prompt)/4, len(result)/4, int(elapsed), 0, true)
	estTokens := len(result) / 4
	return &AIChatResponse{Content: result, Tokens: estTokens, Model: cfg.Model}, nil
}

func (sr *SmartRouter) selectProvider(feature string) string {
	simpleFeatures := []string{"jargon", "search", "simple_analysis", "alert_message", "social_post", "health_check", "data_impute"}
	complexFeatures := []string{"agent_analysis", "research_paper", "thesis", "compliance", "fraud_check", "event_impact", "forecast"}

	for _, f := range simpleFeatures {
		if f == feature {
			return "deepseek"
		}
	}
	for _, f := range complexFeatures {
		if f == feature {
			return "claude"
		}
	}
	return "openai"
}

func (s *AIUsageService) GetAvgLatency(provider string, days int) (float64, error) {
	if days <= 0 {
		days = 30
	}
	var avg float64
	err := s.DB.Get(&avg, `SELECT COALESCE(AVG(latency_ms), 0) FROM ai_usage_log WHERE provider = ? AND created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)`, provider, days)
	return avg, err
}

func (s *AIUsageService) GetFastestProvider() string {
	providers := []string{"deepseek", "openai", "claude", "ollama"}
	var bestProvider string
	var bestLatency float64 = 999999

	for _, p := range providers {
		lat, err := s.GetAvgLatency(p, 7)
		if err == nil && lat > 0 && lat < bestLatency {
			bestLatency = lat
			bestProvider = p
		}
	}

	if bestProvider == "" {
		return "deepseek"
	}
	return bestProvider
}

func (s *AIUsageService) GenerateSpendReport(month, year int) (*SpendReport, error) {
	if month < 1 || month > 12 {
		now := time.Now()
		month = int(now.Month())
	}
	if year <= 0 {
		year = time.Now().Year()
	}

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	report := &SpendReport{
		ByProvider: make(map[string]float64),
		ByFeature:  make(map[string]float64),
		Month:      month,
		Year:       year,
	}

	s.DB.Get(&report.TotalCost, `SELECT COALESCE(SUM(cost), 0) FROM ai_usage_log WHERE created_at >= ? AND created_at < ?`, monthStart, monthEnd)

	rows, err := s.DB.Query(`SELECT provider, COALESCE(SUM(cost), 0) FROM ai_usage_log WHERE created_at >= ? AND created_at < ? GROUP BY provider`, monthStart, monthEnd)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var provider string
			var cost float64
			if rows.Scan(&provider, &cost) == nil {
				report.ByProvider[provider] = cost
			}
		}
	}

	rows2, err := s.DB.Query(`SELECT feature, COALESCE(SUM(cost), 0) FROM ai_usage_log WHERE created_at >= ? AND created_at < ? GROUP BY feature`, monthStart, monthEnd)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var feature string
			var cost float64
			if rows2.Scan(&feature, &cost) == nil {
				report.ByFeature[feature] = cost
			}
		}
	}

	dailyRows, err := s.DB.Query(`SELECT DATE(created_at) as dt, COALESCE(SUM(tokens_in), 0) as ti, COALESCE(SUM(tokens_out), 0) as `+"`to`"+`, COALESCE(SUM(cost), 0) as cost, COUNT(*) as calls FROM ai_usage_log WHERE created_at >= ? AND created_at < ? GROUP BY DATE(created_at) ORDER BY dt`, monthStart, monthEnd)
	if err == nil {
		defer dailyRows.Close()
		for dailyRows.Next() {
			var du DailyUsage
			if dailyRows.Scan(&du.Date, &du.TokensIn, &du.TokensOut, &du.Cost, &du.CallsCount) == nil {
				report.DailyBreakdown = append(report.DailyBreakdown, du)
			}
		}
	}

	return report, nil
}

func (s *AIUsageService) RateResponse(usageLogID, userID int64, rating int, comment string) error {
	_, err := s.DB.Exec(`INSERT INTO ai_response_ratings (usage_log_id, user_id, rating, comment) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE rating = VALUES(rating), comment = VALUES(comment)`,
		usageLogID, userID, rating, comment)
	return err
}

func (s *AIUsageService) CheckDailyLimit(userID int64) (bool, error) {
	limitStr, _ := s.SettingRepo.Get("max_ai_calls_per_day")
	limit := 500
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	today := time.Now().Format("2006-01-02")
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM ai_usage_log WHERE user_id = ? AND DATE(created_at) = ?`, userID, today)
	if err != nil {
		return true, err
	}

	return count < limit, nil
}

func (s *AIUsageService) GetRecentLogs(limit, offset int, userID int64) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}

	var query string
	var args []interface{}

	if userID > 0 {
		query = `SELECT l.id, l.provider, l.feature, l.model, l.tokens_in, l.tokens_out, l.latency_ms, l.cost, l.success, l.created_at, COALESCE(r.rating, 0) as rating FROM ai_usage_log l LEFT JOIN ai_response_ratings r ON l.id = r.usage_log_id WHERE l.user_id = ? ORDER BY l.created_at DESC LIMIT ? OFFSET ?`
		args = []interface{}{userID, limit, offset}
	} else {
		query = `SELECT l.id, l.provider, l.feature, l.model, l.tokens_in, l.tokens_out, l.latency_ms, l.cost, l.success, l.created_at, COALESCE(r.rating, 0) as rating FROM ai_usage_log l LEFT JOIN ai_response_ratings r ON l.id = r.usage_log_id ORDER BY l.created_at DESC LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	}

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var provider, feature, model string
		var tokensIn, tokensOut, latencyMs int
		var cost float64
		var success bool
		var createdAt time.Time
		var rating int

		if err := rows.Scan(&id, &provider, &feature, &model, &tokensIn, &tokensOut, &latencyMs, &cost, &success, &createdAt, &rating); err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":         id,
			"provider":   provider,
			"feature":    feature,
			"model":      model,
			"tokens_in":  tokensIn,
			"tokens_out": tokensOut,
			"latency_ms": latencyMs,
			"cost":       cost,
			"success":    success,
			"created_at": createdAt.Format("2006-01-02 15:04:05"),
			"rating":     rating,
		})
	}

	return results, nil
}

func (s *AIUsageService) EncryptKey(key string) string {
	if key == "" {
		return ""
	}

	secret := s.getEncryptionSecret()
	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return base64.StdEncoding.EncodeToString([]byte(key))
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return base64.StdEncoding.EncodeToString([]byte(key))
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(key))
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(key), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func (s *AIUsageService) DecryptKey(encoded string) string {
	if encoded == "" {
		return ""
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}

	secret := s.getEncryptionSecret()
	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return ""
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return ""
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}

	return string(plaintext)
}

func (s *AIUsageService) getEncryptionSecret() string {
	if s.SettingRepo != nil {
		secret, _ := s.SettingRepo.Get("encryption_secret")
		if secret != "" {
			for len(secret) < 32 {
				secret += secret
			}
			return secret[:32]
		}
	}
	return "investo-ai-encrypt-secret-key-32b!"
}

func (s *AIUsageService) GetKeyHealthAll() map[string]interface{} {
	providers := []string{"deepseek", "openai", "claude", "ollama"}
	result := make(map[string]interface{})

	for _, p := range providers {
		status := "unknown"
		ok, err := s.CheckKeyHealth(p)
		if err == nil && ok {
			status = "healthy"
		} else if err != nil {
			status = fmt.Sprintf("unhealthy: %v", err)
		} else {
			status = "unhealthy"
		}

		lat, _ := s.GetAvgLatency(p, 1)
		budgetOK, _ := s.CheckBudget(p)
		budgetStr, _ := s.SettingRepo.Get("ai_budget_" + p)

		result[p] = map[string]interface{}{
			"status":    status,
			"latency":   lat,
			"budget_ok": budgetOK,
			"budget":    budgetStr,
		}
	}

	return result
}

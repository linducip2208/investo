package service

import (
	"math"
	"strings"
	"unicode"
)

var positiveKeywords = []string{
	"naik", "cuan", "profit", "laba", "melonjak", "rekomendasi beli",
	"target harga naik", "kontrak baru", "ekspansi", "dividen", "buyback",
	"rights issue menguntungkan", "kinerja positif", "pertumbuhan",
	"revenue naik", "akuisisi", "ekspor", "buy", "bullish", "outperform",
	"upgrade", "strong buy", "overweight", "positive", "growth", "gain",
	"rally", "rebound", "breakout", "uptrend", "record high", "beat",
	"exceed", "surge", "soar", "jump", "optimis", "prospek cerah",
	"rekomendasi overweight", "technical buy", "target price upgrade",
}

var negativeKeywords = []string{
	"turun", "rugi", "loss", "anjlok", "terjun", "default", "gagal bayar",
	"hukum", "sanksi", "ojk", "fraud", "delisting", "suspensi", "pailit",
	"restrukturisasi", "phk", "kerugian", "penurunan", "utang", "gagal",
	"sell", "bearish", "underperform", "downgrade", "strong sell",
	"underweight", "negative", "decline", "drop", "crash", "plunge",
	"downtrend", "breakdown", "warning", "miss", "below", "risk",
	"pesimis", "prospek suram", "rekomendasi jual", "technical sell",
	"target price downgrade", "net sell", "foreign outflow",
}

type SentimentService struct{}

type SentimentResult struct {
	Score    float64  `json:"score"`
	Label    string   `json:"label"`
	Keywords []string `json:"keywords"`
}

func (s *SentimentService) Analyze(text string) SentimentResult {
	textLower := strings.ToLower(text)
	words := tokenize(textLower)

	var foundKeywords []string
	positiveCount := 0
	negativeCount := 0

	for _, keyword := range positiveKeywords {
		if strings.Contains(textLower, keyword) {
			positiveCount++
			foundKeywords = append(foundKeywords, "+"+keyword)
		}
	}

	for _, keyword := range negativeKeywords {
		if strings.Contains(textLower, keyword) {
			negativeCount++
			foundKeywords = append(foundKeywords, "-"+keyword)
		}
	}

	total := positiveCount + negativeCount
	if total == 0 {
		return SentimentResult{
			Score:    0,
			Label:    "neutral",
			Keywords: []string{},
		}
	}

	score := float64(positiveCount-negativeCount) / float64(len(words))
	score = math.Max(-1, math.Min(1, score))

	label := "neutral"
	if score > 0.2 {
		label = "positive"
	} else if score < -0.2 {
		label = "negative"
	}

	return SentimentResult{
		Score:    math.Round(score*100) / 100,
		Label:    label,
		Keywords: foundKeywords,
	}
}

func (s *SentimentService) AnalyzeBatch(texts []string) []SentimentResult {
	results := make([]SentimentResult, len(texts))
	for i, text := range texts {
		results[i] = s.Analyze(text)
	}
	return results
}

func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

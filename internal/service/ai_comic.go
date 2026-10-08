package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"investo/internal/repository"
)

type ComicResult struct {
	Title  string      `json:"title"`
	Panels []ComicPanel `json:"panels"`
	Date   string      `json:"date"`
}

type ComicPanel struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Emoji    string `json:"emoji"`
	Text     string `json:"text"`
	Dialogue string `json:"dialogue"`
}

type AIComicService struct {
	AI        *AIService
	StockRepo *repository.StockRepository
}

func (s *AIComicService) GenerateComic() (*ComicResult, error) {
	now := time.Now()
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta != nil {
		now = now.In(jakarta)
	}

	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	if aiAvailable {
		result, err := s.generateComicWithAI(now)
		if err == nil && result != nil && len(result.Panels) >= 4 {
			return result, nil
		}
	}

	return s.generateComicFallback(now), nil
}

func (s *AIComicService) generateComicWithAI(now time.Time) (*ComicResult, error) {
	prompt := fmt.Sprintf(`Buat komik 4-panel tentang pasar saham Indonesia hari ini (%s). 
Format JSON:
{
  "title": "judul komik lucu",
  "panels": [
    {"number": 1, "title": "Situasi", "emoji": "emoji tunggal", "text": "narasi panel", "dialogue": "dialog karakter"},
    {"number": 2, "title": "Masalah", "emoji": "emoji tunggal", "text": "narasi panel", "dialogue": "dialog karakter"},
    {"number": 3, "title": "Plot Twist", "emoji": "emoji tunggal", "text": "narasi panel", "dialogue": "dialog karakter"},
    {"number": 4, "title": "Pelajaran", "emoji": "emoji tunggal", "text": "narasi panel", "dialogue": "dialog karakter"}
  ]
}

Gaya: humor pasar saham Indonesia, dark humor trader, bahasa Indonesia gaul. Bikin ketawa tapi ada benernya.`, now.Format("02 January 2006"))

	resp, err := s.AI.Chat("Kamu adalah komikus pasar saham. Buat komik 4-panel lucu tentang trading. Response HARUS JSON valid.", prompt)
	if err != nil {
		return nil, err
	}

	var result ComicResult
	if err := json.Unmarshal([]byte(cleanJSON(resp)), &result); err != nil {
		return nil, fmt.Errorf("parse AI comic: %w", err)
	}

	result.Date = now.Format("02 January 2006")
	return &result, nil
}

func (s *AIComicService) generateComicFallback(now time.Time) *ComicResult {
	comedies := [][]ComicPanel{
		{
			{1, "Situasi Pasar", "📈", "IHSG baru buka, semua saham langsung ijo. Trader-tader pada senyum-senyum.", "\"Wah hari ini cuan nih!\""},
			{2, "Masalah Datang", "📰", "Tiba-tiba rilis berita: The Fed naikin suku bunga lagi. IHSG langsung merah semua.", "\"Lah kok gini?! Barusan ijo!\""},
			{3, "Plot Twist", "🫠", "Trader mulai panic selling. Tapi ada satu orang yang malah avg down.", "\"Justru ini saatnya borong!\""},
			{4, "Pelajaran", "🧠", "Ternyata si trader itu beli saham yang sama kayak bandar. Besoknya auto ARA.", "\"Bandarmu siapa? Ya bandar saya.\""},
		},
		{
			{1, "Saham Gorengan", "🍳", "Ada saham gorengan naik 35% hari ini. Semua FOMO masuk.", "\"Wih cuy naik terus!\""},
			{2, "FOMO Masuk", "🏃", "Retail pada beli di harga puncak. Volume meledak.", "\"Ini mah ke ARA!\""},
			{3, "Plot Twist", "📉", "Bandar udah distribute semua. Saham langsung ARB.", "\"...\""},
			{4, "Pelajaran", "💀", "RIP portfolio. Yang beli di bawah malah udah take profit duluan.", "\"Jangan FOMO di saham gorengan.\""},
		},
		{
			{1, "Analisa Teknikal", "📊", "Chart udah breakout triangle pattern. Support kuat di 5000.", "\"Technical analysis says BUY!\""},
			{2, "Entri Posisi", "✅", "Beli di 5100. Pasang TP di 5500, SL di 4900. Perfect setup.", "\"Risk reward 1:4, mantap!\""},
			{3, "Market Ngakak", "🤡", "Harga langsung nyentuh SL. Terus balik naik ke 5500. Tanpa elo.", "\"Stop loss hunting...\""},
			{4, "Pelajaran", "🔮", "Market ga peduli sama analisa lo. Tapi next time, coba lagi.", "\"Chart is always right... eventually.\""},
		},
	}

	idx := rand.Intn(len(comedies))
	return &ComicResult{
		Title:  "Komik Pasar Modal",
		Panels: comedies[idx],
		Date:   now.Format("02 January 2006"),
	}
}

func (s *AIComicService) GenerateHaiku(portfolioID int64) (string, error) {
	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	if aiAvailable {
		haiku, err := s.generateHaikuWithAI(portfolioID)
		if err == nil && haiku != "" {
			return haiku, nil
		}
	}

	return s.generateHaikuFallback(), nil
}

func (s *AIComicService) generateHaikuWithAI(portfolioID int64) (string, error) {
	prompt := fmt.Sprintf(`Buat haiku 5-7-5 dalam bahasa Indonesia tentang performa portofolio investasi ID %d.
Haiku harus mengikuti pola 5-7-5 suku kata.
Gaya: puitis tapi realistis, seperti trader yang merenung.`, portfolioID)

	resp, err := s.AI.Chat("Kamu adalah penyair trader. Buat haiku 5-7-5 bahasa Indonesia. Hanya haiku, tanpa penjelasan.", prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp), nil
}

func (s *AIComicService) generateHaikuFallback() string {
	haikus := []string{
		"Porto merah semua\nInvestor kecil menangis\nBesok pasti ijo",
		"IHSG naik turun\nSeperti ombak di laut\nSabar itu kunci",
		"Beli saham murah\nTunggu dividen cair nanti\nHidup tenang saja",
		"Market tak menentu\nBandar sedang bermain\nKita menunggu",
		"Analisa teknikal\nSupport resistance tertembus\nFundamental kuat",
	}
	return haikus[rand.Intn(len(haikus))]
}



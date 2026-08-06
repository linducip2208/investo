package service

import (
	"fmt"
	"strings"

	"investo/internal/repository"
)

type BotFormatResult struct {
	Platform  string `json:"platform"`
	Command   string `json:"command"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RawFormat string `json:"raw_format"`
}

type AIBotFormatService struct {
	AI                   *AIService
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	PortfolioRepo        *repository.PortfolioRepository
}

func (s *AIBotFormatService) FormatBotMessage(platform, command, code string) (*BotFormatResult, error) {
	result := &BotFormatResult{
		Platform: platform,
		Command:  command,
		Code:     strings.ToUpper(code),
	}

	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	switch command {
	case "/analyze":
		result.Message = s.formatAnalysis(code, aiAvailable)
	case "/signal":
		result.Message = s.formatSignals(aiAvailable)
	case "/portfolio":
		result.Message = s.formatPortfolioSummary(aiAvailable)
	default:
		result.Message = "Perintah tidak dikenal. Gunakan: /analyze CODE, /signal, /portfolio"
	}

	if platform == "discord" {
		result.RawFormat = s.toDiscordFormat(result)
	} else {
		result.RawFormat = s.toSlackFormat(result)
	}

	return result, nil
}

func (s *AIBotFormatService) formatAnalysis(code string, aiAvailable bool) string {
	if !aiAvailable {
		return fmt.Sprintf("📊 *Analisis %s*\n\n> Analisa teknikal dan fundamental %s tersedia di dashboard Investo.\n> Gunakan web app untuk analisa lengkap dengan AI.", code, code)
	}

	return fmt.Sprintf("📊 *Analisis %s*\n\n"+
		"> 🔍 Analisa teknikal: support/resistance, pola chart, indikator\n"+
		"> 📈 Analisa fundamental: PER, PBV, ROE, dividend yield\n"+
		"> 🤖 AI commentary: sinyal dan prediksi pergerakan\n\n"+
		"Detail lengkap: http://localhost:8080/saham/%s", code, code)
}

func (s *AIBotFormatService) formatSignals(aiAvailable bool) string {
	return "🚨 *Sinyal Trading Hari Ini*\n\n" +
		"> 📈 BUY: BBCA (+2.4%), TLKM (+1.8%)\n" +
		"> 📉 SELL: UNVR (-3.1%)\n" +
		"> ⏳ WATCH: ASII, BMRI\n\n" +
		"Sinyal update real-time di dashboard Investo."
}

func (s *AIBotFormatService) formatPortfolioSummary(aiAvailable bool) string {
	return "💼 *Portofolio Summary*\n\n" +
		"> 💰 Total Value: Rp 125.000.000\n" +
		"> 📊 Return: +12.4% YTD\n" +
		"> ⭐ Best: BBCA (+28%)\n" +
		"> 📉 Worst: UNVR (-15%)\n\n" +
		"Update real-time: http://localhost:8080/dashboard/portfolios"
}

func (s *AIBotFormatService) toSlackFormat(result *BotFormatResult) string {
	msg := result.Message
	msg = strings.ReplaceAll(msg, "*", "*")
	msg = strings.ReplaceAll(msg, "📊", ":chart_with_upwards_trend:")
	msg = strings.ReplaceAll(msg, "📈", ":chart_with_upwards_trend:")
	msg = strings.ReplaceAll(msg, "📉", ":chart_with_downwards_trend:")
	msg = strings.ReplaceAll(msg, "🚨", ":rotating_light:")
	msg = strings.ReplaceAll(msg, "💼", ":briefcase:")
	msg = strings.ReplaceAll(msg, "💰", ":moneybag:")
	msg = strings.ReplaceAll(msg, "⭐", ":star:")
	msg = strings.ReplaceAll(msg, "🔍", ":mag:")
	msg = strings.ReplaceAll(msg, "🤖", ":robot_face:")
	msg = strings.ReplaceAll(msg, "⏳", ":hourglass_flowing_sand:")
	return msg
}

func (s *AIBotFormatService) toDiscordFormat(result *BotFormatResult) string {
	return "```md\n" + result.Message + "\n```"
}

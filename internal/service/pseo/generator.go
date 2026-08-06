package pseo

import (
	"fmt"
	"investo/internal/repository"
	"strings"
	"time"
)

type PSEORoute struct {
	Path       string
	Title      string
	Description string
	Priority   float64
	ChangeFreq string
}

type PSEOGenerator struct {
	StockRepo  *repository.StockRepository
	SectorRepo *repository.SectorRepository
}

func (g *PSEOGenerator) GenerateStockRoutes() ([]PSEORoute, error) {
	var routes []PSEORoute

	stocks, _, err := g.StockRepo.List(0, 10000, "", 0)
	if err != nil {
		return nil, fmt.Errorf("PSEOGenerator.GenerateStockRoutes: %w", err)
	}

	sectors, err := g.SectorRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("PSEOGenerator.GenerateStockRoutes sectors: %w", err)
	}

	for _, stock := range stocks {
		code := strings.ToLower(strings.TrimSpace(stock.Code))

		routes = append(routes,
			PSEORoute{
				Path:       fmt.Sprintf("/saham/%s", code),
				Title:      fmt.Sprintf("Saham %s - Harga & Analisis Terbaru %d", stock.Code, time.Now().Year()),
				Description: fmt.Sprintf("Informasi lengkap saham %s (%s). Cek harga, grafik, fundamental, dan berita terkini.", stock.Code, stock.Name),
				Priority:   0.9,
				ChangeFreq: "daily",
			},
			PSEORoute{
				Path:       fmt.Sprintf("/saham/%s/harga", code),
				Title:      fmt.Sprintf("Harga Saham %s Hari Ini - Grafik & Historis", stock.Code),
				Description: fmt.Sprintf("Pantau harga saham %s real-time. Lihat grafik pergerakan harga harian, mingguan, dan bulanan.", stock.Code),
				Priority:   0.8,
				ChangeFreq: "hourly",
			},
			PSEORoute{
				Path:       fmt.Sprintf("/saham/%s/fundamental", code),
				Title:      fmt.Sprintf("Fundamental Saham %s - ROE, PER, PBV, DER", stock.Code),
				Description: fmt.Sprintf("Analisa fundamental %s (%s). ROE, ROA, PER, PBV, DER, NPM, dan dividend yield.", stock.Code, stock.Name),
				Priority:   0.8,
				ChangeFreq: "weekly",
			},
			PSEORoute{
				Path:       fmt.Sprintf("/saham/%s/berita", code),
				Title:      fmt.Sprintf("Berita Saham %s - Update & Sentimen Pasar", stock.Code),
				Description: fmt.Sprintf("Kumpulan berita terbaru tentang saham %s. Analisa sentimen dan dampak ke harga.", stock.Code),
				Priority:   0.7,
				ChangeFreq: "daily",
			},
		)
	}

	for _, sector := range sectors {
		slug := strings.ToLower(strings.TrimSpace(sector.Slug))
		if slug == "" {
			slug = strings.ToLower(strings.ReplaceAll(sector.Name, " ", "-"))
		}

		routes = append(routes, PSEORoute{
			Path:       fmt.Sprintf("/best-saham-%s", slug),
			Title:      fmt.Sprintf("10 Saham %s Terbaik %d - Rekomendasi & Analisa", sector.Name, time.Now().Year()),
			Description: fmt.Sprintf("Daftar saham %s terbaik pilihan analis. Lengkap dengan fundamental, teknikal, dan prospek.", sector.Name),
			Priority:   0.7,
			ChangeFreq: "weekly",
		})

		for _, year := range []int{2024, 2025, 2026} {
			routes = append(routes, PSEORoute{
				Path:       fmt.Sprintf("/best-saham-%s-%d", slug, year),
				Title:      fmt.Sprintf("10 Saham %s Terbaik %d - Rekomendasi Tahun Ini", sector.Name, year),
				Description: fmt.Sprintf("Rekomendasi saham %s terbaik tahun %d. Berdasarkan fundamental dan teknikal.", sector.Name, year),
				Priority:   0.6,
				ChangeFreq: "monthly",
			})
		}
	}

	maxPairs := 2500
	pairCount := 0
	stockCount := len(stocks)
	if stockCount > 50 {
		stockCount = 50
	}
	for i := 0; i < stockCount && pairCount < maxPairs; i++ {
		for j := i + 1; j < stockCount && pairCount < maxPairs; j++ {
			codeA := strings.ToLower(strings.TrimSpace(stocks[i].Code))
			codeB := strings.ToLower(strings.TrimSpace(stocks[j].Code))

			routes = append(routes, PSEORoute{
				Path:       fmt.Sprintf("/compare/%s-vs-%s", codeA, codeB),
				Title:      fmt.Sprintf("%s vs %s - Perbandingan Saham Lengkap", stocks[i].Code, stocks[j].Code),
				Description: fmt.Sprintf("Bandingkan saham %s dan %s. Analisa fundamental, teknikal, dan prospek.", stocks[i].Code, stocks[j].Code),
				Priority:   0.5,
				ChangeFreq: "weekly",
			})
			pairCount++
		}
	}

	seoRoutes := []struct {
		path, title, desc string
	}{
		{"/beli-aplikasi-saham", "Beli Aplikasi Saham - Source Code Investo", "Beli source code aplikasi saham dan trading. Full source code Go + MySQL."},
		{"/beli-aplikasi-forex", "Beli Aplikasi Forex - Source Code Trading", "Beli source code aplikasi forex trading. Fitur lengkap chart, indikator, alert."},
		{"/source-code-trading", "Source Code Aplikasi Trading Saham & Forex", "Source code aplikasi trading lengkap. Go, MySQL, real-time chart, indikator teknikal."},
		{"/aplikasi-analisa-saham", "Aplikasi Analisa Saham - Source Code Siap Pakai", "Aplikasi analisa saham dengan fundamental & teknikal. Beli source code sekarang."},
		{"/aplikasi-chart-saham", "Aplikasi Chart Saham Profesional - Source Code", "Aplikasi chart saham dengan indikator lengkap: SMA, EMA, RSI, MACD, Bollinger Bands."},
		{"/aplikasi-investasi", "Aplikasi Investasi - Source Code Go", "Aplikasi investasi saham dan forex. Portfolio tracking, alert harga, berita pasar."},
		{"/aplikasi-pasar-modal", "Aplikasi Pasar Modal Indonesia - Source Code", "Aplikasi pasar modal dengan data IDX. Analisa fundamental ratusan saham BEI."},
		{"/jual-source-code-trading", "Jual Source Code Aplikasi Trading", "Jual source code aplikasi trading saham forex. Whitelabel ready, full documented."},
		{"/whitelabel-aplikasi-saham", "Whitelabel Aplikasi Saham - Source Code Investo", "Whitelabel aplikasi saham dengan brand sendiri. Full source code + dokumentasi."},
		{"/software-analisa-teknikal", "Software Analisa Teknikal - Source Code", "Software analisa teknikal saham. Indikator lengkap, multi-timeframe, alert otomatis."},
		{"/aplikasi-screener-saham", "Aplikasi Screener Saham - Source Code Go", "Aplikasi stock screener dengan filter lengkap. Fundamental dan teknikal."},
		{"/aplikasi-portofolio-saham", "Aplikasi Portofolio Saham - Source Code", "Aplikasi portfolio saham. Track profit/loss, alokasi, dan return investasi."},
		{"/aplikasi-alert-harga", "Aplikasi Alert Harga Saham - Source Code", "Aplikasi alert harga saham otomatis. Notifikasi saat target harga tercapai."},
		{"/aplikasi-bandarmologi", "Aplikasi Bandarmologi - Source Code", "Aplikasi analisa bandarmologi. Lacak akumulasi dan distribusi saham."},
		{"/aplikasi-forex-trading", "Aplikasi Forex Trading - Source Code", "Aplikasi forex trading dengan chart real-time. Multiple pair: USDIDR, EURUSD, GBPJPY."},
		{"/aplikasi-data-bei", "Aplikasi Data BEI - Source Code Scraper", "Aplikasi data BEI dengan scraper otomatis. Data saham, fundamental, aksi korporasi."},
		{"/beli-source-code-investo", "Beli Source Code Investo - Full Package", "Beli source code Investo lengkap. Free installation, 1 year support, lifetime update."},
		{"/aplikasi-manajemen-investasi", "Aplikasi Manajemen Investasi - Source Code", "Aplikasi manajemen investasi untuk kantor sekuritas dan fund manager."},
		{"/aplikasi-robot-trading", "Aplikasi Robot Trading - Source Code Go", "Aplikasi robot trading otomatis. Backtest strategi, auto trade, risk management."},
		{"/aplikasi-komunitas-saham", "Aplikasi Komunitas Saham - Source Code", "Aplikasi komunitas saham dengan fitur diskusi, sinyal, dan leaderboard."},
	}

	for _, r := range seoRoutes {
		routes = append(routes, PSEORoute{
			Path:        r.path,
			Title:       r.title,
			Description: r.desc,
			Priority:    0.5,
			ChangeFreq:  "monthly",
		})
	}

	return routes, nil
}

func (g *PSEOGenerator) GenerateSitemapXML(routes []PSEORoute, baseURL string) string {
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	for _, route := range routes {
		loc := strings.TrimRight(baseURL, "/") + route.Path
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", xmlEscape(loc)))
		sb.WriteString(fmt.Sprintf("    <changefreq>%s</changefreq>\n", route.ChangeFreq))
		sb.WriteString(fmt.Sprintf("    <priority>%.1f</priority>\n", route.Priority))
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>\n")

	return sb.String()
}

func (g *PSEOGenerator) GenerateRobotsTXT() string {
	var sb strings.Builder

	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /$\n")
	sb.WriteString("Allow: /saham/\n")
	sb.WriteString("Allow: /forex/\n")
	sb.WriteString("Allow: /best-\n")
	sb.WriteString("Allow: /compare/\n")
	sb.WriteString("Allow: /blog\n")
	sb.WriteString("Allow: /docs\n")
	sb.WriteString("Allow: /investasi/\n")
	sb.WriteString("Allow: /beli-\n")
	sb.WriteString("Allow: /aplikasi-\n")
	sb.WriteString("Allow: /source-code-\n")
	sb.WriteString("Allow: /jual-\n")
	sb.WriteString("Allow: /whitelabel-\n")
	sb.WriteString("Allow: /software-\n")
	sb.WriteString("Allow: /marketing/\n")
	sb.WriteString("Disallow: /admin\n")
	sb.WriteString("Disallow: /api\n")
	sb.WriteString("Disallow: /__pair\n")
	sb.WriteString("Disallow: /webhooks\n")
	sb.WriteString("Sitemap: /sitemap.xml\n")

	return sb.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

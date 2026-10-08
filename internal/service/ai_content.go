package service

import (
	"fmt"
	"strings"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type AIContentService struct {
	AI             *AIService
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	StockFundRepo  *repository.StockFundamentalRepository
	SectorRepo     *repository.SectorRepository
	NewsRepo       *repository.NewsRepository
	ForexRepo      *repository.ForexRepository
	BreadthSvc     *MarketBreadthService
}

func (s *AIContentService) GenerateDailyBriefing() (string, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return "", fmt.Errorf("list stocks: %w", err)
	}

	var stockIDs []int64
	for _, st := range stocks {
		stockIDs = append(stockIDs, st.ID)
	}

	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	type mover struct {
		Code   string
		Name   string
		Price  float64
		Change float64
	}
	var gainers, losers []mover
	for _, st := range stocks {
		price := priceMap[st.ID]
		change, _ := s.StockPriceRepo.GetPriceChange(st.ID, 1)
		m := mover{Code: st.Code, Name: st.Name, Price: price, Change: change}
		if change > 0 {
			gainers = append(gainers, m)
		} else if change < 0 {
			losers = append(losers, m)
		}
	}

	for i := 0; i < len(gainers); i++ {
		for j := i + 1; j < len(gainers); j++ {
			if gainers[j].Change > gainers[i].Change {
				gainers[i], gainers[j] = gainers[j], gainers[i]
			}
		}
	}
	for i := 0; i < len(losers); i++ {
		for j := i + 1; j < len(losers); j++ {
			if losers[j].Change < losers[i].Change {
				losers[i], losers[j] = losers[j], losers[i]
			}
		}
	}

	topGainers := gainers
	if len(topGainers) > 5 {
		topGainers = topGainers[:5]
	}
	topLosers := losers
	if len(topLosers) > 5 {
		topLosers = topLosers[:5]
	}

	breadth, _ := s.BreadthSvc.Calculate()

	var forexData []string
	pairs, _ := s.ForexRepo.FindAllPairs()
	rates, _ := s.ForexRepo.FindLatestRates()
	rateMap := make(map[int64]float64, len(rates))
	for _, r := range rates {
		rateMap[r.PairID] = r.Close
	}
	for _, p := range pairs {
		if rate, ok := rateMap[p.ID]; ok {
			forexData = append(forexData, fmt.Sprintf("%s/%s: %.4f", p.BaseCurrency, p.QuoteCurrency, rate))
		}
	}

	var gainerLines, loserLines string
	for _, g := range topGainers {
		gainerLines += fmt.Sprintf("- %s (%s): Rp %.0f (%.2f%%)\n", g.Code, g.Name, g.Price, g.Change)
	}
	for _, l := range topLosers {
		loserLines += fmt.Sprintf("- %s (%s): Rp %.0f (%.2f%%)\n", l.Code, l.Name, l.Price, l.Change)
	}

	systemPrompt := "Kamu adalah analis pasar modal profesional untuk pasar saham Indonesia. Berikan analisis yang informatif, objektif, dan mudah dipahami dalam Bahasa Indonesia. Gunakan gaya bahasa profesional namun bersahabat."

	userMessage := fmt.Sprintf(`Tolong buatkan market briefing pagi ini dalam 3 paragraf dalam Bahasa Indonesia berdasarkan data berikut:

📊 MARKET BREADTH:
- Saham Naik: %d
- Saham Turun: %d
- Saham Stagnan: %d

🔥 TOP 5 GAINERS:
%s

📉 TOP 5 LOSERS:
%s

💱 FOREX RATES:
%s

Tolong buat briefing yang mencakup:
1. Paragraf 1: Ringkasan kondisi pasar secara umum hari ini
2. Paragraf 2: Analisis saham-saham yang menonjol (gainers dan losers)
3. Paragraf 3: Outlook dan rekomendasi singkat untuk trader/investor

Format: HTML dengan tag <p> untuk setiap paragraf. Gunakan bahasa Indonesia yang baik.`,
		breadth.Advance, breadth.Decline, breadth.Unchanged,
		gainerLines,
		loserLines,
		strings.Join(forexData, "\n"))

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func (s *AIContentService) GenerateStockReport(code string) (string, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return "", fmt.Errorf("stock not found: %w", err)
	}

	prices, _ := s.StockPriceRepo.FindLatest(stock.ID, 252)
	fundamentals, _ := s.StockFundRepo.FindByStockID(stock.ID, 8)
	sector, _ := s.SectorRepo.FindByID(stock.SectorID)

	var latestPrice float64
	if len(prices) > 0 {
		latestPrice = prices[0].Close
	}

	var priceHistory string
	for i := len(prices) - 1; i >= 0; i-- {
		p := prices[i]
		if i >= len(prices)-30 || i < 5 {
			priceHistory += fmt.Sprintf("%s: Open=%.0f High=%.0f Low=%.0f Close=%.0f Vol=%d\n",
				p.Date.Format("2006-01-02"), p.Open, p.High, p.Low, p.Close, p.Volume)
		}
	}

	var fundData string
	for _, f := range fundamentals {
		fundData += fmt.Sprintf(`Periode %s (%s):
- Revenue: Rp %.0f M | Net Income: Rp %.0f M | EPS: %.2f | BVPS: %.2f
- ROE: %.2f%% | ROA: %.2f%% | PER: %.2fx | PBV: %.2fx
- DER: %.2fx | NPM: %.2f%% | Div Yield: %.2f%%
`,
			f.Period, f.ReportType,
			f.Revenue, f.NetIncome, f.EPS, f.BVPS,
			f.ROE, f.ROA, f.PER, f.PBV,
			f.DER, f.NetProfitMargin, f.DividendYield)
	}

	systemPrompt := "Kamu adalah analis riset saham senior di sebuah firma sekuritas terkemuka di Indonesia. Berikan analisis fundamental dan teknikal yang mendalam, objektif, dan actionable dalam Bahasa Indonesia."

	userMessage := fmt.Sprintf(`Buatkan laporan riset saham untuk %s (%s) dalam Bahasa Indonesia dengan format HTML.

📋 DATA SAHAM:
- Kode: %s
- Nama: %s
- Sektor: %s
- Subsektor: %s
- Deskripsi: %s
- Harga Terakhir: Rp %.0f
- Website: %s

📈 DATA FUNDAMENTAL:
%s

📊 DATA HARGA (30 hari terakhir + sample):
%s

Tolong buat laporan riset dengan struktur berikut (dalam HTML):
1. <h2>Ringkasan Eksekutif</h2> - paragraf pembuka
2. <h2>Analisis SWOT</h2> - tabel dengan strengths, weaknesses, opportunities, threats
3. <h2>Penilaian Valuasi</h2> - analisis PER, PBV, dan valuasi relatif
4. <h2>Outlook & Rekomendasi</h2> - prospek 6-12 bulan ke depan + rekomendasi (Buy/Hold/Sell) dengan target harga

Gunakan Bahasa Indonesia yang profesional. Buat dalam format HTML yang rapi.`,
		stock.Code, stock.Name,
		stock.Code, stock.Name,
		func() string {
			if sector != nil {
				return sector.Name
			}
			return "N/A"
		}(),
		stock.Subsector, stock.Description, latestPrice, stock.Website,
		fundData,
		priceHistory)

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func (s *AIContentService) SummarizeEarnings(code, period string) (string, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return "", fmt.Errorf("stock not found: %w", err)
	}

	fund, err := s.StockFundRepo.FindByStockPeriod(stock.ID, period, "annual")
	if err != nil {
		fund, err = s.StockFundRepo.FindLatest(stock.ID)
		if err != nil {
			return "", fmt.Errorf("no fundamental data: %w", err)
		}
	}

	systemPrompt := "Kamu adalah analis keuangan yang ahli dalam menganalisis laporan keuangan perusahaan Indonesia. Berikan ringkasan yang jelas dan informatif dalam Bahasa Indonesia."

	userMessage := fmt.Sprintf(`Buatkan ringkasan laporan keuangan untuk %s (%s) periode %s dalam Bahasa Indonesia.

📊 DATA:
- Revenue: Rp %.0f M
- Net Income: Rp %.0f M
- EPS: %.2f | BVPS: %.2f
- Total Assets: Rp %.0f M | Total Liabilities: Rp %.0f M | Equity: Rp %.0f M
- ROE: %.2f%% | ROA: %.2f%%
- PER: %.2fx | PBV: %.2fx | DER: %.2fx
- Net Profit Margin: %.2f%% | Dividend Yield: %.2f%%

Tolong buat ringkasan yang mencakup:
1. Kinerja top-line (revenue)
2. Profitabilitas (bottom-line, margin)
3. Kesehatan neraca (leverage, aset)
4. Metrik valuasi
5. Kesimpulan singkat

Gunakan Bahasa Indonesia. Format dalam HTML yang rapi.`,
		stock.Code, stock.Name, period,
		fund.Revenue, fund.NetIncome, fund.EPS, fund.BVPS,
		fund.TotalAssets, fund.TotalLiabilities, fund.Equity,
		fund.ROE, fund.ROA, fund.PER, fund.PBV, fund.DER,
		fund.NetProfitMargin, fund.DividendYield)

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func (s *AIContentService) SummarizeNews(newsIDs []int64) (string, error) {
	var newsArticles []string
	for _, id := range newsIDs {
		n, err := s.NewsRepo.FindByID(id)
		if err != nil {
			continue
		}
		newsArticles = append(newsArticles, fmt.Sprintf(`Judul: %s
Sumber: %s | Tanggal: %s
Konten: %s
---`, n.Title, n.Source, n.PublishedAt.Format("2006-01-02 15:04"), n.Content))
	}

	if len(newsArticles) == 0 {
		return "<p>Tidak ada berita yang bisa dirangkum.</p>", nil
	}

	systemPrompt := "Kamu adalah news aggregator profesional untuk pasar modal Indonesia. Rangkum berita-berita penting dengan objektif, ringkas, dan informatif dalam Bahasa Indonesia."

	userMessage := fmt.Sprintf(`Rangkum berita-berita berikut menjadi satu briefing ringkas dalam Bahasa Indonesia:

%s

Buat dalam format HTML yang rapi dengan struktur:
1. <h3>Ringkasan Berita Pasar</h3>
2. Poin-poin penting dari berita
3. Implikasi untuk pasar

Gunakan Bahasa Indonesia yang baik.`,
		strings.Join(newsArticles, "\n"))

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func (s *AIContentService) AnalyzeIPO(code string) (string, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return "", fmt.Errorf("stock not found: %w", err)
	}

	fund, err := s.StockFundRepo.FindLatest(stock.ID)
	if err != nil {
		fund = &model.StockFundamental{}
	}

	systemPrompt := "Kamu adalah analis IPO (Initial Public Offering) berpengalaman di pasar modal Indonesia. Berikan analisis prospektus IPO yang komprehensif dan objektif dalam Bahasa Indonesia."

	userMessage := fmt.Sprintf(`Buatkan analisis IPO untuk %s (%s) dalam Bahasa Indonesia.

📋 DATA IPO:
- Kode: %s
- Nama: %s
- Sektor: %s
- Deskripsi: %s
- Shares Outstanding: %d
- Website: %s

📊 DATA FUNDAMENTAL (jika tersedia):
- Revenue: Rp %.0f M
- Net Income: Rp %.0f M
- EPS: %.2f | PER: %.2fx

Tolong buat analisis yang mencakup:
1. Profil bisnis dan model bisnis
2. Analisis penggunaan dana IPO
3. Valuasi dan perbandingan dengan peers
4. Risiko dan mitigasi
5. Rekomendasi (Subscribe/Avoid) dengan reasoning

Gunakan Bahasa Indonesia yang profesional. Format dalam HTML.`,
		stock.Code, stock.Name,
		stock.Code, stock.Name,
		func() string {
			s, _ := s.SectorRepo.FindByID(stock.SectorID)
			if s != nil {
				return s.Name
			}
			return "N/A"
		}(),
		stock.Description, stock.SharesOutstanding, stock.Website,
		fund.Revenue, fund.NetIncome, fund.EPS, fund.PER)

	return s.AI.QuickChat(systemPrompt, userMessage)
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func _formatPrice(v interface{}) string {
	switch val := v.(type) {
	case float64:
		return fmt.Sprintf("Rp %.0f", val)
	case int64:
		return fmt.Sprintf("Rp %d", val)
	default:
		return fmt.Sprintf("Rp %v", v)
	}
}

func _timeNow() time.Time {
	return time.Now()
}

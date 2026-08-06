package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"investo/internal/repository"
)

type AIToolsService struct {
	AI                    *AIService
	StockRepo             *repository.StockRepository
	StockPriceRepo        *repository.StockPriceRepository
	StockFundamentalRepo  *repository.StockFundamentalRepository
	PortfolioRepo         *repository.PortfolioRepository
	PortfolioItemRepo     *repository.PortfolioItemRepository
	SectorRepo            *repository.SectorRepository
}

type PortfolioSummaryData struct {
	Name        string
	ItemCount   int
	TotalValue  float64
	TotalReturn float64
	ReturnPct   float64
	Holdings    []HoldingSummary
}

type HoldingSummary struct {
	Code       string
	Name       string
	Quantity   float64
	AvgPrice   float64
	CurrPrice  float64
	Value      float64
	Return     float64
	ReturnPct  float64
	Weight     float64
	PER        float64
	PBV        float64
	ROE        float64
	DivYield   float64
}

type ReportData struct {
	GeneratedAt    time.Time
	Portfolio      PortfolioSummaryData
	BestPerformer  HoldingSummary
	WorstPerformer HoldingSummary
	AvgPER         float64
	AvgPBV         float64
	AvgROE         float64
	AvgDER         float64
	DivYieldAvg    float64
}

type SemanticSearchResult struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	ChangePct   float64 `json:"change_pct"`
	PER         float64 `json:"per"`
	DivYield    float64 `json:"dividend_yield"`
	MatchReason string  `json:"match_reason"`
}

func (s *AIToolsService) GenerateReport(portfolioID int64) (string, error) {
	now := time.Now()

	items, err := s.PortfolioItemRepo.FindByPortfolioID(portfolioID)
	if err != nil {
		return "", fmt.Errorf("load portfolio items: %w", err)
	}

	portfolio, err := s.PortfolioRepo.FindByID(portfolioID)
	if err != nil {
		return "", fmt.Errorf("load portfolio: %w", err)
	}

	if len(items) == 0 {
		return s.emptyReportHTML(portfolio.Name, now), nil
	}

	var holdings []HoldingSummary
	totalValue := 0.0
	totalCost := 0.0
	var stockIDs []int64
	for _, item := range items {
		stockIDs = append(stockIDs, item.StockID)
	}

	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	for _, item := range items {
		price := priceMap[item.StockID]
		value := price * float64(item.Quantity)
		cost := item.AvgPrice * float64(item.Quantity)

		h := HoldingSummary{
			Quantity:  item.Quantity,
			AvgPrice:  item.AvgPrice,
			CurrPrice: price,
			Value:     value,
			Return:    value - cost,
		}
		if cost > 0 {
			h.ReturnPct = ((price - item.AvgPrice) / item.AvgPrice) * 100
		}

		stock, err := s.StockRepo.FindByID(item.StockID)
		if err == nil && stock != nil {
			h.Code = stock.Code
			h.Name = stock.Name
		}

		fund, _ := s.StockFundamentalRepo.FindLatest(item.StockID)
		if fund != nil {
			h.PER = fund.PER
			h.PBV = fund.PBV
			h.ROE = fund.ROE
			h.DivYield = fund.DividendYield
		}

		holdings = append(holdings, h)
		totalValue += value
		totalCost += cost
	}

	totalReturn := totalValue - totalCost
	returnPct := 0.0
	if totalCost > 0 {
		returnPct = (totalReturn / totalCost) * 100
	}

	for i := range holdings {
		if totalValue > 0 {
			holdings[i].Weight = (holdings[i].Value / totalValue) * 100
		}
	}

	bestIdx, worstIdx := 0, 0
	for i := range holdings {
		if holdings[i].ReturnPct > holdings[bestIdx].ReturnPct {
			bestIdx = i
		}
		if holdings[i].ReturnPct < holdings[worstIdx].ReturnPct {
			worstIdx = i
		}
	}

	var sumPER, sumPBV, sumROE, sumDiv float64
	fundCount := 0
	for _, h := range holdings {
		if h.PER > 0 || h.PBV > 0 {
			sumPER += h.PER
			sumPBV += h.PBV
			sumROE += h.ROE
			sumDiv += h.DivYield
			fundCount++
		}
	}
	if fundCount == 0 {
		fundCount = 1
	}

	report := ReportData{
		GeneratedAt: now,
		Portfolio: PortfolioSummaryData{
			Name:        portfolio.Name,
			ItemCount:   len(items),
			TotalValue:  totalValue,
			TotalReturn: totalReturn,
			ReturnPct:   returnPct,
			Holdings:    holdings,
		},
		BestPerformer:  holdings[bestIdx],
		WorstPerformer: holdings[worstIdx],
		AvgPER:         sumPER / float64(fundCount),
		AvgPBV:         sumPBV / float64(fundCount),
		AvgROE:         sumROE / float64(fundCount),
		AvgDER:         0,
		DivYieldAvg:    sumDiv / float64(fundCount),
	}

	aiNarrative := s.generateNarrative(report)

	return s.buildReportHTML(report, aiNarrative), nil
}

func (s *AIToolsService) generateNarrative(r ReportData) string {
	var sb strings.Builder

	perfWord := "positif"
	if r.Portfolio.ReturnPct < 0 {
		perfWord = "negatif"
	} else if r.Portfolio.ReturnPct < 1 {
		perfWord = "flat"
	}

	tag := map[bool]string{true: "untung", false: "rugi"}[r.Portfolio.TotalReturn >= 0]

	sb.WriteString(fmt.Sprintf("Portofolio %s menunjukkan performa %s dengan return %.2f%% (%s Rp %.0f) dari total nilai Rp %.0f. ",
		r.Portfolio.Name, perfWord, r.Portfolio.ReturnPct, tag, math.Abs(r.Portfolio.TotalReturn), r.Portfolio.TotalValue))

	if r.Portfolio.ItemCount <= 3 {
		conc := map[bool]string{true: "cukup terkonsentrasi — pastikan diversifikasi sektor terjaga", false: "cukup terdiversifikasi"}[r.Portfolio.ItemCount <= 2]
		sb.WriteString(fmt.Sprintf("Dengan %d posisi, portofolio ini %s. ", r.Portfolio.ItemCount, conc))
	} else {
		sb.WriteString(fmt.Sprintf("Dengan %d posisi, portofolio ini cukup terdiversifikasi. ", r.Portfolio.ItemCount))
	}

	if r.BestPerformer.ReturnPct > 10 {
		sb.WriteString(fmt.Sprintf("Saham %s menjadi top performer dengan kenaikan %.1f%%. ", r.BestPerformer.Code, r.BestPerformer.ReturnPct))
	}
	if r.WorstPerformer.ReturnPct < -5 {
		sb.WriteString(fmt.Sprintf("Saham %s menjadi laggard dengan penurunan %.1f%%. Pertimbangkan evaluasi ulang posisi ini. ", r.WorstPerformer.Code, r.WorstPerformer.ReturnPct))
	}

	if r.AvgPER > 0 {
		switch {
		case r.AvgPER < 12:
			sb.WriteString(fmt.Sprintf("Rata-rata PER %.1fx tergolong rendah — valuasi portofolio relatif murah. ", r.AvgPER))
		case r.AvgPER < 20:
			sb.WriteString(fmt.Sprintf("Rata-rata PER %.1fx tergolong wajar. ", r.AvgPER))
		default:
			sb.WriteString(fmt.Sprintf("Rata-rata PER %.1fx tergolong tinggi — perhatikan potensi overvalued. ", r.AvgPER))
		}
	}

	if r.AvgROE > 0 {
		if r.AvgROE > 15 {
			sb.WriteString(fmt.Sprintf("ROE rata-rata %.1f%% sangat baik — emiten dalam portofolio efisien menghasilkan laba. ", r.AvgROE))
		} else {
			sb.WriteString(fmt.Sprintf("ROE rata-rata %.1f%% — masih ada ruang perbaikan profitabilitas. ", r.AvgROE))
		}
	}

	riskLevel := "Moderate"
	if r.Portfolio.ItemCount <= 2 {
		riskLevel = "High (terkonsentrasi)"
	} else if r.Portfolio.ItemCount >= 6 && r.AvgPER < 15 {
		riskLevel = "Low (terdiversifikasi, valuasi wajar)"
	}
	sb.WriteString(fmt.Sprintf("Level risiko: %s. ", riskLevel))

	result := sb.String()
	aiEnhanced := s.AI.ChatWithFallback(
		"Kamu adalah analis portofolio profesional. Tulis analisa naratif portofolio dalam bahasa Indonesia, 2-3 paragraf, profesional dan informatif. Jangan gunakan placeholder. Sertakan rekomendasi.",
		fmt.Sprintf("Data portofolio: %s", result),
		result,
	)
	return aiEnhanced
}

func (s *AIToolsService) emptyReportHTML(name string, now time.Time) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><title>Laporan %s</title>
<style>body{font-family:'Inter',sans-serif;max-width:800px;margin:40px auto;padding:20px;color:#1e293b;background:#fff}
.header{border-bottom:3px solid #2563eb;padding-bottom:16px;margin-bottom:24px}
.header h1{font-size:24px;color:#1e3a8a;margin:0}.header p{color:#64748b;margin:4px 0 0}.empty{text-align:center;padding:60px 20px;color:#94a3b8;font-size:16px}
.btn{display:inline-block;background:#2563eb;color:#fff;padding:10px 24px;border-radius:8px;text-decoration:none;font-weight:600;margin-top:16px}
@media print{.btn{display:none}}</style></head><body>
<div class="header"><h1>Laporan Portofolio: %s</h1><p>Generated: %s</p></div>
<div class="empty"><p>Portofolio ini masih kosong. Tambahkan saham untuk mendapatkan laporan analisa.</p>
<a href="/dashboard/portfolios" class="btn">Kelola Portofolio</a></div></body></html>`, name, name, now.Format("02 Jan 2006 15:04 WIB"))
}

func (s *AIToolsService) buildReportHTML(r ReportData, narrative string) string {
	var holdingsHTML strings.Builder
	for _, h := range r.Portfolio.Holdings {
		returnClass := "positive"
		returnSign := "+"
		if h.ReturnPct < 0 {
			returnClass = "negative"
			returnSign = ""
		}
		holdingsHTML.WriteString(fmt.Sprintf(`<tr>
<td class="code">%s</td><td>%s</td><td>%.0f</td><td class="num">Rp %.0f</td><td class="num">Rp %.0f</td>
<td class="num">Rp %.0f</td><td class="num %s">%s%.1f%%</td><td class="num">%.1f%%</td>
</tr>`, h.Code, h.Name, h.Quantity, h.AvgPrice, h.CurrPrice, h.Value, returnClass, returnSign, h.ReturnPct, h.Weight))
	}

	perfClass := "positive"
	perfSign := "+"
	if r.Portfolio.ReturnPct < 0 {
		perfClass = "negative"
		perfSign = ""
	}

	riskLevel := "Moderate"
	riskColor := "#f59e0b"
	if r.Portfolio.ItemCount <= 2 {
		riskLevel = "High"
		riskColor = "#ef4444"
	} else if r.Portfolio.ItemCount >= 6 && r.AvgPER < 15 {
		riskLevel = "Low"
		riskColor = "#10b981"
	}

	return fmt.Sprintf(`<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><title>Laporan %s · Investo</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}body{font-family:'Inter',sans-serif;font-size:13px;color:#1e293b;background:#fff;max-width:900px;margin:0 auto;padding:20px}
.header{display:flex;justify-content:space-between;align-items:flex-end;border-bottom:4px solid #2563eb;padding-bottom:20px;margin-bottom:28px}
.header-left h1{font-size:26px;color:#1e3a8a;font-weight:800}.header-left p{color:#64748b;font-size:12px;margin-top:2px}
.header-right{text-align:right}.header-right .date{color:#64748b;font-size:11px}
.stats-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin-bottom:28px}
.stat-card{background:#f8fafc;border:1px solid #e2e8f0;border-radius:12px;padding:16px}
.stat-card .label{font-size:10px;color:#94a3b8;text-transform:uppercase;letter-spacing:.06em;font-weight:600;margin-bottom:4px}
.stat-card .value{font-size:20px;font-weight:800;color:#1e293b}.stat-card .value.positive{color:#10b981}.stat-card .value.negative{color:#ef4444}
.section-title{font-size:14px;font-weight:700;color:#1e3a8a;margin:28px 0 12px;padding-bottom:8px;border-bottom:1px solid #e2e8f0}
.narrative{background:#eff6ff;border:1px solid #bfdbfe;border-radius:12px;padding:16px 20px;margin-bottom:24px;font-size:13px;line-height:1.7;color:#1e40af}
table{width:100%%;border-collapse:collapse;font-size:12px}thead th{background:#f1f5f9;color:#475569;font-weight:600;padding:10px 12px;text-align:left;font-size:10px;text-transform:uppercase;letter-spacing:.04em;border-bottom:2px solid #e2e8f0}
tbody td{padding:10px 12px;border-bottom:1px solid #f1f5f9;vertical-align:middle}tbody tr:hover{background:#f8fafc}
td.code{font-weight:700;color:#1e3a8a;font-family:'JetBrains Mono',monospace}.num{text-align:right;font-family:'JetBrains Mono',monospace}.num.positive{color:#10b981}.num.negative{color:#ef4444}
.risk-section{display:flex;gap:16px;margin-top:20px;flex-wrap:wrap}.risk-card{flex:1;min-width:140px;border-radius:12px;padding:16px;border:1px solid #e2e8f0}
.risk-card .risk-label{font-size:10px;color:#94a3b8;text-transform:uppercase;letter-spacing:.04em;font-weight:600}.risk-card .risk-value{font-size:18px;font-weight:800;margin-top:4px}
.footer{text-align:center;color:#94a3b8;font-size:10px;margin-top:32px;padding-top:16px;border-top:1px solid #e2e8f0}
@media print{body{padding:0}}
</style></head><body>
<div class="header">
<div class="header-left"><h1>Laporan Portofolio: %s</h1><p>Analisa performa & risiko portofolio investasi</p></div>
<div class="header-right"><p class="date">Dibuat: %s</p></div>
</div>
<div class="stats-grid">
<div class="stat-card"><div class="label">Total Nilai</div><div class="value">Rp %.0f</div></div>
<div class="stat-card"><div class="label">Total Return</div><div class="value %s">%sRp %.0f</div></div>
<div class="stat-card"><div class="label">Return %%</div><div class="value %s">%s%.2f%%%%</div></div>
<div class="stat-card"><div class="label">Jumlah Posisi</div><div class="value">%d</div></div>
</div>
<div class="narrative"><strong>Analisa AI:</strong> %s</div>
<div class="section-title">Top Holdings</div>
<table><thead><tr><th>Kode</th><th>Nama</th><th>Lot</th><th>Avg</th><th>Harga</th><th>Nilai</th><th>Return</th><th>Bobot</th></tr></thead><tbody>%s</tbody></table>
<div class="risk-section">
<div class="risk-card"><div class="risk-label">Level Risiko</div><div class="risk-value" style="color:%s">%s</div></div>
<div class="risk-card"><div class="risk-label">Rata-rata PER</div><div class="risk-value">%.1fx</div></div>
<div class="risk-card"><div class="risk-label">Rata-rata PBV</div><div class="risk-value">%.1fx</div></div>
<div class="risk-card"><div class="risk-label">Rata-rata ROE</div><div class="risk-value">%.1f%%%%</div></div>
<div class="risk-card"><div class="risk-label">Div Yield Rata-rata</div><div class="risk-value">%.2f%%%%</div></div>
</div>
<div class="footer">Laporan ini dibuat otomatis oleh Investo AI &mdash; bukan rekomendasi investasi. Investasi mengandung risiko.</div>
</body></html>`,
		r.Portfolio.Name,
		r.Portfolio.Name, r.GeneratedAt.Format("02 Jan 2006 15:04 WIB"),
		r.Portfolio.TotalValue,
		perfClass, perfSign, math.Abs(r.Portfolio.TotalReturn),
		perfClass, perfSign, r.Portfolio.ReturnPct,
		r.Portfolio.ItemCount,
		narrative,
		holdingsHTML.String(),
		riskColor, riskLevel,
		r.AvgPER, r.AvgPBV, r.AvgROE, r.DivYieldAvg)
}

func (s *AIToolsService) GenerateAlertMessage(code string, price float64, condition string) string {
	systemPrompt := "Anda adalah analis pasar saham profesional Indonesia. Buat pesan alert yang engaging, informatif, dan menggunakan bahasa Indonesia pasar modal. Sertakan emoji yang sesuai. Berikan konteks teknikal singkat. Jangan lebih dari 2 kalimat."
	userPrompt := fmt.Sprintf("Tulis pesan alert untuk saham %s yang harganya %.0f dan memicu kondisi: %s. Gunakan format: [emoji] [KODE] [aksi/deskripsi singkat] [analisa singkat]. Bahasa Indonesia.", code, price, condition)

	return s.AI.ChatWithFallback(systemPrompt, userPrompt, s.buildAlertFallback(code, price, condition))
}

func (s *AIToolsService) buildAlertFallback(code string, price float64, condition string) string {
	lower := strings.ToLower(condition)
	switch {
	case strings.Contains(lower, "break") || strings.Contains(lower, "breakout"):
		return fmt.Sprintf("🔔 %s breakout di %.0f! Momentum bullish kuat, perhatikan volume untuk konfirmasi. Resistance berikutnya di %.0f.", code, price, price*1.05)
	case strings.Contains(lower, "support") || strings.Contains(lower, "bottom"):
		return fmt.Sprintf("⚠️ %s mendekati support di %.0f. Waspada potensi breakdown jika volume meningkat. Pertimbangkan cut-loss jika support jebol.", code, price)
	case strings.Contains(lower, "resistance") || strings.Contains(lower, "top"):
		return fmt.Sprintf("🔔 %s menembus %.0f! Resistance berhasil ditembus dengan volume kuat. Target berikutnya di %.0f.", code, price, price*1.08)
	case strings.Contains(lower, "volume") || strings.Contains(lower, "lonjakan"):
		return fmt.Sprintf("📊 %s mengalami lonjakan volume di harga %.0f — %.0fx rata-rata. Ada akumulasi besar, pantau kelanjutannya.", code, price, 2.5)
	case strings.Contains(lower, "ma") || strings.Contains(lower, "golden"):
		return fmt.Sprintf("📈 %s golden cross terkonfirmasi di %.0f! MA50 memotong MA200 ke atas — sinyal bullish jangka menengah.", code, price)
	case strings.Contains(lower, "rsi") || strings.Contains(lower, "oversold") || strings.Contains(lower, "overbought"):
		return fmt.Sprintf("⚡ %s RSI oversold di harga %.0f. Potensi technical rebound, tapi tunggu konfirmasi candle reversal.", code, price)
	case strings.Contains(lower, "naik") || strings.Contains(lower, "bullish"):
		return fmt.Sprintf("📈 %s melanjutkan rally ke %.0f. Trend bullish dengan higher high dan higher low. Trailing stop recommended.", code, price)
	case strings.Contains(lower, "turun") || strings.Contains(lower, "bearish"):
		return fmt.Sprintf("📉 %s turun ke %.0f. Momentum bearish, support berikutnya di %.0f. Wait and see sampai ada reversal signal.", code, price, price*0.95)
	default:
		return fmt.Sprintf("🔔 %s mencapai %.0f. Kondisi: %s. Pantau pergerakan selanjutnya untuk konfirmasi sinyal.", code, price, condition)
	}
}

func (s *AIToolsService) GenerateSocialPost(code string, theme string, platform string) string {
	systemPrompt := "Anda adalah analis saham yang membuat konten sosial media tentang pasar modal Indonesia. Gunakan bahasa Indonesia yang engaging dan profesional."
	length := "maksimal 280 karakter (Twitter/X style)"
	maxChars := 280
	if platform == "linkedin" {
		length = "800-1300 karakter (LinkedIn professional style)"
		maxChars = 1300
	}

	userPrompt := fmt.Sprintf("Buat postingan %s tentang saham %s. Tema: %s. Panjang: %s. Sertakan data dan analisa singkat. Jangan gunakan hashtag berlebihan. Bahasa Indonesia.", platform, code, theme, length)

	return s.AI.ChatWithFallback(systemPrompt, userPrompt, s.buildPostFallback(code, theme, platform, maxChars))
}

func (s *AIToolsService) buildPostFallback(code string, theme string, platform string, maxChars int) string {
	var sb strings.Builder

	switch theme {
	case "bullish":
		sb.WriteString(fmt.Sprintf("📈 $%s menunjukkan pola bullish yang menarik. Akumulasi di support kuat dengan volume meningkat, potensi breakout ke resistance berikutnya. Secara fundamental, valuasi masih reasonable dengan pertumbuhan laba double digit. Saya akumulasi di area ini. Bukan rekomendasi, DYOR.\n\n", code))
	case "bearish":
		sb.WriteString(fmt.Sprintf("⚠️ Hati-hati dengan $%s. Pola distribusi mulai terbentuk dengan volume penjualan meningkat. Support kunci di bawah terancam. Saya personally wait and see dulu — lebih baik preserve capital daripada memaksakan entry di saat momentum bearish.\n\n", code))
	case "analysis":
		sb.WriteString(fmt.Sprintf("🔍 Analisa $%s:\n- P/E: valuasi di bawah rata-rata sektor\n- ROE solid, menunjukkan efisiensi manajemen\n- Pertumbuhan laba konsisten 5 tahun terakhir\n- DER manageable, tidak ada masalah solvabilitas\nKesimpulan: perusahaan solid dengan valuasi wajar, tinggal timing entry yang tepat.\n\n", code))
	case "dividend":
		sb.WriteString(fmt.Sprintf("💰 $%s — dividend play yang menarik. Yield di atas rata-rata deposito dengan payout ratio yang sehat. Dividen konsisten naik 5 tahun berturut-turut. Bukan saham untuk trading — ini saham untuk dikoleksi jangka panjang. Compound interest is the 8th wonder!\n\n", code))
	case "breakout":
		sb.WriteString(fmt.Sprintf("🚀 $%s BREAKOUT! Resistance berhasil ditembus dengan volume 3x rata-rata — konfirmasi valid. Target berikutnya di area resistance selanjutnya. Stop loss di bawah breakout level. Risk/reward ratio menarik — worth the entry.\n\n", code))
	default:
		sb.WriteString(fmt.Sprintf("📊 Update $%s: pergerakan menarik hari ini. Pantau level support dan resistance untuk setup trading. Selalu gunakan money management yang baik.\n\n", code))
	}

	if platform == "linkedin" {
		sb.WriteString(fmt.Sprintf("Disclaimer: Ini bukan rekomendasi investasi. Selalu lakukan riset mandiri sebelum mengambil keputusan investasi. Pasar modal mengandung risiko. Pastikan Anda paham profil risiko dan tujuan investasi Anda.\n\n#SahamIndonesia #Investasi #%s #Investo", strings.ToUpper(code)))
	} else {
		sb.WriteString("#SahamIDX #" + code)
	}

	result := sb.String()
	if len(result) > maxChars {
		result = result[:maxChars]
	}
	return result
}

func (s *AIToolsService) SemanticSearch(query string) ([]SemanticSearchResult, error) {
	systemPrompt := `Anda adalah sistem pencarian saham berbasis AI untuk Bursa Efek Indonesia.
Diberikan query natural language dari user, ekstrak kriteria pencarian dalam format JSON.

Kategori sektor: perbankan, teknologi, consumer, properti, tambang, energi, farmasi, telekomunikasi, otomotif, retail, konstruksi, transportasi, media, agroindustri, infrastruktur, kesehatan
Atribut: dividen_tinggi, valuasi_murah, growth_tinggi, market_cap_besar, market_cap_kecil, likuid, volatil, defensif, siklikal, turnaround
Tipe: BUMN, swasta, multinasional

Return HANYA JSON: {"criteria": {"sector": "...", "attribute": "...", "company_type": "...", "keywords": ["..."]}, "explanation": "penjelasan singkat dalam bahasa Indonesia"}`

	userPrompt := fmt.Sprintf("Query user: %s", query)

	fallback := fmt.Sprintf(`{"criteria": {"sector": "", "attribute": "", "company_type": "", "keywords": ["%s"]}, "explanation": "Mencari saham berdasarkan kata kunci: %s"}`, query, query)
	aiResult := s.AI.ChatWithFallback(systemPrompt, userPrompt, fallback)

	type aiCriteria struct {
		Criteria struct {
			Sector      string   `json:"sector"`
			Attribute   string   `json:"attribute"`
			CompanyType string   `json:"company_type"`
			Keywords    []string `json:"keywords"`
		} `json:"criteria"`
		Explanation string `json:"explanation"`
	}

	var criteria aiCriteria
	if b, err := extractJSONFromText(aiResult); err == nil {
		json.Unmarshal(b, &criteria)
	}

	results := s.matchByKeywords(criteria.Criteria.Keywords)
	for i := range results {
		if criteria.Explanation != "" {
			results[i].MatchReason = criteria.Explanation
		}
	}

	return results, nil
}

func extractJSONFromText(s string) ([]byte, error) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return []byte(s[start : end+1]), nil
	}
	return nil, fmt.Errorf("no JSON found")
}

func (s *AIToolsService) matchByKeywords(keywords []string) []SemanticSearchResult {
	var results []SemanticSearchResult

	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return results
	}

	var stockIDs []int64
	for _, stock := range stocks {
		stockIDs = append(stockIDs, stock.ID)
	}
	priceMap, _ := s.StockPriceRepo.GetLatestPrices(stockIDs)

	for _, stock := range stocks {
		score := 0
		codeLower := strings.ToLower(stock.Code)
		nameLower := strings.ToLower(stock.Name)

		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(codeLower, kwLower) {
				score += 3
			}
			if strings.Contains(nameLower, kwLower) {
				score += 2
			}
		}

		sectorKeywords := map[string][]string{
			"perbankan":      {"bank", "bpr", "finance", "finansial", "kredit", "tabungan"},
			"teknologi":      {"tech", "digital", "it", "software", "teknologi", "data", "internet"},
			"consumer":       {"consumer", "konsumsi", "makanan", "minuman", "rokok", " farmasi"},
			"tambang":        {"tambang", "mining", "coal", "mineral", "nikel", "emas", "batu bara"},
			"energi":         {"energi", "energy", "oil", "gas", "listrik", "power"},
			"telekomunikasi": {"telekomunikasi", "telco", "tower", "fiber", "seluler"},
		}

		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			for _, sectorKws := range sectorKeywords {
				for _, skw := range sectorKws {
					if kwLower == skw || strings.Contains(kwLower, skw) {
						score += 1
					}
				}
			}
		}

		if score > 0 {
			price := priceMap[stock.ID]
			prev, _ := s.StockPriceRepo.GetPriceChange(stock.ID, 1)
			changePct := 0.0
			if prev > 0 && price > 0 {
				changePct = ((price - prev) / prev) * 100
			}

			var per, divYield float64
			if fund, err := s.StockFundamentalRepo.FindLatest(stock.ID); err == nil && fund != nil {
				per = fund.PER
				divYield = fund.DividendYield
			}

			results = append(results, SemanticSearchResult{
				Code:        stock.Code,
				Name:        stock.Name,
				Price:       price,
				ChangePct:   changePct,
				PER:         per,
				DivYield:    divYield,
				MatchReason: fmt.Sprintf("Cocok berdasarkan keyword dengan skor relevansi %d", score),
			})
		}
	}

	return results
}

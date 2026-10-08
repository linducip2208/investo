package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/pseo"

	"github.com/go-chi/chi/v5"
)

type PSEOHandler struct {
	StockRepo    *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo   *repository.SectorRepository
	PSEOService  *pseo.Service
	Templates    *template.Template
}

func (h *PSEOHandler) baseURL() string {
	if h.PSEOService == nil || h.PSEOService.AppURL == "" {
		return "https://investo.whitelabel.co.id"
	}
	return strings.TrimRight(h.PSEOService.AppURL, "/")
}

func (h *PSEOHandler) StockPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	prices, _ := h.StockPriceRepo.FindLatest(stock.ID, 1)
	var latestPrice float64
	if len(prices) > 0 {
		latestPrice = prices[0].Close
	}

	fundamental, _ := h.StockFundamentalRepo.FindLatest(stock.ID)

	sector, _ := h.SectorRepo.FindByID(stock.SectorID)

	jsonLD := h.PSEOService.GenerateJSONLDStock(stock, latestPrice)

	metaDescription := fmt.Sprintf("Analisa saham %s (%s) - Harga terkini %.0f. Cek fundamental, PER, PBV, ROE, DER, dividend yield, dan berita terbaru %s.", stock.Name, stock.Code, latestPrice, stock.Code)

	data := map[string]interface{}{
		"Title":            stock.Code + " - Harga Saham " + stock.Name + " Hari Ini - Investo",
		"MetaDescription":  metaDescription,
		"JSONLD":           jsonLD,
		"Stock":            stock,
		"LatestPrice":      latestPrice,
		"Fundamental":      fundamental,
		"Sector":           sector,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/stock.html", data)
}

func (h *PSEOHandler) SectorPage(w http.ResponseWriter, r *http.Request) {
	sectorSlug := chi.URLParam(r, "sector")

	sector, err := h.SectorRepo.FindBySlug(sectorSlug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stocks, _, _ := h.StockRepo.List(0, 10, "", sector.ID)

	type StockWithPrice struct {
		Stock  model.Stock
		Price  float64
		Change float64
	}

	var results []StockWithPrice
	for _, s := range stocks {
		prices, _ := h.StockPriceRepo.FindLatest(s.ID, 1)
		var price float64
		if len(prices) > 0 {
			price = prices[0].Close
		}
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 30)
		results = append(results, StockWithPrice{Stock: s, Price: price, Change: change})
	}

	currentYear := time.Now().Year()
	jsonLD := h.PSEOService.GenerateJSONLDSector(sector, currentYear)

	metaDescription := fmt.Sprintf("Daftar saham terbaik di sektor %s %d. Analisa fundamental lengkap: PER, PBV, ROE, DER. Rekomendasi saham pilihan di sektor %s.", sector.Name, currentYear, sector.Name)

	data := map[string]interface{}{
		"Title":            fmt.Sprintf("Saham Terbaik Sektor %s %d - Investo", sector.Name, currentYear),
		"MetaDescription":  metaDescription,
		"JSONLD":           jsonLD,
		"Sector":           sector,
		"Stocks":           results,
		"Year":             currentYear,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/sector.html", data)
}

func (h *PSEOHandler) SectorYearPage(w http.ResponseWriter, r *http.Request) {
	sectorSlug := chi.URLParam(r, "sector")
	yearStr := chi.URLParam(r, "year")
	year, _ := strconv.Atoi(yearStr)
	if year == 0 {
		year = time.Now().Year()
	}

	sector, err := h.SectorRepo.FindBySlug(sectorSlug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stocks, _, _ := h.StockRepo.List(0, 10, "", sector.ID)

	type StockWithPrice struct {
		Stock  model.Stock
		Price  float64
		Change float64
	}

	var results []StockWithPrice
	for _, s := range stocks {
		prices, _ := h.StockPriceRepo.FindLatest(s.ID, 1)
		var price float64
		if len(prices) > 0 {
			price = prices[0].Close
		}
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 30)
		results = append(results, StockWithPrice{Stock: s, Price: price, Change: change})
	}

	jsonLD := h.PSEOService.GenerateJSONLDSector(sector, year)

	metaDescription := fmt.Sprintf("Saham terbaik di sektor %s tahun %d berdasarkan performa dan fundamental. Data lengkap PER, PBV, ROE, DER untuk investasi jangka panjang.", sector.Name, year)

	data := map[string]interface{}{
		"Title":            fmt.Sprintf("Saham Terbaik Sektor %s %d - Investo", sector.Name, year),
		"MetaDescription":  metaDescription,
		"JSONLD":           jsonLD,
		"Sector":           sector,
		"Stocks":           results,
		"Year":             year,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/sector.html", data)
}

func (h *PSEOHandler) ComparePage(w http.ResponseWriter, r *http.Request) {
	codeA := chi.URLParam(r, "code-a")
	codeB := chi.URLParam(r, "code-b")

	stockA, errA := h.StockRepo.FindByCode(codeA)
	stockB, errB := h.StockRepo.FindByCode(codeB)

	if errA != nil || errB != nil {
		http.NotFound(w, r)
		return
	}

	fundA, _ := h.StockFundamentalRepo.FindLatest(stockA.ID)
	fundB, _ := h.StockFundamentalRepo.FindLatest(stockB.ID)

	pricesA, _ := h.StockPriceRepo.FindLatest(stockA.ID, 1)
	pricesB, _ := h.StockPriceRepo.FindLatest(stockB.ID, 1)

	var priceA, priceB float64
	if len(pricesA) > 0 {
		priceA = pricesA[0].Close
	}
	if len(pricesB) > 0 {
		priceB = pricesB[0].Close
	}

	changeA, _ := h.StockPriceRepo.GetPriceChange(stockA.ID, 30)
	changeB, _ := h.StockPriceRepo.GetPriceChange(stockB.ID, 30)

	jsonLD := h.PSEOService.GenerateJSONLDCompare(stockA, stockB)

	metaDescription := fmt.Sprintf("Perbandingan head-to-head %s (%s) vs %s (%s): harga, PER, PBV, ROE, DER, dividend yield. Mana yang lebih bagus untuk investasi?", stockA.Name, stockA.Code, stockB.Name, stockB.Code)

	data := map[string]interface{}{
		"Title":            fmt.Sprintf("%s vs %s - Perbandingan Saham - Investo", stockA.Code, stockB.Code),
		"MetaDescription":  metaDescription,
		"JSONLD":           jsonLD,
		"StockA":           stockA,
		"StockB":           stockB,
		"FundA":            fundA,
		"FundB":            fundB,
		"PriceA":           priceA,
		"PriceB":           priceB,
		"ChangeA":          changeA,
		"ChangeB":          changeB,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/compare.html", data)
}

func (h *PSEOHandler) AlternativesPage(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stock, err := h.StockRepo.FindByCode(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stocks, _, _ := h.StockRepo.List(0, 20, "", stock.SectorID)

	type StockWithPrice struct {
		Stock  model.Stock
		Price  float64
		Change float64
	}

	var alternatives []StockWithPrice
	for _, s := range stocks {
		if s.ID == stock.ID {
			continue
		}
		prices, _ := h.StockPriceRepo.FindLatest(s.ID, 1)
		var price float64
		if len(prices) > 0 {
			price = prices[0].Close
		}
		change, _ := h.StockPriceRepo.GetPriceChange(s.ID, 30)
		alternatives = append(alternatives, StockWithPrice{Stock: s, Price: price, Change: change})
	}

	sector, _ := h.SectorRepo.FindByID(stock.SectorID)

	metaDescription := fmt.Sprintf("Alternatif saham %s (%s) - Cari saham sejenis di sektor %s. Bandingkan fundamental, harga, dan prospek investasi.", stock.Name, stock.Code, sector.Name)

	data := map[string]interface{}{
		"Title":            fmt.Sprintf("Alternatif Saham %s - Investo", stock.Code),
		"MetaDescription":  metaDescription,
		"Stock":            stock,
		"Alternatives":     alternatives,
		"Sector":           sector,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/alternatives.html", data)
}

func (h *PSEOHandler) SourceCodePage(w http.ResponseWriter, r *http.Request) {
	metaDescription := "Beli aplikasi saham dan trading platform siap pakai. Source code Go + MySQL lengkap dengan fitur: screener fundamental, chart teknikal, portfolio tracking, alert harga, berita saham. Whitelabel ready."

	data := map[string]interface{}{
		"Title":            "Beli Aplikasi Saham - Source Code Platform Investasi - Investo",
		"MetaDescription":  metaDescription,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/source-code.html", data)
}

func (h *PSEOHandler) GenericPSEOPage(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Path

	metaDescription := "Platform investasi saham dan forex Indonesia. Data fundamental lengkap, chart teknikal, screener, portfolio tracker."

	data := map[string]interface{}{
		"Title":           fmt.Sprintf("%s - Investo", u),
		"MetaDescription": metaDescription,
		"URL":             u,
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/generic.html", data)
}

func (h *PSEOHandler) GlossaryPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	terms := pseo.GetAllGlossaryTerms()
	var term *pseo.GlossaryTerm
	for _, t := range terms {
		if t.Slug == slug {
			term = &t
			break
		}
	}

	if term == nil {
		http.NotFound(w, r)
		return
	}

	var related []map[string]string
	count := 0
	for _, t := range terms {
		if t.Slug != slug && t.Category == term.Category && count < 5 {
			shortDef := t.Definition
			if len(shortDef) > 80 {
				shortDef = shortDef[:80] + "..."
			}
			related = append(related, map[string]string{
				"Term":    t.Term,
				"Slug":    t.Slug,
				"ShortDef": shortDef,
			})
			count++
		}
	}

	shortDef := term.Definition
	if len(shortDef) > 150 {
		shortDef = shortDef[:150] + "..."
	}

	defTruncated := term.Definition
	if len(defTruncated) > 200 {
		defTruncated = defTruncated[:200] + "..."
	}

	categoryLabels := map[string]string{
		"fundamental": "Fundamental",
		"teknikal":    "Teknikal",
		"pasar":       "Pasar",
		"saham":       "Saham",
		"trading":     "Trading",
		"korporasi":   "Aksi Korporasi",
		"instrumen":   "Instrumen",
		"strategi":    "Strategi",
		"psikologi":   "Psikologi Trading",
		"dasar":       "Dasar",
		"umum":        "Umum",
	}

	categoryLabel := categoryLabels[term.Category]
	if categoryLabel == "" {
		categoryLabel = "Umum"
	}

	data := map[string]interface{}{
		"Term":               term.Term,
		"Slug":               term.Slug,
		"Definition":         term.Definition,
		"ShortDefinition":    shortDef,
		"DefinitionTruncated": defTruncated,
		"Category":           term.Category,
		"CategoryLabel":      categoryLabel,
		"RelatedTerms":       related,
		"UsageContext":       "menganalisis dan mengevaluasi saham di Bursa Efek Indonesia",
		"MetaDescription":    fmt.Sprintf("Pengertian %s dalam investasi saham. Penjelasan lengkap, rumus, contoh penggunaan, dan tips untuk investor pemula Indonesia.", term.Term),
		"CanonicalUrl":       fmt.Sprintf("%s/istilah/%s", h.baseURL(), slug),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/glossary.html", data)
}

func (h *PSEOHandler) CityPage(w http.ResponseWriter, r *http.Request) {
	citySlug := chi.URLParam(r, "city")

	cities := pseo.GetIndonesianCities()
	var city *pseo.CityPage
	for _, c := range cities {
		if c.Slug == citySlug {
			city = &c
			break
		}
	}

	if city == nil {
		http.NotFound(w, r)
		return
	}

	stocks, _, _ := h.StockRepo.List(0, 6, "", 0)

	type RecStock struct {
		Code   string
		Name   string
		Sector string
	}
	var recommended []RecStock
	for _, s := range stocks {
		sector, _ := h.SectorRepo.FindByID(s.SectorID)
		sectorName := ""
		if sector != nil {
			sectorName = sector.Name
		}
		recommended = append(recommended, RecStock{
			Code:   s.Code,
			Name:   s.Name,
			Sector: sectorName,
		})
	}

	data := map[string]interface{}{
		"Title":             fmt.Sprintf("Belajar Saham di %s - Panduan Lengkap", city.City),
		"MetaDescription":   fmt.Sprintf("Panduan belajar investasi saham untuk pemula di %s. Temukan komunitas, sekuritas, dan mentor saham terdekat di %s, %s.", city.City, city.City, city.Province),
		"City":              city.City,
		"Province":          city.Province,
		"IsCommunity":       false,
		"IsApp":             false,
		"RecommendedStocks": recommended,
		"CanonicalUrl":      fmt.Sprintf("%s/belajar-saham-%s", h.baseURL(), citySlug),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/city.html", data)
}

func (h *PSEOHandler) CityCommunityPage(w http.ResponseWriter, r *http.Request) {
	citySlug := chi.URLParam(r, "city")

	cities := pseo.GetIndonesianCities()
	var city *pseo.CityPage
	for _, c := range cities {
		if c.Slug == citySlug {
			city = &c
			break
		}
	}

	if city == nil {
		http.NotFound(w, r)
		return
	}

	stocks, _, _ := h.StockRepo.List(0, 6, "", 0)

	type RecStock struct {
		Code   string
		Name   string
		Sector string
	}
	var recommended []RecStock
	for _, s := range stocks {
		sector, _ := h.SectorRepo.FindByID(s.SectorID)
		sectorName := ""
		if sector != nil {
			sectorName = sector.Name
		}
		recommended = append(recommended, RecStock{
			Code:   s.Code,
			Name:   s.Name,
			Sector: sectorName,
		})
	}

	data := map[string]interface{}{
		"Title":             fmt.Sprintf("Komunitas Saham %s - Grup Diskusi & Belajar Investasi", city.City),
		"MetaDescription":   fmt.Sprintf("Gabung komunitas investor saham di %s. Temukan grup diskusi, workshop, dan sesi sharing pengalaman investasi di %s, %s.", city.City, city.City, city.Province),
		"City":              city.City,
		"Province":          city.Province,
		"IsCommunity":       true,
		"IsApp":             false,
		"RecommendedStocks": recommended,
		"CanonicalUrl":      fmt.Sprintf("%s/komunitas-saham-%s", h.baseURL(), citySlug),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/city.html", data)
}

func (h *PSEOHandler) CityAppPage(w http.ResponseWriter, r *http.Request) {
	citySlug := chi.URLParam(r, "city")

	cities := pseo.GetIndonesianCities()
	var city *pseo.CityPage
	for _, c := range cities {
		if c.Slug == citySlug {
			city = &c
			break
		}
	}

	if city == nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":           fmt.Sprintf("Aplikasi Saham Terbaik untuk Investor di %s", city.City),
		"MetaDescription": fmt.Sprintf("Rekomendasi aplikasi trading saham terbaik untuk investor di %s. Bandingkan fitur, biaya, dan kemudahan penggunaan.", city.City),
		"City":            city.City,
		"Province":        city.Province,
		"IsCommunity":     false,
		"IsApp":           true,
		"CanonicalUrl":    fmt.Sprintf("%s/aplikasi-saham-%s", h.baseURL(), citySlug),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/city.html", data)
}

type HowToStep struct {
	Name        string
	Description string
	Tips        string
}

func (h *PSEOHandler) HowToPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	howTos := map[string]struct {
		PageTitle       string
		IsStockSpecific bool
		Steps           []HowToStep
	}{
		"mulai-investasi-saham": {
			PageTitle: "Cara Mulai Investasi Saham untuk Pemula",
			Steps: []HowToStep{
				{Name: "Pelajari Dasar Pasar Modal", Description: "Sebelum mulai investasi, pahami dulu apa itu saham, bagaimana cara kerjanya, risiko dan potensi return. Baca buku, ikuti webinar, dan pelajari istilah-istilah dasar seperti IHSG, lot, bid/offer.", Tips: "Alokasikan waktu 1-2 minggu untuk belajar dasar sebelum terjun."},
				{Name: "Buka Rekening Saham", Description: "Pilih sekuritas yang terdaftar di OJK. Siapkan KTP, NPWP, dan rekening bank. Buka rekening saham secara online melalui aplikasi sekuritas. Anda akan mendapatkan RDN (Rekening Dana Nasabah) dan SID (Single Investor ID).", Tips: "Bandingkan biaya transaksi antar sekuritas. Pilih yang fee-nya kompetitif."},
				{Name: "Deposit Modal Awal", Description: "Transfer dana ke RDN Anda. Modal minimal pembelian saham adalah 1 lot (100 lembar). Untuk pemula, disarankan mulai dengan modal Rp 500.000 - Rp 1.000.000.", Tips: "Gunakan dana dingin (uang yang tidak terpakai untuk kebutuhan sehari-hari)."},
				{Name: "Pilih Saham Pertama", Description: "Untuk pemula, pilih saham blue chip atau LQ45 yang fundamentalnya bagus dan likuid. Gunakan aplikasi Investo untuk screening saham berdasarkan PER, PBV, ROE, dan DER.", Tips: "Jangan langsung beli banyak saham. Mulai dengan 1-2 saham dulu."},
				{Name: "Lakukan Pembelian", Description: "Buka aplikasi trading sekuritas Anda, cari kode saham yang diinginkan, masukkan jumlah lot dan harga beli (bid price). Gunakan limit order untuk kontrol harga.", Tips: "Belilah secara bertahap, jangan sekaligus semua modal dalam satu kali transaksi."},
				{Name: "Pantau dan Evaluasi", Description: "Monitor portofolio Anda secara berkala, baca berita terkait saham yang dimiliki, dan evaluasi kinerja investasi setiap 3-6 bulan.", Tips: "Jangan panic selling saat harga turun. Fokus pada fundamental jangka panjang."},
			},
		},
		"baca-laporan-keuangan": {
			PageTitle: "Cara Membaca Laporan Keuangan Perusahaan",
			Steps: []HowToStep{
				{Name: "Kenali 3 Laporan Utama", Description: "Laporan keuangan terdiri dari: Laporan Laba Rugi (Income Statement), Neraca (Balance Sheet), dan Laporan Arus Kas (Cash Flow). Masing-masing memberikan informasi berbeda tentang kesehatan perusahaan.", Tips: "Mulailah dengan laporan laba rugi karena paling mudah dipahami."},
				{Name: "Analisis Pendapatan dan Laba", Description: "Lihat tren pendapatan (revenue) dan laba bersih (net income) dari tahun ke tahun. Pertumbuhan yang konsisten adalah tanda positif. Perhatikan juga beban operasional.", Tips: "Bandingkan pertumbuhan revenue dengan kompetitor di sektor yang sama."},
				{Name: "Evaluasi Neraca", Description: "Periksa total aset, total liabilitas (utang), dan ekuitas. Hitung DER (Debt to Equity Ratio) untuk melihat tingkat utang. DER di bawah 1 umumnya sehat.", Tips: "Perhatikan piutang dan persediaan yang meningkat tajam tanpa kenaikan pendapatan."},
				{Name: "Cek Arus Kas", Description: "Arus kas operasional positif menunjukkan perusahaan menghasilkan uang dari bisnis intinya. Arus kas investasi negatif bisa berarti perusahaan sedang ekspansi.", Tips: "Laba bersih tinggi tapi arus kas operasional negatif = red flag."},
				{Name: "Hitung Rasio Keuangan", Description: "Gunakan rasio seperti PER, PBV, ROE, ROA, NPM untuk mengevaluasi valuasi dan profitabilitas. Bandingkan dengan rata-rata industri.", Tips: "Jangan terpaku pada satu rasio. Gunakan kombinasi untuk gambaran menyeluruh."},
			},
		},
		"analisa-teknikal": {
			PageTitle: "Cara Analisa Teknikal Saham",
			Steps: []HowToStep{
				{Name: "Pahami Dasar Chart", Description: "Chart adalah representasi visual pergerakan harga. Kenali jenis chart: line, bar, dan candlestick. Candlestick paling populer karena informatif.", Tips: "Mulai dengan timeframe daily untuk belajar pola dasar."},
				{Name: "Identifikasi Support & Resistance", Description: "Support adalah level harga di mana tekanan beli muncul (harga sulit turun lebih jauh). Resistance adalah level di mana tekanan jual muncul. Gambar garis horizontal di puncak dan lembah signifikan.", Tips: "Level yang sudah di-test berkali-kali lebih valid."},
				{Name: "Gunakan Moving Average", Description: "Tambahkan MA20, MA50, dan MA200 di chart. Posisi harga terhadap MA menunjukkan tren. Golden cross (MA50 potong MA200 ke atas) = sinyal bullish.", Tips: "MA berfungsi sebagai support/resistance dinamis."},
				{Name: "Aplikasikan Indikator Momentum", Description: "RSI untuk mengukur overbought (>70) dan oversold (<30). MACD untuk sinyal crossover. Stochastic untuk konfirmasi momentum.", Tips: "Jangan pakai terlalu banyak indikator sekaligus. 2-3 cukup."},
				{Name: "Kenali Pola Candlestick", Description: "Pola reversal seperti Hammer, Shooting Star, Engulfing. Pola lanjutan seperti Flag, Pennant. Konfirmasi dengan volume.", Tips: "Tunggu konfirmasi candle berikutnya sebelum entry."},
			},
		},
		"pilih-saham": {
			PageTitle: "Cara Memilih Saham yang Bagus",
			Steps: []HowToStep{
				{Name: "Tentukan Strategi Investasi", Description: "Apakah Anda value investor (cari saham murah), growth investor (cari pertumbuhan), atau dividend investor (cari dividen)? Strategi menentukan kriteria screening.", Tips: "Value + dividend = strategi paling aman untuk pemula."},
				{Name: "Screen dengan Rasio Fundamental", Description: "Gunakan stock screener untuk filter: PER < 15, PBV < 2, ROE > 15%, DER < 1, dividend yield > 2%. Sesuaikan dengan strategi Anda.", Tips: "Gunakan screener Investo untuk filter multi-kriteria."},
				{Name: "Analisis Sektor dan Prospek", Description: "Pilih sektor dengan prospek positif. Misalnya: perbankan (suku bunga rendah), consumer goods (daya beli naik), teknologi (digitalisasi).", Tips: "Hindari sektor yang sedang downtrend struktural."},
				{Name: "Baca Laporan Tahunan", Description: "Download annual report dari website perusahaan atau IDX. Baca bagian management discussion untuk memahami strategi dan risiko.", Tips: "Fokus pada kata kunci: ekspansi, efisiensi, inovasi, market share."},
				{Name: "Cek Kepemilikan Institusi", Description: "Saham dengan kepemilikan institusi tinggi (reksadana, asuransi, dana pensiun) cenderung lebih stabil dan likuid.", Tips: "Data kepemilikan tersedia di laporan bulanan KSEI."},
			},
		},
		"buka-rekening-saham": {
			PageTitle: "Cara Buka Rekening Saham Online",
			Steps: []HowToStep{
				{Name: "Pilih Sekuritas", Description: "Bandingkan sekuritas berdasarkan: biaya transaksi (buy/sell fee), platform trading, riset/rekomendasi, dan kemudahan pembukaan rekening.", Tips: "Pilih sekuritas yang sudah terdaftar di OJK dan menjadi anggota BEI."},
				{Name: "Siapkan Dokumen", Description: "Siapkan KTP (asli), NPWP (jika ada), buku tabungan/rekening bank, dan data pribadi lainnya.", Tips: "Foto/scan dokumen dengan jelas untuk verifikasi online."},
				{Name: "Isi Formulir Online", Description: "Download aplikasi sekuritas atau buka website-nya. Isi formulir pembukaan rekening dengan data diri lengkap. Upload dokumen yang diminta.", Tips: "Pastikan data sesuai KTP untuk mempermudah verifikasi."},
				{Name: "Verifikasi & Aktivasi", Description: "Sekuritas akan melakukan verifikasi data. Proses biasanya 1-3 hari kerja. Setelah disetujui, Anda akan menerima: user ID, password, PIN transaksi, RDN, dan SID.", Tips: "Simpan semua informasi login di tempat aman."},
				{Name: "Top Up RDN & Mulai Trading", Description: "Transfer dana ke RDN via mobile banking atau ATM. Setelah dana masuk, Anda sudah bisa mulai membeli saham!", Tips: "Lakukan test pembelian 1 lot saham untuk memastikan sistem berjalan."},
			},
		},
		"hitung-capital-gain": {
			PageTitle: "Cara Menghitung Capital Gain Saham",
			Steps: []HowToStep{
				{Name: "Pahami Rumus Dasar", Description: "Capital Gain = Harga Jual - Harga Beli. Return (%) = (Capital Gain / Harga Beli) x 100%. Capital gain positif = untung, negatif = rugi.", Tips: "Catat setiap transaksi untuk menghitung gain/loss dengan akurat."},
				{Name: "Hitung Biaya Transaksi", Description: "Biaya beli: fee broker beli (0.15%-0.25%) + PPN 10% dari fee + Levy BEI 0.01%. Biaya jual: fee broker jual + PPN + Levy BEI + PPh final 0.1% dari nilai transaksi.", Tips: "Net capital gain = capital gain - total biaya transaksi."},
				{Name: "Contoh Perhitungan", Description: "Beli 10 lot (1000 lembar) saham BBCA di harga Rp 10.000. Biaya beli ~Rp 35.000. Jual di harga Rp 11.000. Biaya jual ~Rp 148.500. Capital gain = (Rp 11.000.000 - Rp 10.000.000) - Rp 183.500 = Rp 816.500.", Tips: "Gunakan kalkulator saham online untuk perhitungan cepat."},
				{Name: "Perhitungkan Pajak", Description: "Pajak final 0.1% dari nilai transaksi penjualan langsung dipotong oleh broker. Untuk dividen, pajak 10% untuk WP dalam negeri.", Tips: "Simpan bukti potong pajak dari broker untuk pelaporan SPT."},
			},
		},
		"klaim-dividen": {
			PageTitle: "Cara Klaim Dividen Saham",
			Steps: []HowToStep{
				{Name: "Pahami Jadwal Dividen", Description: "Ada 4 tanggal penting: Cum Date (terakhir beli saham dapat dividen), Ex Date (mulai tanpa hak dividen), Recording Date (tanggal pencatatan pemegang saham berhak), Payment Date (tanggal pembayaran).", Tips: "Beli saham sebelum cum date untuk dapat dividen."},
				{Name: "Pastikan Punya Saham Saat Cum Date", Description: "Anda harus memiliki saham di portofolio pada akhir cum date. Jika beli saat cum date dan jual keesokan harinya (ex date), Anda tetap berhak dividen.", Tips: "Strategi dividend capturing berisiko karena harga turun di ex date."},
				{Name: "Dividen Otomatis Masuk ke RDN", Description: "Pada payment date, dividen tunai akan otomatis ditransfer ke RDN (Rekening Dana Nasabah) Anda. Tidak perlu klaim manual.", Tips: "Cek mutasi RDN di aplikasi sekuritas pada payment date +1 hari."},
				{Name: "Perhitungkan Pajak Dividen", Description: "Dividen tunai dikenakan PPh final 10% untuk WP dalam negeri. Jumlah yang masuk ke RDN sudah dipotong pajak. Contoh: dividen Rp 500.000 → diterima Rp 450.000.", Tips: "Dividen dari saham yang dibeli kurang dari 3 bulan dikenakan pajak progresif."},
			},
		},
		"hindari-saham-gorengan": {
			PageTitle: "Cara Menghindari Saham Gorengan",
			Steps: []HowToStep{
				{Name: "Kenali Ciri-cirinya", Description: "Saham gorengan memiliki: fundamental buruk (rugi, DER tinggi), market cap kecil, harga fluktuatif ekstrem, volume tidak wajar, bid-ask queue mencurigakan.", Tips: "Cek laporan keuangan: jika rugi bertahun-tahun, hati-hati."},
				{Name: "Gunakan Screener untuk Filter", Description: "Set filter minimal: PER < 30, DER < 2, ROE > 5%, market cap > Rp 1 triliun. Ini otomatis menyaring mayoritas saham gorengan.", Tips: "Gunakan screener Investo dengan preset filter anti-gorengan."},
				{Name: "Waspada Rekomendasi Sosmed", Description: "Hindari saham yang direkomendasikan influencer dengan iming-iming cuan cepat. Grup pompom (pump and dump) sering menggunakan Telegram/Discord.", Tips: "Selalu verifikasi fundamental sebelum ikut rekomendasi."},
				{Name: "Disiplin Cut Loss", Description: "Jika tidak sengaja masuk saham gorengan, segera cut loss. Jangan averaging down di saham yang fundamentalnya buruk.", Tips: "Rugi kecil lebih baik daripada harapan yang membuat Anda trapped."},
			},
		},
		"diversifikasi-portofolio": {
			PageTitle: "Cara Diversifikasi Portofolio Saham",
			Steps: []HowToStep{
				{Name: "Diversifikasi Sektoral", Description: "Jangan semua saham di satu sektor. Sebar di minimal 3-4 sektor berbeda: perbankan, consumer goods, energi, infrastruktur, teknologi.", Tips: "Jika satu sektor turun, sektor lain mungkin naik — mengurangi volatilitas portofolio."},
				{Name: "Diversifikasi Kapitalisasi", Description: "Kombinasikan big cap (stabil), mid cap (pertumbuhan), dan small cap (agresif). Proporsi: 50% big cap, 30% mid cap, 20% small cap untuk profil moderat.", Tips: "Sesuaikan proporsi dengan profil risiko Anda."},
				{Name: "Batasi Exposure per Saham", Description: "Maksimal 15-20% portofolio untuk 1 saham. Ini mencegah kerugian besar jika satu saham bermasalah.", Tips: "Idealnya 5-10 saham untuk portofolio di bawah Rp 100 juta."},
				{Name: "Rebalancing Berkala", Description: "Setiap 6 bulan, evaluasi ulang alokasi. Jual sebagian yang sudah overweight, beli yang underweight sesuai target alokasi.", Tips: "Rebalancing memaksa Anda jual tinggi dan beli rendah secara disiplin."},
			},
		},
		"pakai-screener": {
			PageTitle: "Cara Pakai Stock Screener",
			Steps: []HowToStep{
				{Name: "Tentukan Kriteria", Description: "Mulai dengan kriteria dasar: PER < 15 (valuasi murah), ROE > 15% (profitabel), DER < 1 (utang terkendali), dividend yield > 2% (dividen).", Tips: "Jangan terlalu ketat di awal. Longgarkan filter jika hasil terlalu sedikit."},
				{Name: "Filter Sektor", Description: "Pilih 1-2 sektor yang Anda pahami. Screener akan menampilkan hanya saham di sektor tersebut yang memenuhi kriteria.", Tips: "Fokus pada sektor yang Anda kenali model bisnisnya."},
				{Name: "Sortir dan Bandingkan", Description: "Sortir hasil berdasarkan PER (terendah) atau ROE (tertinggi). Bandingkan 3-5 saham teratas untuk memilih yang terbaik.", Tips: "Gunakan fitur Compare di Investo untuk head-to-head comparison."},
				{Name: "Simpan Screen", Description: "Simpan kriteria screening Anda untuk digunakan kembali. Setiap bulan, jalankan ulang screen yang sama untuk menemukan peluang baru.", Tips: "Market berubah, saham yang bulan lalu mahal mungkin murah bulan ini."},
			},
		},
	}

	howTo, exists := howTos[slug]
	if !exists {

		code := slug
		if strings.HasPrefix(slug, "beli-saham-") {
			code = strings.TrimPrefix(slug, "beli-saham-")
		} else if strings.HasPrefix(slug, "analisa-saham-") {
			code = strings.TrimPrefix(slug, "analisa-saham-")
		} else {
			http.NotFound(w, r)
			return
		}

		code = strings.ToUpper(code)
		stock, err := h.StockRepo.FindByCode(code)
		if err != nil {
			genericHowTo, ok := howTos[slug]
			if !ok {
				http.NotFound(w, r)
				return
			}
			howTo = genericHowTo
		} else {
			howTo.IsStockSpecific = true
			howTo.Steps = []HowToStep{}
			if strings.HasPrefix(slug, "beli-saham-") {
				howTo.PageTitle = fmt.Sprintf("Cara Beli Saham %s (%s)", code, stock.Name)
				howTo.Steps = []HowToStep{
					{Name: fmt.Sprintf("Kenali Profil %s", code), Description: fmt.Sprintf("%s (%s) adalah perusahaan di sektor %s. Pahami model bisnis, produk/layanan, dan prospek industrinya sebelum membeli.", code, stock.Name, "terkait")},
					{Name: "Buka Aplikasi Trading", Description: "Login ke aplikasi sekuritas Anda. Pastikan RDN memiliki saldo cukup untuk pembelian.", Tips: "Cek harga terakhir di aplikasi Investo sebelum order."},
					{Name: fmt.Sprintf("Masukkan Order Beli %s", code), Description: fmt.Sprintf("Cari kode saham %s. Masukkan jumlah lot (minimal 1 lot = 100 lembar) dan harga beli. Gunakan limit order untuk kontrol harga.", code)},
					{Name: "Konfirmasi dan Monitor", Description: "Setelah order terisi, saham masuk ke portofolio Anda. Pantau pergerakan harga dan berita terkait secara berkala."},
				}
			} else {
				howTo.PageTitle = fmt.Sprintf("Cara Analisa Saham %s (%s)", code, stock.Name)
				howTo.Steps = []HowToStep{
					{Name: fmt.Sprintf("Analisa Fundamental %s", code), Description: fmt.Sprintf("Cek laporan keuangan %s: PER, PBV, ROE, DER, revenue growth. Bandingkan dengan kompetitor di sektor yang sama.", code), Tips: "Gunakan halaman fundamental di Investo untuk data lengkap."},
					{Name: fmt.Sprintf("Analisa Teknikal %s", code), Description: "Buka chart harga. Identifikasi tren (naik/turun/sideways), support/resistance, dan pola candlestick.", Tips: "Gunakan multi-timeframe: daily untuk tren, hourly untuk entry."},
					{Name: "Cek Sentimen Pasar", Description: "Baca berita terbaru tentang emiten. Cek foreign flow (net buy/net sell asing). Lihat rekomendasi analis.", Tips: "Sentimen bisa menggerakkan harga lebih cepat dari fundamental."},
					{Name: "Tentukan Target Harga", Description: "Berdasarkan analisis fundamental dan teknikal, tentukan target harga beli dan jual. Hitung risk/reward ratio minimal 1:2."},
				}
			}
		}
	}

	var relatedHowTos []map[string]string
	count := 0
	for k, v := range howTos {
		if k != slug && count < 4 {
			relatedHowTos = append(relatedHowTos, map[string]string{
				"URL":         "/cara/" + k,
				"Title":       v.PageTitle,
				"Description": v.PageTitle,
			})
			count++
		}
	}

	data := map[string]interface{}{
		"Title":           howTo.PageTitle + " | Investo",
		"MetaDescription": fmt.Sprintf("Panduan langkah demi langkah: %s. Pelajari cara dan tips praktis untuk investasi saham di Indonesia.", howTo.PageTitle),
		"PageTitle":       howTo.PageTitle,
		"Steps":           howTo.Steps,
		"IsStockSpecific": howTo.IsStockSpecific,
		"RelatedHowTo":    relatedHowTos,
		"CanonicalUrl":    fmt.Sprintf("%s/cara/%s", h.baseURL(), slug),
		"User":  safeUser(middleware.GetUser(r)),
	}
	h.Templates.ExecuteTemplate(w, "pseo/howto.html", data)
}

func (h *PSEOHandler) last(idx int, items interface{}) bool {
	return idx == 0
}

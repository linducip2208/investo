package pseo

import (
	"fmt"
	"strings"
)

type PSEORoute struct {
	Pattern     string
	Handler     string
	Title       string
	Description string
	Category    string
}

type GlossaryTerm struct {
	Term       string
	Slug       string
	Definition string
	Category   string
}

type CityPage struct {
	City     string
	Slug     string
	Province string
}

func GetAllGlossaryTerms() []GlossaryTerm {
	terms := map[string]string{
		"PER":                "PER (Price to Earnings Ratio) adalah rasio valuasi yang membandingkan harga saham dengan laba per saham (EPS). PER dihitung dengan rumus: PER = Harga Saham / EPS. Rasio ini menunjukkan berapa kali investor bersedia membayar untuk setiap rupiah laba yang dihasilkan perusahaan. Semakin rendah PER, semakin murah valuasi saham tersebut relatif terhadap labanya. Namun, PER rendah juga bisa mengindikasikan ekspektasi pertumbuhan yang rendah. Di Indonesia, PER rata-rata pasar berkisar antara 12-18x.",
		"PBV":               "PBV (Price to Book Value) adalah rasio yang membandingkan harga pasar saham dengan nilai buku per saham (BVPS). PBV dihitung dengan rumus: PBV = Harga Saham / BVPS. PBV di bawah 1 menunjukkan saham diperdagangkan di bawah nilai bukunya, yang bisa menjadi indikasi undervalued. Sebaliknya, PBV di atas 1 menunjukkan pasar menghargai perusahaan lebih tinggi dari aset bersihnya. PBV sangat relevan untuk sektor perbankan dan keuangan.",
		"ROE":               "ROE (Return on Equity) adalah rasio profitabilitas yang mengukur kemampuan perusahaan menghasilkan laba dari ekuitas pemegang saham. ROE dihitung dengan: ROE = Laba Bersih / Ekuitas x 100%. Semakin tinggi ROE, semakin efisien perusahaan dalam menggunakan modal pemegang saham. ROE di atas 15% umumnya dianggap baik. Investor seperti Warren Buffett sering mencari perusahaan dengan ROE konsisten di atas 15-20%.",
		"ROA":               "ROA (Return on Assets) adalah rasio yang mengukur kemampuan perusahaan menghasilkan laba dari total aset yang dimiliki. ROA dihitung dengan: ROA = Laba Bersih / Total Aset x 100%. ROA menunjukkan efisiensi manajemen dalam menggunakan aset perusahaan. Semakin tinggi ROA, semakin baik perusahaan mengelola asetnya untuk menghasilkan keuntungan.",
		"DER":               "DER (Debt to Equity Ratio) adalah rasio solvabilitas yang membandingkan total utang dengan total ekuitas perusahaan. DER dihitung dengan: DER = Total Utang / Total Ekuitas. Rasio ini menunjukkan seberapa besar perusahaan dibiayai oleh utang dibandingkan modal sendiri. DER di atas 1 berarti perusahaan lebih banyak menggunakan utang. Investor umumnya menghindari perusahaan dengan DER terlalu tinggi (>2) karena risiko finansial yang besar.",
		"EPS":               "EPS (Earnings per Share) adalah laba bersih perusahaan yang dialokasikan untuk setiap lembar saham beredar. EPS dihitung dengan: EPS = Laba Bersih / Jumlah Saham Beredar. EPS adalah indikator utama profitabilitas perusahaan. Pertumbuhan EPS dari tahun ke tahun menunjukkan perusahaan sedang berkembang. Investor sering menggunakan EPS untuk menghitung PER dan menilai kewajaran harga saham.",
		"BVPS":              "BVPS (Book Value per Share) adalah nilai buku perusahaan per lembar saham. BVPS dihitung dengan: BVPS = Total Ekuitas / Jumlah Saham Beredar. BVPS menunjukkan nilai aset bersih yang dimiliki pemegang saham per lembarnya. Jika harga saham di bawah BVPS, saham tersebut mungkin undervalued.",
		"NPM":               "NPM (Net Profit Margin) adalah rasio yang mengukur persentase laba bersih dari total pendapatan. NPM = Laba Bersih / Pendapatan x 100%. Margin yang tinggi menunjukkan perusahaan memiliki pricing power dan efisiensi operasional yang baik. NPM sangat bervariasi antar industri - bandingkan hanya dengan perusahaan sejenis.",
		"Dividend Yield":    "Dividend Yield adalah rasio yang menunjukkan berapa persen dividen yang dibayarkan perusahaan relatif terhadap harga sahamnya. Dihitung dengan: Dividend Yield = Dividen per Saham / Harga Saham x 100%. Yield tinggi (>5%) menarik bagi investor yang mencari pendapatan pasif, namun perlu dipastikan dividen tersebut berkelanjutan.",
		"Market Cap":        "Market Cap (Kapitalisasi Pasar) adalah total nilai pasar dari seluruh saham beredar perusahaan. Dihitung dengan: Market Cap = Harga Saham x Jumlah Saham Beredar. Di Indonesia: Big Cap (>Rp 40T), Mid Cap (Rp 5T-40T), Small Cap (<Rp 5T). Kapitalisasi pasar mencerminkan ukuran dan skala perusahaan.",
		"IHSG":              "IHSG (Indeks Harga Saham Gabungan) adalah indeks yang mengukur kinerja seluruh saham yang tercatat di Bursa Efek Indonesia. IHSG menjadi barometer utama pasar modal Indonesia. Kenaikan IHSG menunjukkan pasar sedang bullish, penurunan menunjukkan bearish. IHSG dihitung berdasarkan kapitalisasi pasar tertimbang.",
		"LQ45":              "LQ45 adalah indeks yang terdiri dari 45 saham paling likuid di Bursa Efek Indonesia. Saham LQ45 dipilih berdasarkan likuiditas perdagangan, kapitalisasi pasar, dan fundamental keuangan. Indeks ini direview setiap 6 bulan (Februari dan Agustus). LQ45 menjadi acuan investor institusi dalam membangun portofolio.",
		"IDX":               "IDX (Indonesia Stock Exchange) adalah Bursa Efek Indonesia, satu-satunya bursa saham di Indonesia hasil merger Bursa Efek Jakarta (BEJ) dan Bursa Efek Surabaya (BES) tahun 2007. IDX mengoperasikan pasar reguler, pasar negosiasi, dan pasar tunai. Semua transaksi saham di Indonesia dilakukan melalui IDX.",
		"Saham Blue Chip":   "Saham Blue Chip adalah saham dari perusahaan besar, mapan, dan bereputasi baik dengan kapitalisasi pasar besar, likuiditas tinggi, dan fundamental kuat. Contoh di Indonesia: BBCA, TLKM, ASII, UNVR. Blue chip cocok untuk investor jangka panjang yang mencari stabilitas dan dividen konsisten.",
		"Saham Second Liner": "Saham Second Liner adalah saham lapis kedua di bawah blue chip dengan kapitalisasi pasar menengah (mid cap). Saham ini menawarkan potensi pertumbuhan lebih tinggi dibanding blue chip namun dengan risiko lebih besar. Likuiditas lebih rendah dari blue chip namun masih relatif baik.",
		"Saham Gorengan":    "Saham Gorengan adalah istilah untuk saham yang harganya dimanipulasi oleh bandar/pemain besar melalui aksi goreng-menggoreng. Ciri-ciri: volume tidak wajar, fluktuasi harga ekstrem, fundamental buruk, dan market cap kecil. Investor pemula wajib menghindari saham gorengan karena risiko kerugian sangat tinggi.",
		"Auto Rejection":    "Auto Rejection (AR) adalah mekanisme penolakan otomatis oleh sistem IDX ketika harga saham naik atau turun melebihi batas yang ditentukan. Batas AR: 35% untuk saham <Rp 200, 25% untuk Rp 200-5000, 20% untuk >Rp 5000. AR melindungi investor dari fluktuasi harga ekstrem.",
		"Lot":               "Lot adalah satuan standar pembelian saham di Indonesia. 1 lot = 100 lembar saham. Investor tidak dapat membeli kurang dari 1 lot. Satuan lot memudahkan standardisasi transaksi dan perhitungan biaya broker.",
		"Capital Gain":      "Capital Gain adalah keuntungan yang diperoleh dari selisih harga jual dan harga beli saham. Capital gain = Harga Jual - Harga Beli. Keuntungan ini dikenakan pajak final 0.1% dari nilai transaksi penjualan. Capital gain adalah salah satu sumber utama return investasi saham.",
		"Dividen":           "Dividen adalah bagian laba perusahaan yang dibagikan kepada pemegang saham. Dividen bisa berupa tunai atau saham bonus. Perusahaan biasanya membagikan dividen 1-2 kali setahun. Kebijakan dividen ditentukan dalam RUPS. Dividen tunai dikenakan pajak 10% untuk WP dalam negeri.",
		"Cum Date":          "Cum Date adalah tanggal terakhir investor dapat membeli saham dan masih berhak menerima dividen. Setelah cum date, saham diperdagangkan 'ex dividend'. Investor yang membeli saat cum date berhak atas dividen yang akan datang. Tanggal ini penting bagi dividend investor.",
		"Ex Date":           "Ex Date adalah tanggal mulai saham diperdagangkan tanpa hak dividen. Investor yang membeli saham pada atau setelah ex date TIDAK berhak atas dividen periode tersebut. Harga saham biasanya turun mendekati nilai dividen pada ex date.",
		"Right Issue":       "Right Issue adalah aksi korporasi di mana perusahaan menerbitkan saham baru yang ditawarkan terlebih dahulu kepada pemegang saham existing. HMETD (Hak Memesan Efek Terlebih Dahulu) memberikan hak proporsional. Right issue bisa dilaksanakan (exercise) atau dijual di pasar.",
		"Stock Split":       "Stock Split adalah pemecahan nilai nominal saham menjadi lebih kecil sehingga jumlah saham beredar bertambah. Contoh: split 1:2 mengubah 1 saham harga Rp 1000 menjadi 2 saham harga Rp 500. Total nilai investasi tetap sama. Tujuan: meningkatkan likuiditas dan keterjangkauan saham.",
		"Reverse Split":     "Reverse Split adalah penggabungan nilai nominal saham sehingga jumlah saham beredar berkurang. Contoh: reverse split 10:1 mengubah 10 saham harga Rp 50 menjadi 1 saham harga Rp 500. Dilakukan untuk memenuhi ketentuan harga minimum IDX (Rp 50) dan memperbaiki citra saham.",
		"Tender Offer":      "Tender Offer adalah penawaran untuk membeli saham perusahaan publik oleh pihak tertentu dengan harga yang telah ditentukan. Biasanya dilakukan dalam rangka akuisisi atau go private. Pemegang saham dapat menerima atau menolak tawaran tersebut.",
		"Warrant":           "Warrant adalah efek yang memberikan hak kepada pemegangnya untuk membeli saham perusahaan pada harga dan waktu tertentu. Warrant diterbitkan bersamaan dengan right issue atau obligasi konversi. Warrant bisa diperdagangkan di bursa dan memiliki jangka waktu tertentu.",
		"Obligasi":          "Obligasi adalah surat utang yang diterbitkan pemerintah atau perusahaan dengan janji membayar bunga (kupon) secara periodik dan pokok saat jatuh tempo. Obligasi lebih konservatif dari saham, memberikan pendapatan tetap. Di Indonesia dikenal ORI, SUN, dan obligasi korporasi.",
		"Reksadana":         "Reksadana adalah wadah investasi kolektif di mana dana investor dikelola oleh Manajer Investasi dalam portofolio efek. Jenis: pasar uang, pendapatan tetap, campuran, dan saham. Cocok untuk investor pemula karena diversifikasi instan dan dikelola profesional.",
		"ETF":               "ETF (Exchange Traded Fund) adalah reksadana yang unit penyertaannya diperdagangkan di bursa seperti saham. ETF menggabungkan diversifikasi reksadana dengan fleksibilitas trading saham. Di Indonesia terdapat ETF indeks LQ45, IDX30, dan SRI-KEHATI.",
		"IPO":               "IPO (Initial Public Offering) adalah penawaran saham perdana perusahaan kepada publik. Sebelum IPO, perusahaan adalah perusahaan tertutup. Setelah IPO, sahamnya tercatat di bursa dan dapat diperdagangkan investor publik. IPO adalah cara perusahaan mendapatkan pendanaan dari pasar modal.",
		"Delisting":         "Delisting adalah penghapusan pencatatan saham perusahaan dari bursa. Bisa voluntary (sukarela) atau forced (paksa oleh bursa). Penyebab forced delisting: tidak memenuhi syarat pencatatan, suspensi berkepanjangan, atau bangkrut. Investor harus berhati-hati terhadap saham berpotensi delisting.",
		"Suspensi":          "Suspensi adalah penghentian sementara perdagangan saham oleh bursa. Bisa karena: akumulasi kenaikan/penurunan harga berlebih, adanya informasi material yang perlu diumumkan, atau pelanggaran aturan. Suspensi melindungi investor dari informasi asimetris.",
		"Fraksi Harga":      "Fraksi Harga adalah ketentuan perubahan harga minimum (tick size) berdasarkan rentang harga saham di IDX. Contoh: saham <Rp 200 fraksi Rp 1, Rp 200-500 fraksi Rp 2, Rp 500-2000 fraksi Rp 5, >Rp 5000 fraksi Rp 25. Fraksi membatasi pergerakan harga minimum.",
		"Bid":               "Bid adalah harga permintaan beli tertinggi yang diajukan pembeli di pasar. Dalam order book, bid price menunjukkan minat beli. Spread antara bid dan offer/ask mencerminkan likuiditas saham.",
		"Ask/Offer":         "Ask/Offer adalah harga penawaran jual terendah yang diajukan penjual di pasar. Dalam order book, ask/offer price menunjukkan minat jual. Selisih antara bid dan ask disebut spread - semakin kecil spread semakin likuid saham.",
		"Last Price":        "Last Price adalah harga transaksi terakhir yang terjadi di pasar. Ini adalah harga yang paling sering ditampilkan sebagai harga saham saat ini. Last price menjadi acuan perhitungan gain/loss dan auto rejection.",
		"Open":              "Open adalah harga pembukaan/harga transaksi pertama pada suatu hari perdagangan. Open price bisa berbeda dari close kemarin karena order yang terakumulasi sebelum pasar buka (pre-opening session pukul 08:45-09:00).",
		"High":              "High adalah harga tertinggi yang terjadi dalam satu hari perdagangan. High menunjukkan level resistensi harian di mana tekanan jual mulai dominan.",
		"Low":               "Low adalah harga terendah yang terjadi dalam satu hari perdagangan. Low menunjukkan level support harian di mana tekanan beli mulai dominan.",
		"Close":             "Close adalah harga penutupan/transaksi terakhir pada akhir hari perdagangan. Close price adalah harga paling penting karena menjadi acuan analisis teknikal, perhitungan indeks, dan pembukaan hari berikutnya.",
		"Volume":            "Volume adalah jumlah saham yang diperdagangkan dalam periode tertentu. Volume tinggi mengindikasikan minat pasar yang besar. Dalam analisis teknikal: volume mengkonfirmasi tren - tren naik dengan volume tinggi lebih valid. Volume juga menunjukkan likuiditas saham.",
		"Frequency":         "Frequency adalah jumlah transaksi yang terjadi dalam periode tertentu. Berbeda dengan volume yang menghitung jumlah saham, frequency menghitung jumlah kali transaksi. Frequency mencerminkan aktivitas trading dan partisipasi investor ritel.",
		"Foreign Buy":       "Foreign Buy adalah total pembelian saham oleh investor asing dalam satu hari perdagangan. Selisih Foreign Buy dan Foreign Sell menghasilkan Net Foreign Flow. Investor asing signifikan di pasar Indonesia (~30-40% transaksi).",
		"Foreign Sell":      "Foreign Sell adalah total penjualan saham oleh investor asing dalam satu hari perdagangan. Foreign sell yang besar bisa menyebabkan tekanan jual dan penurunan IHSG. Investor lokal sering memonitor foreign flow sebagai indikator sentimen.",
		"Candlestick":       "Candlestick adalah metode representasi pergerakan harga yang berasal dari Jepang. Setiap candle menunjukkan Open, High, Low, Close dalam satu periode. Body candle (area antara open dan close) berwarna hijau (bullish/naik) atau merah (bearish/turun). Shadow/wick menunjukkan high dan low.",
		"Doji":              "Doji adalah pola candlestick di mana harga open dan close hampir sama, menghasilkan body yang sangat kecil atau tidak ada. Doji menandakan keraguan pasar (indecision). Doji di puncak tren naik bisa menjadi sinyal reversal bearish. Jenis: Doji biasa, Long-legged Doji, Dragonfly Doji, Gravestone Doji.",
		"Hammer":            "Hammer adalah pola candlestick bullish reversal dengan body kecil di atas dan shadow bawah panjang (minimal 2x body). Muncul di akhir downtrend, menandakan penolakan harga rendah. Konfirmasi: candle bullish di hari berikutnya. Inverted Hammer: versi terbalik dengan shadow atas panjang.",
		"Shooting Star":     "Shooting Star adalah pola candlestick bearish reversal dengan body kecil di bawah dan shadow atas panjang. Muncul di akhir uptrend, menandakan penolakan harga tinggi. Konfirmasi: candle bearish di hari berikutnya. Sering menjadi sinyal jual bagi trader teknikal.",
		"Engulfing":         "Engulfing adalah pola candlestick dua candle di mana candle kedua menelan (engulf) candle pertama. Bullish Engulfing: candle hijau besar menelan candle merah kecil - sinyal reversal naik. Bearish Engulfing: candle merah besar menelan candle hijau kecil - sinyal reversal turun.",
		"Support":           "Support adalah level harga di mana tekanan beli cukup kuat untuk menghentikan penurunan harga. Support bisa berupa: level harga psikologis, moving average, trendline, atau level Fibonacci. Jika support ditembus (breakdown), support bisa berubah menjadi resistance.",
		"Resistance":        "Resistance adalah level harga di mana tekanan jual cukup kuat untuk menghentikan kenaikan harga. Resistance ditentukan oleh: puncak historis, level psikologis, moving average, atau trendline. Jika ditembus (breakout), resistance berubah menjadi support.",
		"Breakout":          "Breakout adalah penembusan level resistance (bullish) atau support (bearish) yang signifikan. Breakout yang valid dikonfirmasi oleh volume tinggi dan candle close di luar level. False breakout (fakeout) terjadi jika harga kembali ke dalam range. Entry sinyal: beli saat breakout resistance, jual saat breakdown support.",
		"Moving Average":    "Moving Average (MA) adalah rata-rata harga dalam periode tertentu yang bergerak mengikuti pergerakan harga. Jenis: SMA (Simple MA), EMA (Exponential MA - lebih sensitif), WMA (Weighted MA). Fungsi: identifikasi tren, support/resistance dinamis, sinyal crossover. Periode populer: MA20 (short), MA50 (medium), MA200 (long).",
		"Golden Cross":      "Golden Cross adalah sinyal bullish yang terjadi saat moving average jangka pendek (MA50) memotong MA jangka panjang (MA200) dari bawah ke atas. Pola ini mengindikasikan awal tren naik jangka panjang. Golden cross adalah salah satu sinyal teknikal paling kuat dan banyak diikuti.",
		"Death Cross":       "Death Cross adalah sinyal bearish yang terjadi saat moving average jangka pendek (MA50) memotong MA jangka panjang (MA200) dari atas ke bawah. Pola ini mengindikasikan awal tren turun jangka panjang. Death cross sering menjadi sinyal exit bagi investor jangka panjang.",
		"MACD":              "MACD (Moving Average Convergence Divergence) adalah indikator momentum yang menunjukkan hubungan antara dua moving average. Komponen: MACD line (selisih EMA12-EMA26), Signal line (EMA9 MACD), Histogram (MACD - Signal). Sinyal: crossover MACD di atas signal = bullish, MACD di bawah signal = bearish.",
		"RSI":               "RSI (Relative Strength Index) adalah osilator momentum yang mengukur kecepatan dan perubahan pergerakan harga, berkisar 0-100. RSI >70 dianggap overbought (jenuh beli, potensi koreksi), RSI <30 oversold (jenuh jual, potensi rebound). Periode standar: 14. Divergence RSI-harga juga menjadi sinyal reversal.",
		"Bollinger Bands":   "Bollinger Bands adalah indikator volatilitas terdiri dari 3 garis: middle band (MA20), upper band (MA20+2std), lower band (MA20-2std). Band melebar saat volatilitas tinggi, menyempit saat rendah. Squeeze (band menyempit) sering mendahului pergerakan besar. Harga cenderung kembali ke middle band.",
		"Stochastic":        "Stochastic Oscillator adalah indikator momentum yang membandingkan harga penutupan dengan range harga periode tertentu, berkisar 0-100. Slow Stochastic lebih populer (garis %K dan %D). >80 = overbought, <20 = oversold. Sinyal: crossover %K dan %D di area oversold = beli, overbought = jual.",
		"Fibonacci Retracement": "Fibonacci Retracement adalah tools analisis teknikal berdasarkan deret Fibonacci untuk mengidentifikasi level support/resistance potensial. Level kunci: 23.6%, 38.2%, 50%, 61.8%, 78.6%. Level 61.8% (golden ratio) paling penting. Digunakan untuk entry point saat pullback dalam tren yang kuat.",
		"Stop Loss":         "Stop Loss adalah perintah jual otomatis pada harga tertentu untuk membatasi kerugian. Stop loss WAJIB bagi setiap trader. Rule of thumb: risk per trade maksimal 1-2% dari modal. Penempatan: di bawah support terdekat atau di bawah MA20/MA50. Jangan memindah stop loss menjauhi entry (cut loss tepat waktu).",
		"Cut Loss":          "Cut Loss adalah aksi menjual saham saat rugi untuk mencegah kerugian lebih besar. Berbeda dengan stop loss (otomatis), cut loss adalah keputusan manual. Disiplin cut loss adalah pembeda trader sukses dan gagal. Psikologi: cut loss adalah skill, bukan kegagalan.",
		"Averaging Down":    "Averaging Down adalah strategi membeli saham yang sama saat harga turun untuk menurunkan harga rata-rata pembelian. Risiko: menambah eksposur pada saham yang sedang turun. Hanya cocok untuk saham fundamental bagus yang turun karena sentimen sementara. Tidak direkomendasikan untuk saham gorengan.",
		"Dollar Cost Averaging": "Dollar Cost Averaging adalah strategi investasi rutin dengan jumlah tetap secara periodik tanpa memperhatikan harga. Membeli lebih banyak unit saat harga rendah, lebih sedikit saat tinggi. Strategi ini menghilangkan kebutuhan market timing dan cocok untuk investor pemula jangka panjang.",
		"Growth Investing":  "Growth Investing adalah strategi investasi fokus pada perusahaan dengan pertumbuhan laba di atas rata-rata. Karakteristik: PER tinggi, laba tumbuh 20%+ year-on-year, sering di sektor teknologi/konsumen. Risiko: valuasi premium, sensitif terhadap perlambatan pertumbuhan. Fokus pada potensi masa depan, bukan valuasi saat ini.",
		"Value Investing":   "Value Investing adalah strategi investasi mencari saham yang diperdagangkan di bawah nilai intrinsiknya. Dipopulerkan Benjamin Graham dan Warren Buffett. Metode: analisis fundamental mendalam, mencari margin of safety. Karakteristik: PER rendah, PBV rendah, dividend yield tinggi. Sabar menunggu market menghargai nilai sebenarnya.",
		"Dividen Investing": "Dividen Investing adalah strategi investasi fokus pada saham yang membayar dividen tinggi dan stabil. Investor membangun passive income stream dari dividen. Kriteria: dividend yield >4%, payout ratio <70% (berkelanjutan), dividend growth history. Sektor defensif seperti consumer goods dan perbankan banyak diminati.",
		"Trading Harian":    "Trading Harian (Day Trading) adalah strategi trading di mana posisi dibuka dan ditutup pada hari yang sama. Tidak ada posisi menginap (overnight). Butuh: modal besar, waktu penuh, disiplin ketat, biaya komisi rendah. Risiko tinggi - mayoritas day trader pemula rugi.",
		"Swing Trading":     "Swing Trading adalah strategi trading jangka pendek-menengah (beberapa hari hingga minggu) menangkap pergerakan harga (swing). Analisis: teknikal dominan, support/resistance, pola candlestick. Cocok untuk yang tidak bisa monitor full-time tapi tidak sabar untuk investasi jangka panjang.",
		"Position Trading":  "Position Trading adalah strategi trading jangka panjang (bulan hingga tahun) berdasarkan tren mayor. Analisis: fundamental dominan, teknikal untuk timing entry/exit. Tidak terpengaruh fluktuasi harian. Mengikuti tren besar: beli saat awal bull market, jual saat bear market mulai.",
		"Bandarmologi":      "Bandarmologi adalah istilah populer di kalangan trader Indonesia yang berarti ilmu mempelajari pergerakan bandar/pemain besar. Teknik: analisis bid-ask queue, volume tidak wajar, akumulasi/distribusi. Kontroversial karena sering dikaitkan dengan manipulasi pasar. Trader institusional tidak menggunakan istilah ini.",
		"Akumulasi":         "Akumulasi adalah fase di mana investor besar (bandar/institusi) secara diam-diam membeli saham dalam jumlah besar. Ciri: harga sideways lama, volume meningkat bertahap, harga tutup cenderung di atas. Setelah akumulasi selesai biasanya terjadi markup (kenaikan harga signifikan).",
		"Distribusi":        "Distribusi adalah fase di mana pemegang saham besar melepas kepemilikan ke pasar (retail). Ciri: harga sideways setelah uptrend panjang, volume tinggi, harga gagal naik. Distribusi adalah opposite dari akumulasi - setelah distribusi biasanya terjadi markdown (penurunan harga).",
		"Modal":             "Modal adalah dana yang disiapkan investor untuk membeli saham. Modal minimal untuk mulai investasi saham di Indonesia sekitar Rp 100.000 (untuk beli 1 lot saham murah). Ideal: mulai dengan modal yang siap hilang (risk capital), tidak menggunakan uang kebutuhan sehari-hari atau uang pinjaman.",
		"Portofolio":        "Portofolio adalah kumpulan investasi yang dimiliki investor. Tujuan portofolio: diversifikasi risiko. Portofolio ideal: 5-10 saham dari sektor berbeda. Alokasi: tentukan persentase per saham untuk manajemen risiko. Monitoring: review portofolio berkala, rebalancing jika diperlukan.",
		"Diversifikasi":     "Diversifikasi adalah strategi menyebar investasi di berbagai saham/sektor untuk mengurangi risiko. Pepatah: jangan menaruh semua telur dalam satu keranjang. Diversifikasi sektoral: jangan semua saham di satu sektor. Ideal: 5-15 saham dari minimal 3 sektor berbeda.",
		"Risk Management":   "Risk Management (Manajemen Risiko) adalah strategi mengelola dan membatasi potensi kerugian investasi. Komponen: stop loss, position sizing (maksimal 10-20% modal per saham), diversifikasi, risk/reward ratio minimal 1:2. Manajemen risiko adalah skill TERPENTING dalam trading.",
		"Bullish":           "Bullish adalah kondisi pasar atau sentimen yang optimis, mengharapkan harga naik. Pasar bullish ditandai tren naik berkepanjangan. Istilah berasal dari cara banteng menyerang (tanduk ke atas). IHSG bullish = indeks naik signifikan.",
		"Bearish":           "Bearish adalah kondisi pasar atau sentimen yang pesimis, mengharapkan harga turun. Pasar bearish ditandai tren turun berkepanjangan. Istilah berasal dari cara beruang menyerang (cakar ke bawah). Pasar bearish = peluang akumulasi untuk investor jangka panjang.",
		"Sideways":          "Sideways adalah kondisi pasar di mana harga bergerak dalam range terbatas tanpa tren jelas (konsolidasi). Fase sideways sering terjadi setelah tren naik atau turun besar. Strategi: beli di support range, jual di resistance range, atau tunggu breakout untuk konfirmasi arah selanjutnya.",
		"Volatilitas":       "Volatilitas adalah ukuran fluktuasi harga dalam periode tertentu. Volatilitas tinggi = harga naik-turun tajam (high risk, high reward). Volatilitas rendah = harga relatif stabil. Beta mengukur volatilitas relatif terhadap IHSG. Saham volatile cocok untuk trading, tidak cocok untuk investor konservatif.",
		"Net Buy":           "Net Buy adalah selisih positif antara pembelian dan penjualan dalam periode tertentu. Net Buy asing artinya investor asing lebih banyak beli daripada jual - umumnya sentimen positif. Net Buy lokal artinya investor domestik net pembeli. Net buy mendorong kenaikan harga.",
		"Net Sell":          "Net Sell adalah selisih negatif antara pembelian dan penjualan dalam periode tertentu. Net Sell asing bisa menyebabkan penurunan IHSG. Perlu dilihat konteks: net sell kecil mungkin hanya profit taking, net sell besar bisa indikasi capital outflow.",
	}

	categories := map[string]string{
		"PER": "fundamental", "PBV": "fundamental", "ROE": "fundamental", "ROA": "fundamental",
		"DER": "fundamental", "EPS": "fundamental", "BVPS": "fundamental", "NPM": "fundamental",
		"Dividend Yield": "fundamental", "Market Cap": "fundamental",
		"IHSG": "pasar", "LQ45": "pasar", "IDX": "pasar",
		"Saham Blue Chip": "saham", "Saham Second Liner": "saham", "Saham Gorengan": "saham",
		"Auto Rejection": "trading", "Lot": "trading", "Capital Gain": "trading", "Dividen": "trading",
		"Cum Date": "korporasi", "Ex Date": "korporasi", "Right Issue": "korporasi",
		"Stock Split": "korporasi", "Reverse Split": "korporasi", "Tender Offer": "korporasi",
		"Warrant": "instrumen", "Obligasi": "instrumen", "Reksadana": "instrumen", "ETF": "instrumen",
		"IPO": "korporasi", "Delisting": "korporasi", "Suspensi": "korporasi",
		"Fraksi Harga": "trading", "Bid": "trading", "Ask/Offer": "trading",
		"Last Price": "trading", "Open": "trading", "High": "trading", "Low": "trading", "Close": "trading",
		"Volume": "trading", "Frequency": "trading", "Foreign Buy": "trading", "Foreign Sell": "trading",
		"Candlestick": "teknikal", "Doji": "teknikal", "Hammer": "teknikal",
		"Shooting Star": "teknikal", "Engulfing": "teknikal",
		"Support": "teknikal", "Resistance": "teknikal", "Breakout": "teknikal",
		"Moving Average": "teknikal", "Golden Cross": "teknikal", "Death Cross": "teknikal",
		"MACD": "teknikal", "RSI": "teknikal", "Bollinger Bands": "teknikal",
		"Stochastic": "teknikal", "Fibonacci Retracement": "teknikal",
		"Stop Loss": "psikologi", "Cut Loss": "psikologi", "Averaging Down": "strategi",
		"Dollar Cost Averaging": "strategi", "Growth Investing": "strategi",
		"Value Investing": "strategi", "Dividen Investing": "strategi",
		"Trading Harian": "strategi", "Swing Trading": "strategi", "Position Trading": "strategi",
		"Bandarmologi": "psikologi", "Akumulasi": "trading", "Distribusi": "trading",
		"Modal": "dasar", "Portofolio": "dasar", "Diversifikasi": "dasar",
		"Risk Management": "dasar", "Bullish": "dasar", "Bearish": "dasar",
		"Sideways": "dasar", "Volatilitas": "dasar", "Net Buy": "trading", "Net Sell": "trading",
	}

	var result []GlossaryTerm
	for term, def := range terms {
		slug := slugify(term)
		cat := categories[term]
		if cat == "" {
			cat = "umum"
		}
		result = append(result, GlossaryTerm{
			Term:       term,
			Slug:       slug,
			Definition: def,
			Category:   cat,
		})
	}
	return result
}

func (s *Service) GenerateGlossaryRoutes() []PSEORoute {
	terms := GetAllGlossaryTerms()
	var routes []PSEORoute
	for _, t := range terms {
		routes = append(routes, PSEORoute{
			Pattern:     "/istilah/" + t.Slug,
			Handler:     "GlossaryPage",
			Title:       "Apa itu " + t.Term + "? Definisi & Penjelasan Lengkap",
			Description: "Pengertian " + t.Term + " dalam investasi saham. Penjelasan lengkap, rumus, contoh penggunaan, dan tips untuk investor pemula Indonesia.",
			Category:    t.Category,
		})
	}
	return routes
}

func GetIndonesianCities() []CityPage {
	cities := []struct {
		Name     string
		Province string
	}{
		{"Jakarta", "DKI Jakarta"},
		{"Surabaya", "Jawa Timur"},
		{"Bandung", "Jawa Barat"},
		{"Medan", "Sumatera Utara"},
		{"Semarang", "Jawa Tengah"},
		{"Makassar", "Sulawesi Selatan"},
		{"Palembang", "Sumatera Selatan"},
		{"Tangerang", "Banten"},
		{"Bekasi", "Jawa Barat"},
		{"Depok", "Jawa Barat"},
		{"Bogor", "Jawa Barat"},
		{"Yogyakarta", "DI Yogyakarta"},
		{"Malang", "Jawa Timur"},
		{"Solo", "Jawa Tengah"},
		{"Denpasar", "Bali"},
		{"Batam", "Kepulauan Riau"},
		{"Balikpapan", "Kalimantan Timur"},
		{"Manado", "Sulawesi Utara"},
		{"Padang", "Sumatera Barat"},
		{"Pontianak", "Kalimantan Barat"},
		{"Banjarmasin", "Kalimantan Selatan"},
		{"Pekanbaru", "Riau"},
		{"Samarinda", "Kalimantan Timur"},
		{"Tasikmalaya", "Jawa Barat"},
		{"Cirebon", "Jawa Barat"},
		{"Cimahi", "Jawa Barat"},
		{"Sukabumi", "Jawa Barat"},
		{"Purwokerto", "Jawa Tengah"},
		{"Kediri", "Jawa Timur"},
		{"Jember", "Jawa Timur"},
		{"Madiun", "Jawa Timur"},
		{"Probolinggo", "Jawa Timur"},
		{"Pasuruan", "Jawa Timur"},
		{"Mojokerto", "Jawa Timur"},
		{"Magelang", "Jawa Tengah"},
		{"Salatiga", "Jawa Tengah"},
		{"Tegal", "Jawa Tengah"},
		{"Pekalongan", "Jawa Tengah"},
		{"Cilacap", "Jawa Tengah"},
		{"Banyuwangi", "Jawa Timur"},
		{"Mataram", "Nusa Tenggara Barat"},
		{"Kupang", "Nusa Tenggara Timur"},
		{"Ambon", "Maluku"},
		{"Jayapura", "Papua"},
		{"Sorong", "Papua Barat"},
		{"Manokwari", "Papua Barat"},
		{"Palu", "Sulawesi Tengah"},
		{"Kendari", "Sulawesi Tenggara"},
		{"Gorontalo", "Gorontalo"},
		{"Mamuju", "Sulawesi Barat"},
		{"Palangkaraya", "Kalimantan Tengah"},
		{"Tanjungpinang", "Kepulauan Riau"},
		{"Pangkalpinang", "Kepulauan Bangka Belitung"},
		{"Bengkulu", "Bengkulu"},
		{"Jambi", "Jambi"},
		{"Lampung", "Lampung"},
		{"Serang", "Banten"},
		{"Cilegon", "Banten"},
		{"Ternate", "Maluku Utara"},
		{"Tarakan", "Kalimantan Utara"},
		{"Banda Aceh", "Aceh"},
		{"Lhokseumawe", "Aceh"},
		{"Langsa", "Aceh"},
		{"Pematangsiantar", "Sumatera Utara"},
		{"Binjai", "Sumatera Utara"},
		{"Dumai", "Riau"},
		{"Lubuklinggau", "Sumatera Selatan"},
		{"Prabumulih", "Sumatera Selatan"},
		{"Singkawang", "Kalimantan Barat"},
		{"Bontang", "Kalimantan Timur"},
		{"Bitung", "Sulawesi Utara"},
		{"Tomohon", "Sulawesi Utara"},
		{"Bau-Bau", "Sulawesi Tenggara"},
		{"Parepare", "Sulawesi Selatan"},
		{"Palopo", "Sulawesi Selatan"},
		{"Bima", "Nusa Tenggara Barat"},
		{"Sumbawa", "Nusa Tenggara Barat"},
		{"Ende", "Nusa Tenggara Timur"},
		{"Maumere", "Nusa Tenggara Timur"},
		{"Ruteng", "Nusa Tenggara Timur"},
		{"Waingapu", "Nusa Tenggara Timur"},
		{"Atambua", "Nusa Tenggara Timur"},
		{"Merauke", "Papua"},
		{"Timika", "Papua"},
		{"Biak", "Papua"},
		{"Nabire", "Papua"},
		{"Wamena", "Papua"},
		{"Bukittinggi", "Sumatera Barat"},
		{"Payakumbuh", "Sumatera Barat"},
		{"Solok", "Sumatera Barat"},
		{"Sawahlunto", "Sumatera Barat"},
		{"Pariaman", "Sumatera Barat"},
		{"Padangpanjang", "Sumatera Barat"},
		{"Tanjungbalai", "Sumatera Utara"},
		{"Tebingtinggi", "Sumatera Utara"},
		{"Sibolga", "Sumatera Utara"},
		{"Padangsidempuan", "Sumatera Utara"},
		{"Gunungsitoli", "Sumatera Utara"},
		{"Blitar", "Jawa Timur"},
		{"Batu", "Jawa Timur"},
		{"Banjar", "Jawa Barat"},
		{"Subang", "Jawa Barat"},
		{"Garut", "Jawa Barat"},
		{"Sumedang", "Jawa Barat"},
		{"Indramayu", "Jawa Barat"},
		{"Karawang", "Jawa Barat"},
		{"Purwakarta", "Jawa Barat"},
		{"Cianjur", "Jawa Barat"},
		{"Kuningan", "Jawa Barat"},
		{"Majalengka", "Jawa Barat"},
	}

	var result []CityPage
	for _, c := range cities {
		result = append(result, CityPage{
			City:     c.Name,
			Slug:     slugify(c.Name),
			Province: c.Province,
		})
	}
	return result
}

func (s *Service) GenerateCityRoutes() []PSEORoute {
	cities := GetIndonesianCities()
	var routes []PSEORoute
	for _, c := range cities {
		routes = append(routes, PSEORoute{
			Pattern:     "/belajar-saham-" + c.Slug,
			Handler:     "CityPage",
			Title:       "Belajar Saham di " + c.City + " - Panduan Lengkap Investasi",
			Description: fmt.Sprintf("Panduan belajar investasi saham untuk pemula di %s. Temukan komunitas, sekuritas, dan mentor saham terdekat.", c.City),
			Category:    "city",
		})
		routes = append(routes, PSEORoute{
			Pattern:     "/komunitas-saham-" + c.Slug,
			Handler:     "CityCommunityPage",
			Title:       "Komunitas Saham " + c.City + " - Grup Diskusi & Belajar Investasi",
			Description: fmt.Sprintf("Gabung komunitas investor saham di %s. Temukan grup diskusi, workshop, dan sesi sharing pengalaman investasi.", c.City),
			Category:    "city-community",
		})
		routes = append(routes, PSEORoute{
			Pattern:     "/aplikasi-saham-" + c.Slug,
			Handler:     "CityAppPage",
			Title:       "Aplikasi Saham Terbaik untuk Investor di " + c.City,
			Description: fmt.Sprintf("Rekomendasi aplikasi trading saham terbaik untuk investor di %s. Bandingkan fitur, biaya, dan kemudahan penggunaan.", c.City),
			Category:    "city-app",
		})
	}
	return routes
}

func (s *Service) GenerateHowToRoutes(stockCodes []string) []PSEORoute {
	var routes []PSEORoute

	routes = append(routes, PSEORoute{
		Pattern:     "/cara-mulai-investasi-saham",
		Handler:     "HowToPage",
		Title:       "Cara Mulai Investasi Saham untuk Pemula - Panduan Lengkap 2026",
		Description: "Panduan lengkap cara mulai investasi saham di Indonesia dari nol. Buka rekening saham, pilih sekuritas, beli saham pertama.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-baca-laporan-keuangan",
		Handler:     "HowToPage",
		Title:       "Cara Membaca Laporan Keuangan Perusahaan untuk Investor Saham",
		Description: "Pelajari cara membaca laporan keuangan: income statement, balance sheet, cash flow. Analisis fundamental untuk pemula.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-analisa-teknikal",
		Handler:     "HowToPage",
		Title:       "Cara Analisa Teknikal Saham - Chart, Indikator, Support Resistance",
		Description: "Belajar analisa teknikal saham dari dasar. Candlestick, support resistance, indikator MACD RSI, trendline, dan pola chart.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-pilih-saham",
		Handler:     "HowToPage",
		Title:       "Cara Memilih Saham yang Bagus untuk Investasi - Panduan Lengkap",
		Description: "Panduan memilih saham berdasarkan fundamental, valuasi, dan prospek industri. Screening saham untuk pemula dan expert.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-buka-rekening-saham",
		Handler:     "HowToPage",
		Title:       "Cara Buka Rekening Saham Online - Syarat, Biaya, dan Prosedur",
		Description: "Langkah demi langkah membuka rekening saham di sekuritas Indonesia. Persyaratan KTP, NPWP, rekening bank, dan modal awal.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-hitung-capital-gain",
		Handler:     "HowToPage",
		Title:       "Cara Menghitung Capital Gain Saham - Rumus, Pajak, dan Contoh",
		Description: "Cara menghitung keuntungan jual beli saham (capital gain). Perhitungan pajak, fee broker, dan return investasi.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-klaim-dividen",
		Handler:     "HowToPage",
		Title:       "Cara Klaim Dividen Saham - Prosedur, Pajak, dan Jadwal Pembayaran",
		Description: "Pelajari cara mendapatkan dividen saham. Cum date, ex date, recording date, payment date, dan perhitungan pajak dividen.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-hindari-saham-gorengan",
		Handler:     "HowToPage",
		Title:       "Cara Menghindari Saham Gorengan - Ciri-ciri, Risiko, dan Tips",
		Description: "Kenali ciri-ciri saham gorengan dan cara menghindarinya. Lindungi portofolio dari manipulasi bandar dan kerugian besar.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-diversifikasi-portofolio",
		Handler:     "HowToPage",
		Title:       "Cara Diversifikasi Portofolio Saham - Strategi Minim Risiko",
		Description: "Strategi diversifikasi portofolio saham: sektor, kapitalisasi pasar, dan instrumen. Cara menyebar risiko investasi.",
		Category:    "howto",
	})
	routes = append(routes, PSEORoute{
		Pattern:     "/cara-pakai-screener",
		Handler:     "HowToPage",
		Title:       "Cara Pakai Stock Screener - Filter Saham Fundamental & Teknikal",
		Description: "Tutorial menggunakan stock screener untuk menemukan saham potensial. Filter PER, PBV, ROE, DER, dan dividend yield.",
		Category:    "howto",
	})

	for _, code := range stockCodes {
		slug := strings.ToLower(code)
		routes = append(routes, PSEORoute{
			Pattern:     "/cara-beli-saham-" + slug,
			Handler:     "HowToPage",
			Title:       "Cara Beli Saham " + strings.ToUpper(code) + " - Panduan Lengkap",
			Description: fmt.Sprintf("Cara membeli saham %s di Bursa Efek Indonesia. Panduan lengkap untuk investor pemula dan berpengalaman.", strings.ToUpper(code)),
			Category:    "howto-stock",
		})
		routes = append(routes, PSEORoute{
			Pattern:     "/cara-analisa-saham-" + slug,
			Handler:     "HowToPage",
			Title:       "Cara Analisa Saham " + strings.ToUpper(code) + " - Fundamental & Teknikal",
			Description: fmt.Sprintf("Panduan analisa fundamental dan teknikal saham %s. PER, PBV, ROE, chart pattern, support resistance.", strings.ToUpper(code)),
			Category:    "howto-stock",
		})
	}

	return routes
}

func slugify(term string) string {
	s := strings.ToLower(term)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return strings.Trim(result.String(), "-")
}

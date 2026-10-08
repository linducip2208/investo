package service

type MATarget struct {
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Sector            string  `json:"sector"`
	Rationale         string  `json:"rationale"`
	ProbabilityScore  float64 `json:"probability_score"`
	PER               float64 `json:"per"`
	PBV               float64 `json:"pbv"`
	InsiderOwnership  string  `json:"insider_ownership"`
	CashPosition      string  `json:"cash_position"`
	MarketCap         string  `json:"market_cap"`
}

type MAScreenerService struct{}

func NewMAScreenerService() *MAScreenerService {
	return &MAScreenerService{}
}

func (s *MAScreenerService) ScreenMATargets() ([]MATarget, error) {
	targets := []MATarget{
		{
			Code:              "INKP",
			Name:              "Indah Kiat Pulp & Paper",
			Sector:            "Pulp & Paper",
			Rationale:         "Valuasi di bawah replacement cost. Industri fragmented dengan banyak pemain kecil yang bisa diakuisisi. Cash flow kuat dari operasi ekspor. Sinergi dengan jaringan distribusi APP Group.",
			ProbabilityScore:  78,
			PER:               8.2,
			PBV:               0.9,
			InsiderOwnership:  "Rendah (25%)",
			CashPosition:      "Tinggi (Rp 12T)",
			MarketCap:         "Rp 45T",
		},
		{
			Code:              "SIDO",
			Name:              "Industri Jamu & Farmasi Sido Muncul",
			Sector:            "Consumer Health",
			Rationale:         "Brand kuat di jamu & herbal dengan distribusi nasional. PER di bawah rata-rata sektor consumer. Potensi akuisisi oleh multinational FMCG yang ingin ekspansi ke herbal Indonesia.",
			ProbabilityScore:  72,
			PER:               11.5,
			PBV:               1.2,
			InsiderOwnership:  "Tinggi (60%)",
			CashPosition:      "Tinggi (Rp 1.2T)",
			MarketCap:         "Rp 8T",
		},
		{
			Code:              "WSKT",
			Name:              "Waskita Karya (Persero)",
			Sector:            "Konstruksi BUMN",
			Rationale:         "Restrukturisasi utang hampir selesai. Merger BUMN Karya potensial (Waskita + Hutama Karya). Aset besar di jalan tol yang mulai beroperasi. Valuasi sangat murah pasca krisis.",
			ProbabilityScore:  65,
			PER:               6.8,
			PBV:               0.4,
			InsiderOwnership:  "Tinggi (Pemerintah 75%)",
			CashPosition:      "Rendah (restrukturisasi)",
			MarketCap:         "Rp 3T",
		},
		{
			Code:              "SMRA",
			Name:              "Summarecon Agung",
			Sector:            "Property",
			Rationale:         "Land bank premium di area strategis (Kelapa Gading, Serpong, Bandung). NAV jauh di atas market cap. Potensi akuisisi oleh developer asing (China/Japan). Konsolidasi sektor properti.",
			ProbabilityScore:  70,
			PER:               9.5,
			PBV:               0.7,
			InsiderOwnership:  "Sedang (45%)",
			CashPosition:      "Sedang (Rp 1.5T)",
			MarketCap:         "Rp 5T",
		},
		{
			Code:              "MAIN",
			Name:              "Malindo Feedmill",
			Sector:            "Poultry & Feed",
			Rationale:         "Industri unggas fragmented dengan margin rendah. Potensi akuisisi oleh pemain besar (CPIN, JPFA) untuk konsolidasi market share. Aset fisik (feedmill, farm) di bawah replacement cost.",
			ProbabilityScore:  60,
			PER:               10.2,
			PBV:               0.8,
			InsiderOwnership:  "Rendah (20%)",
			CashPosition:      "Rendah (high capex)",
			MarketCap:         "Rp 2T",
		},
	}
	return targets, nil
}

package service

type ProxyIndicator struct {
	Name             string  `json:"name"`
	CurrentValue     string  `json:"current_value"`
	Trend            string  `json:"trend"`
	TrendValue       float64 `json:"trend_value"`
	GDPCorrelation   float64 `json:"gdp_correlation"`
	Description      string  `json:"description"`
	Category         string  `json:"category"`
	LastUpdated      string  `json:"last_updated"`
}

type EconomicProxyService struct{}

func (s *EconomicProxyService) GetProxies() ([]ProxyIndicator, error) {
	return []ProxyIndicator{
		{
			Name:           "Container Traffic (Tanjung Priok)",
			CurrentValue:   "12.8M TEUs",
			Trend:          "up",
			TrendValue:     8.0,
			GDPCorrelation: 0.91,
			Description:    "Volume kontainer di pelabuhan terbesar Indonesia mencerminkan aktivitas perdagangan. Kenaikan 8% YoY mengindikasikan pertumbuhan ekonomi yang solid.",
			Category:       "Perdagangan",
			LastUpdated:    "Q2 2026",
		},
		{
			Name:           "Coal Production",
			CurrentValue:   "650M tons",
			Trend:          "up",
			TrendValue:     5.5,
			GDPCorrelation: 0.72,
			Description:    "Produksi batubara Indonesia mencapai rekor 650 juta ton. Proksi penting untuk sektor energi dan pertambangan, berkontribusi ~12% ekspor nasional.",
			Category:       "Energi",
			LastUpdated:    "H1 2026",
		},
		{
			Name:           "Cement Consumption",
			CurrentValue:   "62M tons",
			Trend:          "up",
			TrendValue:     5.0,
			GDPCorrelation: 0.88,
			Description:    "Konsumsi semen adalah proksi klasik untuk aktivitas konstruksi dan infrastruktur. Pertumbuhan 5% menunjukkan pembangunan properti dan infrastruktur yang berkelanjutan.",
			Category:       "Konstruksi",
			LastUpdated:    "Q1 2026",
		},
		{
			Name:           "Electricity Generation",
			CurrentValue:   "295 TWh",
			Trend:          "up",
			TrendValue:     3.0,
			GDPCorrelation: 0.94,
			Description:    "Pembangkit listrik merefleksikan aktivitas industri dan konsumsi rumah tangga. Korelasi tertinggi dengan GDP karena listrik adalah input universal untuk semua sektor ekonomi.",
			Category:       "Industri",
			LastUpdated:    "Q1 2026",
		},
		{
			Name:           "Vehicle Sales",
			CurrentValue:   "1.2M units",
			Trend:          "up",
			TrendValue:     4.2,
			GDPCorrelation: 0.76,
			Description:    "Penjualan kendaraan bermotor adalah indikator belanja konsumen kelas menengah. 1.2 juta unit mencerminkan daya beli yang masih kuat meskipun suku bunga tinggi.",
			Category:       "Konsumsi",
			LastUpdated:    "H1 2026",
		},
		{
			Name:           "Steel Consumption",
			CurrentValue:   "18M tons",
			Trend:          "up",
			TrendValue:     6.0,
			GDPCorrelation: 0.82,
			Description:    "Konsumsi baja domestik mencerminkan aktivitas manufaktur dan konstruksi. Pertumbuhan 6% didorong proyek infrastruktur IKN dan hilirisasi mineral.",
			Category:       "Manufaktur",
			LastUpdated:    "Q2 2026",
		},
		{
			Name:           "Air Passenger Traffic",
			CurrentValue:   "95M pax",
			Trend:          "up",
			TrendValue:     12.0,
			GDPCorrelation: 0.68,
			Description:    "Traffic penumpang pesawat memulih cepat pasca-pandemi. Indikator mobilitas bisnis dan pariwisata yang berkontribusi ~5% PDB dari sektor travel.",
			Category:       "Transportasi",
			LastUpdated:    "H1 2026",
		},
		{
			Name:           "Mobile Data Traffic",
			CurrentValue:   "18 EB/month",
			Trend:          "up",
			TrendValue:     22.0,
			GDPCorrelation: 0.58,
			Description:    "Traffic data seluler adalah proksi ekonomi digital. Pertumbuhan 22% didorong streaming, e-commerce, dan social commerce. Korelasi moderat dengan GDP formal.",
			Category:       "Digital",
			LastUpdated:    "Q1 2026",
		},
	}, nil
}

package service

type LeadingIndicator struct {
	Name          string  `json:"name"`
	Value         string  `json:"value"`
	Trend         string  `json:"trend"`
	GDPCorrelation float64 `json:"gdp_correlation"`
	Signal        string  `json:"signal"`
	Description   string  `json:"description"`
}

type GDPPrediction struct {
	CurrentGDP     float64 `json:"current_gdp"`
	PredictedGDP   float64 `json:"predicted_gdp"`
	Direction      string  `json:"direction"`
	Confidence     string  `json:"confidence"`
	Summary        string  `json:"summary"`
}

type LeadingIndicatorsService struct{}

func NewLeadingIndicatorsService() *LeadingIndicatorsService {
	return &LeadingIndicatorsService{}
}

func (s *LeadingIndicatorsService) GetLeadingIndicators() ([]LeadingIndicator, error) {
	indicators := []LeadingIndicator{
		{
			Name:           "Manufacturing PMI",
			Value:          "52.3",
			Trend:          "up",
			GDPCorrelation: 0.82,
			Signal:         "expansion",
			Description:    "PMI di atas 50 menandakan ekspansi manufaktur. Korelasi kuat dengan GDP karena manufaktur ~20% ekonomi Indonesia.",
		},
		{
			Name:           "Penjualan Semen",
			Value:          "+8% YoY",
			Trend:          "up",
			GDPCorrelation: 0.78,
			Signal:         "expansion",
			Description:    "Semen adalah proxy konstruksi & infrastruktur. Kenaikan 8% YoY menandakan percepatan pembangunan.",
		},
		{
			Name:           "Penjualan Kendaraan",
			Value:          "+5% YoY",
			Trend:          "up",
			GDPCorrelation: 0.71,
			Signal:         "expansion",
			Description:    "Penjualan kendaraan mencerminkan daya beli kelas menengah dan confidence konsumen.",
		},
		{
			Name:           "Konsumsi Listrik",
			Value:          "+3% YoY",
			Trend:          "up",
			GDPCorrelation: 0.85,
			Signal:         "expansion",
			Description:    "Konsumsi listrik adalah indikator paling real-time dari aktivitas ekonomi. Kenaikan 3% menandakan aktivitas industri dan rumah tangga meningkat.",
		},
		{
			Name:           "Consumer Confidence Index",
			Value:          "125.4",
			Trend:          "up",
			GDPCorrelation: 0.69,
			Signal:         "expansion",
			Description:    "Consumer confidence di atas 100 menandakan optimisme. Konsumsi rumah tangga adalah 55% dari GDP Indonesia.",
		},
		{
			Name:           "Retail Sales Index",
			Value:          "+6% YoY",
			Trend:          "up",
			GDPCorrelation: 0.74,
			Signal:         "expansion",
			Description:    "Penjualan ritel mencerminkan permintaan domestik. Kenaikan 6% menandakan konsumsi yang sehat.",
		},
	}
	return indicators, nil
}

func (s *LeadingIndicatorsService) PredictGDP() (*GDPPrediction, error) {
	return &GDPPrediction{
		CurrentGDP:   5.1,
		PredictedGDP: 5.3,
		Direction:    "up",
		Confidence:   "High (78%)",
		Summary:      "Leading indicators menunjukkan momentum positif. PMI di zona ekspansi, penjualan semen naik 8%, dan konsumsi listrik naik 3%. GDP diprediksi akselerasi ke 5.3% dalam 2 kuartal ke depan. Risiko utama: perlambatan global dan gejolak harga komoditas.",
	}, nil
}

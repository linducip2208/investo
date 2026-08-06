package service

type MacroIndicator struct {
	Name           string   `json:"name"`
	Value          string   `json:"value"`
	Previous       string   `json:"previous"`
	Change         string   `json:"change"`
	Impact         string   `json:"impact"`
	AffectedSectors []string `json:"affected_sectors"`
}

type MacroDashboardService struct{}

func NewMacroDashboardService() *MacroDashboardService {
	return &MacroDashboardService{}
}

func (s *MacroDashboardService) GetMacroSnapshot() ([]MacroIndicator, error) {
	indicators := []MacroIndicator{
		{
			Name:    "BI Rate (BI7DRR)",
			Value:   "5.75%",
			Previous: "6.00%",
			Change:  "-25 bps",
			Impact:  "positive",
			AffectedSectors: []string{"Perbankan", "Properti", "Otomotif", "Consumer Finance"},
		},
		{
			Name:    "Inflasi (CPI YoY)",
			Value:   "2.8%",
			Previous: "3.1%",
			Change:  "-0.3%",
			Impact:  "positive",
			AffectedSectors: []string{"Consumer Goods", "Retail", "Food & Beverage"},
		},
		{
			Name:    "Pertumbuhan GDP",
			Value:   "5.1%",
			Previous: "5.0%",
			Change:  "+0.1%",
			Impact:  "positive",
			AffectedSectors: []string{"Infrastruktur", "Manufaktur", "Konstruksi", "Perbankan"},
		},
		{
			Name:    "Current Account",
			Value:   "-0.5% GDP",
			Previous: "-0.8% GDP",
			Change:  "+0.3%",
			Impact:  "positive",
			AffectedSectors: []string{"Perbankan", "Export-Oriented", "Commodities"},
		},
		{
			Name:    "USD/IDR",
			Value:   "Rp 17.990",
			Previous: "Rp 18.250",
			Change:  "-260 (-1.4%)",
			Impact:  "positive",
			AffectedSectors: []string{"Import-Dependent", "Pharma", "Aviation", "Technology"},
		},
		{
			Name:    "Cadangan Devisa",
			Value:   "$140 Miliar",
			Previous: "$138 Miliar",
			Change:  "+$2B",
			Impact:  "positive",
			AffectedSectors: []string{"Perbankan", "Sovereign Bonds", "Nilai Tukar"},
		},
		{
			Name:    "Manufacturing PMI",
			Value:   "52.3",
			Previous: "51.8",
			Change:  "+0.5",
			Impact:  "positive",
			AffectedSectors: []string{"Manufaktur", "Industri", "Logistik", "Bahan Baku"},
		},
		{
			Name:    "Consumer Confidence",
			Value:   "125.4",
			Previous: "123.8",
			Change:  "+1.6",
			Impact:  "positive",
			AffectedSectors: []string{"Consumer Goods", "Retail", "Property", "Automotive"},
		},
	}
	return indicators, nil
}

package service

import "fmt"

type MoatResult struct {
	Code       string          `json:"code"`
	Dimensions []MoatDimension `json:"dimensions"`
	TotalStars float64         `json:"total_stars"`
	MoatWidth  string          `json:"moat_width"`
	Narrative  string          `json:"narrative"`
}

type MoatDimension struct {
	Name        string  `json:"name"`
	Stars       float64 `json:"stars"`
	MaxStars    float64 `json:"max_stars"`
	Percentage  float64 `json:"percentage"`
	Description string  `json:"description"`
}

type MoatService struct{}

func NewMoatService() *MoatService {
	return &MoatService{}
}

func (s *MoatService) AnalyzeMoat(code string) (*MoatResult, error) {
	dimensions := []MoatDimension{
		{
			Name:        "Brand Power",
			Stars:       4.0,
			MaxStars:    5.0,
			Percentage:  80,
			Description: "Kekuatan merek di pasar Indonesia. Brand recognition, pricing power, dan customer loyalty.",
		},
		{
			Name:        "Switching Cost",
			Stars:       3.5,
			MaxStars:    5.0,
			Percentage:  70,
			Description: "Biaya dan friksi yang dihadapi customer untuk pindah ke kompetitor. Produk terintegrasi ke workflow customer?",
		},
		{
			Name:        "Network Effect",
			Stars:       3.0,
			MaxStars:    5.0,
			Percentage:  60,
			Description: "Apakah nilai produk bertambah seiring lebih banyak user? Marketplace, platform, atau social network effect.",
		},
		{
			Name:        "Cost Advantage",
			Stars:       4.0,
			MaxStars:    5.0,
			Percentage:  80,
			Description: "Keunggulan biaya struktural: economies of scale, akses bahan baku murah, lokasi strategis, atau teknologi proprietary.",
		},
		{
			Name:        "Intangible Assets",
			Stars:       3.5,
			MaxStars:    5.0,
			Percentage:  70,
			Description: "Paten, lisensi, regulasi, izin eksklusif, spektrum frekuensi, atau hak tambang yang tidak bisa direplikasi.",
		},
	}

	totalStars := 0.0
	for _, d := range dimensions {
		totalStars += d.Stars
	}

	var moatWidth, narrative string
	switch {
	case totalStars >= 20:
		moatWidth = "Wide Moat"
		narrative = "Dengan total " + formatFloat(totalStars) + " dari 25 bintang, " + code + " memiliki WIDE MOAT. Keunggulan kompetitif yang dalam dan sustainable. Perusahaan kemungkinan bisa mempertahankan ROE di atas cost of capital selama 10+ tahun ke depan. Sangat layak untuk investasi jangka panjang."
	case totalStars >= 14:
		moatWidth = "Narrow Moat"
		narrative = "Dengan total " + formatFloat(totalStars) + " dari 25 bintang, " + code + " memiliki NARROW MOAT. Ada keunggulan kompetitif tapi tidak sedalam wide moat. Perusahaan harus terus berinovasi untuk mempertahankan posisi. Masih layak investasi dengan margin of safety yang cukup."
	default:
		moatWidth = "No Moat"
		narrative = "Dengan total " + formatFloat(totalStars) + " dari 25 bintang, " + code + " tidak memiliki economic moat yang signifikan. Keunggulan kompetitif rendah; kompetitor bisa masuk dengan mudah. Sangat berisiko untuk investasi jangka panjang. Hanya cocok untuk trading jangka pendek."
	}

	return &MoatResult{
		Code:       code,
		Dimensions: dimensions,
		TotalStars: totalStars,
		MoatWidth:  moatWidth,
		Narrative:  narrative,
	}, nil
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

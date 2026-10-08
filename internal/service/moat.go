package service

import (
	"fmt"
	"math"
	"time"

	"investo/internal/repository"
)

type MoatResult struct {
	Code           string          `json:"code"`
	Dimensions     []MoatDimension `json:"dimensions"`
	TotalStars     float64         `json:"total_stars"`
	MoatWidth      string          `json:"moat_width"`
	Narrative      string          `json:"narrative"`
	Source         string          `json:"source,omitempty"`
	AsOf           string          `json:"as_of,omitempty"`
	IsIllustrative bool            `json:"is_illustrative,omitempty"`
}

type MoatDimension struct {
	Name        string  `json:"name"`
	Stars       float64 `json:"stars"`
	MaxStars    float64 `json:"max_stars"`
	Percentage  float64 `json:"percentage"`
	Description string  `json:"description"`
}

type MoatService struct {
	StockRepo *repository.StockRepository
	FundRepo  *repository.StockFundamentalRepository
}

func NewMoatService() *MoatService {
	return &MoatService{}
}

func NewMoatServiceWithFundamentals(stockRepo *repository.StockRepository, fundRepo *repository.StockFundamentalRepository) *MoatService {
	return &MoatService{StockRepo: stockRepo, FundRepo: fundRepo}
}

// SetFundamentals wires optional fundamental-data access after construction.
func (s *MoatService) SetFundamentals(stockRepo *repository.StockRepository, fundRepo *repository.StockFundamentalRepository) {
	s.StockRepo = stockRepo
	s.FundRepo = fundRepo
}

func (s *MoatService) AnalyzeMoat(code string) (*MoatResult, error) {
	if s != nil && s.FundRepo != nil {
		if fund, err := s.FundRepo.FindLatestByCode(code); err == nil && fund != nil {
			return s.analyzeFromFundamentals(code, fund.ROE, fund.DER, fund.NetProfitMargin, s.revenueGrowth(code)), nil
		}
	}
	return s.estimateFallback(code), nil
}

// revenueGrowth computes YoY revenue growth from the two latest fundamental
// rows. Returns 0 when history is unavailable.
func (s *MoatService) revenueGrowth(code string) float64 {
	if s == nil || s.StockRepo == nil || s.FundRepo == nil {
		return 0
	}
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil || stock == nil {
		return 0
	}
	rows, err := s.FundRepo.FindByStockID(stock.ID, 2)
	if err != nil || len(rows) < 2 || rows[1].Revenue <= 0 {
		return 0
	}
	return (rows[0].Revenue - rows[1].Revenue) / rows[1].Revenue * 100
}

// scoreMoatStars maps fundamentals (all in % except der which is a multiple)
// to 1-5 stars per dimension. Pure function — covered by honesty tests.
func scoreMoatStars(roe, der, netMargin, revenueGrowth float64) (brand, switching, network, cost, intangible float64) {
	switch {
	case netMargin >= 25:
		brand = 5
	case netMargin >= 18:
		brand = 4.5
	case netMargin >= 12:
		brand = 4
	case netMargin >= 8:
		brand = 3.5
	case netMargin >= 5:
		brand = 3
	case netMargin >= 2:
		brand = 2.5
	case netMargin >= 0:
		brand = 2
	default:
		brand = 1
	}

	switch {
	case roe >= 25:
		cost = 5
	case roe >= 20:
		cost = 4.5
	case roe >= 15:
		cost = 4
	case roe >= 12:
		cost = 3.5
	case roe >= 8:
		cost = 3
	case roe >= 5:
		cost = 2.5
	case roe >= 0:
		cost = 2
	default:
		cost = 1
	}

	switch {
	case der <= 0.2:
		intangible = 5
	case der <= 0.4:
		intangible = 4.5
	case der <= 0.7:
		intangible = 4
	case der <= 1.0:
		intangible = 3.5
	case der <= 1.5:
		intangible = 3
	case der <= 2.0:
		intangible = 2.5
	case der <= 3.0:
		intangible = 2
	default:
		intangible = 1
	}

	switch {
	case revenueGrowth >= 25:
		network = 5
	case revenueGrowth >= 15:
		network = 4.5
	case revenueGrowth >= 10:
		network = 4
	case revenueGrowth >= 5:
		network = 3.5
	case revenueGrowth >= 0:
		network = 3
	case revenueGrowth >= -5:
		network = 2.5
	case revenueGrowth >= -10:
		network = 2
	default:
		network = 1
	}

	switching = math.Round((brand+cost+intangible+network)/4*2) / 2
	if switching < 1 {
		switching = 1
	}
	if switching > 5 {
		switching = 5
	}
	return brand, switching, network, cost, intangible
}

func (s *MoatService) analyzeFromFundamentals(code string, roe, der, netMargin, revenueGrowth float64) *MoatResult {
	brand, switching, network, cost, intangible := scoreMoatStars(roe, der, netMargin, revenueGrowth)

	dimensions := []MoatDimension{
		{
			Name: "Brand Power", Stars: brand, MaxStars: 5.0, Percentage: brand / 5 * 100,
			Description: fmt.Sprintf("Pricing power dari net margin %.1f%%. Margin tebal = pelanggan rela bayar premium.", netMargin),
		},
		{
			Name: "Switching Cost", Stars: switching, MaxStars: 5.0, Percentage: switching / 5 * 100,
			Description: fmt.Sprintf("Rata-rata kekuatan 4 dimensi lain (ROE %.1f%%, DER %.2fx). Skor tinggi = pelanggan sulit pindah.", roe, der),
		},
		{
			Name: "Network Effect", Stars: network, MaxStars: 5.0, Percentage: network / 5 * 100,
			Description: fmt.Sprintf("Daya tarik pertumbuhan pendapatan %.1f%% YoY. Tumbuh cepat = skala & jaringan menguat.", revenueGrowth),
		},
		{
			Name: "Cost Advantage", Stars: cost, MaxStars: 5.0, Percentage: cost / 5 * 100,
			Description: fmt.Sprintf("Efisiensi modal dari ROE %.1f%%. ROE tinggi konsisten = keunggulan biaya struktural.", roe),
		},
		{
			Name: "Intangible Assets", Stars: intangible, MaxStars: 5.0, Percentage: intangible / 5 * 100,
			Description: fmt.Sprintf("Kekuatan neraca dari DER %.2fx. Utang rendah = fleksibilitas & lisensi/posisi tawar lebih kuat.", der),
		},
	}

	return s.buildResult(code, dimensions, "fundamental internal (stock_fundamentals)", time.Now().Format("2006-01-02"), false)
}

// estimateFallback is used only when no fundamental data exists.
func (s *MoatService) estimateFallback(code string) *MoatResult {
	dimensions := []MoatDimension{
		{
			Name: "Brand Power", Stars: 2.5, MaxStars: 5.0, Percentage: 50,
			Description: "Belum ada data fundamental — estimasi netral, bukan hasil analisis.",
		},
		{
			Name: "Switching Cost", Stars: 2.5, MaxStars: 5.0, Percentage: 50,
			Description: "Belum ada data fundamental — estimasi netral, bukan hasil analisis.",
		},
		{
			Name: "Network Effect", Stars: 2.5, MaxStars: 5.0, Percentage: 50,
			Description: "Belum ada data fundamental — estimasi netral, bukan hasil analisis.",
		},
		{
			Name: "Cost Advantage", Stars: 2.5, MaxStars: 5.0, Percentage: 50,
			Description: "Belum ada data fundamental — estimasi netral, bukan hasil analisis.",
		},
		{
			Name: "Intangible Assets", Stars: 2.5, MaxStars: 5.0, Percentage: 50,
			Description: "Belum ada data fundamental — estimasi netral, bukan hasil analisis.",
		},
	}

	return s.buildResult(code, dimensions, "estimasi ilustratif — data fundamental tidak tersedia", time.Now().Format("2006-01-02"), true)
}

func (s *MoatService) buildResult(code string, dimensions []MoatDimension, source, asOf string, illustrative bool) *MoatResult {
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
	if illustrative {
		narrative += " (Estimasi ilustratif — belum didukung data fundamental.)"
	}

	return &MoatResult{
		Code:           code,
		Dimensions:     dimensions,
		TotalStars:     totalStars,
		MoatWidth:      moatWidth,
		Narrative:      narrative,
		Source:         source,
		AsOf:           asOf,
		IsIllustrative: illustrative,
	}
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

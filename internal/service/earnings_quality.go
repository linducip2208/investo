package service

import (
	"fmt"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type QualityScore struct {
	Code           string               `json:"code"`
	Score          float64              `json:"score"`
	Rating         string               `json:"rating"`
	Components     []QualityComponent   `json:"components"`
	Interpretation string               `json:"interpretation"`
	Source         string               `json:"source,omitempty"`
	AsOf           string               `json:"as_of,omitempty"`
	IsIllustrative bool                 `json:"is_illustrative,omitempty"`
}

type QualityComponent struct {
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
	Weight    float64 `json:"weight"`
	Detail    string  `json:"detail"`
	IsEstimate bool   `json:"is_estimate,omitempty"`
}

type EarningsQualityService struct {
	StockFundamentalRepo *repository.StockFundamentalRepository
}

func NewEarningsQualityService(fundRepo *repository.StockFundamentalRepository) *EarningsQualityService {
	return &EarningsQualityService{StockFundamentalRepo: fundRepo}
}

func (s *EarningsQualityService) CalculateQualityScore(code string) (*QualityScore, error) {
	fund, err := s.StockFundamentalRepo.FindLatestByCode(code)
	if err != nil || fund == nil {
		return s.sampleScore(code), nil
	}

	components := earningsComponents(fund)

	total := 0.0
	for _, c := range components {
		total += c.Score * c.Weight
	}

	rating := "Poor"
	switch {
	case total >= 80:
		rating = "Excellent"
	case total >= 60:
		rating = "Good"
	case total >= 40:
		rating = "Fair"
	}

	return &QualityScore{
		Code:           code,
		Score:          total,
		Rating:         rating,
		Components:     components,
		Interpretation: s.interpretRating(rating, code),
		Source:         "fundamental internal (stock_fundamentals); komponen tanpa data arus kas ditandai estimasi",
		AsOf:           fund.Period,
	}, nil
}

// earningsComponents scores each quality pillar from actual fundamental
// ratios. Pillars without backing data in the local feed (arus kas, piutang,
// pos luar biasa) return fixed neutral scores explicitly flagged IsEstimate.
func earningsComponents(fund *model.StockFundamental) []QualityComponent {
	accrualScore := 55.0
	switch {
	case fund.ROA >= 10 && fund.DER <= 1:
		accrualScore = 85
	case fund.ROA >= 5:
		accrualScore = 75
	case fund.ROA >= 2:
		accrualScore = 65
	case fund.ROA >= 0:
		accrualScore = 55
	default:
		accrualScore = 40
	}

	cfoScore := 75.0
	if fund.NetIncome > 0 && fund.Revenue > 0 {
		if fund.NetIncome/fund.Revenue > 0.15 {
			cfoScore = 82.0
		}
	}

	turnover := 0.0
	if fund.TotalAssets > 0 {
		turnover = fund.Revenue / fund.TotalAssets
	}
	assetScore := 50.0
	switch {
	case turnover >= 1.0:
		assetScore = 85
	case turnover >= 0.6:
		assetScore = 78
	case turnover >= 0.3:
		assetScore = 70
	case turnover > 0:
		assetScore = 60
	}

	return []QualityComponent{
		{Name: "Accrual Ratio", Score: accrualScore, Weight: 0.25, Detail: fmt.Sprintf("Proksi akrual dari ROA %.1f%% & DER %.2fx. ROA tinggi + leverage rendah = akrual rendah, laba berkualitas.", fund.ROA, fund.DER)},
		{Name: "CFO / Net Income", Score: cfoScore, Weight: 0.25, Detail: "Arus kas operasi tidak tersedia di feed lokal — skor asumsi dari margin laba, bukan rasio kas aktual.", IsEstimate: true},
		{Name: "Asset Turnover Trend", Score: assetScore, Weight: 0.20, Detail: fmt.Sprintf("Perputaran aset (revenue/total aset) %.2fx. Semakin tinggi semakin efisien.", turnover)},
		{Name: "Receivables / Revenue", Score: 70.0, Weight: 0.15, Detail: "Rincian piutang tidak tersedia di feed lokal — skor netral, bukan hasil analisis.", IsEstimate: true},
		{Name: "One-Time Items", Score: 75.0, Weight: 0.15, Detail: "Pos luar biasa tidak dirinci di feed lokal — skor netral, bukan hasil analisis.", IsEstimate: true},
	}
}

func (s *EarningsQualityService) sampleScore(code string) *QualityScore {
	components := []QualityComponent{
		{Name: "Accrual Ratio", Score: 68.0, Weight: 0.25, Detail: "Rasio akrual terhadap total aset.", IsEstimate: true},
		{Name: "CFO / Net Income", Score: 78.0, Weight: 0.25, Detail: "Arus kas operasi terhadap laba bersih.", IsEstimate: true},
		{Name: "Asset Turnover Trend", Score: 72.0, Weight: 0.20, Detail: "Perubahan efisiensi penggunaan aset.", IsEstimate: true},
		{Name: "Receivables / Revenue", Score: 65.0, Weight: 0.15, Detail: "Rasio piutang terhadap pendapatan.", IsEstimate: true},
		{Name: "One-Time Items", Score: 82.0, Weight: 0.15, Detail: "Pendapatan non-recurring.", IsEstimate: true},
	}
	total := 68.0 + 78.0 + 72.0 + 65.0 + 82.0
	total = 72.6

	return &QualityScore{
		Code:       code,
		Score:      total,
		Rating:     "Good",
		Components: components,
		Interpretation: s.interpretRating("Good", code),
		Source: "estimasi ilustratif — data fundamental tidak tersedia",
		AsOf: time.Now().Format("2006-01-02"),
		IsIllustrative: true,
	}
}

func (s *EarningsQualityService) interpretRating(rating, code string) string {
	switch rating {
	case "Excellent":
		return code + " memiliki kualitas laba sangat baik. Arus kas kuat, akrual rendah, dan pendapatan berulang dominan. Laba yang dilaporkan sangat mencerminkan realitas ekonomi bisnis."
	case "Good":
		return code + " memiliki kualitas laba yang baik. Mayoritas laba didukung arus kas operasi. Beberapa item non-recurring ada, tapi tidak signifikan. Investor bisa cukup percaya pada earnings."
	case "Fair":
		return code + " memiliki kualitas laba cukup. Ada indikasi akrual yang meningkat atau selisih laba vs arus kas. Perlu analisis lebih dalam sebelum berinvestasi."
	default:
		return code + " memiliki kualitas laba rendah. Terdapat red flags: akrual tinggi, arus kas jauh di bawah laba, atau banyak item non-recurring. Investor harus sangat berhati-hati."
	}
}

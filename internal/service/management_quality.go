package service

import (
	"fmt"
	"time"

	"investo/internal/repository"
)

type ManagementScore struct {
	Code           string              `json:"code"`
	Score          float64             `json:"score"`
	Rating         string              `json:"rating"`
	TrackRecord    *TrackRecordScore   `json:"track_record"`
	CapitalAlloc   *CapitalAllocScore  `json:"capital_allocation"`
	Interpretation string              `json:"interpretation"`
	Source         string              `json:"source,omitempty"`
	AsOf           string              `json:"as_of,omitempty"`
	IsIllustrative bool                `json:"is_illustrative,omitempty"`
}

type TrackRecordScore struct {
	Score     float64 `json:"score"`
	Tenure    string  `json:"tenure"`
	StockPerf string  `json:"stock_performance"`
	Detail    string  `json:"detail"`
}

type CapitalAllocScore struct {
	Score          float64 `json:"score"`
	ROETrend       string  `json:"roe_trend"`
	BuybackHistory string  `json:"buyback_history"`
	DividendConsistency string `json:"dividend_consistency"`
	Detail         string  `json:"detail"`
}

type ManagementQualityService struct {
	StockFundamentalRepo *repository.StockFundamentalRepository
	StockRepo            *repository.StockRepository
}

func NewManagementQualityService(fundRepo *repository.StockFundamentalRepository, stockRepo *repository.StockRepository) *ManagementQualityService {
	return &ManagementQualityService{StockFundamentalRepo: fundRepo, StockRepo: stockRepo}
}

func (s *ManagementQualityService) AssessManagement(code string) (*ManagementScore, error) {
	fund, err := s.StockFundamentalRepo.FindLatestByCode(code)
	if err != nil || fund == nil {
		return s.sampleScore(code), nil
	}

	caScore := 45.0
	switch {
	case fund.ROE > 20 && fund.DER < 1:
		caScore = 85.0
	case fund.ROE > 15:
		caScore = 78.0
	case fund.ROE > 10:
		caScore = 68.0
	case fund.ROE > 5:
		caScore = 58.0
	}

	trScore := 60.0
	if fund.ROE > 15 {
		trScore = 75.0
	} else if fund.ROE > 8 {
		trScore = 68.0
	}

	roeTrend := s.roeTrend(code, fund.ROE)

	buyback := "riwayat buyback tidak tersedia di feed lokal"
	dividend := "tidak ada dividen tercatat pada periode terakhir"
	if fund.DividendYield > 0 {
		dividend = fmt.Sprintf("dividend yield %.2f%% pada periode terakhir (konsistensi historis tidak tersedia di feed lokal)", fund.DividendYield)
	}

	tr := &TrackRecordScore{
		Score:     trScore,
		Tenure:    "masa jabatan direksi tidak tersedia di feed lokal",
		StockPerf: "kinerja harga tidak tersedia — hubungkan price history",
		Detail:    fmt.Sprintf("Skor track record %.0f diturunkan dari ROE %.1f%% periode %s. Data tenure & kinerja harga aktual belum tersedia.", trScore, fund.ROE, fund.Period),
	}

	ca := &CapitalAllocScore{
		Score:              caScore,
		ROETrend:           roeTrend,
		BuybackHistory:     buyback,
		DividendConsistency: dividend,
		Detail:             fmt.Sprintf("Skor alokasi modal %.0f diturunkan dari ROE %.1f%% & DER %.2fx periode %s. Buyback & konsistensi dividen belum terverifikasi dari feed lokal.", caScore, fund.ROE, fund.DER, fund.Period),
	}

	total := (tr.Score*0.4 + ca.Score*0.6)

	rating := "Poor"
	switch {
	case total >= 80:
		rating = "Excellent"
	case total >= 60:
		rating = "Good"
	case total >= 40:
		rating = "Fair"
	}

	return &ManagementScore{
		Code:           code,
		Score:          total,
		Rating:         rating,
		TrackRecord:    tr,
		CapitalAlloc:   ca,
		Interpretation: s.interpretRating(rating, code),
		Source:         "fundamental internal (stock_fundamentals); klaim tanpa data ditandai tidak tersedia",
		AsOf:           fund.Period,
	}, nil
}

// roeTrend describes the ROE trajectory from up to 5 recent fundamental rows.
func (s *ManagementQualityService) roeTrend(code string, currentROE float64) string {
	if s == nil || s.StockRepo == nil || s.StockFundamentalRepo == nil {
		return fmt.Sprintf("ROE %.1f%% (tren historis tidak tersedia)", currentROE)
	}
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil || stock == nil {
		return fmt.Sprintf("ROE %.1f%% (tren historis tidak tersedia)", currentROE)
	}
	rows, err := s.StockFundamentalRepo.FindByStockID(stock.ID, 5)
	if err != nil || len(rows) < 2 {
		return fmt.Sprintf("ROE %.1f%% (hanya 1 periode tersedia)", currentROE)
	}
	first, last := rows[len(rows)-1].ROE, rows[0].ROE
	direction := "stabil"
	if last-first > 2 {
		direction = "membaik"
	} else if first-last > 2 {
		direction = "melemah"
	}
	return fmt.Sprintf("%s: ROE %.1f%% → %.1f%% (%d periode)", direction, first, last, len(rows))
}

func (s *ManagementQualityService) sampleScore(code string) *ManagementScore {
	tr := &TrackRecordScore{
		Score:     60,
		Tenure:    "tidak tersedia — data fundamental belum ada",
		StockPerf: "tidak tersedia — data fundamental belum ada",
		Detail:    "Estimasi netral, bukan hasil analisis.",
	}
	ca := &CapitalAllocScore{
		Score:               60,
		ROETrend:            "tidak tersedia — data fundamental belum ada",
		BuybackHistory:      "tidak tersedia — data fundamental belum ada",
		DividendConsistency: "tidak tersedia — data fundamental belum ada",
		Detail:              "Estimasi netral, bukan hasil analisis.",
	}

	return &ManagementScore{
		Code:           code,
		Score:          60,
		Rating:         "Fair",
		TrackRecord:    tr,
		CapitalAlloc:   ca,
		Interpretation: s.interpretRating("Fair", code),
		Source:         "estimasi ilustratif — data fundamental tidak tersedia",
		AsOf:           time.Now().Format("2006-01-02"),
		IsIllustrative: true,
	}
}

func (s *ManagementQualityService) interpretRating(rating, code string) string {
	switch rating {
	case "Excellent":
		return "Manajemen " + code + " berkualitas excellent. Track record panjang dengan kinerja saham superior. Alokasi modal sangat efisien dengan ROE tinggi, buyback oportunis, dan dividen konsisten. Governance kuat."
	case "Good":
		return "Manajemen " + code + " berkualitas baik. Track record solid, alokasi modal prudent, dan shareholder value menjadi prioritas. Beberapa area minor bisa ditingkatkan."
	case "Fair":
		return "Manajemen " + code + " berkualitas cukup. Kinerja mixed, ada beberapa keputusan alokasi modal yang perlu dicermati. Investor sebaiknya monitor governance lebih dekat."
	default:
		return "Manajemen " + code + " berkualitas rendah. Track record buruk, alokasi modal tidak efisien, atau governance bermasalah. Investor harus sangat berhati-hati."
	}
}

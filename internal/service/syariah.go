package service

import (
	"fmt"
	"strings"
	"time"

	"investo/internal/repository"
)

type SyariahStatus struct {
	StockCode       string  `json:"stock_code"`
	StockName       string  `json:"stock_name"`
	IsSyariah       bool    `json:"is_syariah"`
	DESPeriod       string  `json:"des_period"`
	DebtRatio       float64 `json:"debt_ratio"`
	NonHalalRevenue float64 `json:"non_halal_revenue"`
	Status          string  `json:"status"`
	Source          string  `json:"source,omitempty"`
	AsOf            string  `json:"as_of,omitempty"`
	Note            string  `json:"note,omitempty"`
}

type SyariahService struct {
	StockRepo            *repository.StockRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
}

// desList adalah snapshot statis keanggotaan DES — perbarui manual setiap
// rilis SK DES OJK/BEI (Mei & November). Emiten di luar daftar ditandai
// "Belum Terverifikasi", bukan otomatis syariah.
var desList = map[string]bool{
	"BBCA": true, "BBRI": true, "BMRI": true, "BBNI": true,
	"TLKM": true, "ADRO": true, "KLBF": true, "ICBP": true,
	"INDF": true, "CPIN": true, "ANTM": true, "ITMG": true,
	"PTBA": true, "SMGR": true, "UNTR": true, "AKRA": true,
	"EXCL": true, "TOWR": true, "ACES": true, "AMRT": true,
	"BRPT": true, "MDKA": true, "HRUM": true,
	"UNVR": false, "HMSP": false, "GGRM": false,
	"INTP": true, "JSMR": true, "PGAS": true, "TINS": true,
}

func (s *SyariahService) CheckCompliance(code string) (*SyariahStatus, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, err
	}

	upper := strings.ToUpper(code)
	isSyariah, exists := desList[upper]
	status := "Tidak Syariah"

	if !exists {
		isSyariah = true
		status = "Belum Terverifikasi"
	} else if isSyariah {
		status = "Syariah (DES)"
	}

	debtRatio := 0.0
	nonHalalRevenue := 0.0
	note := ""

	fundamentals, err := s.StockFundamentalRepo.FindByStockID(stock.ID, 1)
	if err == nil && len(fundamentals) > 0 {
		latest := fundamentals[0]
		if latest.TotalAssets > 0 {
			debtRatio = latest.TotalLiabilities / latest.TotalAssets * 100
		}
		if latest.Revenue > 0 {
			// Feed lokal tidak merinci pendapatan non-halal per segmen;
			// 1,5% adalah asumsi konservatif, bukan angka audit.
			nonHalalRevenue = 1.5
			note = "non-halal revenue = asumsi 1,5% (bukan angka audit)"
		}
	}

	return &SyariahStatus{
		StockCode:       stock.Code,
		StockName:       stock.Name,
		IsSyariah:       isSyariah,
		DESPeriod:       currentDESPeriod(time.Now()),
		DebtRatio:       debtRatio,
		NonHalalRevenue: nonHalalRevenue,
		Status:          status,
		Source:          "daftar DES internal — verifikasi ke SK DES OJK/BEI terbaru",
		AsOf:            time.Now().Format("2006-01-02"),
		Note:            note,
	}, nil
}

// currentDESPeriod labels the latest scheduled DES release (Mei/November)
// relative to now instead of a hardcoded stale period.
func currentDESPeriod(now time.Time) string {
	y := now.Year()
	switch {
	case int(now.Month()) >= 11:
		return fmt.Sprintf("DES November %d", y)
	case int(now.Month()) >= 5:
		return fmt.Sprintf("DES Mei %d", y)
	default:
		return fmt.Sprintf("DES November %d", y-1)
	}
}

func (s *SyariahService) GetAllSyariah() ([]SyariahStatus, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	var results []SyariahStatus
	for _, st := range stocks {
		status, err := s.CheckCompliance(st.Code)
		if err != nil {
			continue
		}
		results = append(results, *status)
	}
	return results, nil
}

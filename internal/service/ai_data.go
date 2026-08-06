package service

import (
	"fmt"
	"math"
	"strings"

	"investo/internal/repository"
)

type AIDataService struct {
	AI                   *AIService
	StockRepo            *repository.StockRepository
	StockPriceRepo       *repository.StockPriceRepository
	StockFundamentalRepo *repository.StockFundamentalRepository
	SectorRepo           *repository.SectorRepository
}

type ImputedFundamental struct {
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	EstimatedPER float64 `json:"estimated_per"`
	EstimatedPBV float64 `json:"estimated_pbv"`
	EstimatedROE float64 `json:"estimated_roe"`
	EstimatedDER float64 `json:"estimated_der"`
	Confidence   string  `json:"confidence"`
	Explanation  string  `json:"explanation"`
}

type AnomalyExplanation struct {
	StockCode    string   `json:"stock_code"`
	StockName    string   `json:"stock_name"`
	AnomalyType  string   `json:"anomaly_type"`
	Severity     string   `json:"severity"`
	Explanation  string   `json:"explanation"`
	PossibleCauses []string `json:"possible_causes"`
	Recommendation string  `json:"recommendation"`
}

func (s *AIDataService) ImputeMissingFundamentals(stockCode string) (*ImputedFundamental, error) {
	stock, err := s.StockRepo.FindByCode(stockCode)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", stockCode)
	}

	existingFund, _ := s.StockFundamentalRepo.FindLatest(stock.ID)
	price, _ := s.StockPriceRepo.GetLatestPrice(stock.ID)

	result := &ImputedFundamental{
		StockCode: stock.Code,
		StockName: stock.Name,
	}

	sectorAvgPER, sectorAvgPBV, sectorAvgROE, sectorAvgDER := s.getSectorAverages(stock.SectorID)

	hasPartialData := existingFund != nil

	if hasPartialData {
		if existingFund.PER > 0 {
			result.EstimatedPER = existingFund.PER
		} else {
			result.EstimatedPER = sectorAvgPER
		}
		if existingFund.PBV > 0 {
			result.EstimatedPBV = existingFund.PBV
		} else {
			result.EstimatedPBV = sectorAvgPBV
		}
		if existingFund.ROE > 0 {
			result.EstimatedROE = existingFund.ROE
		} else {
			result.EstimatedROE = sectorAvgROE
		}
		if existingFund.DER > 0 {
			result.EstimatedDER = existingFund.DER
		} else {
			result.EstimatedDER = sectorAvgDER
		}
	} else {
		result.EstimatedPER = sectorAvgPER
		result.EstimatedPBV = sectorAvgPBV
		result.EstimatedROE = sectorAvgROE
		result.EstimatedDER = sectorAvgDER
	}

	switch {
	case hasPartialData && existingFund.PER > 0 && existingFund.PBV > 0:
		result.Confidence = "Tinggi"
	case hasPartialData:
		result.Confidence = "Sedang"
	default:
		result.Confidence = "Rendah (estimasi dari rata-rata sektor)"
	}

	var explanations []string
	if !hasPartialData {
		explanations = append(explanations, fmt.Sprintf("Data fundamental %s belum tersedia. Estimasi menggunakan rata-rata sektor.", stock.Code))
	} else {
		if existingFund.PER <= 0 {
			explanations = append(explanations, fmt.Sprintf("PER diestimasi dari rata-rata sektor (%.1fx) karena data belum tersedia.", sectorAvgPER))
		}
		if existingFund.PBV <= 0 {
			explanations = append(explanations, fmt.Sprintf("PBV diestimasi dari rata-rata sektor (%.1fx).", sectorAvgPBV))
		}
		if existingFund.ROE <= 0 {
			explanations = append(explanations, fmt.Sprintf("ROE diestimasi %.1f%% berdasarkan rata-rata sektor.", sectorAvgROE))
		}
	}

	if price > 0 {
		explanations = append(explanations, fmt.Sprintf("Harga terakhir: Rp %.0f.", price))
	}

	if result.EstimatedPER > 0 {
		if result.EstimatedPER < 12 {
			explanations = append(explanations, fmt.Sprintf("Dengan PER estimasi %.1fx, valuasi tergolong rendah dibanding rata-rata sektor.", result.EstimatedPER))
		} else if result.EstimatedPER > 25 {
			explanations = append(explanations, fmt.Sprintf("PER estimasi %.1fx tergolong premium — pastikan growth story mendukung.", result.EstimatedPER))
		}
	}

	result.Explanation = strings.Join(explanations, " ")

	aiEnhanced := s.AI.ChatWithFallback(
		"Kamu adalah analis fundamental. Jelaskan hasil imputasi data fundamental dalam bahasa Indonesia, 2-3 kalimat. Jangan gunakan placeholder. Spesifik ke saham yang disebutkan.",
		fmt.Sprintf("Saham %s (%s): PER=%.1f, PBV=%.1f, ROE=%.1f%%, DER=%.1f. Confidence: %s. Data asli: %v. Jelaskan implikasinya secara singkat.",
			stock.Code, stock.Name, result.EstimatedPER, result.EstimatedPBV, result.EstimatedROE, result.EstimatedDER, result.Confidence, hasPartialData),
		result.Explanation,
	)
	result.Explanation = aiEnhanced

	return result, nil
}

func (s *AIDataService) getSectorAverages(sectorID int64) (per, pbv, roe, der float64) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return 15.0, 2.0, 12.0, 1.0
	}

	var sumPER, sumPBV, sumROE, sumDER float64
	count := 0

	for _, stock := range stocks {
		if stock.SectorID != sectorID || sectorID == 0 {
			if sectorID != 0 {
				continue
			}
		}

		fund, err := s.StockFundamentalRepo.FindLatest(stock.ID)
		if err != nil || fund == nil {
			continue
		}

		if fund.PER > 0 {
			sumPER += fund.PER
			count++
		}
		if fund.PBV > 0 {
			sumPBV += fund.PBV
		}
		if fund.ROE > 0 {
			sumROE += fund.ROE
		}
		if fund.DER >= 0 {
			sumDER += math.Abs(fund.DER)
		}
	}

	if count == 0 {
		return 15.0, 2.0, 12.0, 1.0
	}

	per = sumPER / float64(count)
	pbv = sumPBV / float64(count)
	roe = sumROE / float64(count)
	der = sumDER / float64(count)

	if per <= 0 {
		per = 15.0
	}
	if pbv <= 0 {
		pbv = 2.0
	}
	if roe <= 0 {
		roe = 12.0
	}
	if der <= 0 {
		der = 1.0
	}

	return
}

func (s *AIDataService) ExplainAnomaly(stockCode string, anomalyType string) (*AnomalyExplanation, error) {
	stock, err := s.StockRepo.FindByCode(stockCode)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %s", stockCode)
	}

	price, _ := s.StockPriceRepo.GetLatestPrice(stock.ID)

	result := &AnomalyExplanation{
		StockCode:   stock.Code,
		StockName:   stock.Name,
		AnomalyType: anomalyType,
	}

	var typeLabel string
	switch anomalyType {
	case "price_spike":
		typeLabel = "lonjakan harga"
		result.Severity = "Medium"
		result.PossibleCauses = []string{
			"Rilis laporan keuangan yang lebih baik dari ekspektasi",
			"Rekomendasi buy dari analis atau investment bank",
			"Aksi korporasi: stock split, dividen besar, atau right issue",
			"Akumulasi oleh investor institusi atau asing",
			"Short squeeze — tekanan pada posisi short",
		}
	case "price_drop":
		typeLabel = "penurunan harga tajam"
		result.Severity = "High"
		result.PossibleCauses = []string{
			"Laporan keuangan di bawah ekspektasi pasar",
			"Downgrade rekomendasi oleh analis",
			"Sentimen negatif sektoral atau makroekonomi",
			"Profit taking setelah kenaikan signifikan",
			"Capital outflow dari investor asing",
		}
	case "volume_spike":
		typeLabel = "lonjakan volume"
		result.Severity = "Low"
		result.PossibleCauses = []string{
			"Akumulasi atau distribusi oleh institusi besar",
			"Transaksi negosiasi dalam jumlah besar",
			"Rebalancing indeks (LQ45, IDX30, dll.)",
			"Spekulasi menjelang pengumuman korporasi",
			"Algoritmic trading atau program trading",
		}
	case "gap":
		typeLabel = "gap harga"
		result.Severity = "Medium"
		result.PossibleCauses = []string{
			"Pengumuman material setelah jam pasar tutup",
			"Dividen cum/ex date yang signifikan",
			"Stock split atau reverse split",
			"Sentimen global yang mempengaruhi sektor terkait",
			"Rumor pasar atau berita yang beredar di media sosial",
		}
	default:
		typeLabel = fmt.Sprintf("anomali %s", anomalyType)
		result.Severity = "Medium"
		result.PossibleCauses = []string{
			"Fluktuasi pasar normal",
			"Perubahan sentimen investor",
			"Aktivitas trading institusional",
		}
	}

	fallbackExplanation := fmt.Sprintf(
		"Saham %s mengalami %s di harga Rp %.0f. Beberapa kemungkinan penyebab: %s. Untuk investor jangka panjang, perhatikan apakah anomali ini didukung oleh perubahan fundamental atau hanya bersifat teknikal sesaat.",
		stock.Code, typeLabel, price, strings.Join(result.PossibleCauses[:minInt(3, len(result.PossibleCauses))], "; "),
	)

	aiPrompt := fmt.Sprintf(
		"Saham %s (%s) mengalami %s di harga Rp %.0f. Berikan penjelasan kemungkinan penyebab dan rekomendasi dalam bahasa Indonesia. 2-3 kalimat maksimal. Spesifik ke saham dan situasi Indonesia.",
		stock.Code, stock.Name, typeLabel, price,
	)

	systemPrompt := "Kamu adalah analis pasar modal Indonesia yang berpengalaman. Berikan analisa objektif tentang anomali pasar."
	result.Explanation = s.AI.ChatWithFallback(systemPrompt, aiPrompt, fallbackExplanation)

	switch {
	case strings.Contains(result.Explanation, "waspada") || strings.Contains(result.Explanation, "hati-hati"):
		result.Recommendation = "Pantau dengan ketat. Pertimbangkan untuk mengurangi posisi jika anomali berlanjut."
	case strings.Contains(result.Explanation, "peluang") || strings.Contains(result.Explanation, "positif"):
		result.Recommendation = "Anomali ini bisa menjadi peluang entry jika didukung fundamental yang solid. Konfirmasi dengan analisa teknikal."
	default:
		result.Recommendation = "Pantau pergerakan selanjutnya. Jangan mengambil keputusan impulsif berdasarkan satu anomali. Konfirmasi dengan data fundamental dan teknikal."
	}

	return result, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

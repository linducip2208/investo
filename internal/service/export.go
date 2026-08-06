package service

import (
	"encoding/csv"
	"fmt"
	"math"
	"strings"
)

type ExportService struct{}

type StockCSVRow struct {
	Code          string
	Name          string
	SectorName    string
	Price         float64
	ChangePercent float64
	PER           float64
	PBV           float64
	ROE           float64
	DER           float64
	MarketCap     float64
}

type PortfolioCSVRow struct {
	Code          string
	Name          string
	Quantity      float64
	AvgPrice      float64
	CurrentPrice  float64
	MarketValue   float64
	Gain          float64
	GainPercent   float64
}

type ScreenerCSVRow struct {
	Code          string
	Name          string
	SectorName    string
	Price         float64
	ChangePercent float64
	PER           float64
	PBV           float64
	ROE           float64
	DER           float64
	NPM           float64
	DivYield      float64
	MarketCap     float64
	Score         int
}

func NewExportService() *ExportService {
	return &ExportService{}
}

func (s *ExportService) ExportStocksCSV(stocks []StockCSVRow) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(&buf)

	headers := []string{"Kode", "Nama", "Sektor", "Harga", "Perubahan%", "PER", "PBV", "ROE", "DER", "Market Cap"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("ExportStocksCSV header: %w", err)
	}

	for _, stock := range stocks {
		row := []string{
			stock.Code,
			stock.Name,
			stock.SectorName,
			floatStr(stock.Price),
			floatStr(stock.ChangePercent),
			floatStr(stock.PER),
			floatStr(stock.PBV),
			floatStr(stock.ROE),
			floatStr(stock.DER),
			floatStr(stock.MarketCap),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("ExportStocksCSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("ExportStocksCSV flush: %w", err)
	}

	return []byte(buf.String()), nil
}

func (s *ExportService) ExportPortfolioCSV(holdings []PortfolioCSVRow) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(&buf)

	headers := []string{"Kode", "Nama", "Jumlah", "Harga Rata", "Harga Saat Ini", "Nilai Pasar", "Gain/Loss", "Gain/Loss%"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("ExportPortfolioCSV header: %w", err)
	}

	for _, h := range holdings {
		row := []string{
			h.Code,
			h.Name,
			floatStr(h.Quantity),
			floatStr(h.AvgPrice),
			floatStr(h.CurrentPrice),
			floatStr(h.MarketValue),
			floatStr(h.Gain),
			floatStr(h.GainPercent),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("ExportPortfolioCSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("ExportPortfolioCSV flush: %w", err)
	}

	return []byte(buf.String()), nil
}

func (s *ExportService) ExportScreenerCSV(results []ScreenerCSVRow) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(&buf)

	headers := []string{"Kode", "Nama", "Sektor", "Harga", "Perubahan%", "PER", "PBV", "ROE", "DER", "NPM", "Div Yield", "Market Cap", "Score"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("ExportScreenerCSV header: %w", err)
	}

	for _, r := range results {
		row := []string{
			r.Code,
			r.Name,
			r.SectorName,
			floatStr(r.Price),
			floatStr(r.ChangePercent),
			floatStr(r.PER),
			floatStr(r.PBV),
			floatStr(r.ROE),
			floatStr(r.DER),
			floatStr(r.NPM),
			floatStr(r.DivYield),
			floatStr(r.MarketCap),
			fmt.Sprintf("%d", r.Score),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("ExportScreenerCSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("ExportScreenerCSV flush: %w", err)
	}

	return []byte(buf.String()), nil
}

func floatStr(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return fmt.Sprintf("%.2f", v)
}

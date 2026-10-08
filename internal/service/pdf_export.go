package service

import (
	"fmt"
	"strings"

	"investo/internal/model"
)

type PDFExportService struct {
	AppName string
}

func NewPDFExportService(appName string) *PDFExportService {
	return &PDFExportService{AppName: appName}
}

func (s *PDFExportService) GenerateStockReportPDF(stock model.Stock, price float64, change float64, fundamental *model.StockFundamental) ([]byte, error) {
	var fundRows string
	if fundamental != nil {
		fundRows = fmt.Sprintf(`
		<tr><td><strong>PER</strong></td><td>%.2fx</td></tr>
		<tr><td><strong>PBV</strong></td><td>%.2fx</td></tr>
		<tr><td><strong>ROE</strong></td><td>%.2f%%</td></tr>
		<tr><td><strong>DER</strong></td><td>%.2fx</td></tr>
		<tr><td><strong>EPS</strong></td><td>Rp %s</td></tr>
		<tr><td><strong>NPM</strong></td><td>%.2f%%</td></tr>
		<tr><td><strong>Dividend Yield</strong></td><td>%.2f%%</td></tr>`,
			fundamental.PER, fundamental.PBV, fundamental.ROE, fundamental.DER,
			formatRupiahShort(fundamental.EPS), fundamental.NetProfitMargin, fundamental.DividendYield)
	}

	changeClass := "up"
	changeSign := "+"
	if change < 0 {
		changeClass = "down"
		changeSign = ""
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Laporan %s - %s | %s</title>
<style>
@page{size:A4;margin:15mm}@media print{body{-webkit-print-color-adjust:exact;print-color-adjust:exact}.no-print{display:none!important}}
*{box-sizing:border-box;margin:0;padding:0}body{font-family:'Inter',system-ui,-apple-system,sans-serif;font-size:13px;color:#1e293b;line-height:1.6;max-width:210mm;margin:0 auto;padding:20px;background:#fff}.header{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:3px solid #3b82f6;padding-bottom:16px;margin-bottom:24px}.header h1{font-size:24px;font-weight:700;color:#0f172a}.header .code{font-size:14px;color:#64748b}.price{font-size:28px;font-weight:700;color:#0f172a}.change{font-size:15px;font-weight:600}.up{color:#16a34a}.down{color:#dc2626}.section{margin-bottom:24px}.section h2{font-size:16px;font-weight:600;color:#1e40af;border-bottom:2px solid #e2e8f0;padding-bottom:6px;margin-bottom:12px}table{width:100%%;border-collapse:collapse}td,th{padding:8px 10px;text-align:left;border-bottom:1px solid #e2e8f0}th{background:#f8fafc;font-weight:600;color:#475569;font-size:11px;text-transform:uppercase;letter-spacing:.04em}.footer{margin-top:32px;padding-top:12px;border-top:1px solid #e2e8f0;text-align:center;color:#94a3b8;font-size:11px}.print-btn{position:fixed;top:16px;right:16px;background:#3b82f6;color:#fff;border:none;padding:10px 22px;border-radius:8px;font-size:13px;font-weight:600;cursor:pointer;z-index:100;transition:background .2s}.print-btn:hover{background:#2563eb}
</style>
</head>
<body>
<button class="print-btn no-print" onclick="window.print()">Cetak PDF</button>
<div class="header">
  <div>
    <div class="code">%s</div>
    <h1>%s</h1>
  </div>
  <div style="text-align:right">
    <div class="price">Rp %s</div>
    <div class="change %s">%s%.2f%%</div>
  </div>
</div>
<div class="section">
  <h2>Data Fundamental</h2>
  %s
</div>
<div class="footer">
  <p>Laporan digenerate oleh %s &mdash; Platform Analisa Saham & Forex</p>
  <p>Disclaimer: Laporan ini bukan rekomendasi investasi.</p>
</div>
</body>
</html>`, stock.Code, stock.Name, s.AppName,
		stock.Code, stock.Name, formatRupiahShort(price), changeClass, changeSign, change,
		fundRows, s.AppName)

	return []byte(html), nil
}

func (s *PDFExportService) GeneratePortfolioPDF(portfolio model.Portfolio, holdings []PortfolioHoldingExport, totalValue, totalCost, totalGain, totalGainPercent float64) ([]byte, error) {
	var holdingRows strings.Builder
	for _, h := range holdings {
		gainClass := "up"
		gainSign := "+"
		if h.Gain < 0 {
			gainClass = "down"
			gainSign = ""
		}
		holdingRows.WriteString(fmt.Sprintf(`<tr><td style="font-weight:600">%s</td><td>%s</td><td>%.0f</td><td>Rp %s</td><td>Rp %s</td><td>Rp %s</td><td class="%s">%sRp %s</td><td class="%s">%s%.2f%%</td></tr>`,
			h.Code, h.Name, h.Quantity, formatRupiahShort(h.AvgPrice), formatRupiahShort(h.CurrentPrice),
			formatRupiahShort(h.MarketValue), gainClass, gainSign, formatRupiahShort(h.Gain), gainClass, gainSign, h.GainPercent))
	}

	totalClass := "up"
	totalSign := "+"
	if totalGain < 0 {
		totalClass = "down"
		totalSign = ""
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Laporan Portfolio %s | %s</title>
<style>
@page{size:A4 landscape;margin:12mm}@media print{body{-webkit-print-color-adjust:exact;print-color-adjust:exact}.no-print{display:none!important}}
*{box-sizing:border-box;margin:0;padding:0}body{font-family:'Inter',system-ui,sans-serif;font-size:12px;color:#1e293b;line-height:1.5;max-width:297mm;margin:0 auto;padding:16px;background:#fff}.header{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:3px solid #3b82f6;padding-bottom:12px;margin-bottom:20px}.header h1{font-size:22px;font-weight:700;color:#0f172a}.header .desc{font-size:13px;color:#64748b}.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:20px}.stat{background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px;padding:12px;text-align:center}.stat .label{font-size:10px;color:#64748b;text-transform:uppercase;letter-spacing:.04em}.stat .value{font-size:18px;font-weight:700;color:#0f172a;margin-top:4px}.up{color:#16a34a}.down{color:#dc2626}table{width:100%%;border-collapse:collapse}td,th{padding:7px 10px;text-align:left;border-bottom:1px solid #e2e8f0}th{background:#f8fafc;font-weight:600;color:#475569;font-size:10px;text-transform:uppercase;letter-spacing:.04em}.footer{margin-top:24px;padding-top:10px;border-top:1px solid #e2e8f0;text-align:center;color:#94a3b8;font-size:10px}.print-btn{position:fixed;top:12px;right:12px;background:#3b82f6;color:#fff;border:none;padding:8px 20px;border-radius:6px;font-size:12px;font-weight:600;cursor:pointer;z-index:100}.print-btn:hover{background:#2563eb}
</style>
</head>
<body>
<button class="print-btn no-print" onclick="window.print()">Cetak PDF</button>
<div class="header">
  <div>
    <h1>%s</h1>
    <div class="desc">%s</div>
  </div>
</div>
<div class="stats">
  <div class="stat"><div class="label">Total Nilai</div><div class="value">Rp %s</div></div>
  <div class="stat"><div class="label">Total Modal</div><div class="value">Rp %s</div></div>
  <div class="stat"><div class="label">Gain/Loss</div><div class="value %s">%sRp %s</div></div>
  <div class="stat"><div class="label">Return</div><div class="value %s">%s%.2f%%</div></div>
</div>
<table>
  <tr><th>Kode</th><th>Nama</th><th>Jumlah</th><th>Harga Rata</th><th>Harga Saat Ini</th><th>Nilai Pasar</th><th>Gain/Loss</th><th>G/L%%</th></tr>
  %s
</table>
<div class="footer">
  <p>Laporan digenerate oleh %s &mdash; Platform Analisa Saham & Forex</p>
  <p>Disclaimer: Laporan ini bukan rekomendasi investasi.</p>
</div>
</body>
</html>`, portfolio.Name, s.AppName,
		portfolio.Name, portfolio.Description,
		formatRupiahShort(totalValue), formatRupiahShort(totalCost),
		totalClass, totalSign, formatRupiahShort(totalGain),
		totalClass, totalSign, totalGainPercent,
		holdingRows.String(),
		s.AppName)

	return []byte(html), nil
}

type PortfolioHoldingExport struct {
	Code          string
	Name          string
	Quantity      float64
	AvgPrice      float64
	CurrentPrice  float64
	MarketValue   float64
	Gain          float64
	GainPercent   float64
}

func formatRupiahShort(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

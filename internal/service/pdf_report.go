package service

import (
	"fmt"
	"strings"
	"text/template"

	"investo/internal/model"
)

type PDFReportService struct{}

type StockReportData struct {
	Stock        model.Stock
	Price        float64
	Change       float64
	Fundamental  *model.StockFundamental
	Valuation    *ValuationResult
	Peers        []PeerData
	IndustryAvg  map[string]float64
}

type PortfolioReportData struct {
	Portfolio    model.Portfolio
	Performance  *PortfolioPerformance
	Attribution  *Attribution
	Holdings     []HoldingPerformance
}

const stockReportHTML = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Laporan Saham {{.Stock.Code}} - {{.Stock.Name}}</title>
<style>
  @page { size: A4; margin: 20mm 15mm; }
  @media print {
    body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    .page-break { page-break-after: always; }
    .no-print { display: none !important; }
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Segoe UI', system-ui, -apple-system, sans-serif; font-size: 13px; color: #1e293b; line-height: 1.6; max-width: 210mm; margin: 0 auto; padding: 20px; background: #fff; }
  .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 3px solid #3b82f6; padding-bottom: 16px; margin-bottom: 24px; }
  .header-left h1 { font-size: 26px; font-weight: 700; color: #0f172a; }
  .header-left .code { font-size: 14px; color: #64748b; }
  .header-right { text-align: right; }
  .header-right .price { font-size: 32px; font-weight: 700; color: #0f172a; }
  .header-right .change { font-size: 16px; font-weight: 600; }
  .up { color: #16a34a; }
  .down { color: #dc2626; }
  .section { margin-bottom: 28px; }
  .section h2 { font-size: 18px; font-weight: 600; color: #1e40af; border-bottom: 2px solid #e2e8f0; padding-bottom: 8px; margin-bottom: 16px; }
  table { width: 100%; border-collapse: collapse; }
  table th, table td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #e2e8f0; }
  table th { background: #f8fafc; font-weight: 600; color: #475569; font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em; }
  .metric-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
  .metric-card { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 14px; text-align: center; }
  .metric-card .label { font-size: 11px; color: #64748b; text-transform: uppercase; letter-spacing: 0.04em; margin-bottom: 4px; }
  .metric-card .value { font-size: 20px; font-weight: 700; color: #0f172a; }
  .valuation-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
  .val-card { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 14px; }
  .val-card .label { font-size: 12px; color: #64748b; }
  .val-card .value { font-size: 18px; font-weight: 700; color: #0f172a; margin-top: 4px; }
  .recommendation { display: inline-block; background: #1e40af; color: white; padding: 8px 24px; border-radius: 6px; font-weight: 700; font-size: 16px; }
  .recommendation.buy { background: #16a34a; }
  .recommendation.sell { background: #dc2626; }
  .footer { margin-top: 40px; padding-top: 16px; border-top: 1px solid #e2e8f0; text-align: center; color: #94a3b8; font-size: 11px; }
  .print-btn { position: fixed; top: 20px; right: 20px; background: #3b82f6; color: white; border: none; padding: 10px 24px; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; z-index: 100; }
  .print-btn:hover { background: #2563eb; }
</style>
</head>
<body>
<button class="print-btn no-print" onclick="window.print()">Cetak PDF</button>

<div class="header">
  <div class="header-left">
    <div class="code">{{.Stock.Code}}</div>
    <h1>{{.Stock.Name}}</h1>
  </div>
  <div class="header-right">
    <div class="price">Rp {{formatPrice .Price}}</div>
    <div class="change {{if ge .Change 0}}up{{else}}down{{end}}">
      {{if ge .Change 0}}+{{end}}{{.Change}}%
    </div>
  </div>
</div>

<div class="section">
  <h2>Metrik Utama</h2>
  <div class="metric-grid">
    {{if .Fundamental}}
    <div class="metric-card">
      <div class="label">PER</div>
      <div class="value">{{formatFloat .Fundamental.PER}}x</div>
    </div>
    <div class="metric-card">
      <div class="label">PBV</div>
      <div class="value">{{formatFloat .Fundamental.PBV}}x</div>
    </div>
    <div class="metric-card">
      <div class="label">ROE</div>
      <div class="value">{{formatFloat .Fundamental.ROE}}%</div>
    </div>
    <div class="metric-card">
      <div class="label">DER</div>
      <div class="value">{{formatFloat .Fundamental.DER}}x</div>
    </div>
    <div class="metric-card">
      <div class="label">EPS</div>
      <div class="value">Rp {{formatPrice .Fundamental.EPS}}</div>
    </div>
    <div class="metric-card">
      <div class="label">BVPS</div>
      <div class="value">Rp {{formatPrice .Fundamental.BVPS}}</div>
    </div>
    <div class="metric-card">
      <div class="label">NPM</div>
      <div class="value">{{formatFloat .Fundamental.NetProfitMargin}}%</div>
    </div>
    <div class="metric-card">
      <div class="label">Div Yield</div>
      <div class="value">{{formatFloat .Fundamental.DividendYield}}%</div>
    </div>
    {{else}}
    <p style="grid-column: 1/-1; color: #94a3b8; text-align: center; padding: 20px;">Data fundamental belum tersedia</p>
    {{end}}
  </div>
</div>

{{if .Fundamental}}
<div class="section">
  <h2>Analisa Fundamental</h2>
  <table>
    <tr><th style="width: 200px;">Indikator</th><th>Nilai</th><th>Interpretasi</th></tr>
    <tr><td>Revenue</td><td>Rp {{formatPrice .Fundamental.Revenue}}</td><td>Pendapatan total perusahaan</td></tr>
    <tr><td>Net Income</td><td>Rp {{formatPrice .Fundamental.NetIncome}}</td><td>Laba bersih setelah pajak</td></tr>
    <tr><td>EPS</td><td>Rp {{formatPrice .Fundamental.EPS}}</td><td>Laba per lembar saham</td></tr>
    <tr><td>Total Assets</td><td>Rp {{formatPrice .Fundamental.TotalAssets}}</td><td>Total aset perusahaan</td></tr>
    <tr><td>Total Liabilities</td><td>Rp {{formatPrice .Fundamental.TotalLiabilities}}</td><td>Total kewajiban perusahaan</td></tr>
    <tr><td>Equity</td><td>Rp {{formatPrice .Fundamental.Equity}}</td><td>Ekuitas perusahaan</td></tr>
    <tr><td>PER</td><td>{{formatFloat .Fundamental.PER}}x</td><td>Rasio harga terhadap laba{{if gt .Fundamental.PER 25.0}} — di atas rata-rata{{end}}</td></tr>
    <tr><td>PBV</td><td>{{formatFloat .Fundamental.PBV}}x</td><td>Rasio harga terhadap buku{{if lt .Fundamental.PBV 1.0}} — undervalued{{end}}</td></tr>
    <tr><td>ROE</td><td>{{formatFloat .Fundamental.ROE}}%</td><td>Return on equity{{if ge .Fundamental.ROE 15.0}} — bagus{{end}}</td></tr>
    <tr><td>DER</td><td>{{formatFloat .Fundamental.DER}}x</td><td>Rasio utang terhadap ekuitas{{if le .Fundamental.DER 1.0}} — sehat{{end}}</td></tr>
  </table>
</div>
{{end}}

{{if .Valuation}}
<div class="section">
  <h2>Valuasi Saham</h2>
  <div class="valuation-grid">
    <div class="val-card">
      <div class="label">DCF Value</div>
      <div class="value">Rp {{formatPrice .Valuation.DCFValue}}</div>
    </div>
    <div class="val-card">
      <div class="label">Graham Value</div>
      <div class="value">Rp {{formatPrice .Valuation.GrahamValue}}</div>
    </div>
    <div class="val-card">
      <div class="label">Lynch Value</div>
      <div class="value">Rp {{formatPrice .Valuation.LynchValue}}</div>
    </div>
    <div class="val-card">
      <div class="label">PBV Value</div>
      <div class="value">Rp {{formatPrice .Valuation.PBVValue}}</div>
    </div>
    <div class="val-card">
      <div class="label">Average Target</div>
      <div class="value">Rp {{formatPrice .Valuation.AverageTarget}}</div>
    </div>
    <div class="val-card">
      <div class="label">Upside/Downside</div>
      <div class="value">{{formatFloat .Valuation.UpsidePercent}}%</div>
    </div>
  </div>
  <div style="margin-top: 16px; text-align: center;">
    <span class="recommendation {{if eq .Valuation.Recommendation "BUY"}}buy{{else if eq .Valuation.Recommendation "SELL"}}sell{{end}}">
      {{.Valuation.Recommendation}}
    </span>
  </div>
</div>
{{end}}

{{if .Peers}}
<div class="section page-break">
  <h2>Perbandingan dengan Kompetitor</h2>
  <table>
    <tr>
      <th>Kode</th>
      <th>Nama</th>
      <th>PER</th>
      <th>PBV</th>
      <th>ROE</th>
      <th>DER</th>
      <th>NPM</th>
      <th>Div Yield</th>
    </tr>
    {{range .Peers}}
    <tr>
      <td style="font-weight: 600;">{{.Code}}</td>
      <td>{{.Name}}</td>
      <td>{{formatFloat .PER}}x</td>
      <td>{{formatFloat .PBV}}x</td>
      <td>{{formatFloat .ROE}}%</td>
      <td>{{formatFloat .DER}}x</td>
      <td>{{formatFloat .NPM}}%</td>
      <td>{{formatFloat .DivYield}}%</td>
    </tr>
    {{end}}
  </table>
</div>
{{end}}

<div class="footer">
  <p>Laporan ini digenerate oleh Investo &mdash; Platform Analisa Saham & Forex</p>
  <p>Disclaimer: Laporan ini bukan rekomendasi investasi. Selalu lakukan riset sendiri.</p>
</div>
</body>
</html>`

const portfolioReportHTML = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Laporan Portfolio - {{.Portfolio.Name}}</title>
<style>
  @page { size: A4; margin: 20mm 15mm; }
  @media print {
    body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    .page-break { page-break-after: always; }
    .no-print { display: none !important; }
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Segoe UI', system-ui, -apple-system, sans-serif; font-size: 13px; color: #1e293b; line-height: 1.6; max-width: 210mm; margin: 0 auto; padding: 20px; background: #fff; }
  .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 3px solid #3b82f6; padding-bottom: 16px; margin-bottom: 24px; }
  .header h1 { font-size: 26px; font-weight: 700; color: #0f172a; }
  .header .desc { font-size: 14px; color: #64748b; margin-top: 4px; }
  .stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 28px; }
  .stat-card { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 14px; text-align: center; }
  .stat-card .label { font-size: 11px; color: #64748b; text-transform: uppercase; letter-spacing: 0.04em; margin-bottom: 4px; }
  .stat-card .value { font-size: 20px; font-weight: 700; color: #0f172a; }
  .up { color: #16a34a; }
  .down { color: #dc2626; }
  .section { margin-bottom: 28px; }
  .section h2 { font-size: 18px; font-weight: 600; color: #1e40af; border-bottom: 2px solid #e2e8f0; padding-bottom: 8px; margin-bottom: 16px; }
  table { width: 100%; border-collapse: collapse; }
  table th, table td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #e2e8f0; }
  table th { background: #f8fafc; font-weight: 600; color: #475569; font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em; }
  .footer { margin-top: 40px; padding-top: 16px; border-top: 1px solid #e2e8f0; text-align: center; color: #94a3b8; font-size: 11px; }
  .print-btn { position: fixed; top: 20px; right: 20px; background: #3b82f6; color: white; border: none; padding: 10px 24px; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; z-index: 100; }
  .print-btn:hover { background: #2563eb; }
</style>
</head>
<body>
<button class="print-btn no-print" onclick="window.print()">Cetak PDF</button>

<div class="header">
  <div>
    <h1>{{.Portfolio.Name}}</h1>
    <div class="desc">{{.Portfolio.Description}}</div>
  </div>
</div>

{{if .Performance}}
<div class="stats">
  <div class="stat-card">
    <div class="label">Total Nilai</div>
    <div class="value">Rp {{formatPrice .Performance.TotalValue}}</div>
  </div>
  <div class="stat-card">
    <div class="label">Total Modal</div>
    <div class="value">Rp {{formatPrice .Performance.TotalCost}}</div>
  </div>
  <div class="stat-card">
    <div class="label">Gain/Loss</div>
    <div class="value {{if ge .Performance.TotalGain 0}}up{{else}}down{{end}}">
      Rp {{formatPrice .Performance.TotalGain}}
    </div>
  </div>
  <div class="stat-card">
    <div class="label">Return</div>
    <div class="value {{if ge .Performance.TotalGainPercent 0}}up{{else}}down{{end}}">
      {{if ge .Performance.TotalGainPercent 0}}+{{end}}{{formatFloat .Performance.TotalGainPercent}}%
    </div>
  </div>
</div>

<div class="section">
  <h2>Holdings</h2>
  <table>
    <tr>
      <th>Kode</th>
      <th>Nama</th>
      <th>Jumlah</th>
      <th>Harga Rata</th>
      <th>Harga Saat Ini</th>
      <th>Nilai Pasar</th>
      <th>Gain/Loss</th>
      <th>G/L%</th>
      <th>Bobot</th>
    </tr>
    {{range .Performance.Holdings}}
    <tr>
      <td style="font-weight: 600;">{{.Stock.Code}}</td>
      <td>{{.Stock.Name}}</td>
      <td>{{formatFloat .Quantity}}</td>
      <td>Rp {{formatPrice .AvgPrice}}</td>
      <td>Rp {{formatPrice .CurrentPrice}}</td>
      <td>Rp {{formatPrice .MarketValue}}</td>
      <td class="{{if ge .Gain 0}}up{{else}}down{{end}}">Rp {{formatPrice .Gain}}</td>
      <td class="{{if ge .GainPercent 0}}up{{else}}down{{end}}">{{if ge .GainPercent 0}}+{{end}}{{formatFloat .GainPercent}}%</td>
      <td>{{formatFloat .Weight}}%</td>
    </tr>
    {{end}}
  </table>
</div>
{{end}}

{{if .Attribution}}
<div class="section page-break">
  <h2>Alokasi Sektor</h2>
  {{if .Attribution.SectorAllocation}}
  <table>
    <tr><th>Sektor</th><th>Alokasi (%)</th></tr>
    {{range $sector, $weight := .Attribution.SectorAllocation}}
    <tr><td>{{$sector}}</td><td>{{formatFloat $weight}}%</td></tr>
    {{end}}
  </table>
  {{end}}
</div>

{{if .Attribution.TopGainers}}
<div class="section">
  <h2>Top Gainers</h2>
  <table>
    <tr><th>Kode</th><th>Nama</th><th>Gain/Loss%</th></tr>
    {{range .Attribution.TopGainers}}
    <tr><td style="font-weight: 600;">{{.Stock.Code}}</td><td>{{.Stock.Name}}</td><td class="up">+{{formatFloat .GainPercent}}%</td></tr>
    {{end}}
  </table>
</div>
{{end}}

{{if .Attribution.TopLosers}}
<div class="section">
  <h2>Top Losers</h2>
  <table>
    <tr><th>Kode</th><th>Nama</th><th>Gain/Loss%</th></tr>
    {{range .Attribution.TopLosers}}
    <tr><td style="font-weight: 600;">{{.Stock.Code}}</td><td>{{.Stock.Name}}</td><td class="down">{{formatFloat .GainPercent}}%</td></tr>
    {{end}}
  </table>
</div>
{{end}}

<div class="section">
  <h2>Statistik Tambahan</h2>
  <table>
    <tr><th style="width: 200px;">Metrik</th><th>Nilai</th></tr>
    <tr><td>Win Rate</td><td>{{formatFloat .Attribution.WinRate}}%</td></tr>
    {{if .Attribution.BestDay}}<tr><td>Best Day</td><td>{{.Attribution.BestDay}}</td></tr>{{end}}
    {{if .Attribution.WorstDay}}<tr><td>Worst Day</td><td>{{.Attribution.WorstDay}}</td></tr>{{end}}
  </table>
</div>
{{end}}

<div class="footer">
  <p>Laporan ini digenerate oleh Investo &mdash; Platform Analisa Saham & Forex</p>
  <p>Disclaimer: Laporan ini bukan rekomendasi investasi. Selalu lakukan riset sendiri.</p>
</div>
</body>
</html>`

func (s *PDFReportService) GenerateStockReport(stock model.Stock, fundamental *model.StockFundamental, valuation *ValuationResult, peers []PeerData, industryAvg map[string]float64) ([]byte, error) {
	price := 0.0
	if valuation != nil {
		price = valuation.CurrentPrice
	}
	change := 0.0
	if valuation != nil {
		change = valuation.UpsidePercent
	}

	funcMap := template.FuncMap{
		"formatPrice": func(v float64) string {
			return formatRupiah(v)
		},
		"formatFloat": func(v float64) string {
			return fmt.Sprintf("%.2f", v)
		},
	}

	tpl, err := template.New("stock_report").Funcs(funcMap).Parse(stockReportHTML)
	if err != nil {
		return nil, fmt.Errorf("PDFReportService.GenerateStockReport parse: %w", err)
	}

	data := map[string]interface{}{
		"Stock":        stock,
		"Price":        price,
		"Change":       change,
		"Fundamental":  fundamental,
		"Valuation":    valuation,
		"Peers":        peers,
		"IndustryAvg":  industryAvg,
	}

	var buf strings.Builder
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("PDFReportService.GenerateStockReport execute: %w", err)
	}

	return []byte(buf.String()), nil
}

func (s *PDFReportService) GeneratePortfolioReport(portfolio model.Portfolio, performance *PortfolioPerformance, attribution *Attribution) ([]byte, error) {
	funcMap := template.FuncMap{
		"formatPrice": func(v float64) string {
			return formatRupiah(v)
		},
		"formatFloat": func(v float64) string {
			return fmt.Sprintf("%.2f", v)
		},
	}

	tpl, err := template.New("portfolio_report").Funcs(funcMap).Parse(portfolioReportHTML)
	if err != nil {
		return nil, fmt.Errorf("PDFReportService.GeneratePortfolioReport parse: %w", err)
	}

	data := map[string]interface{}{
		"Portfolio":   portfolio,
		"Performance": performance,
		"Attribution": attribution,
	}

	var buf strings.Builder
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("PDFReportService.GeneratePortfolioReport execute: %w", err)
	}

	return []byte(buf.String()), nil
}

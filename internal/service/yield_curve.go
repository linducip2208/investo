package service

type YieldPoint struct {
	Tenor string  `json:"tenor"`
	Rate  float64 `json:"rate"`
}

type CurveAnalysis struct {
	Shape       string  `json:"shape"`
	Spread      float64 `json:"spread"`
	Signal      string  `json:"signal"`
	Implication string  `json:"implication"`
}

type YieldCurveService struct{}

func NewYieldCurveService() *YieldCurveService {
	return &YieldCurveService{}
}

func (s *YieldCurveService) GetYieldCurve() ([]YieldPoint, error) {
	points := []YieldPoint{
		{Tenor: "1M", Rate: 5.30},
		{Tenor: "3M", Rate: 5.55},
		{Tenor: "6M", Rate: 5.72},
		{Tenor: "1Y", Rate: 5.85},
		{Tenor: "2Y", Rate: 6.10},
		{Tenor: "5Y", Rate: 6.45},
		{Tenor: "10Y", Rate: 6.85},
		{Tenor: "15Y", Rate: 7.05},
		{Tenor: "20Y", Rate: 7.15},
		{Tenor: "30Y", Rate: 7.20},
	}
	return points, nil
}

func (s *YieldCurveService) AnalyzeCurve() (*CurveAnalysis, error) {
	curve, _ := s.GetYieldCurve()

	var rate2Y, rate10Y float64
	for _, p := range curve {
		if p.Tenor == "2Y" {
			rate2Y = p.Rate
		}
		if p.Tenor == "10Y" {
			rate10Y = p.Rate
		}
	}

	spread := rate10Y - rate2Y

	var shape, signal, implication string
	if spread > 1.0 {
		shape = "Normal (Steep)"
		signal = "expansion"
		implication = "Kurva yield curam menandakan ekspektasi pertumbuhan ekonomi yang kuat. Bank mendapat margin bunga lebar (lend long, borrow short). Sektor perbankan dan properti cenderung outperform. Investor asing tertarik masuk karena carry trade."
	} else if spread > 0.3 {
		shape = "Normal"
		signal = "neutral"
		implication = "Kurva yield normal menandakan ekspektasi ekonomi stabil. Tidak ada tekanan resesi jangka pendek. Pasar obligasi dan saham dalam keseimbangan. Fokus pada fundamental perusahaan."
	} else if spread >= 0 {
		shape = "Flat"
		signal = "neutral"
		implication = "Kurva yield datar adalah sinyal transisi. Pasar mengantisipasi perubahan kebijakan moneter. Investor sebaiknya defensif: kurangi cyclical, tambah defensif (consumer staples, healthcare)."
	} else {
		shape = "Inverted"
		signal = "recession risk"
		implication = "Kurva yield terbalik (inverted) adalah sinyal resesi klasik. Pasar obligasi memprediksi pemotongan suku bunga agresif. Investor harus defensif: pindah ke cash, obligasi jangka pendek, dan saham defensif (consumer staples, utilities, healthcare)."
	}

	return &CurveAnalysis{
		Shape:       shape,
		Spread:      spread,
		Signal:      signal,
		Implication: implication,
	}, nil
}

package service

type TrendCorrelation struct {
	Keyword       string  `json:"keyword"`
	Correlation   float64 `json:"correlation"`
	Direction     string  `json:"direction"`
	Significance  string  `json:"significance"`
	Description   string  `json:"description"`
	InterestData  []TrendPoint `json:"interest_data"`
	IHSGData      []TrendPoint `json:"ihsg_data"`
}

type TrendPoint struct {
	Date   string  `json:"date"`
	Value  float64 `json:"value"`
}

type GoogleTrendsService struct{}

func (s *GoogleTrendsService) GetTrendCorrelations() ([]TrendCorrelation, error) {
	return []TrendCorrelation{
		{
			Keyword:      "beli saham",
			Correlation:  0.72,
			Direction:    "positive",
			Significance: "high",
			Description:  "Pencarian 'beli saham' berkorelasi kuat dengan pergerakan IHSG. Lonjakan pencarian biasanya mendahului rally pasar 1-2 minggu.",
			InterestData: []TrendPoint{
				{Date: "Jan", Value: 45}, {Date: "Feb", Value: 52}, {Date: "Mar", Value: 48},
				{Date: "Apr", Value: 58}, {Date: "May", Value: 55}, {Date: "Jun", Value: 62},
				{Date: "Jul", Value: 70}, {Date: "Aug", Value: 65}, {Date: "Sep", Value: 60},
				{Date: "Oct", Value: 72}, {Date: "Nov", Value: 68}, {Date: "Dec", Value: 75},
			},
			IHSGData: []TrendPoint{
				{Date: "Jan", Value: 6850}, {Date: "Feb", Value: 6900}, {Date: "Mar", Value: 6880},
				{Date: "Apr", Value: 6950}, {Date: "May", Value: 6920}, {Date: "Jun", Value: 7000},
				{Date: "Jul", Value: 7120}, {Date: "Aug", Value: 7080}, {Date: "Sep", Value: 7050},
				{Date: "Oct", Value: 7180}, {Date: "Nov", Value: 7150}, {Date: "Dec", Value: 7250},
			},
		},
		{
			Keyword:      "BBCA",
			Correlation:  0.65,
			Direction:    "positive",
			Significance: "medium",
			Description:  "Volume pencarian 'BBCA' meningkat signifikan saat saham BBCA mendekati rekor tertinggi. Investor ritel mencari informasi sebelum membeli.",
			InterestData: []TrendPoint{
				{Date: "Jan", Value: 60}, {Date: "Feb", Value: 55}, {Date: "Mar", Value: 65},
				{Date: "Apr", Value: 70}, {Date: "May", Value: 62}, {Date: "Jun", Value: 75},
				{Date: "Jul", Value: 80}, {Date: "Aug", Value: 72}, {Date: "Sep", Value: 68},
				{Date: "Oct", Value: 85}, {Date: "Nov", Value: 78}, {Date: "Dec", Value: 90},
			},
			IHSGData: []TrendPoint{
				{Date: "Jan", Value: 9200}, {Date: "Feb", Value: 9150}, {Date: "Mar", Value: 9300},
				{Date: "Apr", Value: 9450}, {Date: "May", Value: 9350}, {Date: "Jun", Value: 9500},
				{Date: "Jul", Value: 9600}, {Date: "Aug", Value: 9550}, {Date: "Sep", Value: 9480},
				{Date: "Oct", Value: 9700}, {Date: "Nov", Value: 9650}, {Date: "Dec", Value: 9800},
			},
		},
		{
			Keyword:      "resesi",
			Correlation:  -0.45,
			Direction:    "inverse",
			Significance: "medium",
			Description:  "Lonjakan pencarian 'resesi' berkorelasi negatif dengan IHSG. Sentimen fear-driven menyebabkan aksi jual saat ketakutan resesi meningkat.",
			InterestData: []TrendPoint{
				{Date: "Jan", Value: 30}, {Date: "Feb", Value: 25}, {Date: "Mar", Value: 35},
				{Date: "Apr", Value: 28}, {Date: "May", Value: 40}, {Date: "Jun", Value: 55},
				{Date: "Jul", Value: 50}, {Date: "Aug", Value: 32}, {Date: "Sep", Value: 28},
				{Date: "Oct", Value: 38}, {Date: "Nov", Value: 42}, {Date: "Dec", Value: 35},
			},
			IHSGData: []TrendPoint{
				{Date: "Jan", Value: 7100}, {Date: "Feb", Value: 7150}, {Date: "Mar", Value: 7050},
				{Date: "Apr", Value: 7120}, {Date: "May", Value: 6950}, {Date: "Jun", Value: 6800},
				{Date: "Jul", Value: 6850}, {Date: "Aug", Value: 7050}, {Date: "Sep", Value: 7100},
				{Date: "Oct", Value: 6980}, {Date: "Nov", Value: 6900}, {Date: "Dec", Value: 7000},
			},
		},
		{
			Keyword:      "bitcoin",
			Correlation:  0.32,
			Direction:    "positive",
			Significance: "low",
			Description:  "Korelasi lemah antara pencarian Bitcoin dan IHSG. Minat crypto kadang parallel dengan risk appetite pasar saham, terutama di kalangan investor muda.",
			InterestData: []TrendPoint{
				{Date: "Jan", Value: 90}, {Date: "Feb", Value: 85}, {Date: "Mar", Value: 95},
				{Date: "Apr", Value: 80}, {Date: "May", Value: 88}, {Date: "Jun", Value: 92},
				{Date: "Jul", Value: 100}, {Date: "Aug", Value: 95}, {Date: "Sep", Value: 85},
				{Date: "Oct", Value: 98}, {Date: "Nov", Value: 100}, {Date: "Dec", Value: 95},
			},
			IHSGData: []TrendPoint{
				{Date: "Jan", Value: 6900}, {Date: "Feb", Value: 7000}, {Date: "Mar", Value: 7100},
				{Date: "Apr", Value: 7050}, {Date: "May", Value: 6980}, {Date: "Jun", Value: 7120},
				{Date: "Jul", Value: 7200}, {Date: "Aug", Value: 7150}, {Date: "Sep", Value: 7080},
				{Date: "Oct", Value: 7250}, {Date: "Nov", Value: 7300}, {Date: "Dec", Value: 7280},
			},
		},
	}, nil
}

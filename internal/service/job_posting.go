package service

type JobPostingSignal struct {
	StockCode     string `json:"stock_code"`
	StockName     string `json:"stock_name"`
	PostingCount  int    `json:"posting_count"`
	ChangeQoQ     float64 `json:"change_qoq"`
	Signal        string `json:"signal"`
	Implication   string `json:"implication"`
	Sector        string `json:"sector"`
}

type JobPostingService struct{}

func (s *JobPostingService) GetJobSignals() ([]JobPostingSignal, error) {
	return []JobPostingSignal{
		{
			StockCode:    "BBCA",
			StockName:    "Bank Central Asia Tbk",
			PostingCount: 2450,
			ChangeQoQ:    15.0,
			Signal:       "expansion",
			Implication:  "BBCA aggressively hiring across digital banking and IT roles. Expansion signal indicates confidence in future growth and digital transformation investment.",
			Sector:       "Perbankan",
		},
		{
			StockCode:    "TLKM",
			StockName:    "Telkom Indonesia Tbk",
			PostingCount: 1820,
			ChangeQoQ:    8.0,
			Signal:       "stable_growth",
			Implication:  "Steady hiring in data center, cloud, and cybersecurity roles. Reflects Telkom's pivot toward digital infrastructure and B2B services.",
			Sector:       "Telekomunikasi",
		},
		{
			StockCode:    "GOTO",
			StockName:    "GoTo Gojek Tokopedia Tbk",
			PostingCount: 520,
			ChangeQoQ:    -20.0,
			Signal:       "restructuring",
			Implication:  "Significant reduction in job postings suggests ongoing cost optimization. Focus shifting to profitability over growth, fewer new initiatives.",
			Sector:       "Teknologi",
		},
		{
			StockCode:    "ASII",
			StockName:    "Astra International Tbk",
			PostingCount: 1680,
			ChangeQoQ:    6.0,
			Signal:       "stable_growth",
			Implication:  "Consistent hiring across automotive, heavy equipment, and financial services. Diversified conglomerate maintaining steady workforce expansion.",
			Sector:       "Konglomerasi",
		},
		{
			StockCode:    "UNVR",
			StockName:    "Unilever Indonesia Tbk",
			PostingCount: 340,
			ChangeQoQ:    -5.0,
			Signal:       "contraction",
			Implication:  "Slight decline in hiring amid consumer spending pressure. Cost-saving measures and automation reducing headcount needs.",
			Sector:       "Konsumer",
		},
		{
			StockCode:    "ADRO",
			StockName:    "Adaro Energy Indonesia Tbk",
			PostingCount: 890,
			ChangeQoQ:    12.0,
			Signal:       "expansion",
			Implication:  "Strong hiring for renewable energy and green aluminum projects. Diversification beyond coal driving new talent acquisition.",
			Sector:       "Energi",
		},
		{
			StockCode:    "ISAT",
			StockName:    "Indosat Ooredoo Hutchison Tbk",
			PostingCount: 720,
			ChangeQoQ:    10.0,
			Signal:       "stable_growth",
			Implication:  "Post-merger integration complete, now hiring for 5G expansion and enterprise solutions. Steady growth trajectory.",
			Sector:       "Telekomunikasi",
		},
		{
			StockCode:    "BUKA",
			StockName:    "Bukalapak.com Tbk",
			PostingCount: 180,
			ChangeQoQ:    -35.0,
			Signal:       "contraction",
			Implication:  "Sharp decline in job postings indicates major restructuring. Focus on core profitable segments, divesting non-core units.",
			Sector:       "Teknologi",
		},
		{
			StockCode:    "BRIS",
			StockName:    "Bank Syariah Indonesia Tbk",
			PostingCount: 1560,
			ChangeQoQ:    22.0,
			Signal:       "expansion",
			Implication:  "Aggressive expansion in sharia banking. Hiring across branch operations, digital banking, and SME lending to capture growing halal economy.",
			Sector:       "Perbankan",
		},
		{
			StockCode:    "MDKA",
			StockName:    "Merdeka Copper Gold Tbk",
			PostingCount: 450,
			ChangeQoQ:    18.0,
			Signal:       "expansion",
			Implication:  "Rapid hiring for mining operations expansion and downstream processing. Nickel and gold projects driving workforce growth.",
			Sector:       "Pertambangan",
		},
	}, nil
}

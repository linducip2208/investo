package service

import (
	"sort"
	"time"
)

type InsiderTransaction struct {
	StockCode       string `json:"stock_code"`
	StockName       string `json:"stock_name"`
	InsiderName     string `json:"insider_name"`
	Position        string `json:"position"`
	TransactionType string `json:"transaction_type"`
	Quantity        int64  `json:"quantity"`
	Price           float64 `json:"price"`
	Value           float64 `json:"value"`
	Date            string `json:"date"`
	Significance    string `json:"significance"`
}

type InsiderService struct{}

func (s *InsiderService) GetRecentTransactions(limit int) []InsiderTransaction {
	demo := []InsiderTransaction{
		{
			StockCode: "BBCA", StockName: "Bank Central Asia Tbk",
			InsiderName: "Djohan Emir Setijoso", Position: "Komisaris Utama",
			TransactionType: "buy", Quantity: 500000, Price: 10250, Value: 5125000000,
			Date: time.Now().AddDate(0, 0, -3).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "BBRI", StockName: "Bank Rakyat Indonesia Tbk",
			InsiderName: "Sunarso", Position: "Direktur Utama",
			TransactionType: "buy", Quantity: 250000, Price: 5750, Value: 1437500000,
			Date: time.Now().AddDate(0, 0, -5).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "TLKM", StockName: "Telkom Indonesia Tbk",
			InsiderName: "Ririek Adriansyah", Position: "Direktur Utama",
			TransactionType: "buy", Quantity: 100000, Price: 3980, Value: 398000000,
			Date: time.Now().AddDate(0, 0, -7).Format("02 Jan 2006"), Significance: "medium",
		},
		{
			StockCode: "ASII", StockName: "Astra International Tbk",
			InsiderName: "Djony Bunarto Tjondro", Position: "Direktur Utama",
			TransactionType: "sell", Quantity: 75000, Price: 6800, Value: 510000000,
			Date: time.Now().AddDate(0, 0, -10).Format("02 Jan 2006"), Significance: "medium",
		},
		{
			StockCode: "UNVR", StockName: "Unilever Indonesia Tbk",
			InsiderName: "Benjie Yap", Position: "Direktur",
			TransactionType: "sell", Quantity: 200000, Price: 4200, Value: 840000000,
			Date: time.Now().AddDate(0, 0, -14).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "ADRO", StockName: "Adaro Energy Indonesia Tbk",
			InsiderName: "Garibaldi Thohir", Position: "Komisaris Utama",
			TransactionType: "buy", Quantity: 1000000, Price: 3800, Value: 3800000000,
			Date: time.Now().AddDate(0, 0, -2).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "BMRI", StockName: "Bank Mandiri Tbk",
			InsiderName: "Darmawan Junaidi", Position: "Direktur Utama",
			TransactionType: "buy", Quantity: 150000, Price: 7200, Value: 1080000000,
			Date: time.Now().AddDate(0, 0, -4).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "ICBP", StockName: "Indofood CBP Sukses Makmur Tbk",
			InsiderName: "Anthoni Salim", Position: "Komisaris Utama",
			TransactionType: "sell", Quantity: 300000, Price: 11900, Value: 3570000000,
			Date: time.Now().AddDate(0, 0, -8).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "KLBF", StockName: "Kalbe Farma Tbk",
			InsiderName: "Vidjongtius", Position: "Direktur Utama",
			TransactionType: "buy", Quantity: 80000, Price: 1650, Value: 132000000,
			Date: time.Now().AddDate(0, 0, -6).Format("02 Jan 2006"), Significance: "low",
		},
		{
			StockCode: "GGRM", StockName: "Gudang Garam Tbk",
			InsiderName: "Susilo Wonowidjojo", Position: "Direktur Utama",
			TransactionType: "buy", Quantity: 50000, Price: 32500, Value: 1625000000,
			Date: time.Now().AddDate(0, 0, -1).Format("02 Jan 2006"), Significance: "medium",
		},
		{
			StockCode: "BRPT", StockName: "Barito Pacific Tbk",
			InsiderName: "Prajogo Pangestu", Position: "Komisaris Utama",
			TransactionType: "buy", Quantity: 5000000, Price: 1500, Value: 7500000000,
			Date: time.Now().AddDate(0, 0, -12).Format("02 Jan 2006"), Significance: "high",
		},
		{
			StockCode: "PGAS", StockName: "Perusahaan Gas Negara Tbk",
			InsiderName: "Arief Setiawan Handoko", Position: "Direktur Keuangan",
			TransactionType: "sell", Quantity: 40000, Price: 1800, Value: 72000000,
			Date: time.Now().AddDate(0, 0, -15).Format("02 Jan 2006"), Significance: "low",
		},
	}

	sort.Slice(demo, func(i, j int) bool {
		ti, _ := time.Parse("02 Jan 2006", demo[i].Date)
		tj, _ := time.Parse("02 Jan 2006", demo[j].Date)
		return ti.After(tj)
	})

	if limit > 0 && limit < len(demo) {
		return demo[:limit]
	}
	return demo
}

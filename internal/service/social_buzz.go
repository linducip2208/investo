package service

import (
	"fmt"
	"time"
)

type BuzzItem struct {
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Mentions    int     `json:"mentions"`
	Sentiment   string  `json:"sentiment"`
	Change24h   int     `json:"change_24h"`
	Unusual     bool    `json:"unusual"`
	TopSources  []string `json:"top_sources"`
	IsIllustrative bool `json:"is_illustrative"`
	Source      string `json:"source,omitempty"`
	AsOf        string `json:"as_of,omitempty"`
}

type SocialBuzzService struct{}

func (s *SocialBuzzService) GetStockBuzz(code string) (*BuzzItem, error) {
	all, _ := s.GetTrendingStocks()
	for _, b := range all {
		if b.StockCode == code {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("no buzz data for %s", code)
}

func (s *SocialBuzzService) GetTrendingStocks() ([]BuzzItem, error) {
	items := []BuzzItem{
		{StockCode: "BBCA", StockName: "Bank Central Asia Tbk", Mentions: 12500, Sentiment: "positive", Change24h: +15, Unusual: false, TopSources: []string{"Twitter/X", "Stockbit", "Telegram"}},
		{StockCode: "TLKM", StockName: "Telkom Indonesia Tbk", Mentions: 8900, Sentiment: "neutral", Change24h: -3, Unusual: false, TopSources: []string{"Twitter/X", "Instagram", "YouTube"}},
		{StockCode: "GOTO", StockName: "GoTo Gojek Tokopedia Tbk", Mentions: 18500, Sentiment: "negative", Change24h: +120, Unusual: true, TopSources: []string{"Twitter/X", "TikTok", "Stockbit"}},
		{StockCode: "ASII", StockName: "Astra International Tbk", Mentions: 5600, Sentiment: "positive", Change24h: +8, Unusual: false, TopSources: []string{"Twitter/X", "CNBC", "Telegram"}},
		{StockCode: "BRIS", StockName: "Bank Syariah Indonesia Tbk", Mentions: 7200, Sentiment: "positive", Change24h: +45, Unusual: false, TopSources: []string{"Twitter/X", "Stockbit", "Instagram"}},
		{StockCode: "ADRO", StockName: "Adaro Energy Indonesia Tbk", Mentions: 3400, Sentiment: "neutral", Change24h: -10, Unusual: false, TopSources: []string{"Twitter/X", "Telegram"}},
		{StockCode: "UNVR", StockName: "Unilever Indonesia Tbk", Mentions: 2800, Sentiment: "negative", Change24h: +25, Unusual: false, TopSources: []string{"Twitter/X", "Instagram"}},
		{StockCode: "BUKA", StockName: "Bukalapak.com Tbk", Mentions: 15200, Sentiment: "negative", Change24h: +250, Unusual: true, TopSources: []string{"Twitter/X", "TikTok", "Stockbit"}},
		{StockCode: "ANTM", StockName: "Aneka Tambang Tbk", Mentions: 6100, Sentiment: "positive", Change24h: +35, Unusual: false, TopSources: []string{"Twitter/X", "Telegram", "Stockbit"}},
		{StockCode: "MDKA", StockName: "Merdeka Copper Gold Tbk", Mentions: 4100, Sentiment: "neutral", Change24h: -5, Unusual: false, TopSources: []string{"Twitter/X", "Stockbit"}},
	}
	asOf := time.Now().Format("2006-01-02")
	for i := range items {
		items[i].IsIllustrative = true
		items[i].Source = "ilustratif — contoh, bukan data social listening real-time"
		items[i].AsOf = asOf
	}
	return items, nil
}

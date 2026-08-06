package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"investo/internal/model"
)

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.FixedZone("WIB", 7*3600)
	}
}

type YahooScraper struct {
	mu         sync.Mutex
	crumb      string
	cookieJar  *cookiejar.Jar
	crumbAge   time.Time
	httpClient *http.Client
}

func (s *YahooScraper) getClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	jar, _ := cookiejar.New(nil)
	s.cookieJar = jar
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
	return s.httpClient
}

func (s *YahooScraper) getCrumb() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.crumb != "" && time.Since(s.crumbAge) < 5*time.Minute {
		return s.crumb, nil
	}

	client := s.getClient()

	req, _ := http.NewRequest("GET", "https://fc.yahoo.com/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[YahooScraper] Cookie request failed: %v", err)
	} else {
		resp.Body.Close()
	}

	crumbReq, _ := http.NewRequest("GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	crumbReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	crumbResp, err := client.Do(crumbReq)
	if err != nil {
		return "", fmt.Errorf("crumb request failed: %w", err)
	}
	defer crumbResp.Body.Close()

	crumbBytes, err := io.ReadAll(crumbResp.Body)
	if err != nil {
		return "", fmt.Errorf("read crumb: %w", err)
	}

	s.crumb = strings.TrimSpace(string(crumbBytes))
	s.crumbAge = time.Now()
	return s.crumb, nil
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote    []yahooQuote    `json:"quote"`
				AdjClose []yahooAdjClose `json:"adjclose"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

type yahooQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []int64   `json:"volume"`
}

type yahooAdjClose struct {
	AdjClose []float64 `json:"adjclose"`
}

func (s *YahooScraper) FetchHistorical(stockCode string, start, end time.Time) ([]model.StockPrice, error) {
	crumb, err := s.getCrumb()
	if err != nil {
		log.Printf("[YahooScraper] WARNING: crumb unavailable, trying without: %v", err)
	}

	period1 := start.Unix()
	period2 := end.Unix()

	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&events=history&crumb=%s",
		stockCode, period1, period2, crumb,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo scraper: create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	client := s.getClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo scraper: fetch %s: %w", stockCode, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo scraper: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo scraper: unexpected status %d for %s: %s", resp.StatusCode, stockCode, string(body))
	}

	var chartResp yahooChartResponse
	if err := json.Unmarshal(body, &chartResp); err != nil {
		return nil, fmt.Errorf("yahoo scraper: parse json: %w", err)
	}

	if chartResp.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo scraper: chart error for %s: %v", stockCode, chartResp.Chart.Error)
	}

	if len(chartResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo scraper: no results for %s", stockCode)
	}

	result := chartResp.Chart.Result[0]
	timestamps := result.Timestamp

	quoteLen := len(result.Indicators.Quote)
	adjCloseLen := len(result.Indicators.AdjClose)

	if quoteLen == 0 || len(result.Indicators.Quote[0].Close) == 0 {
		return nil, fmt.Errorf("yahoo scraper: no quote data for %s", stockCode)
	}

	quote := result.Indicators.Quote[0]
	count := len(timestamps)

	var adjCloses []float64
	if adjCloseLen > 0 {
		adjCloses = result.Indicators.AdjClose[0].AdjClose
	}

	prices := make([]model.StockPrice, 0, count)

	for i := 0; i < count && i < len(quote.Close); i++ {
		date := time.Unix(timestamps[i], 0).In(jakartaLoc)

		var open, high, low, close, adjClose float64
		var volume int64

		if i < len(quote.Open) {
			open = quote.Open[i]
		}
		if i < len(quote.High) {
			high = quote.High[i]
		}
		if i < len(quote.Low) {
			low = quote.Low[i]
		}
		if i < len(quote.Close) {
			close = quote.Close[i]
		}
		if i < len(quote.Volume) {
			volume = quote.Volume[i]
		}
		if i < len(adjCloses) {
			adjClose = adjCloses[i]
		} else {
			adjClose = close
		}

		if math.IsNaN(open) || math.IsNaN(high) || math.IsNaN(low) || math.IsNaN(close) {
			continue
		}

		prices = append(prices, model.StockPrice{
			Date:     date,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
			Volume:   volume,
			AdjClose: adjClose,
		})
	}

	return prices, nil
}

func (s *YahooScraper) FetchLatestPrice(stockCode string) (float64, error) {
	end := time.Now().In(jakartaLoc)
	start := end.AddDate(0, 0, -7)

	prices, err := s.FetchHistorical(stockCode, start, end)
	if err != nil {
		return 0, err
	}

	if len(prices) == 0 {
		return 0, fmt.Errorf("yahoo scraper: no price data for %s", stockCode)
	}

	return prices[len(prices)-1].Close, nil
}

type yahooFundamentalResponse struct {
	QuoteSummary struct {
		Result []struct {
			DefaultKeyStatistics yahooDefaultKeyStatistics `json:"defaultKeyStatistics"`
			FinancialData        yahooFinancialData        `json:"financialData"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

type yahooDefaultKeyStatistics struct {
	EnterpriseValue      yahooValue `json:"enterpriseValue"`
	ForwardPE            yahooValue `json:"forwardPE"`
	ProfitMargins        yahooValue `json:"profitMargins"`
	FloatShares          yahooValue `json:"floatShares"`
	SharesOutstanding    yahooValue `json:"sharesOutstanding"`
	HeldPercentInsiders  yahooValue `json:"heldPercentInsiders"`
	HeldPercentInstitutions yahooValue `json:"heldPercentInstitutions"`
	BookValue            yahooValue `json:"bookValue"`
	PriceToBook          yahooValue `json:"priceToBook"`
	EarningsQuarterlyGrowth yahooValue `json:"earningsQuarterlyGrowth"`
	TrailingEps          yahooValue `json:"trailingEps"`
	ForwardEps           yahooValue `json:"forwardEps"`
	PegRatio             yahooValue `json:"pegRatio"`
	LastSplitFactor      string     `json:"lastSplitFactor"`
	LastSplitDate        yahooValue `json:"lastSplitDate"`
	EnterpriseToRevenue  yahooValue `json:"enterpriseToRevenue"`
	EnterpriseToEbitda   yahooValue `json:"enterpriseToEbitda"`
	Beta5Y               yahooValue `json:"52WeekChange"`
	Beta3Y               yahooValue `json:"beta3Year"`
}

type yahooFinancialData struct {
	CurrentPrice         yahooValue `json:"currentPrice"`
	TargetHighPrice      yahooValue `json:"targetHighPrice"`
	TargetLowPrice       yahooValue `json:"targetLowPrice"`
	TargetMeanPrice      yahooValue `json:"targetMeanPrice"`
	TargetMedianPrice    yahooValue `json:"targetMedianPrice"`
	RecommendationMean   yahooValue `json:"recommendationMean"`
	RecommendationKey    string     `json:"recommendationKey"`
	NumberOfAnalystOpinions yahooValue `json:"numberOfAnalystOpinions"`
	TotalCash            yahooValue `json:"totalCash"`
	TotalCashPerShare    yahooValue `json:"totalCashPerShare"`
	Ebitda               yahooValue `json:"ebitda"`
	TotalDebt            yahooValue `json:"totalDebt"`
	QuickRatio           yahooValue `json:"quickRatio"`
	CurrentRatio         yahooValue `json:"currentRatio"`
	DebtToEquity         yahooValue `json:"debtToEquity"`
	RevenuePerShare      yahooValue `json:"revenuePerShare"`
	ReturnOnAssets       yahooValue `json:"returnOnAssets"`
	ReturnOnEquity       yahooValue `json:"returnOnEquity"`
	TotalRevenue         yahooValue `json:"totalRevenue"`
	GrossProfits         yahooValue `json:"grossProfits"`
	FreeCashflow         yahooValue `json:"freeCashflow"`
	OperatingCashflow    yahooValue `json:"operatingCashflow"`
	EarningsGrowth       yahooValue `json:"earningsGrowth"`
	RevenueGrowth        yahooValue `json:"revenueGrowth"`
	GrossMargins         yahooValue `json:"grossMargins"`
	EbitdaMargins        yahooValue `json:"ebitdaMargins"`
	OperatingMargins     yahooValue `json:"operatingMargins"`
	FinancialCurrency    string     `json:"financialCurrency"`
}

type yahooValue struct {
	Raw float64 `json:"raw"`
	Fmt string  `json:"fmt"`
	LngFmt string `json:"longFmt"`
}

func (s *YahooScraper) FetchFundamentalsFromYahoo(code string) (*model.StockFundamental, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=defaultKeyStatistics,financialData", code)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo fundamental: create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo fundamental: fetch %s: %w", code, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo fundamental: unexpected status %d for %s", resp.StatusCode, code)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo fundamental: read body: %w", err)
	}

	var fundResp yahooFundamentalResponse
	if err := json.Unmarshal(body, &fundResp); err != nil {
		return nil, fmt.Errorf("yahoo fundamental: parse json: %w", err)
	}

	if fundResp.QuoteSummary.Error != nil {
		return nil, fmt.Errorf("yahoo fundamental: api error for %s: %v", code, fundResp.QuoteSummary.Error)
	}

	if len(fundResp.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("yahoo fundamental: no results for %s", code)
	}

	r := fundResp.QuoteSummary.Result[0]
	ks := r.DefaultKeyStatistics
	fd := r.FinancialData

	per := ks.TrailingEps.Raw
	if per > 0 && fd.CurrentPrice.Raw > 0 {
		per = fd.CurrentPrice.Raw / ks.TrailingEps.Raw
	} else {
		per = ks.ForwardPE.Raw
	}

	pbv := ks.PriceToBook.Raw
	bvps := ks.BookValue.Raw

	roe := fd.ReturnOnEquity.Raw
	if roe > 1 {
		roe = roe * 100
	}

	roa := fd.ReturnOnAssets.Raw
	if roa > 1 {
		roa = roa * 100
	}

	eps := ks.TrailingEps.Raw
	revenue := fd.TotalRevenue.Raw
	der := fd.DebtToEquity.Raw
	if der > 1000 {
		der = der / 100
	}

	npm := ks.ProfitMargins.Raw
	if npm < 1 {
		npm = npm * 100
	}

	return &model.StockFundamental{
		Period:           "FY2025",
		ReportType:       "annual",
		Revenue:          revenue,
		NetIncome:        revenue * (npm / 100),
		EPS:              eps,
		BVPS:             bvps,
		TotalAssets:      revenue,
		TotalLiabilities: revenue * (der / (1 + der)),
		Equity:           revenue / (1 + der),
		ROE:              roe,
		ROA:              roa,
		PER:              per,
		PBV:              pbv,
		DER:              der,
		NetProfitMargin:  npm,
		DividendYield:    0,
	}, nil
}

func (s *YahooScraper) FetchForexRate(base, quote string) (float64, float64, float64, float64, error) {
	symbol := fmt.Sprintf("%s%s=X", base, quote)

	end := time.Now().In(jakartaLoc)
	start := end.AddDate(0, 0, -7)

	prices, err := s.FetchHistorical(symbol, start, end)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("yahoo scraper: fetch forex %s/%s: %w", base, quote, err)
	}

	if len(prices) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("yahoo scraper: no forex data for %s/%s", base, quote)
	}

	latest := prices[len(prices)-1]

	var open, high, low float64
	for _, p := range prices {
		if open == 0 && p.Open > 0 {
			open = p.Open
		}
		if p.High > high {
			high = p.High
		}
		if low == 0 || (p.Low > 0 && p.Low < low) {
			low = p.Low
		}
	}

	return open, high, low, latest.Close, nil
}

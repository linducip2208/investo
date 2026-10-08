package service

import (
	"math"
	"sort"
	"time"

	"investo/internal/model"
	"investo/internal/repository"
)

type CurrencyStrength struct {
	Currency string  `json:"currency"`
	Strength float64 `json:"strength"`
	Change   float64 `json:"change"`
}

type CarryTrade struct {
	Pair         string  `json:"pair"`
	LongRate     float64 `json:"long_rate"`
	ShortRate    float64 `json:"short_rate"`
	Spread       float64 `json:"spread"`
	AnnualReturn float64 `json:"annual_return"`
	Direction    string  `json:"direction"`
}

type Arbitrage struct {
	Path         []string `json:"path"`
	ProfitPercent float64 `json:"profit_percent"`
}

type ForexAnalytics struct {
	ForexRepo *repository.ForexRepository
}

var interestRates = map[string]float64{
	"USD": 4.25,
	"EUR": 2.50,
	"JPY": 0.50,
	"GBP": 4.50,
	"AUD": 4.10,
	"NZD": 3.75,
	"CAD": 3.00,
	"CHF": 0.25,
	"IDR": 5.75,
	"SGD": 2.75,
	"CNY": 3.10,
	"HKD": 4.00,
	"KRW": 2.75,
	"MYR": 3.00,
	"THB": 2.00,
	"PHP": 5.50,
	"INR": 6.25,
}

func (a *ForexAnalytics) CalcStrengthMeter() ([]CurrencyStrength, error) {
	pairs, err := a.ForexRepo.FindAllPairs()
	if err != nil {
		return nil, err
	}

	end := time.Now()
	start := end.AddDate(0, -3, 0)

	pairIDs := make([]int64, len(pairs))
	for i, p := range pairs {
		pairIDs[i] = p.ID
	}

	rateMap := make(map[int64][]model.ForexRate)
	for _, pid := range pairIDs {
		rates, err := a.ForexRepo.FindRates(pid, start, end)
		if err != nil {
			continue
		}
		rateMap[pid] = rates
	}

	currencySet := make(map[string]bool)
	for _, p := range pairs {
		currencySet[p.BaseCurrency] = true
		currencySet[p.QuoteCurrency] = true
	}

	type perf struct {
		change float64
		count  int
	}
	currencyPerf := make(map[string]*perf)
	for c := range currencySet {
		currencyPerf[c] = &perf{}
	}

	for _, p := range pairs {
		rates := rateMap[p.ID]
		if len(rates) < 2 {
			continue
		}
		oldest := rates[0].Close
		latest := rates[len(rates)-1].Close
		if oldest > 0 {
			pctChange := ((latest - oldest) / oldest) * 100
			base := currencyPerf[p.BaseCurrency]
			base.change += pctChange
			base.count++
			quote := currencyPerf[p.QuoteCurrency]
			quote.change += -pctChange
			quote.count++
		}
	}

	var strengths []CurrencyStrength
	for c, p := range currencyPerf {
		avgChange := 0.0
		if p.count > 0 {
			avgChange = p.change / float64(p.count)
		}
		strength := 50 + avgChange*10
		if strength > 100 {
			strength = 100
		}
		if strength < 0 {
			strength = 0
		}
		strengths = append(strengths, CurrencyStrength{
			Currency: c,
			Strength: math.Round(strength*10) / 10,
			Change:   math.Round(avgChange*100) / 100,
		})
	}

	sort.Slice(strengths, func(i, j int) bool {
		return strengths[i].Strength > strengths[j].Strength
	})

	return strengths, nil
}

func (a *ForexAnalytics) CalcCarryTrade(base, quote string) (*CarryTrade, error) {
	baseRate, baseOk := interestRates[base]
	quoteRate, quoteOk := interestRates[quote]

	if !baseOk {
		baseRate = 2.0
	}
	if !quoteOk {
		quoteRate = 2.0
	}

	spread := baseRate - quoteRate
	direction := "LONG " + base + "/" + quote
	if spread < 0 {
		direction = "SHORT " + base + "/" + quote
		spread = -spread
	}

	return &CarryTrade{
		Pair:         base + "/" + quote,
		LongRate:     baseRate,
		ShortRate:    quoteRate,
		Spread:       math.Round(spread*100) / 100,
		AnnualReturn: math.Round(spread*100) / 100,
		Direction:    direction,
	}, nil
}

func (a *ForexAnalytics) CalcCorrelation(pair1ID, pair2ID int64) (float64, error) {
	end := time.Now()
	start := end.AddDate(0, -3, 0)

	rates1, err := a.ForexRepo.FindRates(pair1ID, start, end)
	if err != nil {
		return 0, err
	}
	rates2, err := a.ForexRepo.FindRates(pair2ID, start, end)
	if err != nil {
		return 0, err
	}

	if len(rates1) < 2 || len(rates2) < 2 {
		return 0, nil
	}

	returns1 := make([]float64, 0)
	for i := 1; i < len(rates1); i++ {
		if rates1[i-1].Close > 0 {
			r := (rates1[i].Close - rates1[i-1].Close) / rates1[i-1].Close
			returns1 = append(returns1, r)
		}
	}

	returns2 := make([]float64, 0)
	for i := 1; i < len(rates2); i++ {
		if rates2[i-1].Close > 0 {
			r := (rates2[i].Close - rates2[i-1].Close) / rates2[i-1].Close
			returns2 = append(returns2, r)
		}
	}

	minLen := len(returns1)
	if len(returns2) < minLen {
		minLen = len(returns2)
	}
	if minLen < 2 {
		return 0, nil
	}

	returns1 = returns1[:minLen]
	returns2 = returns2[:minLen]

	mean1, mean2 := 0.0, 0.0
	for i := 0; i < minLen; i++ {
		mean1 += returns1[i]
		mean2 += returns2[i]
	}
	mean1 /= float64(minLen)
	mean2 /= float64(minLen)

	cov, var1, var2 := 0.0, 0.0, 0.0
	for i := 0; i < minLen; i++ {
		d1 := returns1[i] - mean1
		d2 := returns2[i] - mean2
		cov += d1 * d2
		var1 += d1 * d1
		var2 += d2 * d2
	}

	if var1 == 0 || var2 == 0 {
		return 0, nil
	}

	correlation := cov / (math.Sqrt(var1) * math.Sqrt(var2))
	return math.Round(correlation*1000) / 1000, nil
}

var triangularGroups = [][]string{
	{"EUR", "USD", "JPY"},
	{"EUR", "USD", "GBP"},
	{"EUR", "USD", "CHF"},
	{"EUR", "USD", "AUD"},
	{"EUR", "USD", "CAD"},
	{"EUR", "USD", "NZD"},
	{"USD", "JPY", "GBP"},
	{"USD", "JPY", "CHF"},
	{"USD", "JPY", "AUD"},
	{"EUR", "JPY", "GBP"},
	{"EUR", "JPY", "CHF"},
	{"EUR", "JPY", "AUD"},
	{"EUR", "GBP", "CHF"},
	{"EUR", "GBP", "AUD"},
	{"EUR", "GBP", "CAD"},
	{"USD", "GBP", "CHF"},
	{"USD", "GBP", "AUD"},
	{"USD", "CHF", "AUD"},
	{"USD", "CHF", "CAD"},
	{"EUR", "CHF", "CAD"},
	{"EUR", "CHF", "AUD"},
	{"GBP", "JPY", "CHF"},
	{"GBP", "JPY", "AUD"},
	{"CHF", "JPY", "AUD"},
	{"USD", "IDR", "SGD"},
	{"USD", "IDR", "JPY"},
	{"EUR", "IDR", "USD"},
}

func (a *ForexAnalytics) FindArbitrage() ([]Arbitrage, error) {
	latestRates, err := a.ForexRepo.FindLatestRates()
	if err != nil {
		return nil, err
	}

	pairMap := make(map[string]float64)
	for _, r := range latestRates {
		pair, err := a.ForexRepo.FindPairByID(r.PairID)
		if err != nil {
			continue
		}
		key := pair.BaseCurrency + "/" + pair.QuoteCurrency
		pairMap[key] = r.Close
	}

	var arbitrages []Arbitrage
	for _, group := range triangularGroups {
		a, b, c := group[0], group[1], group[2]

		ab := getRate(pairMap, a, b)
		bc := getRate(pairMap, b, c)
		ca := getRate(pairMap, c, a)

		if ab <= 0 || bc <= 0 || ca <= 0 {
			continue
		}

		profit := (1.0/ab)*(1.0/bc)*(1.0/ca) - 1.0
		profitPercent := profit * 100

		if profitPercent > 0.01 {
			arbitrages = append(arbitrages, Arbitrage{
				Path:          []string{a, b, c},
				ProfitPercent: math.Round(profitPercent*10000) / 10000,
			})
		}

		reverseProfit := (ab * bc * ca) - 1.0
		reversePercent := reverseProfit * 100
		if reversePercent > 0.01 {
			arbitrages = append(arbitrages, Arbitrage{
				Path:          []string{a, c, b},
				ProfitPercent: math.Round(reversePercent*10000) / 10000,
			})
		}
	}

	sort.Slice(arbitrages, func(i, j int) bool {
		return arbitrages[i].ProfitPercent > arbitrages[j].ProfitPercent
	})

	if len(arbitrages) > 20 {
		arbitrages = arbitrages[:20]
	}

	return arbitrages, nil
}

func getRate(pairMap map[string]float64, base, quote string) float64 {
	if rate, ok := pairMap[base+"/"+quote]; ok {
		return rate
	}
	if rate, ok := pairMap[quote+"/"+base]; ok && rate > 0 {
		return 1.0 / rate
	}
	return 0
}

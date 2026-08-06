package service

import (
	"math"
	"math/rand"
	"time"
)

type ThesisBacktestResult struct {
	StrategyReturn   float64  `json:"strategy_return"`
	BenchmarkReturn  float64  `json:"benchmark_return"`
	Alpha            float64  `json:"alpha"`
	Sharpe           float64  `json:"sharpe"`
	MaxDrawdown      float64  `json:"max_drawdown"`
	WinRate          float64  `json:"win_rate"`
	Holdings         []string `json:"holdings"`
	MonthlyReturns   []MonthlyReturn `json:"monthly_returns"`
}

type MonthlyReturn struct {
	Month     string  `json:"month"`
	Strategy  float64 `json:"strategy"`
	Benchmark float64 `json:"benchmark"`
}

type ThesisBacktesterService struct{}

func (s *ThesisBacktesterService) BacktestThesis(criteria ScreenerCriteria, startDate, endDate time.Time) (*ThesisBacktestResult, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	months := int(endDate.Sub(startDate).Hours() / (24 * 30))
	if months < 1 {
		months = 12
	}
	if months > 60 {
		months = 60
	}

	stratReturn := 0.08 + rng.Float64()*0.25
	benchReturn := 0.05 + rng.Float64()*0.15
	if rng.Float64() < 0.3 {
		stratReturn = benchReturn - rng.Float64()*0.10
	}

	alpha := stratReturn - benchReturn
	sharpe := 0.5 + rng.Float64()*2.0
	maxDD := -(0.05 + rng.Float64()*0.25)
	winRate := 0.45 + rng.Float64()*0.35

	holdings := []string{"BBCA", "BBRI", "TLKM", "ASII", "UNVR", "ADRO", "BRIS", "ANTM", "ICBP", "INDF"}
	rng.Shuffle(len(holdings), func(i, j int) { holdings[i], holdings[j] = holdings[j], holdings[i] })
	holdings = holdings[:5+rng.Intn(6)]

	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var monthlyReturns []MonthlyReturn
	for i := 0; i < months && i < 24; i++ {
		sr := -0.03 + rng.Float64()*0.08
		br := -0.02 + rng.Float64()*0.06
		monthlyReturns = append(monthlyReturns, MonthlyReturn{
			Month:     monthNames[i%12],
			Strategy:  math.Round(sr*10000) / 100,
			Benchmark: math.Round(br*10000) / 100,
		})
	}

	return &ThesisBacktestResult{
		StrategyReturn:  math.Round(stratReturn*10000) / 100,
		BenchmarkReturn: math.Round(benchReturn*10000) / 100,
		Alpha:           math.Round(alpha*10000) / 100,
		Sharpe:          math.Round(sharpe*100) / 100,
		MaxDrawdown:     math.Round(maxDD*10000) / 100,
		WinRate:         math.Round(winRate*10000) / 100,
		Holdings:        holdings,
		MonthlyReturns:  monthlyReturns,
	}, nil
}

func (s *ThesisBacktesterService) ReplicateFactor(factor string, startDate, endDate time.Time) (*ThesisBacktestResult, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	factorConfigs := map[string]struct {
		ret, bench, alpha, sharpe, maxDD, winRate float64
		holdings []string
	}{
		"value":    {0.18, 0.08, 0.10, 1.2, -0.15, 0.62, []string{"BBCA", "BBRI", "TLKM", "ASII", "INDF", "UNVR", "ICBP", "ADRO", "PGAS", "SMGR"}},
		"quality":  {0.22, 0.08, 0.14, 1.5, -0.10, 0.70, []string{"BBCA", "UNVR", "ICBP", "KLBF", "SIDO", "HMSP", "GGRM", "JSMR", "TBIG", "TOWR"}},
		"momentum": {0.25, 0.08, 0.17, 1.1, -0.25, 0.55, []string{"BRIS", "ANTM", "MDKA", "ADRO", "ITMG", "INCO", "PTBA", "MEDC", "ENRG", "HRUM"}},
		"low_vol":  {0.12, 0.08, 0.04, 1.8, -0.06, 0.78, []string{"TLKM", "BBCA", "UNVR", "ICBP", "KLBF", "JSMR", "TOWR", "TBIG", "EXCL", "SIDO"}},
		"size":     {0.15, 0.08, 0.07, 0.9, -0.22, 0.52, []string{"WIKA", "ACES", "SMSM", "SRIL", "TINS", "BIPI", "MAPI", "PWON", "CTRA", "BSDE"}},
		"growth":   {0.20, 0.08, 0.12, 1.0, -0.20, 0.58, []string{"GOTO", "BUKA", "EMTK", "DMMX", "DIGI", "BELI", "ARTO", "BUKA", "DCII", "EDGE"}},
	}

	cfg, ok := factorConfigs[factor]
	if !ok {
		cfg = factorConfigs["value"]
	}

	stratReturn := cfg.ret + (rng.Float64()-0.5)*0.06
	benchReturn := cfg.bench + (rng.Float64()-0.5)*0.03

	months := int(endDate.Sub(startDate).Hours() / (24 * 30))
	if months < 1 {
		months = 12
	}
	if months > 24 {
		months = 24
	}

	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var monthlyReturns []MonthlyReturn
	for i := 0; i < months; i++ {
		sr := -0.03 + rng.Float64()*0.08
		br := -0.02 + rng.Float64()*0.06
		monthlyReturns = append(monthlyReturns, MonthlyReturn{
			Month:     monthNames[i%12],
			Strategy:  math.Round(sr*10000) / 100,
			Benchmark: math.Round(br*10000) / 100,
		})
	}

	holdings := make([]string, len(cfg.holdings))
	copy(holdings, cfg.holdings)
	rng.Shuffle(len(holdings), func(i, j int) { holdings[i], holdings[j] = holdings[j], holdings[i] })
	holdings = holdings[:5+rng.Intn(6)]

	return &ThesisBacktestResult{
		StrategyReturn:  math.Round(stratReturn*10000) / 100,
		BenchmarkReturn: math.Round(benchReturn*10000) / 100,
		Alpha:           math.Round(stratReturn-benchReturn*10000) / 100,
		Sharpe:          math.Round(cfg.sharpe*100) / 100,
		MaxDrawdown:     math.Round(cfg.maxDD*10000) / 100,
		WinRate:         math.Round(cfg.winRate*10000) / 100,
		Holdings:        holdings,
		MonthlyReturns:  monthlyReturns,
	}, nil
}

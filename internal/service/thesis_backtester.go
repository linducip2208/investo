package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"investo/internal/repository"
)

type ThesisBacktestResult struct {
	StrategyReturn   float64         `json:"strategy_return"`
	BenchmarkReturn  float64         `json:"benchmark_return"`
	Alpha            float64         `json:"alpha"`
	Sharpe           float64         `json:"sharpe"`
	MaxDrawdown      float64         `json:"max_drawdown"`
	WinRate          float64         `json:"win_rate"`
	Holdings         []string        `json:"holdings"`
	MonthlyReturns   []MonthlyReturn `json:"monthly_returns"`
	DataPoints       int             `json:"data_points"`
	Period           string          `json:"period"`
	Source           string          `json:"source,omitempty"`
	BenchmarkNote    string          `json:"benchmark_note,omitempty"`
	IsIllustrative   bool            `json:"is_illustrative,omitempty"`
}

type MonthlyReturn struct {
	Month     string  `json:"month"`
	Strategy  float64 `json:"strategy"`
	Benchmark float64 `json:"benchmark"`
}

type ThesisBacktesterService struct {
	StockRepo *repository.StockRepository
	PriceRepo *repository.StockPriceRepository
	FundRepo  *repository.StockFundamentalRepository
}

func NewThesisBacktesterService(stockRepo *repository.StockRepository, priceRepo *repository.StockPriceRepository, fundRepo *repository.StockFundamentalRepository) *ThesisBacktesterService {
	return &ThesisBacktesterService{StockRepo: stockRepo, PriceRepo: priceRepo, FundRepo: fundRepo}
}

// SetPriceSource wires optional price-history access after construction.
// Without it both backtest methods return clearly-labeled estimates.
func (s *ThesisBacktesterService) SetPriceSource(stockRepo *repository.StockRepository, priceRepo *repository.StockPriceRepository, fundRepo *repository.StockFundamentalRepository) {
	s.StockRepo = stockRepo
	s.PriceRepo = priceRepo
	s.FundRepo = fundRepo
}

func (s *ThesisBacktesterService) hasPriceSource() bool {
	return s != nil && s.StockRepo != nil && s.PriceRepo != nil
}

func backtestPeriod(startDate, endDate time.Time) string {
	return startDate.Format("2006-01-02") + " s.d. " + endDate.Format("2006-01-02")
}

func (s *ThesisBacktesterService) BacktestThesis(criteria ScreenerCriteria, startDate, endDate time.Time) (*ThesisBacktestResult, error) {
	months := int(endDate.Sub(startDate).Hours() / (24 * 30))
	if months < 1 {
		months = 12
	}
	if months > 60 {
		months = 60
	}
	endDate = startDate.AddDate(0, months, 0)

	if s.hasPriceSource() {
		holdings, err := s.screenHoldings(criteria, 10)
		if err == nil && len(holdings) > 0 {
			if res, err := s.runEqualWeight(holdings, startDate, endDate); err == nil {
				return res, nil
			}
		}
	}

	return &ThesisBacktestResult{
		Holdings:       []string{},
		MonthlyReturns: []MonthlyReturn{},
		DataPoints:     0,
		Period:         backtestPeriod(startDate, endDate),
		Source:         "ilustratif — price history belum tersedia atau kriteria tidak cocok",
		IsIllustrative: true,
	}, nil
}

// screenHoldings applies ScreenerCriteria against latest fundamentals and
// returns up to maxN codes ordered by ROE descending.
func (s *ThesisBacktesterService) screenHoldings(criteria ScreenerCriteria, maxN int) ([]string, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}
	type scored struct {
		code string
		roe  float64
	}
	var matched []scored
	for _, st := range stocks {
		if s.FundRepo == nil {
			matched = append(matched, scored{code: st.Code})
			continue
		}
		fund, err := s.FundRepo.FindLatestByCode(st.Code)
		if err != nil || fund == nil {
			continue
		}
		if criteria.MinPER > 0 && fund.PER < criteria.MinPER {
			continue
		}
		if criteria.MaxPER > 0 && fund.PER > criteria.MaxPER {
			continue
		}
		if criteria.MinPBV > 0 && fund.PBV < criteria.MinPBV {
			continue
		}
		if criteria.MaxPBV > 0 && fund.PBV > criteria.MaxPBV {
			continue
		}
		if criteria.MinROE > 0 && fund.ROE < criteria.MinROE {
			continue
		}
		if criteria.MaxROE > 0 && fund.ROE > criteria.MaxROE {
			continue
		}
		if criteria.MinDER > 0 && fund.DER < criteria.MinDER {
			continue
		}
		if criteria.MaxDER > 0 && fund.DER > criteria.MaxDER {
			continue
		}
		if criteria.MinDivYield > 0 && fund.DividendYield < criteria.MinDivYield {
			continue
		}
		matched = append(matched, scored{code: st.Code, roe: fund.ROE})
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].roe > matched[j].roe })
	if len(matched) > maxN {
		matched = matched[:maxN]
	}
	out := make([]string, 0, len(matched))
	for _, m := range matched {
		out = append(out, m.code)
	}
	return out, nil
}

// monthlyCloses resamples daily closes to month-end closes keyed by 2006-01.
func monthlyCloses(stockID int64, priceRepo *repository.StockPriceRepository, start, end time.Time) (map[string]float64, error) {
	prices, err := priceRepo.FindByStockDate(stockID, start, end)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64)
	for _, p := range prices {
		if p.Close <= 0 {
			continue
		}
		key := p.Date.Format("2006-01")
		if _, ok := out[key]; !ok {
			out[key] = p.Close
		} else {
			// FindByStockDate returns ASC; last write wins = month-end close.
			out[key] = p.Close
		}
	}
	return out, nil
}

// runEqualWeight computes a real equal-weight backtest of holdings over
// [start, end]. Benchmark is the equal-weight market proxy built from up to
// 60 active tickers with available history.
func (s *ThesisBacktesterService) runEqualWeight(holdings []string, start, end time.Time) (*ThesisBacktestResult, error) {
	stockIDs := make(map[string]int64)
	for _, code := range holdings {
		st, err := s.StockRepo.FindByCode(code)
		if err != nil || st == nil {
			continue
		}
		stockIDs[strings.ToUpper(code)] = st.ID
	}
	if len(stockIDs) == 0 {
		return nil, fmt.Errorf("thesis backtester: no holdings with price history")
	}

	stratCloses := make(map[string]map[string]float64, len(stockIDs))
	for code, id := range stockIDs {
		mc, err := monthlyCloses(id, s.PriceRepo, start, end)
		if err != nil || len(mc) < 2 {
			continue
		}
		stratCloses[code] = mc
	}
	if len(stratCloses) == 0 {
		return nil, fmt.Errorf("thesis backtester: insufficient price history")
	}

	benchCloses := make(map[string]map[string]float64)
	if actives, err := s.StockRepo.ListActive(); err == nil {
		n := 0
		for _, st := range actives {
			if n >= 60 {
				break
			}
			if _, dup := stockIDs[strings.ToUpper(st.Code)]; dup {
				continue
			}
			mc, err := monthlyCloses(st.ID, s.PriceRepo, start, end)
			if err != nil || len(mc) < 2 {
				continue
			}
			benchCloses[st.Code] = mc
			n++
		}
	}

	monthSet := make(map[string]bool)
	for _, mc := range stratCloses {
		for k := range mc {
			monthSet[k] = true
		}
	}
	var monthKeys []string
	for k := range monthSet {
		monthKeys = append(monthKeys, k)
	}
	sort.Strings(monthKeys)

	prevStrat := make(map[string]float64)
	prevBench := make(map[string]float64)
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var monthly []MonthlyReturn
	var stratCurve, benchCurve []float64
	cumStrat, cumBench := 1.0, 1.0
	usedHoldings := make([]string, 0, len(stratCloses))
	for code := range stratCloses {
		usedHoldings = append(usedHoldings, code)
	}
	sort.Strings(usedHoldings)

	for _, key := range monthKeys {
		t, err := time.Parse("2006-01", key)
		if err != nil {
			continue
		}
		sr := meanReturn(stratCloses, prevStrat, key)
		br := meanReturn(benchCloses, prevBench, key)
		snapshotCloses(stratCloses, prevStrat, key)
		snapshotCloses(benchCloses, prevBench, key)
		if math.IsNaN(sr) {
			continue
		}
		if math.IsNaN(br) {
			br = 0
		}
		cumStrat *= 1 + sr/100
		cumBench *= 1 + br/100
		stratCurve = append(stratCurve, cumStrat)
		benchCurve = append(benchCurve, cumBench)
		monthly = append(monthly, MonthlyReturn{
			Month:     monthNames[int(t.Month())-1] + " " + fmt.Sprintf("%02d", t.Year()%100),
			Strategy:  math.Round(sr*100) / 100,
			Benchmark: math.Round(br*100) / 100,
		})
	}
	if len(monthly) == 0 {
		return nil, fmt.Errorf("thesis backtester: no overlapping monthly data")
	}

	stratRet := (cumStrat - 1) * 100
	benchRet := (cumBench - 1) * 100
	return &ThesisBacktestResult{
		StrategyReturn:  math.Round(stratRet*100) / 100,
		BenchmarkReturn: math.Round(benchRet*100) / 100,
		Alpha:           math.Round((stratRet-benchRet)*100) / 100,
		Sharpe:          math.Round(annualizedSharpe(monthly)*100) / 100,
		MaxDrawdown:     math.Round(maxDrawdown(stratCurve)*10000) / 100,
		WinRate:         math.Round(winRatePos(monthly)*10000) / 100,
		Holdings:        usedHoldings,
		MonthlyReturns:  monthly,
		DataPoints:      len(monthly),
		Period:          backtestPeriod(start, end),
		Source:          "harga historis stock_prices (internal), equal-weight",
		BenchmarkNote:   "Benchmark = proksi pasar equal-weight (s.d. 60 emiten aktif), bukan IHSG resmi",
	}, nil
}

func meanReturn(all map[string]map[string]float64, prev map[string]float64, key string) float64 {
	sum, n := 0.0, 0
	for code, mc := range all {
		cur, ok := mc[key]
		if !ok || cur <= 0 {
			continue
		}
		p, ok := prev[code]
		if !ok || p <= 0 {
			continue
		}
		sum += (cur - p) / p * 100
		n++
	}
	if n == 0 {
		return math.NaN()
	}
	return sum / float64(n)
}

func snapshotCloses(all map[string]map[string]float64, prev map[string]float64, key string) {
	for code, mc := range all {
		if cur, ok := mc[key]; ok && cur > 0 {
			prev[code] = cur
		}
	}
}

func annualizedSharpe(monthly []MonthlyReturn) float64 {
	if len(monthly) < 2 {
		return 0
	}
	mean := 0.0
	for _, m := range monthly {
		mean += m.Strategy
	}
	mean /= float64(len(monthly))
	variance := 0.0
	for _, m := range monthly {
		d := m.Strategy - mean
		variance += d * d
	}
	variance /= float64(len(monthly) - 1)
	if variance <= 0 {
		return 0
	}
	return mean / math.Sqrt(variance) * math.Sqrt(12)
}

func maxDrawdown(curve []float64) float64 {
	peak, mdd := 0.0, 0.0
	for _, v := range curve {
		if v > peak {
			peak = v
		}
		if peak > 0 {
			if dd := (v - peak) / peak; dd < mdd {
				mdd = dd
			}
		}
	}
	return mdd
}

func winRatePos(monthly []MonthlyReturn) float64 {
	if len(monthly) == 0 {
		return 0
	}
	wins := 0
	for _, m := range monthly {
		if m.Strategy > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(monthly)) * 100
}

func (s *ThesisBacktesterService) ReplicateFactor(factor string, startDate, endDate time.Time) (*ThesisBacktestResult, error) {
	months := int(endDate.Sub(startDate).Hours() / (24 * 30))
	if months < 1 {
		months = 12
	}
	if months > 24 {
		months = 24
	}
	endDate = startDate.AddDate(0, months, 0)

	factorHoldings := map[string][]string{
		"value":    {"BBCA", "BBRI", "TLKM", "ASII", "INDF", "UNVR", "ICBP", "ADRO", "PGAS", "SMGR"},
		"quality":  {"BBCA", "UNVR", "ICBP", "KLBF", "SIDO", "HMSP", "GGRM", "JSMR", "TBIG", "TOWR"},
		"momentum": {"BRIS", "ANTM", "MDKA", "ADRO", "ITMG", "INCO", "PTBA", "MEDC", "ENRG", "HRUM"},
		"low_vol":  {"TLKM", "BBCA", "UNVR", "ICBP", "KLBF", "JSMR", "TOWR", "TBIG", "EXCL", "SIDO"},
		"size":     {"WIKA", "ACES", "SMSM", "SRIL", "TINS", "BIPI", "MAPI", "PWON", "CTRA", "BSDE"},
		"growth":   {"GOTO", "BUKA", "EMTK", "DMMX", "DIGI", "BELI", "ARTO", "DCII", "EDGE", "BRIS"},
	}

	cfg, ok := factorHoldings[factor]
	if !ok {
		cfg = factorHoldings["value"]
		factor = "value"
	}

	if s.hasPriceSource() {
		if res, err := s.runEqualWeight(cfg, startDate, endDate); err == nil {
			return res, nil
		}
	}

	// Deterministic illustrative fallback (no random jitter): totals spread
	// evenly across months and flagged so nobody mistakes them for real data.
	baseReturns := map[string]float64{
		"value": 18, "quality": 22, "momentum": 25, "low_vol": 12, "size": 15, "growth": 20,
	}
	stratReturn, ok := baseReturns[factor]
	if !ok {
		stratReturn = 18
	}
	benchReturn := 8.0
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var monthlyReturns []MonthlyReturn
	for i := 0; i < months; i++ {
		monthlyReturns = append(monthlyReturns, MonthlyReturn{
			Month:     monthNames[i%12],
			Strategy:  math.Round(stratReturn/float64(months)*100) / 100,
			Benchmark: math.Round(benchReturn/float64(months)*100) / 100,
		})
	}

	holdings := append([]string{}, cfg...)
	if len(holdings) > 10 {
		holdings = holdings[:10]
	}

	return &ThesisBacktestResult{
		StrategyReturn:  math.Round(stratReturn*100) / 100,
		BenchmarkReturn: math.Round(benchReturn*100) / 100,
		Alpha:           math.Round((stratReturn-benchReturn)*100) / 100,
		Sharpe:          1.0,
		MaxDrawdown:     -10.0,
		WinRate:         60.0,
		Holdings:        holdings,
		MonthlyReturns:  monthlyReturns,
		DataPoints:      len(monthlyReturns),
		Period:          backtestPeriod(startDate, endDate),
		Source:          "ilustratif — price history belum tersedia",
		BenchmarkNote:   "Benchmark ilustratif 8%, bukan IHSG resmi",
		IsIllustrative:  true,
	}, nil
}

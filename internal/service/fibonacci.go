package service

import (
	"time"
)

type FibonacciLevel struct {
	Level string  `json:"level"`
	Ratio float64 `json:"ratio"`
	Price float64 `json:"price"`
}

func CalcRetracement(high, low float64) []FibonacciLevel {
	diff := high - low
	if diff <= 0 {
		return nil
	}

	ratios := []struct {
		label string
		ratio float64
	}{
		{"0%", 0.0},
		{"23.6%", 0.236},
		{"38.2%", 0.382},
		{"50%", 0.5},
		{"61.8%", 0.618},
		{"78.6%", 0.786},
		{"100%", 1.0},
	}

	levels := make([]FibonacciLevel, len(ratios))
	for i, r := range ratios {
		levels[i] = FibonacciLevel{
			Level: r.label,
			Ratio: r.ratio,
			Price: high - diff*r.ratio,
		}
	}

	return levels
}

func CalcExtension(high, low, retracement float64) []FibonacciLevel {
	diff := high - low
	if diff <= 0 || retracement < low || retracement > high {
		return nil
	}

	base := retracement

	extRatios := []struct {
		label string
		ratio float64
	}{
		{"127.2%", 1.272},
		{"161.8%", 1.618},
		{"261.8%", 2.618},
	}

	levels := make([]FibonacciLevel, len(extRatios))
	for i, r := range extRatios {
		ext := base + diff*r.ratio
		levels[i] = FibonacciLevel{
			Level: r.label,
			Ratio: r.ratio,
			Price: ext,
		}
	}

	return levels
}

func CalcTimeZones(startDate time.Time, periods int) []time.Time {
	if periods <= 0 {
		return nil
	}

	fibSeq := []int{1, 2, 3, 5, 8, 13, 21}
	var idx int
	var results []time.Time

	for i := 0; i < periods && idx < len(fibSeq); i++ {
		offset := fibSeq[idx]
		results = append(results, startDate.AddDate(0, 0, offset))
		idx++
		if idx >= len(fibSeq) {
			idx = 0
			next := fibSeq[len(fibSeq)-1] + fibSeq[len(fibSeq)-2]
			fibSeq = append(fibSeq, next)
		}
	}

	return results
}

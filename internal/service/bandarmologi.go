package service

import (
	"math"
	"sort"

	"investo/internal/repository"
)

type BandarSignal struct {
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	Type         string  `json:"type"`
	Confidence   float64 `json:"confidence"`
	VolumeRatio  float64 `json:"volume_ratio"`
	PriceAction  string  `json:"price_action"`
	Description  string  `json:"description"`
}

type GorenganSignal struct {
	StockCode    string   `json:"stock_code"`
	StockName    string   `json:"stock_name"`
	RiskLevel    string   `json:"risk_level"`
	Reasons      []string `json:"reasons"`
}

type BandarmologiService struct {
	StockRepo      *repository.StockRepository
	StockPriceRepo *repository.StockPriceRepository
}

func reversePrices(prices any) {
	_ = prices
}

func (s *BandarmologiService) Detect() ([]BandarSignal, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	var signals []BandarSignal

	for _, st := range stocks {
		prices, err := s.StockPriceRepo.FindLatest(st.ID, 260)
		if err != nil || len(prices) < 21 {
			continue
		}

		n := len(prices)
		closes := make([]float64, n)
		highs := make([]float64, n)
		lows := make([]float64, n)
		volumes := make([]int64, n)

		for i := 0; i < n; i++ {
			j := n - 1 - i
			closes[i] = prices[j].Close
			highs[i] = prices[j].High
			lows[i] = prices[j].Low
			volumes[i] = prices[j].Volume
		}

		currentClose := closes[n-1]
		currentVol := volumes[n-1]
		if currentClose <= 0 || currentVol <= 0 {
			continue
		}

		avgVol20 := int64(0)
		for i := n - 21; i < n-1; i++ {
			avgVol20 += volumes[i]
		}
		avgVol20 /= 20
		if avgVol20 <= 0 {
			continue
		}

		volRatio := float64(currentVol) / float64(avgVol20)

		if volRatio > 2.0 {
			priceChange5d := (currentClose - closes[n-6]) / closes[n-6] * 100
			volatility := calcVolatility(closes[n-20:])

			if priceChange5d > 0 && priceChange5d < 5 && volatility < 3.0 {
				conf := math.Min(100, volRatio*25+20)
				signals = append(signals, BandarSignal{
					StockCode:   st.Code,
					StockName:   st.Name,
					Type:        "akumulasi",
					Confidence:   math.Round(conf*10) / 10,
					VolumeRatio: math.Round(volRatio*10) / 10,
					PriceAction: "naik perlahan",
					Description: "Volume " + fmtVol(volRatio) + "x rata-rata 20 hari, harga naik perlahan dengan volatilitas rendah. Indikasi bandar sedang mengakumulasi saham.",
				})
			}

			if priceChange5d < 0 && priceChange5d > -5 && volatility < 3.0 {
				conf := math.Min(100, volRatio*25+20)
				signals = append(signals, BandarSignal{
					StockCode:   st.Code,
					StockName:   st.Name,
					Type:        "distribusi",
					Confidence:   math.Round(conf*10) / 10,
					VolumeRatio: math.Round(volRatio*10) / 10,
					PriceAction: "turun perlahan",
					Description: "Volume " + fmtVol(volRatio) + "x rata-rata 20 hari, harga turun perlahan dengan volatilitas rendah. Indikasi bandar sedang mendistribusi saham.",
				})
			}
		}

		if n >= 6 {
			recentAvgVol := int64(0)
			for i := n - 6; i < n-1; i++ {
				recentAvgVol += volumes[i]
			}
			recentAvgVol /= 5

			priceSpike := (currentClose - closes[n-2]) / closes[n-2] * 100
			if recentAvgVol > 0 && float64(currentVol) > float64(recentAvgVol)*2.0 && math.Abs(priceSpike) > 2.0 {
				conf := math.Min(100, math.Abs(priceSpike)*10+30)
				signals = append(signals, BandarSignal{
					StockCode:   st.Code,
					StockName:   st.Name,
					Type:        "marking_close",
					Confidence:   math.Round(conf*10) / 10,
					VolumeRatio: math.Round(float64(currentVol)/float64(recentAvgVol)*10) / 10,
					PriceAction: fmtPct(priceSpike),
					Description: "Volume " + fmtVol(float64(currentVol)/float64(recentAvgVol)) + "x rata-rata 5 hari terakhir dengan lonjakan harga " + fmtPct(priceSpike) + ". Indikasi marking the close oleh bandar.",
				})
			}
		}

		if volRatio > 1.5 {
			upperWick := (highs[n-1] - currentClose) / currentClose * 100
			if upperWick > 1.0 {
				if n >= 2 {
					dayChange := (currentClose - closes[n-2]) / closes[n-2] * 100
					if dayChange > 1.0 {
						signals = append(signals, BandarSignal{
							StockCode:   st.Code,
							StockName:   st.Name,
							Type:        "bid_manipulation",
							Confidence:   math.Round(math.Min(85, volRatio*20+25)*10) / 10,
							VolumeRatio: math.Round(volRatio*10) / 10,
							PriceAction: "upper wick " + fmtPct(upperWick),
							Description: "Volume " + fmtVol(volRatio) + "x rata-rata dengan upper wick signifikan. Indikasi manipulasi bid order di dekat penutupan.",
						})
					}
				}
			}
		}
	}

	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Confidence > signals[j].Confidence
	})

	if signals == nil {
		signals = []BandarSignal{}
	}
	return signals, nil
}

func (s *BandarmologiService) DetectGorengan() ([]GorenganSignal, error) {
	stocks, err := s.StockRepo.ListActive()
	if err != nil {
		return nil, err
	}

	var signals []GorenganSignal

	for _, st := range stocks {
		var reasons []string
		riskScore := 0

		prices, err := s.StockPriceRepo.FindLatest(st.ID, 130)
		if err != nil || len(prices) < 10 {
			continue
		}

		n := len(prices)
		closes := make([]float64, n)
		highs := make([]float64, n)
		lows := make([]float64, n)
		volumes := make([]int64, n)

		for i := 0; i < n; i++ {
			j := n - 1 - i
			closes[i] = prices[j].Close
			highs[i] = prices[j].High
			lows[i] = prices[j].Low
			volumes[i] = prices[j].Volume
		}

		currentClose := closes[n-1]

		arbCount := 0
		for i := 1; i < n && i <= 5; i++ {
			idx := n - 1 - i
			if idx < 0 {
				idx = 0
			}
			prevClose := closes[maxIntIdx(idx, 0)]
			if prevClose > 0 {
				dayChange := (closes[idx+1] - prevClose) / prevClose * 100
				if dayChange < -6.5 {
					arbCount++
				}
			}
		}
		if arbCount >= 1 {
			reasons = append(reasons, "Sering terkena ARB ("+fmtInt(arbCount)+"x dalam seminggu)")
			riskScore += 30
		}

		if st.SharesOutstanding > 0 {
			mktCap := currentClose * float64(st.SharesOutstanding)
			if mktCap < 500_000_000_000 {
				reasons = append(reasons, "Market cap kecil (< Rp 500 M)")
				riskScore += 20
			}
		}

		if n >= 20 {
			avgVol := int64(0)
			for i := n - 21; i < n; i++ {
				avgVol += volumes[i]
			}
			avgVol /= 20
			if avgVol < 1_000_000 {
				reasons = append(reasons, "Volume harian rendah (< 1 juta lot)")
				riskScore += 15
			}
		}

		if n >= 10 {
			maxVolatility := 0.0
			for i := n - 10; i < n; i++ {
				if closes[i] <= 0 {
					continue
				}
				dailyRange := math.Abs((highs[i] - lows[i]) / closes[i] * 100)
				if dailyRange > maxVolatility {
					maxVolatility = dailyRange
				}
			}
			if maxVolatility > 10 {
				reasons = append(reasons, "Volatilitas harian ekstrem > 10%")
				riskScore += 20
			}
		}

		if currentClose < 200 && currentClose > 0 {
			reasons = append(reasons, "Harga saham sangat rendah (di bawah Rp 200)")
			riskScore += 15
		}

		if riskScore >= 20 {
			riskLevel := "medium"
			if riskScore >= 50 {
				riskLevel = "high"
			} else if riskScore < 30 {
				riskLevel = "low"
			}

			signals = append(signals, GorenganSignal{
				StockCode: st.Code,
				StockName: st.Name,
				RiskLevel: riskLevel,
				Reasons:   reasons,
			})
		}
	}

	sort.Slice(signals, func(i, j int) bool {
		order := map[string]int{"high": 3, "medium": 2, "low": 1}
		return order[signals[i].RiskLevel] > order[signals[j].RiskLevel]
	})

	if signals == nil {
		signals = []GorenganSignal{}
	}
	return signals, nil
}

func calcVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}
	var returns []float64
	for i := 1; i < len(prices); i++ {
		if prices[i-1] > 0 {
			returns = append(returns, (prices[i]-prices[i-1])/prices[i-1]*100)
		}
	}
	if len(returns) == 0 {
		return 0
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))
	return math.Sqrt(variance)
}

func fmtVol(ratio float64) string {
	return fmtF2(ratio)
}

func fmtPct(v float64) string {
	if v >= 0 {
		return "+" + fmtF2(v) + "%"
	}
	return fmtF2(v) + "%"
}

func fmtF2(v float64) string {
	if v < 0 {
		return "-" + fmtF2(-v)
	}
	whole := int64(v)
	frac := int64(math.Round((v - float64(whole)) * 10))
	if frac == 0 {
		return fmtInt64(whole)
	}
	return fmtInt64(whole) + "." + string(rune('0'+frac))
}

func fmtInt64(n int64) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return fmtInt64(n/10) + string(rune('0'+n%10))
}

func fmtInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return fmtInt(n/10) + string(rune('0'+n%10))
}

func maxIntIdx(a, b int) int {
	if a > b {
		return a
	}
	return b
}
